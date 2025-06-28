# 多集群管理功能

Nexus 支持管理多个 Kubernetes 集群，让您可以在一个界面中轻松切换和管理不同的集群环境。

## 功能特性

### 🚀 核心功能

- **自动集群发现**: 自动扫描并加载本地 kubeconfig 文件中的集群配置
- **实时健康监控**: 定期检查集群状态，提供实时的健康状态指示
- **无缝切换**: 在不同集群间快速切换，无需重启应用
- **集群标签管理**: 为集群添加自定义标签，便于分类和管理
- **统一操作界面**: 所有 Kubernetes 资源操作都支持多集群

### 📊 状态监控

- **健康状态**: 绿色(健康)、黄色(异常)、红色(不可达)、灰色(未知)
- **版本信息**: 显示 Kubernetes 集群版本
- **连接测试**: 自动检测集群连接状态
- **最后检查时间**: 显示最近一次健康检查的时间

## 使用指南

### 初始化设置

当您首次启动 Nexus 时，系统会自动进行以下操作：

1. **In-Cluster 检测**: 如果运行在 Kubernetes 集群内，自动添加当前集群
2. **Kubeconfig 扫描**: 扫描 `~/.kube/` 目录下的配置文件
3. **默认集群选择**: 自动选择默认集群或第一个可用集群

### 添加新集群

#### 方法一：通过界面添加

1. 点击顶部导航栏的集群选择器
2. 点击"添加集群"按钮
3. 填写集群信息：
   - **集群名称**: 为集群指定一个友好的名称
   - **描述**: 可选的集群描述信息
   - **Kubeconfig**: 上传文件或直接粘贴配置内容

#### 方法二：通过 API 添加

```bash
curl -X POST http://localhost:8080/api/v1/clusters \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "生产环境集群",
    "description": "生产环境 Kubernetes 集群",
    "kubeconfigContent": "YOUR_KUBECONFIG_CONTENT"
  }'
```

### 集群管理

#### 查看集群列表

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/clusters
```

#### 设置默认集群

```bash
curl -X PUT http://localhost:8080/api/v1/clusters/{cluster-id}/default \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### 删除集群

```bash
curl -X DELETE http://localhost:8080/api/v1/clusters/{cluster-id} \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 集群切换

#### 在界面中切换

1. 点击顶部导航栏的集群选择器
2. 从下拉列表中选择目标集群
3. 系统会自动切换到选定的集群

#### 通过 API 参数指定集群

在任何 API 调用中，您可以通过以下方式指定集群：

```bash
# 通过查询参数
curl "http://localhost:8080/api/v1/pods?cluster=cluster-id"

# 通过请求头
curl -H "X-Cluster-ID: cluster-id" \
  http://localhost:8080/api/v1/pods
```

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `KUBECONFIG` | `~/.kube/config` | Kubeconfig 文件路径 |
| `CLUSTER_HEALTH_CHECK_INTERVAL` | `30s` | 集群健康检查间隔 |

### 集群配置文件格式

支持标准的 Kubernetes kubeconfig 格式：

```yaml
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: LS0t...
    server: https://kubernetes.example.com:6443
  name: production-cluster
contexts:
- context:
    cluster: production-cluster
    user: admin
  name: production-context
current-context: production-context
users:
- name: admin
  user:
    client-certificate-data: LS0t...
    client-key-data: LS0t...
```

## 最佳实践

### 1. 集群命名规范

建议使用有意义的集群名称，例如：
- `生产环境-北京`
- `测试环境-开发团队`
- `预发布环境`

### 2. 标签管理

为集群添加标签以便分类管理：

```json
{
  "environment": "production",
  "region": "beijing",
  "team": "platform"
}
```

### 3. 权限控制

- 确保每个集群的 kubeconfig 具有适当的权限
- 定期轮换访问凭证
- 使用 RBAC 限制用户权限

### 4. 监控和告警

- 定期检查集群健康状态
- 设置集群不可达时的告警
- 监控集群资源使用情况

## 故障排除

### 常见问题

#### 1. 集群显示为"不可达"状态

**可能原因**:
- 网络连接问题
- 集群证书过期
- 集群 API Server 不可用

**解决方案**:
```bash
# 检查网络连接
kubectl --kubeconfig=/path/to/config cluster-info

# 验证证书有效性
kubectl --kubeconfig=/path/to/config auth can-i get pods

# 查看详细错误信息
kubectl --kubeconfig=/path/to/config get nodes -v=6
```

#### 2. 无法添加新集群

**可能原因**:
- Kubeconfig 格式错误
- 缺少必要的权限
- 证书配置问题

**解决方案**:
```bash
# 验证 kubeconfig 格式
kubectl --kubeconfig=/path/to/config config view

# 测试连接
kubectl --kubeconfig=/path/to/config get namespaces
```

#### 3. 集群列表为空

**可能原因**:
- Kubeconfig 文件不存在
- 权限不足
- 配置文件格式错误

**解决方案**:
```bash
# 检查文件是否存在
ls -la ~/.kube/config

# 检查文件权限
chmod 600 ~/.kube/config

# 验证配置
kubectl config view
```

### 日志调试

启用详细日志来调试集群连接问题：

```bash
# 启动时启用调试日志
./nexus --log-level=debug

