# Resource Naming Conventions

This document defines the naming conventions for all VC Stack resources.

## Resource ID Format

All resource IDs use an **AWS-style prefixed format**:

```
{prefix}-{12hex}
```

- **prefix**: A short, unique identifier for the resource type (1-5 chars)
- **hex**: 12 lowercase hexadecimal characters from `crypto/rand`
- **Total length**: 14-18 characters

### Prefix Registry

| Resource Type       | Prefix   | Example                | Module    |
|:--------------------|:---------|:-----------------------|:----------|
| Instance (VM)       | `i`      | `i-7fa3b2c4d5e6`      | compute   |
| Flavor              | `flv`    | `flv-a1b2c3d4e5f6`    | compute   |
| Image               | `img`    | `img-b2c3d4e5f6a7`    | compute   |
| Volume              | `vol`    | `vol-c3d4e5f6a7b8`    | storage   |
| Snapshot            | `snap`   | `snap-d4e5f6a7b8c9`   | storage   |
| Storage Pool        | `spool`  | `spool-d8e9f0a1b2c3`  | storage   |
| Backup              | `bak`    | `bak-e9f0a1b2c3d4`    | backup    |
| Network             | `net`    | `net-e5f6a7b8c9d0`    | network   |
| Subnet              | `sub`    | `sub-f6a7b8c9d0e1`    | network   |
| Router              | `rtr`    | `rtr-a7b8c9d0e1f2`    | network   |
| Floating IP         | `eip`    | `eip-b8c9d0e1f2a3`    | network   |
| Security Group      | `sg`     | `sg-c9d0e1f2a3b4`     | network   |
| Port                | `port`   | `port-d0e1f2a3b4c5`   | network   |
| Load Balancer       | `lb`     | `lb-e1f2a3b4c5d6`     | network   |
| VPC                 | `vpc`    | `vpc-f2a3b4c5d6e7`    | network   |
| Firewall Policy     | `fw`     | `fw-c5d6e7f8a9b0`     | network   |
| ACL                 | `acl`    | `acl-7fa3b2c4d5e6`    | network   |
| QoS Policy          | `qos`    | `qos-e5f6a7b8c9d0`    | network   |
| Port Forwarding     | `pf`     | `pf-a1b2c3d4e5f6`     | network   |
| Port Mirror         | `pm`     | `pm-b2c3d4e5f6a7`     | network   |
| Trunk Port          | `trunk`  | `trunk-c3d4e5f6a7b8`  | network   |
| Static Route        | `srt`    | `srt-d4e5f6a7b8c9`    | network   |
| BGP Peer            | `bgp`    | `bgp-d6e7f8a9b0c1`    | network   |
| ASN Range           | `asnr`   | `asnr-e7f8a9b0c1d2`   | network   |
| ASN Allocation      | `asna`   | `asna-f8a9b0c1d2e3`   | network   |
| Advertised Route    | `advr`   | `advr-a9b0c1d2e3f4`   | network   |
| Route Policy        | `rpol`   | `rpol-b0c1d2e3f4a5`   | network   |
| Network Offering    | `noff`   | `noff-c7d8e9f0a1b2`   | network   |
| DNS Record          | `dns`    | `dns-e3f4a5b6c7d8`    | network   |
| DNS Zone            | `dnsz`   | `dnsz-f4a5b6c7d8e9`   | network   |
| VPN Gateway         | `vgw`    | `vgw-a3b4c5d6e7f8`    | vpn       |
| VPN Customer GW     | `cgw`    | `cgw-b4c5d6e7f8a9`    | vpn       |
| VPN Connection      | `vpn`    | `vpn-c5d6e7f8a9b0`    | vpn       |
| Zone                | `zone`   | `zone-a9b0c1d2e3f4`   | infra     |
| Cluster             | `cls`    | `cls-b0c1d2e3f4a5`    | infra     |
| Host                | `host`   | `host-f8a9b0c1d2e3`   | infra     |
| ASN                 | `asn`    | `asn-c1d2e3f4a5b6`    | infra     |
| K8s Cluster         | `k8s`    | `k8s-c1d2e3f4a5b6`    | caas      |
| K8s Node            | `k8sn`   | `k8sn-d2e3f4a5b6c7`   | caas      |
| Bare Metal Server   | `bms`    | `bms-d2e3f4a5b6c7`    | baremetal  |
| Provision           | `prov`   | `prov-e3f4a5b6c7d8`   | baremetal  |
| Project             | `proj`   | `proj-a5b6c7d8e9f0`   | identity  |
| User                | `usr`    | `usr-b6c7d8e9f0a1`    | identity  |
| Domain              | `dom`    | `dom-c6d7e8f9a0b1`    | identity  |
| Role                | `role`   | `role-d7e8f9a0b1c2`   | identity  |
| Task                | `task`   | `task-f4a5b6c7d8e9`   | system    |
| Event               | `evt`    | `evt-a5b6c7d8e9f0`    | system    |
| Tag                 | `tag`    | `tag-b6c7d8e9f0a1`    | system    |
| Migration           | `mig`    | `mig-c7d8e9f0a1b2`    | compute   |

