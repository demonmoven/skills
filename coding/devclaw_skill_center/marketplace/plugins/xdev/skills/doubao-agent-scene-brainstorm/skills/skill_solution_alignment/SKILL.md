---
name: skill_solution_alignment
description: "豆包-创作 agent 链路 brainstorm 阶段三：基于 link_analysis 整合输出 brainstorm_core_findings.md，作为 RD 写技术方案前的决策辅助报告。**严禁代写技术方案**，只输出方向与风险。"
---

<!-- @format -->

# skill_solution_alignment — 阶段三：方案对齐（brainstorm 核心发现）

本 sub-skill 整合 `skill_link_explore` 产出的 link_analysis，按 [`../../report_template.md`](../../report_template.md) 的 7 章节输出**brainstorm 核心发现报告**。

## 与 ecom-buy `skill_solution_design` 的本质差异

| ecom-buy | 本 skill |
|---|---|
| 输出**技术方案文档**（含 diff / 接口 QPS / 改动点坐标） | 输出 **brainstorm 核心发现**（含方向 / 风险 / Gap） |
| 给 RD 拿走交付 | 给 RD 拿走**自己写技术方案** |
| 强制代码片段 | **严禁代码片段** |
| 强制 file:line 坐标 | 强制流程图 + 清单 |

## 输入

- `workspace/user/story.md`
- `workspace/user/link_analysis-*.md`（每个功能点一份）

## 前置条件

- 阶段一 + 二都已完成
- 所有功能点都有 link_analysis

## 输出

- `workspace/artifacts/research/brainstorm_core_findings.md`（决策 Q4 命名）

## 必读资源

- [`./resources/alignment_writing_guide.md`](./resources/alignment_writing_guide.md)（写作指导）
- [`../../report_template.md`](../../report_template.md)（7 章节模板）

---

## 执行步骤

### 第一步：理解现状与需求

1. 读 `workspace/user/story.md`：理解需求功能点 + 复杂度
2. 读所有 `workspace/user/link_analysis-*.md`：理解每个功能点的链路探索结论
3. **不重新做链路分析**（阶段二已做完）

### 第二步：识别共性发现

跨功能点找共性：

- 多个功能点都触达 SP regression？
- 多个功能点都需要新 Libra 实验？
- 多个功能点都改 Tool schema？
- 多个功能点都涉及 Block 协议改动？

把共性发现整合到报告 § 五"三处分散点盘点"和 § 六"Gap 与风险"。

### 第三步：对照 Gap 表

强制读取 [`../../knowledge_repository/doubao_creation/Z. 主索引/Z.2 PRD vs 实现 Gap 表.md`](../../knowledge_repository/doubao_creation/Z.%20主索引/Z.2%20PRD%20vs%20实现%20Gap%20表.md)，**逐条**评估本需求是否撞上：

- G1 多租户隔离 gap
- G2 错误码 flow/ocerr gap
- G3 SP 管理 regression
- G4 DAG 异步转同步 undocumented
- G5 ToolInfo 协议简化 silent_breaking
- G6 工具 schema JSON 化 partial
- G7 上屏 Block 流式块级 undocumented
- G8 Memory 大 key 倾向 partial
- G9 RAG / Skills gap

把命中的 Gap 列入报告 § 六.1。

### 第四步：写"方案方向建议"（**最关键**）

参考 `report_template.md` § 七：

- 改动定位建议（应该改 X 模块，不应该改 Y 模块）
- 扩展机制建议（新增 Tool 走 schema 注册 / 新增 SP 走 Fornax / 灰度走 Libra…）
- 必须验证的事项（让 RD 自检的 checklist）
- 待用户决策的问题（如有）

**严禁**：
- ❌ "我建议把 foo.go::bar() 第 N 行改成..."
- ❌ diff 代码片段
- ❌ "请用 X 框架而不是 Y"（除非有明确依据）

**应该**：
- ✅ "本次改动应主要落在 aigc_dag/biz/handler_sync/doubao/text2image.go，避免散到 creation_agent 主链路"
- ✅ "建议走 Libra 实验（短期 A/B），不要走 TCC（长期固化）"
- ✅ "❗ 触达 SP regression，请在 3 处分散代码都验证"

### 第五步：生成 brainstorm_core_findings.md

按 `report_template.md` 7 章节模板生成：

1. **需求复述**：从 story.md 取
2. **入口与代际定位**：从 link_analysis §1 取，整合
3. **命中动线**：从 link_analysis §2 取
4. **链路探索结论**：从 link_analysis §3-§7 取，按功能点分小节
5. **三处分散点盘点**：跨功能点综合（SP / Libra / Tool）
6. **Gap 与风险**：含 Z.2 Gap 表对照 + 业务风险 + 测试风险
7. **方案方向建议**：见第四步

### 第六步：质量检查

按下面清单逐条检查。

---

## 质量检查清单

输出前必须检查：

**独立性**：
- [ ] 报告**不引用** `workspace/user/` 下中间产物（独立可读）
- [ ] 所有内容已整合
- [ ] 关联中间产物只在"附录"以"追溯"形式列出

**完整性**：
- [ ] 7 章节都有内容
- [ ] 所有功能点都有对应章节
- [ ] 复杂度 / 业务线 / 子方向已标注

**写作风格**（强制）：
- [ ] **严禁代码片段**（diff / 伪代码 / 文字代码描述都不可）
- [ ] **严禁文件路径 + 行号堆砌**（用流程图 + 清单替代）
- [ ] **严禁"我建议这样写代码"**（只能给方向）
- [ ] **严禁字母 A/B/C 编号改动点**（用功能点命名）
- [ ] 中文一二三四编号
- [ ] ≥ 3 张 mermaid（时序图 / 调用拓扑 / SP 树 / Block 序列等）

**专项检查**：
- [ ] **SP 触达检查表**已列（4 层 + 3 处分散）
- [ ] **Libra 7 处盘点表**已列（即使无关也明示）
- [ ] **Tool 4 级链路**至少 1 个完整
- [ ] **Z.2 Gap 表对照**已显式
- [ ] **方案方向建议**至少 3 条

**业务风险**：
- [ ] 隐藏依赖已识别
- [ ] 新老并存窗口影响已评估
- [ ] 数据兼容已评估
- [ ] 强弱依赖变更已列

---

## 错误处理

- 输入文件缺失 → 提示先执行阶段一 / 二
- 链路分析不充分 → 反馈给阶段二补充，**不要硬编 brainstorm**
- 涉及 scope 外（社区 / 海外）→ 明示"建议改用对应 skill"
- 用户希望"代写方案" → 礼貌拒绝，引导回 brainstorm 报告价值

---

## 与 ecom-buy 的关键差异（再强调）

| 维度 | ecom-buy solution_design | doubao solution_alignment |
|---|---|---|
| 输出文档名 | conclusions.md | **brainstorm_core_findings.md** |
| 是否含 diff 代码 | ✅ 强制 | **❌ 严禁** |
| 是否含 file:line | ✅ 强制 | **❌ 严禁** |
| 章节数 | 4-5（按模板） | **7 固定章节** |
| 强制 Gap 对照 | ❌ | **✅ 强制** |
| 强制散落点盘点 | ❌ | **✅ 强制（SP/Libra/Tool）** |
| 读者动作 | 拿走交付 | 拿走 **自己写方案** |
