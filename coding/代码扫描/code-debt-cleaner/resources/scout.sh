#!/bin/bash

# Code-Debt-Cleaner Scout - 扫描 Go 项目中的代码债务
# 输出格式：JSON
# 日志输出到 stdout（用 [LOG] 标记），JSON 结果也输出到 stdout
# 基于 golangci-lint 的统一静态分析
# ⚠️ 重要：此脚本只扫描，不修改任何文件！

set -e

# 日志函数（输出到 stdout，用 [LOG] 标记）
log() {
    echo "[LOG] [scout] $1"
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 设置 PATH，优先从 GOPATH/bin 查找工具
if command -v go &> /dev/null; then
    export PATH="$(go env GOPATH)/bin:$PATH"
fi

# 默认参数
SCAN_DIR="./"
SCAN_TYPES=""
LIMIT=""
SCAN_FILES=""
MAX_ISSUES=20

# 所有支持的类型（共 35 种：21 种代码坏味道 + 14 种工具可检测）
SUPPORTED_TYPES="long_method,large_class,long_parameter_list,divergent_change,shotgun_surgery,feature_envy,data_clumps,primitive_obsession,switch_statements,parallel_inheritance_hierarchies,lazy_class,speculative_generality,temporary_field,message_chains,middle_man,inappropriate_intimacy,alternative_classes_with_different_interfaces,incomplete_library_class,data_class,refused_bequest,comments,duplicated_code,staticcheck,gosimple,stylecheck,gofmt,goimports,gocyclo,errcheck,ineffassign,unused,revive,cyclop,govet,typecheck"

# 按风险分类的类型（仅用于随机化的是真正的 golangci-lint linters）
LOW_RISK_LINTERS="gofmt,goimports,gosimple,stylecheck,typecheck"
MEDIUM_RISK_LINTERS="ineffassign,unused,revive,duplicated_code"
HIGH_RISK_LINTERS="staticcheck,gocyclo,errcheck,cyclop,govet"

# 完整的所有支持的类型列表
ALL_LOW_RISK="long_method,large_class,long_parameter_list,data_clumps,primitive_obsession,lazy_class,temporary_field,middle_man,data_class,comments,gofmt,goimports,gosimple,stylecheck,typecheck"
ALL_MEDIUM_RISK="ineffassign,unused,revive,duplicated_code"
ALL_HIGH_RISK="divergent_change,shotgun_surgery,feature_envy,switch_statements,parallel_inheritance_hierarchies,speculative_generality,message_chains,inappropriate_intimacy,alternative_classes_with_different_interfaces,incomplete_library_class,refused_bequest,staticcheck,gocyclo,errcheck,cyclop,govet"

# 帮助信息
show_help() {
    echo "Code-Debt-Cleaner Scout - 扫描 Go 项目中的代码债务"
    echo ""
    echo "用法: bash scout.sh [选项]"
    echo ""
    echo "选项:"
    echo "  -d, --dir <目录>        指定扫描目录 (默认: ./)"
    echo "  -t, --types <类型>        指定要扫描的问题类型，多个类型用逗号分隔"
    echo "  -l, --limit <数量>        限制扫描结果数量"
    echo "  -f, --files <文件>        只扫描指定文件，多个文件用逗号分隔"
    echo "  -h, --help                显示帮助信息"
    echo ""
    echo "示例:"
    echo "  bash scout.sh"
    echo "  bash scout.sh -d ./pkg"
    echo "  bash scout.sh -t staticcheck,errcheck"
    echo "  bash scout.sh -l 10"
    echo "  bash scout.sh -f file1.go,file2.go"
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
            -t|--types)
                SCAN_TYPES="$2"
                shift 2
                ;;
            -l|--limit)
                LIMIT="$2"
                shift 2
                ;;
            -f|--files)
                SCAN_FILES="$2"
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

# 检查类型是否应该扫描
should_scan_type() {
    local type="$1"
    if [ -z "$SCAN_TYPES" ]; then
        return 0
    fi
    if echo "$SCAN_TYPES" | grep -q "$type"; then
        return 0
    fi
    return 1
}

