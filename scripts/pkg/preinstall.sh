#!/bin/bash
# Pre-install script: create vcstack user/group if they don't exist.
set -e

if ! getent group vcstack > /dev/null 2>&1; then
    groupadd --system vcstack
fi

if ! getent passwd vcstack > /dev/null 2>&1; then
    useradd --system --gid vcstack --home-dir /var/lib/vc-stack --shell /usr/sbin/nologin vcstack
fi

# Ensure required directories exist.
mkdir -p /opt/vc-stack/bin
mkdir -p /var/log/vc-stack
mkdir -p /var/lib/vc-stack
mkdir -p /etc/vc-stack

echo "VC Stack: pre-install complete."
