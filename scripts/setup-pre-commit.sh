#!/bin/bash

# Setup script for pre-commit hooks
# This script helps new team members quickly set up the development environment

set -e

echo "🚀 Setting up pre-commit hooks for vc-stack..."
echo ""

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if pre-commit is installed
if ! command -v pre-commit &> /dev/null; then
    echo -e "${YELLOW}⚠️  pre-commit is not installed${NC}"
    echo "Installing pre-commit..."

    if command -v brew &> /dev/null; then
        brew install pre-commit
    elif command -v pip3 &> /dev/null; then
        pip3 install pre-commit
    elif command -v pip &> /dev/null; then
        pip install pre-commit
    else
        echo -e "${RED}❌ Cannot install pre-commit. Please install it manually:${NC}"
        echo "   brew install pre-commit"
        echo "   OR"
        echo "   pip install pre-commit"
        exit 1
    fi
fi

echo -e "${GREEN}✅ pre-commit found${NC}"

# Install the hooks
echo ""
echo "Installing pre-commit hooks..."
pre-commit install
pre-commit install --hook-type commit-msg

echo ""
echo -e "${GREEN}✅ Pre-commit hooks installed${NC}"

# Ask if user wants to run on all files
echo ""
read -p "Do you want to run pre-commit on all files now? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Running pre-commit on all files (this may take a while)..."
    pre-commit run --all-files || true
fi

echo ""
echo -e "${GREEN}🎉 Setup complete!${NC}"
echo ""
echo "Pre-commit will now run automatically on git commit."
echo ""
echo "Useful commands:"
echo "  make pre-commit-run       - Run checks on all files"
echo "  make pre-commit-update    - Update hooks to latest versions"
echo "  git commit --no-verify    - Skip pre-commit checks (not recommended)"
echo ""
echo "For more information, see docs/pre-commit.md"
