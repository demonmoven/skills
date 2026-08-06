# Delta Spec 合并算法

> 把一个 change 内的 delta specs(`changes/{name}/specs/<cap>/spec.md`)合并到主 specs(`docs/xdev/openspec/specs/<cap>/spec.md`)。被 `actions/archive.md` 在 Step 5b `[1] 现在 sync` 路径调用。

## 输入参数

- `CHANGE_NAME`:要合并的 change 名(kebab-case)
- `PRIMARY_REPO`:主仓库根目录绝对路径

## 输出

- 把 4 类 delta 操作(ADDED / MODIFIED / REMOVED / RENAMED)应用到主 spec
- 输出每个 capability 的合并摘要

---

## Step 1:扫描 delta specs

```bash
ls -d "{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/specs/"*/ 2>/dev/null
```

不存在 → 输出 "本 change 没有 delta specs,无需 sync。" 终止(由调用方决定后续)。

存在 → 列出每个 capability:

```text
DELTA_CAPS = [<cap1>, <cap2>, ...]
```

## Step 2:对每个 delta spec 执行合并

对 `DELTA_CAPS` 中的每个 capability:

### 2a. 读取 delta spec

```text
DELTA_SPEC_PATH = {PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/specs/{cap}/spec.md
MAIN_SPEC_PATH  = {PRIMARY_REPO}/docs/xdev/openspec/specs/{cap}/spec.md
```

Read `DELTA_SPEC_PATH`,识别 4 种 delta 操作段:

- `## ADDED Requirements` — 新增 requirement(直接追加到主 spec)
- `## MODIFIED Requirements` — 修改既有 requirement(用新版替换主 spec 中的同名块)
- `## REMOVED Requirements` — 删除既有 requirement(从主 spec 中移除同名块)
- `## RENAMED Requirements` — 重命名(改主 spec 中的 requirement 名称)

### 2b. Read 主 spec(如存在)

```bash
ls "{MAIN_SPEC_PATH}" 2>/dev/null
```

- **不存在** → 视为新 capability,直接把 delta 中的 ADDED Requirements 作为初始 spec 内容 + 标准头部
- **存在** → Read 主 spec,准备做合并

### 2c. 执行合并

按 4 种 delta 操作分别处理:

1. **ADDED**:从 delta spec 中提取每个 `### Requirement: <name>` 块,追加到主 spec 末尾。
   - 如果主 spec 已有同名 requirement,**报错**:"ADDED 但同名 requirement 已存在,请改用 MODIFIED"

2. **MODIFIED**:从 delta spec 中提取每个 requirement 块,在主 spec 中找到同名块(`### Requirement: <name>`,whitespace-insensitive),用新版替换整个块(从 `### Requirement:` 到下一个 `###` 之前)。
   - 如果主 spec 中找不到同名块,**报错**:"MODIFIED 但主 spec 中找不到同名 requirement"

3. **REMOVED**:从 delta spec 中提取 requirement 名称,在主 spec 中找到同名块并删除。
   - delta 中必须包含 `**Reason**:` 与 `**Migration**:`,缺一不可 → **报错**
   - 删除前在主 spec 中保留一行注释:`<!-- REMOVED: <name> ({date}) - see archive/{date}-{change}/specs/{cap}/spec.md for migration -->`

4. **RENAMED**:解析 delta 中的 `FROM:` / `TO:`,在主 spec 中把 `### Requirement: <FROM>` 改为 `### Requirement: <TO>`

### 2d. Write 合并后的主 spec

如果主 spec 不存在,先 `mkdir -p $(dirname {MAIN_SPEC_PATH})`。

Write 合并后的内容到 `{MAIN_SPEC_PATH}`。

### 2e. 输出本 capability 的合并摘要

```
✓ Synced {cap}
  - ADDED:    {N} requirements
  - MODIFIED: {N} requirements
  - REMOVED:  {N} requirements
  - RENAMED:  {N} requirements
```

## Step 3:输出最终摘要

```
✅ Sync 完成

Change: {CHANGE_NAME}
合并的 capabilities: {N}

主 specs 已更新:
  - {PRIMARY_REPO}/docs/xdev/openspec/specs/{cap1}/spec.md
  - {PRIMARY_REPO}/docs/xdev/openspec/specs/{cap2}/spec.md
  ...
```

---

## Guardrails

- 合并是 ACID 风格:**全部成功或全部失败**,不要留下半成品。任何 cap 报错 → 立即停止,不写任何文件
- ADDED 同名冲突 / MODIFIED 找不到同名块 / REMOVED 缺 Reason+Migration → 立即报错暂停
- 合并前先 `git status` 确认主 specs 没有未提交改动(避免与用户手动改动冲突);有改动 → 提示用户先 stash 或 commit
- 处理 H4 scenarios(`####`)时严格保持 4 个 hashtags,**不能误改成 3 个**
- 合并后建议提示用户 `git diff docs/xdev/openspec/specs/` 检查
