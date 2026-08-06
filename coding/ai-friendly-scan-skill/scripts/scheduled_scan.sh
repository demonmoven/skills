#!/bin/bash
# ── AI 友好度定期巡检脚本 ──────────────────────────────
# 每周执行一次，扫描所有仓库并更新飞书多维表格 + 发送群通知
#
# 用法:
#   bash scripts/scheduled_scan.sh
#
# 首次使用前，请修改下方配置项。
# 建议通过 crontab 定期执行:
#   crontab -e
#   0 10 * * 1 cd /Users/bytedance/Documents/go/sawyer-workspace && bash scripts/scheduled_scan.sh >> log/scan.log 2>&1
# ────────────────────────────────────────────────────────

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# ── 配置项（按实际情况修改）──────────────────────────────
REPO_LIST="$SCRIPT_DIR/repos.txt"
HISTORY_DIR="$SCRIPT_DIR/scan_history"
BITABLE_APP="LUl0bFMooanjGoszjhkcyhQyn8d"
SNAPSHOT_TABLE="tbl73jFL8zFz137d"
HISTORY_TABLE="tblDbwTLLhm0PUnf"
BITABLE_URL="https://bytedance.larkoffice.com/base/LUl0bFMooanjGoszjhkcyhQyn8d"
# 飞书群机器人 webhook（创建方式：群设置 → 群机器人 → 添加机器人 → 自定义机器人）
WEBHOOK=""  # 填入你的 webhook URL，例如 https://open.feishu.cn/open-apis/bot/v2/hook/xxx
CONCURRENCY=5
# ────────────────────────────────────────────────────────

SCAN_ID=$(date +%Y-%m-%d)
CSV_FILE="$HISTORY_DIR/scan_${SCAN_ID}.csv"

mkdir -p "$HISTORY_DIR"

echo "============================================"
echo "AI 友好度巡检 - $SCAN_ID"
echo "============================================"

# 找到上次扫描的 CSV（用于对比）
PREV_CSV=$(ls -t "$HISTORY_DIR"/scan_*.csv 2>/dev/null | head -1 || true)
PREV_FLAG=""
if [[ -n "$PREV_CSV" && -f "$PREV_CSV" ]]; then
    echo "上次扫描: $PREV_CSV"
    PREV_FLAG="--prev-csv $PREV_CSV"
fi

# 构建命令
CMD="python3 $SCRIPT_DIR/scan_repos.py \
  --input $REPO_LIST \
  --output $CSV_FILE \
  --bitable-app $BITABLE_APP \
  --bitable-table $SNAPSHOT_TABLE \
  --clear-snapshot \
  --history-table $HISTORY_TABLE \
  --scan-id $SCAN_ID \
  --bitable-url $BITABLE_URL \
  --concurrency $CONCURRENCY \
  $PREV_FLAG"

# 如果配置了 webhook，加上通知参数
if [[ -n "$WEBHOOK" ]]; then
    CMD="$CMD --webhook $WEBHOOK"
fi

# 执行扫描
eval $CMD

echo ""
echo "扫描完成，CSV 已归档: $CSV_FILE"
# HTML 报告由 scan_repos.py 在扫描后自动生成
echo "HTML 报告: $HISTORY_DIR/report_${SCAN_ID}.html"
echo "============================================"
