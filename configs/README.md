# VC Stack 配置文件目录

本目录包含 VC Stack 的所有配置文件模板和示例。

## 📁 目录结构

```
configs/
├── vc-controller.yaml.example      # ✅ Controller 完整配置模板
├── vc-node.yaml.example            # ✅ Node 完整配置模板
├── docker-compose.yaml.example     # ✅ Docker Compose 部署
├── env/                            # ✅ 环境变量配置
│   ├── controller.env.example      #    Controller 环境变量
│   └── node.env.example            #    Node 环境变量
├── systemd/                        # ✅ Systemd 服务文件
│   ├── vc-controller.service       #    Controller 服务
│   └── vc-node.service             #    Node 服务
├── compute.yaml                    # ⚠️  已废弃 (使用 vc-node.yaml)
├── identity.yaml                   # ⚠️  已废弃 (使用 vc-controller.yaml)
├── network.yaml                    # ⚠️  已废弃 (使用 vc-controller.yaml)
├── lite.yaml                       # ⚠️  已废弃 (使用 vc-node.yaml)
├── config.yaml                     # ⚠️  已废弃 (已拆分)
├── lite-with-agent.yaml.example    # ℹ️  参考用 (Agent 配置示例)
└── netplugin.yaml.example          # ℹ️  参考用 (网络插件配置)
```

## 🚀 快速开始

### 1. Controller 部署

```bash
# 复制配置模板
cp configs/vc-controller.yaml.example /etc/vc-stack/controller.yaml

# 编辑配置（修改数据库密码、JWT secret 等）
vim /etc/vc-stack/controller.yaml

# 启动 Controller
./bin/vc-controller --config /etc/vc-stack/controller.yaml
```

### 2. Node 部署

```bash
# 复制配置模板
cp configs/vc-node.yaml.example /etc/vc-stack/node.yaml

# 编辑配置（修改 controller_url、节点名称等）
vim /etc/vc-stack/node.yaml

# 启动 Node
sudo ./bin/vc-node --config /etc/vc-stack/node.yaml
```

### 3. 使用环境变量 (可选)

```bash
# Controller
cp configs/env/controller.env.example /etc/vc-stack/controller.env
vim /etc/vc-stack/controller.env
source /etc/vc-stack/controller.env
./bin/vc-controller

# Node
cp configs/env/node.env.example /etc/vc-stack/node.env
vim /etc/vc-stack/node.env
source /etc/vc-stack/node.env
sudo ./bin/vc-node
```

### 4. 使用 Systemd (生产环境推荐)

```bash
# 安装服务文件
sudo cp configs/systemd/vc-controller.service /etc/systemd/system/
sudo cp configs/systemd/vc-node.service /etc/systemd/system/

# 配置环境变量
sudo cp configs/env/controller.env.example /etc/vc-stack/controller.env
sudo cp configs/env/node.env.example /etc/vc-stack/node.env

# 编辑配置
sudo vim /etc/vc-stack/controller.env
sudo vim /etc/vc-stack/node.env

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable vc-controller vc-node
sudo systemctl start vc-controller vc-node

# 查看状态
sudo systemctl status vc-controller
sudo systemctl status vc-node
```

### 5. 使用 Docker Compose (开发环境)

```bash
# 复制配置
cp configs/docker-compose.yaml.example docker-compose.yaml

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f vc-controller
```

## 📋 配置文件说明

### ✅ 新架构配置 (当前使用)

#### vc-controller.yaml.example

**用途**: VC Stack Controller 完整配置
**包含服务**: Gateway, Identity, Network, Scheduler
**关键配置**:

- 数据库连接 (PostgreSQL)
- JWT 认证配置
- OVN 网络配置
- 调度器策略
- 日志和监控

**最小配置示例**:

```yaml
database:
  host: localhost
  name: vcstack
  username: vcstack
  password: vcstack123

identity:
  jwt:
    secret: your-secret-key
```

#### vc-node.yaml.example

**用途**: VC Stack Node 完整配置
**包含服务**: Compute, Lite Agent, Network Plugin
**关键配置**:

- 节点标识和标签
- Agent 自动注册
- Libvirt/KVM 配置
- 存储后端配置
- 网络插件配置

**最小配置示例**:

```yaml
agent:
  enabled: true
  controller_url: http://controller:8080

libvirt:
  uri: qemu:///system

storage:
  default_backend: local
```

#### env/controller.env.example

**用途**: Controller 环境变量配置
**适用场景**: 容器化部署、CI/CD、简化配置
**包含**: 数据库、JWT、OVN、日志等核心配置

#### env/node.env.example

**用途**: Node 环境变量配置
**适用场景**: 容器化部署、批量节点部署
**包含**: Agent、Libvirt、存储等核心配置

#### systemd/vc-controller.service

**用途**: Controller systemd 服务文件
**特点**:

- 依赖管理 (PostgreSQL)
- 自动重启
- 资源限制
- 日志集成

#### systemd/vc-node.service

**用途**: Node systemd 服务文件
**特点**:

- 依赖管理 (Libvirtd)
- Root 权限运行
- 自动重启

#### docker-compose.yaml.example

**用途**: 开发环境一键部署
**包含服务**:

- PostgreSQL 数据库
- Redis (可选)
- VC Controller
- OVN NB/SB (可选)
- Prometheus (可选)
- Grafana (可选)

### ⚠️ 已废弃配置 (不再使用)

