package cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ysicing/nexus/pkg/database"
	"github.com/ysicing/nexus/pkg/kube"
	"github.com/ysicing/nexus/pkg/models"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

// ManagerWithDB 带数据库支持的集群管理器
type ManagerWithDB struct {
	clusters      map[string]*ClusterInfo
	defaultID     string
	mu            sync.RWMutex
	healthChecker *HealthChecker
	db            *database.Database
	repo          models.ClusterRepository
}

// NewManagerWithDB 创建带数据库支持的集群管理器
func NewManagerWithDB(db *database.Database) *ManagerWithDB {
	m := &ManagerWithDB{
		clusters: make(map[string]*ClusterInfo),
		db:       db,
		repo:     db.GetClusterRepository(),
	}

	// 创建一个适配器来兼容 HealthChecker
	adapter := &Manager{
		clusters:      m.clusters,
		defaultID:     m.defaultID,
		mu:            m.mu,
		healthChecker: nil, // 避免循环引用
	}
	m.healthChecker = NewHealthChecker(adapter)

	return m
}

// Initialize 初始化集群管理器，参考 k8m 项目的四步加载机制
func (m *ManagerWithDB) Initialize() error {
	klog.Info("开始初始化集群管理器...")

	// 第一步：从数据库加载已存储的集群
	if err := m.loadClustersFromDB(); err != nil {
		klog.Warningf("从数据库加载集群失败: %v", err)
	}

	// 第二步：注册集群内配置（如果运行在集群内）
	if err := m.registerInCluster(); err != nil {
		klog.Warningf("注册集群内配置失败: %v", err)
	}

	// 第三步：扫描本地 kubeconfig 文件
	if err := m.scanClustersInDir(); err != nil {
		klog.Warningf("扫描本地集群配置失败: %v", err)
	}

	// 第四步：确保有默认集群
	if err := m.ensureDefaultCluster(); err != nil {
		klog.Warningf("设置默认集群失败: %v", err)
	}

	// 启动健康检查
	go m.healthChecker.Start()

	klog.Infof("集群管理器初始化完成，共加载 %d 个集群", len(m.clusters))
	return nil
}

// loadClustersFromDB 第一步：从数据库加载已存储的集群
func (m *ManagerWithDB) loadClustersFromDB() error {
	klog.Info("正在从数据库加载集群配置...")

	clusters, err := m.repo.GetAll()
	if err != nil {
		return fmt.Errorf("获取数据库集群列表失败: %w", err)
	}

	for _, clusterModel := range clusters {
		clusterInfo, err := m.modelToClusterInfo(clusterModel)
		if err != nil {
			klog.Warningf("转换集群模型失败 %s: %v", clusterModel.ID, err)
			continue
		}

		m.mu.Lock()
		m.clusters[clusterInfo.ID] = clusterInfo
		if clusterModel.IsDefault {
			m.defaultID = clusterInfo.ID
		}
		m.mu.Unlock()

		klog.Infof("从数据库加载集群: %s (%s)", clusterInfo.Name, clusterInfo.ID)
	}

	klog.Infof("从数据库加载了 %d 个集群", len(clusters))
	return nil
}

