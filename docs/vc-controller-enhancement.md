# VC-Controller 能力完善说明

## 概述

vc-controller 是 VC Stack 的控制平面核心组件，负责管理和协调整个云平台的各项服务。经过此次完善，新增了多个关键服务模块，使其功能更加完整和健全。

## 新增服务模块

### 1. 元数据服务 (Metadata Service)

**位置**: `internal/controlplane/metadata/`

**功能**:

- 提供实例元数据 API，类似于 AWS EC2 metadata 和 OpenStack metadata
- 支持 cloud-init 集成
- 提供 user-data、vendor-data 和 network-data

**API 端点**:

- `GET /latest/meta-data` - 获取实例元数据
- `GET /latest/meta-data/:key` - 获取特定元数据键
- `GET /latest/user-data` - 获取 cloud-init user-data
- `GET /latest/vendor-data` - 获取 vendor-data
- `POST /api/v1/metadata/instances` - 创建实例元数据
- `GET /api/v1/metadata/instances/:id` - 查询实例元数据
- `PUT /api/v1/metadata/instances/:id` - 更新实例元数据
- `DELETE /api/v1/metadata/instances/:id` - 删除实例元数据

**使用场景**:

- 虚拟机内部获取自身配置信息
- Cloud-init 初始化配置
- 动态配置和服务发现

### 2. 事件与审计服务 (Event Service)

**位置**: `internal/controlplane/event/`

**功能**:

- 系统事件记录和审计追踪
- 类似 OpenStack Panko 和 AWS CloudTrail
- 自动清理过期事件（默认保留 90 天）

**API 端点**:

- `POST /api/v1/events` - 创建事件记录
- `GET /api/v1/events` - 查询事件列表（支持多种过滤条件）
- `GET /api/v1/events/:id` - 获取特定事件
- `GET /api/v1/events/resource/:resource_type/:resource_id` - 获取资源相关事件
- `DELETE /api/v1/events/cleanup` - 手动触发清理

**特性**:

- 支持按资源类型、操作、状态、用户、租户等过滤
- 时间范围查询
- 分页支持
- 自动后台清理

**事件类型**:

- create, update, delete, action
- 资源类型: vm, network, volume, router, etc.
- 状态: success, failure, pending

### 3. 配额管理服务 (Quota Service)

**位置**: `internal/controlplane/quota/`

**功能**:

- 资源配额限制和使用统计
- 类似 OpenStack Nova/Cinder 配额系统
- 支持租户级别和全局默认配额

**API 端点**:

- `GET /api/v1/quotas/tenants/:tenant_id` - 获取租户配额
- `PUT /api/v1/quotas/tenants/:tenant_id` - 更新租户配额
- `DELETE /api/v1/quotas/tenants/:tenant_id` - 重置为默认配额
- `GET /api/v1/quotas/tenants/:tenant_id/usage` - 获取使用情况
- `GET /api/v1/quotas/defaults` - 获取默认配额
- `PUT /api/v1/quotas/defaults` - 更新默认配额

**配额项目**:

- instances - 实例数量
- vcpus - 虚拟CPU核心数
- ram_mb - 内存（MB）
- disk_gb - 磁盘空间（GB）
- volumes - 卷数量
- snapshots - 快照数量
- floating_ips - 浮动IP数量
- networks - 网络数量
- subnets - 子网数量
- routers - 路由器数量
- security_groups - 安全组数量

**默认配额**:

```
实例: 10
vCPUs: 20
内存: 50GB
磁盘: 1TB
其他资源: 10
```

**编程接口**:

```go
// 检查配额
err := quotaSvc.CheckQuota(tenantID, "instances", 1)

// 更新使用量
err := quotaSvc.UpdateUsage(tenantID, "instances", +1)  // 创建时 +1
err := quotaSvc.UpdateUsage(tenantID, "instances", -1)  // 删除时 -1
```

### 4. 监控与健康检查服务 (Monitoring Service)

**位置**: `internal/controlplane/monitoring/`

**功能**:

- 系统健康检查
- 性能指标收集
- Kubernetes 就绪性和存活性探针支持

**API 端点**:

- `GET /health` - 整体健康状态
- `GET /health/liveness` - Kubernetes 存活性探针
- `GET /health/readiness` - Kubernetes 就绪性探针
- `GET /health/details` - 详细健康信息
- `GET /metrics` - 系统指标
- `GET /metrics/system` - 系统资源使用情况
- `GET /api/v1/monitoring/status` - 组件状态

