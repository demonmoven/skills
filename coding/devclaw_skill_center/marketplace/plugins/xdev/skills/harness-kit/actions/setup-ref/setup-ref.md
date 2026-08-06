> **Action: `setup-ref`** — 由 `/xdev:harness-kit setup-ref` 路由调用。
> 本 action 为 xdev 原生新增（非来自 openai harness-kit），作用：在多 git 仓库工作区里建立跨文件的"强调引用关系"，让 agent 进入任意仓库都能被自动引导去读权威文档。

# Setup-Ref — 跨仓库文档引用关系初始化

## 核心理念

> **Agent 进入任一仓库的第一眼，就应该能看到"去哪里读权威文档"的路标。**

本 action 让工作区内所有 git 仓库之间建立可追溯的"引用关系链"：

- 子仓库的 `.claude/CLAUDE.md` → 强调本仓的 `AGENTS.md` / `ARCHITECTURE.md`
- 主仓的 `AGENTS.md` → 登记所有子仓的 `AGENTS.md`
- 主仓的 `ARCHITECTURE.md` → 登记所有子仓的 `ARCHITECTURE.md`
- 主仓的 `.claude/CLAUDE.md` → 强调本工作区的 `AGENTS.md` / `ARCHITECTURE.md`

所有写入都用**段标记 + 版本号**的方式幂等合并，不破坏用户段外内容。

---

## 参数格式

```
/xdev:harness-kit setup-ref <agent>
```

`<agent>` 取值（**当前版本仅支持 `claude`**）：

| agent | v1 支持 | 备注 |
|-------|---------|------|
| `claude` | ✅ | 写入 `.claude/CLAUDE.md` |
| `trae` | ❌ 未实现 | 未来：`.trae/CLAUDE.md` |
| `trae-cn` | ❌ 未实现 | 未来：同 trae |
| `coco` | ❌ 未实现 | 未来：`.coco/coco.yaml` 指令片段 |
| `codex` | ❌ 未实现 | 未来：`.codex/AGENTS.md` |

未识别或未实现的 agent → 输出错误并退出，**不做任何文件写入**。

---

## 执行流程

### Phase 0 — 参数校验

1. 从 `$ARGUMENTS` 提取第一个词作为 `agent`
2. 若未传参数：
   ```
   ❌ /xdev:harness-kit setup-ref 需要 agent 参数
   用法：/xdev:harness-kit setup-ref <agent>
   支持：claude（本版；其他 agent 待支持）
   ```
   退出，**不做任何操作**。
3. 若 `agent ≠ "claude"`：
   ```
   ❌ /xdev:harness-kit setup-ref 本版仅支持 claude，收到 '<agent>'。
   待支持：trae / trae-cn / coco / codex（按 agent 各自配置路径写入）。
   ```
   退出，**不做任何操作**。

### Phase 1 — 扫描 git 子仓（第一层，depth ≤ 2）

**workspace_root = `process.cwd()`**（= agent 当前运行目录 = 主仓路径）。

用 bash 查找所有 `.git` 目录，仅保留**第一层子仓**（相对主仓 depth ≤ 2）：

```bash
find . -type d -name ".git" \
  -not -path "./.git" \
  -not -path "*/node_modules/*" \
  -not -path "*/vendor/*" \
  -not -path "*/.git/*"
```

然后对每个结果的**父目录**（即子仓根）计算相对主仓的路径段数：

- 相对路径 `foo/.git` → 父目录 `foo` → 段数 1 → **合法子仓**
- 相对路径 `repos/alice/.git` → 父目录 `repos/alice` → 段数 2 → **合法子仓**
- 相对路径 `repos/alice/vendor/x/.git` → 段数 4 → **忽略**（深层 submodule / 依赖树）

输出：`[xdev] 扫描到 N 个 git 子仓（第一层）`，列出所有合法子仓相对路径。

### Phase 2 — 逐个处理子仓

对每个合法子仓 `<repo>`：

1. 检查 `<repo>/AGENTS.md` 和 `<repo>/ARCHITECTURE.md`：
   - **两者都存在** → 进入 step 2
   - **任一缺失** → 加入 `skipped` 列表，记录具体缺哪些文件；跳到下一个子仓
