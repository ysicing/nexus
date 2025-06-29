package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	"github.com/ysicing/nexus/pkg/auth"
	"github.com/ysicing/nexus/pkg/cluster"
	"github.com/ysicing/nexus/pkg/common"
	"github.com/ysicing/nexus/pkg/database"
	"github.com/ysicing/nexus/pkg/handlers"
	"github.com/ysicing/nexus/pkg/handlers/resources"
	"github.com/ysicing/nexus/pkg/kube"
	"github.com/ysicing/nexus/pkg/middleware"
	"github.com/ysicing/nexus/pkg/prometheus"
	"github.com/ysicing/nexus/pkg/utils"
	"k8s.io/klog/v2"
)

//go:embed static
var static embed.FS

// ClusterManager 通用集群管理器接口
type ClusterManager interface {
	Initialize() error
	Stop()
	GetDefaultCluster() (*cluster.ClusterInfo, error)
	GetCluster(clusterID string) (*cluster.ClusterInfo, error)
	ListClusters() []*cluster.ClusterInfo
	AddCluster(name, description, kubeconfigContent string, labels map[string]string) (*cluster.ClusterInfo, error)
	RemoveCluster(clusterID string) error
	SetDefaultCluster(clusterID string) error
	UpdateClusterLabels(clusterID string, labels map[string]string) error
}

func setupStatic(r *gin.Engine) {
	assertsFS, err := fs.Sub(static, "static/assets")
	if err != nil {
		panic(err)
	}
	r.StaticFS("/assets", http.FS(assertsFS))
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 5 && path[:5] == "/api/" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		content, err := static.ReadFile("static/index.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read index.html"})
			return
		}

		htmlContent := string(content)
		if common.EnableAnalytics {
			// Inject analytics if enabled
			htmlContent = utils.InjectAnalytics(string(content))
		}

		// Set content type and serve modified HTML
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, htmlContent)
	})
}

func setupAPIRouter(r *gin.Engine, k8sClient *kube.K8sClient, promClient *prometheus.Client, clusterManager ClusterManager) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Auth routes (no auth required)
	authHandler := auth.NewAuthHandler()
	authGroup := r.Group("/api/auth")
	{
		authGroup.GET("/providers", authHandler.GetProviders)
		authGroup.POST("/login/password", authHandler.PasswordLogin)
		authGroup.GET("/login", authHandler.Login)
		authGroup.GET("/callback", authHandler.Callback)
		authGroup.POST("/logout", authHandler.Logout)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.GET("/user", authHandler.RequireAuth(), authHandler.GetUser)
	}

	// API routes group (protected)
	api := r.Group("/api/v1")
	api.Use(authHandler.RequireAuth(), middleware.ReadonlyMiddleware())
	{
		// 注册集群管理路由（支持所有类型的集群管理器）
		clusterManagerHandler := cluster.NewHandlerWithInterface(clusterManager)
		clusterManagerHandler.RegisterRoutes(api)

		// 根据实际的集群管理器类型来注册其他路由
		switch mgr := clusterManager.(type) {
		case *cluster.Manager:
			// 传统的内存集群管理器
			clusterHandler := handlers.NewClusterHandler(mgr)

			// 需要集群上下文的路由组
			clusterAPI := api.Group("")
			clusterAPI.Use(clusterHandler.ClusterMiddleware())
			{
				overviewHandler := handlers.NewOverviewHandler(k8sClient, promClient)
				clusterAPI.GET("/overview", overviewHandler.GetOverview)

				promHandler := handlers.NewPromHandler(promClient, k8sClient)
				clusterAPI.GET("/prometheus/resource-usage-history", promHandler.GetResourceUsageHistory)
				clusterAPI.GET("/prometheus/pods/:namespace/:podName/metrics", promHandler.GetPodMetrics)

				logsHandler := handlers.NewLogsHandler(k8sClient)
				clusterAPI.GET("/logs/:namespace/:podName", logsHandler.GetPodLogs)

				terminalHandler := handlers.NewTerminalHandler(k8sClient)
				clusterAPI.GET("/terminal/:namespace/:podName/ws", terminalHandler.HandleTerminalWebSocket)

				nodeTerminalHandler := handlers.NewNodeTerminalHandler(k8sClient)
				clusterAPI.GET("/node-terminal/:nodeName/ws", nodeTerminalHandler.HandleNodeTerminalWebSocket)

				searchHandler := handlers.NewSearchHandler(k8sClient)
				clusterAPI.GET("/search", searchHandler.GlobalSearch)

				resourceApplyHandler := handlers.NewResourceApplyHandler(k8sClient)
				clusterAPI.POST("/resources/apply", resourceApplyHandler.ApplyResource)

				// 注册资源路由，使用集群中间件
				resources.RegisterRoutesWithCluster(clusterAPI, mgr)
			}
		case *cluster.ManagerWithDB:
			// 数据库集群管理器 - 创建一个简化的集群处理器
			klog.Info("Using database cluster manager with full API support")

			// 创建一个简化的集群中间件（不依赖具体的 Manager 类型）
			clusterMiddleware := func(c *gin.Context) {
				clusterID := c.Query("cluster")
				if clusterID == "" {
					clusterID = c.GetHeader("X-Cluster-ID")
				}

				var clusterInfo *cluster.ClusterInfo
				var err error

				if clusterID != "" {
					clusterInfo, err = clusterManager.GetCluster(clusterID)
					if err != nil {
						klog.Warningf("Failed to get cluster %s: %v", clusterID, err)
						c.Set("k8sClient", nil)
						c.Next()
						return
					}
				} else {
					// 使用默认集群
					clusterInfo, err = clusterManager.GetDefaultCluster()
					if err != nil {
						klog.Warningf("Failed to get default cluster: %v", err)
						c.Set("k8sClient", nil)
						c.Next()
						return
					}
				}

				if clusterInfo.Client == nil {
					klog.Warningf("Cluster client not available for cluster: %s", clusterInfo.Name)
					c.Set("k8sClient", nil)
				} else {
					c.Set("k8sClient", clusterInfo.Client)
				}
				c.Next()
			}

			// 需要集群上下文的路由组
			clusterAPI := api.Group("")
			clusterAPI.Use(clusterMiddleware)
			{
				overviewHandler := handlers.NewOverviewHandler(k8sClient, promClient)
				clusterAPI.GET("/overview", overviewHandler.GetOverview)

				promHandler := handlers.NewPromHandler(promClient, k8sClient)
				clusterAPI.GET("/prometheus/resource-usage-history", promHandler.GetResourceUsageHistory)
				clusterAPI.GET("/prometheus/pods/:namespace/:podName/metrics", promHandler.GetPodMetrics)

				logsHandler := handlers.NewLogsHandler(k8sClient)
				clusterAPI.GET("/logs/:namespace/:podName", logsHandler.GetPodLogs)

				terminalHandler := handlers.NewTerminalHandler(k8sClient)
				clusterAPI.GET("/terminal/:namespace/:podName/ws", terminalHandler.HandleTerminalWebSocket)

				nodeTerminalHandler := handlers.NewNodeTerminalHandler(k8sClient)
				clusterAPI.GET("/node-terminal/:nodeName/ws", nodeTerminalHandler.HandleNodeTerminalWebSocket)

				searchHandler := handlers.NewSearchHandler(k8sClient)
				clusterAPI.GET("/search", searchHandler.GlobalSearch)

				resourceApplyHandler := handlers.NewResourceApplyHandler(k8sClient)
				clusterAPI.POST("/resources/apply", resourceApplyHandler.ApplyResource)

				// TODO: 注册资源路由 - 需要适配支持接口的版本
				// resources.RegisterRoutesWithCluster(clusterAPI, mgr)
			}
		default:
			klog.Errorf("Unknown cluster manager type: %T", clusterManager)
		}
	}
}