这些配置文件对应的独立服务已经合并到 `vc-controller` 和 `vc-node`：

- **compute.yaml** → 迁移到 `vc-node.yaml`
- **identity.yaml** → 迁移到 `vc-controller.yaml` (identity 段)
- **network.yaml** → 迁移到 `vc-controller.yaml` (network 段)
- **lite.yaml** → 迁移到 `vc-node.yaml`
- **config.yaml** → 已拆分到 controller 和 node 配置

### ℹ️ 参考配置

- **lite-with-agent.yaml.example**: Agent 自动注册配置参考
- **netplugin.yaml.example**: 网络插件配置参考

## 🔑 重要配置项

### Controller 必须配置

| 配置项 | 说明 | 默认值 | 必须修改 |
|--------|------|--------|---------|
| `database.host` | PostgreSQL 地址 | localhost | ❌ |
| `database.password` | 数据库密码 | vcstack123 | ✅ 生产环境 |
| `identity.jwt.secret` | JWT 密钥 | - | ✅ 生产环境 |
| `server.port` | API 端口 | 8080 | ❌ |

### Node 必须配置

| 配置项 | 说明 | 默认值 | 必须修改 |
|--------|------|--------|---------|
| `agent.controller_url` | Controller 地址 | - | ✅ |
| `node.name` | 节点名称 | 主机名 | 推荐 |
| `libvirt.uri` | Libvirt 连接 | qemu:///system | ❌ |
| `storage.default_backend` | 存储类型 | local | ❌ |

## 🔐 安全建议

### 生产环境必须修改的配置

```yaml
# vc-controller.yaml
identity:
  jwt:
    secret: <生成 64 位随机字符串>  # 使用 openssl rand -hex 32

database:
  password: <强密码>  # 使用复杂密码

identity:
  default_admin:
    password: <修改默认密码>  # 不要使用 admin123
```

### 文件权限

```bash
# 设置配置文件权限
sudo chmod 600 /etc/vc-stack/*.yaml
sudo chmod 600 /etc/vc-stack/*.env
sudo chown vcstack:vcstack /etc/vc-stack/*
```

### 生成安全密钥

```bash
# 生成 JWT secret
openssl rand -hex 32

# 生成强密码
openssl rand -base64 32
```

## 📊 配置优先级

配置加载顺序（后面的会覆盖前面的）：

1. 程序默认值
2. 配置文件 (`--config` 参数)
3. 环境变量 (`DATABASE_HOST` 等)
4. 命令行参数

示例：

```bash
# 配置文件 + 环境变量组合
export DATABASE_PASSWORD=secret
./bin/vc-controller --config /etc/vc-stack/controller.yaml
```

## 🌐 不同场景的配置

### 开发环境 (单机)

```bash
# 使用默认配置 + 环境变量
export DATABASE_HOST=localhost
export AGENT_CONTROLLER_URL=http://localhost:8080
./bin/vc-controller &
sudo ./bin/vc-node &
```

### 测试环境 (小规模)

```bash
# 使用配置文件
./bin/vc-controller --config configs/vc-controller.yaml.example
sudo ./bin/vc-node --config configs/vc-node.yaml.example
```

### 生产环境 (推荐 systemd)

```bash
# 使用 systemd 服务
sudo systemctl start vc-controller
sudo systemctl start vc-node
```

## 🔍 验证配置

### 检查配置语法

```bash
# 使用 yamllint
yamllint configs/vc-controller.yaml.example
yamllint configs/vc-node.yaml.example

# 使用 Python
python3 -c "import yaml; yaml.safe_load(open('configs/vc-controller.yaml.example'))"
```

### 测试连接

```bash
# 测试数据库连接
psql -h localhost -U vcstack -d vcstack -c "SELECT 1;"

# 测试 Controller API
curl http://localhost:8080/health

# 测试 Metrics
curl http://localhost:9090/metrics
```

## 📚 相关文档

- [快速部署指南](../docs/QUICK-DEPLOY-NEW-ARCH.md) - 完整部署流程
- [配置指南](../docs/CONFIGURATION-GUIDE.md) - 详细配置说明
- [多节点部署](../docs/MULTI-NODE-DEPLOYMENT.md) - 多节点集群部署
- [CLI 迁移指南](../docs/CLI-MIGRATION-GUIDE.md) - 命令行工具使用

## ❓ 常见问题

### Q: 应该使用配置文件还是环境变量？

**A**: 推荐使用配置文件作为基础配置，环境变量覆盖敏感信息（如密码）。

### Q: 如何从旧配置迁移？

**A**: 参考 [配置指南](../docs/CONFIGURATION-GUIDE.md) 中的迁移章节。

### Q: 生产环境推荐哪种部署方式？

**A**: 推荐使用 systemd 管理服务，配置文件 + 环境变量的组合方式。

### Q: 配置文件放在哪里？

**A**: 推荐放在 `/etc/vc-stack/` 目录，这是 systemd 服务的默认路径。

## 🔄 配置更新

修改配置后需要重启服务：

```bash
# Systemd 方式
sudo systemctl restart vc-controller
sudo systemctl restart vc-node

# 直接运行方式
# 先停止进程，再用新配置启动
```

## 💡 提示

- 🔒 生产环境必须修改所有默认密码和密钥
- 📝 使用配置管理工具（Ansible/Terraform）管理多节点配置
- 🔄 定期备份配置文件
- 📊 启用监控收集配置变更历史
- 🧪 在测试环境验证配置后再应用到生产环境
