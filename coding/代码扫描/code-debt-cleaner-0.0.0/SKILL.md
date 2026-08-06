---
name: code-debt-cleaner
description: 单仓库代码债务维护 worker。仓库自己声明规则，skill 负责预检环境、扫描问题、筛选低风险项、辅助修复和验证。第一版优先打通扫描链路，不把 GitOps 作为主流程。
---

# Code-Debt-Cleaner

## 定位

Code-Debt-Cleaner 不是通用治理平台，也不是代替仓库定义规范的“中心服务”。

这个 skill 是给 agent 用的，不是给用户手工执行命令用的。用户给出目标仓库后，agent 应该自己完成初始化、扫描、分析、验证，再把结果汇报给用户。

它的职责只有一件事：消费单个仓库已经声明好的质量规则，执行统一的维护流程。

角色边界必须固定：

- `repo` 负责声明规则
  - 提交 `.golangci.yml`
  - 在 `go.mod` 中声明 Go 版本
  - 决定默认开启哪些检查、排除哪些路径
- `code_debt_cleaner` 负责执行流程
  - 预检环境
  - 运行扫描
  - 输出结构化结果
  - 挑选低风险问题
  - 生成修复建议
  - 做验证

第一版只优先保证扫描链路可信：

- `golangci-lint` v2 可用
- 环境不满足时返回 `blocked`
- 配置缺失或调用错误时返回 `error`
- 默认不走 formatter 和 `typecheck`
- 结果同时输出终端摘要和 JSON 文件

## 接入契约

一个仓库要接入，至少满足下面条件：

1. 仓库根目录存在 `.golangci.yml`
2. `go.mod` 明确声明 Go 版本
3. 运行环境固定 Go 和 `golangci-lint` 版本

不要让 skill 自己安装 `@latest` 版本工具。工具版本必须由运行环境显式钉死。

建议直接从这两个文件开始：

- 模板：[`resources/repo-template.golangci.yml`](/Users/bytedance/.agents/skills/code_debt_cleaner/resources/repo-template.golangci.yml)
- 接入说明：[`resources/repo-onboarding.md`](/Users/bytedance/.agents/skills/code_debt_cleaner/resources/repo-onboarding.md)

如果要给一个新仓库生成初始配置，优先跑：

- 初始化器：[`resources/bootstrap.py`](/Users/bytedance/.agents/skills/code_debt_cleaner/resources/bootstrap.py)

## 当前支持的问题类型

### LLM 分析类

- `long_method`
- `large_class`
- `long_parameter_list`
- `divergent_change`
- `shotgun_surgery`
- `feature_envy`
- `data_clumps`
- `primitive_obsession`
- `switch_statements`
- `parallel_inheritance_hierarchies`
- `lazy_class`
- `speculative_generality`
- `temporary_field`
- `message_chains`
- `middle_man`
- `inappropriate_intimacy`
- `alternative_classes_with_different_interfaces`
- `incomplete_library_class`
- `data_class`
- `refused_bequest`
- `comments`

### 工具扫描类

- `duplicated_code`
- `staticcheck`
- `gosimple`
- `stylecheck`
- `gocyclo`
- `errcheck`
- `ineffassign`
- `unused`
- `revive`
- `cyclop`
- `govet`
- `gofmt`，仅允许显式单独处理，不在主扫描链路
- `goimports`，仅允许显式单独处理，不在主扫描链路
- `typecheck`，默认关闭，需显式开启

## v2 约束

`golangci-lint` v2 下要遵守这些规则：

- 主扫描只走 `golangci-lint run`
- formatter 不能再混进 `run --enable-only`
- `gofmt` / `goimports` 不属于第一版主扫描链路
- `typecheck` 默认关闭，只有显式 opt-in 才允许执行
- 配置文件以仓库内 `.golangci.yml` 为准

## 工作流程

### 1. 准备

1. 检查当前目录是否存在 `go.mod`
2. agent 自己运行 `python3 resources/prepare.py`
3. 解析输出：
   - `all_selected_types`
   - `tool_types`
   - `llm_types`
   - `directories`

### 2. 预检

