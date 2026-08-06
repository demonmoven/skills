# Compound Engineering 用户手册

## 一句话介绍

Compound Engineering（复利工程）——每次工程产出都让下一次更容易。80% 规划复盘，20% 执行，通过 `compound` 步骤把解决方案沉淀为可检索的团队知识。

---

## 核心循环

```
ideate → brainstorm → plan → work → review → compound
  ^                                              |
  └──────── 知识复利，越用越顺 ────────────────────┘
```

你不需要每次都走完整循环。每个 action 独立可用，随时切入。

---

## 快速开始

### 全流程（最推荐）

```
/ce brainstorm 用户反馈列表加载太慢，需要优化
```

Agent 会引导你完成需求探索，产出 requirements.md，然后提示下一步。

### 单步使用

```
/ce plan                    # 从已有 requirements.md 生成实施计划
/ce work                    # 从已有 plan 开始执行
/ce review main             # 对比 main 做 code review
/ce compound                # 刚修完 bug？沉淀一下
/ce ideate                  # 给我一些改进建议
/ce debug 登录接口偶尔 500   # 系统化根因定位
/ce optimize 首页加载速度    # 迭代优化 + 度量门控
```

---

## 各 Action 详解

### brainstorm

交互式 Q&A 探索需求。Agent 会：
1. 评估范围（轻量/标准/深度）
2. 扫描代码库上下文
3. 对需求做压力测试（是否在解决真正的问题？）
4. 协作对话，逐个问题确认
5. 产出 `docs/xdev/ce/brainstorms/YYYY-MM-DD-<topic>-requirements.md`
6. 用 7 个 document-review agent 审阅需求文档

### plan

将需求转化为结构化实施计划。Agent 会：
1. 并行调研（repo-research + learnings-researcher + best-practices）
2. 分析差距，决定是否需要外部调研
3. 解决规划期疑问（只在影响架构/风险时才问用户）
4. 拆分为可执行的 implementation units
5. 置信度检查（自动加强薄弱环节）
6. 产出 `docs/xdev/ce/plans/YYYY-MM-DD-NNN-<type>-<name>-plan.md`

### work

按计划执行开发。Agent 会：
1. 判断输入类型（plan 文件 / 裸 prompt）
2. 创建任务列表 + 选择执行策略（inline / serial subagent / parallel subagent）
3. 每个任务：发现测试 → 实现 → 持续测试 → 增量提交
4. 完成后走 shipping workflow（质量检查 → review → PR）

### review

**CE 最强差异点**——多 persona 并行 Code review。

- **23 个审阅 persona**（6 always-on + 17 conditional）
- 按 diff 内容自动选择相关 persona
- 每个 persona 独立 subagent 并行运行
- **置信度门控**：低于 0.60 的 finding 被过滤（P0 例外，阈值 0.50）
- **自动去重**：同文件同行同标题的 finding 合并
- **分级呈现**：P0（致命）→ P1（高）→ P2（中）→ P3（低）
- **自动修复**：safe_auto 类 finding 自动修复

### compound

**CE 的灵魂**——知识沉淀。

刚修完一个 bug 或解决了一个棘手问题？立即运行 `compound`，趁上下文还在。Agent 会：
1. 并行研究（上下文分析 + 方案提取 + 相关文档查找）
2. 检测与已有 solution 的重叠度（High → 更新已有 / Low → 新建）
3. 写入 `docs/xdev/ce/solutions/<category>/<filename>.md`（含 YAML frontmatter，可检索）
4. 检查可发现性（必要时更新 AGENTS.md）

### ideate

主动发现代码库中的高价值改进点。Agent 会：
1. 3 个并行扫描 agent（代码、知识库、Issue）
2. 3-4 个并行发散 agent（不同视角：用户痛点、反转思维、打破假设、杠杆点）
3. 对抗过滤（淘汰不 grounded 的想法）
4. 产出 Top 5-7 排名建议 → `docs/xdev/ce/ideation/`

### debug

系统化根因定位 + 修复。4 阶段：
1. 复现 → 2. 隔离（二分 / 最小用例） → 3. 根因分析 → 4. 修复 + 回归测试

### optimize

迭代优化，带度量门控。定义优化目标 → 跑基线 → 设计实验 → 执行 → 度量 → 判断是否达标 → 循环。

---

## 产物目录

```
docs/xdev/ce/
├── brainstorms/    # 需求探索文档（brainstorm 产出）
├── plans/          # 实施计划（plan 产出）
├── solutions/      # 知识沉淀（compound 产出，按 category 分子目录）
│   ├── build-errors/
│   ├── runtime-errors/
│   ├── best-practices/
│   └── ...
├── ideation/       # 改进建议（ideate 产出）
└── reviews/        # Code review 记录（review 产出）
```

---

## 与其他 xdev skill 的搭配

| 场景 | 推荐组合 |
|------|---------|
| 小需求快速交付 | `/exec-plan` 代替 brainstorm+plan |
| 重规格后端需求 | `/speckit` 做规格，`/ce review` 做审阅 |
| 修完 bug 沉淀经验 | `/ce compound` |
| 新仓库工程化 | `/harness init` 先装基础设施，再用 `/ce` 开始开发 |
| 想要改进建议 | `/ce ideate` |
