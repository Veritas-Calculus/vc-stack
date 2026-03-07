# VC Stack Deployment Guide

This guide covers deploying VC Stack in development, single-node,
and multi-node (cluster) configurations.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Development Setup](#development-setup)
- [Single-Node Deployment](#single-node-deployment)
- [Multi-Node Cluster](#multi-node-cluster)
- [Docker Compose (Production)](#docker-compose-production)
- [Helm (Kubernetes)](#helm-kubernetes)
- [Configuration Reference](#configuration-reference)
- [TLS and Security](#tls-and-security)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### All Environments

| Requirement | Version | Purpose |
|:---|:---|:---|
| Go | 1.24+ | Build backend binaries |
| Node.js | 18+ | Build frontend console |
| PostgreSQL | 15+ | Primary database |
| Make | any | Build automation |

### Production Only

| Requirement | Version | Purpose |
|:---|:---|:---|
| QEMU/KVM | 8.0+ | VM hypervisor |
| OVN/OVS | 23.03+ | Software-defined networking |
| Ceph | Reef (18)+ | Distributed block/object storage |
| systemd | 250+ | Service management |

---

## Development Setup

### 1. Clone and Build

```bash
git clone https://github.com/Veritas-Calculus/vc-stack.git
cd vc-stack

# Install Go tools (golangci-lint, protoc-gen-go, etc.)
make install-tools

# Build all binaries
make build
# Outputs: bin/vc-management, bin/vc-compute, bin/vcctl
```

### 2. Start Database

```bash
# Starts PostgreSQL 15 in Docker
make dev-start

# Confirm it's running
docker compose -f configs/docker-compose.dev.yaml ps
```

### 3. Initialize Secrets

```bash
# Generate a master encryption key
./bin/vcctl secrets init -f /etc/vc-stack/master.key

# Encrypt the DB password
./bin/vcctl secrets encrypt "postgres"
# Copy the ENC(...) output for config
```

### 4. Configure and Run

```bash
# Copy example config
cp configs/env/vc-management.env.example .env

# Edit .env — set DB_PASSWORD to your ENC() value
# Run management plane
./bin/vc-management
```

### 5. Frontend Development

```bash
cd web/console
npm install
npm run dev
# Open http://localhost:5173
```

### 6. Run Tests

```bash
# Backend
make test

# Frontend unit tests
cd web/console && npm run test

# Frontend E2E tests
cd web/console && npm run test:e2e
```

---

## Single-Node Deployment

A single-node deployment runs both `vc-management` and `vc-compute`
on the same machine. Suitable for small environments or evaluation.

### System Requirements

| Resource | Minimum | Recommended |
|:---|:---|:---|
| CPU | 4 cores | 8+ cores |
| RAM | 8 GB | 16+ GB |
| Disk | 100 GB SSD | 500+ GB NVMe |
| OS | Ubuntu 22.04 / Rocky 9 | Ubuntu 24.04 |

### Installation Steps

```bash
# 1. Install system dependencies
sudo apt update
sudo apt install -y qemu-kvm libvirt-daemon-system \
  openvswitch-switch ovn-host ovn-central \
  postgresql-15

# 2. Build VC Stack
make build

# 3. Install binaries
sudo cp bin/vc-management /usr/local/bin/
sudo cp bin/vc-compute /usr/local/bin/
sudo cp bin/vcctl /usr/local/bin/

# 4. Create config directory
sudo mkdir -p /etc/vc-stack
sudo cp configs/env/vc-management.env.example /etc/vc-stack/vc-management.env
sudo cp configs/env/vc-compute.env.example /etc/vc-stack/vc-compute.env

# 5. Initialize master key
sudo vcctl secrets init -f /etc/vc-stack/master.key
sudo chmod 0400 /etc/vc-stack/master.key

# 6. Setup database
sudo -u postgres createuser vcstack
sudo -u postgres createdb -O vcstack vcstack

# 7. Install systemd services
sudo cp configs/systemd/vc-management.service /etc/systemd/system/
sudo cp configs/systemd/vc-compute.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now vc-management vc-compute
```

### Verify

```bash
# Check services
sudo systemctl status vc-management vc-compute

# Test API
curl -s http://localhost:8080/api/v1/health | jq

# Login
vcctl server set http://localhost:8080
vcctl compute instance list
```

---

## Multi-Node Cluster

A multi-node deployment separates the management plane from compute nodes.

### Architecture

```text
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Management  │     │  Compute 1  │     │  Compute N  │
│   Node      │     │    Node     │     │    Node     │
│             │     │             │     │             │
│ vc-mgmt     │◄────│ vc-compute  │     │ vc-compute  │
│ PostgreSQL  │     │ QEMU/KVM    │     │ QEMU/KVM    │
│ OVN Central │     │ OVN Host    │     │ OVN Host    │
│ Web Console │     │ OVS         │     │ OVS         │
└─────────────┘     └─────────────┘     └─────────────┘
```

### Management Node Setup

Follow the single-node steps for the management plane only.
Do NOT run `vc-compute` on the management node.

### Compute Node Setup

```bash
# 1. Install hypervisor and networking
sudo apt install -y qemu-kvm openvswitch-switch ovn-host

# 2. Copy vc-compute binary
sudo cp bin/vc-compute /usr/local/bin/

# 3. Configure
sudo mkdir -p /etc/vc-stack
sudo cp configs/env/vc-compute.env.example /etc/vc-stack/vc-compute.env

# Edit to point at management node:
# VC_CONTROLLER_URL=http://management-node:8080
# VC_NODE_NAME=compute-01

# 4. Start
sudo systemctl enable --now vc-compute
```

### Network Configuration

Each compute node needs OVN/OVS configured:

```bash
# Set OVN encapsulation
sudo ovs-vsctl set open_vswitch . \
  external_ids:ovn-remote="tcp:<management-ip>:6642" \
  external_ids:ovn-encap-type="geneve" \
  external_ids:ovn-encap-ip="<compute-node-ip>"
```

---

## Docker Compose (Production)

A production-ready Docker Compose is provided for containerized deployments:

```bash
# Start all services
docker compose -f configs/docker-compose.yaml up -d

# Check status
docker compose -f configs/docker-compose.yaml ps
```

This starts: PostgreSQL, vc-management (with embedded web console),
and an nginx reverse proxy with TLS termination.

---

## Helm (Kubernetes)

For Kubernetes deployments, Helm charts are provided:

```bash
# Add the VC Stack Helm repository
helm repo add vc-stack https://charts.vc-stack.io

# Install the management plane
helm install vc-management vc-stack/vc-management \
  --namespace vc-system --create-namespace \
  --set database.host=postgres.default.svc \
  --set database.password=ENC\(your-encrypted-password\)

# Install compute agents (DaemonSet on worker nodes)
helm install vc-compute vc-stack/vc-compute \
  --namespace vc-system \
  --set controller.url=http://vc-management:8080
```

---

## Configuration Reference

### Management Plane (`vc-management.env`)

| Variable | Default | Description |
|:---|:---|:---|
| `VC_DB_HOST` | `localhost` | PostgreSQL host |
| `VC_DB_PORT` | `5432` | PostgreSQL port |
| `VC_DB_NAME` | `vcstack` | Database name |
| `VC_DB_USER` | `vcstack` | Database user |
| `VC_DB_PASSWORD` | — | DB password (supports `ENC()`) |
| `VC_LISTEN_ADDR` | `:8080` | API listen address |
| `VC_JWT_SECRET` | — | JWT signing secret |
| `VC_MASTER_KEY` | — | Master encryption key (alt to file) |
| `VC_LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `VC_SENTRY_DSN` | — | Sentry error tracking DSN |

### Compute Node (`vc-compute.env`)

| Variable | Default | Description |
|:---|:---|:---|
| `VC_CONTROLLER_URL` | `http://localhost:8080` | Management API URL |
| `VC_NODE_NAME` | hostname | Node identifier |
| `VC_STORAGE_BACKEND` | `local` | Storage: `local` or `ceph` |
| `VC_CEPH_POOL` | `vm-pool` | Ceph RBD pool name |
| `VC_DATA_DIR` | `/var/lib/vc-stack` | Local data directory |
| `VC_LOG_LEVEL` | `info` | Log level |

---

## TLS and Security

### Enable HTTPS

Use the provided nginx config for TLS termination:

```bash
sudo cp configs/nginx/vc-stack.conf /etc/nginx/sites-available/
sudo ln -s /etc/nginx/sites-available/vc-stack.conf /etc/nginx/sites-enabled/

# Install certificate (Let's Encrypt recommended)
sudo certbot --nginx -d cloud.example.com
```

### Master Key Management

The master key encrypts database credentials and sensitive configuration.

```bash
# Generate key
vcctl secrets init -f /etc/vc-stack/master.key

# Encrypt a value
vcctl secrets encrypt "my-secret-password"
# Output: ENC(aes256gcm:base64...)

# Use in config
VC_DB_PASSWORD=ENC(aes256gcm:base64...)
```

> **Warning**: The master key at `/etc/vc-stack/master.key` must have
> restrictive permissions (`0400`). Loss of this key means encrypted
> values cannot be recovered.

---

## Troubleshooting

### Management plane won't start

```bash
# Check logs
journalctl -u vc-management -f

# Verify PostgreSQL connectivity
psql -h localhost -U vcstack -d vcstack -c "SELECT 1"

# Check master key permissions
ls -la /etc/vc-stack/master.key
# Should be: -r-------- root root
```

### Compute node can't connect to management

```bash
# Verify network connectivity
curl -s http://<management-ip>:8080/api/v1/health

# Check compute logs
journalctl -u vc-compute -f

# Verify VC_CONTROLLER_URL is set correctly
grep CONTROLLER /etc/vc-stack/vc-compute.env
```

### VMs have no network

```bash
# Check OVN/OVS status
sudo ovs-vsctl show
sudo ovn-sbctl show

# Verify encapsulation
sudo ovs-vsctl get open_vswitch . external_ids

# Check tunnel connectivity
sudo ovs-appctl ofproto/list-tunnels
```

### Storage errors

```bash
# Local backend: check disk space
df -h /var/lib/vc-stack

# Ceph backend: verify pool
sudo rbd pool stats vm-pool
sudo ceph health detail
```