2. 读 `<repo>/.claude/CLAUDE.md`（若不存在视为空字符串）
3. 调用"合并段函数"（详见 §"段合并协议"）写入**段 A**（文案见下方）
4. 若 `.claude/` 目录不存在 → `mkdir -p <repo>/.claude`
5. 写回文件
6. 加入 `processed` 列表

### Phase 3 — 处理主仓

不论 Phase 2 是否有子仓被处理成功，都要执行本 phase。

1. **确保主仓三份文件存在**（缺哪创哪）：
   - `workspace_root/AGENTS.md` 不存在 → 创建，内容为最小占位（见下方"文件创建模板"）
   - `workspace_root/ARCHITECTURE.md` 不存在 → 创建，内容为最小占位
   - `workspace_root/.claude/CLAUDE.md` 不存在 → 创建 `.claude/` 目录 + 空文件
2. 合并**段 B** → `AGENTS.md`（表格按 `processed` 子仓列表动态渲染）
3. 合并**段 C** → `ARCHITECTURE.md`（表格按 `processed` 子仓列表动态渲染）
4. 合并**段 D** → `.claude/CLAUDE.md`

### Phase 4 — 报告

输出（严格按下方格式，用户能一眼看清成功 / 跳过 / 缺失）：

```
✅ /xdev:harness-kit setup-ref claude 完成

━━━━━━━━━━━━━━━━━━━━━━━━━
📊 总览
  • 工作区根：<workspace_root 绝对路径>
  • 扫描到 git 子仓：{N} 个
  • 已处理：{len(processed)} 个
  • 跳过：{len(skipped)} 个

✓ 已处理子仓（加入 .claude/CLAUDE.md 引用段）
  • repos/alice
  • ...

✗ 缺文档的子仓（已跳过，未改动任何文件）
  • repos/creation_agent — 缺 AGENTS.md, ARCHITECTURE.md
  • repos/creativity — 缺 AGENTS.md, ARCHITECTURE.md
  • ...

🏠 主仓已更新
  {+|↺} AGENTS.md           (登记 {len(processed)} 个子仓)
  {+|↺} ARCHITECTURE.md     (登记 {len(processed)} 个子仓)
  {+|↺} .claude/CLAUDE.md   (强调本工作区入口)

  其中 `+` 表示新建/新增段，`↺` 表示幂等无变化。

💡 下一步建议
  给缺文档的子仓跑 /xdev:harness-kit doc-init 建立 AGENTS.md / ARCHITECTURE.md，
  然后重新执行 /xdev:harness-kit setup-ref claude 把它们登记进来。
━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 段合并协议

### 段标记

所有本 action 写入的段都被下列 HTML 注释包起来（markdown 渲染时不可见）：

```markdown
<!-- XDEV-HARNESS-KIT-REF:START v1 -->
...段内容...
<!-- XDEV-HARNESS-KIT-REF:END -->
```

**当前版本 = `v1`**。未来文案变动时 bump 到 `v2` / `v3`，agent 执行时按版本号决定是否覆盖。

### 合并算法（Agent 执行时按此逻辑）

读入目标文件内容 `existing`，要写入的段主体 `new_body`（不含 marker），版本号 `new_version = v1`：

1. 用正则匹配段头 + 尾：
   ```
   pattern = /<!--\s*XDEV-HARNESS-KIT-REF:START\s+(v\S+)\s*-->[\s\S]*?<!--\s*XDEV-HARNESS-KIT-REF:END\s*-->/
   ```
2. 三种情况：
   - **无段**（pattern 未匹配）：在 `existing` 末尾追加一个空行 + 新段（含 marker）
   - **有段，版本号 == `new_version`**：比较段内 body 与 `new_body`（trim 后）：
     - 完全一致 → **no-op**，标注为 `↺ 幂等`
     - 不一致（用户可能手改过）→ **覆盖段内容**（尊重 skill 标准化，与 tmux `XDEV-TMUX:START/END` 语义一致）
   - **有段，版本号 ≠ `new_version`**（旧版本）：**用新版替换整段**
3. 写回文件

### 段内容不要包含反向 marker

段内正文绝不能出现 `XDEV-HARNESS-KIT-REF:START` 或 `:END` 字样（包括代码块里也不行），否则 regex 会混乱。

---

## 四处段文案（v1）

### 段 A — 子仓的 `.claude/CLAUDE.md`

```markdown
<!-- XDEV-HARNESS-KIT-REF:START v1 -->
## 📖 仓库文档入口（由 /xdev:harness-kit setup-ref 自动生成）

