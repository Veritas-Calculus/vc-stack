# VC Stack: Gemini CLI Context & Instructions

This document provides essential context and instructions for the VC Stack project, a modern, lightweight
IaaS (Infrastructure as a Service) platform.

## Project Overview

VC Stack is designed as a simplified alternative to OpenStack, offering core cloud infrastructure services
(compute, network, storage, identity) using a modern tech stack.

### Architecture

The project follows a two-component architecture for simplified deployment:

- **`vc-management`**: The centralized management plane. It aggregates services like identity (IAM),
  networking (OVN orchestration), scheduling, and global compute management.
- **`vc-compute`**: The compute node agent. It manages local virtual machines (via QEMU/KVM),
  configures local networking (via OVN/OVS), and handles storage (Ceph/RBD).
- **`vcctl`**: The official CLI tool for interacting with the VC Stack API.

### Technical Stack

- **Backend**: Go 1.24+, Gin (Web Framework), GORM (ORM), gRPC/Protobuf, Cobra (CLI), Zap (Logging),
  Sentry (Error Tracking).
- **Frontend**: React 18+, TypeScript, TailwindCSS, Vite, Zustand (State Management), xterm.js (WebShell), noVNC.
- **Database**: PostgreSQL (Primary), Redis (Cache/Session), InfluxDB (Metrics).
- **Infrastructure**: OVN/OVS (Software Defined Networking), Ceph/RBD (Distributed Storage),
  QEMU/KVM (Virtualization).

---

## Development Workflow

### Prerequisites

- Go 1.21 or higher
- Node.js 18+ and npm
- Docker and Docker Compose (for local development environment)
- Protobuf compiler (`protoc`) if modifying APIs

### Key Commands

#### Backend (Root Directory)

| Command | Description |
| :--- | :--- |
| `make build` | Builds all binaries (`vc-management`, `vc-compute`, `vcctl`) into `bin/`. |
| `make test` | Runs all backend tests with race detection and coverage. |
| `make lint` | Runs `golangci-lint` for static analysis. |
| `make proto` | Generates Go code from Protobuf definitions in `api/proto/`. |
| `make dev-start` | Starts the development infrastructure (DBs, etc.) using Docker Compose. |
| `make install-tools` | Installs necessary Go development tools (`golangci-lint`, `protoc-gen-go`, etc.). |
| `make pre-commit-install` | Installs the pre-commit hooks for consistent code quality. |

#### Frontend (`web/console/`)

| Command | Description |
| :--- | :--- |
| `npm run dev` | Starts the Vite development server. |
| `npm run build` | Builds the production-ready frontend assets. |
| `npm run lint` | Runs ESLint and Prettier checks. |
| `npm run test` | Runs frontend unit tests using Vitest. |

---

## Codebase Structure

- `api/proto/`: Protobuf API definitions.
- `cmd/`: Entry points for `vc-management`, `vc-compute`, and `vcctl`.
- `configs/`: Example configuration files and systemd service templates.
- `docs/`: Technical documentation, including security and API guides.
- `internal/`: Core business logic.
  - `management/`: Implementation of all management plane services.
  - `compute/`: Compute node logic (VM orchestration, lifecycle management).
    - `vm/`: Low-level VM driver (QEMU/KVM).
    - `network/`: OVN/OVS network agent.
- `pkg/`: Shared utility packages (logger, database, security, sentry).
- `web/console/`: React-based management dashboard.
- `migrations/`: SQL migration files for PostgreSQL.

---

## Development Conventions

### Coding Standards

- **Go**: Follow standard Go idioms. Use `make fmt` and `make lint` before submitting.
- **Frontend**: Use functional components with hooks. Prefer TailwindCSS for styling.
- **Error Handling**: Use the central logging and Sentry integration. Wrap errors with context where appropriate.

### Security

- **Credentials**: NEVER hardcode secrets. Use environment variables.
- **Validation**: Rigorously validate all user inputs, especially for network configurations and VM parameters.
- **Documentation**: Refer to `docs/SECURITY.md` for production hardening guidelines.

### Git & Commits

- Use **Conventional Commits** (e.g., `feat:`, `fix:`, `docs:`, `chore:`).
- Ensure all pre-commit hooks pass.

---

## Interaction Instructions for Gemini CLI

- **Researching**: When investigating issues, check both the management and compute logic as they are often
  interdependent.
- **Testing**: Always run `make test` after backend changes and `npm run test` after frontend changes.
- **API Changes**: If modifying Protobuf files, remember to run `make proto` and update both backend services
  and frontend clients.
- **Database**: New features requiring schema changes must include a new migration file in `migrations/`.
