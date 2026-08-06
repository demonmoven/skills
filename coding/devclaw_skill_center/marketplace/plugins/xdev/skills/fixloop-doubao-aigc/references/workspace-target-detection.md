# Workspace 目标嗅探算法

面向 multi-repo workspace 场景（例：`~/TraeProjects/doubao/` 下含 4 个 Go sub-repo）。目标是自动识别本次 fix-loop 要部署/修改的 sub-repo。

## 输入

- `$BUSINESS_REPO`：可能是单仓路径（有 `go.mod`）或 workspace 路径
- `$BRANCH`：业务分支
- 可选 `$IDL_REPO` + `$IDL_BRANCH`：IDL 仓库及变更分支
- 可选 `$SPEC_DIR`：需求文档

## 输出

一组三元组 `(sub_repo_path, psm, binary_name)`，存储到 `$OUTPUT_DIR/detected_targets.jsonl`，每行一个：

```json
{"sub_repo_path": "/Users/x/TraeProjects/doubao/creativity", "psm": "flow.alice.creativity", "binary_name": "flow.alice.creativity", "source_signal": "git"}
```

`source_signal` 取值：`git` / `idl` / `spec` / `explicit`。

## 算法

### Step 1 判仓型

```
if [ -f "$BUSINESS_REPO/go.mod" ]; then
  # 单仓模式
  binary_name = 从 $BUSINESS_REPO/Makefile 中解析（多数服务 Makefile 有 `binary=` 或 `BINARY=` 等变量；
                如果找不到，退化为目录名）
  psm = $PSM (必须用户传入，或从 $BUSINESS_REPO/conf/app.yaml / conf/*.toml 的 PSM 字段推断)
  记录 source_signal = "explicit"
  退出
else
  # workspace 模式：枚举所有含 go.mod 的直接子目录
  sub_repos = $(for d in "$BUSINESS_REPO"/*/; do [ -f "$d/go.mod" ] && echo "$d"; done)
  if [ -z "$sub_repos" ]; then
    ERROR "BUSINESS_REPO 既非单 Go 仓也非 multi-Go workspace"
  fi
```

### Step 2 解析 PSM 映射表

读 `$BUSINESS_REPO/CLAUDE.md`，查找包含 `PSM` 列的 markdown 表格。典型形如：

```markdown
| Repo | Module path | PSM | Binary name | Role |
|------|-------------|-----|-------------|------|
| `alice_creativity_api` | ... | — | `flow.alice.creativity_api` | ... |
| `creativity` | ... | `flow.alice.creativity` | `flow.alice.creativity` | ... |
```

建立双向字典 `psm_map`（PSM ↔ sub_repo 目录名）。容错：
- 单元格值为 `—` / `-` 视为无 PSM，跳过
- 表格格式错乱：跳过解析，`psm_map` 留空（后续信号仍可工作）

### Step 3 三路信号嗅探

```
candidates = []  # 保序数组，去重

# P1 · git 嗅探（硬信号）
for repo in sub_repos:
  cd $repo
  # 条件 1：有 uncommitted 改动
  if ! git diff --quiet OR ! git diff --cached --quiet:
    candidates.append((repo, "git"))
    continue
  # 条件 2：当前分支（预期 = $BRANCH）超前 origin/$BRANCH
  current_branch = git branch --show-current
  if [ "$current_branch" = "$BRANCH" ]:
    ahead = git rev-list --count "origin/$BRANCH..HEAD" 2>/dev/null || echo 0
    if [ "$ahead" -gt 0 ]:
      candidates.append((repo, "git"))

# P2 · IDL 嗅探（硬信号）
if [ -n "$IDL_REPO" ] AND psm_map 非空:
  cd $IDL_REPO_PATH
  git fetch origin
  changed_idls = git diff --name-only "origin/main...origin/$IDL_BRANCH" 2>/dev/null
                 或 git diff --name-only "origin/master...origin/$IDL_BRANCH"
  for idl in $changed_idls:
    # 从 IDL 路径推 PSM。常见几种路径模式：
    # - thrift/flow.alice.creativity/foo.thrift  → 取第 2 段作为 PSM
    # - services/creativity/foo.thrift            → 取第 2 段作为 sub_repo 名，查 psm_map
    # - flow/alice/creativity/foo.thrift          → 取前 3 段拼 PSM
    # - flow.agent.creation.thrift                → 文件 basename 去后缀即 PSM
    psm = try_parse_psm_from_path($idl)
    if psm in psm_map:
      repo = $BUSINESS_REPO/psm_map[psm]
      if repo in sub_repos AND repo 未在 candidates:
        candidates.append((repo, "idl"))

# P3 · SPEC 嗅探（软信号，仅兜底）
if len(candidates) == 0 AND [ -d "$SPEC_DIR" ]:
  启动 subagent（general-purpose），prompt：
    "读以下文件并回答："
    "1. $SPEC_DIR 下所有 .md 文件（递归）"
    "2. $BUSINESS_REPO/CLAUDE.md 里的 PSM 映射表"
    "任务：判断 SPEC 描述的功能最可能涉及哪些 PSM。输出 JSON 数组。"
    "约束：只能从 psm_map 里给出的 PSM 中选；若不确定选保守的子集；
     无把握时返回空数组。"
  接收返回的 PSM 列表，转 sub_repo 加入 candidates（source_signal="spec"）
```

### Step 4 收敛

```
if candidates 为空:
  ERROR: "三路信号全空。请显式传 BUSINESS_REPO=<单仓路径>（含 go.mod），
          或手工 checkout 到 $BRANCH 并在目标 sub-repo 里做出改动后重试。"

# 打印嗅探轨迹
echo "=== Workspace 目标嗅探结果 ==="
for (repo, signal) in candidates:
  psm = psm_map_reverse.get(repo, "(未知)")
  binary = resolve_binary_name(repo)
  echo "  → $(basename $repo)  PSM=$psm  binary=$binary  信号=$signal"

# 写入 detected_targets.jsonl
for (repo, signal) in candidates:
  echo '{"sub_repo_path":"'$repo'","psm":"...","binary_name":"...","source_signal":"'$signal'"}' >> $OUTPUT_DIR/detected_targets.jsonl
```

## binary_name 推断

```
resolve_binary_name(repo):
  1. 读 $repo/Makefile，找 `binary = XXX` 或 `BINARY=XXX`，返回
  2. 读 $repo/build.sh，找 `output/bin/XXX` 或 `go build -o XXX`，返回
  3. 退化：返回 $(basename $repo)
```

## 与原 fixloop 的差异

原 fixloop 假设 `BUSINESS_REPO` 始终是单仓，所以 `PSM` 是必填单值。本 skill 在 workspace 模式下：
- `PSM` 降为条件必填（嗅探自动填）
- 候选可能 ≥ 2 个，Stage 1 部署要串行处理
- Stage 4 修复跨多个 sub-repo commit/push
