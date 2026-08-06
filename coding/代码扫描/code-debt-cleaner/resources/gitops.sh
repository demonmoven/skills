#!/bin/bash

# Code-Debt-Cleaner GitOps - Git 操作脚本
# 支持创建分支、提交、推送
# 日志输出到 stdout（用 [LOG] 标记）

set -e

# 日志函数（输出到 stdout，用 [LOG] 标记）
log() {
    echo "[LOG] [gitops] $1"
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 默认参数
BRANCH_NAME=""
COMMIT_MESSAGE="feat: clean_code_debt"
AUTO_PUSH=false

# 帮助信息
show_help() {
    echo "Code-Debt-Cleaner GitOps - Git 操作脚本"
    echo ""
    echo "用法: bash gitops.sh <命令> [选项]"
    echo ""
    echo "命令:"
    echo "  create-branch   创建新分支"
    echo "  commit          提交修改"
    echo "  push            推送到远程"
    echo "  all             执行完整流程：创建分支 → 提交 → 推送"
    echo ""
    echo "选项:"
    echo "  -b, --branch <分支名>    指定分支名（默认：code-debt-cleaner-{timestamp}）"
    echo "  -m, --message <提交信息>  指定提交信息（默认：feat: clean_code_debt）"
    echo "  -p, --push               自动推送到远程（仅用于 create-branch 或 commit）"
    echo "  -h, --help               显示帮助信息"
    echo ""
    echo "示例:"
    echo "  bash gitops.sh create-branch"
    echo "  bash gitops.sh create-branch -b my-fix-branch"
    echo "  bash gitops.sh commit -m \"fix unused imports\""
    echo "  bash gitops.sh push"
    echo "  bash gitops.sh all -m \"clean code debt\""
    exit 0
}

# 解析命令行参数
parse_args() {
    COMMAND=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -b|--branch)
                BRANCH_NAME="$2"
                shift 2
                ;;
            -m|--message)
                COMMIT_MESSAGE="$2"
                shift 2
                ;;
            -p|--push)
                AUTO_PUSH=true
                shift 1
                ;;
            -h|--help)
                show_help
                ;;
            create-branch|commit|push|all)
                COMMAND="$1"
                shift 1
                ;;
            *)
                log "错误: 未知选项 $1"
                show_help
                ;;
        esac
    done
}

# 创建分支
create_branch() {
    log "创建新分支..."
    
    if [ -z "$BRANCH_NAME" ]; then
        local timestamp=$(date +"%Y%m%d-%H%M%S")
        BRANCH_NAME="code-debt-cleaner-$timestamp"
    fi
    
    log "  分支名: $BRANCH_NAME"
    git checkout -b "$BRANCH_NAME"
    
    log "✓ 分支创建成功"
    echo "$BRANCH_NAME"
}

# 提交修改
commit() {
    log "提交修改..."
    
    log "  提交信息: \"$COMMIT_MESSAGE\""
    git add .
    git commit -m "$COMMIT_MESSAGE"
    
    log "✓ 提交成功"
}

# 推送
push() {
    log "推送到远程..."
    
    git push -u origin HEAD
    
    log "✓ 推送成功"
}

# 主函数
main() {
    parse_args "$@"
    
    log "=========================================="
    log "Code-Debt-Cleaner GitOps 开始运行"
    log "=========================================="
    
    if [ -z "$COMMAND" ]; then
        log "错误: 未指定命令"
        show_help
    fi
    
    case "$COMMAND" in
        create-branch)
            create_branch
            if [ "$AUTO_PUSH" = true ]; then
                push
            fi
            ;;
        commit)
            commit
            if [ "$AUTO_PUSH" = true ]; then
                push
            fi
            ;;
        push)
            push
            ;;
        all)
            create_branch
            commit
            push
            ;;
        *)
            log "错误：未知命令 $COMMAND"
            log "用法: $0 {create-branch|commit|push|all}"
            exit 1
            ;;
    esac
}

main "$@"