在调用 `scout.py` 前，必须先确认：

- `.golangci.yml` 存在
- `go.mod` 中的 Go 版本可被当前 `go version` 满足
- `golangci-lint` 已安装且为 v2

如果不满足：

- `status=blocked`：环境前置条件不满足
- `status=error`：调用方式或配置本身错误

不要把失败伪装成 “0 个问题”。

### 3. 扫描

agent 自己运行：

```bash
python3 resources/scout.py \
  --config .golangci.yml \
  --report-json /tmp/code-debt-cleaner-report.json \
  -t <tool_types> \
  -d <directories>
```

第一版说明：

- 默认扫描集合不包含 `gofmt` / `goimports` / `typecheck`
- `typecheck` 只有显式 `--enable-typecheck` 才允许执行
- formatter 需要独立链路处理，不进入主扫描

### 4. 输出

`scout.py` 必须产出两类输出：

- 终端摘要
- JSON 文件

JSON 结果结构固定包含：

- `scanner`
- `status`
- `timestamp`
- `toolchain`
- `command`
- `blocked_reasons`
- `issue_count`
- `issues`

agent 对用户的输出应该是结论，不是把命令甩给用户。命令、日志、JSON 路径由 agent 自己处理，用户只需要看到：

- 当前仓库是否已接入
- 扫描是否成功、被阻塞还是配置错误
- 发现了哪些值得处理的问题
- 下一步建议做什么

## Agent 汇报格式

agent 完成一次初始化、扫描或修复后，对用户的汇报必须尽量收敛成下面 5 段。没有内容的段可以省略，但顺序不要乱。

### 1. 接入状态

- `未接入`：仓库缺 `.golangci.yml`，或初始化还没完成
- `已接入`：仓库已有可用 `.golangci.yml`
- `已更新`：agent 本轮更新了仓库规则或接入配置

### 2. 执行结果

- `ok`：扫描成功
- `blocked`：环境不满足，无法可信执行
- `error`：配置或调用错误

这里必须直接说清主因，不要只贴原始日志。

### 3. 当前规则摘要

只汇报高信号信息：

- 当前启用的核心 linter
- 是否启用了 `typecheck`
- 是否仍然把 formatter 排除在主扫描外

不要把整份 `.golangci.yml` 原样倒给用户。

### 4. 问题摘要

如果扫描成功：

- 总问题数
- 建议优先处理的前 3-5 个问题
- 每个问题只保留：类型、文件、为什么值得先处理

如果扫描未成功：

- 这一段改成阻塞项摘要

### 5. 下一步建议

只能给 1-3 条，按优先级排序。

优先给这种建议：

- 先修环境
- 先确认是否接入该仓库
- 先处理低风险项
- 暂时不要开启 `typecheck`

不要给一串开放式 brainstorming。

## Agent 汇报示例

```text
接入状态：已接入

执行结果：blocked
主因：仓库 go.mod 要求 1.25，当前机器是 go1.24.11，扫描结果不可信。

当前规则摘要：
- 已启用 gosimple/stylecheck/ineffassign/unused/revive/errcheck/govet/staticcheck
- 未启用 typecheck
- formatter 仍然不在主扫描链路

阻塞项摘要：
- Go 版本低于 go.mod

下一步建议：
1. 先把执行机器 Go 版本提到不低于 go.mod
2. 环境修好后先跑低风险项扫描
```

## 修复策略

扫描结果出来后，优先处理：

- 低风险
- 改动小
- 易验证
- 不涉及生成代码

单次建议修复 3-5 个问题，不要把一次治理做成大 PR。

## 验证

修复前后都必须验证。验证顺序不能乱：

1. 检查 Git 状态
2. `go build ./...`
3. 重新跑对应扫描
4. `go test ./...`，如果耗时可说明原因，但不能假装已经通过

## GitOps

第一版不把 GitOps 作为主链路。

`resources/gitops.py` 当前不是主链路：

- 默认不再 `git add .`
- `commit` 必须显式传 `--paths ...` 或 `--all`

即使如此，第一版仍然不建议把 GitOps 放进自动日常治理主流程。
