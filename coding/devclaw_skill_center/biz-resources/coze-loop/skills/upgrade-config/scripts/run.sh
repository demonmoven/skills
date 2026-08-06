#!/bin/bash

set -e

echo "========================================="
echo "Starting config validation process..."
echo "========================================="
echo ""

# Get the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

echo "Project root: $PROJECT_ROOT"
echo ""

# Check if yq is installed
if ! command -v yq &> /dev/null; then
    echo "Warning: 'yq' is not installed. Skipping YAML validation."
    echo "To install yq:"
    echo "  macOS: brew install yq"
    echo "  Linux: sudo snap install yq"
    echo ""
    exit 0
fi

echo "Step 1: Validating Docker deployment configs..."
echo "-----------------------------------------"
DOCKER_CONF_DIR="$PROJECT_ROOT/release/deployment/docker-compose/conf"

if [ -d "$DOCKER_CONF_DIR" ]; then
    for file in "$DOCKER_CONF_DIR"/*.yaml; do
        if [ -f "$file" ]; then
            filename=$(basename "$file")
            echo "Validating: $filename"
            yq eval '.' "$file" > /dev/null 2>&1
            if [ $? -ne 0 ]; then
                echo "Error: Invalid YAML in Docker config: $filename"
                exit 1
            fi
        fi
    done
    echo "Docker configs validated successfully!"
else
    echo "Warning: Docker config directory not found"
fi

echo ""
echo "Step 2: Validating Kubernetes deployment configs..."
echo "-----------------------------------------"
K8S_CONF_DIR="$PROJECT_ROOT/release/deployment/helm-chart/umbrella/conf"

if [ -d "$K8S_CONF_DIR" ]; then
    for file in "$K8S_CONF_DIR"/*.yaml; do
        if [ -f "$file" ]; then
            filename=$(basename "$file")
            echo "Validating: $filename"
            yq eval '.' "$file" > /dev/null 2>&1
            if [ $? -ne 0 ]; then
                echo "Error: Invalid YAML in Kubernetes config: $filename"
                exit 1
            fi
        fi
    done
    echo "Kubernetes configs validated successfully!"
else
    echo "Warning: Kubernetes config directory not found"
fi

echo ""
echo "========================================="
echo "Config validation completed successfully!"
echo "========================================="
echo ""
echo "Next steps:"
echo "1. Review the config changes"
echo "2. Test the configuration locally"
echo "3. Commit the changes"
echo ""
