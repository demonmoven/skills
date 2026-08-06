#!/bin/bash

set -e

echo "========================================="
echo "Starting SQL upgrade process..."
echo "========================================="
echo ""

# Get the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

echo "Project root: $PROJECT_ROOT"
echo ""

# Navigate to backend directory
cd "$PROJECT_ROOT/backend"

echo "Step 1: Running GORM code generation..."
echo "-----------------------------------------"
go run script/gorm_gen/generate.go

if [ $? -ne 0 ]; then
    echo ""
    echo "Error: GORM code generation failed!"
    exit 1
fi

echo ""
echo "========================================="
echo "SQL upgrade process completed successfully!"
echo "========================================="
echo ""
echo "Next steps:"
echo "1. Review the generated code in backend/app/repository/model/"
echo "2. Test the database changes locally"
echo "3. Commit the changes"
echo ""