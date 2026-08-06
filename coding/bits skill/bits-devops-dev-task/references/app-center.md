# Bits 研发流程应用中心能力

## 按关键词搜索项目

```bash
bitscli devops app search --keyword <关键词> [--project-type <类型>] [--control-planes <控制面>]
```

**重要：搜索时必须传入正确的项目类型，否则默认只搜索 TCE 类型，导致搜不到其他类型项目。**

当用户已知项目类型时，必须通过 `--project-type` 传入对应枚举值：

| 项目类型（用户描述） | `--project-type` 枚举值 |
|---------------------|------------------------|
| TCE（默认） | `PROJECT_TYPE_TCE` |
| 自定义项目 / CUSTOM | `PROJECT_TYPE_CUSTOM` |
| FaaS / FAAS | `PROJECT_TYPE_FAAS` |
| 定时任务 / CRONJOB | `PROJECT_TYPE_CRONJOB` |
| Web / WEB | `PROJECT_TYPE_WEB` |
| Hybrid / HYBRID | `PROJECT_TYPE_HYBRID` |

控制面 `--control-planes` 枚举值对照（多个用逗号分隔）：

| 控制面 | `--control-planes` 枚举值 |
|--------|--------------------------|
| CN（默认） | `CONTROL_PLANE_CN` |
| I18N | `CONTROL_PLANE_I18N` |
| EU_TTP | `CONTROL_PLANE_EU_TTP` |
| US_TTP | `CONTROL_PLANE_US_TTP` |
| I18N_BD | `CONTROL_PLANE_I18N_BD` |

**示例**：用户描述"Gecko 控制面（feat/gecko-control-panel，自定义项目，CN）"时，项目类型为 CUSTOM、控制面为 CN，搜索命令为：

```bash
bitscli devops app search --keyword 'Gecko 控制面' --project-type PROJECT_TYPE_CUSTOM --control-planes CONTROL_PLANE_CN --limit 20
```

**规则**：
- 用户已说明项目类型（如"自定义项目"、"FaaS"等）→ 必须传对应的 `--project-type`
- 用户已说明控制面（如"CN"、"I18N"等）→ 必须传对应的 `--control-planes`
- 用户未说明项目类型 → 默认 `PROJECT_TYPE_TCE`，可能需要多次尝试不同类型

---

## 按项目唯一标识查详情（components + SCM）

```bash
bitscli devops app components --project-unique-id <projectUniqueId>
```

用于确认该项目在应用中心存在、并查看类型与控制面等信息。

---