func setupWebhookRouter(r *gin.Engine, k8sClient *kube.K8sClient) {
	webhookGroup := r.Group("/api/v1/webhooks", gin.BasicAuth(gin.Accounts{
		common.WebhookUsername: common.WebhookPassword,
	}))
	{
		webhookHandler := handlers.NewWebhookHandler(k8sClient)
		webhookGroup.POST("/events", webhookHandler.HandleWebhook)
	}
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	common.LoadEnvs()
	gin.SetMode(gin.DebugMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// 初始化数据库（如果配置了 DATABASE_DSN）
	var clusterManager ClusterManager

	if databaseDSN := os.Getenv("DATABASE_DSN"); databaseDSN != "" {
		// 使用数据库集成的集群管理器
		klog.Info("Using database-integrated cluster manager")
		dbConfig := database.GetDefaultConfig()
		db, err := database.NewDatabase(dbConfig)
		if err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		defer db.Close()

		// 执行数据库迁移
		if err := db.MigrateDatabase(); err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
		}

		clusterManager = cluster.NewManagerWithDB(db)
	} else {
		// 使用传统的内存集群管理器
		klog.Info("Using memory-based cluster manager")
		clusterManager = cluster.NewManager()
	}

	if err := clusterManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize cluster manager: %v", err)
	}
	defer clusterManager.Stop()

	// 获取默认集群客户端用于向后兼容
	var k8sClient *kube.K8sClient
	defaultCluster, err := clusterManager.GetDefaultCluster()
	if err != nil {
		klog.Warningf("No default cluster available: %v", err)
		klog.Info("Application will start without default cluster - you can add clusters via the web interface")
		// 创建一个空的占位符客户端
		k8sClient = &kube.K8sClient{}
	} else {
		k8sClient = defaultCluster.Client
		if k8sClient == nil {
			klog.Warningf("Default cluster client is nil")
			k8sClient = &kube.K8sClient{}
		}
	}

	// 初始化 Prometheus 客户端（向后兼容）
	// 注意：新架构中，Prometheus 配置存储在数据库中，由 prometheus.Manager 管理
	// 这里保留一个默认客户端用于向后兼容
	var promClient *prometheus.Client
	if defaultCluster != nil && defaultCluster.PrometheusEnabled && defaultCluster.PrometheusURL != "" {
		promClient, err = prometheus.NewClientWithAuth(
			defaultCluster.PrometheusURL,
			defaultCluster.PrometheusUsername,
			defaultCluster.PrometheusPassword,
		)
		if err != nil {
			klog.Errorf("Failed to create Prometheus client for default cluster: %v", err)
			promClient = nil
		} else {
			klog.Infof("Initialized Prometheus client for default cluster: %s", defaultCluster.PrometheusURL)
		}
	} else {
		klog.Info("No Prometheus configuration found for default cluster - monitoring features may be limited")
	}

	// Setup router
	setupAPIRouter(r, k8sClient, promClient, clusterManager)
	setupWebhookRouter(r, k8sClient)
	setupStatic(r)

	srv := &http.Server{
		Addr:    ":" + common.Port,
		Handler: r.Handler(),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Fatalf("Failed to start server: %v", err)
		}
	}()
	klog.Infof("Kite server started on port %s", common.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	klog.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		klog.Fatalf("Failed to shutdown server: %v", err)
	}
}
