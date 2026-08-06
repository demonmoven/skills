# PSM 自动推断算法

> 主 context 在 wizard Step 3 之前 Read 本文件，按以下优先级生成 PSM 候选。

## 输入

- `CWD` = `$PWD`
- `REPO_BASENAME` = `$(basename "$CWD")`
- `GIT_REMOTE_URL` = `$(git -C "$CWD" remote get-url origin 2>/dev/null)`
- `BYTEDCLI_AVAILABLE` = `command -v bytedcli >/dev/null && echo true || echo false`

## 输出

`PSM_CANDIDATES`：一个候选 PSM 列表，按推荐度排序。每项含 PSM 字符串 + 来源说明。

## 推断顺序

### 1. `.tcerc` 文件（最高优先级）

```bash
infer_psm_from_tcerc() {
  if [ -f "$CWD/.tcerc" ]; then
    # .tcerc 可能是 JSON / YAML 格式
    if command -v jq >/dev/null; then
      jq -r '.psm // .PSM // empty' "$CWD/.tcerc" 2>/dev/null
    fi
    # 或者直接 grep
    grep -E '^\s*psm\s*[:=]' "$CWD/.tcerc" 2>/dev/null \
      | head -1 \
      | sed -E 's/^\s*psm\s*[:=]\s*["'\''"]?([^"'\''"]+)["'\''"]?.*/\1/'
  fi
}
```

### 2. `package.json` 字段

```bash
infer_psm_from_package_json() {
  if [ -f "$CWD/package.json" ] && command -v jq >/dev/null; then
    jq -r '.psm // .config.psm // empty' "$CWD/package.json" 2>/dev/null
  fi
}
```

### 3. `bytedcli.yaml` / `bytedcli.toml` / `tce.yaml` 等内部约定

```bash
infer_psm_from_byted_config() {
  for f in bytedcli.yaml bytedcli.toml tce.yaml deploy.yaml; do
    if [ -f "$CWD/$f" ]; then
      grep -E '^\s*psm\s*[:=]' "$CWD/$f" 2>/dev/null \
        | head -1 \
        | sed -E 's/^\s*psm\s*[:=]\s*["'\''"]?([^"'\''"]+)["'\''"]?.*/\1/'
    fi
  done
}
```

### 4. git remote URL 推断（启发式，准确率不高但作为基线候选）

```bash
infer_psm_from_git_remote() {
  local url="$1"
  # 输入示例: git@code.byted.org:stone/cozeloop_backend.git
  # 提取 group 和 repo
  
  # 去掉协议和主机
  local path=$(echo "$url" | sed -E 's|^[^:]+:||' | sed -E 's|^//[^/]+/||' | sed 's|\.git$||')
  # path 现在是 stone/cozeloop_backend
  
  local group=$(echo "$path" | cut -d'/' -f1)
  local repo=$(echo "$path" | cut -d'/' -f2-)
  
  # 候选 1: {group}.{repo}
  echo "${group}.${repo}"
  
  # 候选 2: {group}.{repo with _ → .}
  echo "${group}.${repo}" | tr '_' '.'
  
  # 候选 3: 假设 repo = <product>_<service>，PSM = group.product.service
  if [[ "$repo" == *_* ]]; then
    local product=$(echo "$repo" | cut -d'_' -f1)
    local service=$(echo "$repo" | cut -d'_' -f2-)
    echo "${group}.${product}.${service}"
  fi
  
  # 候选 4: 去掉 _backend / _server / _api 后缀 + 加 .backend
  for suffix in _backend _server _api _service; do
    if [[ "$repo" == *"$suffix" ]]; then
      local stripped="${repo%$suffix}"
      echo "${group}.${stripped}${suffix//_/.}"
    fi
  done
}
```

> **注意**：git remote 推断只是经验性的猜测,**命中率可能 < 50%**。所以这些候选都打"推断,可能不准"标签让用户选。

### 5. bytedcli `tce service search` 模糊匹配（用户主动触发）

```bash
search_psm_via_bytedcli() {
  if [ "$BYTEDCLI_AVAILABLE" = "true" ]; then
    bytedcli --json tce service search --keyword "$REPO_BASENAME" --page-size 5 2>/dev/null \
      | jq -r '.data[]? | .psm // empty' 2>/dev/null
  fi
}
```

> **注意**：这一步**仅在用户明确选择 "调用 bytedcli 模糊搜索" option 时才执行**，因为：
> - 需要 bytedcli 已认证（precheck 在后面才做）
> - 网络调用较慢
> - 不主动调用以加快 wizard 体验

## 整体推断流程

```bash
generate_psm_candidates() {
  PSM_CANDIDATES=()
  
  # 优先级 1: .tcerc
  local tcerc_psm=$(infer_psm_from_tcerc)
  if [ -n "$tcerc_psm" ]; then
    PSM_CANDIDATES+=("${tcerc_psm}|.tcerc 文件,可信")
  fi
  
  # 优先级 2: package.json
  local pkg_psm=$(infer_psm_from_package_json)
  if [ -n "$pkg_psm" ]; then
    PSM_CANDIDATES+=("${pkg_psm}|package.json,可信")
  fi
  
  # 优先级 3: byted 配置文件
  local cfg_psm=$(infer_psm_from_byted_config)
  if [ -n "$cfg_psm" ]; then
    PSM_CANDIDATES+=("${cfg_psm}|byted 配置文件")
  fi
  
  # 优先级 4: git remote 推断（最多 4 个候选,标记"推断"）
  if [ -n "$GIT_REMOTE_URL" ]; then
    while IFS= read -r p; do
      [ -n "$p" ] && PSM_CANDIDATES+=("${p}|git remote 推断,可能不准")
    done < <(infer_psm_from_git_remote "$GIT_REMOTE_URL")
  fi
  
  # 去重
  PSM_CANDIDATES=$(printf '%s\n' "${PSM_CANDIDATES[@]}" | awk '!seen[$0]++')
}
```

## 候选标签格式

每个 PSM 候选转换为 wizard option 标签：

```
- "stone.cozeloop.prompt  (.tcerc 文件, 可信)"     ← 推荐
- "stone.cozeloop.prompt  (git remote 推断, 可能不准)"
- "stone.cozeloop_backend  (git remote 推断, 可能不准)"
- "调用 bytedcli 模糊搜索"
- "手动输入"
```

如果 `.tcerc` / `package.json` / 配置文件都不存在，**git remote 推断的候选标记 "可能不准"**，并把 "调用 bytedcli 模糊搜索" 选项突出显示。

## 用户选择 "调用 bytedcli 模糊搜索" 后的流程

1. 主 context 检查 bytedcli 是否已认证（先执行 `bytedcli auth status`）
2. 如未认证，先报错提示用户先认证再重试
3. 已认证 → 执行 `bytedcli --json tce service search --keyword "$REPO_BASENAME" --page-size 5`
4. 解析 JSON 提取 PSM 列表
5. 把这些 PSM 作为新一轮 AskUserQuestion 的 options 让用户选

## 失败处理

- 所有候选都为空（连 git remote 推断都失败）→ 仅给出 "手动输入" 和 "调用 bytedcli 模糊搜索" 两个 option
- 用户手动输入但 PSM 格式明显错误 → 提示重新输入
