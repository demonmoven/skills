#!/usr/bin/env bash
#
# AI Coding Batch Pipeline
# 功能: 读取 CSV 文件，逐行调用单任务脚本 ai_coding_pipeline.sh
#
# 用法:
#   chmod +x ai_coding_batch.sh && bash ai_coding_batch.sh \
#     --version <版本号> \
#     --csv <CSV文件路径>
#
# CSV 格式 (首行为表头，逗号分隔):
#   mr_iid,repo_web_url,spec_url
#   592,https://code.byted.org/ad/star_settlement,星立方预借款功能描述性文档
#

set -uo pipefail
# 注意: 不使用 set -e，因为单行失败不应中止整个批量流程

# ============================================================
# 颜色与日志
# ============================================================
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

log_info()  { echo -e "${GREEN}[BATCH-INFO]${NC}  $(date '+%Y-%m-%d %H:%M:%S') $*"; }
log_warn()  { echo -e "${YELLOW}[BATCH-WARN]${NC}  $(date '+%Y-%m-%d %H:%M:%S') $*"; }
log_error() { echo -e "${RED}[BATCH-ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') $*"; }
log_step()  { echo -e "${BLUE}[BATCH-STEP]${NC}  $(date '+%Y-%m-%d %H:%M:%S') ===== $* ====="; }

# ============================================================
# 全局变量
# ============================================================
VERSION=""
CSV_FILE=""
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PIPELINE_SCRIPT="${SCRIPT_DIR}/ai_coding_pipeline.sh"

# 统计
TOTAL_COUNT=0
SUCCESS_COUNT=0
FAILED_COUNT=0
SKIPPED_COUNT=0

# 结果记录 (用数组存储，兼容 bash 3.2)
RESULT_LINES=""

# ============================================================
# 参数解析
# ============================================================
usage() {
    cat <<EOF
用法:
  chmod +x $(basename "$0") && bash $(basename "$0") \\
    --version  <版本号> \\
    --csv      <CSV文件路径>

参数说明:
  --version    版本号，如 v1.0.0 (必填)
  --csv        CSV 文件路径 (必填)
  -h, --help   显示帮助信息

CSV 格式 (首行为表头):
  mr_iid,repo_web_url,spec_url
EOF
    exit 0
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --version)  VERSION="$2";   shift 2 ;;
            --csv)      CSV_FILE="$2";  shift 2 ;;
            -h|--help)  usage ;;
            *)          log_error "未知参数: $1"; usage ;;
        esac
    done
    [[ -z "$VERSION"  ]] && { log_error "--version 参数为必填项"; usage; }
    [[ -z "$CSV_FILE" ]] && { log_error "--csv 参数为必填项";     usage; }

    # 校验文件存在
    if [[ ! -f "$CSV_FILE" ]]; then
        log_error "CSV 文件不存在: $CSV_FILE"
        exit 1
    fi

    # 校验单任务脚本存在
    if [[ ! -f "$PIPELINE_SCRIPT" ]]; then
        log_error "单任务脚本不存在: $PIPELINE_SCRIPT"
        log_error "请确保 ai_coding_pipeline.sh 与本脚本在同一目录"
        exit 1
    fi

    log_info "参数解析完成:"
    log_info "  VERSION  = $VERSION"
    log_info "  CSV_FILE = $CSV_FILE"
    log_info "  PIPELINE = $PIPELINE_SCRIPT"
}

