package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/ysicing/nexus/pkg/common"
	"github.com/ysicing/nexus/pkg/handlers/resources"
	"github.com/ysicing/nexus/pkg/kube"
	"k8s.io/klog/v2"
)

type WebhookHandler struct {
	k8sClient *kube.K8sClient
}

func NewWebhookHandler(k8sClient *kube.K8sClient) *WebhookHandler {
	return &WebhookHandler{
		k8sClient: k8sClient,
	}
}

func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	var body common.WebhookRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": "Invalid request body " + err.Error(),
		})
		return
	}
	klog.V(2).Infof("Received webhook request: %+v", body)
	switch body.Action {
	case common.ActionRestart:
		handler, err := resources.GetHandler(body.Resource)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "Invalid resource type",
			})
			return
		}
		if restartable, ok := handler.(resources.Restartable); ok {
			ctx := c.Request.Context()
			if err := restartable.Restart(ctx, body.Namespace, body.Name); err != nil {
				c.JSON(500, gin.H{
					"error": "Failed to restart resource: " + err.Error(),
				})
				return
			}
			c.JSON(200, gin.H{
				"message": "Resource restarted successfully",
			})
			return
		}
	case common.ActionUpdateImage:
	default:
		c.JSON(400, gin.H{
			"error": "Invalid action",
		})
	}
}
