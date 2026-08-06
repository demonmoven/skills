#!/bin/bash

set -e

echo "========================================="
echo "Starting Wire code generation process..."
echo "========================================="
echo ""

# Get the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

echo "Project root: $PROJECT_ROOT"
echo ""

# Check if wire is installed
if ! command -v wire &> /dev/null; then
    echo "Error: 'wire' command not found!"
    echo ""
    echo "Please install wire first:"
    echo "  go install github.com/google/wire/cmd/wire@latest"
    echo ""
    exit 1
fi

echo "Wire version: $(wire version 2>&1 || echo 'unknown')"
echo ""

# Navigate to backend directory
cd "$PROJECT_ROOT/backend"

echo "Step 1: Finding all wire.go files..."
echo "-----------------------------------------"

# Find all wire.go files
WIRE_FILES=$(find . -name "wire.go" -type f)

if [ -z "$WIRE_FILES" ]; then
    echo "Warning: No wire.go files found!"
    exit 0
fi

echo "Found wire.go files:"
echo "$WIRE_FILES"
echo ""

echo "Step 2: Generating wire_gen.go files..."
echo "-----------------------------------------"

# Generate wire code for each directory containing wire.go
FAILED=0
for wire_file in $WIRE_FILES; do
    # Get the directory containing wire.go
    wire_dir=$(dirname "$wire_file")

    echo "Processing: $wire_dir"

    # Run wire in the directory
    (cd "$wire_dir" && wire) || {
        echo "Error: Wire generation failed in $wire_dir"
        FAILED=1
    }
done

if [ $FAILED -eq 1 ]; then
    echo ""
    echo "Error: Some wire generations failed!"
    echo "Please check the error messages above."
    exit 1
fi

echo ""
echo "========================================="
echo "Wire code generation completed successfully!"
echo "========================================="
echo ""
echo "Generated files:"
find . -name "wire_gen.go" -type f -newer "$0" 2>/dev/null || find . -name "wire_gen.go" -type f
echo ""
echo "Next steps:"
echo "1. Review the generated wire_gen.go files"
echo "2. Test your application to ensure dependency injection works correctly"
echo "3. DO NOT manually edit wire_gen.go files"
echo "4. Commit the changes"
echo ""