# VC Stack

**现代化的开源 IaaS 云平台**

类似于 OpenStack 但更加轻量、易用、现代化的基础设施即服务平台

---

<div align="center">

![Version](https://img.shields.io/badge/version-v1.0.0--dev-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![React](https://img.shields.io/badge/React-18+-61DAFB?logo=react)

</div>

---

## 目录

- [项目简介](#项目简介)
- [安全提醒](#安全提醒)
- [核心特性](#核心特性)
- [技术架构](#技术架构)
- [核心组件](#核心组件架构)
- [快速开始](#快速开始)
- [文档](#文档导航)
- [参与贡献](#参与贡献)
- [许可证](#许可证)

---

## 开发计划 (TODO)

### 计算

- cloud init
- 调整配置
- 扩容
- 配置模板
- 加密计算
  - tpm
- UEFI VM

### 监控

- perf 性能检测
- prometheus 指标
- grafana 面板

### 网络

- ovn 网络
- LB

### 认证

- oidc
- 本地认证
- 密码重置
- OTP

### 部署

- helm charts
- k8s deployment
- docker compose
- 二进制部署脚本
- deb 包
- gentoo overlays
- 前端包特别要注意是全离线 方便后续管理和维护

### CI/CD

- [x] build
- [x] test
- [x] release
- [x] pre commit 配置
- [x] 全静态编译
- [x] 代码扫描 (SonarQube)
- [x] sentry 错误追踪
- [x] 代码质量检查 (golangci-lint)
- [x] 安全扫描 (gosec)

### 存储

- s3 共享存储 -> ceph rgw

### 计量

- 资源使用统计
- 成本分析

### 安全

- Key 管理
- 加密包
- IAM

### Orchestration

- 告警
- workflow

### Web Console

- 功能完善
- 用户体验优化

### 文档

- API 文档完善
- 用户手册
- 开发指南

### IaC

- terraform 支持

### Debug

- 操作失败要有对应的 uuid 可以在日志里面进行查询

---

## 当前目标 (OKR)

1. 创建虚拟机并启动
2. 完成简单的网络配置
3. ssh key 注入
4. cloud init 支持
5. 网络拓扑
6. webshell 登陆
7. openQA 自动化测试

---

## 项目简介

VC Stack 是一个现代化的开源 IaaS（Infrastructure as a Service）平台，旨在提供比 OpenStack 更简洁、更易用的云基础设施管理解决方案。它采用云原生架构设计，支持多云管理，为企业和开发者提供完整的虚拟化基础设施服务。

---

## 安全提醒

**在生产环境部署前，请务必：**

1. **更改所有默认密码和凭据** - 查看 [安全配置文档](docs/SECURITY.md)
2. **生成强随机 JWT 密钥** - 使用 `openssl rand -base64 64`
3. **启用数据库 SSL/TLS** - 设置 `sslmode: verify-full`
4. **使用密钥管理工具** - 如 HashiCorp Vault、AWS Secrets Manager
5. **配置 HTTPS** - 为所有 API 端点启用 TLS

**WARNING: 默认凭据仅用于开发/测试，切勿在生产环境使用！**

详细的安全配置指南请参阅：[docs/SECURITY.md](docs/SECURITY.md)

---

## 核心特性

### 部署与管理

- **快速部署**：支持 Kubernetes 和 Ansible 自动化部署
- **多云管理**：统一管理多个云平台资源
- **现代化 Dashboard**：基于 React 的直观管理界面
- **Infrastructure as Code**：完整的 Terraform 支持

### 计算服务

- **多虚拟化支持**：
  - **KVM 虚拟机**：完整的虚拟机生命周期管理
  - **LXC 容器**：轻量级容器化解决方案
- **裸金属支持**：集成 Ironic 服务，支持物理机管理
- **原生 ISO 启动**：支持自定义镜像和系统安装
- **虚拟机高可用**：自动故障转移和恢复机制
- **AI 训练扩展平台**：GPU 资源调度和 AI 工作负载优化

### 网络服务

- **多种网络模型**：支持扁平网络、VLAN、VXLAN 等
- **负载均衡服务**：类似 Octavia 的 L4/L7 负载均衡
- **DNS 服务**：类似 Designate 的域名管理和解析
- **软件定义网络**：灵活的网络虚拟化和策略管理

### 存储服务

- **Ceph 分布式存储**：高可用的后端存储集群
- **镜像服务**：类似 Glance 的虚拟机镜像管理
- **多存储类型**：块存储、对象存储、文件存储全支持

### 安全与认证

- **多重认证系统**：
  - 内建基础认证
  - LDAP/Active Directory 集成
  - SSO 单点登录支持
- **RBAC 权限控制**：基于角色的细粒度权限管理
- **密钥管理**：类似 Barbican 的密钥和证书服务
- **API 安全**：统一的 API 网关和访问控制

### 监控与运维

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

## 技术架构

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

## 核心组件架构

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

## 快速开始

### 系统要求

- **操作系统**: Ubuntu 20.04+ / CentOS 8+ / RHEL 8+
- **CPU**: 8+ 核心
- **内存**: 16+ GB
- **存储**: 100+ GB SSD

### 本地开发环境

```bash
# 克隆项目
git clone https://github.com/Veritas-Calculus/vc-stack.git
cd vc-stack

# 构建项目
make build

# 运行测试
make test
```

### 部署说明

详细的部署文档正在编写中。目前支持：

- Docker Compose 开发环境部署
- 二进制文件直接部署
- Kubernetes 部署（计划中）

配置文件示例请参考 `configs/` 目录。

---

## 文档导航

- [安全指南](docs/SECURITY.md) - 安全配置和最佳实践
- [IAM API 文档](docs/iam-api.md) - 身份认证与访问控制 API
- [Sentry 集成](docs/sentry-integration.md) - 错误追踪和性能监控配置
- [SonarQube 集成](docs/sonarqube-integration.md) - 代码质量扫描和分析
- [Pre-commit 钩子](docs/pre-commit.md) - 代码提交前检查配置

---

## 代码质量与监控

项目集成了以下工具来保证代码质量：

- **Sentry**: 错误追踪和性能监控，详见 [Sentry 集成文档](docs/sentry-integration.md)
- **SonarQube**: 代码质量分析，详见 [SonarQube 集成文档](docs/sonarqube-integration.md)
- **golangci-lint**: Go 代码静态分析
- **gosec**: 安全漏洞扫描

本地运行代码质量检查：

```bash
# 安装开发工具
make install-tools

# 运行所有检查
make lint

# 运行测试
make test
```

---

## 参与贡献

欢迎提交 Issue 和 Pull Request！

### 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
feat: 添加新功能
fix: 修复 bug
docs: 更新文档
style: 代码格式调整
refactor: 代码重构
test: 添加测试
chore: 构建过程或辅助工具的变动
```

---

## 许可证

本项目基于 [Apache License 2.0](LICENSE) 许可证开源。

---

## 相关链接

- 官方网站: [https://vc-stack.org](https://vc-stack.org)
- 社区讨论: [GitHub Discussions](https://github.com/Veritas-Calculus/vc-stack/discussions)
- 问题反馈: [GitHub Issues](https://github.com/Veritas-Calculus/vc-stack/issues)

---

<div align="center">

**如果这个项目对你有帮助，请给我们一个 Star！**

[![GitHub stars](https://img.shields.io/github/stars/Veritas-Calculus/vc-stack?style=social)](https://github.com/Veritas-Calculus/vc-stack/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/Veritas-Calculus/vc-stack?style=social)](https://github.com/Veritas-Calculus/vc-stack/network/members)

Made with care by VC Stack Team

</div>
