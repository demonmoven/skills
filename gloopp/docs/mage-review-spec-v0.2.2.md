# Spec：Gloop Mage Review — 剑士输出的抗偏差层

> **版本**：v0.2.2（2026-06-23，v0.2.1 的工程落地补丁）
> **v0.2.1 → v0.2.2 补丁摘要（修 4 个 implementation-level 坑，不改变产品/架构方向）**：
> - **测试策略**：显式引入 `tsx` 作为测试执行器（不是 vitest/jest 级的测试框架），承认「无 React 测试框架」≠「无 TS 执行器」；spec 里不再自相矛盾
> - **ReviewSource 写入责任**：从 `fsstore.AppendReview` 下沉到 4 个调用点分别构造；fsstore 只持久化，绝不反向读 quest/phases 做推断
> - **Fixtures**：从 `web/src/__fixtures__/` 移到 `web/src/dev/`（dev-only），通过 Vite `import.meta.env.DEV` 动态 import + tree-shake 保证生产 bundle 0 字节污染；验收明确加「fixtures 不进生产包」
> - **review.structured_invalid event**：原 structured JSON 不再全量塞 event，只放 `schema_version / error_list / field_paths / raw_size / raw_hash(SHA-256) / 截断 ≤16KB 的 middle snippet`；完整原文在 review comment / structured_review artifact 中可追溯，避免 events.jsonl 膨胀 + 敏感内容重复落盘
> **核心边界 / 产品方向**（继承 v0.2.1，完全不变）：严格遵守 PRDv2「系统做机制，agent 做决策」；前端不做语义抽取；contract 按成熟度切 Phase；通用 Agent Loop 编排平台定位；§10 三条红线不变。

---

## 0. 可信信号来源白名单（完全继承 v0.2.1）

v0.2.2 不改动白名单，只在 §0.2.1 ReviewSource 中**明确写入责任**。下面只给补丁摘要。

### 0.2.1 补丁：`ReviewRecord.ReviewSource` 的写入责任（修复 v0.2.1「AppendReview 自动补」的错误假设）

**错误根因**：`fsstore.QuestStore.AppendReview(qid string, r *ReviewRecord) error` 只有 qid 和 record 两个参数，没有 quest/phases/session/actor 上下文；如果让它反向 LoadQuest/LoadPhases 会污染存储层。**ReviewSource 只能由写入调用点（调用方）构造后传入 AppendReview**。fsstore 只做字段搬运 + 持久化，绝不推断。

当前 4 个（含未来 1 个）写入调用点和各自能构造的 ReviewSource 如下：

| 调用点 | 文件 & 行号（现状） | 可用上下文 | 能填的 ReviewSource 字段 | 构造策略 |
|---|---|---|---|---|
| **宏循环 mage 评审写入** | `internal/orchestrator/macro_loop.go` runMageReviewStep（~404 行） | `mage` 对象（ID 明确）、当前 phase 的 `phase_idx / session_id / class=mage / role=mage`、完整的 QuestMeta + phases[] | 5 个字段**全部能填完整** | `SourceRole="mage" / SourceClass=model.ClassMage / SourcePhaseIdx=phase.Index / SourceAdventurerID=mage.ID / SourceSessionID=phase.SessionID`。**这是 Phase 1.5 唯一能全量填齐的路径** |
| **domain.CompleteReview**（legacy mage 服务入口） | `internal/domain/quest/service.go` CompleteReview（~500 行） | 只有 qid + opts.Verdict/Comment；opts 不含 session/phase 上下文。当前 ReviewedBy 写的是字面量 `"mage"` | 现阶段能填：`SourceRole="mage"`；其余字段未知。**标记 `SourceIncomplete=true`** | 方案二选一：<br>**A. 轻量改法（推荐）**：给 `CompleteReviewOptions` 加 `AdventurerID / PhaseIdx / SessionID` 三个可选字段，调用方有上下文就填；没有就留空 + `SourceIncomplete=true`。<br>**B. 不改签名**：只填 `SourceRole="mage"` + `SourceIncomplete=true`，前端按 legacy 兼容路径走。Phase 1 用 §0.1.1 的 `selectLatestMageReview()` 兜底，不影响 UI 正确性。|
| **用户 ResolveUserReview** | `internal/domain/quest/service.go` ResolveUserReview（~192 行） | ReviewedBy 写的是 `"user"`；`opts` 带 userId / actor 上下文；没有 phase/session 概念（user review = phase 外部决策） | `SourceRole="user" / SourceAdventurerID=actorUserId（有则填；没有留空）/ SourcePhaseIdx = -1`（phase=-1 表示"非 phase 内决策"） | 完整填 `SourceRole`，PhaseIdx=-1 作为"非 phase 内决策"的 sentinel。前端永远不把 `role=user` 的 review 当 Mage Review |
| **automation/policy 自动评审**（**当前不写 reviews[]，只写 quest.final_***；未来可能写**） | N/A（当前不存在写入） | Policy/automation actor id、decision id | 如果未来开启：`SourceRole="automation" 或 "policy" / SourceAdventurerID="" 或 policy_name / SourcePhaseIdx = -1 / SourceIncomplete=false` | **Phase 1.5 不做**。保持当前 final_* 写入路径即可，不在 reviews[] 增加新写入 |

