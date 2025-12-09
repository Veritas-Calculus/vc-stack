#!/bin/bash
# Pre-commit helper script
# This script ensures the correct environment is set for pre-commit hooks

# Ensure Go binaries are in PATH
export PATH="$HOME/go/bin:$PATH"

# Disable CGO for golangci-lint to skip CGO-dependent files
export CGO_ENABLED=0

# Run pre-commit
exec pre-commit "$@"
