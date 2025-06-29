# Nexus 数据库迁移指南

本文档指导如何将 Nexus 从内存存储的集群管理迁移到数据库持久化存储。

## 迁移概述

### 当前架构（内存存储）

```go
// 使用内存存储的集群管理器
clusterManager := cluster.NewManager()
clusterManager.Initialize()
```

### 目标架构（数据库存储）

```go
// 使用数据库存储的集群管理器
db := database.NewDatabase(config)
clusterManager := cluster.NewManagerWithDB(db)
clusterManager.Initialize()
```

## 迁移步骤

### 第一步：准备数据库环境

#### 选择数据库类型

根据你的部署环境选择合适的数据库：

- **SQLite**: 适合单实例部署、开发环境
- **MySQL**: 适合生产环境、高并发场景
- **PostgreSQL**: 适合企业级部署、复杂查询需求

#### 配置环境变量

```bash
# SQLite 配置（推荐用于开始迁移）
export DB_DRIVER=sqlite
export SQLITE_PATH=/var/lib/nexus/clusters.db

# MySQL 配置
export DB_DRIVER=mysql
export DB_HOST=mysql-server
export DB_PORT=3306
export DB_NAME=nexus
export DB_USER=nexus_user
export DB_PASSWORD=secure_password

# PostgreSQL 配置
export DB_DRIVER=postgresql
export DB_HOST=postgres-server
export DB_PORT=5432
export DB_NAME=nexus
export DB_USER=nexus_user
export DB_PASSWORD=secure_password
```

### 第二步：备份现有集群配置

在迁移前，建议备份当前的集群配置：

```bash
# 导出当前集群信息（如果有 API 接口）
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/v1/clusters > clusters_backup.json

# 或者备份 kubeconfig 文件
cp -r ~/.kube ~/.kube.backup
```

### 第三步：更新主程序代码

修改 `main.go` 文件，集成数据库支持：

```go
package main

import (
    // ... 其他导入
    "github.com/ysicing/nexus/pkg/database"
)

func main() {
    // ... 现有代码 ...

    // 初始化数据库（新增）
    dbConfig := database.GetDefaultConfig()
    db, err := database.NewDatabase(dbConfig)
    if err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer db.Close()

    // 执行数据库迁移（新增）
    if err := db.MigrateDatabase(); err != nil {
        log.Fatalf("Failed to migrate database: %v", err)
    }

    // 替换集群管理器初始化
    // 旧代码：
    // clusterManager := cluster.NewManager()
    
    // 新代码：
    clusterManager := cluster.NewManagerWithDB(db)
    
    if err := clusterManager.Initialize(); err != nil {
        log.Fatalf("Failed to initialize cluster manager: %v", err)
    }
    defer clusterManager.Stop()

    // ... 其余代码保持不变 ...
}
```

### 第四步：测试迁移

#### 4.1 本地测试

```bash
# 使用 SQLite 进行本地测试
export DB_DRIVER=sqlite
export SQLITE_PATH=./test_migration.db

# 启动应用
go run main.go
```

#### 4.2 验证集群加载

检查应用日志，确认集群加载过程：

```
正在初始化集群管理器...
正在从数据库加载集群配置...
从数据库加载了 0 个集群
正在检查集群内配置...
正在扫描本地 kubeconfig 文件...
成功加载集群: minikube (minikube)
成功加载集群: production (production-cluster)
集群管理器初始化完成，共加载 2 个集群
```

#### 4.3 验证数据持久化

```bash
# 重启应用，检查集群是否从数据库恢复
# 第一次启动
go run main.go
# Ctrl+C 停止

# 第二次启动，应该看到从数据库加载集群
go run main.go
```

### 第五步：生产环境部署

#### 5.1 数据库准备

**MySQL 示例：**

```sql
-- 创建数据库
CREATE DATABASE nexus CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建用户
CREATE USER 'nexus_user'@'%' IDENTIFIED BY 'secure_password';
GRANT ALL PRIVILEGES ON nexus.* TO 'nexus_user'@'%';
FLUSH PRIVILEGES;
```

**PostgreSQL 示例：**

