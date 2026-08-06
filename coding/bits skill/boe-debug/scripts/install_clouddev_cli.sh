#!/bin/bash
set -euo pipefail

INSTALLER_URL="https://lf0-bytesuite-cli.bytedance.net/obj/bytesuite-center/clouddev-cli-install.sh"
INSTALLER_DIR="${TMPDIR:-/tmp}/clouddev-cli-installer"
INSTALLER_PATH="${INSTALLER_DIR}/clouddev-cli-install.sh"

mkdir -p "$INSTALLER_DIR"
curl -fsSL "$INSTALLER_URL" -o "$INSTALLER_PATH"

bash "$INSTALLER_PATH"
