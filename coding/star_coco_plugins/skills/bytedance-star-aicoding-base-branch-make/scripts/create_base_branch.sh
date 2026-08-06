#!/bin/bash
set -euo pipefail

# ============================================================
# 单条 MR 基座分支创建脚本
# 用法: ./create_base_branch.sh <REPO> <MR_ID> <VERSION> [CHERRY_PICK_CONFIG]
#
# 参数:
#   REPO                仓库地址
#   MR_ID               MR 编号
#   VERSION             版本号
#   CHERRY_PICK_CONFIG  可选，cherry-pick 配置文件路径（JSON）
#                       不传则使用脚本同目录下的 cherry_pick_config.json
#
# 退出码:
#   0 = 成功
#   1 = 参数错误
#   2 = MR 引用未找到
#   3 = merge commit 未找到
#   4 = 推送失败
#   5 = cherry-pick 失败（accept theirs 后仍失败）
# ============================================================

if [ $# -lt 3 ]; then
    echo "用法: $0 <REPO> <MR_ID> <VERSION> [CHERRY_PICK_CONFIG]"
    echo "示例: $0 https://code.byted.org/ad/star_settlement 844 v1"
    echo "      $0 https://code.byted.org/ad/star_settlement 844 v1 /path/to/cherry_pick_config.json"
    exit 1
fi

# ============ 输入参数 ============
REPO=$1
MR_ID=$2
VERSION=$3

# cherry-pick 配置文件：优先用传入参数，否则用脚本同目录下的默认配置
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
CHERRY_PICK_CONFIG=${4:-"${SCRIPT_DIR}/cherry_pick_config.json"}

# ============ 自动推导参数 ============
REPO_NAME=$(basename ${REPO})
BASE_BRANCH_NAME="ai_coding_test/base/${VERSION}/${MR_ID}"
WORK_DIR="${REPO_NAME}_mr${MR_ID}_$$"

echo "📌 [${REPO_NAME}] MR-${MR_ID} → ${BASE_BRANCH_NAME}"

# ============ 清理函数 ============
cleanup() {
    local exit_code=$?
    cd "${START_DIR}" 2>/dev/null || true
    if [ -d "${WORK_DIR}" ]; then
        rm -rf "${WORK_DIR}"
    fi
    exit $exit_code
}
START_DIR=$(pwd)
trap cleanup EXIT

# ============ Step 0: 准备环境 ============
mkdir -p ${WORK_DIR}
git clone --no-checkout --filter=blob:none ${REPO} ${WORK_DIR}/${REPO_NAME} 2>&1
cd ${WORK_DIR}/${REPO_NAME}

echo "   ✅ Step 0: Clone 完成"

# ============ Step 1: 自动检测默认分支 ============
DEFAULT_BRANCH=$(git ls-remote --symref origin HEAD 2>/dev/null \
    | sed -n 's#^ref: refs/heads/\([^ ]*\)[[:space:]]\+HEAD$#\1#p' \
    | head -n 1)
test -n "$DEFAULT_BRANCH" || DEFAULT_BRANCH=master

echo "   ✅ Step 1: 默认分支: ${DEFAULT_BRANCH}"

# ============ Step 2: Fetch 默认分支 ============
git fetch --no-tags --prune --quiet origin \
    refs/heads/${DEFAULT_BRANCH}:refs/remotes/origin/${DEFAULT_BRANCH}

echo "   ✅ Step 2: Fetch 默认分支完成"

# ============ Step 3: 自动发现 MR 引用并 Fetch ============
MR_REF=$(git ls-remote origin "refs/merge-requests/*/${MR_ID}/*" \
    | sort -t'/' -k5 -n \
    | tail -1 \
    | awk '{print $2}')

if [ -z "$MR_REF" ]; then
    echo "   ❌ 未找到 MR-${MR_ID} 的远端引用"
    exit 2
fi

git fetch --no-tags --quiet origin ${MR_REF}:refs/tmp/mr_tip

echo "   ✅ Step 3: MR 引用: ${MR_REF}"

# ============ Step 4: 计算分叉点 ============
MR_TIP=$(git rev-parse refs/tmp/mr_tip)
BASE_SHA=$(git merge-base ${MR_TIP} origin/${DEFAULT_BRANCH})

echo "   ✅ Step 4: 分叉点: ${BASE_SHA:0:8}"

# ============ Step 5: 找目标 merge commit ============
TARGET_SHA=$(git log --first-parent --format=%H -n 1 \
    --fixed-strings --grep="into '${DEFAULT_BRANCH}'" ${BASE_SHA})

if [ -z "$TARGET_SHA" ]; then
    echo "   ❌ 未找到 merge into '${DEFAULT_BRANCH}' 的 commit"
    exit 3
fi

echo "   ✅ Step 5: 目标 commit: ${TARGET_SHA:0:8}"

# ============ Step 6: 创建基座分支 ============
git checkout --detach ${TARGET_SHA} 2>/dev/null
git checkout -b ${BASE_BRANCH_NAME} 2>/dev/null

echo "   ✅ Step 6: 分支已创建: ${BASE_BRANCH_NAME}"

# ============ Step 6.5: Cherry-Pick（按配置文件） ============
# 从 JSON 配置中读取当前仓库的 cherry-pick 列表
if [ -f "${CHERRY_PICK_CONFIG}" ]; then
    # 用 python3 解析 JSON，提取当前 REPO_NAME 对应的 commit 列表
    CHERRY_COMMITS=$(python3 -c "
import json, sys
with open('${CHERRY_PICK_CONFIG}', 'r') as f:
    config = json.load(f)
commits = config.get('${REPO_NAME}', [])
for c in commits:
    print(c)
" 2>/dev/null || true)
else
    CHERRY_COMMITS=""
fi

if [ -n "$CHERRY_COMMITS" ]; then
    echo "   🍒 检测到 ${REPO_NAME} 的 cherry-pick 配置，开始处理..."

    # cherry-pick 需要完整对象
    git fetch --quiet origin ${DEFAULT_BRANCH} 2>&1

    PICKED=0
    SKIPPED=0
    CONFLICT_RESOLVED=0

    while IFS= read -r COMMIT; do
        # 跳过空行
        [ -z "$COMMIT" ] && continue

        SHORT_SHA=${COMMIT:0:8}

        if git merge-base --is-ancestor ${COMMIT} HEAD 2>/dev/null; then
            echo "      ⏭️  ${SHORT_SHA} 已存在于当前分支，跳过"
            SKIPPED=$((SKIPPED + 1))
        else
            if git cherry-pick ${COMMIT} --no-edit 2>/dev/null; then
                echo "      🍒 ${SHORT_SHA} cherry-pick 成功"
                PICKED=$((PICKED + 1))
            else
                echo "      ⚠️  ${SHORT_SHA} 冲突，执行 accept theirs..."

                CONFLICT_FILES=$(git diff --name-only --diff-filter=U)
                if [ -n "$CONFLICT_FILES" ]; then
                    echo "$CONFLICT_FILES" | while read -r file; do
                        git checkout --theirs -- "$file" 2>/dev/null
                        git add "$file" 2>/dev/null
                    done
                fi

                DELETED_FILES=$(git diff --name-only --diff-filter=DU 2>/dev/null || true)
                if [ -n "$DELETED_FILES" ]; then
                    echo "$DELETED_FILES" | while read -r file; do
                        git rm --quiet "$file" 2>/dev/null || true
                    done
                fi

                if git cherry-pick --continue --no-edit 2>/dev/null; then
                    echo "      🔧 ${SHORT_SHA} 冲突已解决 (accept theirs)"
                    PICKED=$((PICKED + 1))
                    CONFLICT_RESOLVED=$((CONFLICT_RESOLVED + 1))
                else
                    echo "      ❌ ${SHORT_SHA} accept theirs 后仍然失败，中止"
                    git cherry-pick --abort 2>/dev/null || true
                    exit 5
                fi
            fi
        fi
    done <<< "$CHERRY_COMMITS"

    echo "   ✅ Step 6.5: Cherry-pick 完成 — ${PICKED} 个新增 (${CONFLICT_RESOLVED} 个冲突已解决), ${SKIPPED} 个跳过"
else
    echo "   ⏭️  Step 6.5: ${REPO_NAME} 无 cherry-pick 配置，跳过"
fi

# ============ Step 7: 推送到远端 ============
if ! git push origin ${BASE_BRANCH_NAME} 2>&1; then
    echo "   ❌ 推送失败"
    exit 4
fi

echo "   ✅ Step 7: 已推送到远端"
echo "   🎉 完成！${BASE_BRANCH_NAME} (基于 ${TARGET_SHA:0:8})"