# 检查并处理代码坏味道类型（需要人工识别）
handle_code_smell_types() {
    local code_smell_types="long_method,large_class,long_parameter_list,divergent_change,shotgun_surgery,feature_envy,data_clumps,primitive_obsession,switch_statements,parallel_inheritance_hierarchies,lazy_class,speculative_generality,temporary_field,message_chains,middle_man,inappropriate_intimacy,alternative_classes_with_different_interfaces,incomplete_library_class,data_class,refused_bequest,comments"
    
    if [ -z "$SCAN_TYPES" ]; then
        return
    fi
    
    local found_smell_types=""
    IFS=',' read -ra types <<< "$SCAN_TYPES"
    for type in "${types[@]}"; do
        if echo "$code_smell_types" | grep -q ",$type," || echo "$code_smell_types" | grep -q "^$type," || echo "$code_smell_types" | grep -q ",$type$" || [ "$code_smell_types" = "$type" ]; then
            if [ -z "$found_smell_types" ]; then
                found_smell_types="$type"
            else
                found_smell_types="$found_smell_types,$type"
            fi
        fi
    done
    
    if [ -n "$found_smell_types" ]; then
        log "⚠️  注意：以下代码坏味道类型需要人工识别，无法自动扫描："
        log "    $found_smell_types"
        log "    这些类型需要通过代码审查来识别，请手动检查代码或使用 IDE 进行分析"
    fi
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

# 扫描使用 golangci-lint
# 结果格式：每行 "type|/path/to/file.go"
scan_golangci_lint() {
    local output_file="$1"
    
    # 检查是否有类型需要扫描
    if [ -z "$SCAN_TYPES" ]; then
        log "扫描所有类型（使用 golangci-lint）..."
    else
        log "扫描指定类型（使用 golangci-lint）: $SCAN_TYPES"
    fi
    
    # 检查 golangci-lint 是否可用
    if ! command -v golangci-lint &> /dev/null; then
        log "错误：golangci-lint 未安装"
        log "提示：运行 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest' 安装"
        return 1
    fi
    
    # 从 SCAN_TYPES 中提取需要启用的 golangci-lint linters（duplicated_code 用 dupl 工具单独处理）
    local linters=""
    if [ -n "$SCAN_TYPES" ]; then
        IFS=',' read -ra types <<< "$SCAN_TYPES"
        for type in "${types[@]}"; do
            if [ "$type" != "duplicated_code" ]; then
                if [ -z "$linters" ]; then
                    linters="$type"
                else
                    linters="$linters,$type"
                fi
            fi
        done
    else
        # 如果没有指定类型，启用所有 golangci-lint linters
        linters="staticcheck,gosimple,stylecheck,gofmt,goimports,gocyclo,errcheck,ineffassign,unused,revive,cyclop,govet"
    fi
    
    # 如果最终没有 linters 了（只指定了 duplicated_code），直接返回
    if [ -z "$linters" ]; then
        log "  没有需要 golangci-lint 扫描的类型"
        return 0
    fi
    
    log "  启用的 linters: $linters"
    
    # 解析多个目录
    local dirs=()
    IFS=' ' read -ra dirs <<< "$SCAN_DIR"
    
    local seen_files=()
    local total_count=0
    
    # 遍历每个目录进行扫描
    for dir in "${dirs[@]}"; do
        if [ -n "$dir" ]; then
            log "  扫描目录: $dir/..."
            
            # 运行 golangci-lint 并获取 JSON 输出
            log "  正在运行 golangci-lint 扫描..."
            log "  提示：可以用 Ctrl+C 中断扫描"
            log "  （使用 golangci-lint 内置缓存加速重复扫描）"
            
            local temp_file=$(mktemp)
            local temp_file2=$(mktemp)
            local temp_file3=$(mktemp)
            
            # 运行扫描（golangci-lint 默认已启用缓存）
            if [ -n "$SCAN_FILES" ]; then
                IFS=',' read -ra files <<< "$SCAN_FILES"
                golangci-lint run --no-config --disable-all --enable="$linters" --out-format=json --timeout=2m --concurrency=4 "${files[@]}" 2>"$temp_file" > "$temp_file2" || true
            else
                golangci-lint run --no-config --disable-all --enable="$linters" --out-format=json --timeout=2m --concurrency=4 "$dir"/... 2>"$temp_file" > "$temp_file2" || true
            fi
            
            # 显示扫描过程中的日志（如果有）
            if [ -s "$temp_file" ]; then
                log "  扫描日志:"
                while IFS= read -r line; do
                    if [ -n "$line" ]; then
                        log "    $line"
                    fi
                done < "$temp_file"
            fi
            
            log "  golangci-lint 执行完成，正在解析结果..."
            
            # 使用 Python 解析 JSON，提取 linter 和 filename
            local count=0
            if command -v python3 &> /dev/null; then
                # 创建一个临时的 Python 脚本文件，避免 heredoc 问题
                local py_script=$(mktemp)
                cat > "$py_script" << 'PYTHON'
import json
import sys

try:
    data = json.load(sys.stdin)
    issues = data.get('Issues', [])
    seen = {}
    # 按优先级排序的 linters（优先选择容易修复的）
    priority_order = ['gofmt', 'goimports', 'stylecheck', 'gosimple', 'ineffassign', 'unused', 'revive', 'typecheck', 'gocyclo', 'cyclop', 'errcheck', 'govet', 'staticcheck']
    
    # 按优先级排序问题
    def get_priority(linter):
        try:
            return priority_order.index(linter)
        except ValueError:
            return len(priority_order)
    
    sorted_issues = sorted(issues, key=lambda x: get_priority(x.get('FromLinter', '')))
    
    for issue in sorted_issues:
        from_linter = issue.get('FromLinter', '')
        pos = issue.get('Pos', {})
        filename = pos.get('Filename', '')
        line = pos.get('Line', 0)
        if filename and from_linter:
            # 更宽松的去重：同一个文件的同一个 linter 最多记录 3 次
            key = f"{filename}|{from_linter}"
            count = seen.get(key, 0)
            if count < 3:
                seen[key] = count + 1
                print(f"{from_linter}\t{filename}\t{line}")
except Exception as e:
    pass
PYTHON
                # 将 JSON 传递给 Python
                python3 "$py_script" < "$temp_file2" > "$temp_file3"
                rm -f "$py_script"
                
                # 处理解析结果
                while IFS=$'\t' read -r linter file line; do
                    if [ -n "$file" ] && [ -f "$file" ]; then
                        # 检查该类型是否需要扫描
                        if ! should_scan_type "$linter"; then
                            continue
                        fi
                        
                        # 过滤掉依赖包的路径
                        if [[ "$file" == ../* ]] || [[ "$file" == /* ]] && ! [[ "$file" == "$(pwd)"* ]]; then
                            continue
                        fi
                        
                        # 如果指定了文件列表，检查是否在列表中
                        if [ -n "$SCAN_FILES" ] && ! echo "$SCAN_FILES" | grep -q "$file"; then
                            continue
                        fi
                        
                        # 检查是否已经处理过这个文件、类型和行号
                        local already_seen=false
                        local seen_key="$file|$linter|$line"
                        for seen in "${seen_files[@]}"; do
                            if [ "$seen" = "$seen_key" ]; then
                                already_seen=true
                                break
                            fi
                        done
                        
                        if [ "$already_seen" = false ]; then
                            echo "$linter|$file" >> "$output_file"
                            if [ -n "$line" ] && [ "$line" != "0" ]; then
                                log "  发现 $linter: $file:$line"
                            else
                                log "  发现 $linter: $file"
                            fi
                            seen_files+=("$seen_key")
                            count=$((count + 1))
                        fi
                    fi
                done < "$temp_file3"
            else
                # 如果 Python 不可用，使用简化方式
                log "  Python 不可用，使用简化扫描方式"
                grep -o '"FromLinter":"[^"]*"' "$temp_file2" | sed 's/"FromLinter":"//;s/"$//' | sort -u > "$temp_file3"
                grep -o '"Filename":"[^"]*"' "$temp_file2" | sed 's/"Filename":"//;s/"$//' | sort -u > "$temp_file3.lst"
                
                while read -r file; do
                    if [ -n "$file" ] && [ -f "$file" ]; then
                        local issue_type="staticcheck"
                        
                        if ! should_scan_type "$issue_type"; then
                            continue
                        fi
                        
                        if [[ "$file" == ../* ]] || [[ "$file" == /* ]] && ! [[ "$file" == "$(pwd)"* ]]; then
                            continue
                        fi
                        
                        if [ -n "$SCAN_FILES" ] && ! echo "$SCAN_FILES" | grep -q "$file"; then
                            continue
                        fi
                        
                        # 检查是否已经处理过这个文件和类型
                        local already_seen=false
                        for seen in "${seen_files[@]}"; do
                            if [ "$seen" = "$file|$issue_type" ]; then
                                already_seen=true
                                break
                            fi
                        done
                        
                        if [ "$already_seen" = false ]; then
                            echo "$issue_type|$file" >> "$output_file"
                            log "  发现 $issue_type: $file"
                            seen_files+=("$file|$issue_type")
                            count=$((count + 1))
                        fi
                    fi
                done < "$temp_file3.lst"
            fi
            
            rm -f "$temp_file" "$temp_file2" "$temp_file3" "$temp_file3.lst" 2>/dev/null
            total_count=$((total_count + count))
        fi
    done
    
    log "  golangci-lint 扫描完成，找到 $total_count 个文件"
    return 0
}

# 扫描重复代码（使用 dupl 工具）
# 结果格式：每行 "duplicated_code|/path/to/file.go"
scan_duplicate_code() {
    local output_file="$1"
    
    # 检查是否需要扫描此类型
    if ! should_scan_type "duplicated_code"; then
        log "跳过 duplicated_code 扫描（根据参数）"
        return
    fi
    
    log "扫描重复代码..."
    
    # 检查 dupl 是否可用
    if ! command -v dupl &> /dev/null; then
        log "  dupl 未安装，跳过重复代码扫描"
        log "  提示：可以运行 'go install github.com/mibk/dupl@latest' 安装"
        return
    fi
    
    # 解析多个目录
    local dirs=()
    IFS=' ' read -ra dirs <<< "$SCAN_DIR"
    
    # 统计项目 Go 文件数量，如果超过 1000 个，提高阈值以提升性能
    local go_file_count=0
    for dir in "${dirs[@]}"; do
        if [ -n "$dir" ] && [ -d "$dir" ]; then
            local count=$(find "$dir" -name "*.go" -not -path "*/.git/*" -not -path "*/vendor/*" 2>/dev/null | wc -l | awk '{print $1}')
            go_file_count=$((go_file_count + count))
        fi
    done
    log "  项目 Go 文件数量: $go_file_count"
    
    # 根据文件数量动态调整阈值
    local threshold=100
    if [ "$go_file_count" -gt 1000 ]; then
        threshold=200
        log "  大项目检测，提高阈值到 $threshold tokens 以提升性能"
    elif [ "$go_file_count" -gt 5000 ]; then
        threshold=300
        log "  超大项目检测，提高阈值到 $threshold tokens 以提升性能"
    fi
    log "  重复代码阈值: $threshold tokens"
    
    # 构建需要排除的目录
    local exclude_dirs=()
    exclude_dirs+=("-not" "-path" "*/.git/*")
    exclude_dirs+=("-not" "-path" "*/vendor/*")
    exclude_dirs+=("-not" "-path" "*/testdata/*")
    exclude_dirs+=("-not" "-path" "*/mocks/*")
    exclude_dirs+=("-not" "-path" "*/opp/*")
    
    # 收集所有需要扫描的 Go 文件
    local target_files=$(mktemp)
    for dir in "${dirs[@]}"; do
        if [ -n "$dir" ] && [ -d "$dir" ]; then
            find "$dir" -name "*.go" "${exclude_dirs[@]}" 2>/dev/null >> "$target_files"
        fi
    done
    
    local file_count=$(wc -l < "$target_files")
    log "  实际扫描文件数量: $file_count"
    
    if [ "$file_count" -eq 0 ]; then
        log "  没有找到需要扫描的文件"
        rm -f "$target_files"
        return
    fi
    
    log "  运行命令: dupl -t $threshold [files...]"
    
    local temp_file=$(mktemp)
    # dupl 输出格式示例：
    # path/to/file1.go:123-145: duplicate of path/to/file2.go:678-700
    # 使用 xargs 传递文件列表给 dupl，避免命令行过长问题
    xargs dupl -t "$threshold" < "$target_files" 2>"$temp_file" > "$temp_file.2" || true
    
    local count=0
    local seen_files=()
    
    while IFS= read -r line; do
        if [ -n "$line" ]; then
            # 提取第一个文件路径
            local file=$(echo "$line" | awk -F: '{print $1}')
            
            # 检查是否已经处理过这个文件
            local already_seen=false
            for seen in "${seen_files[@]}"; do
                if [ "$seen" = "$file" ]; then
                    already_seen=true
                    break
                fi
            done
            
            if [ "$already_seen" = true ]; then
                continue
            fi
            
            if [ -n "$file" ] && [ -f "$file" ]; then
                # 如果指定了文件列表，检查是否在列表中
                if [ -n "$SCAN_FILES" ] && ! echo "$SCAN_FILES" | grep -q "$file"; then
                    continue
                fi
                echo "duplicated_code|$file" >> "$output_file"
                log "  发现 duplicated_code: $file"
                seen_files+=("$file")
                count=$((count + 1))
            fi
        fi
    done < "$temp_file.2"
    
    rm -f "$temp_file" "$temp_file.2" "$target_files"
    log "  重复代码扫描完成，找到 $count 个文件"
}

# 根据类型获取描述
get_description() {
    local type="$1"
    case "$type" in
        duplicated_code)
            echo "Duplicate code: duplicate code fragments detected"
            ;;
        long_method)
            echo "Long method: function does too many things"
            ;;
        large_class)
            echo "Large class: class has too many responsibilities"
            ;;
        long_parameter_list)
            echo "Long parameter list: function has too many parameters"
            ;;
        divergent_change)
            echo "Divergent change: class is changed for many different reasons"
            ;;
        shotgun_surgery)
            echo "Shotgun surgery: one change requires modifying many places"
            ;;
        feature_envy)
            echo "Feature envy: method seems more interested in another class"
            ;;
        data_clumps)
            echo "Data clumps: data items that always go together"
            ;;
        primitive_obsession)
            echo "Primitive obsession: using primitives instead of objects"
            ;;
        switch_statements)
            echo "Switch statements: overuse of switch or if-else chains"
            ;;
        parallel_inheritance_hierarchies)
            echo "Parallel inheritance hierarchies: adding a subclass forces adding another"
            ;;
        lazy_class)
            echo "Lazy class: class does too little"
            ;;
        speculative_generality)
            echo "Speculative generality: building for a future that never comes"
            ;;
        temporary_field)
            echo "Temporary field: field only used in certain circumstances"
            ;;
        message_chains)
            echo "Message chains: long chains of method calls"
            ;;
        middle_man)
            echo "Middle man: class delegates everything"
            ;;
        inappropriate_intimacy)
            echo "Inappropriate intimacy: classes know too much about each other"
            ;;
        alternative_classes_with_different_interfaces)
            echo "Alternative classes: similar functionality but different interfaces"
            ;;
        incomplete_library_class)
            echo "Incomplete library class: library class needs patching"
            ;;
        data_class)
            echo "Data class: class has only data, no behavior"
            ;;
        refused_bequest)
            echo "Refused bequest: subclass doesn't use parent's methods"
            ;;
        comments)
            echo "Comments: too many comments indicate code is hard to understand"
            ;;
        staticcheck)
            echo "Static check: finds bugs and performance issues"
            ;;
        gosimple)
            echo "Simplifies code: suggests more idiomatic Go constructs"
            ;;
        stylecheck)
            echo "Style check: enforces Go style guidelines"
            ;;
        gofmt)
            echo "Code formatting: file needs gofmt"
            ;;
        goimports)
            echo "Import formatting: file needs goimports"
            ;;
        gocyclo)
            echo "Cyclomatic complexity: function has high complexity"
            ;;
        errcheck)
            echo "Error check: unchecked errors detected"
            ;;
        ineffassign)
            echo "Ineffective assignment: unused assignment"
            ;;
        unused)
            echo "Unused code: unused constants, variables, functions or types"
            ;;
        revive)
            echo "Revive: fast, configurable, extensible Go linter"
            ;;
        cyclop)
            echo "Cyclop: checks function cyclomatic complexity"
            ;;
        govet)
            echo "Go vet: suspicious constructs detected"
            ;;
        typecheck)
            echo "Type check: type checking issues"
            ;;
        *)
            echo "Unknown issue type"
            ;;
    esac
}

# 根据类型获取风险等级
get_risk() {
    local type="$1"
    if echo ",$ALL_LOW_RISK," | grep -q ",$type,"; then
        echo "low"
    elif echo ",$ALL_MEDIUM_RISK," | grep -q ",$type,"; then
        echo "medium"
    elif echo ",$ALL_HIGH_RISK," | grep -q ",$type,"; then
        echo "high"
    else
        echo "low"
    fi
}

# 主函数
main() {
    # 解析命令行参数
    parse_args "$@"
    
    log "=========================================="
    log "Code-Debt-Cleaner Scout 开始运行"
    log "⚠️  此脚本只扫描，不修改任何文件"
    log "扫描目录: $SCAN_DIR"
    if [ -n "$SCAN_TYPES" ]; then
        log "扫描类型: $SCAN_TYPES"
    fi
    if [ -n "$SCAN_FILES" ]; then
        log "扫描文件: $SCAN_FILES"
    fi
    if [ -n "$LIMIT" ]; then
        log "结果限制: $LIMIT"
    fi
    log "=========================================="
    
    handle_code_smell_types
    check_go_project
    
    # 创建临时文件存储扫描结果
    # 格式: type|filepath
    local result_file=$(mktemp)
    
    # 使用 golangci-lint 进行主要扫描
    if ! scan_golangci_lint "$result_file"; then
        log "错误：golangci-lint 扫描失败"
        rm -f "$result_file"
        exit 1
    fi
    
    # duplicated_code 扫描（独立处理，因为它使用专门的工具）
    # 只有在用户明确指定时才扫描 duplicated_code（因为它很慢）
    if [ -n "$SCAN_TYPES" ] && ! echo "$SCAN_TYPES" | grep -q "duplicated_code"; then
        log "跳过 duplicated_code 扫描（如需扫描请明确指定 -t duplicated_code）"
    else
        scan_duplicate_code "$result_file"
    fi
    
    # 去重
    sort -u "$result_file" -o "$result_file"
    
    # 读取所有问题
    local issues=()
    while IFS= read -r line; do
        if [ -n "$line" ]; then
            issues+=("$line")
        fi
    done < "$result_file"
    
    rm -f "$result_file"
    
    # 应用最多 $MAX_ISSUES 个问题的限制
    if [ ${#issues[@]} -gt "$MAX_ISSUES" ]; then
        log "应用限制，取前 $MAX_ISSUES 个问题"
        issues=("${issues[@]:0:$MAX_ISSUES}")
    fi
    
    # 应用用户自定义限制（如果设置了）
    if [ -n "$LIMIT" ] && [ ${#issues[@]} -gt "$LIMIT" ]; then
        log "应用用户结果限制，取前 $LIMIT 个问题"
        issues=("${issues[@]:0:$LIMIT}")
    fi
    
    log "=========================================="
    log "扫描完成，共找到 ${#issues[@]} 个问题"
    log "=========================================="
    
    # 输出 JSON
    echo "{"
    echo "  \"scanner\": \"code-debt-cleaner-scout\","
    echo "  \"timestamp\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\","
    echo "  \"issue_count\": ${#issues[@]},"
    echo "  \"issues\": ["
    
    local first=true
    for issue in "${issues[@]}"; do
        IFS='|' read -r type file <<< "$issue"
        
        if [ "$first" = true ]; then
            first=false
        else
            echo ","
        fi
        
        local description=$(get_description "$type")
        local risk=$(get_risk "$type")
        
        echo "    {"
        echo "      \"type\": \"$type\","
        echo "      \"file\": \"$file\","
        echo "      \"risk\": \"$risk\","
        echo "      \"description\": \"$description\""
        echo -n "    }"
    done
    
    echo ""
    echo "  ]"
    echo "}"
}

main "$@"
