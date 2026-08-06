**⚠️ 本指令仅针对服务端（backend）进行技术设计，包括业务逻辑代码、IDL（Contracts）、Docker 部署等。不要涉及任何前端（frontend/web/UI）相关内容。**

---

# mining_diff 指令（Mining 差异化发现提取）

> 对比 Mining 文档与 Analyze、Research 文档，提取 Mining 中独特的、未被 Analyze/Research 覆盖的发现

## 输入文档（documents preparation）

> **变量路径说明**：以下变量通过 system prompt 注入，值为绝对路径。

1. Spec 文档：SPEC_DOC
   - 对需求的详细描述，也是功能实现的目标，具有严格的结构化表达模式
2. 代码库现状分析文档：ANALYZE_DOC
   - 前序流程产物，已结合 SPEC_DOC 对当前代码库中需求相关代码进行了分析和探索
3. 技术方案调研文档：RESEARCH_DOC
   - 前序流程产物，基于 SPEC_DOC、TECH_GUIDANCE_DOC 充分调研后的结果
4. 隐性需求挖掘文档：MINING_DOC
   - 前序流程产物，通过分层代码分析挖掘出的隐藏技术需求

## 输出

1. Mining 差异化文档：MINING_DIFF_DOC，文档名必须为 "mining-diff"

---

**重要规则**：
- **保留 MINING_DOC 的原始文档格式**：输出文档的结构、标题层级、章节组织方式必须与 MINING_DOC 保持一致
- 只提取 MINING_DOC 中**独特的、在 ANALYZE_DOC 和 RESEARCH_DOC 中未覆盖的发现**
- 对于 MINING_DOC 中与 ANALYZE_DOC / RESEARCH_DOC 重叠的内容，**直接剔除**，不要保留
- 如果某个章节的所有发现都与 ANALYZE_DOC / RESEARCH_DOC 重叠，则保留该章节标题但注明"无独特发现"

---

## 执行流程

### 步骤 1：阅读输入文档

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 阅读输入文档..."

* 阅读 SPEC_DOC 理解需求范围
* 阅读 ANALYZE_DOC 理解代码库现状分析结果
* 阅读 RESEARCH_DOC 理解技术调研结论
* 阅读 MINING_DOC 理解隐性需求挖掘结果

### 步骤 2：对比分析

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 对比分析 Mining 与 Analyze/Research 的差异..."

对 MINING_DOC 中的每一项发现，逐一对比 ANALYZE_DOC 和 RESEARCH_DOC：
- **重叠判断标准**：如果 MINING_DOC 中的某项发现在 ANALYZE_DOC 或 RESEARCH_DOC 中已有**实质相同的描述或结论**（即使措辞不同），则视为重叠
- **独特判断标准**：MINING_DOC 中的发现如果提供了 ANALYZE_DOC 和 RESEARCH_DOC 中**完全未提及的视角、约束、风险或技术细节**，则视为独特发现

### 步骤 3：输出 Mining Diff 文档

1. 向用户报告 MESSAGE 提示 "【当前状态】开始 >> 输出 Mining Diff 文档..."

将差异化分析结果按照 MINING_DOC 的**原始文档格式**输出到 MINING_DIFF_DOC：
- 保持与 MINING_DOC 相同的章节结构和标题层级
- 每个章节只保留独特的发现
- 对于无独特发现的章节，保留标题并注明 `> 本章节发现已在 Analyze/Research 文档中覆盖，无独特发现。`
- 在文档开头添加一个"差异化摘要"章节，汇总本文档提取出的独特发现数量和分布

2. 向用户报告 MESSAGE 提示 "【当前状态】完成 >> Mining 差异化发现提取，并生成 Mining Diff 文档"
