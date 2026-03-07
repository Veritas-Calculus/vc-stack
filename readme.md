# VC Stack

A modern, full-featured IaaS (Infrastructure as a Service) platform designed
as a simplified alternative to OpenStack and CloudStack. VC Stack delivers
enterprise cloud infrastructure services — compute, network, storage,
identity, Kubernetes, bare metal, and more — with a clean two-component
architecture and a modern tech stack.

## Architecture

```text
┌───────────────────────────────────────────────────────────────────────┐
│                          vc-management                               │
│                  (Management Plane - Single Binary)                   │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌──────────┐  │
│  │ Identity │ │ Compute  │ │ Scheduler │ │ Network  │ │  Image   │  │
│  │  (IAM)   │ │ (Inst.)  │ │           │ │  (OVN)   │ │ Service  │  │
│  └──────────┘ └──────────┘ └───────────┘ └──────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌──────────┐  │
│  │  Quota   │ │  Event   │ │  Gateway  │ │ Metadata │ │  Domain  │  │
│  └──────────┘ └──────────┘ └───────────┘ └──────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌──────────┐  │
│  │   Host   │ │Monitoring│ │  Backup   │ │   VPN    │ │  Usage   │  │
│  │ Manager  │ │          │ │           │ │          │ │ Billing  │  │
│  └──────────┘ └──────────┘ └───────────┘ └──────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌──────────┐  │
│  │   KMS    │ │   CaaS   │ │ BareMetal │ │ Catalog  │ │   DNS    │  │
│  │          │ │  (K8s)   │ │  (BMaaS)  │ │          │ │          │  │
│  └──────────┘ └──────────┘ └───────────┘ └──────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌──────────┐               │
│  │   HA     │ │    DR    │ │ Self-Heal │ │ Encrypt  │  Web Console  │
│  └──────────┘ └──────────┘ └───────────┘ └──────────┘  (React 18)   │
├────────────────────────── REST API ────────────────────────────────────┤
│                         PostgreSQL 15                                 │
└──────────┬──────────────────────────────────────────┬─────────────────┘
           │ Schedule / Dispatch                      │ Heartbeat
           ▼                                          ▼
┌─────────────────────────┐            ┌─────────────────────────┐
│    vc-compute (Node 1)  │            │    vc-compute (Node N)  │
│   ┌───────────────────┐ │            │   ┌───────────────────┐ │
│   │   Orchestrator    │ │            │   │   Orchestrator    │ │
│   │  (VM Lifecycle)   │ │            │   │  (VM Lifecycle)   │ │
│   ├───────────────────┤ │            │   ├───────────────────┤ │
│   │    VM Driver      │ │            │   │    VM Driver      │ │
│   │   (QEMU/KVM)      │ │            │   │   (QEMU/KVM)      │ │
│   ├───────────────────┤ │            │   ├───────────────────┤ │
│   │  Network Agent    │ │            │   │  Network Agent    │ │
│   │   (OVN/OVS)       │ │            │   │   (OVN/OVS)       │ │
│   ├───────────────────┤ │            │   ├───────────────────┤ │
│   │  Storage Agent    │ │            │   │  Storage Agent    │ │
│   │  (Local/Ceph)     │ │            │   │  (Local/Ceph)     │ │
│   └───────────────────┘ │            │   └───────────────────┘ │
└─────────────────────────┘            └─────────────────────────┘
```

The system has only **two binaries** plus a CLI:

- **`vc-management`** — Centralized management plane. Aggregates 40+
  service modules including identity (IAM/RBAC/MFA/OIDC), compute scheduling,
  OVN network orchestration, storage, Kubernetes CaaS, bare metal provisioning,
  backup/DR, VPN, DNS, KMS encryption, billing, and the Web Console.

- **`vc-compute`** — Compute node agent. Runs on each hypervisor host
  with three internal services via direct in-process calls (no internal HTTP):
  - **Orchestrator**: VM lifecycle, image/volume/SSH key management
  - **VM Driver** (`vm/`): QEMU/KVM process management, cloud-init, QMP, VNC
  - **Network Agent** (`network/`): Local OVN/OVS port configuration

- **`vcctl`** — CLI tool covering compute, network, storage, identity,
  cluster, and secrets management.

## Technology Stack

| Layer | Technologies |
| :--- | :--- |
| Backend | Go 1.24, Gin, GORM, Cobra, Zap, Sentry |
| Frontend | React 18, TypeScript, TailwindCSS, Vite, Zustand |
| Console | xterm.js (WebShell), noVNC (VNC Console) |
| Database | PostgreSQL 15 |
| Metrics | InfluxDB, Prometheus |
| Virtualization | QEMU/KVM (direct process management) |
| Networking | OVN/OVS (SDN, security groups, floating IPs) |
| Storage | Local filesystem (dev), Ceph/RBD (production) |
| Object Storage | S3-compatible (Ceph RGW) |
| API | REST + Protobuf (Identity gRPC) |
| Testing | Vitest, Playwright, Go testing |

## Features

### Compute

