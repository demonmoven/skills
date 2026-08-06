---
name: fixloop-universal
description: 通用前端 E2E 测试与自动修复循环（v2）。基于 Playwright MCP 的五层架构：执行→观测→探索→判定→诊断。当用户提到"E2E 测试"、"端到端测试"、"fixloop"、"自动测试修复"、"测试循环"、"回归测试"，或者需要对前端项目运行浏览器自动化测试、生成测试用例、启动测试修复闭环时，使用此技能。即使用户只是说"帮我跑一下测试"或"这个功能需要验证一下"，只要上下文涉及前端项目的 E2E 场景，都应该触发此技能。
---

# Fixloop Universal v2 — 五层 E2E 测试引擎

给定一份**仓库配置文件**（`fixloop-profile.yaml`），对任意前端项目执行：测试用例生成 → 环境启动 → 浏览器 E2E 测试 → 状态探索 → AI 自动修复 → 代码审查 → 重测，直到全部通过或达到迭代上限。

## 命令

```
/fixloop-universal <command>

init <repo-path>    交互式生成仓库配置文件
gen-tests           从需求文档生成测试用例
run                 执行完整的测试修复循环
run --test-only     仅执行测试，不进入修复循环
run --fix-only      仅对已有测试结果执行修复循环
review              仅运行代码审查 Agent
```

## 前置条件

- Node.js >= 18
- 目标仓库已克隆到本地
- 仓库根目录存在 `fixloop-profile.yaml`（或通过 `--profile` 指定）
- **首次运行时自动安装**：Playwright MCP、浏览器、工程化依赖（通过 `scripts/setup.sh`）

---

## 核心概念：仓库配置文件

每个仓库一份 `fixloop-profile.yaml`，描述所有仓库特有的信息。

配置文件包含九个段落（v1 的六个 + v2 新增三个）：

| 段落 | 职责 | v2 新增 |
|------|------|---------|
| `project` | 项目元数据 | |
| `setup` | 环境启动 | |
| `auth` | 浏览器认证 | |
| `testing` | 测试配置 | |
| `fix_loop` | 修复循环参数 | |
| `output` | 输出配置 | |
| `browser` | 浏览器模式 | ✅ |
| `observation` | 时序观测 | ✅ |
| `exploration` | 状态探索 | ✅ |

完整 Schema 见 `references/profile-schema.yaml`。
示例见 `assets/profiles/react-vite.yaml` 和 `assets/profiles/nextjs.yaml`。

> **注意**：Agent 提示词中的 `{{var}}`、`{{#switch}}`、`{{#each}}` 是伪模板语法，由编排 Agent 在运行时解释并替换为实际值，不是真正的模板引擎。

**向后兼容**：v1 的 profile 在 v2 中直接可用，新增段落均有默认值。

---

## 五层架构

```
┌─────────────────────────────────────────────────────┐
│                  编排层 (SKILL.md)                     │
│  配置加载 → 环境启动 → 认证 → 测试 → 探索 → 修复循环   │
└───┬──────────┬──────────┬──────────┬──────────┬─────┘
    │          │          │          │          │
 Layer 1    Layer 2    Layer 3    Layer 4    Layer 5
 执行层      观测层      探索层      判定层      诊断层
    │          │          │          │          │
Playwright  save-session state-    pixelmatch  trace-
  MCP       + trace-    explorer + + axe-core  analyzer
(用户Chrome) analyzer    gremlins.js           + patterns
```

| 层 | 工程化工具 | 作用 |
|---|-----------|------|
| 执行 | `@playwright/mcp` | 浏览器自动化（a11y-tree-first） |
| 观测 | Playwright `--save-session` + `trace-analyzer.js` | 录制交互过程 → 结构化 timeline |
| 探索 | `state-explorer.js` + `gremlins.js` | 发现未覆盖的边界 bug |
| 判定 | `assertion-engine.js` + `pixelmatch` + `@axe-core/playwright` | 多信号断言（视觉+a11y+console+网络） |
| 诊断 | `trace-analyzer.js` 模式检测 + `diagnosis-agent.md` | 时序根因分析 |

---

## 铁律（全流程强制执行）

### 1. 禁止提前宣告完成

"完成"意味着**所有**测试用例都已通过，且每一条都有截图证据。
- 跳过任何测试用例而声称完成 → 违规
- 将未执行的测试标记为 PASSED → 违规
- `BLOCKED ≠ PASSED`

