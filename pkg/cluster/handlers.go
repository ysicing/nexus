package cluster

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// ClusterManagerInterface 集群管理器接口
type ClusterManagerInterface interface {
	GetCluster(clusterID string) (*ClusterInfo, error)
	ListClusters() []*ClusterInfo
	AddCluster(name, description, kubeconfigContent string, labels map[string]string) (*ClusterInfo, error)
	RemoveCluster(clusterID string) error
	SetDefaultCluster(clusterID string) error
	UpdateClusterLabels(clusterID string, labels map[string]string) error
}

// Handler 集群管理处理器
type Handler struct {
	manager ClusterManagerInterface
}

// NewHandler 创建新的集群处理器
func NewHandler(manager *Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

// NewHandlerWithInterface 创建支持接口的集群处理器
func NewHandlerWithInterface(manager ClusterManagerInterface) *Handler {
	return &Handler{
		manager: manager,
	}
}

// ListClusters 列出所有集群
func (h *Handler) ListClusters(c *gin.Context) {
	clusters := h.manager.ListClusters()

	// 转换为响应格式，排除敏感信息
	response := make([]map[string]interface{}, 0, len(clusters))
	for _, cluster := range clusters {
		clusterData := map[string]interface{}{
			"id":          cluster.ID,
			"name":        cluster.Name,
			"description": cluster.Description,
			"server":      cluster.Server,
			"version":     cluster.Version,
			"status":      cluster.Status,
			"context":     cluster.Context,
			"labels":      cluster.Labels,
			"createdAt":   cluster.CreatedAt,
			"updatedAt":   cluster.UpdatedAt,
			"lastCheck":   cluster.LastCheck,
			"isDefault":   cluster.IsDefault,
		}
		response = append(response, clusterData)
	}

	c.JSON(http.StatusOK, gin.H{
		"clusters": response,
		"total":    len(response),
	})
}

// GetCluster 获取指定集群信息
func (h *Handler) GetCluster(c *gin.Context) {
	clusterID := c.Param("id")

	cluster, err := h.manager.GetCluster(clusterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"id":          cluster.ID,
		"name":        cluster.Name,
		"description": cluster.Description,
		"server":      cluster.Server,
		"version":     cluster.Version,
		"status":      cluster.Status,
		"context":     cluster.Context,
		"labels":      cluster.Labels,
		"createdAt":   cluster.CreatedAt,
		"updatedAt":   cluster.UpdatedAt,
		"lastCheck":   cluster.LastCheck,
		"isDefault":   cluster.IsDefault,
	}

	c.JSON(http.StatusOK, response)
}

// AddClusterRequest 添加集群请求
type AddClusterRequest struct {
	Name              string            `json:"name" binding:"required"`
	Description       string            `json:"description"`
	KubeconfigContent string            `json:"kubeconfigContent" binding:"required"`
	Labels            map[string]string `json:"labels"`
}

// AddCluster 添加新集群
func (h *Handler) AddCluster(c *gin.Context) {
	var req AddClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cluster, err := h.manager.AddCluster(req.Name, req.Description, req.KubeconfigContent, req.Labels)
	if err != nil {
		klog.Errorf("Failed to add cluster: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"id":          cluster.ID,
		"name":        cluster.Name,
		"description": cluster.Description,
		"server":      cluster.Server,
		"version":     cluster.Version,
		"status":      cluster.Status,
		"context":     cluster.Context,
		"labels":      cluster.Labels,
		"createdAt":   cluster.CreatedAt,
		"updatedAt":   cluster.UpdatedAt,
		"isDefault":   cluster.IsDefault,
	}

	c.JSON(http.StatusCreated, response)
}

// RemoveCluster 删除集群
func (h *Handler) RemoveCluster(c *gin.Context) {
	clusterID := c.Param("id")

	err := h.manager.RemoveCluster(clusterID)
	if err != nil {
		if err.Error() == "cluster "+clusterID+" not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cluster removed successfully"})
}

// SetDefaultCluster 设置默认集群
func (h *Handler) SetDefaultCluster(c *gin.Context) {
	clusterID := c.Param("id")

	err := h.manager.SetDefaultCluster(clusterID)
	if err != nil {
		if err.Error() == "cluster "+clusterID+" not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default cluster set successfully"})
}

// UpdateClusterLabelsRequest 更新集群标签请求
type UpdateClusterLabelsRequest struct {
	Labels map[string]string `json:"labels" binding:"required"`
}

// UpdateClusterLabels 更新集群标签
func (h *Handler) UpdateClusterLabels(c *gin.Context) {
	clusterID := c.Param("id")

	var req UpdateClusterLabelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.manager.UpdateClusterLabels(clusterID, req.Labels)
	if err != nil {
		if err.Error() == "cluster "+clusterID+" not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cluster labels updated successfully"})
}

// GetClusterStats 获取集群统计信息
func (h *Handler) GetClusterStats(c *gin.Context) {
	clusterID := c.Param("id")

	cluster, err := h.manager.GetCluster(clusterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if cluster.Client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cluster client not available"})
		return
	}

	ctx := c.Request.Context()

	// 获取节点信息
	nodes, err := cluster.Client.ClientSet.CoreV1().Nodes().List(ctx,
		metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nodes: " + err.Error()})
		return
	}

	// 获取Pod信息
	pods, err := cluster.Client.ClientSet.CoreV1().Pods("").List(ctx,
		metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pods: " + err.Error()})
		return
	}

	// 获取命名空间信息
	namespaces, err := cluster.Client.ClientSet.CoreV1().Namespaces().List(ctx,
		metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get namespaces: " + err.Error()})
		return
	}

	// 统计Ready节点数量
	readyNodes := 0
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				readyNodes++
				break
			}
		}
	}

	// 统计Running Pod数量
	runningPods := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			runningPods++
		}
	}

	stats := map[string]interface{}{
		"nodes": map[string]interface{}{
			"total": len(nodes.Items),
			"ready": readyNodes,
		},
		"pods": map[string]interface{}{
			"total":   len(pods.Items),
			"running": runningPods,
		},
		"namespaces": map[string]interface{}{
			"total": len(namespaces.Items),
		},
	}

	c.JSON(http.StatusOK, stats)
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	clusterGroup := group.Group("/clusters")
	{
		clusterGroup.GET("", h.ListClusters)
		clusterGroup.POST("", h.AddCluster)
		clusterGroup.GET("/:id", h.GetCluster)
		clusterGroup.DELETE("/:id", h.RemoveCluster)
		clusterGroup.PUT("/:id/default", h.SetDefaultCluster)
		clusterGroup.PUT("/:id/labels", h.UpdateClusterLabels)
		clusterGroup.GET("/:id/stats", h.GetClusterStats)
	}
}
