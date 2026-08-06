#!/bin/bash

# Code-Debt-Cleaner Prepare - 随机选择问题类型和目录
# 输出格式：JSON
# 日志输出到 stdout（用 [LOG] 标记），JSON 结果也输出到 stdout

set -e

# 日志函数（输出到 stdout，用 [LOG] 标记）
log() {
    echo "[LOG] [prepare] $1"
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 默认参数
SCAN_DIR="./"

# 所有支持的类型（共 35 种）
SUPPORTED_TYPES="long_method,large_class,long_parameter_list,divergent_change,shotgun_surgery,feature_envy,data_clumps,primitive_obsession,switch_statements,parallel_inheritance_hierarchies,lazy_class,speculative_generality,temporary_field,message_chains,middle_man,inappropriate_intimacy,alternative_classes_with_different_interfaces,incomplete_library_class,data_class,refused_bequest,comments,duplicated_code,staticcheck,gosimple,stylecheck,gofmt,goimports,gocyclo,errcheck,ineffassign,unused,revive,cyclop,govet,typecheck"

# 类型处理方式映射
# format: "type:handler"
# handler: "tool" 或 "llm"
TYPE_HANDLERS=(
    "long_method:llm"
    "large_class:llm"
    "long_parameter_list:llm"
    "divergent_change:llm"
    "shotgun_surgery:llm"
    "feature_envy:llm"
    "data_clumps:llm"
    "primitive_obsession:llm"
    "switch_statements:llm"
    "parallel_inheritance_hierarchies:llm"
    "lazy_class:llm"
    "speculative_generality:llm"
    "temporary_field:llm"
    "message_chains:llm"
    "middle_man:llm"
    "inappropriate_intimacy:llm"
    "alternative_classes_with_different_interfaces:llm"
    "incomplete_library_class:llm"
    "data_class:llm"
    "refused_bequest:llm"
    "comments:llm"
    "duplicated_code:tool"
    "staticcheck:tool"
    "gosimple:tool"
    "stylecheck:tool"
    "gofmt:tool"
    "goimports:tool"
    "gocyclo:tool"
    "errcheck:tool"
    "ineffassign:tool"
    "unused:tool"
    "revive:tool"
    "cyclop:tool"
    "govet:tool"
    "typecheck:tool"
)

# 按风险分类的类型
ALL_LOW_RISK="long_method,large_class,long_parameter_list,data_clumps,primitive_obsession,lazy_class,temporary_field,middle_man,data_class,comments,gofmt,goimports,gosimple,stylecheck,typecheck"
ALL_MEDIUM_RISK="ineffassign,unused,revive,duplicated_code"
ALL_HIGH_RISK="divergent_change,shotgun_surgery,feature_envy,switch_statements,parallel_inheritance_hierarchies,speculative_generality,message_chains,inappropriate_intimacy,alternative_classes_with_different_interfaces,incomplete_library_class,refused_bequest,staticcheck,gocyclo,errcheck,cyclop,govet"

# 获取类型的处理方式
get_handler() {
    local type="$1"
    for handler in "${TYPE_HANDLERS[@]}"; do
        if [[ "$handler" == "$type:"* ]]; then
            echo "${handler#$type:}"
            return
        fi
    done
    echo "llm"  # 默认用llm
}

# 随机选择一些问题类型用于 autoloop 模式
randomize_types() {
    # 从所有低、中、高风险类型中随机选择
    local low=(${ALL_LOW_RISK//,/ })
    local medium=(${ALL_MEDIUM_RISK//,/ })
    local high=(${ALL_HIGH_RISK//,/ })
    local all_types=("${low[@]}" "${medium[@]}" "${high[@]}")
    
    # 随机选择 2-4 个类型
    local num_to_pick=$((2 + RANDOM % 3))
    if [ $num_to_pick -gt ${#all_types[@]} ]; then
        num_to_pick=${#all_types[@]}
    fi
    
    local selected=()
    local temp=("${all_types[@]}")
    
    # 洗牌算法随机选择
    while [ ${#selected[@]} -lt $num_to_pick ] && [ ${#temp[@]} -gt 0 ]; do
        local idx=$((RANDOM % ${#temp[@]}))
        selected+=("${temp[$idx]}")
        # 移除已选择的
        temp=(${temp[@]:0:$idx} ${temp[@]:$((idx + 1))})
    done
    
    # 用逗号连接
    local result=$(IFS=,; echo "${selected[*]}")
    echo "$result"
}

# 随机选择一些目录用于 autoloop 模式（兼容多种目录结构）
randomize_directories() {
    local scan_dir="$1"
    
    # 收集所有有效目录，同时记录深度
    local temp_file=$(mktemp)
    
    while IFS= read -r dir; do
        # 跳过根目录
        if [ "$dir" = "$scan_dir" ] || [ "$dir" = "." ]; then
            continue
        fi
        
        # 检查目录是否存在
        if [ ! -d "$dir" ]; then
            continue
        fi
        
        # 检查是否有 Go 文件
        if ! ls "$dir"/*.go 1>/dev/null 2>&1; then
            continue
        fi
        
        # 排除以 . 开头的目录
        local basename=$(basename "$dir")
        if [[ "$basename" =~ ^\. ]]; then
            continue
        fi
        
        # 计算路径深度
        local depth=$(echo "$dir" | tr '/' '\n' | grep -v '^$' | wc -l | awk '{print $1}')
        
        # 保存深度和目录（深度在前，方便排序）
        echo "$depth $dir" >> "$temp_file"
    done < <(find "$scan_dir" -maxdepth 4 -type d -not -path "*/.git/*" -not -path "*/vendor/*" -not -path "*/testdata/*" -not -path "*/mocks/*" -not -path "*/opp/*" 2>/dev/null)
    
    # 按深度从大到小排序，然后提取目录
    local all_dirs=()
    while read -r depth dir; do
        if [ -n "$dir" ] && [ "$depth" -ge 2 ] && [ -d "$dir" ]; then
            all_dirs+=("$dir")
        fi
    done < <(sort -nr "$temp_file" 2>/dev/null)
    
    rm -f "$temp_file"
    
    # 如果还是没有目录，返回当前目录
    if [ ${#all_dirs[@]} -eq 0 ]; then
        echo "./"
        return
    fi
    
    # 随机选择 1-3 个目录
    local num_to_pick=$((1 + RANDOM % 3))
    if [ $num_to_pick -gt ${#all_dirs[@]} ]; then
        num_to_pick=${#all_dirs[@]}
    fi
    
    local selected=()
    local temp_dirs=("${all_dirs[@]}")
    
    # 洗牌算法随机选择
    while [ ${#selected[@]} -lt $num_to_pick ] && [ ${#temp_dirs[@]} -gt 0 ]; do
        local idx=$((RANDOM % ${#temp_dirs[@]}))
        local candidate_dir="${temp_dirs[$idx]}"
        # 再次验证目录是否存在且有 Go 文件
        if [ -d "$candidate_dir" ] && ls "$candidate_dir"/*.go 1>/dev/null 2>&1; then
            selected+=("$candidate_dir")
        fi
        # 移除已选择的（无论是否有效都移除，避免死循环）
        temp_dirs=(${temp_dirs[@]:0:$idx} ${temp_dirs[@]:$((idx + 1))})
    done
    
    # 如果没找到有效目录，返回当前目录
    if [ ${#selected[@]} -eq 0 ]; then
        echo "./"
        return
    fi
    
    # 用 ./dir 格式输出空格分隔的字符串（不带 /...，scout.sh 需要正确的目录路径）
    # 最后再检查一次所有目录是否存在
    local result=""
    for dir in "${selected[@]}"; do
        # 确保路径格式正确
        local clean_dir="${dir#./}"
        local full_dir="./$clean_dir"
        # 最后再检查一次
        if [ -d "$full_dir" ] && ls "$full_dir"/*.go 1>/dev/null 2>&1; then
            if [ -z "$result" ]; then
                result="$full_dir"
            else
                result="$result $full_dir"
            fi
        fi
    done
    
    # 如果过滤后没有目录了，返回当前目录
    if [ -z "$result" ]; then
        echo "./"
        return
    fi
    
    echo "$result"
}

# 帮助信息
show_help() {
    echo "Code-Debt-Cleaner Prepare - 随机选择问题类型和目录"
    echo ""
    echo "用法: bash prepare.sh [选项]"
    echo ""
    echo "选项:"
    echo "  -d, --dir <目录>        指定扫描目录 (默认: ./)"
    echo "  -h, --help                显示帮助信息"
    echo ""
    echo "示例:"
    echo "  bash prepare.sh"
    echo "  bash prepare.sh -d ./pkg"
    exit 0
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -d|--dir)
                SCAN_DIR="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                ;;
            *)
                log "错误: 未知选项 $1"
                show_help
                ;;
        esac
    done
}

# 检查是否在 Go 项目根目录
check_go_project() {
    log "检查是否在 Go 项目根目录..."
    if [ ! -f "go.mod" ]; then
        log "错误：未找到 go.mod 文件"
        echo "{\"error\": \"Not in a Go project root (go.mod not found)\"}"
        exit 1
    fi
    log "✓ 找到 go.mod 文件，确认是 Go 项目"
}

# 主函数
main() {
    # 解析命令行参数
    parse_args "$@"
    
    log "=========================================="
    log "Code-Debt-Cleaner Prepare 开始运行"
    log "扫描目录: $SCAN_DIR"
    log "=========================================="
    
    check_go_project
    
    # 随机选择问题类型
    log "随机选择问题类型..."
    local all_selected=$(randomize_types)
    log "  随机选择的问题类型: $all_selected"
    
    # 为每个类型确定处理方式
    local tool_types=""
    local llm_types=""
    IFS=',' read -ra selected_types <<< "$all_selected"
    for type in "${selected_types[@]}"; do
        local handler=$(get_handler "$type")
        if [ "$handler" = "tool" ]; then
            if [ -z "$tool_types" ]; then
                tool_types="$type"
            else
                tool_types="$tool_types,$type"
            fi
        else
            if [ -z "$llm_types" ]; then
                llm_types="$type"
            else
                llm_types="$llm_types,$type"
            fi
        fi
    done
    
    # 随机选择目录
    log "随机选择目录..."
    local random_dirs=$(randomize_directories "$SCAN_DIR")
    log "  随机选择的目录: $random_dirs"
    
    log "=========================================="
    log "准备完成"
    log "=========================================="
    
    # 把空格分隔的目录字符串转换成 JSON 数组
    local json_dirs="["
    local first=true
    for dir in $random_dirs; do
        if [ "$first" = true ]; then
            first=false
        else
            json_dirs="$json_dirs,"
        fi
        json_dirs="$json_dirs\"$dir\""
    done
    json_dirs="$json_dirs]"
    
    # 输出 JSON
    echo "{"
    echo "  \"preparer\": \"code-debt-cleaner-prepare\","
    echo "  \"timestamp\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\","
    echo "  \"all_selected_types\": \"$all_selected\","
    echo "  \"tool_types\": \"$tool_types\","
    echo "  \"llm_types\": \"$llm_types\","
    echo "  \"directories\": $json_dirs"
    echo "}"
}

main "$@"
