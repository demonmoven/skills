#!/usr/bin/env bash
# ExecPlan 完成度守护（pre-push 阶段）
# 扫描 docs/plans/active/ 下的 markdown 文件
# 发现未完成项（[ ]）则阻止推送
# 全部完成则自动归档到 docs/plans/completed/
set -euo pipefail

PLANS_DIR="docs/plans/active"
COMPLETED_DIR="docs/plans/completed"

if [[ ! -d "$PLANS_DIR" ]]; then
  exit 0
fi

if ! ls "$PLANS_DIR"/*.md >/dev/null 2>&1; then
  exit 0
fi

has_unfinished=0
has_archived=0

mkdir -p "$COMPLETED_DIR"

for plan in "$PLANS_DIR"/*.md; do
  if grep -nE '^[[:space:]]*-[[:space:]]*\[[[:space:]]\][[:space:]]' "$plan" >/dev/null; then
    has_unfinished=1
    echo "ERROR: unfinished ExecPlan: $plan"
    echo "   Incomplete items:"
    grep -nE '^[[:space:]]*-[[:space:]]*\[[[:space:]]\][[:space:]]' "$plan" | sed 's/^/   - /'
    continue
  fi

  base_name="$(basename "$plan")"
  target_path="$COMPLETED_DIR/$base_name"
  if [[ -e "$target_path" ]]; then
    ts="$(date +%Y%m%d-%H%M%S)"
    target_path="$COMPLETED_DIR/${base_name%.md}-$ts.md"
  fi

  mv "$plan" "$target_path"
  has_archived=1
  echo "OK: archived completed plan: $plan -> $target_path"
done

if [[ "$has_unfinished" -eq 1 ]]; then
  echo ""
  echo "Push blocked: docs/plans/active has unfinished work."
  echo "Complete all items (change [ ] to [x]) or move to docs/plans/completed."
  exit 1
fi

if [[ "$has_archived" -eq 1 ]]; then
  echo ""
  echo "Completed plans were auto-archived. Please git add the changes and retry."
  exit 1
fi

exit 0
