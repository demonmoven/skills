# 路径自动发现算法

> 主 context 在 wizard 预热阶段 Read 本文件，执行下面的 bash 脚本收集所有路径候选。

## 通用约定

- `CWD` = `$PWD`（用户调用 skill 时的当前目录）
- `PARENT` = `$(dirname "$CWD")`
- `GRANDPARENT` = `$(dirname "$PARENT")`
- `REPO_BASENAME` = `$(basename "$CWD")`

## 通用工具：拿一个路径的当前 git 分支

```bash
# 用法: get_branch /path/to/repo
get_branch() {
  local repo="$1"
  if [ -d "$repo/.git" ] || git -C "$repo" rev-parse --git-dir >/dev/null 2>&1; then
    git -C "$repo" branch --show-current 2>/dev/null
  fi
}
```

> 这个 helper 用于给候选标签附加 `(当前分支: xxx)` 后缀。

## 通用约束：候选去重 + 排序

- 对每类候选,**去重**（同一路径只出现一次）
- 排序优先级：
  1. cwd 内（最高）
  2. cwd 父目录
  3. cwd 祖父目录
- 候选数量上限 **8 个**,超出截断
- 最后追加一个 "手动输入" 选项

## BUSINESS_REPO 发现

```bash
discover_business_repo() {
  # L0: cwd 自身（如果是 git 仓库）
  if [ -d "$CWD/.git" ]; then
    echo "$CWD"
  fi

  # L1: 父目录的兄弟 git 仓库
  find "$PARENT" -mindepth 2 -maxdepth 2 -name ".git" -type d 2>/dev/null \
    | sed 's|/.git$||' \
    | grep -v "^${CWD}$" \
    | head -5
}
```

## BRANCH 发现

```bash
discover_branches() {
  local repo="$1"
  # 最近活跃的 10 个分支（按 commit 时间）
  git -C "$repo" branch --sort=-committerdate 2>/dev/null \
    | sed 's/^[* ] *//' \
    | grep -v '^$' \
    | head -10
}
```

## TEST_REPO 发现

```bash
discover_test_repos() {
  # L0: cwd 自身（同仓测试）
  echo "$CWD"

  # L1a: 标准命名兄弟目录
  find "$PARENT" -mindepth 1 -maxdepth 1 -type d \
    \( -name "${REPO_BASENAME}_test" \
       -o -name "${REPO_BASENAME}_api_test" \
       -o -name "${REPO_BASENAME}-test" \
       -o -name "${REPO_BASENAME}-api-test" \) \
    2>/dev/null

  # L1b: 模糊匹配兄弟目录（任意 *test* 名字）
  find "$PARENT" -mindepth 1 -maxdepth 1 -type d \
    \( -name "*_test" -o -name "*_api_test" -o -name "*-test" \) \
    2>/dev/null \
    | grep -v "^${CWD}$"
}
```

## SPEC_DIR 发现（最重要的 fallback 链）

```bash
discover_spec_dirs() {
  # L0: cwd 内的常见 spec 目录名
  for name in specs spec openspec exec-plan features; do
    if [ -d "$CWD/$name" ]; then
      echo "$CWD/$name"
    fi
  done

  # L0.5: cwd 内 docs 目录（低优先级,可能不是 spec）
  if [ -d "$CWD/docs" ]; then
    echo "$CWD/docs"
  fi

  # L0.9: cwd 内更深层的 specs/ 目录（最多 3 层）
  find "$CWD" -mindepth 2 -maxdepth 3 -type d \
    \( -name specs -o -name openspec -o -name "exec-plan" \) \
    2>/dev/null \
    -not -path "*/node_modules/*" \
    -not -path "*/.git/*" \
    -not -path "*/dist/*" \
    -not -path "*/build/*" \
    | head -5

  # L1: 父目录里的兄弟目录（命名包含 spec）
  find "$PARENT" -mindepth 1 -maxdepth 1 -type d \
    \( -name "*-specs" -o -name "*_specs" -o -name "*-spec" -o -name "specs" \) \
    2>/dev/null \
    | grep -v "^${CWD}$" \
    | head -3

  # L2: 祖父目录里的相关目录（最深兜底）
  find "$GRANDPARENT" -mindepth 2 -maxdepth 3 -type d \
    \( -name specs -o -name openspec \) \
    2>/dev/null \
    | head -3
}
```

## IDL_REPO 发现

```bash
discover_idl_repos() {
  # L0: cwd 子目录 idl/ idls/
  for name in idl idls IDL; do
    if [ -d "$CWD/$name" ]; then
      echo "$CWD/$name"
    fi
  done

  # L0.5: git submodule 中含 idl
  if [ -f "$CWD/.gitmodules" ]; then
    grep -A1 'submodule.*idl' "$CWD/.gitmodules" 2>/dev/null \
      | grep 'path' \
      | awk -F= '{print $2}' \
      | tr -d ' \t' \
      | sed "s|^|$CWD/|"
  fi

  # L1: 父目录的兄弟 *-idl / *_idl
  find "$PARENT" -mindepth 1 -maxdepth 1 -type d \
    \( -name "${REPO_BASENAME}-idl" \
       -o -name "${REPO_BASENAME}_idl" \
       -o -name "*-idl" \
       -o -name "*_idl" \
       -o -name "idl" \) \
    2>/dev/null \
    | grep -v "^${CWD}$"

  # L2: 祖父目录里的 idl/
  find "$GRANDPARENT" -mindepth 2 -maxdepth 3 -type d -name "idl" 2>/dev/null \
    | head -3
}
```

## 性能优化

所有 `find` 命令都加上 `-not -path "*/node_modules/*"` 之类的 prune，避免在大型 monorepo 上扫描得太慢：

```bash
FIND_PRUNE='-not -path "*/node_modules/*" -not -path "*/.git/*" -not -path "*/dist/*" -not -path "*/build/*" -not -path "*/vendor/*" -not -path "*/target/*"'
```

## 与 candidate-generation.md 的衔接

发现完所有候选路径后，主 context Read `candidate-generation.md`，按其中规范把每个路径包装成带分支信息的 option 标签，例如：

```
/Users/x/cozeloop_backend  (当前目录, 当前分支: feat/my-feature)
/Users/x/cozeloop_api_test  (兄弟目录 *_test, 当前分支: main)
```

然后传给 `AskUserQuestion` 工具作为 options。
