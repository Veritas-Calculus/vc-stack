# ============================================
# VC Stack Backend - Multi-stage Dockerfile
# ============================================
# Builds vc-management, vc-compute, and vcctl
# with full Ceph SDK support (go-ceph).
#
# Usage:
#   docker build --target vc-management -t vc-management .
#   docker build --target vc-compute -t vc-compute .

# ---- Build stage (Debian for Ceph dev libs) ----
FROM golang:latest AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    git make \
    librados-dev librbd-dev libcephfs-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown

# Build with Ceph SDK (CGO required for go-ceph)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -trimpath -tags "ceph" \
    -ldflags="-s -w \
    -X 'main.Version=${VERSION}' \
    -X 'main.Commit=${COMMIT}' \
    -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /out/vc-management ./cmd/vc-management

RUN CGO_ENABLED=1 GOOS=linux go build \
    -trimpath -tags "ceph" \
    -ldflags="-s -w \
    -X 'main.Version=${VERSION}' \
    -X 'main.Commit=${COMMIT}' \
    -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /out/vc-compute ./cmd/vc-compute

RUN CGO_ENABLED=1 GOOS=linux go build \
    -trimpath -tags "ceph" \
    -ldflags="-s -w \
    -X 'main.Version=${VERSION}' \
    -X 'main.Commit=${COMMIT}' \
    -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /out/vcctl ./cmd/vcctl

# ---- Frontend build stage ----
FROM node:lts-slim AS frontend-builder

WORKDIR /app
COPY web/console/package.json web/console/package-lock.json* ./
RUN npm ci --prefer-offline
COPY web/console/ ./
RUN npm run build

# ---- vc-management runtime ----
FROM debian:bookworm-slim AS vc-management

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata curl \
    ovn-common \
    librados2 librbd1 libcephfs2 \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r vcstack && useradd -r -g vcstack vcstack

# Pre-create directories before switching to non-root user
RUN mkdir -p /tmp/vcstack/images && chown -R vcstack:vcstack /tmp/vcstack

COPY --from=builder /out/vc-management /usr/local/bin/vc-management
COPY --from=builder /out/vcctl /usr/local/bin/vcctl
COPY --from=frontend-builder /app/dist /opt/vc-stack/web/console/dist
COPY migrations/ /opt/vc-stack/migrations/

WORKDIR /opt/vc-stack
USER vcstack

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -sf http://localhost:8080/health || exit 1

ENTRYPOINT ["vc-management"]

# ---- vc-compute runtime ----
FROM debian:bookworm-slim AS vc-compute

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata curl iproute2 \
    qemu-system-x86 qemu-system-arm qemu-utils \
    qemu-efi-aarch64 ovmf \
    genisoimage swtpm \
    openvswitch-switch ovn-host \
    ceph-common librados2 librbd1 libcephfs2 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/vc-compute /usr/local/bin/vc-compute
COPY --from=builder /out/vcctl /usr/local/bin/vcctl
COPY scripts/compute-entrypoint.sh /usr/local/bin/compute-entrypoint.sh

WORKDIR /opt/vc-stack

# Compute node runs as root (needs KVM/OVS access)
EXPOSE 8081
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -sf http://localhost:8081/health || exit 1

ENTRYPOINT ["compute-entrypoint.sh"]
CMD ["vc-compute"]