### 2. 禁止暂停确认（全自动模式）

`run` 模式下 Agent **不得主动暂停等待用户确认**。阶段完成后直接进入下一阶段，测试用例生成完毕后直接开始测试，修复完成后直接重测。用户介入提供反馈后，也必须立即恢复 fixloop 循环，不等待"继续"指令。

唯一允许暂停的场景：**缺少必要输入**（如 PRD 路径未配置）。"必要输入"指**无默认值、无法从 profile/项目上下文推断、且缺失会让下游立即出错**的配置；对有默认值或可推断的字段，直接采用，不要反复就可选字段请示用户。

### 3. 禁止 JS 注入绕过 UI（anti-mock 原则）

**不是人类交互可达的路径不可接受。** 如果功能无法通过正常 UI 交互到达，这本身就是 P0 bug。

唯一允许 JS 注入的场景：认证（cookie_inject / local_storage）和配置文件中声明的 mock。

v2 强化：Playwright 的 accessibility tree 天然强化了这一点——如果元素在 a11y tree 中不可交互，就不应该通过代码路径到达。

### 4. 视觉质量是测试的一部分

**一眼看上去就不对的地方就是要修的，即使不在测试用例动线里。**

v2 三重保障：
- `@axe-core/playwright` 自动扫描每个页面（无障碍问题自动发现）
- `pixelmatch` 与 baseline 截图对比（视觉回归自动检测）
- Agent 肉眼审查截图（兜底安全网）

发现的 `SIDE_FINDING` 自动加入修复队列。

### 5. 不要把自设的限制当作不工作的借口

scope_lock 是保护机制，不是跳过测试的理由。

### 6. 时序证据优先于截图猜测（v2 新增）

诊断失败时，**必须使用 timeline 数据**（DOM 变化、网络事件、console 日志、时序信息）来定位根因。禁止仅凭截图猜测。

Timeline 提供因果链，截图提供视觉确认。两者互补，不可替代。

### 7. 必须使用用户态 Chrome（v2 新增）

**任何情况下**浏览器必须通过 CDP 连接用户正在运行的 Chrome 实例（`--cdp-endpoint`），**禁止启动独立 Chrome/Chromium 实例**。

阶段 0 必须验证：
1. `browser.mode` 配置为 `cdp_connect` 或 `system_chrome`
2. `.mcp.json` 中 Playwright MCP 使用 `--cdp-endpoint` 参数
3. 如果用户 Chrome 未开启 remote debugging，提示用户启动命令而非回退到新实例
4. 浏览器自动化桥**仅使用 Playwright MCP**（开放工具链）；禁止替换或并行使用 Claude 专属桥（如 `claude-in-chrome`），因为 skill 需在非 Claude 用户环境下也能运行

### 8. 决策基于真实依据（v2 新增）

**任何决策必须基于 profile 声明、用户消息或实际存在的文件/配置。** 禁止基于臆想的路径、约定或"常见项目结构"执行探索（如搜索 `.harness/`、`.claude/` 或其它假设目录）。
- 若某字段在 profile 中缺失 → 先查是否有默认值或 v2 推断规则（见"参数自动推断"表）
- 仍然缺失且影响执行 → 按铁律 2 向用户索要，**不要"试着搜索一下"**

### 9. 公开承诺即执行合约（v2 新增）

一旦向用户**显式宣告**了下一步行动（如"现在进入阶段 5 回归验证"），就**必须执行那一步**；禁止静默回退到更早阶段或换成别的动作。
- 若执行中发现该步骤不适合，**显式**向用户说明原因后再切换，不得悄悄漂移
- 阶段 N → 阶段 N+1 切换前，重读自己上条公开计划作为 guardrail

### 10. 完整证据链：失败态必须独立保存（v2 修订新增）

每个失败 case 在**修复前**必须保存一张失败态截图作为独立证据，文件名必须**明确标识**失败阶段，**禁止被后续修复验证截图覆盖或挤掉**。

**为什么**：
- 修复前后对比是 PM 汇报、回归评审、事后追责的基础证据，**无法事后重建**（代码已改、UI 已变、同一序号的 `step-N.png` 被覆盖掉就彻底没了）
- 诊断 agent（`diagnosis-agent.md`）根据失败态截图定位根因，依赖这张图存在
- 仅保留「修复后 PASSED 态」的截图 = 丢失整个 bug 发现过程的视觉记录 = 后续 bug 汇报只能看到"现在对了"，说不清"之前错在哪"