// registerInCluster 第二步：注册集群内配置
func (m *ManagerWithDB) registerInCluster() error {
	klog.Info("正在检查集群内配置...")

	config, err := rest.InClusterConfig()
	if err != nil {
		klog.V(4).Infof("未检测到集群内配置: %v", err)
		return nil
	}

	clusterID := "in-cluster"

	// 检查是否已存在
	m.mu.RLock()
	_, exists := m.clusters[clusterID]
	m.mu.RUnlock()

	if exists {
		klog.Info("集群内配置已存在，跳过注册")
		return nil
	}

	clusterInfo := &ClusterInfo{
		ID:          clusterID,
		Name:        "当前集群 (In-Cluster)",
		Description: "运行在集群内部的配置",
		Server:      config.Host,
		Status:      ClusterStatusUnknown,
		Config:      config,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 创建客户端
	if client, err := kube.NewK8sClientFromConfig(config); err == nil {
		clusterInfo.Client = client
		if version, err := m.getClusterVersion(client); err == nil {
			clusterInfo.Version = version
		}
	} else {
		klog.Warningf("创建集群内客户端失败: %v", err)
		return err
	}

	// 保存到内存
	m.mu.Lock()
	m.clusters[clusterID] = clusterInfo
	if m.defaultID == "" {
		m.defaultID = clusterID
		clusterInfo.IsDefault = true
	}
	m.mu.Unlock()

	// 保存到数据库
	if err := m.saveClusterToDB(clusterInfo, true); err != nil {
		klog.Warningf("保存集群内配置到数据库失败: %v", err)
	}

	klog.Info("成功注册集群内配置")
	return nil
}

// scanClustersInDir 第三步：扫描本地 kubeconfig 文件
func (m *ManagerWithDB) scanClustersInDir() error {
	klog.Info("正在扫描本地 kubeconfig 文件...")

	kubeconfigPath := ""
	if home := homedir.HomeDir(); home != "" {
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		kubeconfigPath = envKubeconfig
	}

	if kubeconfigPath == "" {
		klog.Warning("未找到 kubeconfig 文件路径")
		return nil
	}

	// 扫描kubeconfig目录下的所有配置文件
	configDir := filepath.Dir(kubeconfigPath)
	files, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("读取 kubeconfig 目录失败: %w", err)
	}

	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		configPath := filepath.Join(configDir, file.Name())
		if err := m.loadKubeconfigFile(configPath); err != nil {
			klog.Warningf("加载 kubeconfig 文件失败 %s: %v", configPath, err)
			continue
		}
		count++
	}

	klog.Infof("扫描了 %d 个 kubeconfig 文件", count)
	return nil
}

// loadKubeconfigFile 加载单个 kubeconfig 文件
func (m *ManagerWithDB) loadKubeconfigFile(configPath string) error {
	config, err := clientcmd.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("加载 kubeconfig 失败: %w", err)
	}

	for contextName, context := range config.Contexts {
		clusterName := context.Cluster
		cluster, exists := config.Clusters[clusterName]
		if !exists {
			continue
		}

		clusterID := fmt.Sprintf("kubeconfig-%s", contextName)

		// 检查是否已存在
		m.mu.RLock()
		_, exists = m.clusters[clusterID]
		m.mu.RUnlock()

		if exists {
			klog.V(4).Infof("集群 %s 已存在，跳过", clusterID)
			continue
		}

		// 构建REST配置
		clientConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		})

		restConfig, err := clientConfig.ClientConfig()
		if err != nil {
			klog.Warningf("创建客户端配置失败 %s: %v", contextName, err)
			continue
		}

		clusterInfo := &ClusterInfo{
			ID:          clusterID,
			Name:        fmt.Sprintf("%s (%s)", clusterName, contextName),
			Description: fmt.Sprintf("从 %s 加载的集群", filepath.Base(configPath)),
			Server:      cluster.Server,
			Status:      ClusterStatusUnknown,
			Config:      restConfig,
			Context:     contextName,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// 尝试创建客户端
		if client, err := kube.NewK8sClientFromConfig(restConfig); err == nil {
			clusterInfo.Client = client
			if version, err := m.getClusterVersion(client); err == nil {
				clusterInfo.Version = version
			}
		} else {
			klog.Warningf("创建客户端失败 %s: %v", clusterInfo.Name, err)
			continue
		}

		// 保存到内存
		m.mu.Lock()
		m.clusters[clusterID] = clusterInfo
		if m.defaultID == "" {
			m.defaultID = clusterID
			clusterInfo.IsDefault = true
		}
		m.mu.Unlock()

		// 保存到数据库
		if err := m.saveClusterToDB(clusterInfo, false); err != nil {
			klog.Warningf("保存集群到数据库失败 %s: %v", clusterInfo.Name, err)
		}

		klog.Infof("发现并加载集群: %s", clusterInfo.Name)
	}

	return nil
}

