# Nexus 数据库配置指南

> **✅ 实现状态**: 已完成数据库集成功能，支持 SQLite、MySQL、PostgreSQL，使用 DSN 方式配置连接。

本文档介绍如何为 Nexus 项目配置数据库存储，实现集群信息的持久化管理。

## 概述

Nexus 支持将集群信息存储在数据库中，提供以下优势：

- **持久化存储**: 集群配置在应用重启后自动恢复
- **多种数据库**: 支持 SQLite、MySQL、PostgreSQL
- **DSN 配置**: 使用标准 DSN 格式简化配置
- **自动发现**: 智能加载集群配置的四步机制
- **高可用**: 数据库级别的集群信息备份和恢复

## 支持的数据库

### SQLite（默认）
- **优点**: 无需额外配置，适合开发和小规模部署
- **缺点**: 不支持并发写入，单文件存储
- **使用场景**: 开发环境、单实例部署

### MySQL
- **优点**: 成熟稳定，支持高并发，丰富的管理工具
- **缺点**: 需要额外的数据库服务器
- **使用场景**: 生产环境、多实例部署

### PostgreSQL
- **优点**: 功能强大，支持复杂查询，数据一致性好
- **缺点**: 配置相对复杂
- **使用场景**: 企业级部署、复杂查询需求

## 环境变量配置

### 基础配置

```bash
# 数据库连接字符串（DSN 格式）
DATABASE_DSN="sqlite:./data/nexus.db"  # 默认 SQLite

# 连接池配置
DB_MAX_IDLE_CONNS=10
DB_MAX_OPEN_CONNS=100
DB_CONN_MAX_LIFETIME=3600  # 秒
```

## DSN 配置格式

### SQLite 配置

```bash
# 最简配置 - 使用默认 SQLite
# 无需设置任何环境变量，默认使用 sqlite:./data/nexus.db

# 自定义 SQLite 路径
export DATABASE_DSN="sqlite:/var/lib/nexus/database.db"
export DATABASE_DSN="sqlite:./custom/path/nexus.db"
```

### MySQL 配置

```bash
# 标准 MySQL DSN 格式
export DATABASE_DSN="mysql://nexus_user:secure_password@tcp(mysql.example.com:3306)/nexus?charset=utf8mb4&parseTime=True&loc=Local"

# 本地 MySQL
export DATABASE_DSN="mysql://root:password@tcp(localhost:3306)/nexus?charset=utf8mb4&parseTime=True&loc=Local"

# 带 SSL 的 MySQL
export DATABASE_DSN="mysql://nexus_user:secure_password@tcp(mysql.example.com:3306)/nexus?charset=utf8mb4&parseTime=True&loc=Local&tls=true"
```

### PostgreSQL 配置

```bash
# 标准 PostgreSQL DSN 格式
export DATABASE_DSN="postgres://nexus_user:secure_password@postgres.example.com:5432/nexus?sslmode=require"

# 本地 PostgreSQL
export DATABASE_DSN="postgres://postgres:password@localhost:5432/nexus?sslmode=disable"

# 带更多参数的 PostgreSQL
export DATABASE_DSN="postgres://nexus_user:secure_password@postgres.example.com:5432/nexus?sslmode=require&timezone=UTC"
```

## 集群加载机制

Nexus 参考 k8m 项目设计，实现了智能的四步集群加载机制：

### 第一步：从数据库加载集群（ScanClustersInDB）

```go
// 从数据库恢复已保存的集群配置
clusters := loadClustersFromDB()
```

- 加载所有已保存的集群配置
- 恢复集群状态和标签信息
- 重新建立 Kubernetes 客户端连接
- 保持默认集群设置

### 第二步：注册集群内配置（RegisterInCluster）

```go
// 如果运行在 Kubernetes 集群内，注册当前集群
if inClusterConfig := rest.InClusterConfig(); inClusterConfig != nil {
    registerCurrentCluster(inClusterConfig)
}
```

- 检测是否运行在 Kubernetes 集群内
- 自动注册当前集群配置
- 优先设置为默认集群（如果是第一个集群）

### 第三步：扫描本地配置文件（ScanClustersInDir）

```go
// 扫描 ~/.kube/ 目录下的配置文件
scanKubeconfigDirectory("~/.kube/")
```

- 扫描 `~/.kube/` 目录下的所有 kubeconfig 文件
- 解析多个上下文和集群配置
- 自动跳过已存在的集群
- 测试连接可用性

### 第四步：确保默认集群

```go
// 确保至少有一个默认集群
ensureDefaultCluster()
```

- 检查是否已设置默认集群
- 如果没有，选择第一个可用集群作为默认
- 更新数据库中的默认集群标记

## 数据库表结构

### clusters 表

```sql
CREATE TABLE clusters (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(1000),
    server VARCHAR(500) NOT NULL,
    version VARCHAR(50),
    status VARCHAR(20) DEFAULT 'unknown',
    context VARCHAR(255),
    labels TEXT,  -- JSON 格式存储
    is_default BOOLEAN DEFAULT FALSE,
    is_in_cluster BOOLEAN DEFAULT FALSE,
    kubeconfig_path VARCHAR(500),
    kubeconfig_content TEXT,
    
    -- Prometheus 相关字段
    prometheus_url VARCHAR(500),
    prometheus_username VARCHAR(255),
    prometheus_password VARCHAR(255),
    prometheus_enabled BOOLEAN DEFAULT FALSE,
    
    last_check TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_clusters_is_default ON clusters(is_default);
CREATE INDEX idx_clusters_is_in_cluster ON clusters(is_in_cluster);
CREATE INDEX idx_clusters_status ON clusters(status);
CREATE INDEX idx_clusters_prometheus_enabled ON clusters(prometheus_enabled);
```