**硬性约束**：
1. 失败态截图命名必须含 `-before-fix` 后缀，例：`step-3-before-fix.png`（不许仅用 `step-3.png`）
2. 修复后验证截图命名必须含 `-after-fix` 后缀，例：`step-3-after-fix.png`
3. 两者步骤序号必须对应（同一失败断言的 before/after 成对配对）
4. `case-result.json` 必填 `before_fix_screenshots: [...]` 和 `after_fix_screenshots: [...]` 两个数组字段；失败并被修复的 case 两个数组都不得为空
5. 子 agent **不得**用同名覆盖、**不得**因序号冲突而丢弃失败态；若发现冲突，按 `-before-fix-a/-b/-c` 做后缀扩展

---

## 工作流程

### 阶段 0：环境准备

1. 读取 `fixloop-profile.yaml`，校验必填字段
2. 运行 `scripts/setup.sh` — 自动安装 Playwright MCP、浏览器、工程化依赖
3. 运行 `scripts/mcp-config.js` — 生成项目 `.mcp.json`（必须使用 `--cdp-endpoint` 连接用户态 Chrome，见铁律 7）
4. 运行 `scripts/generate-storage-state.js` — 生成认证状态文件
5. **Playwright MCP 可用性检查**：用 `ToolSearch` 查询 `mcp__playwright__browser_navigate` 是否已加载。如果未加载（新生成的 `.mcp.json` 需要会话重启），**立即告知用户需要重启会话**，不得绕路写脚本替代 Playwright MCP

### 阶段 1：环境启动

读取 `setup` 段，执行：`pre_commands → install → build → dev_server → mock_server → health_check → post_commands`

Agent 提示词：`references/agents/setup-agent.md`

### 阶段 2：认证

| 策略 | v2 实现方式 |
|------|-----------|
| `none` | 直接导航 |
| `form_login` | `browser_snapshot` 找表单字段 → `browser_type` + `browser_click` |
| `cookie_inject` | `--storage-state` 预加载（setup 阶段已生成） |
| `local_storage` | `--storage-state` 预加载 |
| `custom` | `browser_console_execute` 执行自定义脚本 |

### 阶段 3：测试用例生成与发现

**`run` 命令始终包含用例生成。** 不需要单独执行 `gen-tests`。

流程：

1. **PRD Hard-Gate**：检查 `testing.specs_dir` 是否配置且目录非空。如果未配置或为空，**立即向用户索要 PRD/spec 路径**，不得跳过、不得凭代码猜测生成用例。
2. **启动 gen-tests 子 agent**：通过 Agent 工具启动独立子 agent 执行用例生成，隔离上下文不污染主编排器。子 agent 负责：
   - 读取 PRD/spec 文档**以及项目 `docs/`、`plans/`、`requirements/` 等规划文档**，逐条提取功能点
   - **优先识别核心用户动线（critical user journeys）**：从"真实用户打开产品后最想做的事"倒推，列出 3-5 条主流程路径；这些 **100% 必须覆盖** 且排在测试前列
   - 扫描变更范围（`git diff` 聚焦，不扫描全站）
   - 构建页面清单（用 Playwright MCP 的 `browser_snapshot` 获取交互元素）
   - PRD 功能点 × 页面清单交叉匹配
   - 按 PRD scenario 细粒度拆分，生成完整测试用例（通常 20+ 个）
   - **样式细节/边缘交互的用例不得挤占主流程用例的份额**——主流程用例位列最高优先级
   - 输出到 `testing.cases_dir`
3. **扫描 cases_dir**：子 agent 完成后，扫描生成的用例文件（.txt / .md / .yaml 混用）

Agent 提示词：`references/agents/test-gen-agent.md`

> **禁止**：没有 PRD 就从代码推测生成用例。PRD 是测试用例的唯一真相来源。
>
> **禁止**：仅从代码视角（"这个函数有哪些分支"）拆分测试。**必须以用户视角（"真实用户会做什么"）为主轴**；代码视角仅作为页面清单的补充。

### 阶段 4：逐个击破（测一个修一个）

前端 E2E 的最佳模式不是"跑完所有测试再统一修"，而是**逐个测试、逐个修复**。每个 case 在独立的 agent 上下文中完成测试→诊断→修复→验证的完整闭环。

