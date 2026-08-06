#!/usr/bin/env bash
# Go 依赖一致性检查（单 module 版本）
# 跨平台兼容：macOS 用 md5 -q，Linux 用 md5sum
set -euo pipefail

md5_file() {
    if command -v md5 &>/dev/null; then
        md5 -q "$1"
    elif command -v md5sum &>/dev/null; then
        md5sum "$1" | cut -d' ' -f1
    else
        stat -c '%s%Y' "$1" 2>/dev/null || stat -f '%z%m' "$1" 2>/dev/null || echo "unknown"
    fi
}

has_error=0

mod_hash=$(md5_file go.mod)
sum_hash=""
if [ -f go.sum ]; then
    sum_hash=$(md5_file go.sum)
fi

go mod tidy

new_mod_hash=$(md5_file go.mod)
new_sum_hash=""
if [ -f go.sum ]; then
    new_sum_hash=$(md5_file go.sum)
fi

if [ "$mod_hash" != "$new_mod_hash" ] || [ "$sum_hash" != "$new_sum_hash" ]; then
    echo "ERROR: go.mod/go.sum changed after go mod tidy. Please re-stage and commit."
    has_error=1
fi

exit $has_error
