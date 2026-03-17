#!/bin/bash
set -e

# Define image name
IMAGE_NAME="vc-stack-builder"

echo "Step 1: Building Docker image (Build & Test inside container)..."
# Build with USE_MIRROR=true for local Chinese network acceleration
docker build --build-arg USE_MIRROR=true -t "$IMAGE_NAME" -f Dockerfile.build .

echo "Step 2: Copying compiled binaries from image to ./bin/..."
mkdir -p bin
docker run --rm -v "$(pwd)/bin":/app/bin-out "$IMAGE_NAME" cp -r /app/bin/. /app/bin-out/

echo "--------------------------------------------------------"
echo "✅ SUCCESS! Binaries are built, tested, and ready in ./bin/"
ls -l bin/
