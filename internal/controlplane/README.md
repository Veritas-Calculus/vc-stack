# VC-Controller 增强功能

## 📋 概述

本次更新为 vc-controller 添加了多个核心服务模块，大幅提升了控制平面的能力和完整性。

## ✨ 新增功能

### 1. 🏷️ 元数据服务 (Metadata Service)

提供类似 AWS EC2 和 OpenStack 的实例元数据服务：

- **EC2 兼容 API**: `/latest/meta-data`, `/latest/user-data`
- **Cloud-init 支持**: 自动配置虚拟机
- **动态配置**: 运行时获取实例信息

```bash
# 在虚拟机内访问
curl http://169.254.169.254/latest/meta-data?instance_id=vm-123
curl http://169.254.169.254/latest/user-data?instance_id=vm-123
```

### 2. 📝 事件审计服务 (Event Service)

完整的审计追踪和事件日志系统：

- **审计日志**: 记录所有资源操作
- **多维度查询**: 按资源、用户、时间等过滤
- **自动清理**: 90 天保留策略
- **合规性**: 满足审计要求

```bash
# 查询事件
curl "http://localhost:8080/api/v1/events?resource_type=vm&status=success"
curl "http://localhost:8080/api/v1/events/resource/vm/vm-123"
```

### 3. 💹 配额管理服务 (Quota Service)

灵活的资源配额管理系统：

- **多维度配额**: 实例、CPU、内存、磁盘等
- **租户隔离**: 独立的配额限制
- **使用统计**: 实时资源使用情况
- **配额检查**: 创建资源前自动检查

```bash
# 查看配额
curl http://localhost:8080/api/v1/quotas/tenants/tenant-123

# 更新配额
curl -X PUT http://localhost:8080/api/v1/quotas/tenants/tenant-123 \
  -H "Content-Type: application/json" \
  -d '{"instances": 20, "vcpus": 40}'
```

**配额项目**:

- instances (实例数)
- vcpus (虚拟CPU)
- ram_mb (内存)
- disk_gb (磁盘)
- volumes (卷)
- snapshots (快照)
- floating_ips (浮动IP)
- networks (网络)
- subnets (子网)
- routers (路由器)
- security_groups (安全组)

### 4. 🏥 监控健康检查服务 (Monitoring Service)

全面的健康监控和指标收集：

- **健康检查**: `/health`, `/health/liveness`, `/health/readiness`
- **系统指标**: CPU、内存、Goroutines
- **组件状态**: 数据库、服务状态
- **K8s 集成**: 支持 liveness 和 readiness 探针

```bash
# 健康检查
curl http://localhost:8080/health
curl http://localhost:8080/health/readiness

# 系统指标
curl http://localhost:8080/metrics
```

### 5. 🛡️ 中间件系统 (Middleware)

完整的HTTP中间件栈：

- **JWT 认证**: 安全的API访问控制
- **速率限制**: 防止API滥用
- **请求追踪**: Request ID 支持
- **CORS**: 跨域支持
- **日志记录**: 结构化请求日志
- **租户隔离**: 多租户数据隔离
- **权限控制**: 管理员权限检查

## 🏗️ 架构改进

### 服务组成

```
vc-controller
├── Identity Service      (身份认证)
├── Network Service       (网络管理)
├── Host Service          (主机管理)
├── Scheduler Service     (资源调度)
├── Gateway Service       (API网关)
├── Metadata Service      (元数据服务) ✨ NEW
├── Event Service         (事件审计) ✨ NEW
├── Quota Service         (配额管理) ✨ NEW
└── Monitoring Service    (健康监控) ✨ NEW
```

### 技术栈

- **语言**: Go 1.21+
- **框架**: Gin (HTTP), GORM (ORM)
- **数据库**: PostgreSQL 15+
- **认证**: JWT
- **日志**: Zap

## 🚀 快速开始

### 1. 应用数据库迁移