**必须通过 Agent 工具为每个 case 启动独立子 agent**，不得在主编排器 context 中直接执行测试。主 context 只负责编排和收集结果。

**同 phase 内独立 cases 必须并行**：同一 phase 下互相不依赖的 cases 必须在**一条消息内**同时发起多个 Agent 调用，等全部返回后再进入下一 phase。独立 cases 串行是反模式（墙钟时间与上下文成本双输）。

**并行 worker 的资源隔离**（铁律 7 配套约束）：每个并行 subagent 必须在**独立的浏览器 context** 中运行，不得共用同一 CDP 端点或同一 storage-state——
- 每 worker 分配独立端口或独立 profile 目录
- 每 worker 使用独立的 `storage-state-{worker-id}.json`
- 共用会导致 DOM 状态污染、认证 cookie 冲突、并发点击互相覆盖，这些失败并非 case 本身的 bug

```
for case in independent_cases_of_current_phase（一条消息并行发起，等全部返回）:
    === 子 Agent（通过 Agent 工具启动）: 测试 + 修复 ===
    ① 执行测试（每步: snapshot → action → snapshot → screenshot）
    ② 如果通过 → 记录 PASSED，进入下一个 case
    ③ 如果失败 → 运行 trace-analyzer.js 生成 timeline
    ④ 诊断 + 修复（同一上下文，诊断数据不丢失）
    ⑤ 重测验证修复
    ⑥ 如果探索已启用 → 运行状态探索
    ⑦ 输出: case-result.json + 测试精化建议 + 新回归测试
```

每个测试步骤的执行流程（子 agent 内部）：
```
① browser_snapshot → before_state（a11y tree）
② 执行动作（优先使用 a11y ref 点击元素，避免硬编码像素坐标）
③ browser_snapshot → after_state
④ browser_screenshot → 视觉证据
   ↳ 若当前步骤断言 **失败**（首次断言失败、还没修）：截图文件名必须含 `-before-fix` 后缀
     （例 `cases/<case>/step-3-before-fix.png`），**禁止覆盖**，参见铁律 10
   ↳ 若当前步骤是 **修复验证**（修改代码后重测）：截图含 `-after-fix` 后缀
     （例 `cases/<case>/step-3-after-fix.png`），序号必须对应 before-fix
   ↳ 其它（PASSED 的普通 case）：用 `step-N.png` 即可
⑤ 对比 before/after a11y tree，记录变化
⑥ 检查 console 是否有新增错误
⑦ 多信号判断：a11y + 截图 + console
```

Playwright MCP 通过 `--save-session` 自动录制到 `output_dir/traces/`。

**为什么不是"跑完所有再统一修"：**
- 前端 bug 通常是局部的（一个组件、一段 CSS），一个上下文装得下
- 每次修完立即验证，反馈循环最短
- Agent 上下文只有一个 case，不被其他 case 的噪音干扰

**测试进化（GAN 模式）：** 修复 bug 后，Fix Agent 还会输出：
- **测试精化**：让模糊的测试描述更精确（不改意图，只改精度）
- **回归测试生成**：针对刚修复的 bug 生成专门的回归测试

Agent 提示词：`references/agents/test-runner.md`、`references/agents/fix-agent.md`、`references/agents/explorer-agent.md`

**执行机制**：主 context 作为编排器，通过 Agent 工具为每个 test case 启动独立子 agent。子 agent 完成测试+修复后返回结果（case-result.json），主 context 收集所有结果后生成 fix-round-summary.json。阶段 5 的回归验证同样作为独立子 agent 启动，读取 fix-round-summary.json 作为上下文。

### 阶段 5：回归验证（新 agent 上下文）

阶段 4 的多次修复可能互相冲突。在**全新 agent 上下文**中运行全量测试。

```
=== Agent Context B: 回归验证 ===
读取 fix-round-summary.json（阶段 4 的修复记录）
→ 运行所有测试（原始 + 修复的 + 自动生成的回归测试）
→ 全部通过 → fixloop 完成
→ 有回归 → 读取 fix-round-summary.json 定位引入回归的修复 → 修复 → 重测
```

**两个上下文通过文件通信：** `fix-round-summary.json` 包含每个 case 的修复记录、改动文件、修复策略，回归阶段用于定位回归来源。

### 阶段 6：生成报告

