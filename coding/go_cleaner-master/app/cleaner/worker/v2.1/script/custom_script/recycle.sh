#!/bin/bash

function deleteCodeBlocks() {
    local folder="$1"
    for file in "$folder"/*.go; do
        if [ -f "$file" ]; then
            if [[ $file == *"_test.go" ]]; then
                continue
            fi
            inBlock=false
            hasBlock=false
            temp_file="${file}.temp"
            while IFS= read -r line; do
                if ! echo "$line" | grep -q "code_recycler.Begin" && ! echo "$line" | grep -q "code_recycler.End"; then
                    if [ "$inBlock" = false ]; then
                        echo "$line"
                    fi
                else
                    if echo "$line" | grep -q "code_recycler.Begin"; then
                        inBlock=true
                        hasBlock=true
                    elif echo "$line" | grep -q "code_recycler.End"; then
                        inBlock=false
                        echo "// code has been recycled here"
                    fi
                fi
            done < "$file" > "$temp_file"
            mv "$temp_file" "$file"
            if [ "$hasBlock" = true ];then
              bash -c "goimports -w $file"
            fi
        fi
    done
    for subfolder in "$folder"/*/; do
        if [ -d "$subfolder" ]; then
            deleteCodeBlocks "$subfolder"
        fi
    done
}

if [ $# -ne 1 ]; then
    echo "Usage: $0 <folder_path>"
else
    folder_path="$1"
    deleteCodeBlocks "$folder_path"
fi

# 执行前需安装依赖：go install golang.org/x/tools/cmd/goimports@latest