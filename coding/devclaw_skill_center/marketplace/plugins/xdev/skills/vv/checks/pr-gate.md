# PR Gate（PR 级快速门禁）

你负责执行 PR 级别的快速门禁检查——快速、硬性、最小范围。

## 定位

PR 门禁是最快速的验证层，在每次 PR 提交时执行。目标是在 5 分钟内完成检查，拦截最严重的问题。

## 上下文

### 参数
- **TARGET_PATH**: {{TARGET_PATH}} — 待验证代码目录
- **BASE_SHA**: {{BASE_SHA}} — 基准 commit（默认 HEAD~1）
- **OUTPUT_DIR**: {{OUTPUT_DIR}} — 输出目录

## 执行范围

PR 门禁**仅**执行以下检查：

| 序号 | 检查项 | 来源 | 说明 |
|------|--------|------|------|
| 1 | go vet + go build | checks/rule-precheck.md | 编译和静态分析 |
| 2 | Secrets Scan | checks/rule-precheck.md | 硬编码密钥检测 |
| 3 | 禁用依赖检查 | checks/rule-precheck.md | 是否引入黑名单依赖 |
| 4 | 禁止目录检查 | checks/rule-precheck.md | 是否修改了禁止目录 |
| 5 | 范围边界检查 | checks/scope-boundary.md | 是否越界修改 |
| 6 | 回归检测 | checks/regression-detection.md | 是否引入回归 |
| 7 | Mock 检测 | checks/mock-detection.md | 是否引入 mock/fake 代码 |

## 阈值

| 条件 | 阈值 |
|------|------|
| 最低分 | score >= 70 |
| Critical findings | 零容忍 |
| High findings | 允许（但扣分） |
| Medium findings | 允许（但扣分） |
| Low findings | 允许（不扣分） |

## 你的任务

### Step 1: 收集子系统报告

> **前置依赖**: 本检查在 `subsystems/regression-gater.md` 子系统内作为 Step 2 执行。

读取以下文件：

```bash
# 由上游 rule-checker 子系统产出
cat {{OUTPUT_DIR}}/subsystem_1_rule_checks/summary.json 2>/dev/null
# 由本子系统的 Step 1 regression-detection 产出
cat {{OUTPUT_DIR}}/subsystem_5_regression_gates/vv-regression-detection.md 2>/dev/null
```

### Step 2: 评估门禁结果

从 summary.json 中提取：
1. 分数是否 >= 70
2. 是否存在 critical findings
3. 回归检测结果

### Step 3: 生成门禁结果

写入 `{{OUTPUT_DIR}}/gate-result.md`：

```markdown
# PR Gate 门禁结果

## 门禁状态: {PASS / FAIL}

## 检查结果

| 检查项 | 状态 | 说明 |
|--------|------|------|
| go vet + build | {PASS/FAIL} | {说明} |
| Secrets Scan | {PASS/FAIL} | {说明} |
| 禁用依赖 | {PASS/FAIL} | {说明} |
| 禁止目录 | {PASS/FAIL} | {说明} |
| 范围边界 | {PASS/FAIL} | {说明} |
| 回归检测 | {PASS/FAIL} | {说明} |
| Mock 检测 | {PASS/FAIL} | {说明} |

## 分数: {score}/100

## 阻塞性问题

{如果有 critical findings，列出详情}

## 裁决

- **verdict**: {PASS / FAIL}
- **reason**: {裁决原因}
```

## 评分标准

> 使用**门禁模式评分**。

基础分 100 分：

| 问题级别 | 扣分 |
|----------|------|
| critical | 每个 -25 分 |
| high | 每个 -10 分 |
| medium | 每个 -5 分 |
| low | 每个 -0 分 |

**PASS 条件**：score >= 70 且零 critical findings。
