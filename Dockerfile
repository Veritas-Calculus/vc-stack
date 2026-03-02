# ============================================
# VC Stack Backend - Multi-stage Dockerfile
# ============================================
# Builds vc-management, vc-compute, and vcctl
# as statically linked binaries.
#
# Usage:
#   docker build -t vc-stack .
#   docker build --target vc-management -t vc-management .
#   docker build --target vc-compute -t vc-compute .

# ---- Build stage ----
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build all binaries statically (no CGO, no Ceph SDK)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath -tags "netgo osusergo" \
    -ldflags="-s -w -extldflags '-static' \
    -X 'main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)' \
    -X 'main.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)' \
    -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /out/vc-management ./cmd/vc-management

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath -tags "netgo osusergo" \
    -ldflags="-s -w -extldflags '-static' \
    -X 'main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)' \
    -X 'main.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)' \
    -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /out/vc-compute ./cmd/vc-compute

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath -tags "netgo osusergo" \
    -ldflags="-s -w -extldflags '-static' \
    -X 'main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)' \
    -X 'main.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)' \
    -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /out/vcctl ./cmd/vcctl

# ---- Frontend build stage ----
FROM node:18-alpine AS frontend-builder

WORKDIR /app
COPY web/console/package.json web/console/package-lock.json* ./
RUN npm ci --prefer-offline
COPY web/console/ ./
RUN npm run build

# ---- vc-management runtime ----
FROM alpine:3.20 AS vc-management

RUN apk add --no-cache ca-certificates tzdata curl
RUN addgroup -S vcstack && adduser -S vcstack -G vcstack

COPY --from=builder /out/vc-management /usr/local/bin/vc-management
COPY --from=builder /out/vcctl /usr/local/bin/vcctl
COPY --from=frontend-builder /app/dist /opt/vc-stack/web/console/dist
COPY migrations/ /opt/vc-stack/migrations/

WORKDIR /opt/vc-stack
USER vcstack

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["vc-management"]

# ---- vc-compute runtime ----
FROM alpine:3.20 AS vc-compute

# Compute node needs qemu, ovs, and rbd CLI tools
RUN apk add --no-cache ca-certificates tzdata curl \
    qemu-system-x86_64 qemu-img \
    openvswitch ovn \
    ceph-common

COPY --from=builder /out/vc-compute /usr/local/bin/vc-compute
COPY --from=builder /out/vcctl /usr/local/bin/vcctl

WORKDIR /opt/vc-stack

# Compute node runs as root (needs KVM/OVS access)
EXPOSE 8081
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8081/health || exit 1

ENTRYPOINT ["vc-compute"]
