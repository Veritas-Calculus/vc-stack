# VC Stack Architecture

This document describes the system architecture, component interactions,
and design principles of VC Stack.

## Design Principles

1. **Two-Binary Simplicity** — Only `vc-management` and `vc-compute`
   binaries. No separate processes for identity, networking, or storage.
2. **CloudStack-Inspired API** — RESTful API design following CloudStack
   naming conventions with modern JSON payloads.
3. **In-Process Composition** — Compute node services communicate via
   direct function calls, not HTTP or RPC.
4. **GORM AutoMigrate** — Database schema evolves via code-first
   model definitions, supplemented by SQL migration files.

## System Overview

```
                    ┌──────────────────────────────┐
                    │         Load Balancer         │
                    │      (nginx + TLS term)       │
                    └──────────────┬───────────────┘
                                   │
              ┌────────────────────┴────────────────────┐
              │            vc-management                │
              │                                         │
              │  ┌─────────────────────────────────┐   │
              │  │          API Gateway            │   │
              │  │  (rate-limit, CORS, proxy)       │   │
              │  └─────────────┬───────────────────┘   │
              │                │                        │
              │  ┌─────────────┼───────────────────┐   │
              │  │    Service Module Registry       │   │
              │  │                                  │   │
              │  │  Identity  Compute  Network      │   │
              │  │  Scheduler Storage  Image        │   │
              │  │  Quota     Event    Host         │   │
              │  │  Gateway   Monitor  Metadata     │   │
              │  │  VPN       Backup   Autoscale    │   │
              │  │  Usage     Domain   KMS          │   │
              │  │  CaaS      BMaaS   Catalog       │   │
              │  │  DNS       HA      DR            │   │
              │  │  SelfHeal  Encrypt Audit         │   │
              │  │  Notify    Task    Tag           │   │
              │  │  EventBus  Config  Registry      │   │
              │  │  ObjStore  Orchestration         │   │
              │  │  RateLimit APIDocs               │   │
              │  └──────────────────────────────────┘   │
              │                                         │
              │  ┌──────────────────────────────────┐   │
              │  │     Web Console (React SPA)      │   │
              │  │     Served from /console/         │   │
              │  └──────────────────────────────────┘   │
              │                                         │
              └────────────────┬────────────────────────┘
                               │ PostgreSQL
                               │
              ┌────────────────┴────────────────┐
              │                                  │
    ┌─────────┴──────────┐          ┌───────────┴─────────┐
    │   vc-compute (1)   │          │   vc-compute (N)    │
    │                    │          │                     │
    │  Orchestrator      │          │  Orchestrator       │
    │  ├─ VM Driver      │          │  ├─ VM Driver       │
    │  ├─ Network Agent  │          │  ├─ Network Agent   │
    │  └─ Storage Agent  │          │  └─ Storage Agent   │
    │                    │          │                     │
    │  QEMU/KVM + OVS    │          │  QEMU/KVM + OVS     │
    └────────────────────┘          └─────────────────────┘
```

## Component Details

For in-depth information about each component, see the [Component Documentation](components/README.md).

### Management Plane (`vc-management`)

The management plane is a single Go binary that aggregates all control-plane services. Each service registers via `modules.go` and is initialized in dependency order. See the [Detailed Management Documentation](components/management.md) for more info.

#### Service Initialization Flow

```
main()
  → config.Load()
  → database.Connect()        // auto-decrypts ENC() passwords
  → identity.NewService()     // IAM, JWT, RBAC
  → compute.NewService()      // instance scheduling
  → network.NewService()      // OVN orchestration
  → scheduler.NewService()    // VM placement
  → storage.NewService()      // volume management
  → image.NewService()        // OS image management
  → ... (40+ services)
  → gateway.NewService()      // API proxy and routing
  → gateway.SetupRoutes()     // register all HTTP endpoints
  → gin.Run()                 // start HTTP server
```

#### Key Services

| Service | Package | Responsibilities |
|:---|:---|:---|
| Identity | `identity/` | Users, projects, JWT auth, RBAC, MFA, OIDC |
| Compute | `compute/` | Instance CRUD, scheduling dispatch |
| Network | `network/` | OVN networks, subnets, routers, FIPs, SGs, LBs |
| Scheduler | `scheduler/` | Multi-node placement (bin-packing) |
| Storage | `storage/` | Block volume lifecycle |
| Image | `image/` | OS image registration and management |
| Gateway | `gateway/` | API proxy, rate limiting, metrics, WebShell |
| Host | `host/` | Node registration, heartbeat, health |
| KMS | `kms/` | Envelope encryption key management |
| CaaS | `caas/` | Kubernetes cluster lifecycle |
| BareMetal | `baremetal/` | Physical server provisioning via IPMI/PXE |