**ReviewSource 结构补字段**（v0.2.2 在 v0.2.1 的 5 字段基础上仅新增 1 个布尔）：

```go
type ReviewSource struct {
    SourceRole         string `json:"source_role"`            // warrior / mage / user / policy / automation — 永远必填
    SourceClass        string `json:"source_class,omitempty"` // 冒险者职业：warrior/mage。非冒险者角色（user/policy/automation）留空
    SourcePhaseIdx     int    `json:"source_phase_idx"`       // 阶段索引（定位用；非 phase 内决策填 -1，不允许省略 0 —— 0 是 warrior phase）
    SourceAdventurerID string `json:"source_adventurer_id,omitempty"` // 冒险者 ID。user/policy/automation 留空
    SourceSessionID    string `json:"source_session_id,omitempty"`    // phase 对应 session id。非 phase 内决策留空

    // v0.2.2 新增：标记此条 source 是否由调用方提供了完整字段。
    // false = 调用方已按 contract 填齐所有该填的字段（user/policy/automation 不需要 class/adventurer_id/session_id，也算齐）；
    // true  = 调用方上下文不全，此条 review 身份可以信任（Role 对）但溯源（PhaseIdx/SessionID）不完整。
    SourceIncomplete   bool   `json:"source_incomplete,omitempty"`
}
```

**Phase 1/1.5 的 UI 判定策略不变**：Phase 1 仍然走 `selectLatestMageReview()`（用 ReviewedBy 判断，兼容 legacy）；Phase 1.5 ReviewSource 字段存在且非零值时，优先按 `SourceRole == "mage" || SourceClass == "mage" || SourceAdventurerID == quest.mage_id` 判断 Mage Review。当 `SourceIncomplete=true` 时，在 MageReviewBand 加一行 tag「溯源信息不完整，建议升级到最新 agent」。

---

## 8. 验收标准（v0.2.2 重写 §8.1/§8.2/§8.3/§8.4.2 四个工程级坑）

### 8.1 测试策略（v0.2.2 补丁：明确引入 tsx，消除「不引入运行器」和「运行 TS 脚本」的矛盾）

v0.2.1 写了「不引入测试运行器」但又要运行 `.ts` 测试脚本，自相矛盾。v0.2.2 修正措辞：

> **Phase 1 的约束 =「不引入 React 测试框架（vitest / jest / @testing-library/react / jsdom / storybook）」**，**不是「任何执行器都不引入」**。TS 源文件需要一个 TypeScript 执行器才能在 build 之前跑通。选型是 `tsx`（MIT，3MB 安装体积，按需解析 TS，无测试框架 DSL，只是 Node 的 TS 入口）。

#### 8.1.1 引入清单

Phase 1 只加 1 个 devDependency：
- `web/package.json` → `devDependencies.tsx: "^4.19"`（和 Vite 5 + TS 5.6 兼容）。不在根 package 加。

