# vcctl: The Unified CLI

`vcctl` is the official command-line tool for interacting with the VC Stack API. It is designed to be intuitive and familiar for users of other cloud platforms like AWS and OpenStack.

## Command Overview

The CLI follows a hierarchical command structure:
`vcctl <service> <resource> <action> [flags]`

### Major Services

| Service | Description | Command |
| :--- | :--- | :--- |
| **Compute** | Manage instances, images, and flavors. | `vcctl compute` |
| **Network** | Manage networks, subnets, routers, and firewalls. | `vcctl network` |
| **Storage** | Manage volumes and snapshots. | `vcctl storage` |
| **Identity** | Manage users, projects, and roles. | `vcctl identity` |
| **Cluster** | Manage compute nodes and zones. | `vcctl cluster` |
| **Server** | Manage local service startup. | `vcctl server` |
| **Secrets** | Manage encryption keys and secret values. | `vcctl secrets` |

## Feature Highlights

### 1. Secret Management

`vcctl` includes built-in tools for managing system secrets.

- **Initialization**: `vcctl secrets init` initializes the master key for the system.
- **Encryption**: `vcctl secrets encrypt <plain_text>` generates an `ENC()` wrapped value that can be safely stored in configuration files or the database.
- **Decryption**: `vcctl secrets decrypt <encrypted_value>` (Requires `VC_MASTER_KEY`).

### 2. Configuration Profiles

Manage multiple environments (e.g., staging, production) easily:

- `vcctl config init`: Initial configuration.
- `vcctl --profile <name>`: Switch between different profiles.

### 3. Output Formats

Control how data is presented using the `--output` flag:

- `table` (Default): Human-readable formatted table.
- `json`: Machine-readable JSON for integration with other tools like `jq`.
- `yaml`: Formatted YAML for configuration management.

### 4. Interactive Help

Comprehensive built-in documentation:

- `vcctl --help`: Global help.
- `vcctl <service> --help`: Service-specific commands.
- `vcctl <service> <resource> <action> --help`: Action-specific flags and examples.

## Example Usage

### Instance Lifecycle

```bash
# List all instances
vcctl compute instances list

# Create a new instance
vcctl compute instances create --name my-instance --flavor standard-1 --image ubuntu-22.04

# Start/Stop an instance
vcctl compute instances start vm-123
vcctl compute instances stop vm-123
```

### Networking

```bash
# List available subnets
vcctl network subnets list

# Associate a floating IP
vcctl network floating-ips associate vm-123 --ip 1.2.3.4
```

## Internal Architecture

`vcctl` is built using **Cobra**, a powerful library for creating CLI applications in Go. It acts as a client for the `vc-management` REST API.

- **`client.go`**: Core API client logic and HTTP request handling.
- **Command Files**: Each service (compute, network, etc.) has its own file in `cmd/vcctl/`.
- **Formatting**: Centralized output formatting logic for consistency across all commands.