- [x] Instance lifecycle (create, delete, start, stop, reboot, resize)
- [x] Flavors (resource templates: vCPU, RAM, Disk)
- [x] Image management (qcow2, raw, ISO; local and Ceph/RBD backends)
- [x] Volume management with storage types (SSD, HDD, NVMe)
- [x] Snapshots and backups (RBD snapshot export)
- [x] SSH key injection via cloud-init
- [x] UEFI and vTPM support
- [x] VNC console access (noVNC WebSocket proxy)
- [x] WebShell terminal (xterm.js)
- [x] Scheduler-based multi-node VM placement
- [x] Live migration (pre-copy and post-copy)
- [x] Auto-scaling groups with scaling policies
- [x] Async deletion queue with retry

### Kubernetes (CaaS)

- [x] Kubernetes cluster lifecycle (create, scale, delete)
- [x] Calico CNI integration with OVN/OVS
- [x] LoadBalancer service via Floating IP + OVN LB
- [x] Multi-version support (1.28, 1.29, 1.30)
- [x] Worker node scaling

### Bare Metal (BMaaS)

- [x] Physical server lifecycle management
- [x] IPMI-based hardware control (power on/off, PXE boot)
- [x] Hardware inventory and location tracking
- [x] Automated OS provisioning (Ubuntu, Rocky, ESXi, Debian, Windows)
- [x] Datacenter rack visualization

### Storage

- [x] Block storage with SSD/HDD/NVMe types
- [x] Dual-backend: local filesystem or Ceph/RBD
- [x] S3-compatible object storage (Ceph RGW)
- [x] Volume snapshots and cloning
- [x] Backup and restore with scheduling
- [x] Static build without Ceph SDK (uses `rbd` CLI fallback)
- [x] Optional native Ceph SDK via `GO_BUILD_TAGS=ceph`

### Networking

- [x] OVN logical networks and subnets
- [x] IPAM (IP Address Management)
- [x] Security groups and rules (OVN ACLs)
- [x] Floating IPs with NAT
- [x] Router support
- [x] Load balancer (OVN LB)
- [x] VPN gateways and site-to-site connections
- [x] DNS zone and record management
- [x] Network namespace isolation

### Identity & Access

- [x] JWT-based authentication (access + refresh tokens)
- [x] Multi-Factor Authentication (TOTP + recovery codes)
- [x] OIDC/SAML federated SSO
- [x] Fine-grained RBAC with custom policies
- [x] Multi-tenant domain management
- [x] Multi-project support
- [x] User and account management

### Security & Compliance

- [x] Key Management Service (KMS) with envelope encryption
- [x] LUKS volume encryption at-rest
- [x] Mutual TLS (mTLS) for in-transit security
- [x] Tamper-proof audit chain (HMAC + SHA-256)
- [x] Compliance framework assessment (SOC 2, ISO 27001, PCI DSS, GDPR, HIPAA)
- [x] CloudStack-style encrypted secrets (`ENC()` with AES-256-GCM)
- [x] Security headers middleware (CSP, HSTS, X-Frame-Options)
- [x] API rate limiting

### High Availability & DR

- [x] HA cluster with fencing and VM evacuation
- [x] Disaster Recovery with RPO/RTO-based planning
- [x] Multi-site replication and failover orchestration
- [x] Proactive self-healing with health probes

### Platform Services

- [x] Service catalog and marketplace
- [x] Orchestration engine for stack deployment
- [x] Resource tagging and cost allocation
- [x] Usage metering and billing (tariffs, credits)
- [x] Notification system (Webhook, Slack, Email)
- [x] Task management for async operations
- [x] Service registry and config center
- [x] Event bus for async messaging

### Operations

- [x] Host registration with heartbeat monitoring
- [x] Resource quota management
- [x] Event audit logging with export
- [x] InfluxDB metrics collection
- [x] Prometheus metrics endpoint
- [x] Sentry error tracking integration
- [x] API documentation (Swagger/OpenAPI)

### Web Console

- [x] 40+ feature modules (compute, network, storage, IAM, K8s, ...)
- [x] Apple-inspired dark theme with glassmorphism
- [x] Command palette (Ctrl+K) with fuzzy search
- [x] Keyboard shortcuts with help panel
- [x] Dangerous action confirmation with type-to-confirm
- [x] Internationalization (i18n) support
- [x] Responsive layout with collapsible sidebar
- [x] Interactive API documentation viewer

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 18+ and npm
- Docker and Docker Compose
- `make`

### 1. Build

```bash
git clone https://github.com/Veritas-Calculus/vc-stack.git
cd vc-stack

# Standard build (static, no Ceph SDK dependency)
make build

# With native Ceph SDK (requires librados-dev, librbd-dev)
GO_BUILD_TAGS=ceph make build
```

### 2. Start Development Infrastructure

```bash
# Starts PostgreSQL 15
make dev-start
```

### 3. Configure Secrets

```bash
# Generate a master encryption key
./bin/vcctl secrets init -f /etc/vc-stack/master.key

# Encrypt your database password
./bin/vcctl secrets encrypt "your-db-password"
# Output: ENC(base64_ciphertext...)
```

