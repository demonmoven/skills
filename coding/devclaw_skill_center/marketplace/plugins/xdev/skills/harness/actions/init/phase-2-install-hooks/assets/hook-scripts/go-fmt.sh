#!/usr/bin/env bash
# Go 格式化自动修复
# gofmt -s（简化）+ interface{} -> any 替换
set -euo pipefail

gofmt -s -w .
gofmt -r 'interface{} -> any' -w .