```bash
# 设置数据库连接
export DB_HOST=localhost
export DB_NAME=vcstack
export DB_USER=vcstack
export DB_PASS=vcstack

# 运行迁移
./scripts/migrate-controller-enhancements.sh
```

### 2. 启动控制器

```bash
# 配置环境变量
export VC_CONTROLLER_PORT=8080

# 启动
./bin/vc-controller
```

### 3. 验证服务

```bash
# 健康检查
curl http://localhost:8080/health

# 查看所有路由
curl http://localhost:8080/api/v1/quotas/defaults
curl http://localhost:8080/api/v1/events
curl http://localhost:8080/metrics
```

## 📊 API 端点总览

### 元数据服务

- `GET /latest/meta-data`
- `GET /latest/user-data`
- `POST /api/v1/metadata/instances`
- `GET /api/v1/metadata/instances/:id`

### 事件服务

- `POST /api/v1/events`
- `GET /api/v1/events`
- `GET /api/v1/events/:id`
- `GET /api/v1/events/resource/:type/:id`

### 配额服务

- `GET /api/v1/quotas/tenants/:tenant_id`
- `PUT /api/v1/quotas/tenants/:tenant_id`
- `GET /api/v1/quotas/tenants/:tenant_id/usage`
- `GET /api/v1/quotas/defaults`

### 监控服务

- `GET /health`
- `GET /health/liveness`
- `GET /health/readiness`
- `GET /metrics`

## 🔒 安全特性

### 认证授权

- ✅ JWT Token 认证
- ✅ RBAC 权限控制
- ✅ 租户隔离
- ✅ 管理员权限检查

### 审计追踪

- ✅ 完整的操作日志
- ✅ 用户行为追踪
- ✅ 资源变更记录
- ✅ 错误和异常记录

### 资源管理

- ✅ 配额限制
- ✅ 使用统计
- ✅ 配额超限保护
- ✅ 租户级别隔离

## 📈 监控运维

### Kubernetes 部署

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: vc-controller
    image: vc-controller:latest
    livenessProbe:
      httpGet:
        path: /health/liveness
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /health/readiness
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
```

### Prometheus 指标

```yaml
- job_name: 'vc-controller'
  static_configs:
  - targets: ['vc-controller:8080']
  metrics_path: '/metrics'
```

## 🗄️ 数据库表结构

### 新增表

1. **instance_metadata** - 实例元数据
2. **system_events** - 系统事件日志
3. **quota_sets** - 配额限制
4. **quota_usage** - 配额使用

详见 `migrations/002_add_metadata_event_quota_tables.sql`

## 📚 文档

- [完整功能说明](docs/vc-controller-enhancement.md)
- [API 文档](docs/api/)
- [架构设计](docs/architecture/)

## 🎯 下一步计划

### 短期

- [ ] 编排服务 (Orchestration)
- [ ] 工作流引擎 (Workflow)
- [ ] 消息队列集成 (RocketMQ)
- [ ] 分布式追踪 (Jaeger)

### 中期

- [ ] DNS 服务
- [ ] 负载均衡服务
- [ ] 告警服务
- [ ] 密钥管理服务

### 长期

- [ ] 容器服务
- [ ] 裸金属服务
- [ ] 应用目录
- [ ] DBaaS 服务

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📝 更新日志

### v1.1.0 (2025-12-08)

**新增功能**:

- ✨ 元数据服务 - 支持 cloud-init 和实例配置
- ✨ 事件审计服务 - 完整的操作审计追踪
- ✨ 配额管理服务 - 灵活的资源配额控制
- ✨ 监控健康检查 - 系统健康和性能监控
- ✨ 中间件系统 - 认证、限流、日志等

**改进**:

- 🎨 模块化架构 - 更清晰的服务边界
- 🔒 增强安全性 - 完整的认证授权体系
- 📊 可观测性 - 健康检查、指标、审计
- ☸️ 云原生支持 - K8s 探针、优雅关闭

## 📄 License

Apache 2.0
