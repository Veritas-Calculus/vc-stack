#!/bin/bash
# Pre-remove script for VC Compute.
set -e

# Stop and disable the service.
systemctl stop vc-compute.service 2>/dev/null || true
systemctl disable vc-compute.service 2>/dev/null || true

echo "VC Stack Compute: service stopped and disabled."