### Usage

```go
import "github.com/Veritas-Calculus/vc-stack/pkg/naming"

// Generate a new ID
id := naming.GenerateID(naming.PrefixInstance)  // "i-7fa3b2c4d5e6"

// Parse an ID
prefix, hex, ok := naming.ParseID("vpc-f2a3b4c5d6e7")
// prefix="vpc", hex="f2a3b4c5d6e7", ok=true

// Get resource type from ID
resType := naming.ResourceType("rtr-a7b8c9d0e1f2")  // "Router"

// Validate an ID
valid := naming.IsValidID("i-7fa3b2c4d5e6")  // true
```

---

## Name Validation

All resource names are validated using three modes:

### General Mode (`ModeGeneral`)

- **Use for**: VM names, network names, volume names, descriptions
- **Rules**: 1-255 characters, starts with letter/digit, allows UTF-8, spaces, `_.@#&()-`
- **XSS/injection check**: Rejects `<script`, `javascript:`
- **Example**: `"Web Server 01"`, `"生产环境-DB"`

### DNS-Safe Mode (`ModeDNSSafe`)

- **Use for**: Hostnames, slugs, DNS record names
- **Rules**: 1-63 characters, lowercase alphanumeric + hyphens, no leading/trailing hyphens
- **Example**: `"web-server-01"`, `"db-primary"`

### Identifier Mode (`ModeIdentifier`)

- **Use for**: Flavor names, zone names, offering names
- **Rules**: 1-63 characters, starts with letter, `[a-zA-Z0-9_.-]`
- **Example**: `"m1.xlarge"`, `"zone-core"`, `"MyFlavor_v2"`

### Validation Usage

```go
import "github.com/Veritas-Calculus/vc-stack/pkg/naming"

// Validate a general name
err := naming.ValidateName("Web Server 01", naming.ModeGeneral)  // nil

// Validate a DNS-safe name
err = naming.ValidateName("web-server-01", naming.ModeDNSSafe)   // nil

// Generate a slug from a display name
slug := naming.GenerateSlug("Web Server 01")  // "web-server-01"

// Generate an auto-name if user didn't provide one
name := naming.GenerateAutoName("vm")  // "vm-7fa3b2"
```

---

## DisplayName & Auto-Name

Resources support an optional **`display_name`** field for user-friendly labels, while `name` remains DNS-safe and unique within the tenant scope.

### Design

| Field          | Type       | Purpose                             | Constraints                |
|:---------------|:-----------|:-------------------------------------|:---------------------------|
| `name`         | Required   | DNS-safe slug, must be unique        | ModeGeneral validated      |
| `display_name` | Optional   | Human-friendly label (UTF-8, spaces) | Max 255 chars              |

### Auto-Name Behavior

When creating a resource, the `name` field is generated automatically if empty:

