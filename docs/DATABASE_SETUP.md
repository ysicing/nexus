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
    last_check TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_clusters_is_default ON clusters(is_default);
CREATE INDEX idx_clusters_is_in_cluster ON clusters(is_in_cluster);
CREATE INDEX idx_clusters_status ON clusters(status);
```

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

### 定期清理

```sql
-- 清理软删除的记录
DELETE FROM clusters WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - INTERVAL 30 DAY;

-- 更新统计信息
ANALYZE TABLE clusters;
```

## 安全考虑

### 数据库安全

1. **访问控制**: 使用专用数据库用户，限制权限
2. **网络安全**: 配置防火墙，限制数据库访问
3. **传输加密**: 启用 SSL/TLS 连接
4. **数据加密**: 考虑数据库级别的加密

### 敏感信息保护

```bash
# 使用环境变量而非命令行参数
export DB_PASSWORD=secure_password

# 设置文件权限
chmod 600 .env

# 使用密钥管理服务
export DB_PASSWORD=$(aws secretsmanager get-secret-value --secret-id nexus-db-password --query SecretString --output text)
```

## 备份和恢复

### 自动备份

```bash
#!/bin/bash
# backup_nexus.sh
DATE=$(date +%Y%m%d_%H%M%S)
mysqldump nexus > "nexus_backup_${DATE}.sql"
aws s3 cp "nexus_backup_${DATE}.sql" s3://nexus-backups/
```

### 恢复数据

```bash
# 从备份恢复
mysql nexus < nexus_backup_20240101_120000.sql

# 重启 Nexus 服务
systemctl restart nexus
```

## 最佳实践

1. **环境隔离**: 不同环境使用不同的数据库
2. **监控告警**: 设置数据库连接和性能监控
3. **定期备份**: 建立自动化备份策略
4. **版本控制**: 跟踪数据库模式变更
5. **容量规划**: 根据集群数量规划存储容量

## 相关文档

- [多集群管理](MULTI_CLUSTER.md)
- [API 文档](../api/)
- [部署指南](../deploy/)

---

如有问题，请查看 [FAQ](../FAQ.md) 或提交 [Issue](https://github.com/ysicing/nexus/issues)。 
