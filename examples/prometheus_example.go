package main

import (
	"fmt"
	"log"

	"github.com/ysicing/nexus/pkg/cluster"
	"github.com/ysicing/nexus/pkg/database"
	"github.com/ysicing/nexus/pkg/prometheus"
)

func main() {
	fmt.Println("=== Nexus Prometheus 数据库集成示例 ===")

	// 1. 初始化数据库
	fmt.Println("\n1. 初始化数据库...")
	dbConfig := database.GetDefaultConfig()
	db, err := database.NewDatabase(dbConfig)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.Close()

	// 执行数据库迁移
	if err := db.MigrateDatabase(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 2. 初始化集群管理器
	fmt.Println("\n2. 初始化集群管理器...")
	clusterManager := cluster.NewManagerWithDB(db)
	if err := clusterManager.Initialize(); err != nil {
		log.Fatalf("初始化集群管理器失败: %v", err)
	}
	defer clusterManager.Stop()

	// 3. 获取集群仓库
	repo := db.GetClusterRepository()

	// 4. 初始化 Prometheus 管理器
	fmt.Println("\n3. 初始化 Prometheus 管理器...")
	promManager := prometheus.NewManager(repo)
	if err := promManager.Initialize(); err != nil {
		log.Printf("初始化 Prometheus 管理器失败: %v", err)
	}

	// 5. 列出当前集群
	fmt.Println("\n4. 当前集群列表:")
	clusters := clusterManager.ListClusters()
	for i, cluster := range clusters {
		fmt.Printf("  [%d] %s (%s)\n", i+1, cluster.Name, cluster.ID)
		fmt.Printf("      服务器: %s\n", cluster.Server)
		if cluster.PrometheusEnabled {
			fmt.Printf("      Prometheus: %s (已启用)\n", cluster.PrometheusURL)
		} else {
			fmt.Printf("      Prometheus: 未配置\n")
		}
		fmt.Println()
	}

	// 6. 为第一个集群配置 Prometheus
	if len(clusters) > 0 {
		firstCluster := clusters[0]
		fmt.Printf("5. 为集群 '%s' 配置 Prometheus...\n", firstCluster.Name)

		// 配置示例 Prometheus 地址
		prometheusURL := "http://prometheus.example.com:9090"
		username := "admin"
		password := "secret"

		err := clusterManager.UpdateClusterPrometheus(
			firstCluster.ID,
			prometheusURL,
			username,
			password,
			true,
		)
		if err != nil {
			log.Printf("配置 Prometheus 失败: %v", err)
		} else {
			fmt.Printf("   成功配置 Prometheus: %s\n", prometheusURL)
		}

		// 7. 重新加载 Prometheus 管理器
		fmt.Println("\n6. 重新加载 Prometheus 配置...")
		if err := promManager.RefreshFromDatabase(); err != nil {
			log.Printf("重新加载 Prometheus 配置失败: %v", err)
		}

		// 8. 检查 Prometheus 客户端
		fmt.Println("\n7. Prometheus 客户端状态:")
		clients := promManager.GetAllClients()
		if len(clients) == 0 {
			fmt.Println("   没有配置的 Prometheus 客户端")
		} else {
			for clusterID := range clients {
				fmt.Printf("   集群 %s: Prometheus 客户端已创建\n", clusterID)
			}
		}

		// 9. 健康检查
		fmt.Println("\n8. Prometheus 健康检查...")
		healthResults := promManager.HealthCheck()
		for clusterID, err := range healthResults {
			if err != nil {
				fmt.Printf("   集群 %s: 连接失败 - %v\n", clusterID, err)
			} else {
				fmt.Printf("   集群 %s: 连接正常\n", clusterID)
			}
		}
	}

	// 10. 演示数据库查询
	fmt.Println("\n9. 数据库中的 Prometheus 配置:")
	promClusters, err := repo.GetClustersWithPrometheus()
	if err != nil {
		log.Printf("查询 Prometheus 集群失败: %v", err)
	} else {
		if len(promClusters) == 0 {
			fmt.Println("   没有配置 Prometheus 的集群")
		} else {
			for _, cluster := range promClusters {
				fmt.Printf("   集群: %s\n", cluster.Name)
				fmt.Printf("   Prometheus URL: %s\n", cluster.PrometheusURL)
				fmt.Printf("   用户名: %s\n", cluster.PrometheusUsername)
				fmt.Printf("   启用状态: %v\n", cluster.PrometheusEnabled)
				fmt.Println()
			}
		}
	}

	// 11. 演示配置更新
	if len(clusters) > 0 {
		firstCluster := clusters[0]
		fmt.Printf("10. 更新集群 '%s' 的 Prometheus 配置...\n", firstCluster.Name)

		// 更新配置
		newURL := "http://new-prometheus.example.com:9090"
		err := clusterManager.UpdateClusterPrometheus(
			firstCluster.ID,
			newURL,
			"newuser",
			"newsecret",
			true,
		)
		if err != nil {
			log.Printf("更新 Prometheus 配置失败: %v", err)
		} else {
			fmt.Printf("   成功更新 Prometheus URL: %s\n", newURL)
		}

		// 12. 验证更新
		fmt.Println("\n11. 验证配置更新:")
		url, username, password, enabled, err := clusterManager.GetClusterPrometheusConfig(firstCluster.ID)
		if err != nil {
			log.Printf("获取 Prometheus 配置失败: %v", err)
		} else {
			fmt.Printf("   URL: %s\n", url)
			fmt.Printf("   用户名: %s\n", username)
			fmt.Printf("   密码: %s\n", password)
			fmt.Printf("   启用: %v\n", enabled)
		}
	}

	fmt.Println("\n=== 示例完成 ===")
	fmt.Println("Prometheus 数据库集成功能演示完成!")
	fmt.Println("\n关键特性:")
	fmt.Println("  ✓ Prometheus 配置存储到数据库")
	fmt.Println("  ✓ 每个集群独立的 Prometheus 配置")
	fmt.Println("  ✓ 支持用户名密码认证")
	fmt.Println("  ✓ 动态配置更新和重新加载")
	fmt.Println("  ✓ Prometheus 连接健康检查")
}
