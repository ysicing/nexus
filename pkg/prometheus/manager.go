package prometheus

import (
	"fmt"
	"sync"
	"time"

	"github.com/ysicing/nexus/pkg/models"
	"k8s.io/klog/v2"
)

// Manager Prometheus 管理器
type Manager struct {
	clients map[string]*Client // clusterID -> Prometheus Client
	repo    models.ClusterRepository
	mu      sync.RWMutex
}

// NewManager 创建 Prometheus 管理器
func NewManager(repo models.ClusterRepository) *Manager {
	return &Manager{
		clients: make(map[string]*Client),
		repo:    repo,
	}
}

// Initialize 初始化 Prometheus 管理器，从数据库加载配置
func (m *Manager) Initialize() error {
	klog.Info("初始化 Prometheus 管理器...")

	clusters, err := m.repo.GetClustersWithPrometheus()
	if err != nil {
		return fmt.Errorf("获取 Prometheus 配置失败: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cluster := range clusters {
		if cluster.PrometheusEnabled && cluster.PrometheusURL != "" {
			client, err := NewClientWithAuth(cluster.PrometheusURL, cluster.PrometheusUsername, cluster.PrometheusPassword)
			if err != nil {
				klog.Warningf("创建集群 %s 的 Prometheus 客户端失败: %v", cluster.ID, err)
				continue
			}
			m.clients[cluster.ID] = client
			klog.Infof("为集群 %s 创建 Prometheus 客户端: %s", cluster.ID, cluster.PrometheusURL)
		}
	}

	klog.Infof("Prometheus 管理器初始化完成，共加载 %d 个客户端", len(m.clients))
	return nil
}

// GetClient 获取指定集群的 Prometheus 客户端
func (m *Manager) GetClient(clusterID string) (*Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[clusterID]
	if !exists {
		return nil, fmt.Errorf("集群 %s 没有配置 Prometheus", clusterID)
	}
	return client, nil
}

// UpdateClusterPrometheus 更新集群的 Prometheus 配置
func (m *Manager) UpdateClusterPrometheus(clusterID, url, username, password string, enabled bool) error {
	// 更新数据库
	if err := m.repo.UpdatePrometheusConfig(clusterID, url, username, password, enabled); err != nil {
		return fmt.Errorf("更新数据库 Prometheus 配置失败: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 删除旧客户端
	delete(m.clients, clusterID)

	// 如果启用，创建新客户端
	if enabled && url != "" {
		client, err := NewClientWithAuth(url, username, password)
		if err != nil {
			return fmt.Errorf("创建 Prometheus 客户端失败: %w", err)
		}
		m.clients[clusterID] = client
		klog.Infof("更新集群 %s 的 Prometheus 配置: %s", clusterID, url)
	} else {
		klog.Infof("禁用集群 %s 的 Prometheus 配置", clusterID)
	}

	return nil
}

// GetAllClients 获取所有 Prometheus 客户端
func (m *Manager) GetAllClients() map[string]*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make(map[string]*Client)
	for clusterID, client := range m.clients {
		clients[clusterID] = client
	}
	return clients
}

// RemoveCluster 移除集群的 Prometheus 配置
func (m *Manager) RemoveCluster(clusterID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.clients, clusterID)
	klog.Infof("移除集群 %s 的 Prometheus 配置", clusterID)
}

// RefreshFromDatabase 从数据库重新加载 Prometheus 配置
func (m *Manager) RefreshFromDatabase() error {
	clusters, err := m.repo.GetClustersWithPrometheus()
	if err != nil {
		return fmt.Errorf("获取 Prometheus 配置失败: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 清空现有客户端
	m.clients = make(map[string]*Client)

	// 重新加载
	for _, cluster := range clusters {
		if cluster.PrometheusEnabled && cluster.PrometheusURL != "" {
			client, err := NewClientWithAuth(cluster.PrometheusURL, cluster.PrometheusUsername, cluster.PrometheusPassword)
			if err != nil {
				klog.Warningf("创建集群 %s 的 Prometheus 客户端失败: %v", cluster.ID, err)
				continue
			}
			m.clients[cluster.ID] = client
		}
	}

	klog.Infof("从数据库重新加载 Prometheus 配置，共 %d 个客户端", len(m.clients))
	return nil
}

// HealthCheck 检查所有 Prometheus 连接的健康状态
func (m *Manager) HealthCheck() map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]error)
	for clusterID, client := range m.clients {
		// 简单的健康检查：查询 Prometheus 版本
		_, err := client.Query("prometheus_build_info", time.Now())
		results[clusterID] = err
	}
	return results
}