| Scenario                              | Result                      |
|:--------------------------------------|:----------------------------|
| `name` provided                       | Used as-is (after validation)|
| `name` empty, `display_name` provided | `naming.GenerateSlug(display_name)` → e.g., `"my-web-server"` |
| Both empty                            | `naming.GenerateAutoName(prefix)` → e.g., `"net-coral-falcon"` |

### Supported Resources

Auto-name is currently integrated for:

- **Network** — prefix `net`
- **Router** — prefix `rtr`

### API Example

```json
// Only display_name → name auto-generated as slug
POST /v1/networks
{
  "display_name": "My Production Network",
  "cidr": "10.0.0.0/24",
  "tenant_id": "t-123",
  "zone": "default"
}
// Response: { "name": "my-production-network", "display_name": "My Production Network", ... }

// No name at all → auto-generated
POST /v1/routers
{
  "tenant_id": "t-123"
}
// Response: { "name": "rtr-coral-falcon", "display_name": "", ... }
```

### Go Usage

```go
// Generate auto-name for a resource type
name := naming.GenerateAutoName("net")  // "net-coral-falcon"

// Generate slug from display name
slug := naming.GenerateSlug("My Web Server 01")  // "my-web-server-01"
```

---

## Name Uniqueness

| Resource        | Scope              | Constraint                        |
|:----------------|:-------------------|:----------------------------------|
| Instance        | Tenant-scoped      | `UNIQUE(tenant_id, name)` (planned) |
| Network         | Tenant-scoped      | `UNIQUE(tenant_id, name)` ✅       |
| Router          | Tenant-scoped      | `UNIQUE(tenant_id, name)` ✅       |
| VPC             | Tenant-scoped      | `UNIQUE(tenant_id, name)` ✅       |
| Security Group  | Tenant-scoped      | `UNIQUE(tenant_id, name)` ✅       |
| Flavor          | Global             | `UNIQUE(name)` ✅                  |
| Image           | Global             | `UNIQUE(name)` ✅                  |
| Zone            | Global             | `UNIQUE(name)` (planned)          |
| Volume          | Tenant-scoped      | `UNIQUE(tenant_id, name)` (planned) |

---

## Database Table Naming

All tables use a module-specific prefix:

| Module       | Prefix          | Example                     |
|:-------------|:----------------|:----------------------------|
| Network      | `net_`          | `net_networks`, `net_routers` |
| VPN          | `vpn_`          | `vpn_gateways`              |
| K8s/CaaS     | `k8s_`          | `k8s_clusters`              |
| Baremetal    | `bm_`           | `bm_servers`                |
| Audit        | `audit_`        | `audit_logs`                |
| Self-heal    | `sh_`           | `sh_health_checks`          |
| Orchestrate  | `orchestration_`| `orchestration_stacks`      |
| AutoScale    | `autoscale_`    | `autoscale_vm_groups`       |
| Backup       | `backup_`       | `backup_offerings`          |
| Catalog      | `catalog_`      | `catalog_items`             |
| Config       | `config_`       | `config_namespaces`         |
| DNS          | `dns_`          | `dns_zones`                 |
| Notification | `notification_` | `notification_channels`     |
| Registry     | `registry_`     | `registry_instances`        |
| Quota        | `quota_`        | `quota_sets`                |
| Infrastructure| `infra_`       | `infra_zones`               |

---

## Frontend Components

### `<ResourceId>`

Renders a prefixed ID with highlighting and click-to-copy:

```tsx
import { ResourceId } from '@/components/ui/ResourceId'

<ResourceId id="i-7fa3b2c4d5e6" />           // Full ID
<ResourceId id="vpc-f2a3b4c5d6e7" truncate /> // Truncated: vpc-f2a3b4c5...
```

### `<ResourceName>`

Renders a name with auto-truncation:

```tsx
import { ResourceName } from '@/components/ui/ResourceId'

<ResourceName name="My Very Long Resource Name Here" max={20} />
// Shows: "My Very Long Resourc..." with tooltip
```
