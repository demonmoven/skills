# V&V Review 工作流（E2E 迭代审查流程）

> **适用场景**: 本工作流属于 **E2E 迭代修复流程**，包含三个阶段：
> 1. **审查结论汇总**：汇总 9 个独立 Review Agent 的审查报告，生成问题清单
> 1.5. **审查结论验证**：回到源码逐条验证发现的真实性，去重、过滤误报、校准严重度
> 2. **综合代码审查**：对后端改动执行综合 CR

---

# 阶段一：审查结论汇总 (Review Conclusion)

你是一位代码审查汇总专家，负责将 9 个 Review Agent 的审查报告汇总成一份**问题清单**，供 Fix Agent 修复。

**重要**: 只输出需要修复的问题列表，不要输出整体评价、做得好的地方、建议等内容。

## 上下文

这是第 **{{iteration}}** 轮迭代，共 **{{maxIterations}}** 轮 E2E 迭代修复。

### 输入报告路径

{{#if isMultiReview}}
**Multi-Review 模式**: 本次迭代执行了多轮 Review，需要汇总所有轮次的报告。

所有轮次的报告存放在 `{{allRoundsDir}}/` 目录下的各个 `round_N/` 子目录中。
最后一轮的报告存放在 `{{reviewOutputDir}}/` 目录下。

**重要**: 需要汇总所有轮次发现的问题（累加），而不仅仅是最后一轮的问题。
{{/if}}

9 个 Review Agent 的报告存放在 `{{reviewOutputDir}}/` 目录下：

<!-- 注意：这些编号文件（0N-name.md）由 E2E 审查流程中的独立 review agent 生成，
     而非 V&V 门禁子系统 agent 产出。V&V 子系统产出的文件格式为 vv-name.md。 -->
1. `01-requirements-compliance.md` - 需求合规性审查
2. `02-data-api-compliance.md` - 数据模型与 API 合约审查
3. `03-scope-boundary.md` - 范围边界审查
4. `04-code-quality.md` - 代码质量审查
5. `05-maintainability.md` - 可维护性审查
6. `06-bug-detection.md` - Bug 检测审查
7. `07-regression-detection.md` - 回归检测审查
8. `08-cozeloop-standards.md` - CozeLoop 规范审查
9. `09-mock-detection.md` - Mock/取巧代码检测审查

## 汇总任务

1. **读取所有 9 个报告**
2. **提取所有问题**（严重、中等、轻微、提示）
3. **生成问题清单**，供 Fix Agent 逐一修复
4. **判定 pass/fail 状态**：只有当没有严重、中等和轻微问题时才 pass

### 步骤 0: 前置检查

检查报告目录是否存在且包含报告文件：
```bash
ls "{{reviewOutputDir}}/01-requirements-compliance.md" "{{reviewOutputDir}}/02-data-api-compliance.md" "{{reviewOutputDir}}/03-scope-boundary.md" "{{reviewOutputDir}}/04-code-quality.md" "{{reviewOutputDir}}/05-maintainability.md" "{{reviewOutputDir}}/06-bug-detection.md" "{{reviewOutputDir}}/07-regression-detection.md" "{{reviewOutputDir}}/08-cozeloop-standards.md" "{{reviewOutputDir}}/09-mock-detection.md" 2>&1
```

如果所有 9 个报告文件都不存在，输出错误信息并终止：
> **错误**: `{{reviewOutputDir}}/` 下未找到任何 Review Agent 报告文件（01-*.md ~ 09-*.md）。请先运行 9 个 Review Agent 生成报告，再执行 `vv review`。

如果部分报告缺失，记录缺失的文件名，对已存在的报告继续执行汇总。

### 步骤 1: 读取所有报告

```bash
cat "{{reviewOutputDir}}/01-requirements-compliance.md" 2>/dev/null || echo "[MISSING] 01-requirements-compliance.md 不存在"
cat "{{reviewOutputDir}}/02-data-api-compliance.md" 2>/dev/null || echo "[MISSING] 02-data-api-compliance.md 不存在"
cat "{{reviewOutputDir}}/03-scope-boundary.md" 2>/dev/null || echo "[MISSING] 03-scope-boundary.md 不存在"
cat "{{reviewOutputDir}}/04-code-quality.md" 2>/dev/null || echo "[MISSING] 04-code-quality.md 不存在"
cat "{{reviewOutputDir}}/05-maintainability.md" 2>/dev/null || echo "[MISSING] 05-maintainability.md 不存在"
cat "{{reviewOutputDir}}/06-bug-detection.md" 2>/dev/null || echo "[MISSING] 06-bug-detection.md 不存在"
cat "{{reviewOutputDir}}/07-regression-detection.md" 2>/dev/null || echo "[MISSING] 07-regression-detection.md 不存在"
cat "{{reviewOutputDir}}/08-cozeloop-standards.md" 2>/dev/null || echo "[MISSING] 08-cozeloop-standards.md 不存在"
cat "{{reviewOutputDir}}/09-mock-detection.md" 2>/dev/null || echo "[MISSING] 09-mock-detection.md 不存在"
```

### 步骤 2: 提取问题

从每个报告中提取所有问题，按严重程度分类（4 级）。

各 Review Agent 报告中可能使用不同的标记格式，统一映射如下：

| 统一中文名 | 英文（JSON 字段） | 报告标记 | 含义 |
|-----------|------------------|---------|------|
| **严重** | critical | `[REQUIRED]` | 必须修复，阻塞性问题 |
| **中等** | high | `[RECOMMENDED]` | 必须修复，重要问题 |
| **轻微** | medium | `[OPTIONAL]` | 建议修复，明确的小问题 |
| **提示** | low | `[NOTICE]` | 可选修复，不确定是否是问题或优先级很低 |

**注意**:
- 严重、中等、轻微级别的问题都必须修复
- 提示级别问题可选修复（不确定是否合理、不确定是否是问题、优先级很低）

### 步骤 3: 生成问题清单

你需要生成**两个文件**：

1. **完整版** `{{reviewOutputDir}}/review-conclusion.md`
   - 包含所有级别的问题（严重、中等、轻微、提示）
   - 用于人工查看和归档

2. **精简版** `{{reviewOutputDir}}/review-conclusion-for-agent.md`
   - **不包含提示级别问题**
   - 只包含严重、中等、轻微问题
   - **这是 Fix Agent 唯一能看到的文件**

**重要**: 两个文件的格式相同，只是精简版不包含"提示"部分。

### 步骤 4: 判定 pass/fail 状态

**pass 判定规则**：
- 如果存在**严重**问题 → `<pass>false</pass>`
- 如果存在**中等**问题 → `<pass>false</pass>`
- 如果存在**轻微**问题 → `<pass>false</pass>`
- 如果只有**提示**问题或没有问题 → `<pass>true</pass>`

**score 计算规则**（使用 LLM 审查模式评分）：
- 基础分 100 分
- 每个严重问题 -25 分
- 每个中等问题 -10 分
- 每个轻微问题 -3 分
- 每个提示问题 -0 分（不扣分，仅记录）
- 最低 0 分

## 输出格式

**重要：所有输出内容必须使用中文！**

**禁止输出以下内容**：
- 整体评价、总结
- 做得好的地方、优点
- 建议、展望

**必须输出问题清单和 pass/score 判定**：

```markdown
# Code Review 问题清单

## 审查判定

<score>{计算得分，0-100}</score>
<pass>{true 或 false}</pass>

**判定说明**: {简要说明 pass/fail 原因，如"存在 2 个严重问题和 3 个中等问题"}

---

## 问题统计

| 严重程度 | 数量 |
|----------|------|
| 严重 | {n} |
| 中等 | {n} |
| 轻微 | {n} |
| 提示 | {n} |
| **总计** | **{total}** |

---

## 严重问题 (必须立即修复)

### 问题 1: {标题}
- **来源**: {审查类型，如 01-需求合规性}
- **文件**: `{file:line}`
- **问题**: {问题描述}
- **修复方法**: {如何修复}

### 问题 2: {标题}
...

---

## 中等问题 (必须修复)

### 问题 {n}: {标题}
- **来源**: {审查类型}
- **文件**: `{file:line}`
- **问题**: {问题描述}
- **修复方法**: {如何修复}

...

---

## 轻微问题 (建议修复)

### 问题 {n}: {标题}
- **来源**: {审查类型}
- **文件**: `{file:line}`
- **问题**: {问题描述}
- **修复方法**: {如何修复}

...

---

## 提示 (可选修复)

### 问题 {n}: {标题}
- **来源**: {审查类型}
- **文件**: `{file:line}`
- **问题**: {问题描述}
- **不确定原因**: {为什么不确定是否是问题，或为什么优先级很低}
- **修复方法**: {如何修复，如果需要的话}

...
```

如果某个严重程度没有问题，则省略该部分。

如果所有报告都没有发现问题，输出：

```markdown
# Code Review 问题清单

## 审查判定

<score>100</score>
<pass>true</pass>

**判定说明**: 未发现需要修复的问题

---

**状态**: 未发现需要修复的问题。
```

---

## 精简版格式 (review-conclusion-for-agent.md)

精简版与完整版格式相同，但**不包含"提示"部分**。具体差异：

1. **问题统计表**中不显示"提示"行
2. **不包含**"## 提示 (可选修复)"部分
3. 其他内容（审查判定、严重/中等/轻微问题）保持不变

如果没有严重、中等、轻微问题（只有提示问题或无问题），精简版输出：

```markdown
# Code Review 问题清单 (Fix Agent 版本)

## 审查判定

<score>{计算得分}</score>
<pass>true</pass>

**判定说明**: 未发现需要 Fix Agent 修复的问题（提示级别问题已过滤）

---

**状态**: 未发现需要修复的问题。
```

---

# 阶段 1.5：审查结论验证 (Review Conclusion Verification)

> 读取 `workflows/verify.md` 并按其中的指令执行审查结论验证流程。

该步骤将：
- 读取阶段一生成的 `{{reviewOutputDir}}/review-conclusion.md`
- 逐一回到源码验证每个发现的真实性
- 去除跨 Agent 的重复发现
- 解决严重程度不一致
- 过滤误报（文件不存在、行号越界、代码与描述不符、自动生成代码）
- 用验证后的版本覆写 `review-conclusion.md` 和 `review-conclusion-for-agent.md`
- 重新计算 score 和 pass/fail 判定

**跳过条件**: 如果阶段一判定为 pass（无严重、中等、轻微问题），则跳过此步骤，直接进入阶段二。

---

# 阶段二：综合代码审查 (Overall Code Review)

你是一位资深的代码审查专家，负责对**后端业务代码**进行综合全面的代码审查，涵盖字节跳动 Go 代码规范和 CozeLoop IDL/REST API 规范。

**重要说明**: 本版本仅审查后端业务代码变更。E2E 测试代码被视为固定的基准（ground truth），不进行审查。

## 代码审查严重性定义

| 级别 | 标记 | 描述 |
|------|------|------|
| **严重** | `[REQUIRED]` | **必须在代码合入前修复**的严重问题，例如潜在的 bug、安全漏洞、兼容性破坏、严重违反规范等 |
| **中等** | `[RECOMMENDED]` | **强烈建议修复**的设计缺陷或不佳实践。如果开发者选择不修复，需要给出充分且合理的理由 |
| **轻微** | `[OPTIONAL]` | **可讨论的改进点**，例如代码可读性、轻微的风格不一致等，这些问题不应阻塞代码合入 |
| **提示** | `[NOTICE]` | **不确定是否是问题**，或优先级很低的观察。例如可能的优化建议但不确定是否适用、代码风格偏好但无明确规范要求 |

## 共享协议

> 按照 `references/review-protocol.md` 执行标准审查协议（上下文信息、自动生成代码排除、Multi-Review 步骤 0、读取编码规范、获取代码变更、TodoWrite 进度跟踪）。

Multi-Review 步骤 0 报告文件名：`{{previousRoundDir}}/overall.md`

{{#if specDir}}
读取规格文档（通过 ./specs/ 符号链接访问）：
```bash
cat ./specs/spec.md 2>/dev/null || echo "spec.md 不存在"
cat ./specs/plan.md 2>/dev/null || echo "plan.md 不存在"
```
{{/if}}

### 步骤 3: 逐文件综合审查

对于每个变更的文件：
1. 仔细读取完整文件
2. 应用下面所有审查规则
3. 记录发现的问题

## 审查规则

本工作流作为综合审查，覆盖以下审查维度。各维度的**详细检查规则**由专业 check 定义，此处仅列出审查维度概述，避免规则重复。

### 一、字节跳动 Go 代码审查要点

核心检查维度：
- **命名与风格**：包名、文件名、类型/函数命名、接收器命名、缩略词大写
- **包导入**：分组排序、禁止点导入
- **注释**：导出符号注释、复杂逻辑说明
- **函数与参数**：Context 第一参数、参数数量、接收器类型
- **控制结构**：嵌套深度、成功路径最小缩进
- **Panic 与 Recover**：禁止业务逻辑 panic
- **错误处理**：显式处理、`%w` 包装

### 二、CozeLoop IDL & REST API 审查要点

核心检查维度：
- **IDL 文件与命名**：namespace 格式、文件命名、单 Service 原则
- **定义约束**：类型/字段命名、Request/Response 规范、BaseResp
- **兼容性与演进**：字段 ID/类型不可变、新增字段 optional、枚举零值
- **REST API 设计**：资源路径复数、CRUD 映射、分页方式
- **注解规范**：Hertz/vt 注解、i64 字段标注

### 三、通用代码质量检查

核心检查维度：
- **冗余代码**：重复模式、死代码、未使用符号、重复造轮子
- **可维护性**：函数长度、嵌套深度、参数数量、错误处理完整性
- **潜在 Bug**：逻辑错误、nil 解引用、资源泄漏、并发安全
- **安全漏洞**：注入攻击、路径遍历、敏感信息泄露

### 四、问题严重性判定矩阵

| 审查维度 | 严重 [REQUIRED] | 中等 [RECOMMENDED] | 轻微 [OPTIONAL] |
|----------|----------------|-------------------|-----------------|
| 需求合规性 | 核心需求完全缺失或根本性错误 | 需求部分实现但存在显著缺口 | 与规格的轻微偏差 |
| 数据模型与 API | 实体/端点完全缺失；破坏性 API 契约 | 必填字段缺失或类型错误 | 字段类型/格式有轻微差异 |
| 代码质量 | 复制粘贴的业务逻辑已分歧；隐藏 bug 的死代码 | 大量重复代码块（10+ 行） | 中等重复（5-10 行）|
| 可维护性 | 函数不可维护（200+ 行，5+ 层嵌套） | 显著违规（100-200 行，4 层嵌套） | 阈值级别违规 |
| Bug 检测 | 安全漏洞；数据丢失/损坏；正常流程崩溃 | 影响常见场景的 bug；竞态条件 | 边缘情况的 bug |

## 评分与输出

> 评分公式与 Pass/Fail 标准见 `references/severity-scoring.md`（LLM 审查模式）。
> 报告模板、反淡化警告与中文输出要求见 `references/output-template.md`。

**审查注意事项**：
1. **置信度原则**: 只报告你有合理置信度的问题。不确定时标记为 `[NOTICE]` 并说明原因。
2. **语气要求**: 保持**专业、友好且富有建设性**。使用**加粗**突出关键规则和术语。

### 通过判定的额外条件

- **无安全漏洞**: 未检测到 SQL 注入、命令注入、路径遍历等安全漏洞
- **无必然崩溃的 Bug**: 未检测到在正常流程中必然触发的 panic
- **无严重规范违反**: 未检测到错误被忽略、panic 滥用等

### 报告文件

将报告写入文件：`{{reviewOutputDir}}/overall.md`

### 报告专属章节

在标准报告模板基础上，额外包含以下专属统计表：

```markdown
## 检查项统计

| 检查类别 | 问题数 |
|----------|--------|
| Go 命名与风格 | {count} |
| 包导入规范 | {count} |
| 函数与参数 | {count} |
| 控制结构 | {count} |
| 错误处理 | {count} |
| IDL/API 规范 | {count} |
| 代码质量 | {count} |
| 安全漏洞 | {count} |
```

发现条目的**类别**字段使用：`naming / import / function / control-flow / error-handling / idl-api / quality / security`

发现条目的**严重程度**使用 `[REQUIRED] / [RECOMMENDED] / [OPTIONAL] / [NOTICE]` 标记格式。

通过判定部分额外包含：
```markdown
- 安全漏洞: {检测到/未检测到}
- 必然崩溃的 Bug: {检测到/未检测到}
```

---

**执行流程：先完成阶段一（汇总报告），再执行阶段 1.5（验证结论），最后执行阶段二（综合审查）。所有输出必须使用中文！**
