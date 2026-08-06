#!/bin/bash

set -e

echo "========================================="
echo "Starting error code generation process..."
echo "========================================="
echo ""

# Check if business domain is provided
if [ -z "$1" ]; then
    echo "Error: Business domain name is required!"
    echo "Usage: $0 <business_domain>"
    echo ""
    echo "Available business domains (check backend/script/errorx/metadata.yaml):"
    echo "  - evaluation"
    echo "  - prompt"
    echo "  - foundation"
    echo "  - llm"
    echo "  - data"
    echo "  - observability"
    echo ""
    exit 1
fi

BIZ_DOMAIN=$1

# Get the project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

echo "Project root: $PROJECT_ROOT"
echo "Business domain: $BIZ_DOMAIN"
echo ""

# Navigate to errorx directory
cd "$PROJECT_ROOT/backend/script/errorx"

# Check if Python 3 is available
if ! command -v python3 &> /dev/null; then
    echo "Error: python3 is required but not installed!"
    exit 1
fi

# Make sure code_gen.py is executable
chmod +x code_gen.py

echo "Step 1: Generating error code for '$BIZ_DOMAIN'..."
echo "-----------------------------------------"
./code_gen.py "$BIZ_DOMAIN"

if [ $? -ne 0 ]; then
    echo ""
    echo "Error: Error code generation failed!"
    exit 1
fi

echo ""
echo "========================================="
echo "Error code generation completed successfully!"
echo "========================================="
echo ""
echo "Generated code location: backend/module/$BIZ_DOMAIN/pkg/errno/"
echo ""
echo "Next steps:"
echo "1. Review the generated error code in backend/module/$BIZ_DOMAIN/pkg/errno/"
echo "2. Configure i18n messages in:"
echo "   - release/deployment/docker-compose/conf/locales/"
echo "   - release/deployment/helm-chart/umbrella/conf/locales/"
echo "3. Use the error code in your code: errorx.NewByCode(errno.YourErrorCode)"
echo "4. Commit the changes"
echo ""