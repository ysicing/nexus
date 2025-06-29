# Nexus 示例

本目录包含了 Nexus 项目的使用示例。

## 📋 示例列表

### 1. 数据库集成示例 (`database_example.go`)

演示如何使用数据库存储集群配置：

```bash
# 设置数据库连接
export DATABASE_DSN="sqlite:./data/nexus.db"

# 运行示例
go run examples/database_example.go
```

### 2. Prometheus 集成示例 (`prometheus_example.go`)

演示如何在数据库中配置 Prometheus：

```bash
# 设置数据库连接
export DATABASE_DSN="sqlite:./data/nexus.db"

# 运行示例
go run examples/prometheus_example.go
```

## 🔄 迁移指南

### 从环境变量迁移到数据库配置

**旧方式（已废弃）：**
```bash
export PROMETHEUS_URL="http://prometheus:9090"
```

**新方式（推荐）：**
1. 配置数据库连接：
   ```bash
   export DATABASE_DSN="sqlite:./data/nexus.db"
   ```

2. 通过 API 或示例程序配置 Prometheus：
   ```bash
   go run examples/prometheus_example.go
   ```

3. 或者通过 Web 界面在集群管理页面配置 Prometheus

## 🎯 核心优势

### 数据库存储方案
- ✅ **持久化存储**：配置不会因重启丢失
- ✅ **多集群支持**：每个集群独立的 Prometheus 配置
- ✅ **动态配置**：无需重启即可更新配置
- ✅ **统一管理**：集群和监控配置统一存储

### 环境变量方案（已废弃）
- ❌ **单一配置**：只能配置一个全局 Prometheus
- ❌ **静态配置**：需要重启才能更新
- ❌ **配置分散**：集群和监控配置分离

## 🚀 快速开始

1. **启动 Nexus（使用数据库）：**
   ```bash
   export DATABASE_DSN="sqlite:./data/nexus.db"
   go run main.go
   ```

2. **添加集群和 Prometheus 配置：**
   ```bash
   go run examples/prometheus_example.go
   ```

3. **访问 Web 界面：**
   ```
   http://localhost:8080
   ```

## 💡 注意事项

- `PROMETHEUS_URL` 环境变量已被废弃，但仍向后兼容
- 建议使用数据库配置方式以获得更好的功能支持
- 数据库配置支持认证（用户名/密码）
- 支持多种数据库：SQLite、MySQL、PostgreSQL

## 示例说明

### database_example.go

这个示例演示了：

1. **数据库配置初始化** - 如何使用 DSN 方式配置数据库连接
2. **集群管理器集成** - 如何使用带数据库支持的集群管理器
3. **四步加载机制** - 参考 k8m 项目的集群加载策略
4. **集群持久化** - 集群信息的数据库存储和恢复
5. **动态集群管理** - 添加、删除、更新集群配置

### prometheus_example.go

这个示例演示了：

1. **Prometheus 数据库集成** - 如何将 Prometheus 配置存储到数据库
2. **集群级别配置** - 每个集群独立的 Prometheus 配置
3. **认证支持** - 支持用户名密码认证
4. **动态配置管理** - 运行时更新和重新加载配置
5. **健康检查** - Prometheus 连接状态检查

## 运行示例

### 前提条件

确保你已经安装了 Go 1.19+ 并且 Nexus 项目的依赖已经正确安装。

### 基本运行

```bash
# 在项目根目录下运行
go run examples/database_example.go

# 运行 Prometheus 集成示例
go run examples/prometheus_example.go
```

### 使用不同数据库

#### SQLite（默认）

```bash
# 使用默认 SQLite 配置
go run examples/database_example.go

# 自定义 SQLite 路径
DATABASE_DSN="sqlite:/tmp/nexus_demo.db" go run examples/database_example.go
```

#### MySQL

```bash
# 设置 MySQL DSN
export DATABASE_DSN="mysql://nexus_user:your_password@tcp(localhost:3306)/nexus_demo?charset=utf8mb4&parseTime=True&loc=Local"

# 运行示例
go run examples/database_example.go
```

#### PostgreSQL