# ============================================================
# CSV 解析与执行
# ============================================================
run_batch() {
    local line_num=0

    while IFS= read -r line || [[ -n "$line" ]]; do
        line_num=$((line_num + 1))

        # 跳过表头
        if [[ $line_num -eq 1 ]]; then
            # 校验表头格式
            if echo "$line" | grep -q "mr_iid"; then
                log_info "检测到表头，跳过: $line"
                continue
            else
                log_warn "未检测到标准表头，将第一行也作为数据处理"
            fi
        fi

        # 跳过空行
        line=$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [[ -z "$line" ]]; then
            continue
        fi

        # 解析 CSV (逗号分隔，取前3个字段)
        local mr_iid repo_web_url spec_url
        mr_iid=$(echo "$line" | cut -d',' -f1 | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        repo_web_url=$(echo "$line" | cut -d',' -f2 | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        # spec_url 可能包含逗号，取第3个字段到末尾
        spec_url=$(echo "$line" | cut -d',' -f3- | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

        # 校验必填字段
        if [[ -z "$mr_iid" || -z "$repo_web_url" || -z "$spec_url" ]]; then
            log_warn "第 $line_num 行数据不完整，跳过: $line"
            SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
            RESULT_LINES="${RESULT_LINES}SKIP|${mr_iid:-N/A}|数据不完整\n"
            continue
        fi

        TOTAL_COUNT=$((TOTAL_COUNT + 1))

        log_step "任务 $TOTAL_COUNT: MR=$mr_iid REPO=$repo_web_url"
        log_info "  spec_url = $spec_url"

        # 调用单任务脚本
        local start_time end_time duration exit_code
        start_time=$(date +%s)

        bash "$PIPELINE_SCRIPT" \
            --repo-url "$repo_web_url" \
            --version "$VERSION" \
            --mr-id "$mr_iid" \
            --spec-url "$spec_url"
        exit_code=$?

        end_time=$(date +%s)
        duration=$((end_time - start_time))

        if [[ $exit_code -eq 0 ]]; then
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
            RESULT_LINES="${RESULT_LINES}OK|${mr_iid}|${duration}s\n"
            log_info "任务 $TOTAL_COUNT (MR=$mr_iid) 成功，耗时 ${duration}s"
        else
            FAILED_COUNT=$((FAILED_COUNT + 1))
            RESULT_LINES="${RESULT_LINES}FAIL|${mr_iid}|exit_code=${exit_code} ${duration}s\n"
            log_error "任务 $TOTAL_COUNT (MR=$mr_iid) 失败 (exit_code=$exit_code)，耗时 ${duration}s"
        fi

        echo ""

    done < "$CSV_FILE"
}

# ============================================================
# 汇总报告
# ============================================================
print_summary() {
    echo ""
    log_step "批量执行汇总报告"
    echo "=============================================="
    echo "  AI Coding Batch Pipeline 汇总"
    echo "=============================================="
    echo "  版本号:    $VERSION"
    echo "  CSV 文件:  $CSV_FILE"
    echo "  总任务数:  $TOTAL_COUNT"
    echo -e "  ${GREEN}成功:${NC}      $SUCCESS_COUNT"
    echo -e "  ${RED}失败:${NC}      $FAILED_COUNT"
    echo -e "  ${YELLOW}跳过:${NC}      $SKIPPED_COUNT"
    echo "----------------------------------------------"
    echo "  详细结果:"
    echo "----------------------------------------------"
    printf "  %-8s %-10s %s\n" "状态" "MR_ID" "备注"
    echo "  ------   --------   ----"
    echo -e "$RESULT_LINES" | while IFS='|' read -r status mr_id note; do
        [[ -z "$status" ]] && continue
        case "$status" in
            OK)   printf "  ${GREEN}%-8s${NC} %-10s %s\n" "✔ OK" "$mr_id" "$note" ;;
            FAIL) printf "  ${RED}%-8s${NC} %-10s %s\n" "✘ FAIL" "$mr_id" "$note" ;;
            SKIP) printf "  ${YELLOW}%-8s${NC} %-10s %s\n" "– SKIP" "$mr_id" "$note" ;;
        esac
    done
    echo "=============================================="

    if [[ $FAILED_COUNT -gt 0 ]]; then
        log_warn "有 $FAILED_COUNT 个任务执行失败，请检查上方日志"
    fi
    if [[ $FAILED_COUNT -eq 0 && $SKIPPED_COUNT -eq 0 ]]; then
        log_info "全部任务执行成功！"
    fi
}

# ============================================================
# 主流程
# ============================================================
main() {
    echo "=============================================="
    echo "  AI Coding Batch Pipeline"
    echo "  $(date '+%Y-%m-%d %H:%M:%S')"
    echo "=============================================="

    parse_args "$@"
    run_batch
    print_summary

    # 如果有失败任务，以非零退出码结束
    if [[ $FAILED_COUNT -gt 0 ]]; then
        exit 1
    fi
}

main "$@"