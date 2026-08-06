#!/bin/bash
set -uo pipefail

# ============================================================
# 批量 MR 基座分支创建脚本
# 读取 CSV，逐行调用 create_base_branch.sh
#
# 用法: ./batch_create.sh <CSV_FILE> <VERSION> [CHERRY_PICK_CONFIG]
#   CSV_FILE            CSV 文件路径（必须包含 mr_iid 和 repo_web_url 列）
#   VERSION             版本号，如 v1
#   CHERRY_PICK_CONFIG  可选，cherry-pick 配置文件路径
#
# 示例:
#   ./batch_create.sh input.csv v1
#   ./batch_create.sh input.csv v1 /path/to/cherry_pick_config.json
# ============================================================

if [ $# -lt 2 ]; then
    echo "用法: $0 <CSV_FILE> <VERSION> [CHERRY_PICK_CONFIG]"
    echo "示例: $0 input.csv v1"
    exit 1
fi

CSV_FILE=$1
VERSION=$2
CHERRY_PICK_CONFIG=${3:-""}

# ---- 检查 create_base_branch.sh ----
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
CREATE_SCRIPT="${SCRIPT_DIR}/create_base_branch.sh"

if [ ! -f "${CREATE_SCRIPT}" ]; then
    echo "❌ 未找到 create_base_branch.sh，请确保它与本脚本在同一目录下"
    exit 1
fi

chmod +x "${CREATE_SCRIPT}"

# ---- 输出目录 ----
WORK_DIR="batch_result_$(date +%Y%m%d_%H%M%S)"
LOG_DIR="${WORK_DIR}/logs"
mkdir -p ${LOG_DIR}

RESULT_FILE="${WORK_DIR}/result.csv"
echo "mr_iid,repo_web_url,status,branch_name,message" > ${RESULT_FILE}

# ---- 解析 CSV ----
echo "📂 解析 CSV: ${CSV_FILE}"

TASK_FILE="${WORK_DIR}/tasks.txt"
python3 - "${CSV_FILE}" "${TASK_FILE}" << 'PYEOF'
import csv, sys

csv_file = sys.argv[1]
task_file = sys.argv[2]

with open(csv_file, 'r', encoding='utf-8-sig') as f:
    reader = csv.DictReader(f)
    fieldnames = [fn.strip() for fn in reader.fieldnames]

    mr_col = next((fn for fn in fieldnames if fn == 'mr_iid'), None)
    repo_col = next((fn for fn in fieldnames if fn == 'repo_web_url'), None)

    if not mr_col or not repo_col:
        print(f"❌ CSV 中未找到 mr_iid 或 repo_web_url 列")
        print(f"   可用列: {fieldnames}")
        sys.exit(1)

    count = 0
    with open(task_file, 'w') as out:
        for row in reader:
            cleaned = {k.strip(): v for k, v in row.items()}
            mr_iid = cleaned.get(mr_col, '').strip()
            repo_url = cleaned.get(repo_col, '').strip()
            if mr_iid and repo_url and mr_iid.isdigit():
                out.write(f"{mr_iid},{repo_url}\n")
                count += 1

    print(f"✅ 解析完成: {count} 条有效任务")
PYEOF

if [ $? -ne 0 ]; then
    echo "❌ CSV 解析失败"
    exit 1
fi

TOTAL=$(wc -l < ${TASK_FILE})

# 显示 cherry-pick 配置信息
if [ -n "$CHERRY_PICK_CONFIG" ] && [ -f "$CHERRY_PICK_CONFIG" ]; then
    CP_REPOS=$(python3 -c "
import json
with open('${CHERRY_PICK_CONFIG}') as f:
    config = json.load(f)
for repo, commits in config.items():
    if commits:
        print(f'   {repo}: {len(commits)} 个 commit')
" 2>/dev/null || echo "   (解析失败)")
    echo "📋 Cherry-pick 配置:"
    echo "$CP_REPOS"
else
    echo "📋 Cherry-pick: 使用默认配置"
fi

echo "📊 共 ${TOTAL} 条任务，版本: ${VERSION}"
echo ""
echo "============================================"

# ---- 逐条调用 create_base_branch.sh ----
IDX=0
SUCCESS=0
FAIL=0

while IFS=',' read -r MR_ID REPO; do
    IDX=$((IDX + 1))
    REPO_NAME=$(basename ${REPO})
    BASE_BRANCH_NAME="ai_coding_test/base/${VERSION}/${MR_ID}"
    LOG_FILE="${LOG_DIR}/${REPO_NAME}_mr${MR_ID}.log"

    echo ""
    echo "[${IDX}/${TOTAL}] ${REPO_NAME} MR-${MR_ID}"

    # 构造调用参数
    CMD_ARGS=("${REPO}" "${MR_ID}" "${VERSION}")
    if [ -n "$CHERRY_PICK_CONFIG" ]; then
        CMD_ARGS+=("${CHERRY_PICK_CONFIG}")
    fi

    if "${CREATE_SCRIPT}" "${CMD_ARGS[@]}" 2>&1 | tee ${LOG_FILE}; then
        SUCCESS=$((SUCCESS + 1))
        echo "${MR_ID},${REPO},SUCCESS,${BASE_BRANCH_NAME},完成" >> ${RESULT_FILE}
    else
        EXIT_CODE=$?
        FAIL=$((FAIL + 1))
        case $EXIT_CODE in
            2) MSG="MR引用未找到" ;;
            3) MSG="merge_commit未找到" ;;
            4) MSG="推送失败" ;;
            5) MSG="cherry-pick冲突" ;;
            *) MSG="执行异常(exit=${EXIT_CODE})" ;;
        esac
        echo "${MR_ID},${REPO},FAIL,${BASE_BRANCH_NAME},${MSG}" >> ${RESULT_FILE}
    fi

done < ${TASK_FILE}

# ---- 汇总 ----
echo ""
echo "============================================"
echo "📊 执行结果汇总:"
echo ""
echo "   总数:     ${TOTAL}"
echo "   ✅ 成功:  ${SUCCESS}"
echo "   ❌ 失败:  ${FAIL}"
echo ""
echo "   结果文件: ${RESULT_FILE}"
echo "   日志目录: ${LOG_DIR}"

if [ "$FAIL" -gt 0 ]; then
    echo ""
    echo "失败项:"
    grep ',FAIL,' ${RESULT_FILE} | while IFS=',' read -r mr repo status branch msg; do
        echo "   MR-${mr} ($(basename ${repo})): ${msg}"
    done
fi

echo ""
echo "🎉 批量处理完成！"
