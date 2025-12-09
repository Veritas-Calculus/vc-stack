# VC Stack

<div align="center">

![Version](https://img.shields.io/badge/version-v1.0.0--dev-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![React](https://img.shields.io/badge/React-18+-61DAFB?logo=react)

todo:

- 计算
  - cloud init
  - 调整配置
  - 扩容
  - 配置模板
  - 加密计算
    - tpm
  - UEFI VM
- 监控
  - perf 性能检测
  - prometheus 指标
  - grafana面板
- 网络
  - ovn 网络
  - LB
- 认证
  - oidc
  - 本地认证
  - 密码重置
  - OTP
- 部署
  - helm charts
  - k8s deployment
  - docker compose
  - 二进制部署脚本
  - deb包
  - gentoo overlays
  - 前端包特别要注意是全离线 方便后续管理和维护
- cicd
  - ✅ build
  - ✅ test
  - ✅ release
  - ✅ pre commit 配置
  - ✅ 全静态编译
  - ✅ 代码扫描 (SonarQube)
  - ✅ sentry 错误追踪
  - ✅ 代码质量检查 (golangci-lint)
  - ✅ 安全扫描 (gosec)
- 存储
  - s3 共享存储 -> ceph rgw
- 计量
- 安全
  - Key管理
  - 加密包
- Orchestration
  - 告警
  - workflow
- web console
- document
- IaC
  - terraform

OKR：

  1. 创建虚拟机并启动
  2. 完成简单的网络配置
  3. ssh key注入
  4. cloud init 支持
  5. 网络拓扑
  6. webshell 登陆

  openQA自动化测试

**现代化的开源 IaaS 云平台**

*类似于 OpenStack 但更加轻量、易用、现代化的基础设施即服务平台*

</div>

## 📖 项目简介

VC Stack 是一个现代化的开源 IaaS（Infrastructure as a Service）平台，旨在提供比 OpenStack 更简洁、更易用的云基础设施管理解决方案。它采用云原生架构设计，支持多云管理，为企业和开发者提供完整的虚拟化基础设施服务。

## ✨ 核心特性

### 🚀 部署与管理

- **快速部署**：支持 Kubernetes 和 Ansible 自动化部署
- **多云管理**：统一管理多个云平台资源
- **现代化 Dashboard**：基于 React 的直观管理界面
- **Infrastructure as Code**：完整的 Terraform 支持

### 💻 计算服务

- **多虚拟化支持**：
  - **KVM 虚拟机**：完整的虚拟机生命周期管理
  - **LXC 容器**：轻量级容器化解决方案
- **裸金属支持**：集成 Ironic 服务，支持物理机管理
- **原生 ISO 启动**：支持自定义镜像和系统安装
- **虚拟机高可用**：自动故障转移和恢复机制
- **AI 训练扩展平台**：GPU 资源调度和 AI 工作负载优化

### 🌐 网络服务

- **多种网络模型**：支持扁平网络、VLAN、VXLAN 等
- **负载均衡服务**：类似 Octavia 的 L4/L7 负载均衡
- **DNS 服务**：类似 Designate 的域名管理和解析
- **软件定义网络**：灵活的网络虚拟化和策略管理

### 💾 存储服务

- **Ceph 分布式存储**：高可用的后端存储集群
- **镜像服务**：类似 Glance 的虚拟机镜像管理
- **多存储类型**：块存储、对象存储、文件存储全支持

### 🔐 安全与认证

- **多重认证系统**：
  - 内建基础认证
  - LDAP/Active Directory 集成
  - SSO 单点登录支持
- **RBAC 权限控制**：基于角色的细粒度权限管理
- **密钥管理**：类似 Barbican 的密钥和证书服务
- **API 安全**：统一的 API 网关和访问控制

### 📊 监控与运维

- **全面监控**：
  - Prometheus 指标收集
  - 节点和虚拟机监控
  - 性能指标和资源使用统计
- **日志管理**：
  - 集中式日志聚合
  - 实时日志查询和分析
  - 日志归档和检索
- **链路追踪**：分布式系统调用链追踪和性能分析
- **告警管理**：类似 Aodh 的智能告警和通知服务
- **工作流引擎**：类似 Mistral 的自动化任务编排
- **资源计费**：详细的资源使用统计和成本分析

## 🏗️ 技术架构

### 前端技术栈

```
React.js 18+          // 现代化用户界面框架
TypeScript            // 类型安全的 JavaScript
Ant Design            // 企业级 UI 组件库
Redux Toolkit          // 可预测的状态管理
React Query           // 服务端状态管理和缓存
Vite                  // 快速的前端构建工具
```

### 后端技术栈

```
Golang 1.21+         // 高性能后端开发语言
Gin Framework        // 轻量级 Web 框架
GORM                 // 强大的 Go ORM 库
gRPC                 // 高性能 RPC 框架
Protocol Buffers     // 高效的数据序列化
Viper                // 灵活的配置管理
Cobra                // 现代化的 CLI 应用框架
```

### 数据存储层

```
PostgreSQL 15+       // 主要关系型数据库
Redis 7+             // 内存缓存和会话存储
InfluxDB 2.x         // 时序数据库（监控指标）
MinIO                // 高性能对象存储
ETCD 3.5+            // 分布式键值存储和服务发现
```

### 消息与通信

```
RocketMQ 5.x         // 高可靠消息队列中间件
WebSocket            // 实时双向通信
Server-Sent Events   // 服务端推送事件
```

### 基础设施组件

```
Kubernetes 1.28+     // 容器编排平台
Docker/Containerd    // 容器运行时
Prometheus           // 监控和告警系统
Grafana             // 数据可视化和仪表板
Jaeger              // 分布式链路追踪
ELK/EFK Stack       // 日志收集、存储和分析
```

## 🔧 核心组件架构

| 组件 | 功能描述 | 对应 OpenStack 服务 | 技术栈 |
|------|----------|---------------------|--------|
| **vc-compute** | 计算资源管理 | Nova | Go + gRPC + libvirt |
| **vc-network** | 网络服务管理 | Neutron | Go + OpenVSwitch + iptables |
| **vc-storage** | 存储服务管理 | Cinder | Go + Ceph + iSCSI |
| **vc-image** | 镜像服务管理 | Glance | Go + MinIO + qemu-img |
| **vc-identity** | 身份认证服务 | Keystone | Go + JWT + LDAP |
| **vc-dashboard** | Web 管理界面 | Horizon | React + TypeScript + Ant Design |
| **vc-orchestration** | 资源编排服务 | Heat | Go + Terraform |
| **vc-workflow** | 工作流引擎 | Mistral | Go + Temporal |
| **vc-dns** | DNS 服务 | Designate | Go + PowerDNS |
| **vc-loadbalancer** | 负载均衡服务 | Octavia | Go + HAProxy + Nginx |
| **vc-telemetry** | 遥测和监控 | Ceilometer/Aodh | Go + Prometheus + InfluxDB |
| **vc-secrets** | 密钥管理服务 | Barbican | Go + Vault + HSM |
| **vc-gateway** | API 网关 | - | Go + Kong + JWT |
| **vc-ai** | AI 训练平台 | - | Go + CUDA + TensorFlow |

## 🚀 快速开始

### 系统要求

#### 最低配置

- **操作系统**：Ubuntu 20.04+ / CentOS 8+ / RHEL 8+
- **CPU**：8 核心
- **内存**：16 GB
- **存储**：100 GB SSD
- **网络**：千兆网络接口

#### 推荐配置

- **操作系统**：Ubuntu 22.04 LTS
- **CPU**：16+ 核心
- **内存**：32+ GB
- **存储**：500+ GB NVMe SSD
- **网络**：万兆网络接口

### 安装部署

#### 方式一：一键部署脚本

```bash
# 下载安装脚本
curl -fsSL https://get.vc-stack.org | bash

# 或者使用 wget
wget -O- https://get.vc-stack.org | bash
```

#### 方式二：Kubernetes 部署

```bash
# 1. 克隆项目
git clone https://github.com/Veritas-Calculus/vc-stack.git
cd vc-stack

# 2. 安装 Helm 3
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# 3. 添加 VC Stack Helm 仓库
helm repo add vc-stack https://charts.vc-stack.org
helm repo update

# 4. 自定义配置（可选）
cp values.yaml.example values.yaml
# 编辑 values.yaml 配置文件

# 5. 部署 VC Stack
helm install vc-stack Veritas-Calculus/vc-stack \
  --namespace vc-stack \
  --create-namespace \
  --values values.yaml
```

#### 方式三：Ansible 部署

```bash
# 1. 安装 Ansible
pip3 install ansible ansible-core

# 2. 克隆项目
git clone https://github.com/Veritas-Calculus/vc-stack.git
cd vc-stack/ansible

# 3. 配置主机清单
cp inventory/hosts.example inventory/hosts
# 编辑 inventory/hosts 文件，配置目标主机

# 4. 配置部署参数
cp group_vars/all.yml.example group_vars/all.yml
# 编辑配置文件

# 5. 执行部署
ansible-playbook -i inventory/hosts site.yml
```

### 验证安装

```bash
# 检查服务状态
kubectl get pods -n vc-stack

# 检查服务访问
curl -k https://your-domain/api/v1/health
```

### 访问系统

部署完成后，可通过以下方式访问系统：

- **Web Dashboard**：`https://your-domain`
- **API 接口**：`https://your-domain/api/v1`
- **API 文档**：`https://your-domain/docs`
- **监控面板**：`https://your-domain/grafana`

#### 默认管理员账号

```
用户名：admin
密码：VCStack@123
```

> ⚠️ **安全提醒**：首次登录后请立即修改默认密码！

### vc-lite 使用 libvirt 驱动与品牌定制

vc-lite 默认使用内置的 mock 驱动用于开发调试。要切换为 libvirt 真机驱动并在虚拟机中显示自定义品牌（SMBIOS/sysinfo），请按以下步骤：

1. 安装系统依赖（以 Debian/Ubuntu 为例）：

- libvirt-daemon、libvirt-clients、qemu-kvm
- Go 需要能链接 libvirt C 库（libvirt-dev）

1. 构建包含 libvirt 的二进制：

- 使用 Go 构建标签：`-tags libvirt`

1. 运行时环境变量（可选，均有默认值）：

- `LIBVIRT_URI`：libvirt 连接 URI（默认 `qemu:///system`）
- `VC_LITE_NET_NAME`：libvirt 网络名称（默认 `default`）
- `VC_LITE_SMBIOS_MANUFACTURER`：SMBIOS 厂商（默认 `VC Stack`）
- `VC_LITE_SMBIOS_PRODUCT`：SMBIOS 产品名（默认 `VC Stack`）

配置这些变量后创建的虚拟机会在 SMBIOS 中显示为 "VC Stack"，而不是默认的 QEMU 主机信息。

## 📚 文档导航

| 文档类型 | 链接 | 描述 |
|----------|------|------|
| 📖 **用户指南** | [docs/user-guide/](docs/user-guide/) | 详细的用户操作手册 |
| 🔧 **安装指南** | [docs/installation/](docs/installation/) | 各种环境的安装部署指南 |
| 👨‍💻 **开发指南** | [docs/developer/](docs/developer/) | 开发环境搭建和代码贡献 |
| 🏗️ **架构文档** | [docs/architecture/](docs/architecture/) | 系统架构和设计文档 |
| 📋 **API 文档** | [docs/api/](docs/api/) | RESTful API 接口文档 |
| 🚨 **故障排除** | [docs/troubleshooting/](docs/troubleshooting/) | 常见问题和解决方案 |
| ⚙️ **配置参考** | [docs/configuration/](docs/configuration/) | 详细的配置参数说明 |
| 🔐 **安全指南** | [docs/security/](docs/security/) | 安全配置和最佳实践 |
| 🐛 **Sentry 集成** | [docs/sentry-integration.md](docs/sentry-integration.md) | 错误追踪和性能监控配置 |
| 📊 **SonarQube 集成** | [docs/sonarqube-integration.md](docs/sonarqube-integration.md) | 代码质量扫描和分析 |

## 🔍 代码质量与监控

### Sentry 错误追踪

VC Stack 集成了 Sentry 进行实时错误追踪和性能监控：

- **自动错误捕获**：所有未处理的错误和 panic 自动上报
- **性能监控**：HTTP 请求、数据库查询的性能追踪
- **Release 追踪**：每次部署自动关联版本号
- **详细上下文**：请求信息、用户信息、环境变量等

**配置方式**：

```bash
# 设置 Sentry DSN
export SENTRY_DSN=https://your-key@sentry.infra.plz.ac/project-id
export SENTRY_ENVIRONMENT=production

# 重启服务
systemctl restart vc-controller vc-node
```

详细配置请参考：[Sentry 集成文档](docs/sentry-integration.md)

### SonarQube 代码质量

持续的代码质量检查和安全扫描：

- **代码覆盖率**：自动计算测试覆盖率 (目标 >70%)
- **Bug 检测**：静态分析发现潜在问题
- **安全漏洞扫描**：识别常见安全问题
- **代码异味检测**：发现代码质量问题
- **技术债务追踪**：量化技术债务

**本地运行**：

```bash
# 安装开发工具
make install-tools

# 运行代码质量检查
make quality-check

# 运行 SonarQube 分析
make sonar
```

详细说明请参考：[SonarQube 集成文档](docs/sonarqube-integration.md)

### CI/CD 集成

GitHub Actions 自动执行：

- ✅ 代码编译和构建
- ✅ 单元测试和覆盖率
- ✅ 代码质量扫描 (golangci-lint)
- ✅ 安全扫描 (gosec)
- ✅ SonarQube 分析
- ✅ 质量门禁检查

## 🤝 参与贡献

我们热烈欢迎社区贡献！无论是代码、文档、测试还是反馈建议。

### 贡献方式

- 🐛 **报告 Bug**：[提交 Issue](https://github.com/Veritas-Calculus/vc-stack/issues/new?template=bug_report.md)
- 💡 **功能建议**：[提交 Feature Request](https://github.com/Veritas-Calculus/vc-stack/issues/new?template=feature_request.md)
- 📖 **改进文档**：提交文档 PR
- 🔧 **贡献代码**：Fork 项目并提交 PR

### 开发环境搭建

```bash
# 1. Fork 并克隆项目
git clone https://github.com/your-username/vc-stack.git
cd vc-stack

# 2. 安装开发依赖
make dev-install

# 3. 启动开发环境
make dev-start

# 4. 运行测试
make test

# 5. 代码检查
make lint
```

### 提交规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
feat: 添加新功能
fix: 修复 bug
docs: 更新文档
style: 代码格式调整
refactor: 代码重构
test: 添加测试
chore: 构建过程或辅助工具的变动
```

## 🚧 开发路线图

### v1.0.0 - 核心功能 (当前开发中)

- [x] 基础架构搭建
- [x] 认证和权限系统
- [x] 计算服务基础功能
- [ ] 网络服务完善
- [ ] 存储服务集成
- [ ] Web Dashboard 完善
- [ ] API 文档完善

### v1.1.0 - 增强功能

- [ ] 容器服务支持 (Kubernetes 集成)
- [ ] 自动伸缩功能
- [ ] 备份和恢复机制
- [ ] 多租户支持
- [ ] 计费系统优化

### v1.2.0 - 高级特性

- [ ] 边缘计算支持
- [ ] AI/ML 工作负载优化
- [ ] 混合云管理
- [ ] 高级网络功能 (SD-WAN)
- [ ] 容灾和业务连续性

### v2.0.0 - 企业级特性

- [ ] 多数据中心支持
- [ ] 高级安全特性
- [ ] 性能优化和大规模部署
- [ ] 第三方集成生态
- [ ] 企业级支持服务

## 📊 项目状态

| 指标 | 状态 |
|------|------|
| **开发状态** | 🚧 活跃开发中 |
| **最新版本** | v1.0.0-dev |
| **测试覆盖率** | ![Coverage](https://img.shields.io/badge/coverage-75%25-yellow) |
| **代码质量** | ![Code Quality](https://img.shields.io/badge/quality-A-green) |
| **社区活跃度** | ![Contributors](https://img.shields.io/github/contributors/Veritas-Calculus/vc-stack) |

## 📄 许可证

本项目基于 [Apache License 2.0](LICENSE) 许可证开源。

## 🔗 相关链接

- 🌐 [官方网站](https://vc-stack.org)
- 📚 [在线文档](https://docs.vc-stack.org)
- 💬 [社区讨论](https://github.com/Veritas-Calculus/vc-stack/discussions)
- 🐛 [问题反馈](https://github.com/Veritas-Calculus/vc-stack/issues)
- 📰 [更新日志](CHANGELOG.md)
- 📧 [邮件列表](mailto:dev@vc-stack.org)

## 🏆 致谢

感谢所有为 VC Stack 项目做出贡献的开发者和用户！

### 核心贡献者

- [@contributor1](https://github.com/contributor1) - 项目发起人
- [@contributor2](https://github.com/contributor2) - 架构师
- [@contributor3](https://github.com/contributor3) - 前端负责人

### 社区贡献者

![Contributors](https://contrib.rocks/image?repo=Veritas-Calculus/vc-stack)

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给我们一个 Star！**

**📢 关注我们获取最新动态**

[![GitHub stars](https://img.shields.io/github/stars/Veritas-Calculus/vc-stack?style=social)](https://github.com/Veritas-Calculus/vc-stack/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/Veritas-Calculus/vc-stack?style=social)](https://github.com/Veritas-Calculus/vc-stack/network/members)
[![GitHub watchers](https://img.shields.io/github/watchers/Veritas-Calculus/vc-stack?style=social)](https://github.com/Veritas-Calculus/vc-stack/watchers)

Made with ❤️ by VC Stack Team

</div>