# 或设置环境变量
export LOG_LEVEL=debug
./nexus
```

## API 参考

### 集群管理 API

#### 获取集群列表

```http
GET /api/v1/clusters
```

**响应示例**:
```json
{
  "clusters": [
    {
      "id": "cluster-1",
      "name": "生产环境集群",
      "description": "主要生产环境",
      "server": "https://k8s.example.com:6443",
      "version": "v1.28.0",
      "status": "healthy",
      "isDefault": true,
      "labels": {
        "environment": "production"
      },
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z",
      "lastCheck": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 1
}
```

#### 添加集群

```http
POST /api/v1/clusters
Content-Type: application/json

{
  "name": "新集群",
  "description": "描述信息",
  "kubeconfigContent": "kubeconfig内容",
  "labels": {
    "environment": "test"
  }
}
```

#### 获取集群详情

```http
GET /api/v1/clusters/{id}
```

#### 删除集群

```http
DELETE /api/v1/clusters/{id}
```

#### 设置默认集群

```http
PUT /api/v1/clusters/{id}/default
```

#### 更新集群标签

```http
PUT /api/v1/clusters/{id}/labels
Content-Type: application/json

{
  "labels": {
    "environment": "staging",
    "team": "devops"
  }
}
```

#### 获取集群统计信息

```http
GET /api/v1/clusters/{id}/stats
```

**响应示例**:
```json
{
  "nodes": {
    "total": 5,
    "ready": 5
  },
  "pods": {
    "total": 120,
    "running": 115
  },
  "namespaces": {
    "total": 15
  }
}
```

## 安全考虑

### 1. 凭证管理

- 使用最小权限原则配置 kubeconfig
- 定期轮换访问令牌
- 避免在配置中硬编码敏感信息

### 2. 网络安全

- 使用 TLS 加密所有集群通信
- 配置适当的网络策略
- 限制集群 API Server 的访问

### 3. 审计日志

- 启用 Kubernetes 审计日志
- 监控异常的 API 访问
- 记录集群切换操作

## 性能优化

### 集群连接池

为了提高性能，系统会为每个集群维护连接池：

```go
// 在 pkg/cluster/manager.go 中已实现
type ClusterInfo struct {
    Client *kube.K8sClient // 复用连接
}
```

### 缓存策略

- **集群状态缓存**: 30秒内复用健康检查结果
- **资源列表缓存**: 避免频繁请求 API Server
- **前端状态缓存**: localStorage 保存用户选择

### 批量操作

支持批量管理多个集群：

```bash
# 批量检查集群状态
curl -X POST http://localhost:8080/api/v1/clusters/batch/health-check \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"clusterIds": ["cluster1", "cluster2"]}'
```

## 高级配置

### 自定义健康检查

可以通过环境变量自定义健康检查行为：

```bash
# 设置检查间隔（默认30秒）
export CLUSTER_HEALTH_CHECK_INTERVAL=60s

# 设置超时时间（默认10秒）
export CLUSTER_HEALTH_CHECK_TIMEOUT=15s

# 禁用健康检查
export DISABLE_CLUSTER_HEALTH_CHECK=true
```

### 集群优先级

为集群设置优先级，影响默认选择和排序：

```json
{
  "labels": {
    "priority": "high",
    "environment": "production"
  }
}
```

### 网络配置

对于复杂网络环境，可以配置代理和超时：

```yaml
# kubeconfig 中的代理配置
clusters:
- cluster:
    server: https://kubernetes.example.com:6443
    proxy-url: http://proxy.example.com:8080
```

## 安全考虑

### 凭证管理

- **定期轮换**: 建议每90天轮换一次集群访问凭证
- **最小权限**: 为 nexus 创建专门的 ServiceAccount
- **审计日志**: 启用集群访问审计日志

### 网络安全

- **TLS 验证**: 始终验证集群证书
- **网络隔离**: 使用防火墙限制集群访问
- **VPN 连接**: 对于远程集群使用 VPN

### 示例 RBAC 配置

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nexus-readonly
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps", "extensions"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: nexus-readonly
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: nexus-readonly
subjects:
- kind: ServiceAccount
  name: nexus
  namespace: nexus-system
```

## 集成示例

### 与 CI/CD 集成

```bash
# 在 CI/CD 流水线中动态添加集群
curl -X POST "$NEXUS_API/clusters" \
  -H "Authorization: Bearer $NEXUS_TOKEN" \
  -d '{
    "name": "staging-'$BUILD_NUMBER'",
    "description": "Staging cluster for build '$BUILD_NUMBER'",
    "kubeconfigContent": "'$(cat $KUBECONFIG)'"
  }'
```

### 与监控系统集成

```bash
# 导出集群状态到 Prometheus
curl "$NEXUS_API/clusters" | jq -r '.clusters[] | 
  "nexus_cluster_status{cluster=\"\(.name)\",id=\"\(.id)\"} \(if .status == "healthy" then 1 else 0 end)"'
```

## 开发扩展

### 自定义集群提供商

可以扩展支持更多集群提供商：

```go
// pkg/cluster/providers/
type Provider interface {
    DiscoverClusters() ([]*ClusterInfo, error)
    ValidateConfig(config string) error
}
```

### Webhook 集成

支持集群状态变化的 Webhook 通知：

```bash
curl -X POST http://localhost:8080/api/v1/clusters/webhooks \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "url": "https://your-webhook.example.com/cluster-status",
    "events": ["cluster.healthy", "cluster.unreachable"]
  }'
```

## 更新日志

### v1.0.0
- 初始多集群支持
- 自动集群发现
- 基本的集群管理功能

### v1.1.0
- 添加集群健康监控
- 支持集群标签管理
- 改进用户界面

### v1.2.0
- 优化性能和稳定性
- 增强错误处理
- 添加更多 API 端点 
