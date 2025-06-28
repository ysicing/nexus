package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/ysicing/nexus/pkg/cluster"
	"github.com/ysicing/nexus/pkg/kube"
	"k8s.io/klog/v2"
)

// ClusterHandler 集群处理器
type ClusterHandler struct {
	manager *cluster.Manager
}

// NewClusterHandler 创建新的集群处理器
func NewClusterHandler(manager *cluster.Manager) *ClusterHandler {
	return &ClusterHandler{
		manager: manager,
	}
}

// GetClusterClient 从请求中获取集群客户端
func (h *ClusterHandler) GetClusterClient(c *gin.Context) (*kube.K8sClient, error) {
	clusterID := c.Query("cluster")
	if clusterID == "" {
		clusterID = c.GetHeader("X-Cluster-ID")
	}

	var clusterInfo *cluster.ClusterInfo
	var err error

	if clusterID != "" {
		clusterInfo, err = h.manager.GetCluster(clusterID)
		if err != nil {
			return nil, err
		}
	} else {
		// 使用默认集群
		clusterInfo, err = h.manager.GetDefaultCluster()
		if err != nil {
			return nil, err
		}
	}

	if clusterInfo.Client == nil {
		return nil, fmt.Errorf("cluster client not available for cluster: %s", clusterInfo.Name)
	}

	return clusterInfo.Client, nil
}

// ClusterMiddleware 集群中间件，自动注入集群客户端
func (h *ClusterHandler) ClusterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		client, err := h.GetClusterClient(c)
		if err != nil {
			klog.Warningf("Failed to get cluster client: %v", err)
			// 不阻止请求，让处理器自己处理没有客户端的情况
			c.Set("k8sClient", nil)
		} else {
			// 将客户端存储在上下文中
			c.Set("k8sClient", client)
		}
		c.Next()
	}
}

// GetK8sClientFromContext 从gin上下文中获取K8s客户端
func GetK8sClientFromContext(c *gin.Context) (*kube.K8sClient, bool) {
	client, exists := c.Get("k8sClient")
	if !exists {
		return nil, false
	}

	k8sClient, ok := client.(*kube.K8sClient)
	return k8sClient, ok
}
