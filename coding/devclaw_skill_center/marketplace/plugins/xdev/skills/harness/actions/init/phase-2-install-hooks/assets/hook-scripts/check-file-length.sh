#!/usr/bin/env bash
# AI 代码守护: 文件行数上限检查
# AI 倾向于生成超长文件，单文件超过上限应拆分
# 上限可通过环境变量 HARNESS_MAX_FILE_LINES 配置
set -euo pipefail

MAX_LINES=${HARNESS_MAX_FILE_LINES:-600}
exit_code=0

for f in "$@"; do
    lines=$(wc -l < "$f")
    if [ "$lines" -gt "$MAX_LINES" ]; then
        echo "ERROR: $f: ${lines} lines (limit ${MAX_LINES}, please split)"
        exit_code=1
    fi
done

exit $exit_code