**健康检查项**:

- 数据库连接状态
- 连接延迟监控
- 连接池状态

**系统指标**:

- CPU 核心数
- Goroutine 数量
- 内存使用（已用/总量/百分比）
- 运行时间
- 启动时间

**健康状态**:

- healthy - 正常
- degraded - 降级（如高延迟）
- unhealthy - 不健康

### 5. 中间件系统 (Middleware)

**位置**: `internal/controlplane/middleware/`

**功能**:

- JWT 认证中间件
- 速率限制
- 请求 ID 追踪
- CORS 支持
- 日志记录
- 租户隔离
- 管理员权限检查

**中间件列表**:

1. **AuthMiddleware** - JWT 认证

   ```go
   router.Use(middleware.AuthMiddleware(jwtSecret, logger))
   ```

2. **RateLimitMiddleware** - 速率限制

   ```go
   router.Use(middleware.RateLimitMiddleware(10.0, 20)) // 10 req/s, burst 20
   ```

3. **RequestIDMiddleware** - 请求 ID

   ```go
   router.Use(middleware.RequestIDMiddleware())
   ```

4. **CORSMiddleware** - CORS 支持

   ```go
   router.Use(middleware.CORSMiddleware())
   ```

5. **LoggingMiddleware** - 请求日志

   ```go
   router.Use(middleware.LoggingMiddleware(logger))
   ```

6. **TenantIsolationMiddleware** - 租户隔离

   ```go
   router.Use(middleware.TenantIsolationMiddleware())
   ```

7. **AdminOnlyMiddleware** - 管理员权限

   ```go
   adminRoutes.Use(middleware.AdminOnlyMiddleware())
   ```

## 现有服务模块

### 1. 身份认证服务 (Identity Service)

- JWT 认证
- RBAC 权限控制
- 用户和角色管理
- 项目/租户管理
- LDAP/OIDC 集成支持

### 2. 网络服务 (Network Service)

- 虚拟网络管理
- 子网管理
- 路由器管理
- 浮动 IP
- 安全组
- 支持多种网络类型（flat, vlan, vxlan, gre, geneve）
- OVN SDN 集成

### 3. 主机管理服务 (Host Service)

- 计算节点注册
- 心跳监控
- 资源容量管理
- 主机状态管理
- 维护模式

### 4. 调度服务 (Scheduler Service)

- 节点注册和心跳
- 资源调度算法
- VM 分发
- 节点选择策略

### 5. 网关服务 (Gateway Service)

- API 网关
- 请求路由和代理
- 服务发现

## 架构优势

### 1. 模块化设计

每个服务都是独立的模块，具有清晰的职责边界，便于维护和扩展。

### 2. 统一的错误处理

所有服务都遵循统一的错误处理和响应格式。

### 3. 完整的可观测性

- 健康检查
- 性能监控
- 事件审计
- 请求追踪

### 4. 安全性

- JWT 认证
- RBAC 权限控制
- 租户隔离
- 速率限制
- 审计日志

### 5. 云原生支持

- Kubernetes 健康探针
- 优雅关闭
- 指标导出
- 分布式追踪就绪

## 使用示例

### 启动控制器

```bash
# 使用环境变量配置
export DB_HOST=localhost
export DB_NAME=vcstack
export DB_USER=vcstack
export DB_PASS=password
export VC_CONTROLLER_PORT=8080

# 启动控制器
./bin/vc-controller
```

### 健康检查

```bash
# 检查整体健康
curl http://localhost:8080/health

# Kubernetes liveness probe
curl http://localhost:8080/health/liveness

# Kubernetes readiness probe
curl http://localhost:8080/health/readiness

# 获取系统指标
curl http://localhost:8080/metrics
```

### 配额管理

```bash
# 获取租户配额
curl http://localhost:8080/api/v1/quotas/tenants/tenant-123

# 更新配额
curl -X PUT http://localhost:8080/api/v1/quotas/tenants/tenant-123 \
  -H "Content-Type: application/json" \
  -d '{
    "instances": 20,
    "vcpus": 40,
    "ram_mb": 102400,
    "disk_gb": 2000
  }'

# 查看使用情况
curl http://localhost:8080/api/v1/quotas/tenants/tenant-123/usage
```

