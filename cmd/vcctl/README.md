# vcctl - VC Stack CLI

`vcctl` is a unified command-line interface for VC Stack, inspired by AWS CLI.
It provides intuitive commands to manage your cloud infrastructure.

## Quick Start

```bash
# Build vcctl
make build

# Show version
./bin/vcctl version

# Get help
./bin/vcctl --help

# Configure vcctl
./bin/vcctl config init
```

## Features

- **Unified Interface**: Single CLI for all VC Stack operations
- **Hierarchical Commands**: Organized by service (compute, network, storage, cluster, identity)
- **Multiple Output Formats**: table, json, yaml
- **Configuration Profiles**: Manage multiple environments
- **AWS-like Experience**: Familiar command structure

## Command Structure

```
vcctl [global-flags] <service> <resource> <action> [flags]
```

### Examples

```bash
# Compute Management
vcctl compute instances list
vcctl compute instances create --name my-vm --vcpus 2 --memory 4096
vcctl compute instances start <instance-id>
vcctl compute instances stop <instance-id>
vcctl compute instances delete <instance-id>
vcctl compute flavors list
vcctl compute images list

# Network Management
vcctl network list all
vcctl network subnets list
vcctl network routers list
vcctl network security-groups list
vcctl network floating-ips list

# Storage Management
vcctl storage volumes list
vcctl storage volumes create --name my-vol --size 100
vcctl storage volumes attach <volume-id> <instance-id>
vcctl storage snapshots list

# Cluster Management
vcctl cluster nodes list
vcctl cluster nodes show <node-id>
vcctl cluster nodes enable <node-id>
vcctl cluster nodes disable <node-id>
vcctl cluster nodes maintenance <node-id>
vcctl cluster zones list

# Identity Management
vcctl identity users list
vcctl identity projects list
vcctl identity roles list

# Server Management (Start services)
vcctl server controller  # Start control plane
vcctl server node        # Start compute node
vcctl server lite        # Start lite agent
```

## Global Flags

- `--endpoint`: API endpoint URL (default from config or env)
- `--output, -o`: Output format: table, json, yaml (default "table")
- `--profile`: Configuration profile to use (default "default")
- `--debug`: Enable debug logging

## Configuration

vcctl uses a configuration file at `~/.vcctl/config`:

```ini
[default]
endpoint = http://localhost:8080
output = table

[production]
endpoint = https://vcstack.example.com
output = json
```

## Command Aliases

Many commands have convenient aliases:

- `compute` → `ec2`, `vm`
- `network` → `net`, `vpc`
- `storage` → `vol`, `volume`
- `cluster` → `node`, `host`
- `identity` → `iam`, `auth`
- `instances` → `instance`, `vms`, `vm`
- `floating-ips` → `fip`, `eip`
- `security-groups` → `sg`, `secgroup`

## Architecture

### Directory Structure

```
cmd/vcctl/
├── main.go          # Entry point and root command
├── compute.go       # Compute instance commands
├── network.go       # Network management commands
├── storage.go       # Storage volume commands
├── cluster.go       # Cluster node commands
├── identity.go      # Identity and access commands
├── server.go        # Server startup commands
└── config.go        # Configuration commands
```

### Command Organization

- **compute**: VM instances, flavors, images, snapshots
- **network**: Networks, subnets, routers, security groups, floating IPs
- **storage**: Volumes, snapshots
- **cluster**: Nodes (hosts), zones
- **identity**: Users, projects, roles
- **server**: Start VC Stack services (controller, node, lite)
- **config**: CLI configuration management

## Development

### Adding New Commands

1. Add command definition in the appropriate file (e.g., `compute.go`)
2. Implement the handler function
3. Add API client calls (TODO: implement API client)
4. Add tests

### Building

```bash
# Build all services including vcctl
make build

# Build only vcctl
make vcctl

# Build for specific platform
GOOS=linux GOARCH=amd64 make vcctl
```

## Roadmap

- [ ] Implement API client library
- [ ] Add configuration file support
- [ ] Implement all command handlers
- [ ] Add bash/zsh completion
- [ ] Add interactive mode
- [ ] Add batch operations
- [ ] Add JSON/YAML input for complex operations
- [ ] Add output formatting and filtering
- [ ] Add progress indicators for long operations
- [ ] Add dry-run mode

## Comparison with Legacy Commands

### Before (Multiple Binaries)

```bash
# Old way - separate binaries for each service
./bin/vc-controller    # Control plane
./bin/vc-node          # Compute node
./bin/vc-scheduler     # Scheduler (deprecated)
./bin/vc-gateway       # Gateway (deprecated)
./bin/vc-identity      # Identity (deprecated)
./bin/vc-network       # Network (deprecated)
./bin/vc-compute       # Compute (deprecated)
./bin/vc-lite          # Lite agent (deprecated)
```

### After (Unified CLI)

```bash
# New way - single CLI with subcommands
./bin/vcctl compute instances list     # Manage instances
./bin/vcctl cluster nodes list         # Manage nodes
./bin/vcctl network list              # Manage networks
./bin/vcctl server controller         # Start controller
./bin/vcctl server node               # Start node
```

## Benefits

1. **Simplified Operations**: One tool to remember instead of many
2. **Consistent Interface**: All commands follow the same pattern
3. **Better UX**: AWS-like experience, familiar to cloud operators
4. **Easier Automation**: Predictable command structure for scripting
5. **Self-documenting**: Built-in help at every level
6. **Extensible**: Easy to add new commands and features

## License

Same as VC Stack project.
