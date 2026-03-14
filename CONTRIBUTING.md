# Contributing to VC Stack

Thank you for your interest in contributing to VC Stack! This guide will help you get started.

## Prerequisites

- **Go** 1.24+
- **Node.js** 18+ and **npm** 9+
- **Docker** and **Docker Compose** (for local development)
- **protoc** (only if modifying `.proto` files)

## Getting Started

```bash
# Clone and set up
git clone https://github.com/Veritas-Calculus/vc-stack.git
cd vc-stack
make install-tools
make pre-commit-install

# Start dev infrastructure (PostgreSQL)
make dev-start

# Build all binaries
make build

# Run backend tests
make test

# Frontend development
cd web/console
npm install
npm run dev     # Start Vite dev server
npm run test    # Run Vitest
npm run lint    # ESLint + Prettier
```

## Project Structure

```
vc-stack/
├── api/proto/          # Protobuf API definitions
├── cmd/                # Binary entrypoints
│   ├── vc-management/  # Management plane
│   ├── vc-compute/     # Compute node agent
│   └── vcctl/          # CLI tool
├── internal/
│   ├── management/     # Management plane (modular sub-packages)
│   └── compute/        # Compute node (vm/, network/)
├── pkg/                # Shared utility packages
├── web/console/        # React dashboard
├── migrations/         # PostgreSQL migration files
└── configs/            # YAML configs, systemd units
```

## Development Workflow

### 1. Create a Branch

Use feature branches from `main`:

```bash
git checkout -b feat/your-feature
```

### 2. Write Code

#### Backend (Go)

- Follow standard Go idioms and project conventions
- Run `make fmt` and `make lint` before committing
- New modules should implement the `Module` interface and self-register via the Registry pattern — do **not** add fields to the `Service` struct
- Use `appconfig` for configuration — do **not** use `os.Getenv()` directly
- Wrap errors with context: `fmt.Errorf("operation: %w", err)`
- Use `zap` logger (via `pkg/logger`), not `log` or `fmt.Println`
- Use Sentry for error tracking in production paths

#### Frontend (React/TypeScript)

- Use functional components with hooks
- Use TailwindCSS for styling
- API calls go in `src/lib/api/` — import the shared client from `./client`
- State management via Zustand (`appStore`, `dataStore`)
- No emoji in UI — use Lucide icons

### 3. Write Tests

#### Backend Tests

```bash
make test                                    # All backend tests
go test -v ./internal/management/config/...  # Specific package
go test -race -cover ./...                   # With race detection + coverage
```

Follow the established pattern using in-memory SQLite + `gin.TestMode`:

```go
func TestNewService(t *testing.T) {
    gin.SetMode(gin.TestMode)
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    svc := NewService(db)
    assert.NotNil(t, svc)
}
```

#### Frontend Tests

```bash
cd web/console
npm run test                    # All tests
npx vitest run src/__tests__/   # Specific directory
```

Use `vitest` + `@testing-library/react`. Mock API calls with `vi.mock`:

```tsx
vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return { ...actual, fetchFoo: vi.fn().mockResolvedValue([]) }
})
```

### 4. Commit

Use **Conventional Commits**:

```
feat: add subnet CIDR validation
fix: correct volume resize calculation
docs: update API authentication guide
chore: bump golangci-lint to v2.10
refactor: extract security group handlers
test: add config module unit tests
```

### 5. Submit a Pull Request

- Ensure all pre-commit hooks pass
- Ensure `make build`, `make test`, and `npm run test` pass
- Provide a clear description of the changes and motivation
- Link related issues

## Adding a New Backend Module

1. Create `internal/management/yourmodule/service.go`
2. Implement the `Module` interface:

   ```go
   type Service struct { db *gorm.DB }
   func (s *Service) Name() string { return "yourmodule" }
   func (s *Service) SetupRoutes(r *gin.RouterGroup) { /* routes */ }
   ```

3. Register in `internal/management/modules.go` module list
4. Add migration in `migrations/` if schema changes are needed
5. Write tests in `service_test.go`

## Adding a New Frontend Feature

1. Create `web/console/src/features/yourfeature/YourFeature.tsx`
2. Add a lazy import in `App.tsx`:

   ```tsx
   const YourFeature = lazy(() => import('@/features/yourfeature/YourFeature')
     .then(m => ({ default: m.YourFeature })))
   ```

3. Add a `<Route>` in `App.tsx`
4. Add sidebar navigation in `Layout.tsx`
5. Write tests in `src/__tests__/YourFeature.test.tsx`

## Security

- **Never** hardcode secrets. Use `vcctl secrets encrypt` for `ENC()` values
- Validate all user inputs, especially network configs and VM parameters
- See `docs/SECURITY.md` for production hardening

## Getting Help

- Open an issue for bugs or feature requests
- Check existing issues and PRs before creating new ones