## Prometheus 集成

Nexus 支持为每个集群单独配置 Prometheus，实现集群级别的监控数据收集。

### Prometheus 配置字段

- **prometheus_url**: Prometheus 服务器地址
- **prometheus_username**: 认证用户名（可选）
- **prometheus_password**: 认证密码（可选）
- **prometheus_enabled**: 是否启用 Prometheus

### 配置示例

#### 通过 API 配置

```bash
# 为集群配置 Prometheus
curl -X PUT "http://localhost:8080/api/v1/clusters/my-cluster/prometheus" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://prometheus.example.com:9090",
    "username": "admin",
    "password": "secret",
    "enabled": true
  }'
```

#### 通过数据库直接配置

```sql
-- 为集群启用 Prometheus
UPDATE clusters SET 
  prometheus_url = 'http://prometheus.example.com:9090',
  prometheus_username = 'admin',
  prometheus_password = 'secret',
  prometheus_enabled = true
WHERE id = 'my-cluster';
```

### 查询 Prometheus 配置

```sql
-- 查看所有启用 Prometheus 的集群
SELECT id, name, prometheus_url, prometheus_enabled 
FROM clusters 
WHERE prometheus_enabled = true;

-- 查看特定集群的 Prometheus 配置
SELECT prometheus_url, prometheus_username, prometheus_enabled 
FROM clusters 
WHERE id = 'my-cluster';
```

### 安全注意事项

1. **密码保护**: Prometheus 密码以明文存储，建议使用强密码
2. **网络安全**: 确保 Prometheus 服务器的网络访问安全
3. **权限控制**: 为 Prometheus 用户配置最小必要权限

## 使用示例

### 启动 Nexus

```bash
# 使用默认 SQLite
./nexus

# 使用 MySQL
DATABASE_DSN="mysql://user:password@tcp(localhost:3306)/nexus?charset=utf8mb4&parseTime=True&loc=Local" ./nexus

# 使用 PostgreSQL
DATABASE_DSN="postgres://user:password@localhost:5432/nexus?sslmode=disable" ./nexus

# 使用环境文件
source .env && ./nexus
```

### 查看集群信息

```bash
# 列出所有集群
curl http://localhost:8080/api/v1/clusters

# 获取默认集群
curl http://localhost:8080/api/v1/clusters/default

# 查看集群详情
curl http://localhost:8080/api/v1/clusters/in-cluster
```

### 添加新集群

```bash
curl -X POST http://localhost:8080/api/v1/clusters \
  -H "Content-Type: application/json" \
  -d '{
    "name": "生产集群",
    "description": "生产环境 Kubernetes 集群",
    "kubeconfigContent": "...",
    "labels": {
      "environment": "production",
      "region": "us-west-2"
    }
  }'
```

## 数据库迁移

### 自动迁移

Nexus 启动时会自动执行数据库迁移：

```go
// 应用启动时自动执行
db.MigrateDatabase()
```

### 手动迁移

如果需要手动控制迁移过程：

```bash
# 备份现有数据
mysqldump nexus > nexus_backup.sql

# 启动 Nexus 进行迁移
./nexus --migrate-only

# 验证迁移结果
mysql nexus -e "SHOW TABLES;"
```

## 故障排除

### 常见问题

#### 1. 数据库连接失败

```bash
# 检查数据库服务状态
systemctl status mysql

# 测试数据库连接
mysql -h $DB_HOST -u $DB_USER -p$DB_PASSWORD $DB_NAME

# 检查防火墙设置
telnet $DB_HOST $DB_PORT
```

#### 2. 集群加载失败

```bash
# 检查 kubeconfig 文件权限
ls -la ~/.kube/config

# 验证集群连接
kubectl cluster-info

# 查看 Nexus 日志
tail -f /var/log/nexus.log
```

#### 3. 默认集群未设置

```bash
# 手动设置默认集群
curl -X PUT http://localhost:8080/api/v1/clusters/in-cluster/default

# 检查数据库中的默认集群
mysql nexus -e "SELECT id, name, is_default FROM clusters WHERE is_default = TRUE;"
```

### 日志分析

启用详细日志记录：

```bash
# 设置日志级别
export LOG_LEVEL=debug

# 启动 Nexus
./nexus --v=4
```

关键日志信息：

```
INFO: 开始初始化集群管理器...
INFO: 正在从数据库加载集群配置...
INFO: 从数据库加载了 2 个集群
INFO: 正在检查集群内配置...
INFO: 成功注册集群内配置
INFO: 正在扫描本地 kubeconfig 文件...
INFO: 扫描了 3 个 kubeconfig 文件
INFO: 默认集群已设置: in-cluster
INFO: 集群管理器初始化完成，共加载 5 个集群
```

## 性能优化

### 连接池配置

```bash
# 根据并发需求调整连接池
export DB_MAX_IDLE_CONNS=20
export DB_MAX_OPEN_CONNS=200
export DB_CONN_MAX_LIFETIME=7200
```

### 索引优化

```sql
-- 为常用查询添加索引
CREATE INDEX idx_clusters_labels ON clusters(labels);
CREATE INDEX idx_clusters_server ON clusters(server);
CREATE INDEX idx_clusters_updated_at ON clusters(updated_at);
```
