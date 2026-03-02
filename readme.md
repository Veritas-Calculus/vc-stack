# VC Stack

A modern, lightweight IaaS (Infrastructure as a Service) platform designed
as a simplified alternative to OpenStack. VC Stack delivers core cloud
infrastructure services — compute, network, storage, and identity —
with a clean two-component architecture and a modern tech stack.

## Architecture

```text
┌─────────────────────────────────────────────────────────┐
│                    vc-management                        │
│              (Management Plane - Single Binary)         │
│                                                         │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌──────────┐  │
│  │ Identity │ │ Compute  │ │ Scheduler │ │ Network  │  │
│  │  (IAM)   │ │ (Inst.)  │ │           │ │  (OVN)   │  │
│  └──────────┘ └──────────┘ └───────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌──────────┐  │
│  │  Quota   │ │  Event   │ │  Gateway  │ │ Metadata │  │
│  └──────────┘ └──────────┘ └───────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐                              │
│  │   Host   │ │Monitoring│    ┌──────────────────────┐  │
│  │ Manager  │ │          │    │ Web Console (React)  │  │
│  └──────────┘ └──────────┘    └──────────────────────┘  │
├────────────────── REST API ─────────────────────────────┤
│                   PostgreSQL                            │
└──────────┬──────────────────────────────┬───────────────┘
           │ Schedule / Dispatch          │ Heartbeat
           ▼                              ▼
┌─────────────────────┐      ┌─────────────────────┐
│    vc-compute (N1)  │      │    vc-compute (N2)  │
│   (Compute Node)    │      │   (Compute Node)    │
│                     │      │                     │
│ ┌─────────────────┐ │      │ ┌─────────────────┐ │
│ │  Orchestrator   │ │      │ │  Orchestrator   │ │
│ │ (VM Lifecycle)  │ │      │ │ (VM Lifecycle)  │ │
│ ├─────────────────┤ │      │ ├─────────────────┤ │
│ │   VM Driver     │ │      │ │   VM Driver     │ │
│ │  (QEMU/KVM)    │ │      │ │  (QEMU/KVM)    │ │
│ ├─────────────────┤ │      │ ├─────────────────┤ │
│ │ Network Agent   │ │      │ │ Network Agent   │ │
│ │  (OVN/OVS)     │ │      │ │  (OVN/OVS)     │ │
│ ├─────────────────┤ │      │ ├─────────────────┤ │
│ │ Storage Agent   │ │      │ │ Storage Agent   │ │
│ │ (Local/Ceph)   │ │      │ │ (Local/Ceph)   │ │
│ └─────────────────┘ │      │ └─────────────────┘ │
└─────────────────────┘      └─────────────────────┘
```

The system has only **two binaries**:

- **`vc-management`** — The centralized management plane. Exposes a
  REST API and Web Console. Manages identity (IAM with RBAC), instance
  scheduling, OVN network orchestration, host registration, quotas,
  events/audit, monitoring (InfluxDB + Prometheus), and metadata.

- **`vc-compute`** — The compute node agent. Runs on each hypervisor
  host. Composes three internal services via direct in-process calls
  (no internal HTTP):
  - **Orchestrator**: VM lifecycle (create, delete, start, stop,
    reboot, resize, snapshot, backup), image/volume/SSH key management.
  - **VM Driver** (`vm/`): Low-level QEMU/KVM process management,
    cloud-init seed ISO generation, QMP socket control, VNC console.
  - **Network Agent** (`network/`): Local OVN/OVS port configuration
    for tenant network isolation.

- **`vcctl`** — AWS CLI-inspired command-line tool covering compute,
  network, storage, identity, cluster, and secrets management.

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
| API | REST + Protobuf (Identity gRPC) |

## Features

### Compute

- [x] Instance lifecycle (create, delete, start, stop, reboot)
- [x] Flavors (resource templates: vCPU, RAM, Disk)
- [x] Image management (qcow2, raw; local filesystem and Ceph/RBD)
- [x] Volume management (local qcow2 for dev, Ceph/RBD for production)
- [x] Snapshots and backups (RBD snapshot export)
- [x] SSH key injection via cloud-init
- [x] UEFI and vTPM support
- [x] VNC console access (noVNC WebSocket proxy)
- [x] WebShell terminal (xterm.js)
- [x] Scheduler-based multi-node VM placement
- [x] Async deletion queue with retry

### Storage

- [x] Dual-backend: local filesystem or Ceph/RBD (configurable per-service)
- [x] Local backend for images and volumes (dev/test only)
- [x] Ceph/RBD backend for images, volumes, snapshots, backups
- [x] Static build without Ceph SDK (uses `rbd` CLI fallback)
- [x] Optional native Ceph SDK via `GO_BUILD_TAGS=ceph`

