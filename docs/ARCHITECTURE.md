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

The management plane is a single Go binary that aggregates all control-plane services using an **Inversion of Control (IoC)** container. Each service registers via a `ModuleRegistry` and interacts with other modules through interfaces via a `ModuleContext`.

#### Service Initialization Flow (IoC)

```
main()
  → config.Load()
  → ModuleRegistry.New()
  → RegisterCoreModules(registry)
  → registry.InitializeAll(mctx)  // ordered by dependency, resolves interfaces
  → gateway.SetupRoutes()         // dynamic route discovery
  → gin.Run()
```

#### Orchestration via Workflow Engine

Resource lifecycle operations (e.g., creating a VM) are managed by a **Declarative Workflow Engine**. Each operation is a persistent `Task` composed of atomic `Steps` with built-in **Compensation (rollback)** logic to ensure eventual consistency across distributed components.

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

The compute node is a **Stateless Agent**. It does NOT connect to the global database. It receives instructions from the management plane via a secured M2M API and reports status back.

#### Internal Architecture

```
vc-compute (Agent)
  ├─ Agent Handlers (api/v1/agent/*)
  │   ├─ StartVM (Receive models.Instance)
  │   ├─ StopVM
  │   └─ GetVNC (Local port mapping)
  │
  ├─ Orchestrator (service.go)
  │   ├─ Local VM lifecycle management
  │   └─ Heartbeat → vc-management (M2M API)
  │
  ├─ VM Driver (vm/)
  │   ├─ QEMU/KVM process management
  │   └─ QMP socket control
  │
  ├─ Network Agent (network/)
  │   ├─ OVS bridge management (br-int)
  │   └─ OVN logical port binding
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

### VM Creation (Workflow-driven)

```
User → Web Console → POST /api/v1/instances
  → Compute Service (Create DB record "building")
    → Start Workflow "CreateVM":
      1. StepAllocateIP (Network Module)
      2. StepCreateVolume (Storage Module)
      3. StepStartInstance (Call Agent API)
    → Agent executes locally
    → Agent reports "active" via M2M PATCH API
    → Task marks as "completed"
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
