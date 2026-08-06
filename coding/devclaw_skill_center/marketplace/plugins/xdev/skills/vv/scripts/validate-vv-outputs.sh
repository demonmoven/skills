#!/usr/bin/env bash
# validate-vv-outputs.sh — 校验 V&V 输出目录的完整性
#
# 用法: validate-vv-outputs.sh <vv_output_dir>
# 退出码: 0 = 全部有效, 1 = 存在问题

set -euo pipefail

VV_DIR="${1:?用法: validate-vv-outputs.sh <vv_output_dir>}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
HAS_ERROR=0

echo "=== V&V 输出校验 ==="
echo "输出目录: $VV_DIR"
echo ""

# 校验 failure-taxonomy.json
TAXONOMY="$VV_DIR/failure-taxonomy.json"
if [[ ! -f "$TAXONOMY" ]]; then
    echo "ERROR: 缺少 failure-taxonomy.json"
    HAS_ERROR=1
else
    if ! python3 -c "import json; json.load(open('$TAXONOMY'))" 2>/dev/null; then
        echo "ERROR: failure-taxonomy.json 不是有效的 JSON"
        HAS_ERROR=1
    else
        # 检查 6 个 failure type 键是否存在
        MISSING=$(python3 -c "
import json, sys
data = json.load(open('$TAXONOMY'))
required = ['POLICY_VIOLATION','STRUCTURE_DRIFT','OUTCOME_FAIL','TOOL_MISUSE','THRASHING','UNNECESSARY_COMPLEXITY']
missing = [k for k in required if k not in data]
if missing:
    print('缺少 failure type: ' + ', '.join(missing))
    sys.exit(1)
if 'details' not in data:
    print('缺少 details 字段')
    sys.exit(1)
print('OK')
" 2>&1) || true
        if [[ "$MISSING" != "OK" ]]; then
            echo "ERROR: failure-taxonomy.json - $MISSING"
            HAS_ERROR=1
        else
            echo "OK: failure-taxonomy.json 结构有效"
        fi
    fi
fi

# 校验 verdict.json
VERDICT="$VV_DIR/verdict.json"
if [[ ! -f "$VERDICT" ]]; then
    echo "ERROR: 缺少 verdict.json"
    HAS_ERROR=1
else
    if ! python3 -c "import json; json.load(open('$VERDICT'))" 2>/dev/null; then
        echo "ERROR: verdict.json 不是有效的 JSON"
        HAS_ERROR=1
    else
        MISSING=$(python3 -c "
import json, sys
data = json.load(open('$VERDICT'))
required = ['verdict','reason','gate_level','subsystem_scores','failure_taxonomy','blocking_findings']
missing = [k for k in required if k not in data]
if missing:
    print('缺少字段: ' + ', '.join(missing))
    sys.exit(1)
valid_verdicts = ['PASS','FAIL','SOFT_FAIL','NEEDS_APPROVAL']
if data.get('verdict') not in valid_verdicts:
    print(f\"verdict 值无效: {data.get('verdict')}，允许值: {valid_verdicts}\")
    sys.exit(1)
print('OK')
" 2>&1) || true
        if [[ "$MISSING" != "OK" ]]; then
            echo "ERROR: verdict.json - $MISSING"
            HAS_ERROR=1
        else
            echo "OK: verdict.json 结构有效"
        fi
    fi
fi

# 校验 final-report.md
REPORT="$VV_DIR/final-report.md"
if [[ ! -f "$REPORT" ]]; then
    echo "ERROR: 缺少 final-report.md"
    HAS_ERROR=1
else
    # 检查报告是否包含关键章节
    MISSING_SECTIONS=""
    for section in "基本信息" "子系统评分" "裁决说明"; do
        if ! grep -q "$section" "$REPORT" 2>/dev/null; then
            MISSING_SECTIONS="$MISSING_SECTIONS $section"
        fi
    done
    if [[ -n "$MISSING_SECTIONS" ]]; then
        echo "WARNING: final-report.md 缺少章节:$MISSING_SECTIONS"
    else
        echo "OK: final-report.md 包含关键章节"
    fi
fi

# 校验各子系统的 summary.json（如果存在）
for subsystem_dir in "$VV_DIR"/subsystem_*; do
    if [[ ! -d "$subsystem_dir" ]]; then
        continue
    fi
    SUBSYSTEM_NAME=$(basename "$subsystem_dir")
    SUMMARY="$subsystem_dir/summary.json"
    if [[ ! -f "$SUMMARY" ]]; then
        echo "WARNING: $SUBSYSTEM_NAME/summary.json 不存在"
        continue
    fi
    if bash "$SCRIPT_DIR/validate-summary-json.sh" "$SUMMARY" > /dev/null 2>&1; then
        echo "OK: $SUBSYSTEM_NAME/summary.json 校验通过"
    else
        echo "ERROR: $SUBSYSTEM_NAME/summary.json 校验失败"
        bash "$SCRIPT_DIR/validate-summary-json.sh" "$SUMMARY" 2>&1 || true
        HAS_ERROR=1
    fi
done

echo ""
if [[ $HAS_ERROR -eq 1 ]]; then
    echo "=== 校验结果: 存在错误，请修复后重试 ==="
    exit 1
else
    echo "=== 校验结果: 全部通过 ==="
    exit 0
fi