> **Warning**: Local storage provides NO high availability, NO live migration,
> and NO data protection. A node failure means permanent data loss for all VMs
> on that node. **Production deployments MUST use Ceph/RBD.**

### Networking

- [x] OVN logical networks and subnets
- [x] IPAM (IP Address Management)
- [x] Security groups and rules (OVN ACLs)
- [x] Floating IPs
- [x] Router support
- [x] Load balancer (OVN LB)
- [x] Namespace isolation

### Identity & Access

- [x] JWT-based authentication (access + refresh tokens)
- [x] Multi-project support
- [x] IAM policies and RBAC
- [x] User and account management

### Operations

- [x] Host registration with heartbeat monitoring
- [x] Resource quota management
- [x] Event audit logging
- [x] InfluxDB metrics collection
- [x] Prometheus metrics endpoint
- [x] Sentry error tracking integration
- [x] SonarQube code quality integration

### Security

- [x] CloudStack-style encrypted secrets (`ENC()` with AES-256-GCM)
- [x] Master key-based credential management
- [x] Input validation (network CIDRs, usernames, IPs)
- [x] Security headers middleware (CSP, HSTS, X-Frame-Options)
- [x] CORS configuration

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

### 5. Access

- **Web Console**: <http://localhost:8080>
- **API**: <http://localhost:8080/api/v1/>

## Development

| Command | Description |
| :--- | :--- |
| `make build` | Build all binaries to `bin/` |
| `make test` | Run tests with race detection |
| `make lint` | Run golangci-lint |
| `make fmt` | Format Go code |
| `make proto` | Regenerate Protobuf code |
| `make dev-start` | Start dev PostgreSQL |
| `make dev-stop` | Stop dev infrastructure |
| `make install-tools` | Install dev tools |

Frontend (in `web/console/`):

| Command | Description |
| :--- | :--- |
| `npm run dev` | Vite dev server |
| `npm run build` | Production build |
| `npm run lint` | ESLint + Prettier |
| `npm run test` | Vitest |

## Project Structure

```text
cmd/
  vc-management/       Management plane entry point
  vc-compute/          Compute node entry point + VM commands
  vcctl/               CLI (compute, network, storage, identity,
                        cluster, secrets, server, config)
internal/
  management/          Management plane services
    identity/            IAM, JWT auth, RBAC policies
    compute/             Instance scheduling and dispatch
    scheduler/           Multi-node VM placement
    network/             OVN orchestration (19 files)
    gateway/             API proxy, WebShell sessions
    host/                Node registration and health
    quota/               Resource quotas
    event/               Audit event logging
    metadata/            Instance metadata service
    monitoring/          InfluxDB + Prometheus metrics
    middleware/          Auth middleware
  compute/             Compute node (unified package)
    compute.go           Node aggregator (composes all services)
    service.go           Orchestrator (75KB, VM lifecycle)
    handlers.go          REST handlers (83KB)
    ovn_network.go       Local OVN network configuration
    ovn_security.go      Security group enforcement
    qemu_*.go            QEMU config, firmware, handlers
    rbd_manager.go       Ceph/RBD (native SDK, build tag: ceph)
    rbd_manager_nocgo.go Ceph/RBD (CLI fallback, default build)
    controller_client.go Heartbeat and status reporting
    vm/                  QEMU/KVM driver
      service.go           VM service with HTTP + direct API
      driver.go            Driver interface
      driver_qemu.go       QEMU process management (744 lines)
      direct.go            In-process call API
      qemu/                Config, templates, cloud-init
    network/             OVS network agent (local port ops only)
      service.go           Port attach/detach on br-int
      bootstrap.go         Node network init (OVS/OVN/encap)
pkg/
  database/            PostgreSQL connection (auto-decrypts ENC())
  security/            AES-256-GCM crypto, input validation
  logger/              Zap logger setup
  models/              Shared data models (Host, Instance, etc.)
  agent/               Compute node agent configuration
  sentry/              Sentry error tracking
configs/               YAML, env, systemd, nginx, docker-compose
docs/                  SECURITY.md, IAM API, Sentry, SonarQube
web/console/           React dashboard (17 feature modules)
migrations/            PostgreSQL schema (5 migration files)
api/proto/             Protobuf definitions (identity.proto)
scripts/               Deploy, rollback, DB init/migration
```

## Documentation

- [Security Best Practices](docs/SECURITY.md)
- [IAM API Reference](docs/iam-api.md)
- [Sentry Integration](docs/sentry-integration.md)
- [SonarQube Integration](docs/sonarqube-integration.md)
- [Pre-commit Hooks](docs/pre-commit.md)
- [Configuration Guide](configs/README.md)
- [vcctl CLI Reference](cmd/vcctl/README.md)

## License

VC Stack is open-source software. See the LICENSE file for details.
