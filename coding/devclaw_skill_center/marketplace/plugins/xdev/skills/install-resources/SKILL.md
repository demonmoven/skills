---
name: install-resources
description: "安装项目所需的业务 Skills 和 Constitution 到本地。当用户提到「安装资源」「install resources」「安装 skills」「安装 constitution」时使用。"
---

# DevClaw 安装业务资源

将远程仓库 `biz-resources/` 中的 skills 和 constitution 安装到当前项目。

## 执行流程

### Step 0：选择 Coding Agent

使用 AskUserQuestion 询问用户选择哪个 Coding Agent：

```
请选择 Coding Agent：
1. claude
2. trae
```

将用户的选择记为 `{AGENT}`。根据选择确定安装目标目录：

| Agent | Skills 安装目录 |
|-------|---------------|
| claude | `<cwd>/.claude/skills/` |
| trae | `<cwd>/.trae/skills/` |

记为 `{SKILLS_DST}`。

### Step 1：选择分支

使用 AskUserQuestion 询问用户安装资源的分支：

```
请输入资源分支（默认 main）：
```

如果用户未输入或输入为空，使用 `main`。记为 `{BRANCH}`。

### Step 2：列出可用资源包

通过以下命令获取远程仓库中 `biz-resources/` 下的子目录列表：

```bash
git ls-tree --name-only "origin/{BRANCH}" biz-resources/ 2>/dev/null | sed 's|biz-resources/||'
```

> **注意**：上面假设本地已有 fetch。如果失败，先执行：
> ```bash
> git fetch origin {BRANCH} --depth=1
> ```
> 如果当前目录不是 `devclaw_skills_center` 仓库，则需要用临时目录方式：
> ```bash
> TMP_DIR=$(mktemp -d)
> git clone --depth=1 --branch {BRANCH} --filter=blob:none --sparse git@code.byted.org:stone/devclaw_skills_center.git "$TMP_DIR"
> cd "$TMP_DIR" && git sparse-checkout set biz-resources/
> ls biz-resources/
> ```

根据查询结果分两种情况处理：

#### 情况 A：有可用资源包（>= 1 个）

**即使只有 1 个资源包，也必须列出让用户确认选择。** 使用 AskUserQuestion 展示编号列表：

```
当前分支 {BRANCH} 下有以下可用资源包：
1. coze-loop
2. xxx
...

请选择要安装的资源包（输入编号）：
```

将用户的选择记为 `{RESOURCE_PACK}`（如 `coze-loop`），然后继续 Step 3。

#### 情况 B：没有可用资源包（0 个）

使用 AskUserQuestion 提示用户：

```
当前分支 {BRANCH} 的 biz-resources/ 下没有可用的资源包。

你可以手动将资源安装到以下位置：
- Skills → {SKILLS_DST}
- Constitution → <cwd>/constitution/

完成后请回复"继续"以结束流程。
```

等待用户回复后，**跳过 Step 3**，直接进入 Step 4 输出完成提示（说明本次为手动安装模式）。

### Step 3：获取资源并安装

#### 3.1 获取资源文件

如果当前目录就是 `devclaw_skills_center` 仓库，直接从本地仓库读取：

```bash
git show "origin/{BRANCH}:biz-resources/{RESOURCE_PACK}" 2>/dev/null
```

否则，使用 sparse clone 到临时目录（如 Step 2 中已 clone，则复用）：

```bash
TMP_DIR=$(mktemp -d)
git clone --depth=1 --branch {BRANCH} --filter=blob:none --sparse \
  git@code.byted.org:stone/devclaw_skills_center.git "$TMP_DIR"
cd "$TMP_DIR" && git sparse-checkout set "biz-resources/{RESOURCE_PACK}"
```

记资源包根路径为 `{PACK_ROOT}`（即 `biz-resources/{RESOURCE_PACK}/`）。

#### 3.2 安装 Skills

遍历 `{PACK_ROOT}/skills/` 下的每个子目录：

```bash
for skill_dir in {PACK_ROOT}/skills/*/; do
  skill_name=$(basename "$skill_dir")
  dst="{SKILLS_DST}/$skill_name"
  rm -rf "$dst"
  cp -r "$skill_dir" "$dst"
  echo "  ✓ 已安装 skill: $skill_name"
done
```

#### 3.3 安装 Constitution

将 `{PACK_ROOT}/constitution/` 下的所有文件覆盖拷贝到 `<cwd>/constitution/`：

```bash
mkdir -p constitution
cp -f {PACK_ROOT}/constitution/* constitution/
echo "  ✓ 已安装 constitution"
```

#### 3.4 清理

如果使用了临时目录，清理之：

```bash
rm -rf "$TMP_DIR"
```

### Step 4：完成

向用户输出安装报告：

```
资源安装完成！

- Agent: {AGENT}
- 分支: {BRANCH}
- 资源包: {RESOURCE_PACK}
- Skills 安装到: {SKILLS_DST}
- Constitution 安装到: <cwd>/constitution/

已安装的 skills：
  - skill_name_1
  - skill_name_2
  - ...
```
