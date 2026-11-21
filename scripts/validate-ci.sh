#!/bin/bash
# Copyright 2026 Glassbox Users
# SPDX-License-Identifier: Apache-2.0

# Validate CI/CD configuration
set -euo pipefail

# Ensure we are in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." &>/dev/null && pwd)"
cd "${REPO_ROOT}" || { echo "Failed to change directory to project root: ${REPO_ROOT}"; exit 1; }

echo "Validating CI/CD configuration..."
echo "Project root: ${REPO_ROOT}"

# Check if required files exist
if [ ! -f ".github/workflows/ci-test.yml" ]; then
    echo " CI workflow file missing (.github/workflows/ci-test.yml)"
    exit 1
fi

if [ ! -f ".golangci.yml" ]; then
    echo " golangci-lint config missing"
    exit 1
fi

if [ ! -f "go.mod" ]; then
    echo " go.mod missing"
    exit 1
fi

# Validate go.mod version is represented in CI matrix (best-effort, non-fatal)
GO_VERSION=$(grep "^go " go.mod | awk '{print $2}')
if grep -q "go-version:" .github/workflows/ci-test.yml; then
    if grep -q "go-version: \"${GO_VERSION}\"" .github/workflows/ci-test.yml; then
        echo " Go version ${GO_VERSION} is present in CI matrix"
    else
        echo " Warning: Go version mismatch between go.mod (${GO_VERSION}) and CI matrix"
    fi
else
    echo " Warning: No go-version entries found in .github/workflows/ci-test.yml"
fi

# Check if golangci-lint config is valid YAML
if command -v yamllint &> /dev/null; then
    yamllint .golangci.yml
    echo " golangci-lint config is valid YAML"
else
    echo "  yamllint not available, skipping YAML validation"
fi

echo " CI/CD configuration validation passed"
