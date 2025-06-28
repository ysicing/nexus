package cluster

import (
	"context"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// HealthChecker 集群健康检查器
type HealthChecker struct {
	manager  *Manager
	interval time.Duration
	stopCh   chan struct{}
	running  bool
	mu       sync.Mutex
}

// NewHealthChecker 创建新的健康检查器
func NewHealthChecker(manager *Manager) *HealthChecker {
	return &HealthChecker{
		manager:  manager,
		interval: 30 * time.Second, // 默认30秒检查一次
		stopCh:   make(chan struct{}),
	}
}

// Start 启动健康检查
func (h *HealthChecker) Start() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	klog.Info("Starting cluster health checker")

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	// 立即执行一次检查
	h.checkAllClusters()

	for {
		select {
		case <-ticker.C:
			h.checkAllClusters()
		case <-h.stopCh:
			klog.Info("Stopping cluster health checker")
			return
		}
	}
}

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	h.running = false
	close(h.stopCh)
}

// checkAllClusters 检查所有集群的健康状态
func (h *HealthChecker) checkAllClusters() {
	clusters := h.manager.ListClusters()

	var wg sync.WaitGroup
	for _, cluster := range clusters {
		wg.Add(1)
		go func(c *ClusterInfo) {
			defer wg.Done()
			h.checkClusterHealth(c)
		}(cluster)
	}

	wg.Wait()
}

// checkClusterHealth 检查单个集群的健康状态
func (h *HealthChecker) checkClusterHealth(cluster *ClusterInfo) {
	if cluster.Client == nil {
		h.updateClusterStatus(cluster, ClusterStatusUnreachable)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 尝试获取集群版本信息来测试连接
	_, err := cluster.Client.ClientSet.Discovery().ServerVersion()

	h.manager.mu.Lock()
	cluster.LastCheck = time.Now()
	h.manager.mu.Unlock()

	if err != nil {
		klog.V(4).Infof("Health check failed for cluster %s: %v", cluster.Name, err)
		h.updateClusterStatus(cluster, ClusterStatusUnreachable)
		return
	}

	// 检查节点状态
	nodes, err := cluster.Client.ClientSet.CoreV1().Nodes().List(ctx,
		metav1.ListOptions{})
	if err != nil {
		klog.V(4).Infof("Failed to list nodes for cluster %s: %v", cluster.Name, err)
		h.updateClusterStatus(cluster, ClusterStatusUnhealthy)
		return
	}

	// 检查是否有Ready的节点
	hasReadyNode := false
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				hasReadyNode = true
				break
			}
		}
		if hasReadyNode {
			break
		}
	}

	if hasReadyNode {
		h.updateClusterStatus(cluster, ClusterStatusHealthy)
	} else {
		h.updateClusterStatus(cluster, ClusterStatusUnhealthy)
	}
}

// updateClusterStatus 更新集群状态
func (h *HealthChecker) updateClusterStatus(cluster *ClusterInfo, status ClusterStatus) {
	h.manager.mu.Lock()
	defer h.manager.mu.Unlock()

	if cluster.Status != status {
		klog.V(2).Infof("Cluster %s status changed from %s to %s",
			cluster.Name, cluster.Status, status)
		cluster.Status = status
		cluster.UpdatedAt = time.Now()
	}
}

// SetInterval 设置检查间隔
func (h *HealthChecker) SetInterval(interval time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.interval = interval
}
