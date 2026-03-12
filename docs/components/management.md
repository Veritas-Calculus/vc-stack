# vc-management: The Management Plane

The `vc-management` component is the central brain of the VC Stack. It handles identity, resource scheduling, networking orchestration, and the API gateway.

## Core Responsibilities

- **Identity & IAM**: Manages users, projects, roles, and authentication (JWT/MFA).
- **Resource Scheduling**: Decides which compute node should host a new instance based on resource availability (bin-packing).
- **Network Orchestration**: Manages logical networks, subnets, routers, and security groups via OVN.
- **API Gateway**: Provides a single entry point for the Web Console and CLI, handling rate limiting, CORS, and request routing.
- **Service Registry**: Orchestrates over 60 internal modules that provide various cloud services.

## Internal Architecture

`vc-management` is a modular Go binary. Each service (Identity, Compute, Network, etc.) is implemented as an internal package that registers itself with a central `ModuleRegistry`.

### Key Service Modules

| Module | Purpose | Key Features |
| :--- | :--- | :--- |
| **Identity** | Authentication & RBAC | Users, Projects, JWT, MFA, OIDC |
| **Compute** | Instance Lifecycle | CRUD for VMs, power management, snapshots |
| **Network** | SDN Management | OVN networks, routers, FIPs, Security Groups |
| **Scheduler** | Resource Placement | Host selection, bin-packing, GPU scheduling |
| **Storage** | Block Storage | Volume management (Local/RBD) |
| **Image** | OS Image Repository | Image registration and distribution |
| **Gateway** | API Entry Point | Rate limiting, CORS, proxy, WebShell |
| **Metadata** | Cloud-init Support | EC2-compatible metadata service |
| **Event** | Audit Logging | Complete tracking of all resource operations |
| **Quota** | Resource Limits | Tenant-level limits (CPU, RAM, Disk, etc.) |
| Monitoring | Health & Metrics | Prometheus metrics, health probes |

## Advanced Services

Beyond the core infrastructure, `vc-management` provides several advanced services:

- **CaaS (Kubernetes as a Service)**: Full lifecycle management of Kubernetes clusters, integrated with OVN networking.
- **BareMetal (BMaaS)**: Physical server provisioning using IPMI and PXE.
- **KMS (Key Management Service)**: Envelope encryption for sensitive data and volume encryption.
- **Object Storage**: S3-compatible object storage (via Ceph RGW integration).
- **VPN & Networking**: Site-to-Site VPN, NAT Gateways, and Load Balancers.
- **Auto-scaling**: Policy-based scaling for instance groups.
- **Billing & Usage**: Metering, tariffs, and cost allocation.

## Detailed Features

### 1. Metadata Service

Provides an EC2-compatible metadata service for instances. It supports `cloud-init` for automatic configuration of virtual machines.

- EC2-compatible APIs: `/latest/meta-data`, `/latest/user-data`
- Dynamic configuration retrieval within VMs via `169.254.169.254`.

### 2. Event & Audit Service

A comprehensive audit trail and event log system:

- **Audit Logs**: Records all operations on resources.
- **Multi-dimensional Queries**: Filter by resource, user, or time.
- **Retention**: Configurable data retention policies (default: 90 days).

### 3. Quota Management

Flexible resource quota system to ensure fair usage and prevent resource exhaustion:

- **Dimensions**: Instances, vCPUs, RAM, Disk, Volumes, Floating IPs, etc.
- **Enforcement**: Automatic checks before resource creation.

### 4. API Gateway & Middleware

Standardized request handling through a multi-layer middleware stack:

- **Security**: JWT authentication, RBAC checks.
- **Performance**: Rate limiting, request tracing.
- **Usability**: CORS, structured logging, request-id propagation.

## Database & Persistence

`vc-management` uses **PostgreSQL 15+** as its primary data store.

- Schema evolution is managed via GORM AutoMigrate and SQL migration files in `migrations/`.
- Sensitive data (passwords, tokens) is encrypted at rest using `AES-256-GCM`.

## Operational Endpoints

- `/health`: Overall health check.
- `/health/liveness`: Kubernetes liveness probe.
- `/health/readiness`: Kubernetes readiness probe.
- `/metrics`: Prometheus metrics export.

## Configuration

Configuration is handled via a YAML file (e.g., `configs/vc-management.yaml`) or environment variables. Key settings include:

- `VC_MANAGEMENT_PORT`: API listening port (default: 8080).
- `DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASS`: Database connection details.
- `MASTER_KEY`: Used for encrypting/decrypting sensitive database fields.
