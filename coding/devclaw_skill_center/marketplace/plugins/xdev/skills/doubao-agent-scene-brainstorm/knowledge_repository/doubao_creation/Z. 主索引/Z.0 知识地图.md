# Z.0 知识地图（doubao-agent-scene-brainstorm）

> 这是 knowledge_repository 的**总入口**。当你不知道从哪个文件开始读，看这里。

---

## 一、文件树（全部）

```
knowledge_repository/doubao_creation/
├── A. 组织职责与找人地图/
│   ├── 团队组织.md                 ← PM / 工程 owner 速查表
│   └── 误区纠正.md                 ← 8 条新人最容易踩的误区
├── B. Agent 架构专栏/
│   ├── 1. 业务架构与场景全景.md     ← 13 条创作动线总览（C 类目子集）
│   ├── 2. 工程架构演进史.md         ← 5 代演进 + phase3 / agent_phase_26 命名来历
│   ├── 3. ReAct 五层架构介绍.md     ← PRD 五层 → 实际目录映射（核心）
│   ├── 4. ContextMessage 4 路扩散.md ← 上下文管理协议
│   └── 5. 新老架构并存策略.md       ← 双轨分析必读
├── C. 创作能力动线专栏/
│   ├── 0. 动线总览.md
│   ├── 文生图/{4 入口}.md           ← 主bot 闲聊（完整）+ 3 骨架
│   ├── 图生图/{3 入口}.md           ← 主bot 闲聊（完整）+ 2 骨架
│   ├── AI修图/{1 入口}.md           ← 骨架
│   ├── 分身写真/{3 入口}.md         ← 推理 主bot 闲聊（完整）+ 2 骨架
│   ├── 文生视频_图生视频/{1 入口}.md ← 主bot 指令集（完整）
│   └── 文生音乐/{1 入口}.md         ← 骨架
├── D. 中间件_基础设施专栏/
│   ├── Fornax.md                   ← Prompt 配置中心
│   ├── TCC.md                      ← 配置中心
│   ├── Abase.md                    ← KV / 短期 Memory
│   ├── VikingDB.md                 ← 向量库 / 长期 Memory + RAG
│   ├── Libra.md                    ← 实验切流平台
│   ├── ImageX_SmartPlayer.md       ← 多媒体资源
│   └── flow_ocerr.md               ← 错误码（PRD 计划 vs 当前 Gap）
├── E. 关键术语_黑话词典/
│   └── glossary.md                 ← 91 条术语速查
└── Z. 主索引/
    ├── Z.0 知识地图.md              ← 本文件
    ├── Z.1 PSM 索引.md
    ├── Z.2 PRD vs 实现 Gap 表.md
    └── Z.3 痛点应对地图.md          ← 访谈痛点 → skill 应对位置的映射
```

---

## 二、按 RD 角色推荐入门读法

### 2.1 新人（第一次接触豆包-创作 agent）

```
Day 1：
1. A.团队组织.md
2. A.误区纠正.md（先把 8 个误区记住）
3. B.1 业务架构与场景全景.md
4. B.2 工程架构演进史.md（理解为什么有 phase3 / agent_phase_26）

Day 2：
5. B.3 ReAct 五层架构介绍.md
6. B.5 新老架构并存策略.md（关键）
7. ../locators/psm_locator.md（最常用的 locator）

Day 3：
8. E.glossary.md（按需查）
9. 选一个命中本期 PRD 的 C 动线读完整版
10. ../locators/libra_locator.md（P0 痛点）
```

### 2.2 老人（接到一个新需求 brainstorm）

```
1. 走 SKILL.md 三阶段流程
2. 阶段一调 skill_request_brainstorm 时引用 A. + B.1 + C 动线
3. 阶段二调 skill_link_explore 时引用 5 个 locator + B.3/4/5 + D 中间件
4. 阶段三调 skill_solution_alignment 时引用 Z.2 Gap 表
```

### 2.3 工具开发者（要给本 skill 改东西）

```
1. SKILL.md（顶层入口）
2. guidance.md（执行纲领）
3. report_template.md（输出模板）
4. 各 sub-skill 的 SKILL.md
5. 5 个 locator
6. 维护本目录（knowledge）
```

---

## 三、按痛点维度速查

> **完整痛点 → skill 应对映射** 见 [`Z.3 痛点应对地图.md`](./Z.3%20痛点应对地图.md)。

| 用户痛点 | 看哪 |
|---|---|
| 屎山代码 / 隐藏知识 | A.误区纠正 + B.5 新老并存 + C 动线（看本期改造） |
| Agent → Tool → Model 链路分散 | 5 个 locator（特别是 PSM + Tool + Block） |
| Libra 切流散落 | locators/libra_locator.md（**P0 核心**） |
| SP 加载 4 层混乱 | locators/sp_locator.md + D.Fornax + D.TCC |
| 历史包袱 / 隐藏知识口口相传 | A.团队组织（找人）+ E.glossary（黑话）+ Z.2 Gap |
| 改 Tool 不知道在哪 | locators/tool_locator.md |
| 改 SP 不知道在哪 | locators/sp_locator.md |
| 改 Block 不知道在哪 | locators/block_locator.md |
| 多人协作漏同步 | brainstorm 报告 §6.2 业务风险 + A.团队组织（owner 清单） |
| CoCo 输出价值打折 | 用 5 个 locator 校准 CoCo 通用输出（置信度机制） |

---

## 四、最近一次同步

- **同步日期**：2026-04-27
- **同步代码 commit**：见各 locator 顶部的 `last_synced_commits`
- **同步分支**：见各 locator 顶部的 `verified_against_branch`

> 如果 RD 在用 skill 时发现 knowledge / locator 与代码不一致，**优先信代码**，并向本 skill 维护者反馈。
