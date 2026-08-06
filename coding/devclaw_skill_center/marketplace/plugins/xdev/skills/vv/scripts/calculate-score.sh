#!/usr/bin/env bash
# calculate-score.sh — 统一评分计算
#
# 用法: calculate-score.sh --mode <llm|gate> --critical <n> --high <n> --medium <n> --low <n>
# 输出: score=XX pass=true/false

set -euo pipefail

MODE="llm"
CRITICAL=0
HIGH=0
MEDIUM=0
LOW=0
GATE_LEVEL="pr"

while [[ $# -gt 0 ]]; do
    case $1 in
        --mode) MODE="$2"; shift 2 ;;
        --critical) CRITICAL="$2"; shift 2 ;;
        --high) HIGH="$2"; shift 2 ;;
        --medium) MEDIUM="$2"; shift 2 ;;
        --low) LOW="$2"; shift 2 ;;
        --gate-level) GATE_LEVEL="$2"; shift 2 ;;
        *) echo "未知参数: $1"; exit 1 ;;
    esac
done

# 计算分数
if [[ "$MODE" == "llm" ]]; then
    # LLM 审查模式: critical=-25, high=-10, medium=-3, low=0
    SCORE=$((100 - CRITICAL * 25 - HIGH * 10 - MEDIUM * 3))
    # Pass 条件: 无严重/中等/轻微问题
    if [[ $CRITICAL -eq 0 && $HIGH -eq 0 && $MEDIUM -eq 0 ]]; then
        PASS=true
    else
        PASS=false
    fi
elif [[ "$MODE" == "gate" ]]; then
    # 门禁模式: critical=-25, high=-10, medium=-5, low=0
    SCORE=$((100 - CRITICAL * 25 - HIGH * 10 - MEDIUM * 5))
    # Pass 条件取决于 gate level
    case "$GATE_LEVEL" in
        pr)
            [[ $SCORE -ge 70 && $CRITICAL -eq 0 ]] && PASS=true || PASS=false
            ;;
        nightly)
            [[ $SCORE -ge 80 && $CRITICAL -eq 0 && $HIGH -eq 0 ]] && PASS=true || PASS=false
            ;;
        release)
            [[ $SCORE -ge 90 && $CRITICAL -eq 0 && $HIGH -eq 0 && $MEDIUM -eq 0 ]] && PASS=true || PASS=false
            ;;
        *)
            echo "未知的 gate level: $GATE_LEVEL"
            exit 1
            ;;
    esac
else
    echo "未知的 mode: $MODE (允许值: llm, gate)"
    exit 1
fi

# 确保最低 0 分
[[ $SCORE -lt 0 ]] && SCORE=0

echo "score=$SCORE"
echo "pass=$PASS"
