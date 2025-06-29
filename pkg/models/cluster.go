package models

import (
	"time"

	"gorm.io/gorm"
)

// ClusterModel 集群信息数据库模型
type ClusterModel struct {
	ID          string `gorm:"primaryKey;size:255" json:"id"`
	Name        string `gorm:"not null;size:255" json:"name"`
	Description string `gorm:"size:1000" json:"description,omitempty"`
	Server      string `gorm:"not null;size:500" json:"server"`
	Version     string `gorm:"size:50" json:"version,omitempty"`
	Status      string `gorm:"size:20;default:unknown" json:"status"`
	Context     string `gorm:"size:255" json:"context,omitempty"`
	Labels      string `gorm:"type:text" json:"labels,omitempty"` // JSON 字符串存储
	IsDefault   bool   `gorm:"default:false" json:"isDefault"`
	IsInCluster bool   `gorm:"default:false" json:"isInCluster"`

	// Kubeconfig 相关字段
	KubeconfigPath    string `gorm:"size:500" json:"kubeconfigPath,omitempty"`
	KubeconfigContent string `gorm:"type:text" json:"kubeconfigContent,omitempty"`

	// Prometheus 相关字段
	PrometheusURL      string `gorm:"size:500" json:"prometheusUrl,omitempty"`
	PrometheusUsername string `gorm:"size:255" json:"prometheusUsername,omitempty"`
	PrometheusPassword string `gorm:"size:255" json:"prometheusPassword,omitempty"`
	PrometheusEnabled  bool   `gorm:"default:false" json:"prometheusEnabled"`

	// 健康检查相关
	LastCheck time.Time `json:"lastCheck"`

	// 通用字段
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (ClusterModel) TableName() string {
	return "clusters"
}

// ClusterRepository 集群信息仓库接口
type ClusterRepository interface {
	// 基础 CRUD
	Create(cluster *ClusterModel) error
	GetByID(id string) (*ClusterModel, error)
	GetAll() ([]*ClusterModel, error)
	Update(cluster *ClusterModel) error
	Delete(id string) error

	// 业务方法
	GetDefault() (*ClusterModel, error)
	SetDefault(id string) error
	GetByContext(context string) (*ClusterModel, error)
	GetInCluster() (*ClusterModel, error)

	// 批量操作
	CreateBatch(clusters []*ClusterModel) error
	GetByLabels(labels map[string]string) ([]*ClusterModel, error)

	// Prometheus 相关方法
	UpdatePrometheusConfig(id string, url, username, password string, enabled bool) error
	GetClustersWithPrometheus() ([]*ClusterModel, error)
}

// ClusterRepositoryImpl 集群信息仓库实现
type ClusterRepositoryImpl struct {
	db *gorm.DB
}

// NewClusterRepository 创建集群信息仓库
func NewClusterRepository(db *gorm.DB) ClusterRepository {
	return &ClusterRepositoryImpl{db: db}
}

// Create 创建集群
func (r *ClusterRepositoryImpl) Create(cluster *ClusterModel) error {
	return r.db.Create(cluster).Error
}

// GetByID 根据ID获取集群
func (r *ClusterRepositoryImpl) GetByID(id string) (*ClusterModel, error) {
	var cluster ClusterModel
	err := r.db.Where("id = ?", id).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// GetAll 获取所有集群
func (r *ClusterRepositoryImpl) GetAll() ([]*ClusterModel, error) {
	var clusters []*ClusterModel
	err := r.db.Find(&clusters).Error
	return clusters, err
}

// Update 更新集群
func (r *ClusterRepositoryImpl) Update(cluster *ClusterModel) error {
	return r.db.Save(cluster).Error
}

// Delete 删除集群
func (r *ClusterRepositoryImpl) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&ClusterModel{}).Error
}

// GetDefault 获取默认集群
func (r *ClusterRepositoryImpl) GetDefault() (*ClusterModel, error) {
	var cluster ClusterModel
	err := r.db.Where("is_default = ?", true).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// SetDefault 设置默认集群
func (r *ClusterRepositoryImpl) SetDefault(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 清除所有默认标记
		if err := tx.Model(&ClusterModel{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		// 设置新的默认集群
		return tx.Model(&ClusterModel{}).Where("id = ?", id).Update("is_default", true).Error
	})
}

// GetByContext 根据上下文获取集群
func (r *ClusterRepositoryImpl) GetByContext(context string) (*ClusterModel, error) {
	var cluster ClusterModel
	err := r.db.Where("context = ?", context).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// GetInCluster 获取集群内配置
func (r *ClusterRepositoryImpl) GetInCluster() (*ClusterModel, error) {
	var cluster ClusterModel
	err := r.db.Where("is_in_cluster = ?", true).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// CreateBatch 批量创建集群
func (r *ClusterRepositoryImpl) CreateBatch(clusters []*ClusterModel) error {
	return r.db.Create(&clusters).Error
}

// GetByLabels 根据标签获取集群（简单实现，实际应该解析JSON）
func (r *ClusterRepositoryImpl) GetByLabels(labels map[string]string) ([]*ClusterModel, error) {
	var clusters []*ClusterModel
	query := r.db

	// 这里是简化实现，实际应该使用 JSON 查询
	for key, value := range labels {
		query = query.Where("labels LIKE ?", "%\""+key+"\":\""+value+"\"%")
	}

	err := query.Find(&clusters).Error
	return clusters, err
}

// UpdatePrometheusConfig 更新 Prometheus 配置
func (r *ClusterRepositoryImpl) UpdatePrometheusConfig(id string, url, username, password string, enabled bool) error {
	return r.db.Model(&ClusterModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"prometheus_url":      url,
		"prometheus_username": username,
		"prometheus_password": password,
		"prometheus_enabled":  enabled,
	}).Error
}

// GetClustersWithPrometheus 获取具有 Prometheus 配置的集群
func (r *ClusterRepositoryImpl) GetClustersWithPrometheus() ([]*ClusterModel, error) {
	var clusters []*ClusterModel
	err := r.db.Where("prometheus_enabled = ?", true).Find(&clusters).Error
	return clusters, err
}
