#!/usr/bin/env bash

set -e

# 获取项目根目录
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PROJECT_ROOT=$(cd "${SCRIPT_DIR}/../../../.." && pwd)

echo "Project root: ${PROJECT_ROOT}"
cd "${PROJECT_ROOT}"

echo "========================================="
echo "Step 1: Generating code from IDL..."
echo "========================================="
bash "${PROJECT_ROOT}/backend/script/cloudwego/code_gen.sh"

echo ""
echo "========================================="
echo "Step 2: Running go mod tidy..."
echo "========================================="
cd "${PROJECT_ROOT}/backend"
go mod tidy

echo ""
echo "========================================="
echo "IDL upgrade completed successfully!"
echo "========================================="