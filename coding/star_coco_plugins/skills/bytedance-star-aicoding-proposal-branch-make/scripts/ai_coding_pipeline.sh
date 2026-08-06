#!/usr/bin/env bash
#
# AI Coding Pipeline Script
# 功能: 自动化完成代码拉取 → coco环境安装 → 分支管理 → 提案执行 → 推送 → 清理
#
# 用法:
#   chmod +x ai_coding_pipeline.sh && bash ai_coding_pipeline.sh \
#     --repo-url <代码仓库URL> \
#     --version <版本号> \
#     --mr-id <MR ID> \
#     --spec-url <需求规格文档URL>
#

set -euo pipefail

# ============================================================
# 颜色与日志
# ============================================================
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC}  $(date '+%Y-%m-%d %H:%M:%S') $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $(date '+%Y-%m-%d %H:%M:%S') $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') $*"; }
log_step()  { echo -e "${BLUE}[STEP]${NC}  $(date '+%Y-%m-%d %H:%M:%S') ===== $* ====="; }

# ============================================================
# 全局变量
# ============================================================
REPO_URL=""; VERSION=""; MR_ID=""; SPEC_URL=""
WORK_DIR=""; REPO_NAME=""; BASE_BRANCH=""; PROPOSAL_BRANCH=""

# 步骤状态 (普通变量，兼容 bash 3.2 / macOS)
STATUS_STEP0="SKIPPED"
STATUS_STEP1="SKIPPED"
STATUS_STEP2="SKIPPED"
STATUS_STEP3="SKIPPED"
STATUS_STEP4="SKIPPED"
STATUS_STEP5="SKIPPED"

# ============================================================
# 异常处理与清理
# ============================================================
print_report() {
    local label="$1"
    local status="$2"
    case "$status" in
        SUCCESS) echo -e "  ${GREEN}✔${NC} $label: ${GREEN}SUCCESS${NC}" ;;
        FAILED)  echo -e "  ${RED}✘${NC} $label: ${RED}FAILED${NC}" ;;
        *)       echo -e "  ${YELLOW}–${NC} $label: ${YELLOW}SKIPPED${NC}" ;;
    esac
}

cleanup() {
    local exit_code=$?
    echo ""
    log_step "流程结束 - 状态报告"
    echo "=============================================="
    echo "  AI Coding Pipeline 执行报告"
    echo "=============================================="
    print_report "Step0-代码拉取" "$STATUS_STEP0"
    print_report "Step1-环境安装" "$STATUS_STEP1"
    print_report "Step2-分支管理" "$STATUS_STEP2"
    print_report "Step3-提案执行" "$STATUS_STEP3"
    print_report "Step4-代码推送" "$STATUS_STEP4"
    print_report "Step5-环境清理" "$STATUS_STEP5"
    echo "=============================================="
    if [[ $exit_code -ne 0 ]]; then
        log_error "脚本以非零退出码 ($exit_code) 结束"
    else
        log_info "脚本执行完成"
    fi
    exit $exit_code
}
trap cleanup EXIT

fail_step() {
    local step_var="$1"
    local msg="$2"
    eval "$step_var=FAILED"
    log_error "$msg"
    exit 1
}

# ============================================================
# 参数解析
# ============================================================
usage() {
    cat <<EOF
用法:
  chmod +x $(basename "$0") && bash $(basename "$0") \\
    --repo-url  <代码仓库URL> \\
    --version   <版本号> \\
    --mr-id     <MR ID> \\
    --spec-url  <需求规格文档URL>

参数说明:
  --repo-url   代码仓库的 Git 克隆地址 (必填)
  --version    版本号，如 v1.0.0 (必填)
  --mr-id      Merge Request ID (必填)
  --spec-url   需求规格文档 URL (必填)
  -h, --help   显示帮助信息
EOF
    exit 0
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --repo-url)  REPO_URL="$2";  shift 2 ;;
            --version)   VERSION="$2";   shift 2 ;;
            --mr-id)     MR_ID="$2";     shift 2 ;;
            --spec-url)  SPEC_URL="$2";  shift 2 ;;
            -h|--help)   usage ;;
            *)           log_error "未知参数: $1"; usage ;;
        esac
    done
    [[ -z "$REPO_URL" ]] && { log_error "--repo-url 参数为必填项"; usage; }
    [[ -z "$VERSION"  ]] && { log_error "--version 参数为必填项";  usage; }
    [[ -z "$MR_ID"    ]] && { log_error "--mr-id 参数为必填项";    usage; }
    [[ -z "$SPEC_URL" ]] && { log_error "--spec-url 参数为必填项"; usage; }

    REPO_NAME=$(basename "$REPO_URL" .git)
    WORK_DIR="$(pwd)/ai_coding_workspace/${REPO_NAME}"
    BASE_BRANCH="ai_coding_test/base/${VERSION}/${MR_ID}"
    PROPOSAL_BRANCH="ai_coding_test/proposal/${VERSION}/${MR_ID}"

    log_info "参数解析完成:"
    log_info "  REPO_URL         = $REPO_URL"
    log_info "  VERSION          = $VERSION"
    log_info "  MR_ID            = $MR_ID"
    log_info "  SPEC_URL         = $SPEC_URL"
    log_info "  WORK_DIR         = $WORK_DIR"
    log_info "  BASE_BRANCH      = $BASE_BRANCH"
    log_info "  PROPOSAL_BRANCH  = $PROPOSAL_BRANCH"
}