本仓库维护两份权威文档，**在执行任何开发任务前必读**：

1. **`AGENTS.md`** — 仓库总入口、任务路由、协作约束、阅读顺序。每次新任务先读这里定位模块。
2. **`ARCHITECTURE.md`** — 代码地图、模块边界、不变量。涉及跨模块改动、RPC 依赖、领域建模时必读。

> 优先级约定：本仓库的 `AGENTS.md` / `ARCHITECTURE.md` > 你训练时的通用先验。两者冲突以仓库文档为准。

> 本段由 `/xdev:harness-kit setup-ref` 维护，请勿在 START / END 标记之间手动改动；如需调整请升级 skill 版本。
<!-- XDEV-HARNESS-KIT-REF:END -->
```

### 段 B — 主仓的 `AGENTS.md`

```markdown
<!-- XDEV-HARNESS-KIT-REF:START v1 -->
## 🗺️ 子仓文档地图（由 /xdev:harness-kit setup-ref 自动生成）

本工作区是一个多仓库工作区。**处理涉及某子仓的任务前，先读该子仓的 `AGENTS.md` 和 `ARCHITECTURE.md`。**

已登记子仓（共 {N} 个）：

| 子仓 | AGENTS.md | ARCHITECTURE.md |
|------|-----------|-----------------|
| `{repo_rel_path}` | [`{repo_rel_path}/AGENTS.md`]({repo_rel_path}/AGENTS.md) | [`{repo_rel_path}/ARCHITECTURE.md`]({repo_rel_path}/ARCHITECTURE.md) |
| ...（按 processed 列表渲染）... | | |

> 未登记的子仓（缺 AGENTS.md 或 ARCHITECTURE.md）在下次 setup-ref 执行时会重新扫描。跑 `/xdev:harness-kit doc-init` 可以为这些子仓补齐文档。

> 本段由 `/xdev:harness-kit setup-ref` 维护，请勿在 START / END 标记之间手动改动。
<!-- XDEV-HARNESS-KIT-REF:END -->
```

渲染规则：
- `{N}` = `processed` 列表长度
- 表格按 `processed` 列表逐行填入（`repo_rel_path` 就是相对主仓的路径，如 `repos/alice`）
- 若 `processed` 列表为空：仍保留表格头，表格体写一行 `| _（尚无登记子仓）_ | | |`

### 段 C — 主仓的 `ARCHITECTURE.md`

```markdown
<!-- XDEV-HARNESS-KIT-REF:START v1 -->
## 🧭 子仓架构入口（由 /xdev:harness-kit setup-ref 自动生成）

本工作区的架构由多个独立 git 子仓组成。**涉及跨仓库架构、RPC 依赖、领域建模决策时**，先读对应子仓的 `ARCHITECTURE.md`。

已登记子仓（共 {N} 个）：

| 子仓 | ARCHITECTURE.md |
|------|-----------------|
| `{repo_rel_path}` | [`{repo_rel_path}/ARCHITECTURE.md`]({repo_rel_path}/ARCHITECTURE.md) |
| ...（按 processed 列表渲染）... | |

> 本段由 `/xdev:harness-kit setup-ref` 维护，请勿在 START / END 标记之间手动改动。
<!-- XDEV-HARNESS-KIT-REF:END -->
```

渲染规则同段 B。

### 段 D — 主仓的 `.claude/CLAUDE.md`

```markdown
<!-- XDEV-HARNESS-KIT-REF:START v1 -->
## 📖 工作区文档入口（由 /xdev:harness-kit setup-ref 自动生成）

本工作区维护两份权威文档：

