#!/bin/bash

set -e

# IMPORTANT: bump BEFORE build, not after.
#
# build.js injects package.json's `version` field into dist/index.js as a
# compile-time constant (XDEV_VERSION) via esbuild's `define`. If we bump
# the version AFTER building, the published dist still carries the OLD
# version string while package.json carries the NEW one — so a user installing
# `@byted/xdex@latest` ends up with `xdev --version` showing one patch behind
# the actual npm version. The runtime update-check (src/xdev/version-check.ts)
# would then incorrectly report "update available" on a freshly-installed CLI.
#
# Fix: bump first, then build, so dist's XDEV_VERSION matches the version
# that gets published.

echo "Incrementing version..."
current_version=$(node -p "require('./package.json').version")
IFS='.' read -r -a version_parts <<< "$current_version"

major="${version_parts[0]}"
minor="${version_parts[1]}"
patch="${version_parts[2]}"

new_patch=$((patch + 1))
new_version="$major.$minor.$new_patch"

# Use node to update package.json
node -e "
const fs = require('fs');
const pkg = require('./package.json');
pkg.version = '$new_version';
fs.writeFileSync('./package.json', JSON.stringify(pkg, null, 2) + '\n');
"

echo "Version updated: $current_version -> $new_version"

echo "Running build (with bumped version)..."
npm run build
