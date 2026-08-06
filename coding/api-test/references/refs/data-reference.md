# 参考数据

本文档是 api-test skill 的**参考数据单一源**，集中维护跨步骤复用的映射表与规则表。其他文档（`SKILL.md`、`flows/test-flow.md`、`support/runtime-support.md` 等）一律引用本文件，禁止再各自复制，避免多处维护导致漂移。

## 目录
- [参考数据](#参考数据)
  - [目录](#目录)
  - [VRegion 映射表](#vregion-映射表)
  - [环境名称规则](#环境名称规则)
  - [Skill 与 Mode 对照表](#skill-与-mode-对照表)
  - [固定监控字段](#固定监控字段)
  - [Step ID 对照表](#step-id-对照表)
  - [错误分类建议](#错误分类建议)

## VRegion 映射表

用于由 `<VRegion>`（或其别名）推导出 `<ZONE>`、`<VDC>`、`<JWT-VRegion>` 等后续请求必填参数。

| VRegion      | ZONE             | VDC                                            | JWT-VRegion | Alias                       |
| ------------ | ---------------- | ---------------------------------------------- | ----------- | --------------------------- |
| boe          | BOE              | boe                                            | boe         | China-BOE, BOE              |
| boei18n      | BOEI18N          | boei18n                                        | boei18n     | US-BOE, BOEI18N             |
| china-north  | CN               | lf *(default)*, hl, lq, yg                     | cn          | China-North, CN             |
| china-north6 | China-North6     | zb *(default)*, xh, gl2                        | cn          | China-North6, CN6           |
| china-east   | China-East       | hj *(default)*, hjzg, zjg, jj                  | cn          | China-East                  |
| i18n-tt      | SGALI            | sg1 *(default)*, sgdt, my, my2, my3            | i18n        | Singapore-Central, sg, i18n |
| i18n-tt      | MVAALI           | maliva *(default)*, useast3, useastdt, useast4 | i18n        | US-East, va, us             |
| i18n-bd      | Asia-SouthEastBD | mya *(default)*, myb, myc, bddedt, bdsgdt      | i18n-bd     | I18N-BD                     |

## 环境名称规则

用于校准 `<ENV>`（泳道/环境名称）的前缀规则。

| VRegion                               | 环境名称规则    | 示例                  |
| :------------------------------------ | :-------- | :------------------ |
| boe, boei18n                          | `boe_` 开头 | boe\_xxx, boe\_test |
| china-north, us, sg, i18n-tt, i18n-bd | `ppe_` 开头 | ppe\_xxx, ppe\_test |

## Skill 与 Mode 对照表

执行数据打点上报时，`--skill` 与 `--mode` 的固定值定义。

| Skill      | 固定 Mode   | 说明                       |
| ---------- | ---------- | -------------------------- |
| `api-test` | `Fast call` | 正式版 `api-test` 数据上报固定值 |

## 固定监控字段

执行 Step 级指标上报时，以下 `report_info` 字段采用固定口径：

| 字段 | 固定值/规则 | 说明 |
| --- | --- | --- |
| `report_info.action` | Skill 粒度为 `skill`；Step 粒度为 `step_<STEP_ID>` | 例如 `step_S1_VREGION` |
| `report_info.platform` | 调用该 skill 的 agent 名称 | 优先显式传参，其次环境变量，最后按安装路径推导，例如 `.trae/skills/api-test` 推导为 `trae` |
| `report_info.session_id` | 优先显式传参或 `EXEC_SESSION_ID` | 若未提供，脚本会在当前活跃本地会话中自动复用同一个 `session_id`，同一轮状态全部收口后再滚动新 ID |
| `report_info.customs` | 上报前会移除 `event_type`、`step_id` | 若 `customs` 为空或清洗后为空对象，则忽略不上报 |

## Step ID 对照表

执行 Step 级指标上报时，所有步骤标识统一使用以下 ID，并编码在 `report_info.action=step_<STEP_ID>` 中；本地状态和输入归一化阶段仍可使用 `step_id`，但最终上报会从 `customs` 中移除。

| Step ID | 对应流程 | 说明 |
| --- | --- | --- |
| `S1_VREGION` | `SKILL.md` Step 1 | 确认测试 VRegion |
| `S1_5_INITIAL_REPORT` | `SKILL.md` Step 1.5 | 初始打点上报 |
| `S2_JWT` | `SKILL.md` Step 2 | 获取 JWT |
| `S3_IDL` | `SKILL.md` Step 3 | 确认 IDL 仓库与分支 |
| `S4_ENV_PRECHECK` | `SKILL.md` Step 4 | 环境预检 |
| `S5_TEST` | `SKILL.md` Step 5 | 执行接口测试整体流程 |
| `S5_1_SELECT_API` | `test-flow.md` Step 1 | 明确测试接口 |
| `S5_2_BUILD_REQUEST` | `test-flow.md` Step 2 | 确认接口请求参数 |
| `S5_3_SEND_REQUEST` | `test-flow.md` Step 3 | 发送请求 |
| `S5_4_SUMMARY` | `test-flow.md` Step 4 | 总结测试结果 |

## 错误分类建议

执行 Step 级指标上报时，失败事件的 `error_type` 建议使用以下口径，并写入 `report_info.customs.error_type`。

| error_type | 适用场景 |
| --- | --- |
| `user_input_missing` | 用户未提供必要参数，导致当前 Step 无法继续 |
| `auth_failed` | JWT 获取失败、权限不足、鉴权依赖缺失 |
| `idl_unavailable` | IDL 仓库、分支、BAM 元数据不可用 |
| `env_unavailable` | 环境不存在、实例未运行、环境预检失败 |
| `request_failed` | 接口请求执行失败、请求构造失败、返回错误 |
| `report_failed` | 打点脚本自身失败、上报接口不可达 |
| `unknown` | 无法归类但需要记录失败 |
