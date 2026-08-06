# 候选生成与标签格式规范

> 主 context 在 wizard 各 step 发出 question 之前 Read 本文件，按规范把原始候选数据包装成带说明的 option 标签。

## 数量上限

| 参数 | 候选上限 |
|------|--------|
| BUSINESS_REPO | 5 + "手动输入" |
| BRANCH | 8 + "手动输入" |
| TEST_REPO | 5 + "手动输入" |
| SPEC_DIR | 6 + "手动输入" |
| IDL_REPO | 5 + "手动输入" |
| IDL_BRANCH | 5 + "手动输入" |
| PSM | 5 + "调用 bytedcli 搜索" + "手动输入" |

超过上限的候选**截断**，按下面"排序优先级"取前 N 个。

## 排序优先级

候选项按以下顺序排列（每个参数的"推荐"项第一）：

1. **当前 cwd 内**（最近、最相关）
2. **cwd 父目录里的兄弟目录**
3. **cwd 祖父目录里的相关目录**
4. **git submodule / 配置文件 / .tcerc 等其它来源**
5. **手动输入**（永远最后）

## 候选标签格式

### 1. 路径类候选（含分支显示，**关键 UX**）

**所有仓库类候选（BUSINESS_REPO / TEST_REPO / IDL_REPO）必须显示当前 git 分支**：

格式：
```
{path}  ({location_hint}, 当前分支: {branch})
```

例：
```
/Users/x/cozeloop_backend         (当前目录, 当前分支: feat/my-feature)
/Users/x/cozeloop_api_test        (兄弟目录 *_test, 当前分支: main)
/Users/x/workspace/cozeloop-idl   (兄弟目录 *-idl, 当前分支: main)
/Users/x/cozeloop_backend/idl     (cwd 子目录, 当前分支: feat/my-feature)
```

如果路径**不是 git 仓库**或拿不到分支，省略 `, 当前分支: xxx` 部分：
```
/Users/x/some_dir   (cwd 子目录)
```

实现方式：用 `lib/path-discovery.md` 里的 `get_branch()` helper：

```bash
make_repo_label() {
  local path="$1"
  local hint="$2"
  local branch=$(get_branch "$path")
  if [ -n "$branch" ]; then
    echo "${path}  (${hint}, 当前分支: ${branch})"
  else
    echo "${path}  (${hint})"
  fi
}
```

### 2. 非仓库的路径类候选（SPEC_DIR）

SPEC_DIR 一般不是 git 仓库（它在业务仓库内部或外部都可以是普通目录），所以**不显示分支**：

```
/Users/x/cozeloop_backend/specs    (cwd 内 specs/, 推荐)
/Users/x/cozeloop_backend/openspec (cwd 内 openspec/)
/Users/x/cozeloop-specs            (兄弟目录)
```

### 3. 分支类候选（BRANCH / IDL_BRANCH）

```
feat/my-feature   (git 当前分支, 推荐)
main
master
release/2026-04
feat/another      (最近活跃)
手动输入
```

### 4. PSM 类候选

```
stone.cozeloop.prompt   (.tcerc 文件, 可信)        ← 最推荐
stone.cozeloop.prompt   (git remote 推断, 可能不准)
stone.cozeloop_backend  (git remote 推断, 可能不准)
调用 bytedcli 模糊搜索
手动输入
```

## 去重规则

- 路径候选：按 `realpath` 或 `cd <path> && pwd -P` 标准化后去重
- 分支候选：按字符串去重（保持原大小写）
- PSM 候选：按字符串去重（同一 PSM 只显示一次,保留最高优先级的 hint）

## 推荐项标记

每个 question 的**第一个 option** 应该是"最推荐"的候选，标签里显示 "(推荐)" 后缀：

```
✓ /Users/x/cozeloop_backend  (当前目录, 当前分支: feat/my-feature, 推荐)
○ /Users/x/another           (...)
○ 手动输入
```

`AskUserQuestion` 工具默认会把 options 列表的第一项作为默认选项，所以放在第一位即可。

## "手动输入" option

每个 question 都必须有 "手动输入" 作为最后一个 option。用户选了之后：

1. AskUserQuestion 返回选项值（例如 "手动输入"）
2. 主 context 在下一个 message 中向用户单独提问该参数：
   ```
   请输入 SPEC_DIR 的完整路径（可以用 @ 自动补全）：
   ```
3. 用户在自由文本里给路径（cc 输入框天然支持 `@<path>` 自动补全）
4. 主 context 验证路径合法（`test -d`），不合法则重新提问

## 手动输入的路径验证

```bash
validate_path() {
  local path="$1"
  local kind="$2"  # repo / dir / file
  
  if [ "$kind" = "repo" ]; then
    if [ ! -d "$path/.git" ]; then
      echo "ERROR: $path 不是 git 仓库（缺少 .git 目录）"
      return 1
    fi
  elif [ "$kind" = "dir" ]; then
    if [ ! -d "$path" ]; then
      echo "ERROR: $path 不存在或不是目录"
      return 1
    fi
  fi
  
  return 0
}
```

## 候选为空的兜底

如果某个参数的所有自动候选都为空（极端情况，用户在一个完全空白的目录里运行），option 列表只包含 "手动输入" 一项：

```
○ 手动输入
```

并附加提示：
```
未在当前工作区或父目录中找到任何 SPEC_DIR 候选,请手动输入完整路径。
```
