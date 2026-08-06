# star_coco_plugin 简介（commands & skills）

本仓库用于为 `coco` 提供一组可复用的工程化指令（`commands/`）与流程规范技能（`skills/`），主要面向星图/广告后端研发的 OpenSpec 工作流：从初始化规范、生成测试计划、执行冒烟测试与闭环修复，到涉及 IDL/ORM/构建等关键环节的标准化操作。

> 说明：本文仅概述主要能力与使用场景；具体输入/输出与约束以对应文件内容为准。

## Commands（OpenSpec 指令清单）

这些指令位于 `commands/openspec/`，通常作为 OpenSpec 工作流中的“可执行提示词/步骤模板”，用于指导在目标仓库中完成规范化的分析、文档生成与测试闭环。

| 指令文件 | 主要功能 | 典型使用场景 |
| --- | --- | --- |
| `commands/openspec/star_init.md` | 初始化目标仓库的 OpenSpec 框架：探索仓库、识别技术栈/结构/约束，执行 `openspec init`，并生成/完善 `openspec/project.md` 与基础规格结构。 | 新接入一个仓库，希望建立“规范 + 变更流程”的起点。 |
| `commands/openspec/make_project.md` | 基于示例与仓库现状生成 `project.md`：沉淀仓库目的、技术栈、分层约定、测试策略与关键约束。 | 已有 OpenSpec 结构但缺少/需要重写 `project.md`。 |
| `commands/openspec/prepare_test.md` | 基于 `spec.md/design.md/tasks.md` 等材料生成接口级冒烟测试用例 `test_plan.md`（强调接口粒度与可验证输入/输出）。 | 变更完成后，需要产出面向接口的 Smoke Test 计划。 |
| `commands/openspec/do_test.md` | 按 `test_plan.md` 执行冒烟测试：要求使用 `bytedance-mcp-api_test_mcp` 发起调用；失败则定位原因、修改代码并复测，形成闭环。 | 需要对某次变更做“测试执行 + 修复 + 复测”的闭环验证。 |
| `commands/openspec/rework.md` | 对某个 OpenSpec 变更进行返工：补充/修订 `proposal/spec/tasks/design`，重新校验一致性后再推进实现。 | MR/测试/评审反馈后，需要调整方案与任务拆解并重新落地。 |

## Skills（技能清单）

这些技能位于 `skills/`，用于在关键工程环节提供“唯一标准/硬约束”的流程指引。

| Skill | 主要功能 | 关键约束/要点（摘要） |
| --- | --- | --- |
| `skills/compiling-and-building-process/` | 指导编译与构建流程。 | 构建前必须提交并推送；禁止本地构建，必须使用 `bytedance-mcp-scm`；只能创建测试版本；SCM 仓库名需要按规则转换（例如将 `ad/star_control` 转为 `ad/star/control`）。 |
| `skills/orm-process/` | 指导 ORM 层开发与生成：model/dao/base_dal 生成与集成。 | 先确认仓库的 DAL/Model 目录结构；按设计 SQL 生成 model/dao（如存在 `base_dal` 需同步生成）；新增表需在 `conf/database.yml` 注册（若仓库存在该配置）。 |
| `skills/thrift-idl-process/` | 指导 Thrift IDL 的提交、推送、桩代码生成与 Go 依赖下载，是 IDL 开发的唯一标准。 | 必须遵守 IDL 规范；禁止本地生成桩代码，需通过 `star_thrift_gen/overpass` 等方式触发生成；禁止下载或修改 `ad/star_idl_gen`，只能在 `ad/star_idl` 进行 IDL 开发；`go get` 失败需按策略重试（最多 3 次）。 |
| `skills/thrift-idl-to-go-code/` | 在编写 Go 代码时，定位并使用 IDL 对应的客户端桩代码（方法/结构体/枚举）的准确包路径与示例。 | 先获取完整 IDL（必要时递归 include）；再通过 overpass 获取/生成客户端仓库与导入路径；在本地 GOPATH 源码中按目标符号精确搜索，避免全量读取大文件。 |
| `skills/domain-knowledge/` | 检索阅读星图后端核心领域知识（治理/审核/资金/用户等），辅助方案设计。 | 用于需求澄清与设计阶段；知识索引位于 `skills/domain-knowledge/references/` 下的各领域文档。 |

## 相关配置（简要）

- `coco.yaml` 定义了本插件运行时可能需要的 MCP 服务接入（例如 Overpass、SCM、API Test 等），用于支撑代码生成、构建与接口测试等能力。

https://mcp.larkoffice.com/mcp/mcp_G1J0VplWai-w7bvb3fCnrHQRxaBdYFha0blJgUtDQedf6CSGWmMuLhqpfb_y7ZqMAl-kmrwpXl4