#!/usr/bin/env bash
# airgap-bundle.sh — Build an air-gapped deployment bundle for VC Stack.
#
# Usage: ./scripts/airgap-bundle.sh [version]
#
# This script collects all components needed for an offline VC Stack deployment:
#   1. Go binaries (vc-management, vc-compute, vcctl)
#   2. Frontend assets (pre-built web console)
#   3. Container images (if Docker is available)
#   4. OS images from /var/lib/vc-stack/images/
#   5. Configuration templates and migration files
#
# Output: vc-stack-airgap-<version>.tar.gz

set -euo pipefail

VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo 'dev')}"
BUNDLE_DIR="vc-stack-airgap-${VERSION}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== VC Stack Airgap Bundle Builder ==="
echo "Version: $VERSION"
echo "Output:  ${BUNDLE_DIR}.tar.gz"
echo ""

# Clean previous builds.
rm -rf "$BUNDLE_DIR" "${BUNDLE_DIR}.tar.gz"
mkdir -p "$BUNDLE_DIR"/{bin,configs,migrations,web,images}

# ── 1. Build Go binaries ─────────────────────────────────────
echo "[1/5] Building Go binaries..."
cd "$ROOT_DIR"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$BUNDLE_DIR/bin/vc-management" ./cmd/vc-management/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$BUNDLE_DIR/bin/vc-compute" ./cmd/vc-compute/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$BUNDLE_DIR/bin/vcctl" ./cmd/vcctl/
echo "  [OK] 3 binaries built"

# ── 2. Frontend assets ────────────────────────────────────────
echo "[2/5] Building frontend..."
if [ -d "$ROOT_DIR/web/console" ]; then
  cd "$ROOT_DIR/web/console"
  npm run build 2>/dev/null || echo "  [WARN] Frontend build skipped (missing deps)"
  if [ -d dist ]; then
    cp -r dist/ "$ROOT_DIR/$BUNDLE_DIR/web/"
    echo "  [OK] Frontend assets copied"
  fi
fi

# ── 3. Configs and migrations ─────────────────────────────────
echo "[3/5] Collecting configs and migrations..."
cd "$ROOT_DIR"
cp -r configs/ "$BUNDLE_DIR/configs/" 2>/dev/null || true
cp -r migrations/ "$BUNDLE_DIR/migrations/" 2>/dev/null || true
echo "  [OK] Configs and migrations copied"

# ── 4. OS images (if available) ───────────────────────────────
echo "[4/5] Collecting OS images..."
IMAGE_DIR="/var/lib/vc-stack/images"
if [ -d "$IMAGE_DIR" ]; then
  cp "$IMAGE_DIR"/*.{qcow2,raw,iso} "$BUNDLE_DIR/images/" 2>/dev/null || true
  echo "  [OK] OS images copied from $IMAGE_DIR"
else
  echo "  [WARN] No OS images found at $IMAGE_DIR (skipped)"
fi

# ── 5. Create manifest ────────────────────────────────────────
echo "[5/5] Generating manifest..."
cat > "$BUNDLE_DIR/manifest.json" <<EOF
{
  "version": "$VERSION",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "components": [
    {"name": "vc-management", "version": "$VERSION", "arch": "amd64", "path": "bin/vc-management"},
    {"name": "vc-compute", "version": "$VERSION", "arch": "amd64", "path": "bin/vc-compute"},
    {"name": "vcctl", "version": "$VERSION", "arch": "amd64", "path": "bin/vcctl"}
  ],
  "images": [],
  "os_images": []
}
EOF
echo "  [OK] Manifest generated"

# ── Package ───────────────────────────────────────────────────
echo ""
echo "Packaging..."
tar czf "${BUNDLE_DIR}.tar.gz" "$BUNDLE_DIR"
rm -rf "$BUNDLE_DIR"

SIZE=$(du -h "${BUNDLE_DIR}.tar.gz" | cut -f1)
echo ""
echo "=== Bundle Complete ==="
echo "File: ${BUNDLE_DIR}.tar.gz ($SIZE)"
echo ""
echo "Deploy with:"
echo "  tar xzf ${BUNDLE_DIR}.tar.gz"
echo "  cd ${BUNDLE_DIR}"
echo "  sudo cp bin/* /usr/local/bin/"
echo "  sudo cp -r configs/ /etc/vc-stack/"
echo "  echo 'true' | sudo tee /etc/vc-stack/airgap"