```sql
-- 创建数据库
CREATE DATABASE nexus;

-- 创建用户
CREATE USER nexus_user WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE nexus TO nexus_user;
```

#### 5.2 环境变量配置

创建环境配置文件 `.env.production`：

```bash
# 数据库配置
DB_DRIVER=mysql
DB_HOST=mysql.internal
DB_PORT=3306
DB_NAME=nexus
DB_USER=nexus_user
DB_PASSWORD=secure_password
DB_SSL_MODE=require

# 连接池配置
DB_MAX_IDLE_CONNS=10
DB_MAX_OPEN_CONNS=50
DB_CONN_MAX_LIFETIME=3600

# 其他 Nexus 配置
PORT=8080
# ... 其他配置
```

#### 5.3 部署脚本更新

更新 Kubernetes 部署文件或 Docker Compose：

```yaml
# Kubernetes ConfigMap 示例
apiVersion: v1
kind: ConfigMap
metadata:
  name: nexus-config
data:
  DB_DRIVER: "mysql"
  DB_HOST: "mysql-service"
  DB_PORT: "3306"
  DB_NAME: "nexus"

---
apiVersion: v1
kind: Secret
metadata:
  name: nexus-db-secret
type: Opaque
stringData:
  DB_USER: "nexus_user"
  DB_PASSWORD: "secure_password"
```

## 迁移验证清单

### 功能验证

- [ ] 应用启动时从数据库加载集群配置
- [ ] 新添加的集群保存到数据库
- [ ] 集群状态更新持久化到数据库
- [ ] 默认集群设置正确保存和恢复
- [ ] 集群标签和描述信息完整保存

### 性能验证

- [ ] 应用启动时间在可接受范围内
- [ ] 集群列表查询响应时间正常
- [ ] 数据库连接池配置合理
- [ ] 内存使用量在预期范围内

### 可靠性验证

- [ ] 数据库连接失败时应用能优雅降级
- [ ] 应用重启后集群配置完整恢复
- [ ] 并发操作不会导致数据不一致
- [ ] 数据库事务正确处理

## 回滚计划

如果迁移过程中遇到问题，可以按以下步骤回滚：

### 快速回滚

```go
// 在 main.go 中临时禁用数据库
// clusterManager := cluster.NewManagerWithDB(db)
clusterManager := cluster.NewManager()  // 回滚到内存存储
```

### 完整回滚

1. 恢复原始的 `main.go` 文件
2. 恢复 kubeconfig 备份文件
3. 重启应用服务
4. 验证集群功能正常

## 故障排除

### 常见问题

#### 1. 数据库连接失败

```
Error: failed to initialize database: dial tcp: connection refused
```

**解决方案：**
- 检查数据库服务是否运行
- 验证网络连接和防火墙设置
- 确认数据库连接参数正确

#### 2. 集群配置丢失

```
Warning: 从数据库加载了 0 个集群
```

**解决方案：**
- 检查数据库迁移是否成功执行
- 验证表结构是否正确创建
- 检查数据库权限设置

#### 3. 性能问题

```
Warning: 集群列表查询耗时过长
```

**解决方案：**
- 调整数据库连接池配置
- 添加适当的数据库索引
- 考虑使用缓存机制

### 调试工具

```bash
# 检查数据库连接
mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p$DB_PASSWORD $DB_NAME

# 查看集群表数据
SELECT id, name, server, is_default FROM clusters;

# 检查应用日志
tail -f /var/log/nexus/application.log | grep -i cluster
```

## 最佳实践

1. **渐进式迁移**: 先在测试环境验证，再逐步推广到生产环境
2. **数据备份**: 定期备份数据库和配置文件
3. **监控告警**: 设置数据库连接和查询性能监控
4. **文档更新**: 及时更新运维文档和故障处理手册

## 后续优化

迁移完成后，可以考虑以下优化：

1. **添加数据库索引** - 提高查询性能
2. **实现读写分离** - 支持高并发场景
3. **添加缓存层** - 减少数据库查询压力
4. **实现数据归档** - 管理历史数据
5. **添加监控指标** - 完善可观测性 
