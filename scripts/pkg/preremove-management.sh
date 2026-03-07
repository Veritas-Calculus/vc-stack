#!/bin/bash
# Pre-remove script for VC Management.
set -e

# Stop and disable the service.
systemctl stop vc-management.service 2>/dev/null || true
systemctl disable vc-management.service 2>/dev/null || true

echo "VC Stack Management: service stopped and disabled."
