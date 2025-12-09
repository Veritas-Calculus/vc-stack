# VC Stack Makefile
# Follows Google's build system conventions

.PHONY: help build test clean install lint fmt vet proto docker deploy

# Variables
PROJECT_NAME := vc-stack
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build variables
GO_VERSION := 1.21
GOPATH := $(shell go env GOPATH)
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Build flags following Google's recommendations
LDFLAGS := -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.Commit=$(COMMIT)' -X 'main.BuildTime=$(BUILD_TIME)'"
BUILD_FLAGS := -trimpath $(LDFLAGS)
# Optional Go build tags (e.g., ovn_sdk). Can be overridden: `GO_BUILD_TAGS=ovn_sdk make build`.
GO_BUILD_TAGS ?=

# Enable libvirt-backed vc-lite build. Set to 1 to enable CGO/libvirt build.
# Default 0 makes vc-lite build without CGO so it can be cross-platform.
ENABLE_LIBVIRT ?= 0

# Directories
BUILD_DIR := ./build
BIN_DIR := ./bin
PROTO_DIR := ./api/proto
DOCS_DIR := ./docs

# Services
## Reduced service set: build controller and node binaries, plus vcctl CLI.
## These combined binaries aggregate multiple previously-separate services
## to simplify deployment (controller: control-plane services, node: compute/node services).
SERVICES := vc-controller vc-node vcctl

# Mark service targets as phony so Make won't treat existing files in repo root
# (e.g., ./vc-compute) as up-to-date targets and skip building into ./bin/
.PHONY: $(SERVICES)

help: ## Display this help message
	@echo "VC Stack Build System"
	@echo "Usage: make [target]"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ Development

fmt: ## Format Go code
	@echo "Formatting Go code..."
	@go fmt ./...
	@goimports -w .

lint: ## Run linters
	@echo "Running linters..."
	@# Use golangci-lint from PATH if available, otherwise fall back to GOPATH/bin
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		if [ -x "$(GOPATH)/bin/golangci-lint" ]; then \
			"$(GOPATH)/bin/golangci-lint" run ./...; \
		else \
			echo "golangci-lint not found; run 'make install-tools' or add GOPATH/bin to PATH"; exit 1; \
		fi; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

test: ## Run tests
	@echo "Running tests..."
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

test-coverage: ## Run tests with detailed coverage
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -tags=integration ./test/integration/...

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

##@ Code Quality

sonar: ## Run SonarQube analysis locally
	@echo "Running SonarQube analysis..."
	@if ! command -v sonar-scanner >/dev/null 2>&1; then \
		echo "Error: sonar-scanner not found. Please install from https://docs.sonarqube.org/latest/analysis/scan/sonarscanner/"; \
		exit 1; \
	fi
	@$(MAKE) test-coverage
	@golangci-lint run --out-format checkstyle > golangci-lint-report.xml || true
	@sonar-scanner

quality-check: lint vet test-coverage ## Run all quality checks
	@echo "All quality checks passed!"

security-scan: ## Run security scanner (gosec)
	@echo "Running security scan..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -fmt=json -out=gosec-report.json ./... || true
	@gosec ./...

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Development tools installed successfully"

##@ Build

build: build-all ## Build all services

build-all: $(SERVICES) ## Build all services
	@echo "Built all services successfully"


$(SERVICES): ## Build individual service
	@echo "Building $@..."
	@mkdir -p $(BIN_DIR)
	@if [ "$@" = "vc-lite" ]; then \
		if [ "$(ENABLE_LIBVIRT)" = "1" ]; then \
			echo "Building vc-lite with libvirt support (CGO enabled)..."; \
			CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -tags "libvirt $(GO_BUILD_TAGS)" $(BUILD_FLAGS) -o $(BIN_DIR)/$@ ./cmd/$@; \
		else \
			echo "Building vc-lite without libvirt (CGO disabled) for cross-platform compatibility..."; \
			CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -tags "$(GO_BUILD_TAGS)" $(BUILD_FLAGS) -o $(BIN_DIR)/$@ ./cmd/$@; \
		fi; \
	else \
		CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -tags "$(GO_BUILD_TAGS)" $(BUILD_FLAGS) -o $(BIN_DIR)/$@ ./cmd/$@; \
	fi

# Convenience target to build with OVN SDK implementation enabled
.PHONY: build-ovn-sdk
build-ovn-sdk: ## Build all services with OVN SDK enabled (GO build tag ovn_sdk)
	@echo "Building all services with ovn_sdk tag..."
	@GO_BUILD_TAGS=ovn_sdk $(MAKE) build-all

