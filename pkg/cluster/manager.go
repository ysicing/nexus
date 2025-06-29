package cluster

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ysicing/nexus/pkg/kube"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

// ClusterInfo 集群信息
type ClusterInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Server      string            `json:"server"`
	Version     string            `json:"version,omitempty"`
	Status      ClusterStatus     `json:"status"`
	Config      *rest.Config      `json:"-"`
	Client      *kube.K8sClient   `json:"-"`
	Context     string            `json:"context,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	LastCheck   time.Time         `json:"lastCheck"`
	IsDefault   bool              `json:"isDefault"`

	// Kubeconfig 相关字段
	KubeconfigPath    string `json:"kubeconfigPath,omitempty"`
	KubeconfigContent string `json:"kubeconfigContent,omitempty"`

	// Prometheus 相关字段
	PrometheusURL      string `json:"prometheusUrl,omitempty"`
	PrometheusUsername string `json:"prometheusUsername,omitempty"`
	PrometheusPassword string `json:"prometheusPassword,omitempty"`
	PrometheusEnabled  bool   `json:"prometheusEnabled"`
}

// ClusterStatus 集群状态
type ClusterStatus string

const (
	ClusterStatusHealthy     ClusterStatus = "healthy"
	ClusterStatusUnhealthy   ClusterStatus = "unhealthy"
	ClusterStatusUnreachable ClusterStatus = "unreachable"
	ClusterStatusUnknown     ClusterStatus = "unknown"
)

// Manager 集群管理器
type Manager struct {
	clusters      map[string]*ClusterInfo
	defaultID     string
	mu            sync.RWMutex
	healthChecker *HealthChecker
}

// NewManager 创建新的集群管理器
func NewManager() *Manager {
	m := &Manager{
		clusters: make(map[string]*ClusterInfo),
	}
	m.healthChecker = NewHealthChecker(m)
	return m
}

// Initialize 初始化集群管理器，自动发现本地集群
func (m *Manager) Initialize() error {
	// 尝试添加当前集群（in-cluster或本地kubeconfig）
	if err := m.discoverLocalClusters(); err != nil {
		klog.Warningf("Failed to discover local clusters: %v", err)
	}

	// 启动健康检查
	go m.healthChecker.Start()

	return nil
}

// discoverLocalClusters 自动发现本地集群
func (m *Manager) discoverLocalClusters() error {
	// 1. 尝试in-cluster配置
	if config, err := rest.InClusterConfig(); err == nil {
		clusterInfo := &ClusterInfo{
			ID:          "in-cluster",
			Name:        "当前集群 (In-Cluster)",
			Description: "运行在集群内部的配置",
			Server:      config.Host,
			Status:      ClusterStatusUnknown,
			Config:      config,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			IsDefault:   true,
		}

		if client, err := kube.NewK8sClientFromConfig(config); err == nil {
			clusterInfo.Client = client
			if version, err := m.getClusterVersion(client); err == nil {
				clusterInfo.Version = version
			}

			m.mu.Lock()
			m.clusters[clusterInfo.ID] = clusterInfo
			m.defaultID = clusterInfo.ID
			m.mu.Unlock()

			klog.Infof("Added in-cluster configuration")
			return nil
		} else {
			klog.Warningf("Failed to create client for in-cluster config: %v", err)
		}
	}

	// 2. 尝试本地kubeconfig
	return m.discoverKubeconfigClusters()
}

// discoverKubeconfigClusters 从kubeconfig发现集群
func (m *Manager) discoverKubeconfigClusters() error {
	kubeconfigPath := ""
	if home := homedir.HomeDir(); home != "" {
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		kubeconfigPath = envKubeconfig
	}

	if kubeconfigPath == "" {
		return fmt.Errorf("could not find kubeconfig file")
	}

	// 扫描kubeconfig目录下的所有配置文件
	configDir := filepath.Dir(kubeconfigPath)
	files, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig directory: %w", err)
	}

	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		configPath := filepath.Join(configDir, file.Name())
		if err := m.loadKubeconfigFile(configPath); err != nil {
			klog.Warningf("Failed to load kubeconfig %s: %v", configPath, err)
			continue
		}
		count++
	}

	klog.Infof("Discovered %d kubeconfig files", count)
	return nil
}

// loadKubeconfigFile 加载单个kubeconfig文件
func (m *Manager) loadKubeconfigFile(configPath string) error {
	config, err := clientcmd.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	for contextName, context := range config.Contexts {
		clusterName := context.Cluster
		cluster, exists := config.Clusters[clusterName]
		if !exists {
			continue
		}

		// 构建REST配置
		clientConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		})

		restConfig, err := clientConfig.ClientConfig()
		if err != nil {
			klog.Warningf("Failed to create client config for context %s: %v", contextName, err)
			continue
		}

		clusterID := fmt.Sprintf("kubeconfig-%s", contextName)
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
			IsDefault:   m.defaultID == "",
		}

		// 尝试创建客户端
		if client, err := kube.NewK8sClientFromConfig(restConfig); err == nil {
			clusterInfo.Client = client
			if version, err := m.getClusterVersion(client); err == nil {
				clusterInfo.Version = version
			}
		} else {
			klog.Warningf("Failed to create client for cluster %s: %v", clusterInfo.Name, err)
			// 如果无法创建客户端，跳过这个集群
			continue
		}

		m.mu.Lock()
		m.clusters[clusterID] = clusterInfo
		if m.defaultID == "" {
			m.defaultID = clusterID
		}
		m.mu.Unlock()

		klog.Infof("Added cluster: %s", clusterInfo.Name)
	}

	return nil
}

// AddCluster 添加新集群
func (m *Manager) AddCluster(name, description, kubeconfigContent string, labels map[string]string) (*ClusterInfo, error) {
	config, err := clientcmd.Load([]byte(kubeconfigContent))
	if err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %w", err)
	}

	// 使用当前上下文
	currentContext := config.CurrentContext
	if currentContext == "" {
		// 如果没有当前上下文，使用第一个可用的
		for contextName := range config.Contexts {
			currentContext = contextName
			break
		}
	}

	if currentContext == "" {
		return nil, fmt.Errorf("no valid context found in kubeconfig")
	}

	// 构建REST配置
	clientConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{
		CurrentContext: currentContext,
	})

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create client config: %w", err)
	}

	// 测试连接
	client, err := kube.NewK8sClientFromConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
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

	// 获取集群版本
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

	klog.Infof("Added custom cluster: %s", name)
	return clusterInfo, nil
}

// RemoveCluster 移除集群
func (m *Manager) RemoveCluster(clusterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	// 不允许删除in-cluster配置
	if clusterID == "in-cluster" {
		return fmt.Errorf("cannot remove in-cluster configuration")
	}

	delete(m.clusters, clusterID)

	// 如果删除的是默认集群，选择新的默认集群
	if m.defaultID == clusterID {
		m.defaultID = ""
		for id, info := range m.clusters {
			m.defaultID = id
			info.IsDefault = true
			break
		}
	}

	klog.Infof("Removed cluster: %s", cluster.Name)
	return nil
}

// GetCluster 获取指定集群
func (m *Manager) GetCluster(clusterID string) (*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}

	return cluster, nil
}

// GetDefaultCluster 获取默认集群
func (m *Manager) GetDefaultCluster() (*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.defaultID == "" {
		return nil, fmt.Errorf("no default cluster configured")
	}

	return m.clusters[m.defaultID], nil
}

// ListClusters 列出所有集群
func (m *Manager) ListClusters() []*ClusterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clusters := make([]*ClusterInfo, 0, len(m.clusters))
	for _, cluster := range m.clusters {
		clusters = append(clusters, cluster)
	}

	return clusters
}

// SetDefaultCluster 设置默认集群
func (m *Manager) SetDefaultCluster(clusterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	// 取消之前的默认集群
	if m.defaultID != "" {
		if oldDefault, exists := m.clusters[m.defaultID]; exists {
			oldDefault.IsDefault = false
		}
	}

	m.defaultID = clusterID
	cluster.IsDefault = true

	klog.Infof("Set default cluster to: %s", cluster.Name)
	return nil
}

// UpdateClusterLabels 更新集群标签
func (m *Manager) UpdateClusterLabels(clusterID string, labels map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cluster, exists := m.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	cluster.Labels = labels
	cluster.UpdatedAt = time.Now()

	return nil
}

// getClusterVersion 获取集群版本
func (m *Manager) getClusterVersion(client *kube.K8sClient) (string, error) {
	version, err := client.ClientSet.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}

	return version.GitVersion, nil
}

// Stop 停止集群管理器
func (m *Manager) Stop() {
	if m.healthChecker != nil {
		m.healthChecker.Stop()
	}
}
