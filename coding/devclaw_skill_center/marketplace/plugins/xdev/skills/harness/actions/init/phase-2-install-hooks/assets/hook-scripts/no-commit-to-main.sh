#!/usr/bin/env bash
# 禁止直接在 main/master 分支上 commit
# 允许通过环境变量 ALLOW_MAIN_COMMIT=1 绕过
set -euo pipefail

if [ "${ALLOW_MAIN_COMMIT:-0}" = "1" ]; then
  exit 0
fi

branch=$(git symbolic-ref --short HEAD 2>/dev/null || echo "")

if [ "$branch" = "main" ] || [ "$branch" = "master" ]; then
  echo ""
  echo "ERROR: 禁止在 $branch 分支直接 commit！"
  echo ""
  echo "   请切换到功能分支后再提交："
  echo "   git checkout -b feat/your-feature-name"
  echo ""
  echo "   如有特殊需要，可临时绕过："
  echo "   ALLOW_MAIN_COMMIT=1 git commit ..."
  echo ""
  exit 1
fi

exit 0