### 事件查询

```bash
# 查询所有事件
curl http://localhost:8080/api/v1/events

# 按资源类型过滤
curl "http://localhost:8080/api/v1/events?resource_type=vm&status=success"

# 获取资源的所有事件
curl http://localhost:8080/api/v1/events/resource/vm/vm-123
```

### 元数据服务

```bash
# 在虚拟机内访问元数据（模拟）
curl "http://169.254.169.254/latest/meta-data?instance_id=vm-123"

# 获取 user-data
curl "http://169.254.169.254/latest/user-data?instance_id=vm-123"
```

## 数据库迁移

新增服务需要以下数据库表：

```sql
-- 元数据表
CREATE TABLE instance_metadata (
    id SERIAL PRIMARY KEY,
    instance_id VARCHAR(255) UNIQUE NOT NULL,
    hostname VARCHAR(255),
    user_data TEXT,
    metadata JSONB,
    vendor_data TEXT,
    network_data TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 事件表
CREATE TABLE system_events (
    id VARCHAR(36) PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255),
    resource_type VARCHAR(50) NOT NULL,
    action VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    user_id VARCHAR(255),
    tenant_id VARCHAR(255),
    request_id VARCHAR(255),
    source_ip VARCHAR(50),
    user_agent TEXT,
    details JSONB,
    error_message TEXT,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_resource_type ON system_events(resource_type);
CREATE INDEX idx_events_resource_id ON system_events(resource_id);
CREATE INDEX idx_events_action ON system_events(action);
CREATE INDEX idx_events_status ON system_events(status);
CREATE INDEX idx_events_user_id ON system_events(user_id);
CREATE INDEX idx_events_tenant_id ON system_events(tenant_id);
CREATE INDEX idx_events_timestamp ON system_events(timestamp);

-- 配额表
CREATE TABLE quota_sets (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) UNIQUE NOT NULL,
    instances INTEGER DEFAULT -1,
    vcpus INTEGER DEFAULT -1,
    ram_mb INTEGER DEFAULT -1,
    disk_gb INTEGER DEFAULT -1,
    volumes INTEGER DEFAULT -1,
    snapshots INTEGER DEFAULT -1,
    floating_ips INTEGER DEFAULT -1,
    networks INTEGER DEFAULT -1,
    subnets INTEGER DEFAULT -1,
    routers INTEGER DEFAULT -1,
    security_groups INTEGER DEFAULT -1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 配额使用表
CREATE TABLE quota_usage (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) UNIQUE NOT NULL,
    instances INTEGER DEFAULT 0,
    vcpus INTEGER DEFAULT 0,
    ram_mb INTEGER DEFAULT 0,
    disk_gb INTEGER DEFAULT 0,
    volumes INTEGER DEFAULT 0,
    snapshots INTEGER DEFAULT 0,
    floating_ips INTEGER DEFAULT 0,
    networks INTEGER DEFAULT 0,
    subnets INTEGER DEFAULT 0,
    routers INTEGER DEFAULT 0,
    security_groups INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 下一步计划

### 短期目标

1. 添加编排服务（类似 OpenStack Heat）
2. 工作流引擎（类似 Mistral）
3. 消息队列集成（RocketMQ）
4. 分布式追踪（Jaeger）

### 中期目标

1. DNS 服务（类似 Designate）
2. 负载均衡服务（类似 Octavia）
3. 告警服务（类似 Aodh）
4. 密钥管理（类似 Barbican）

### 长期目标

1. 容器服务（类似 Magnum）
2. 裸金属服务（类似 Ironic）
3. 应用目录（类似 Murano）
4. 数据库即服务（类似 Trove）

## 总结

通过此次完善，vc-controller 已经具备了以下核心能力：

✅ **完整的认证授权体系** - Identity + 中间件
✅ **资源配额管理** - Quota Service
✅ **事件审计追踪** - Event Service
✅ **健康监控** - Monitoring Service
✅ **元数据服务** - Metadata Service
✅ **网络服务** - Network Service
✅ **主机管理** - Host Service
✅ **智能调度** - Scheduler Service
✅ **API 网关** - Gateway Service

这些服务共同构成了一个功能完整、架构清晰、易于扩展的云平台控制面，为 VC Stack 的持续发展奠定了坚实基础。