// ensureDefaultCluster 第四步：确保有默认集群
func (m *ManagerWithDB) ensureDefaultCluster() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果已有默认集群，直接返回
	if m.defaultID != "" {
		klog.Infof("默认集群已设置: %s", m.defaultID)
		return nil
	}

	// 如果没有任何集群，返回错误
	if len(m.clusters) == 0 {
		return fmt.Errorf("未发现任何可用集群")
	}

	// 选择第一个可用集群作为默认集群
	for id, cluster := range m.clusters {
		m.defaultID = id
		cluster.IsDefault = true

		// 更新数据库
		if err := m.saveClusterToDB(cluster, cluster.ID == "in-cluster"); err != nil {
			klog.Warningf("更新默认集群到数据库失败: %v", err)
		}

		klog.Infof("设置默认集群: %s (%s)", cluster.Name, id)
		break
	}

	return nil
}

// saveClusterToDB 保存集群信息到数据库
func (m *ManagerWithDB) saveClusterToDB(clusterInfo *ClusterInfo, isInCluster bool) error {
	// 将标签转换为 JSON 字符串
	labelsJSON := ""
	if len(clusterInfo.Labels) > 0 {
		if labelsBytes, err := json.Marshal(clusterInfo.Labels); err == nil {
			labelsJSON = string(labelsBytes)
		}
	}

	clusterModel := &models.ClusterModel{
		ID:                clusterInfo.ID,
		Name:              clusterInfo.Name,
		Description:       clusterInfo.Description,
		Server:            clusterInfo.Server,
		Version:           clusterInfo.Version,
		Status:            string(clusterInfo.Status),
		Context:           clusterInfo.Context,
		Labels:            labelsJSON,
		IsDefault:         clusterInfo.IsDefault,
		IsInCluster:       isInCluster,
		KubeconfigPath:    clusterInfo.KubeconfigPath,
		KubeconfigContent: clusterInfo.KubeconfigContent,
		LastCheck:         clusterInfo.LastCheck,
		CreatedAt:         clusterInfo.CreatedAt,
		UpdatedAt:         clusterInfo.UpdatedAt,
		// Prometheus 配置（如果有的话）
		PrometheusURL:      clusterInfo.PrometheusURL,
		PrometheusUsername: clusterInfo.PrometheusUsername,
		PrometheusPassword: clusterInfo.PrometheusPassword,
		PrometheusEnabled:  clusterInfo.PrometheusEnabled,
	}

	return m.repo.Create(clusterModel)
}

// modelToClusterInfo 将数据库模型转换为集群信息
func (m *ManagerWithDB) modelToClusterInfo(model *models.ClusterModel) (*ClusterInfo, error) {
	// 解析标签
	var labels map[string]string
	if model.Labels != "" {
		if err := json.Unmarshal([]byte(model.Labels), &labels); err != nil {
			klog.Warningf("解析集群标签失败 %s: %v", model.ID, err)
		}
	}

	clusterInfo := &ClusterInfo{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Server:      model.Server,
		Version:     model.Version,
		Status:      ClusterStatus(model.Status),
		Context:     model.Context,
		Labels:      labels,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
		LastCheck:   model.LastCheck,
		IsDefault:   model.IsDefault,

		// Kubeconfig 相关字段
		KubeconfigPath:    model.KubeconfigPath,
		KubeconfigContent: model.KubeconfigContent,

		// Prometheus 相关字段
		PrometheusURL:      model.PrometheusURL,
		PrometheusUsername: model.PrometheusUsername,
		PrometheusPassword: model.PrometheusPassword,
		PrometheusEnabled:  model.PrometheusEnabled,
	}

	// 对于 in-cluster 配置，尝试重新创建 REST 配置
	if model.IsInCluster {
		if config, err := rest.InClusterConfig(); err == nil {
			clusterInfo.Config = config
			if client, err := kube.NewK8sClientFromConfig(config); err == nil {
				clusterInfo.Client = client
			}
		}
	} else if model.Context != "" {
		// 对于外部集群，尝试从 kubeconfig 重新加载
		if err := m.loadClusterFromKubeconfig(clusterInfo, model.Context); err != nil {
			klog.Warningf("重新加载集群配置失败 %s: %v", model.ID, err)
		}
	}

	return clusterInfo, nil
}