### 4. Run

```bash
# Copy and edit configuration
cp configs/env/vc-management.env.example .env

# Start management plane
./bin/vc-management

# Start compute node (requires root for KVM/OVS)
sudo -E ./bin/vc-compute
```

### 5. Frontend Development

```bash
cd web/console
npm install
npm run dev
```

### 6. Access

- **Web Console**: <http://localhost:5173> (dev) or <http://localhost:8080> (prod)
- **API**: <http://localhost:8080/api/v1/>

## Development

### Backend

| Command | Description |
| :--- | :--- |
| `make build` | Build all binaries to `bin/` |
| `make test` | Run Go tests with race detection |
| `make lint` | Run golangci-lint |
| `make fmt` | Format Go code |
| `make proto` | Regenerate Protobuf code |
| `make dev-start` | Start dev PostgreSQL |
| `make dev-stop` | Stop dev infrastructure |
| `make install-tools` | Install dev tools |

### Frontend (`web/console/`)

| Command | Description |
| :--- | :--- |
| `npm run dev` | Vite dev server |
| `npm run build` | Production build |
| `npm run lint` | ESLint + Prettier |
| `npm run test` | Vitest unit tests |
| `npm run test:e2e` | Playwright E2E tests |
| `npm run test:e2e:headed` | E2E with visible browser |

### Test Coverage

| Layer | Tests | Framework |
| :--- | :--- | :--- |
| Backend | 42 packages, 55 files | Go testing |
| Frontend Unit | 13 files, 69 tests | Vitest + Testing Library |
| Frontend E2E | 3 files, 18 tests | Playwright |

## Project Structure

```text
cmd/
  vc-management/       Management plane entry point
  vc-compute/          Compute node entry point
  vcctl/               CLI (compute, network, storage, identity,
                        cluster, secrets, server, config)
internal/
  management/          Management plane services (40+ modules)
    identity/            IAM, JWT, RBAC, MFA, OIDC/SAML federation
    compute/             Instance scheduling and dispatch
    scheduler/           Multi-node VM placement
    network/             OVN orchestration (19 files)
    gateway/             API gateway, rate limiting, WebShell
    host/                Node registration and health
    storage/             Block volume management
    image/               OS image lifecycle
    quota/               Resource quotas
    event/               Audit event logging
    metadata/            Instance metadata service
    monitoring/          InfluxDB + Prometheus metrics
    domain/              Multi-tenant domain hierarchy
    vpn/                 VPN gateways and connections
    backup/              Backup and restore with scheduling
    autoscale/           Auto-scaling groups and policies
    usage/               Metering, tariffs, billing
    kms/                 Key Management Service
    encryption/          Volume encryption (LUKS)
    dns/                 DNS zones and records
    caas/                Kubernetes cluster management
    baremetal/           BMaaS with IPMI and PXE
    catalog/             Service catalog and marketplace
    orchestration/       Stack deployment engine
    ha/                  High availability
    dr/                  Disaster recovery
    selfheal/            Self-healing policies
    audit/               Compliance audit framework
    notification/        Webhooks, Slack, Email
    task/                Async task management
    tag/                 Resource tagging
    eventbus/            Event bus messaging
    configcenter/        Centralized config
    registry/            Service registry
    objectstorage/       S3-compatible object storage
    ratelimit/           API rate limiting
    apidocs/             Swagger/OpenAPI docs
    middleware/          Auth middleware
  compute/             Compute node (unified package)
    service.go           Orchestrator (VM lifecycle)
    handlers.go          REST handlers
    ovn_*.go             OVN network and security
    qemu_*.go            QEMU config, firmware
    rbd_manager.go       Ceph/RBD storage backend
    vm/                  QEMU/KVM driver
    network/             OVS network agent
pkg/
  database/            PostgreSQL (auto-decrypts ENC() passwords)
  security/            AES-256-GCM crypto, input validation
  logger/              Zap logger setup
  models/              Shared data models
  agent/               Compute node agent config
  sentry/              Sentry error tracking
configs/               YAML, env, systemd, nginx, docker-compose
docs/                  Security, IAM API, integration guides
web/console/           React dashboard (40+ feature modules)
  src/features/          Compute, Network, Storage, IAM, K8s, ...
  src/components/ui/     DataTable, Modal, Badge, PageHeader, ...
  e2e/                   Playwright E2E tests
migrations/            PostgreSQL schema migrations
api/proto/             Protobuf definitions
scripts/               Deploy, rollback, DB init/migration
```

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [Deployment Guide](docs/DEPLOYMENT.md)
- [Security Best Practices](docs/SECURITY.md)
- [IAM API Reference](docs/iam-api.md)
- [Sentry Integration](docs/sentry-integration.md)
- [SonarQube Integration](docs/sonarqube-integration.md)
- [Pre-commit Hooks](docs/pre-commit.md)

## License

VC Stack is open-source software. See the LICENSE file for details.
