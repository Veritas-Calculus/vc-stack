# ============================================
# VC Stack Backend - Multi-stage Dockerfile
# ============================================
# Builds vc-management, vc-compute, and vcctl
# with full Ceph SDK support (go-ceph).
#
# Multi-platform: linux/amd64, linux/arm64
#
# Usage:
#   docker build --target vc-management -t vc-management .
#   docker build --target vc-compute -t vc-compute .
#   docker buildx build --platform linux/arm64 --target vc-management -t vc-management:arm64 .

# ---- Build stage (Debian for Ceph dev libs) ----
FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder
ENV GOTOOLCHAIN=auto

ARG TARGETOS=linux
ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends \
    git make \
    && rm -rf /var/lib/apt/lists/*

# Install Ceph dev headers for the TARGET architecture.
# For cross-compilation, we need the target-arch libs.
RUN if [ "$TARGETARCH" = "amd64" ]; then \
        dpkg --add-architecture amd64 || true; \
    elif [ "$TARGETARCH" = "arm64" ]; then \
        dpkg --add-architecture arm64 || true; \
    fi && \
    apt-get update && apt-get install -y --no-install-recommends \
    librados-dev librbd-dev libcephfs-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown

# Build management + vcctl (pure Go, no Ceph dependency)
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w \
    -X 'main.Version=${VERSION}' \
    -X 'main.Commit=${COMMIT}' \
    -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /out/vc-management ./cmd/vc-management

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w \
    -X 'main.Version=${VERSION}' \
    -X 'main.Commit=${COMMIT}' \
    -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /out/vcctl ./cmd/vcctl

# Build compute with Ceph SDK (CGO required for go-ceph).
# Falls back to a pure-Go build without Ceph if SDK headers are missing.
RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath -tags "ceph" \
    -ldflags="-s -w \
    -X 'main.Version=${VERSION}' \
    -X 'main.Commit=${COMMIT}' \
    -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /out/vc-compute ./cmd/vc-compute \
    || (echo "WARN: Ceph build failed, building without Ceph support" && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w \
    -X 'main.Version=${VERSION}' \
    -X 'main.Commit=${COMMIT}' \
    -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /out/vc-compute ./cmd/vc-compute)

# ---- Frontend build stage ----
FROM node:22-slim AS frontend-builder

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
    ca-certificates tzdata curl iproute2 iptables \
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

# Compute node runs as root (needs KVM/OVS access).
# SEC-06: In production, run with restricted capabilities:
#   docker run --cap-drop=ALL --cap-add=NET_ADMIN --cap-add=SYS_ADMIN \
#              --device=/dev/kvm --device=/dev/net/tun vc-compute
EXPOSE 8081
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/usr/local/bin/vc-compute", "healthcheck"] || curl -sf http://localhost:8081/health || exit 1

ENTRYPOINT ["compute-entrypoint.sh"]
CMD ["vc-compute"]