// loadClusterFromKubeconfig 从 kubeconfig 重新加载集群配置
func (m *ManagerWithDB) loadClusterFromKubeconfig(clusterInfo *ClusterInfo, contextName string) error {
	kubeconfigPath := ""
	if home := homedir.HomeDir(); home != "" {
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		kubeconfigPath = envKubeconfig
	}

	if kubeconfigPath == "" {
		return fmt.Errorf("未找到 kubeconfig 文件")
	}

	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("加载 kubeconfig 失败: %w", err)
	}

	clientConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{
		CurrentContext: contextName,
	})

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("创建客户端配置失败: %w", err)
	}

	clusterInfo.Config = restConfig

	if client, err := kube.NewK8sClientFromConfig(restConfig); err == nil {
		clusterInfo.Client = client
	}

	return nil
}

// getClusterVersion 获取集群版本
func (m *ManagerWithDB) getClusterVersion(client *kube.K8sClient) (string, error) {
	version, err := client.ClientSet.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return version.GitVersion, nil
}

// 实现原有 Manager 接口的方法

// AddCluster 添加新集群
func (m *ManagerWithDB) AddCluster(name, description, kubeconfigContent string, labels map[string]string) (*ClusterInfo, error) {
	config, err := clientcmd.Load([]byte(kubeconfigContent))
	if err != nil {
		return nil, fmt.Errorf("无效的 kubeconfig: %w", err)
	}

	currentContext := config.CurrentContext
	if currentContext == "" {
		for contextName := range config.Contexts {
			currentContext = contextName
			break
		}
	}

	if currentContext == "" {
		return nil, fmt.Errorf("kubeconfig 中未找到有效的上下文")
	}

	clientConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{
		CurrentContext: currentContext,
	})

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("创建客户端配置失败: %w", err)
	}

	client, err := kube.NewK8sClientFromConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 kubernetes 客户端失败: %w", err)
	}

	clusterID := fmt.Sprintf("custom-%d", time.Now().Unix())
	clusterInfo := &ClusterInfo{
		ID:          clusterID,
		Name:        name,
		Description: description,
		Server:      restConfig.Host,
		Status:      ClusterStatusUnknown,
		Config:      restConfig,
		Client:      client,
		Context:     currentContext,
		Labels:      labels,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if version, err := m.getClusterVersion(client); err == nil {
		clusterInfo.Version = version
	}

	m.mu.Lock()
	m.clusters[clusterID] = clusterInfo
	if len(m.clusters) == 1 {
		m.defaultID = clusterID
		clusterInfo.IsDefault = true
	}
	m.mu.Unlock()

	// 保存到数据库
	if err := m.saveClusterToDB(clusterInfo, false); err != nil {
		klog.Warningf("保存自定义集群到数据库失败: %v", err)
	}

	klog.Infof("添加自定义集群: %s", name)
	return clusterInfo, nil
}

// RemoveCluster 移除集群
func (m *ManagerWithDB) RemoveCluster(clusterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("集群 %s 不存在", clusterID)
	}

	if clusterID == "in-cluster" {
		return fmt.Errorf("不能删除集群内配置")
	}

	delete(m.clusters, clusterID)

	// 从数据库删除
	if err := m.repo.Delete(clusterID); err != nil {
		klog.Warningf("从数据库删除集群失败: %v", err)
	}

	// 如果删除的是默认集群，选择新的默认集群
	if m.defaultID == clusterID {
		m.defaultID = ""
		for id, info := range m.clusters {
			m.defaultID = id
			info.IsDefault = true
			// 更新数据库
			if err := m.saveClusterToDB(info, id == "in-cluster"); err != nil {
				klog.Warningf("更新新默认集群到数据库失败: %v", err)
			}
			break
		}
	}

	klog.Infof("移除集群: %s", cluster.Name)
	return nil
}

