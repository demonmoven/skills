# PRD → 技术方案 Agent（prd-to-tech-design）

将计费/结算 PRD 转化为有代码证据支撑的飞书技术方案的编排型 Agent 及其关联 skill 集合。

## 结构

| 角色 | Skill | 目录 | 说明 |
| --- | --- | --- | --- |
| 编排 Agent | `prd-tech-design` | `prd-tech-design/` | 串联各阶段，产出飞书技术方案 |
| 阶段 1 | `prd-understand` | `prd-understand/` | 理解并归一化 PRD，资金流阻塞式提问 |
| 阶段 2 | `prd-feature-split` | `prd-feature-split/` | 内含 `llm-wiki-git` 子流程：先查系统现状/已有能力/历史方案，再拆分功能点 + 仓库/接口/链路定位 |
| 补充调查 | `prd-wiki-query` | `prd-wiki-query/` | 功能拆分后的 Wiki/代码补充调查或重查入口 |
| 绘图子 skill | `prd-sequence-diagram` | `prd-sequence-diagram/` | 生成包含 action、participant、判断分支与变更 diff 的业务/系统时序图 |
| 绘图子 skill | `prd-funds-flow-diagram` | `prd-funds-flow-diagram/` | 生成主体与待结算户/现金户/垫资户等账户级资金流图，并输出配套信息流字段链路 |
| 阶段 4 | `prd-design-generate` | `prd-design-generate/` | 整合时序图/资金流图等材料，起草方案并创建飞书文档 |
| 阶段 5（可选） | `prd-design-revise` | `prd-design-revise/` | 按评论原地微调，不重生成；同点全文一致修改 |
| 前置识别门 | `settle-center-recognition` | `settle-center-recognition/` | settle_center 端内/端外、入口接口、conf/mapping → XML 流程识别 |

## 依赖

- 知识库检索强依赖仓库根目录的 `../llm-wiki-git` skill（结算 Git-only 知识库 `charge_settle_llm_wiki`）。
- PRD agent 内部使用方式固定为 `llm-wiki-git query "<问题>"`；语义检索路径按 `Module/System -> Scenario -> PlatformCapability -> Application -> Data/Implementation -> Source -> RawRef` 展开。
- 模板与流程参考集中在 `prd-tech-design/references/`（`tech-design-template.md`、`analysis-workflow.md`、`feishu-delivery.md`）。

## 使用

入口为 `prd-tech-design`：给定 PRD（飞书 URL / 本地文件 / 文本），按阶段顺序执行。需求理解后，功能点拆分阶段必须先通过 `llm-wiki-git query "<问题>"` 检索知识库，并按 `Module/System -> Scenario -> PlatformCapability -> Application -> Data/Implementation -> Source -> RawRef` 理解系统现状，结合代码/配置/接口证据明确已有能力、本次需要增强能力、本次需要新增能力和无需改动能力；涉及资金流不确定点时阻塞确认；改动点必须代码核验，settle_center XML 编排改动需给出完整 XML。

生成方案前会把 `prd-sequence-diagram` 和 `prd-funds-flow-diagram` 作为画图子 skill 调用：时序图写入业务流程分析/系统时序分析，资金流图写入资金流分析，并同步生成信息流分析所需字段链路。

## 工作目录

每个 PRD 需求都会在当前工作目录下使用独立目录：`prd2tech/<需求目录>/`。阶段 1 和阶段 2 是硬性本地产物：

- `prd-understand.md`：需求理解结果，供用户微调。
- `investigation.md`：阶段 2 前置的 Wiki 与代码调查结果，记录系统现状、已有能力、历史方案、代码/配置/接口初步证据和待确认项。
- `prd-feature-split.md`：基于 `investigation.md` 的功能点拆分结果，供用户微调。

后续阶段继续写入同一目录：

- `sequence-diagram.md`：时序图/流程图和 diff。
- `funds-flow-diagram.md`：资金流图、金额赋值和信息流字段链路。
- `tech-design.md`：用于创建飞书文档的技术方案 Markdown 草稿。

用户修改 `prd-understand.md` 或 `prd-feature-split.md` 后，可以要求重跑整个工作流。重跑时以本地微调后的中间文件为输入，只重算受影响的后续阶段。