npm 脚本（Phase 1 加）：
```json
{
  "test:mage-review": "tsx --conditions=development src/__tests__/mage-review.test.ts",
  "test": "npm run test:mage-review"
}
```

**明确：** 这个 `test` 脚本只跑 3 个纯函数的断言，不引入任何 React 测试框架、不做 DOM 渲染。`npm run build` 顺序保持 `tsc && vite build`，**不**挂 `test:mage-review`（§9 开放问题 #3 的默认值）。想本地跑就手动 `npm run test:mage-review`；CI 接入另议。

#### 8.1.2 验收加一条（非阻塞 Phase 1，但要写入 spec）
- Phase 1 PR 中，`npm run test:mage-review` 的输出显示 **PASSED: A1~A8 + B1~B9 + C1~C3 = 20/20**（3 函数 × 用例合计）。

### 8.2 纯函数 TS 单测（Phase 1 阻塞项）

纯函数本体仍写在 `web/src/domain/mageReview.ts`（**从 v0.2.1 的 `__tests__/` 里移出来**，放在 domain 目录）：
- `selectLatestMageReview(reviews: ReviewRecord[], quest: { mage_id: string }): LatestMageReviewMatch | null`
- `mageReviewTone(match: LatestMageReviewMatch | null, quest: { intensity: string; mage_id: string }): Tone`
- `mageReviewState(match: LatestMageReviewMatch | null, quest: { phases?: PhaseTask[]; mage_id: string }): ReviewDisplayState`

测试脚本写在 `web/src/__tests__/mage-review.test.ts`（只被 `npm run test:mage-review` 直接执行，**不被业务代码 import**）。

实现方式：`import node:assert/strict` 原生模块。**不引入 vitest/jest describe/it DSL**，用最简单的 for-loop 跑用例数组：

```typescript
// web/src/__tests__/mage-review.test.ts
import assert from 'node:assert/strict'
import { selectLatestMageReview, mageReviewTone, mageReviewState } from '../domain/mageReview.js'
// fixtures 直接在本文件里构造（不读外部 JSON，避免执行器路径问题）

let pass = 0
let total = 0
for (const c of A_CASES) { total++; try { assert.deepEqual(selectLatestMageReview(c.reviews, c.quest), c.expected); pass++ } catch (e) { console.error(`FAIL A/${c.name}:`, e.message) } }
for (const c of B_CASES) { total++; try { assert.deepEqual(mageReviewTone(c.match, c.quest), c.expected); pass++ } catch (e) { console.error(`FAIL B/${c.name}:`, e.message) } }
for (const c of C_CASES) { total++; try { assert.deepEqual(mageReviewState(c.match, c.quest), c.expected); pass++ } catch (e) { console.error(`FAIL C/${c.name}:`, e.message) } }

console.log(`PASSED: ${pass} / ${total}`)
if (pass !== total) process.exit(1)
```

**注意**：测试脚本只导入 3 个纯函数，不导入 React、不导入 QuestDetail 页面，**保证执行器层面不依赖 DOM**。这就是为什么可以只加 `tsx` 而不加 jsdom/testing-library。

### 8.3 Fixtures（v0.2.2 补丁：dev-only，0 生产污染）

v0.2.1 放在 `web/src/__fixtures__/mage-review/fixtures.tsx` 可能被误 import 进生产 build。v0.2.2 改法：

#### 8.3.1 文件位置与命名

放在 `web/src/dev/MageReviewFixtures.tsx`。目录命名 `dev/` 暗示「开发专用」。

#### 8.3.2 接入方式（生产 tree-shake 为 0 字节）

**不**在 App / Router 里 import。接入方式：在 `web/src/main.tsx` 里用 `import.meta.env.DEV` 显式条件分支 + 动态 import + 一个 local dev 开关：

