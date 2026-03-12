# VC Stack Terraform Provider

Terraform provider for managing VC Stack IaaS resources.

## Authentication

Supports two modes:

```hcl
# Option 1: API Key (Service Account)
provider "vcstack" {
  endpoint      = "https://vc.example.com/api"
  access_key_id = "VC-AKIA-0123456789abcdef"
  secret_key    = "your-secret-key"
}

# Option 2: JWT Token
provider "vcstack" {
  endpoint = "https://vc.example.com/api"
  token    = "eyJhbGciOiJIUzI1NiIs..."
}
```

Environment variables: `VCSTACK_ENDPOINT`, `VCSTACK_ACCESS_KEY_ID`, `VCSTACK_SECRET_KEY`, `VCSTACK_TOKEN`.

## Resources

| Resource | Description |
|:---|:---|
| `vcstack_instance` | Compute VM instance |
| `vcstack_network` | Virtual network |
| `vcstack_subnet` | Subnet within a network |
| `vcstack_volume` | Block storage volume |
| `vcstack_security_group` | Network security group |
| `vcstack_floating_ip` | Public floating IP |
| `vcstack_ssh_key` | SSH key pair |

## Data Sources

| Data Source | Description |
|:---|:---|
| `vcstack_flavor` | Look up flavor by name |
| `vcstack_image` | Look up image by name |

## Example

```hcl
terraform {
  required_providers {
    vcstack = {
      source = "veritas-calculus/vcstack"
    }
  }
}

provider "vcstack" {
  endpoint      = "https://vc.example.com/api"
  access_key_id = var.vc_access_key_id
  secret_key    = var.vc_secret_key
}

data "vcstack_flavor" "medium" {
  name = "m1.medium"
}

data "vcstack_image" "ubuntu" {
  name = "ubuntu-24.04"
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

resource "vcstack_security_group" "web" {
  name        = "web-sg"
  description = "Allow HTTP/HTTPS"
}

resource "vcstack_ssh_key" "deploy" {
  name       = "deploy-key"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "vcstack_instance" "web" {
  name       = "web-server-01"
  flavor_id  = data.vcstack_flavor.medium.id
  image_id   = data.vcstack_image.ubuntu.id
  network_id = vcstack_network.app.id
  ssh_key_id = vcstack_ssh_key.deploy.id
}

resource "vcstack_volume" "data" {
  name    = "web-data"
  size_gb = 50
}

resource "vcstack_floating_ip" "web" {
  network_id = vcstack_network.app.id
  tenant_id  = "1"
}

output "web_ip" {
  value = vcstack_instance.web.ip_address
}

output "floating_ip" {
  value = vcstack_floating_ip.web.address
}
```

## Building

```bash
cd terraform-provider-vcstack
go build -o terraform-provider-vcstack .
```