### Compute Node (`vc-compute`)

The compute node is a single binary with three internal services
composed via direct function calls (no HTTP between them).
See the [Detailed Compute Documentation](components/compute.md) for more info.

#### Internal Architecture

```
vc-compute
  ├─ Orchestrator (service.go)
  │   ├─ VM lifecycle (create, delete, start, stop, resize, migrate)
  │   ├─ Image pull and caching
  │   ├─ Volume attach/detach
  │   └─ Heartbeat → vc-management
  │
  ├─ VM Driver (vm/)
  │   ├─ QEMU process management
  │   ├─ QMP socket control
  │   ├─ Cloud-init ISO generation
  │   ├─ VNC console proxy
  │   └─ UEFI/vTPM firmware
  │
  ├─ Network Agent (network/)
  │   ├─ OVS bridge management (br-int)
  │   ├─ OVN logical port binding
  │   └─ Node network bootstrap
  │
  └─ Storage Agent
      ├─ Local: qcow2 file management
      └─ Ceph: RBD image operations
```

### Web Console (`web/console/`)

A React 18 SPA built with TypeScript, TailwindCSS, and Vite.

### CLI Tool (`vcctl`)

A unified command-line interface for VC Stack. See the [Detailed CLI Documentation](components/vcctl.md) for more info.

#### Frontend Architecture

```
App (BrowserRouter)
  └─ Layout (sidebar + header)
      ├─ CommandPalette (Ctrl+K)
      ├─ KeyboardShortcuts (?)
      └─ <Routes>
          ├─ /dashboard          Dashboard
          ├─ /compute/*          Instances, Flavors
          ├─ /network/*          Networks, Subnets, FIPs, SGs, LBs
          ├─ /storage/*          Volumes, Snapshots
          ├─ /iam/*              Users, Policies, Federation
          ├─ /kubernetes/*       K8s Clusters
          ├─ /baremetal/*        Physical Servers
          ├─ /backup/*           Backups, Schedules
          └─ ... (40+ modules)
```

**State Management**: Zustand stores for auth, settings, and app state.
**API Layer**: Axios with interceptors for JWT refresh and error handling.

## Data Flow

### VM Creation

```
User → Web Console → POST /api/v1/instances
  → Gateway (auth + rate-limit)
    → Compute Service (validate, create record)
      → Scheduler (select host by bin-packing)
        → Dispatch to vc-compute
          → Orchestrator
            → VM Driver (QEMU launch)
            → Network Agent (OVN port bind)
            → Storage Agent (volume create)
          → Heartbeat reports status back
```

### Authentication Flow

```
User → POST /api/v1/auth/login (username + password)
  → Identity Service
    → Validate credentials
    → Check MFA status
      → If MFA: return challenge token
      → POST /api/v1/auth/mfa/verify (TOTP code)
    → Issue JWT (access + refresh tokens)
  → Client stores token in Zustand
  → Subsequent requests include Authorization: Bearer <token>
```

## Database Schema

PostgreSQL 15 with GORM AutoMigrate. Key tables:

| Table | Service | Purpose |
|:---|:---|:---|
| `users` | Identity | User accounts |
| `projects` | Identity | Multi-tenant projects |
| `iam_policies` | Identity | RBAC policy definitions |
| `instances` | Compute | VM records |
| `flavors` | Compute | Resource templates |
| `hosts` | Host | Registered compute nodes |
| `networks` | Network | OVN logical networks |
| `subnets` | Network | Network subnets with IPAM |
| `security_groups` | Network | Firewall rule groups |
| `volumes` | Storage | Block storage volumes |
| `images` | Image | OS images |
| `kms_keys` | KMS | Encryption keys |
| `k8s_clusters` | CaaS | Kubernetes clusters |

## Communication Patterns

| Path | Protocol | Purpose |
|:---|:---|:---|
| Client → Management | HTTPS/REST | All API operations |
| Management → Compute | HTTP | VM dispatch, node commands |
| Compute → Management | HTTP | Heartbeat, status updates |
| Console → Management | WebSocket | VNC, WebShell, events |
| Compute ↔ Compute | Geneve/OVN | VM-to-VM encapsulated traffic |
| Management → PostgreSQL | TCP | Data persistence |
| Management → InfluxDB | HTTP | Metrics collection |
