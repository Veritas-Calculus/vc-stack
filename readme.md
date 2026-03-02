# VC Stack

<p align="center">
  <b>A Modern, Lightweight Cloud Infrastructure Platform (IaaS)</b>
</p>

VC Stack is designed as a simplified, highly-performant alternative to
OpenStack. It offers core cloud infrastructure services — compute, network,
storage, and identity — built on a modern technology stack. It empowers you
to build your own private cloud or edge computing infrastructure with
bare-metal performance and cloud-native management.

---

## Architecture

VC Stack is composed of two primary components and a unified CLI:

1. **`vc-management`** (Management Plane)
   - The centralized controller of your cloud.
   - Responsible for IAM (Identity & Access Management), OVN network
     orchestration, VM scheduling, quota management, and exposing the
     public REST API.
   - Hosts the integrated React-based Web Console.

2. **`vc-compute`** (Compute Node Agent)
   - The worker agent running on the physical hypervisor nodes.
   - Manages local Virtual Machines (via QEMU/KVM).
   - Configures local networking (via OVN/OVS integration).
   - Handles localized storage operations (Ceph/RBD).

3. **`vcctl`** (Command Line Interface)
   - The official, AWS CLI-inspired tool for interacting with the
     VC Stack API.
   - Provides full management capabilities:
     `vcctl compute instances create ...`, `vcctl network list`,
     `vcctl secrets init`, etc.

## Technology Stack

- **Backend**: Go 1.24+, Gin, GORM, gRPC/Protobuf, Cobra, Zap Logger.
- **Frontend**: React 18+, TypeScript, TailwindCSS, Vite, Zustand,
  xterm.js (WebShell), noVNC.
- **Data & Metrics**: PostgreSQL (Primary), Redis (Cache/Session),
  InfluxDB (Metrics).
- **Core Infrastructure**:
  - Virtualization: **QEMU / KVM**
  - SDN (Software Defined Networking): **OVN / OVS**
  - Distributed Storage: **Ceph / RBD**

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Node.js 18+ and npm
- Docker and Docker Compose (for local database & infrastructure)
- `make`, `protoc` (if modifying API definitions)

### 1. Build the Project

```bash
# Clone the repository
git clone https://github.com/Veritas-Calculus/vc-stack.git
cd vc-stack

# Build all binaries (vc-management, vc-compute, vcctl)
make build
```

Binaries will be output to the `bin/` directory.

### 2. Start Local Development Infrastructure

VC Stack requires a PostgreSQL database and optionally Redis/InfluxDB.
You can start these quickly using Docker Compose:

```bash
make dev-start
```

### 3. Setup Configuration

Generate a master encryption key (CloudStack-style encrypted secrets
are supported):

```bash
# Generate the key and save it
./bin/vcctl secrets init -f /tmp/master.key
export VC_MASTER_KEY=$(cat /tmp/master.key)

# Optionally encrypt your database password for the config
./bin/vcctl secrets encrypt "your-db-password"
```

Copy the example configurations and adjust them:

```bash
cp configs/env/controller.env.example configs/env/controller.env
cp configs/env/node.env.example configs/env/node.env
```

### 4. Run the Services

**Run the Management Plane:**

```bash
export $(grep -v '^#' configs/env/controller.env | xargs)
./bin/vc-management
```

**Run the Compute Agent (requires root/sudo for KVM/OVS):**

```bash
export $(grep -v '^#' configs/env/node.env | xargs)
sudo -E ./bin/vc-compute
```

### 5. Access the Web Console

Navigate to `http://localhost:8080` in your browser.

If `ADMIN_DEFAULT_PASSWORD` wasn't set in the environment, check the
`vc-management` startup logs for the auto-generated secure password.

## Development Commands

| Command | Description |
| :--- | :--- |
| `make build` | Compiles the backend binaries into `bin/`. |
| `make test` | Runs backend unit tests with race detection. |
| `make lint` | Runs `golangci-lint` to ensure code quality. |
| `make proto` | Regenerates Go code from Protobuf definitions. |
| `npm run dev` | *(in `web/console/`)* Starts the Vite dev server. |

## Security

VC Stack enforces strict security policies out-of-the-box:

- **No Hardcoded Passwords**: All initial credentials must be injected
  via environment variables or are auto-generated.
- **CloudStack-style Encrypted Storage**: Passwords in configurations
  can be wrapped in `ENC(...)` and dynamically decrypted via AES-256-GCM
  using a Master Key.
- **Strict Validation**: All network CIDRs, usernames, and IDs flow
  through rigid security checks to prevent injection.

See [docs/SECURITY.md](docs/SECURITY.md) for full production deployment
hardening guidelines.

## Codebase Layout

```text
api/proto/          Protobuf API definitions
cmd/
  vc-management/    Management plane entry point
  vc-compute/       Compute node entry point
  vcctl/            CLI utility
configs/            Example config templates and systemd unit files
docs/               Architectural and security documentation
internal/
  management/       API endpoints, Identity, Scheduling, OVN config
  compute/          VM orchestration, hypervisor drivers, OVS logic
    vm/             Low-level QEMU/KVM driver
    network/        OVN/OVS network agent
pkg/                Shared libraries (Security, Database, Logger, Models)
web/console/        React administration dashboard
migrations/         PostgreSQL schema definitions
```

## License

VC Stack is open-source software. See the LICENSE file for details.
