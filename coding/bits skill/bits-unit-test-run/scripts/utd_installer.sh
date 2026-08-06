#!/usr/bin/env bash

set -ex

# ==============================================================================
# Module: Define paths and variables
# ==============================================================================
INSTALL_DIR="$HOME/.bits-ut"
UTD_BIN="$INSTALL_DIR/utd"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Check if binary already exists and is executable
if [ -f "$UTD_BIN" ] && [ -x "$UTD_BIN" ]; then
    exit 0
fi

# ==============================================================================
# Module: Get system info
# ==============================================================================
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH="arm64"
fi

TARGET_BIN="utd_${OS}_${ARCH}"

# ==============================================================================
# Module: Fetch download URL
# ==============================================================================
URL="https://scm.byted.org/api/v2/versions/latest_version/?repo_id=404750&status=build_ok&type=online"
JSON_RESP=$(curl -s "$URL")

if [ -z "$JSON_RESP" ]; then
    echo "Error: Failed to fetch version info from SCM API"
    exit 1
fi

# Extract tar_url from JSON
TAR_URL=$(echo "$JSON_RESP" | python3 -c '
import sys, json
try:
    data = json.load(sys.stdin)
    if "tar_url" in data:
        print(data["tar_url"])
    elif "data" in data and "tar_url" in data["data"]:
        print(data["data"]["tar_url"])
    else:
        print("")
except Exception:
    print("")
')

if [ -z "$TAR_URL" ]; then
    # Fallback: grep/sed
    TAR_URL=$(echo "$JSON_RESP" | grep -o '"tar_url"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"tar_url"[[:space:]]*:[[:space:]]*"//;s/"//')
fi

if [ -z "$TAR_URL" ]; then
    echo "Error: Failed to parse tar_url from API response"
    exit 1
fi

# ==============================================================================
# Module: Download and install
# ==============================================================================
TEMP_DIR=$(mktemp -d)

# Register trap to clean up temp directory
trap 'rm -rf "$TEMP_DIR"' EXIT

# Download
curl -sL -o "$TEMP_DIR/utd.tar.gz" "$TAR_URL"

# Extract
tar -xzf "$TEMP_DIR/utd.tar.gz" -C "$TEMP_DIR"

# Find the target binary
EXTRACTED_BIN=$(find "$TEMP_DIR" -type f -name "$TARGET_BIN" | head -n 1)

if [ -n "$EXTRACTED_BIN" ]; then
    mv "$EXTRACTED_BIN" "$UTD_BIN"
    chmod +x "$UTD_BIN"
else
    echo "Error: Target binary $TARGET_BIN not found in the downloaded archive"
    ls -la "$TEMP_DIR"
    exit 1
fi

# Clean up before normal exit
trap - EXIT
rm -rf "$TEMP_DIR"

exit 0