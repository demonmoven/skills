# 设计审阅（交互式）

由主 Agent 直接执行，与用户交互式校审修订技术方案设计文档。

## 参数

- `FEATURE_NAME`（必需）：功能特性名称，用于定位 specs 目录

## 入参校验

读取并执行 `<skill_dir>/prompts/resolve_feature_name.md`，传入以下参数：
- `ACTION_TYPE = "other"`
- `ARG = {用户传入的参数}`

执行完成后获得 `FEATURE_NAME`。

推导内部变量：

```text
SPECS_DIR = docs/xdev/speckit/{FEATURE_NAME}
DATA_MODEL_DOC = {SPECS_DIR}/data-model.md
CONTRACTS_DOC = {SPECS_DIR}/contracts.md
CONFIGURATION_DOC = {SPECS_DIR}/configuration.md
INTEGRATION_DOC = {SPECS_DIR}/integration.md
PLAN_DOC = {SPECS_DIR}/plan.md
```

继续校验：

1. 以下 5 个技术方案文件全部存在：
   - `DATA_MODEL_DOC` — 不存在 → 提示用户："`DATA_MODEL_DOC` 文件不存在：{路径}，请确认 data-model.md 文件是否已生成"
   - `CONTRACTS_DOC` — 不存在 → 提示用户："`CONTRACTS_DOC` 文件不存在：{路径}，请确认 contracts.md 文件是否已生成"
   - `CONFIGURATION_DOC` — 不存在 → 提示用户："`CONFIGURATION_DOC` 文件不存在：{路径}，请确认 configuration.md 文件是否已生成"
   - `INTEGRATION_DOC` — 不存在 → 提示用户："`INTEGRATION_DOC` 文件不存在：{路径}，请确认 integration.md 文件是否已生成"
   - `PLAN_DOC` — 不存在 → 提示用户："`PLAN_DOC` 文件不存在：{路径}，请确认 plan.md 文件是否已生成"

全部校验通过后，进入执行阶段。

## 执行流程

### 1. 开场提示

向用户输出以下提示，进入审阅模式：

```
现在我来协助您对技术方案设计文档进行校审修订。

技术方案由以下 5 个独立文件组成，请逐一或整体审阅：

1. 数据模型设计：{DATA_MODEL_DOC}
2. API 接口契约：{CONTRACTS_DOC}
3. 配置设计：{CONFIGURATION_DOC}
4. 系统集成设计：{INTEGRATION_DOC}
5. 技术实现方案：{PLAN_DOC}

请告诉我需要修改哪个文件的哪些内容，我来帮您修改。审阅完毕后告诉我即可。
```

### 2. 交互式修改循环

用户会反复提出修改意见，每次收到用户的修改要求后：

1. **定位文件**：根据用户的修改要求确定需要修改的目标文件（可能是上述 5 个文件中的一个或多个）
2. **执行修改**：根据用户要求修改对应文件
3. **告知用户修改完成**：简要说明对哪个文件做了什么修改
4. **询问是否继续**：`还有其他需要修改的地方吗？`

> 用户可能继续提修改意见（重复上述 1-4），也可能表示审阅完毕（进入步骤 3）。

### 3. 完成审阅

用户表示审阅完毕后，输出完成确认：

```
设计审阅完毕！技术方案文档已按您的意见修订完成。

修订后的文档：
- 数据模型设计：{DATA_MODEL_DOC}
- API 接口契约：{CONTRACTS_DOC}
- 配置设计：{CONFIGURATION_DOC}
- 系统集成设计：{INTEGRATION_DOC}
- 技术实现方案：{PLAN_DOC}
```
