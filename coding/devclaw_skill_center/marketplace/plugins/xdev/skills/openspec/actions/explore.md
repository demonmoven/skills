# OpenSpec Explore

探索性对话,不创建 artifact。用于「需求模糊」「想先理清思路」「比较多个方案」的场景。

## 参数

- `ARG`(可选):topic 或问题(自由文本)

---

## 执行流程

> **`<skill_dir>` 约定**:以下路径中 `<skill_dir>` 指代本 action 所属的 openspec skill 目录的绝对路径。

### Step 1:理解 topic

#### 1a. 有参数

`TOPIC = ARG`,直接进入 Step 2。

#### 1b. 无参数

提示用户:

> "想探索什么主题?(例如:'要不要加一个 dark mode'、'比较 OAuth vs JWT'、'重构思路')"

收到用户输入作为 `TOPIC`。

### Step 2:开放式对话

按以下方式与用户讨论:

1. **澄清问题**:用 AskUserQuestion 工具帮用户澄清核心诉求(but 一次问一个问题,不要轰炸)
2. **提出多个方案**:列举 2-3 个可行方案,各自的优缺点 / 取舍
3. **指出风险**:每个方案的潜在风险、未知数、决策点
4. **不写入文件**:Explore 不创建任何 artifact 文件,**纯对话** 模式

> **关键约束**:不要写入 `docs/xdev/openspec/changes/` 下的任何文件。如果用户已经决定要做某件事,引导用户运行 `/openspec propose <需求描述>` 进入正式 propose 流程。

### Step 3:转换出口

对话结束时,根据用户态度提供出口:

- 用户已经有清晰方向 → 建议 `/openspec propose <你的需求>`
- 用户还在犹豫 → 建议继续 explore 或先写一份手工的 `docs/xdev/openspec/changes/<name>/proposal.md` 草稿
- 用户决定不做 → 输出"OK,本次探索结束"

输出示例:

```
🔍 Explore 结束

讨论主题:{TOPIC}
关键发现:
  - {要点 1}
  - {要点 2}
  - {要点 3}

下一步建议:
  /openspec propose {建议的需求描述}    进入正式 propose 流程
  /openspec explore                     继续探索
```

---

## Guardrails

- 不创建任何 artifact 文件
- 一次问一个澄清问题,避免轰炸用户
- 列方案时给出客观比较,不要倾向性太强
- 如果用户明确说"开始做",立即引导到 `/openspec propose`