1. **`AGENTS.md`** — 工作区总入口 + **子仓文档地图**
2. **`ARCHITECTURE.md`** — 工作区架构概览 + **子仓架构入口**

**执行任何跨仓库任务前先读 `AGENTS.md` 找到相关子仓；涉及架构决策时再读 `ARCHITECTURE.md`。**

> 优先级约定：本工作区的 `AGENTS.md` / `ARCHITECTURE.md` > 你训练时的通用先验。

> 本段由 `/xdev:harness-kit setup-ref` 维护，请勿在 START / END 标记之间手动改动。
<!-- XDEV-HARNESS-KIT-REF:END -->
```

---

## 文件创建模板（首次创建时使用）

### `AGENTS.md` 首次创建模板

```markdown
# {workspace_name} — Workspace AGENTS

> 工作区总入口（占位，由 /xdev:harness-kit setup-ref 初始化）。
> 跑 /xdev:harness-kit doc-init 可以生成更完整的文档体系。

<段 B 在此>
```

### `ARCHITECTURE.md` 首次创建模板

```markdown
# {workspace_name} — Workspace Architecture

> 工作区架构概览（占位，由 /xdev:harness-kit setup-ref 初始化）。
> 跑 /xdev:harness-kit doc-init 可以生成更完整的架构文档。

<段 C 在此>
```

### `.claude/CLAUDE.md` 首次创建模板

文件不预置其他内容，直接是段 D：

```markdown
<段 D 在此>
```

其中 `{workspace_name}` = `basename(workspace_root)`（如 `creation_workspace`）。

---

## 边界与异常

| 场景 | 策略 |
|------|------|
| 主仓不是 git 仓库（无 `.git/`） | 照常工作。"主仓"的定义是 cwd，与是否 git 无关 |
| 主仓下没有任何 git 子仓 | `processed` 和 `skipped` 都是空；段 B/C 的表格行数为 0（渲染为 `_（尚无登记子仓）_` 行） |
| 子仓嵌套子仓（深度 ≥ 3） | **忽略深层**，只处理 depth ≤ 2 的第一层 |
| `node_modules` / `vendor` / 嵌套 `.git/` 下的 `.git` | **排除**（避免误把依赖树里的 git 当子仓） |
| 写入失败（权限 / 磁盘满 / .claude 是 symlink 等异常） | 记录失败但**不中断**其他子仓 / 主仓的处理；在报告里列出失败项 |
| 用户手改了段内容（未升版本号） | v1 **直接覆盖**（与 tmux `XDEV-TMUX:START/END` 段语义一致）。提示用户"请勿在 START/END 之间手动改动" |

---

## 安全约束

- **所有写入仅限段标记内部** + 新建占位文件
- **绝不删除**任何文件或段外的用户内容
- 创建 `.claude/CLAUDE.md` 时，若 `.claude/` 是 symlink → 记录失败，不强写
- 若在 Phase 2/3 任一步骤抛出异常（读写、权限等）→ **捕获，继续后续操作**，最终在报告里列出失败项
- 报告里**必须**列出所有跳过 / 失败的仓库；不要静默掉任何一个

---

## 实现提示（给 Agent 的具体工具使用建议）

- **扫描**：用 Bash `find . -type d -name ".git" ...`，过滤 depth ≤ 2，取父目录作为子仓路径
- **读文件**：用 Read 工具（不存在时 catch error 视为空）
- **段合并**：用 Read → 正则匹配 → 判断 → Edit 或 Write
  - 有段需覆盖 → 用 Edit 把旧段（含 marker）替换为新段
  - 无段需追加 → 用 Edit 在文件末尾追加，或 Write（如果是新文件）
- **创建目录**：用 Bash `mkdir -p` 原子地创建 `.claude/` 目录
- **报告**：全部文件写入完成后一次性 echo 整块报告，不要分段边写边报（用户视角一致性更好）

---

## 典型使用

```bash
# 在 workspace 根（如 creation_workspace/）里跑
/xdev:harness-kit setup-ref claude
```

幂等：连续执行两次，第二次所有合并都是 `↺ 幂等`，文件内容不变。