build-linux: ## Build for Linux
	@echo "Building for Linux..."
	@mkdir -p $(BIN_DIR)/linux
	@for service in $(SERVICES); do \
		if [ "$$service" = "vc-lite" ]; then \
			echo "Building $$service for Linux with libvirt (CGO)..."; \
			CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags libvirt $(BUILD_FLAGS) -o $(BIN_DIR)/linux/$$service ./cmd/$$service; \
		else \
			echo "Building $$service for Linux..."; \
			CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BIN_DIR)/linux/$$service ./cmd/$$service; \
		fi; \
	done

proto: ## Generate protobuf files
	@echo "Generating protobuf files..."
	@protoc --proto_path=$(PROTO_DIR) \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

##@ Docker

docker: docker-build ## Build Docker images

docker-build: ## Build Docker images for all services
	@echo "Building Docker images..."
	@for service in $(SERVICES); do \
		echo "Building $$service image..."; \
		docker build -f build/docker/$$service/Dockerfile -t vcstack/$$service:$(VERSION) .; \
	done

docker-push: ## Push Docker images
	@echo "Pushing Docker images..."
	@for service in $(SERVICES); do \
		echo "Pushing $$service image..."; \
		docker push vcstack/$$service:$(VERSION); \
	done

##@ Dependencies

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

deps-vendor: ## Create vendor directory
	@echo "Creating vendor directory..."
	@go mod vendor

##@ Tools

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	@go install github.com/swaggo/swag/cmd/swag@latest

generate: ## Generate code
	@echo "Generating code..."
	@go generate ./...

docs: ## Generate documentation
	@echo "Generating API documentation..."
	@swag init -g cmd/vc-gateway/main.go -o $(DOCS_DIR)/api

##@ Database

migrate-up: ## Run database migrations up
	@echo "Running database migrations up..."
	@migrate -path=./migrations -database="postgres://localhost/vcstack?sslmode=disable" up

migrate-down: ## Run database migrations down
	@echo "Running database migrations down..."
	@migrate -path=./migrations -database="postgres://localhost/vcstack?sslmode=disable" down

migrate-create: ## Create new migration (usage: make migrate-create NAME=create_users)
	@echo "Creating migration $(NAME)..."
	@migrate create -ext sql -dir ./migrations $(NAME)

##@ Development Environment

dev-start: ## Start development environment
	@echo "Starting development environment..."
	@docker-compose -f deployments/docker-compose.dev.yml up -d

dev-stop: ## Stop development environment
	@echo "Stopping development environment..."
	@docker-compose -f deployments/docker-compose.dev.yml down

dev-logs: ## Show development environment logs
	@docker-compose -f deployments/docker-compose.dev.yml logs -f

##@ Deployment

deploy-k8s: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	@kubectl apply -f deployments/kubernetes/

deploy-helm: ## Deploy using Helm
	@echo "Deploying using Helm..."
	@helm upgrade --install vc-stack deployments/helm/vc-stack

##@ Cleanup

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(BIN_DIR)
	@rm -f coverage.out coverage.html
	@go clean -cache -testcache -modcache

clean-docker: ## Clean Docker images and containers
	@echo "Cleaning Docker artifacts..."
	@docker system prune -f
	@for service in $(SERVICES); do \
		docker rmi vcstack/$$service:$(VERSION) 2>/dev/null || true; \
	done

##@ Security

security-scan: ## Run security scan
	@echo "Running security scan..."
	@gosec ./...

vulnerability-check: ## Check for vulnerabilities
	@echo "Checking for vulnerabilities..."
	@govulncheck ./...

##@ Release

release: clean test lint build docker ## Build release artifacts
	@echo "Release $(VERSION) built successfully"

##@ Git Hooks

pre-commit-install: ## Install pre-commit hooks
	@echo "Installing pre-commit hooks..."
	@if ! command -v pre-commit >/dev/null 2>&1; then \
		echo "pre-commit not found. Installing via pip..."; \
		pip install pre-commit || pip3 install pre-commit; \
	fi
	@pre-commit install
	@pre-commit install --hook-type commit-msg
	@echo "✅ Pre-commit hooks installed successfully"

pre-commit-run: ## Run pre-commit hooks on all files
	@echo "Running pre-commit on all files..."
	@pre-commit run --all-files

pre-commit-update: ## Update pre-commit hooks
	@echo "Updating pre-commit hooks..."
	@pre-commit autoupdate

pre-commit-uninstall: ## Uninstall pre-commit hooks
	@echo "Uninstalling pre-commit hooks..."
	@pre-commit uninstall
	@pre-commit uninstall --hook-type commit-msg

##@ Miscellaneous

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "OS/Arch: $(GOOS)/$(GOARCH)"
