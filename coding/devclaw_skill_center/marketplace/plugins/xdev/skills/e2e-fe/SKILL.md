---
name: e2e-fe
description: "前端 E2E 测试引擎 - 用例生成与执行全流程。支持 gen-test-case / run。当用户需要生成 E2E 测试用例、执行端到端测试、从功能规格生成测试、验证前端页面时调用。"
argument-hint: "[action] [参数...]"
---

# e2e-fe

前端 E2E（End-to-End）测试引擎，统一入口。支持从功能规格说明书自动生成测试用例，以及使用 Chrome DevTools 执行端到端验证。

## 用户输入

```text
$ARGUMENTS
```

## 参数格式

`$ARGUMENTS` 格式：`[action] [args...] [参数...]`

第一个词为 `action`（必需），决定执行哪个步骤。后续分为两部分：
- `args...`：用户透传的 args 内容，会无损透传到所有 action 内
- `[参数...]`：该 action 的业务参数

所有的操作和参数优先解析用户的 args 内容。

| action | 后续参数 | 说明 | 类型 |
|--------|---------|------|------|
| `gen-test-case` | `<FeatureSpecPath> [Priority]` | 从 feature_spec.md 生成 E2E 测试用例（YAML 格式） | 自动化 |
| `run` | `<TestCases> [Options...]` | 使用 Chrome DevTools 执行 E2E 测试用例 | 自动化 |

---

## 触发关键词

以下关键词应触发本 skill：
- `e2e-fe` / `e2e` / `端到端测试`
- `gen-e2e-test-case` / `生成E2E用例` / `生成测试用例` / `从功能规格生成测试` / `批量生成E2E case`
- `e2e-run` / `e2e run` / `跑E2E` / `run e2e` / `验证用例` / `执行用例` / `前端端到端验证`

---

## 执行方式

> **`<skill_dir>` 约定**：以下路径中 `<skill_dir>` 指代本 SKILL.md 所在目录的绝对路径。

### 1. 解析 action

从 `$ARGUMENTS` 中提取第一个词作为 `action`，剩余部分作为该 action 的参数。

### 2. 路由分派

根据 `action` 的值，读取对应的 action 文件并执行：

```
action = "gen-test-case"   → 读取并执行 <skill_dir>/actions/gen-test-case.md
action = "run"             → 读取并执行 <skill_dir>/actions/run.md
其他                        → 输出下方的帮助信息
```

### 3. 执行 action

读取对应的 `actions/{action}.md` 文件后：

1. 将文件内容作为当前任务的执行指令
2. 文件中所有 `<skill_dir>` 均指向本 SKILL.md 所在目录
3. 将剩余参数（去掉 action 后的部分）传递给该 action 的执行逻辑

### 4. 帮助信息（action 未匹配时输出）

```
e2e-fe - 前端 E2E 测试引擎

用法：e2e-fe [action] [参数...]

可用 actions：
  gen-test-case              从 feature_spec.md 生成 E2E 测试用例
  run                        使用 Chrome DevTools 执行 E2E 测试

示例：
  e2e-fe gen-test-case /path/to/feature_spec.md                    # 生成 P0+P1 用例（默认）
  e2e-fe gen-test-case /path/to/feature_spec.md p0                 # 仅生成 P0 用例
  e2e-fe run /path/to/test-cases/                                  # 执行目录下所有用例
  e2e-fe run /path/to/case.yaml                                    # 执行单个用例
```