输出到 `output.dir`（默认 `fixloop-output/`）：
- `summary.json` / `summary.md` — 结果统计
- `fix-round-summary.json` — 逐个击破阶段的修复记录
- `cases/<case-name>/` — **每个 case 的证据目录**（per-case 独立子目录，铁律 10）
  - `step-N-before-fix.png` — 失败态截图（失败的 case 必存，**禁止被覆盖**）
  - `step-N-after-fix.png` — 修复后验证截图（序号与 before-fix 一一对应）
  - `step-N.png` — 普通 PASSED 步骤截图（无 before/after 命名）
  - `case-result.json` — 结构化结果（含 `before_fix_screenshots` / `after_fix_screenshots` 数组）
  - `diagnosis.md`（可选）— 诊断报告
- `traces/` — Playwright session 录制
- `timelines/` — 结构化时序数据
- `e2e-cases/regression-generated/` — 修复过程中自动生成的回归测试

**迭代趋势**（回归阶段有多轮时）：
- 通过率变化趋势
- 被修改的文件列表（定位回归来源）
- 每个 case 耗时（效率瓶颈分析）

---

## 浏览器模式

| 模式 | 配置 | 适用场景 |
|------|------|---------|
| 系统 Chrome | `browser.mode: system_chrome` | 默认，使用用户态 Chrome |
| 连接已有实例 | `browser.mode: cdp_connect` | 连接已打开的 Chrome |
| 独立 Chromium | `browser.mode: bundled` | 需要可预测环境 |
| 无头模式 | `browser.mode: headless` | CI/CD 环境 |

---

## 接入新仓库

```bash
# 1. 复制示例配置
cp <skill-path>/assets/profiles/react-vite.yaml fixloop-profile.yaml
# 编辑 project / setup / auth 段

# 2. 创建基线测试
mkdir -p e2e-cases/baseline
echo "首页能正常加载" > e2e-cases/baseline/smoke.txt

# 3. 运行
/fixloop-universal run
```

详见 `references/onboarding.md`。

### 参数自动推断（init 命令）

`/fixloop-universal init` 会自动检测项目信息：

| 推断项 | 检测方法 |
|--------|---------|
| `framework` | package.json deps: react / vue / svelte / @angular/core |
| `build_tool` | 配置文件: vite.config.* / next.config.* / webpack.config.* |
| `install_command` | Lock 文件: pnpm-lock.yaml → `pnpm install`, yarn.lock → `yarn` |
| `port_injection` | 构建工具约定: vite → `arg:--port`, next → `arg:-p` |
| `monorepo_tool` | 配置文件: pnpm-workspace.yaml / rush.json / nx.json |

检测结果以交互式确认呈现给用户，用户可逐项修改。

---

## 文件结构

```
fixloop-universal/
├── SKILL.md                           ← 你正在读的文件
├── scripts/                           ← 工程化脚本
│   ├── setup.sh                       自动安装依赖（首次运行触发）
│   ├── mcp-config.js                  生成 .mcp.json
│   ├── generate-storage-state.js      生成认证状态文件
│   ├── trace-analyzer.js              时序分析 + 失败模式检测
│   ├── state-explorer.js              a11y tree 状态探索
│   ├── assertion-engine.js            多信号断言引擎
│   ├── validate-profile.sh            校验配置文件
│   ├── check-port.sh                  检测可用端口
│   └── cleanup.sh                     清理进程
├── references/                        ← 按需加载的参考文档
│   ├── profile-schema.yaml            配置文件完整 Schema
│   ├── onboarding.md                  接入指南 + 常见问题
│   ├── agents/                        Agent 提示词模板
│   │   ├── setup-agent.md             环境启动
│   │   ├── test-runner.md             测试执行（Playwright MCP）
│   │   ├── test-gen-agent.md          用例生成
│   │   ├── explorer-agent.md          状态探索（v2 新增）
│   │   ├── diagnosis-agent.md         根因诊断（v2 新增）
│   │   └── fix-agent.md              代码修复
│   └── reviewers/                     审查 Agent 提示词
│       ├── lint-check.md
│       ├── code-quality.md
│       ├── requirements-compliance.md
│       └── synthesis.md
└── assets/                            ← 模板和示例
    ├── profiles/
    │   ├── react-vite.yaml            React + Vite 示例
    │   └── nextjs.yaml                Next.js 示例
    └── examples/
        ├── format-comparison.md       三种格式对比
        └── smoke-tests.txt            通用冒烟测试
```