```tsx
// web/src/main.tsx 末尾（或者 App.tsx 某个不影响布局的角落）
if (import.meta.env.DEV) {
  // ?magereview-fixtures 挂载点：
  // http://127.0.0.1:5173/quests/q-xxx?magereview-fixtures=1
  const params = new URLSearchParams(window.location.search)
  if (params.has('magereview-fixtures')) {
    import('./dev/MageReviewFixtures.tsx')
      .then((m) => m.renderFixturesInto(document.body))
      .catch((e) => console.error('Failed to load mage-review fixtures:', e))
  }
}
```

关键点：
- `import.meta.env.DEV` 只在 `vite dev` 时为 `true`；`vite build` 会把整个 `if` 分支和里面的动态 import **完全 tree-shake 掉**，生产 bundle 中不会有 `src/dev/` 的任何代码和 JSON。
- 即使在开发模式下，不挂 `?magereview-fixtures=1` 参数也**不会**加载 fixtures 模块，正常开发 QuestDetail 页面时零额外开销。
- Fixtures 渲染到 `document.body` 上（覆盖正常 UI），验收人员直接访问带 query param 的 URL 即可看到 6 个卡片。

#### 8.3.3 Fixtures 数据源

6 个 JSON 文件放在 `web/src/dev/fixtures/mage-review/` 下（F1~F6），和 TSX 渲染器同目录。**开发模式下由动态 import 拉进来，生产中 0 字节**。

#### 8.3.4 新增验收硬指标（Phase 1 阻塞项）

- **「Fixtures 不进入生产 bundle」**：Phase 1 PR 提交前必须执行 `npm run build`，在 `dist/assets/` 中 **grep 不到 fixture 里的任何唯一字符串**（比如 `F1-fixture-quest-id` 这种每个 fixture 打的标记 id）。grep 命中 → 验收失败。
- `npm run build` 输出的 chunk list 中 **不得出现 `MageReviewFixtures` 或 fixtures 文件名的 chunk**。

### 8.4 Structured Review 校验失败事件（v0.2.2 补丁：防膨胀 + 防敏感内容重复）

v0.2.1 说 `review.structured_invalid` event 放「原 structured JSON + 错误原因」，风险有二：
1. **events.jsonl 膨胀**：structured_review 可能带 evidence.snippet × 20 条 disagreement → 单条 event 几百 KB。
2. **敏感内容重复落盘**：agent 输出里可能包含路径、代码片段、用户上下文信息，已经存在于 review comment / artifact 中，再往 events.jsonl 塞一份等于「多一份索引」，如果日后有清理策略（删 artifacts）会造成不一致。

v0.2.2 改法：`review.structured_invalid` event 的 payload 严格只放以下 8 个字段，**不含完整 structured JSON**：

```typescript
type StructuredInvalidEventPayload = {
  // 1. 错误摘要（机器可读，用于 Mage/Agent 侧做协议修正）
  schema_version: string            // 提交方声明的 schema_version（例："v1"）——未知版本是第一大失败原因
  raw_size_bytes: number            // 提交的 structured JSON 的原始字节数
  raw_sha256: string                // 提交的 structured JSON 的 SHA-256 hex（32 bytes），用于跨系统比对/审计
  errors: Array<{
    code: string                    // 枚举：SCHEMA_VERSION_UNKNOWN / LIST_LENGTH_EXCEEDED / ID_DUPLICATE / ENUM_INVALID / TYPE_MISMATCH / ID_EMPTY / EVIDENCE_KIND_UNKNOWN …（后续可扩展）
    path: string                    // JSON path，例 "disagreements[3].evidence[1].kind"；空字符串 = 根
    message: string                 // ≤200 字的人类可读错误
    got?: unknown                   // 可选：出错位置的原始值（截断 ≤256 字节）—— 小类型可以附，大对象不附
  }>

  // 2. 审计辅助（人类可读，仅当 size ≤16KB 时才附；>16KB 时字段为 undefined，不写）
  truncated_snippet?: string        // 中间截断，总长度 ≤ 16384 bytes；格式 `"...<前 8KB>...[TRUNCATED by ${N} bytes]...<后 8KB>..."`
  snippet_truncated_bytes?: number  // 实际被截掉的字节数（仅 truncated_snippet 存在时写）

  // 3. 溯源：原始 structured JSON 在哪
  attached_review_ts_ms: number     // 本条 review 的时间戳
  reviewed_by: string               // review 的 ReviewedBy（方便追溯是哪个 adventurer 写崩了）
}
```