# ============================================================
# Step 0: 拉取代码 (仅克隆 master 分支)
# ============================================================
step0_clone_repo() {
    log_step "Step 0: 拉取代码仓库 (master 分支)"

    if [[ -d "$WORK_DIR" ]]; then
        log_warn "工作目录已存在: $WORK_DIR，将先删除后重新克隆"
        rm -rf "$WORK_DIR"
    fi
    mkdir -p "$WORK_DIR"
    log_info "工作目录已创建: $WORK_DIR"

    log_info "正在克隆仓库 master 分支: $REPO_URL"
    if ! git clone --branch master --single-branch "$REPO_URL" "$WORK_DIR"; then
        log_warn "master 分支克隆失败，尝试 main 分支..."
        rm -rf "$WORK_DIR"; mkdir -p "$WORK_DIR"
        if ! git clone --branch main --single-branch "$REPO_URL" "$WORK_DIR"; then
            fail_step "STATUS_STEP0" "[Step0] 仓库克隆失败，请检查仓库URL和网络连接"
        fi
    fi

    cd "$WORK_DIR"
    log_info "代码克隆成功，当前目录: $(pwd)"
    STATUS_STEP0="SUCCESS"
}

# ============================================================
# Step 1: Coco 环境安装
# ============================================================
step1_install_env() {
    log_step "Step 1: Coco 环境安装"

    # 1.1 nvm
    log_info "1.1 安装 nvm..."
    export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
    if [[ -s "$NVM_DIR/nvm.sh" ]]; then
        log_info "nvm 已安装，跳过下载"
    else
        if ! curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash; then
            fail_step "STATUS_STEP1" "[Step1] nvm 安装失败"
        fi
    fi
    # shellcheck source=/dev/null
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
    [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"
    log_info "nvm 版本: $(nvm --version 2>/dev/null || echo '加载失败')"

    # 1.2 Node.js 24
    log_info "1.2 安装 Node.js 24..."
    nvm install 24 || fail_step "STATUS_STEP1" "[Step1] Node.js 24 安装失败"
    nvm use 24
    log_info "Node.js: $(node --version) | npm: $(npm --version)"
    npm config set registry https://bnpm.byted.org/

    # 1.3 OpenSpec
    log_info "1.3 安装 OpenSpec..."
    npm install -g @bytedance-dev/openspec@v0.16.0 \
      || fail_step "STATUS_STEP1" "[Step1] OpenSpec 安装失败"
    log_info "OpenSpec: $(openspec --version 2>/dev/null || echo '版本获取失败')"

    # 1.4 Coco CLI (已安装则跳过)
    log_info "1.4 安装 Coco CLI..."
    if command -v coco &>/dev/null; then
        log_info "Coco CLI 已安装 ($(coco --version 2>/dev/null || echo '版本未知'))，跳过"
    else
        sh -c "$(curl -L https://code.byted.org/api/tos-proxy/download/adopt_coco.sh)" \
          || fail_step "STATUS_STEP1" "[Step1] Coco CLI 安装失败"
        export PATH=~/.local/bin:$PATH
        log_info "Coco CLI: $(coco --version 2>/dev/null || echo '版本获取失败')"
    fi

    # 1.5 星图 Coco 插件 (已安装则跳过)
    log_info "1.5 安装 Coco 星图插件..."
    if coco plugin list 2>/dev/null | grep -q "ad/star_coco_plugin"; then
        log_info "Coco 星图插件已安装，跳过"
    else
        coco plugin install --type=codebase ad/star_coco_plugin \
          || fail_step "STATUS_STEP1" "[Step1] Coco 星图插件安装失败"
        log_info "Coco 星图插件安装完成"
    fi

    STATUS_STEP1="SUCCESS"
}

# ============================================================
# Step 2: 分支拉取与切换
# ============================================================
step2_branch_management() {
    log_step "Step 2: 分支拉取与切换"
    cd "$WORK_DIR"

    # 2.1 精准拉取远端 base 分支 (使用完整 refspec 确保创建 origin/xxx 引用)
    log_info "2.1 拉取远端 base 分支: $BASE_BRANCH"
    if ! git fetch origin "${BASE_BRANCH}:refs/remotes/origin/${BASE_BRANCH}"; then
        fail_step "STATUS_STEP2" "[Step2] 远端 base 分支 $BASE_BRANCH 不存在或拉取失败，请先创建该分支"
    fi

    log_info "2.1 切换到 base 分支: $BASE_BRANCH"
    git checkout -B "$BASE_BRANCH" "origin/$BASE_BRANCH" \
      || fail_step "STATUS_STEP2" "[Step2] checkout 到分支 $BASE_BRANCH 失败"

    # 2.2 拉取最新代码
    log_info "2.2 拉取分支最新代码..."
    git pull origin "$BASE_BRANCH" --rebase \
      || fail_step "STATUS_STEP2" "[Step2] 拉取 base 分支最新代码失败"

    # 2.3 创建 proposal 分支 (不可已存在)
    log_info "2.3 基于 $BASE_BRANCH 创建 proposal 分支: $PROPOSAL_BRANCH"
    if git ls-remote --exit-code --heads origin "$PROPOSAL_BRANCH" &>/dev/null; then
        fail_step "STATUS_STEP2" "[Step2] 远端 proposal 分支 $PROPOSAL_BRANCH 已存在，请先处理后重试"
    fi
    if git rev-parse --verify "$PROPOSAL_BRANCH" &>/dev/null; then
        fail_step "STATUS_STEP2" "[Step2] 本地 proposal 分支 $PROPOSAL_BRANCH 已存在，请先处理后重试"
    fi
    git checkout -b "$PROPOSAL_BRANCH" \
      || fail_step "STATUS_STEP2" "[Step2] 创建 proposal 分支 $PROPOSAL_BRANCH 失败"

    log_info "当前分支: $(git branch --show-current)"
    STATUS_STEP2="SUCCESS"
}

# ============================================================
# Step 3: 执行提案流程
# ============================================================
step3_run_proposal() {
    log_step "Step 3: 执行提案流程"
    cd "$WORK_DIR"

    log_info "Spec 文档: $SPEC_URL"
    coco -p \
      --allowed-tool Bash,Edit,MultiEdit,Write \
      -p /project:openspec:proposal \
      "根据文档内容 ${SPEC_URL} 发起提案 不要问我任何问题" \
      || fail_step "STATUS_STEP3" "[Step3] coco 提案执行失败，请检查 coco 配置和 Spec 文档链接"

    log_info "提案流程执行完成"
    STATUS_STEP3="SUCCESS"
}

# ============================================================
# Step 4: 推送分支到远端
# ============================================================
step4_push_branch() {
    log_step "Step 4: 推送代码到远端"
    cd "$WORK_DIR"

    if [[ -n $(git status --porcelain) ]]; then
        log_info "检测到代码变更，正在提交..."
        git add -A
        git commit -m "feat: AI coding proposal for ${VERSION}/${MR_ID}

- Generated by AI Coding Pipeline
- Spec: ${SPEC_URL}
- Base branch: ${BASE_BRANCH}
- Proposal branch: ${PROPOSAL_BRANCH}"
    else
        log_warn "没有检测到新的代码变更"
    fi

    log_info "正在推送分支 $PROPOSAL_BRANCH 到远端..."
    git push origin "$PROPOSAL_BRANCH" --force-with-lease \
      || fail_step "STATUS_STEP4" "[Step4] 代码推送失败，请检查仓库权限和网络连接"

    log_info "代码推送成功: $PROPOSAL_BRANCH"
    STATUS_STEP4="SUCCESS"
}

# ============================================================
# Step 5: 环境清理
# ============================================================
step5_cleanup() {
    log_step "Step 5: 环境清理"

    npm cache clean --force 2>/dev/null || true

    if [[ -n "$WORK_DIR" && -d "$WORK_DIR" ]]; then
        log_info "删除工作目录: $WORK_DIR"
        rm -rf "$WORK_DIR"
    fi

    log_info "环境清理完成"
    STATUS_STEP5="SUCCESS"
}

# ============================================================
# 主流程
# ============================================================
main() {
    echo "=============================================="
    echo "  AI Coding Pipeline"
    echo "  $(date '+%Y-%m-%d %H:%M:%S')"
    echo "=============================================="
    parse_args "$@"
    step0_clone_repo
    step1_install_env
    step2_branch_management
    step3_run_proposal
    step4_push_branch
    step5_cleanup
    log_info "全部流程执行成功！"
}

main "$@"