```bash
# 设置 PostgreSQL DSN
export DATABASE_DSN="postgres://nexus_user:your_password@localhost:5432/nexus_demo?sslmode=disable"

# 运行示例
go run examples/database_example.go
```

## DSN 配置格式

### SQLite
```
sqlite:./data/nexus.db
sqlite:/var/lib/nexus/database.db
```

### MySQL
```
mysql://username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
mysql://nexus_user:secure_password@tcp(mysql.example.com:3306)/nexus?charset=utf8mb4&parseTime=True&loc=Local
```

### PostgreSQL
```
postgres://username:password@host:port/database?sslmode=disable
postgres://nexus_user:secure_password@postgres.example.com:5432/nexus?sslmode=require
```

## 示例输出

运行示例后，你将看到类似以下的输出：

```
=== Nexus 数据库集成示例 ===

1. 初始化数据库配置...
数据库 DSN: sqlite:./data/nexus.db

2. 初始化数据库连接...
Initializing SQLite database at: ./data/nexus.db
Database initialized successfully

3. 执行数据库迁移...
Database migrations completed successfully

4. 初始化集群管理器...
正在初始化集群管理器...
正在从数据库加载集群配置...
从数据库加载了 0 个集群
正在检查集群内配置...
未检测到集群内配置: unable to load in-cluster configuration
正在扫描本地 kubeconfig 文件...
集群管理器初始化完成，共加载 2 个集群

5. 集群加载结果:
总共加载了 2 个集群:
  [1] minikube (minikube)
      服务器: https://127.0.0.1:59478
      状态: unknown
      ✓ 默认集群

  [2] kind-kind (kind-kind)
      服务器: https://127.0.0.1:59479
      状态: unknown

6. 默认集群信息:
   默认集群: minikube (minikube)
   服务器: https://127.0.0.1:59478
   版本: v1.28.3

7. 演示添加新集群...
   成功添加集群: 演示集群 (demo-cluster-12345)

8. 更新后的集群列表:
总共 3 个集群:
  [1] minikube (minikube)
      ✓ 默认集群
  [2] kind-kind (kind-kind)
  [3] 演示集群 (demo-cluster-12345)

9. 数据库持久化测试:
   集群信息已保存到数据库
   重启应用后，集群配置将自动恢复

10. 健康检查演示 (等待 5 秒)...

=== 示例完成 ===
数据库集成功能已成功演示!

支持的 DSN 格式:
  SQLite:     sqlite:./data/nexus.db
  MySQL:      mysql://user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
  PostgreSQL: postgres://user:password@localhost:5432/dbname?sslmode=disable
```

## 关键特性演示

### 1. 四步加载机制

示例展示了参考 k8m 项目的集群加载策略：

1. **从数据库加载** - 恢复已保存的集群配置
2. **注册集群内配置** - 自动检测并注册当前集群
3. **扫描本地配置** - 扫描 ~/.kube/ 目录下的配置文件
4. **确保默认集群** - 自动设置第一个可用集群为默认

### 2. 数据库持久化

- 集群信息自动保存到数据库
- 应用重启后自动恢复集群配置
- 支持集群状态和标签的持久化

### 3. 动态集群管理

- 运行时添加新集群
- 更新集群标签和描述
- 设置默认集群
- 删除不需要的集群

## 环境变量配置

```bash
# 数据库连接字符串
DATABASE_DSN="sqlite:./data/nexus.db"

# 连接池配置
DB_MAX_IDLE_CONNS=10
DB_MAX_OPEN_CONNS=100
DB_CONN_MAX_LIFETIME=3600  # 秒
```

## 注意事项

1. **数据库驱动**: 现在使用真实的 GORM 数据库驱动，支持 SQLite、MySQL、PostgreSQL

2. **生产环境配置**: 在生产环境中，请确保：
   - 数据库服务器的可用性和性能
   - 适当的连接池配置
   - 数据库备份和恢复策略
   - 安全的数据库连接配置

3. **DSN 安全性**: 在生产环境中，避免在命令行中直接暴露数据库密码，建议使用环境变量或配置文件

## 下一步

- 添加数据库连接池监控
- 实现集群配置的导入/导出功能
- 添加集群配置的版本控制
- 实现数据库连接的健康检查 