**回溯完整原文的路径**（不在 event 里存）：
- 如果 review 有 `review_artifact_id` → 下载对应 artifact 读原文
- 如果 review 没有 artifact_id，但 review comment 后面附了 structured JSON（agent 写的）→ 从 `ReviewRecord.Comment` 解析（或搜对应日志）
- 如果都没有 → 靠 `raw_sha256` 在 agent 侧日志里反查（agent 自己的 session 日志应记录自己提交的 structured 哈希）

这样 event payload 的**硬上限**是：`errors[]` 限制最多 20 条（超了合并成一个 LIST_TOO_LONG_MANY_ERRORS），每条 message ≤200 字 + `truncated_snippet` ≤16KB → 单 event **理论最大 ≈ 20KB**（典型失败 < 4KB）。events.jsonl 不会被打爆。

---

## 9. 开放问题（继承 v0.2.1，不改变）

只新增 1 个 implementation-level 的开放问题，其余三个保持 v0.2.1 不变：

| # | 问题 | v0.2.2 默认值 | 待决定 |
|---|---|---|---|
| 4（新增） | `CompleteReview`（domain service 入口）走 §0.2.1 的方案 A（加 opts 3 个字段）还是方案 B（只填 role + SourceIncomplete=true）？ | **默认走方案 A**：一次性补齐签名，让 domain 层的 mage review 也能 Phase 1.5 就走完整溯源路径；因为 macro 路径已经能填齐，不应该让 service 路径拖后腿。代价：`CompleteReviewOptions` 加 3 optional 字段，所有调用点需要 review 一下有没有上下文可传。 | 是 |

---

## 10. 红线（完全继承 v0.2.1，一字不改）

略。红线三条不变：平台不做语义判断 / 法师不变门禁 / 法师不抢话语权。

---

## 附：v0.2.2 补丁的 diff 视角（给 reviewer 快速扫）

| 模块 | v0.2.1 | v0.2.2 | 风险下降 |
|---|---|---|---|
| 测试策略 | 说「不引入测试运行器」又要跑 .ts → 矛盾 | 加 tsx（TS 执行器，不是 React 测试框架）；明确约束是「不引入 React 测试框架」 | 「按 spec 做会跑不起来」→ 可执行 |
| ReviewSource 写入 | 写「AppendReview 按上下文尽量填」→ 存储层越界 | 下沉到 4 调用点；macro 路径填齐 5 字段；service 路径要么加 opts 要么 SourceIncomplete 标记；fsstore 只持久化 | 「实现会污染存储层」→ 分层清晰 |
| Fixtures 产物位置 | `src/__fixtures__/` → 可能进入生产 bundle | `src/dev/` + `import.meta.env.DEV` + query param 开关 + 构建后 grep 验证 0 字节 | 「包体超 10KB 目标」→ 0 污染 |
| invalid event payload | 塞完整 structured JSON → 膨胀 + 敏感重复 | sha256 + 错误列表 + ≤16KB middle snippet + 回溯路径说明 | 「events.jsonl 变大 / 清理不一致」→ 硬上限 20KB / 典型 <4KB |

---

## 下一步建议

1. **15 分钟快速 review v0.2.2 的 4 个补丁点**：
   - tsx 作为唯一执行器引入（不挂 build 链，§8.1）
   - 4 个 AppendReview 调用点的 ReviewSource 构造策略（§0.2.1）和开放问题 #4 拍板
   - fixtures 通过 `?magereview-fixtures=1` 挂载（§8.3.2）+ 生产 grep 验证（§8.3.4）
   - structured_invalid event payload 结构和截断策略（§8.4）
2. 过了 → 我直接开 Phase 1 落地（1.5 天：3 纯函数 TS 脚本 + 4 组件 + 6 dev fixtures + FinalVerdictCard）。Phase 1.5 等后端同步（ReviewSource 5 字段 + structured 分层校验 + invalid event 3 字段写入）。
