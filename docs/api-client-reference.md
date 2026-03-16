# VC Stack API Client Reference

This document provides a comprehensive API reference for all VC Stack client libraries.

## Go SDK

### Installation

```bash
go get github.com/Veritas-Calculus/vc-stack/sdk/vcstack
```

### Authentication

```go
import "github.com/Veritas-Calculus/vc-stack/sdk/vcstack"

// Option 1: JWT token (interactive/user login)
client := vcstack.NewClient("https://vc.example.com/api")
resp, _ := client.Login(ctx, "admin", "password")
// Token is stored automatically after login.

// Option 2: API key (service accounts / automation)
client := vcstack.NewClient("https://vc.example.com/api")
client.SetAPIKey("VC-AKIA-0123456789abcdef", "your-secret-key")
// All requests signed with HMAC-SHA256.
```

### Resource Clients

| Client | Methods | API Prefix |
|:---|:---|:---|
| `Instances` | List, Get, Create, Delete, Action | `/v1/instances` |
| `Flavors` | List, Get, Create, Delete | `/v1/flavors` |
| `Images` | List, Get, Create, Delete | `/v1/images` |
| `Volumes` | List, Get, Create, Delete, Attach, Detach | `/v1/storage/volumes` |
| `Networks` | List, Get, Create, Delete | `/v1/networks` |
| `Subnets` | List, Create, Delete | `/v1/subnets` |
| `Routers` | List, Get, Create, Delete, AddInterface, RemoveInterface | `/v1/routers` |
| `SecurityGroups` | List, Get, Create, Delete, AddRule, DeleteRule | `/v1/security-groups` |
| `FloatingIPs` | List, Create, Associate, Disassociate, Delete | `/v1/floating-ips` |
| `SSHKeys` | List, Create, Delete | `/v1/ssh-keys` |
| `ServiceAccounts` | List, Create, Delete, RotateKey | `/v1/service-accounts` |
| `Projects` | List, Get, Create, Delete | `/v1/projects` |
| `Users` | List, Get, Create, Delete | `/v1/users` |

### Error Handling

```go
_, err := client.Instances.List(ctx)
if apiErr, ok := err.(*vcstack.APIError); ok {
    fmt.Printf("HTTP %d: %s\n", apiErr.StatusCode, apiErr.Message)
}
```

---

## Terraform Provider

### Provider Configuration

```hcl
terraform {
  required_providers {
    vcstack = {
      source = "Veritas-Calculus/vcstack"
    }
  }
}

provider "vcstack" {
  endpoint      = "https://vc.example.com/api"  # or VCSTACK_ENDPOINT
  token         = var.token                      # or VCSTACK_TOKEN
  # For API key auth:
  # access_key_id = "VC-AKIA-..."               # or VCSTACK_ACCESS_KEY_ID
  # secret_key    = var.secret                   # or VCSTACK_SECRET_KEY
}
```

### Resources

| Resource | Attributes | Computed |
|:---|:---|:---|
| `vcstack_instance` | name, flavor_id, image_id, network_ids, ssh_key_id | id, status, ip_address |
| `vcstack_network` | name, description, tenant_id | id, status |
| `vcstack_subnet` | name, cidr, gateway, network_id | id |
| `vcstack_router` | name, description, external_network_id, enable_snat | id, status, gateway_ip |
| `vcstack_volume` | name, size_gb, volume_type | id, status |
| `vcstack_security_group` | name, description, rules | id |
| `vcstack_floating_ip` | network_id | id, address |
| `vcstack_ssh_key` | name, public_key | id, fingerprint |

### Data Sources

| Data Source | Lookup By | Returns |
|:---|:---|:---|
| `vcstack_flavor` | name | vcpus, ram, disk |
| `vcstack_image` | name | status, disk_format, min_disk |
| `vcstack_network` | name | external, shared, status |

### Example

```hcl
data "vcstack_flavor" "medium" {
  name = "vc.medium"
}

data "vcstack_image" "ubuntu" {
  name = "Ubuntu 24.04"
}

resource "vcstack_network" "app" {
  name = "app-network"
}

resource "vcstack_subnet" "app" {
  name       = "app-subnet"
  cidr       = "10.0.1.0/24"
  gateway    = "10.0.1.1"
  network_id = vcstack_network.app.id
}

resource "vcstack_router" "main" {
  name        = "main-router"
  enable_snat = true
}

resource "vcstack_instance" "web" {
  name        = "web-server"
  flavor_id   = data.vcstack_flavor.medium.id
  image_id    = data.vcstack_image.ubuntu.id
  network_ids = [vcstack_network.app.id]
}
```

---

## vcctl CLI

### Global Flags

| Flag | Description |
|:---|:---|
| `--endpoint` | API endpoint URL |
| `--output`, `-o` | Output format: table, json, yaml |
| `--profile` | Configuration profile (default: `default`) |
| `--debug` | Enable debug logging |

### Commands

```
vcctl
├── compute           # VM, flavor, image, snapshot management
│   ├── instances     # list, create, delete, start, stop, reboot, show, console, resize
│   ├── flavors       # list, create, delete
│   ├── images        # list, delete
│   └── snapshots     # list, delete
├── network           # Network resource management
│   ├── list/create/delete
│   ├── subnets       # list, create, delete
│   ├── routers       # list, create, delete
│   ├── security-groups # list, create, delete
│   └── floating-ips  # list, create, delete
├── storage           # Block storage management
│   ├── volumes       # list, create, delete, attach, detach
│   └── snapshots     # list, create, delete
├── image             # OS image management (top-level shortcut)
│   ├── list, show, create, delete
├── identity          # IAM user management
├── cluster           # Cluster node management
├── config            # CLI configuration
├── secrets           # Secrets encryption/decryption
├── server            # Server management (start/status)
├── db                # Database operations (migrate/seed/backup)
└── version           # Version information
```

### Examples

```bash
# List instances
vcctl compute instances list

# Create a VM
vcctl compute instances create --name web-01 \
  --flavor-id 2 --image-id 5 --networks net-01

# Power operations
vcctl compute instances start i-abc123
vcctl compute instances stop i-abc123

# Image management
vcctl image list
vcctl image create --name "Ubuntu 24.04" --format qcow2 --url https://...
vcctl image show img-123

# Network management
vcctl network list
vcctl network routers create --name main-router

# Storage
vcctl storage volumes create --name data-vol --size-gb 100
vcctl storage volumes attach vol-123 --instance i-abc123

# Configuration
vcctl config set endpoint https://vc.example.com/api
vcctl config set token eyJhbGc...
```
