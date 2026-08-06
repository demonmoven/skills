# Code-Debt-Cleaner Repo Onboarding

## 目标

给一个业务仓库补齐最小接入面，让 `code_debt_cleaner` 能稳定跑扫描链路。

这份说明的受众是 agent。用户不需要自己执行命令。agent 应该在目标仓库中自行完成初始化和验证，再把接入结果汇报给用户。

## 仓库侧必须提供的东西

1. 根目录 `.golangci.yml`
2. `go.mod` 中的 Go 版本
3. 一个固定的运行环境
   - Go 版本与 `go.mod` 对齐
   - `golangci-lint` 固定为 v2
   - 如果要跑 `duplicated_code`，额外准备 `dupl`

## 推荐接入步骤

### 1. agent 先跑 bootstrap

agent 在目标仓库根目录执行：

```bash
python3 /Users/bytedance/.agents/skills/code_debt_cleaner/resources/bootstrap.py \
  --repo-root . \
  --output .golangci.yml
```

这个命令只负责生成初始配置，不负责后续托管。生成以后，规则直接在仓库自己的 `.golangci.yml` 里维护。

如果你不想走命令，也可以手工复制模板：

- [`repo-template.golangci.yml`](/Users/bytedance/.agents/skills/code_debt_cleaner/resources/repo-template.golangci.yml)

第一版不要急着开很多规则。先跑通下面这组：

- `gosimple`
- `stylecheck`
- `ineffassign`
- `unused`
- `revive`
- `errcheck`
- `govet`
- `staticcheck`

如果初始化阶段就要做轻微调整，可以直接在 bootstrap 时传：

- `--enable typecheck`
- `--disable cyclop,gocyclo`

### 2. 显式决定是否允许 `typecheck`

默认不要开 `typecheck`。

只有在下面条件都满足时，才建议显式开启：

- 运行环境 Go 版本与仓库一致
- 依赖能稳定加载
- 没有架构相关编译阻塞
- 你确认这个仓库能承受 `typecheck` 带来的噪音

### 3. 不要把 formatter 混进主扫描

第一版主扫描不处理：

- `gofmt`
- `goimports`

如果仓库确实要查 formatter，单独建链路，不要塞回主扫描。

### 4. agent 验证接入

agent 在目标仓库根目录执行：

```bash
python3 /Users/bytedance/.agents/skills/code_debt_cleaner/resources/scout.py \
  --config .golangci.yml \
  --report-json /tmp/code-debt-cleaner-report.json \
  -t gosimple,unused
```

预期只有三种结果：

- `status=ok`：扫描完成
- `status=blocked`：环境不满足
- `status=error`：配置或调用错误

如果出现 `blocked`，agent 先修环境或向用户报告阻塞原因，不要让 cleaner 硬跑。

### 5. 接入阶段汇报格式

接入阶段对用户的汇报也按固定结构来：

- 接入状态：未接入 / 已接入 / 已更新
- 执行结果：ok / blocked / error
- 当前规则摘要：启用了哪些核心 linter，是否启用 `typecheck`
- 阻塞项或问题摘要：只列最关键的 1-5 条
- 下一步建议：只给 1-3 条

不要把初始化命令、长日志、完整 JSON 原样倒给用户，除非用户明确要求看原始输出。

## 建议的接入顺序

1. 先接单仓库
2. 先跑扫描，不开自动修复
3. 先只修低风险项
4. 观察一段时间后，再考虑加 `duplicated_code` 或 `typecheck`

## 不要做的事

- 不要用 `@latest` 安装工具
- 不要一开始就开很多 linter
- 不要把扫描失败当成 “0 issues”
- 不要把自动提交和自动推送直接放进第一版
- 不要把命令步骤原样甩给用户执行，除非用户明确要求手动接管
