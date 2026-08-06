#!/usr/bin/env bash
# secrets-scan.sh — 扫描变更文件中的硬编码密钥/Token
#
# 用法: secrets-scan.sh <target_path> <base_sha> [--json]
# 输出: 匹配结果列表，--json 时输出 JSON 格式

set -euo pipefail

TARGET_PATH="${1:?用法: secrets-scan.sh <target_path> <base_sha> [--json]}"
BASE_SHA="${2:-HEAD~1}"
JSON_OUTPUT=false
[[ "${3:-}" == "--json" ]] && JSON_OUTPUT=true

cd "$TARGET_PATH"

# 获取变更文件列表
CHANGED_FILES=$(git diff --name-only "$BASE_SHA"..HEAD 2>/dev/null || git diff --name-only)

# 排除测试文件
CHANGED_FILES=$(echo "$CHANGED_FILES" | grep -v '_test\.go$' || true)

if [[ -z "$CHANGED_FILES" ]]; then
    if $JSON_OUTPUT; then
        echo '{"findings": [], "count": 0}'
    else
        echo "无变更文件需要扫描"
    fi
    exit 0
fi

# 扫描模式（不区分大小写）
PATTERNS=(
    '(password|passwd)\s*[:=]\s*["\x27][^"\x27]{8,}'
    'secret\s*[:=]\s*["\x27][^"\x27]{8,}'
    'token\s*[:=]\s*["\x27][^"\x27]{8,}'
    'api[_-]?key\s*[:=]\s*["\x27][^"\x27]{8,}'
    'access[_-]?key\s*[:=]\s*["\x27][^"\x27]{8,}'
    'private[_-]?key\s*[:=]\s*["\x27][^"\x27]{8,}'
)

# 占位符白名单（不区分大小写）
PLACEHOLDERS='(your-.*-here|placeholder|xxx+|changeme|example|test|TODO|FIXME|dummy|fake|sample)'

FINDINGS=()
FINDING_COUNT=0

while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    [[ ! -f "$file" ]] && continue

    for pattern in "${PATTERNS[@]}"; do
        # 匹配模式，排除注释行
        matches=$(grep -inE "$pattern" "$file" 2>/dev/null | grep -v '^\s*//' | grep -v '^\s*\*' | grep -v '^\s*#' || true)

        while IFS= read -r match; do
            [[ -z "$match" ]] && continue

            # 检查是否为占位符
            if echo "$match" | grep -iqE "$PLACEHOLDERS"; then
                continue
            fi

            line_num=$(echo "$match" | cut -d: -f1)
            line_content=$(echo "$match" | cut -d: -f2-)

            FINDING_COUNT=$((FINDING_COUNT + 1))

            if $JSON_OUTPUT; then
                FINDINGS+=("{\"id\": \"SEC-$(printf '%03d' $FINDING_COUNT)\", \"file\": \"$file\", \"line\": $line_num, \"severity\": \"critical\", \"match\": $(echo "$line_content" | head -c 200 | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read().strip()))' 2>/dev/null || echo '\"<encoding error>\"')}")
            else
                echo "[$file:$line_num] $line_content"
            fi
        done <<< "$matches"
    done
done <<< "$CHANGED_FILES"

if $JSON_OUTPUT; then
    FINDINGS_JSON=$(IFS=,; echo "${FINDINGS[*]:-}")
    echo "{\"findings\": [${FINDINGS_JSON}], \"count\": $FINDING_COUNT}"
else
    if [[ $FINDING_COUNT -eq 0 ]]; then
        echo "未发现硬编码密钥"
    else
        echo ""
        echo "共发现 $FINDING_COUNT 个疑似硬编码密钥"
    fi
fi

exit 0