// GetCluster 获取指定集群
func (m *ManagerWithDB) GetCluster(clusterID string) (*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return nil, fmt.Errorf("集群 %s 不存在", clusterID)
	}

	return cluster, nil
}

// GetDefaultCluster 获取默认集群
func (m *ManagerWithDB) GetDefaultCluster() (*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.defaultID == "" {
		return nil, fmt.Errorf("未配置默认集群")
	}

	return m.clusters[m.defaultID], nil
}

// ListClusters 列出所有集群
func (m *ManagerWithDB) ListClusters() []*ClusterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clusters := make([]*ClusterInfo, 0, len(m.clusters))
	for _, cluster := range m.clusters {
		clusters = append(clusters, cluster)
	}

	return clusters
}

// SetDefaultCluster 设置默认集群
func (m *ManagerWithDB) SetDefaultCluster(clusterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("集群 %s 不存在", clusterID)
	}

	// 取消之前的默认集群
	if m.defaultID != "" {
		if oldDefault, exists := m.clusters[m.defaultID]; exists {
			oldDefault.IsDefault = false
			if err := m.saveClusterToDB(oldDefault, oldDefault.ID == "in-cluster"); err != nil {
				klog.Warningf("更新旧默认集群状态失败: %v", err)
			}
		}
	}

	m.defaultID = clusterID
	cluster.IsDefault = true

	// 更新数据库
	if err := m.saveClusterToDB(cluster, cluster.ID == "in-cluster"); err != nil {
		klog.Warningf("更新新默认集群状态失败: %v", err)
	}

	klog.Infof("设置默认集群: %s", cluster.Name)
	return nil
}

// UpdateClusterLabels 更新集群标签
func (m *ManagerWithDB) UpdateClusterLabels(clusterID string, labels map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("集群 %s 不存在", clusterID)
	}

	cluster.Labels = labels
	cluster.UpdatedAt = time.Now()

	// 更新数据库
	if err := m.saveClusterToDB(cluster, cluster.ID == "in-cluster"); err != nil {
		klog.Warningf("更新集群标签到数据库失败: %v", err)
	}

	return nil
}

// Stop 停止集群管理器
func (m *ManagerWithDB) Stop() {
	if m.healthChecker != nil {
		m.healthChecker.Stop()
	}
	if m.db != nil {
		m.db.Close()
	}
}

// UpdateClusterPrometheus 更新集群的 Prometheus 配置
func (m *ManagerWithDB) UpdateClusterPrometheus(clusterID, url, username, password string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("集群 %s 不存在", clusterID)
	}

	// 更新内存中的配置
	cluster.PrometheusURL = url
	cluster.PrometheusUsername = username
	cluster.PrometheusPassword = password
	cluster.PrometheusEnabled = enabled
	cluster.UpdatedAt = time.Now()

	// 更新数据库
	if err := m.repo.UpdatePrometheusConfig(clusterID, url, username, password, enabled); err != nil {
		return fmt.Errorf("更新数据库 Prometheus 配置失败: %w", err)
	}

	klog.Infof("更新集群 %s 的 Prometheus 配置: enabled=%v, url=%s", clusterID, enabled, url)
	return nil
}

// GetClusterPrometheusConfig 获取集群的 Prometheus 配置
func (m *ManagerWithDB) GetClusterPrometheusConfig(clusterID string) (url, username, password string, enabled bool, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return "", "", "", false, fmt.Errorf("集群 %s 不存在", clusterID)
	}

	return cluster.PrometheusURL, cluster.PrometheusUsername, cluster.PrometheusPassword, cluster.PrometheusEnabled, nil
}

// GetClustersWithPrometheus 获取所有启用了 Prometheus 的集群
func (m *ManagerWithDB) GetClustersWithPrometheus() []*ClusterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var clusters []*ClusterInfo
	for _, cluster := range m.clusters {
		if cluster.PrometheusEnabled && cluster.PrometheusURL != "" {
			clusters = append(clusters, cluster)
		}
	}
	return clusters
}
