# 上下文加载

本文只负责加载单元测试任务所需的全局知识和语言知识，不负责确定本次生成范围。目标范围、diff 计算、目标文件过滤、测试函数反查和已有测试风格采样统一由 `${SKILL_ROOT}/references/scope.md` 处理。

## 目录

- [职责边界](#职责边界)
- [全局规则（贯穿整个任务，非本阶段执行项）](#全局规则贯穿整个任务非本阶段执行项)
  - [规则优先级](#规则优先级)
  - [RAG 知识检索能力](#rag-知识检索能力)
- [执行步骤](#执行步骤)
  - [Phase 1 识别语言](#phase-1-识别语言)
  - [Phase 2 加载全局上下文](#phase-2-加载全局上下文)
  - [Phase 3 加载语言知识](#phase-3-加载语言知识)
  - [Phase 4 输出上下文摘要](#phase-4-输出上下文摘要)

## 职责边界

本文件负责：

- 注册贯穿全流程的全局规则与能力声明（规则优先级、RAG 知识检索）。
- 识别并记录 `LANG`。
- 设置 `EXEC_SOURCE`。
- 读取用户意图、项目约定和语言级单测规则。

本文件不负责：

- 判断 `TEST_SCOPE`、目标文件、目标函数或目标测试文件。
- 执行 `git diff`、解析 hunk 或计算 diff 函数。
- 根据已有测试文件反推被测业务函数。
- 采样目标目录已有测试风格。

## 全局规则（贯穿整个任务，非本阶段执行项）

以下规则在整个任务执行期间始终有效，不限于本阶段。在此注册以便模型在后续所有步骤中引用。

### 规则优先级

当不同来源的规则冲突时，按以下优先级执行：

1. 用户显式指令。
2. 项目单测约定（`AGENTS.md` / `CLAUDE.md`）。
3. 目标范围已有测试风格（由 `scope.md` 采样）。
4. 语言特定规则（`assets/<lang>/prompt.md`）。

若高优先级规则与低优先级规则冲突，保留高优先级规则，并在报告中说明采用原因。

### RAG 知识检索能力

本 skill 具备通过 RAG 按需检索公共知识的能力（框架用法、mock 示例、错误解决方案等）。完整使用协议见 `${SKILL_ROOT}/references/rag.md`。

## 执行步骤

以下 Phase 1-4 为本阶段（Step2）的顺序执行步骤。

### Phase 1 识别语言

识别 `${PROJECT_ROOT}` 的主要编程语言，并赋值给 `LANG`。`LANG` 只能取以下值之一：

| `LANG`       | 适用项目 |
|--------------| --- |
| `go`         | Go |
| `python`     | Python |
| `java`       | Java |
| `javascript` | JavaScript / TypeScript |
| `kotlin`     | Kotlin |
| `cpp`        | C / C++ |
| `swift`      | Swift |
| `rust`       | Rust |

无法判断语言或语言不在上述列表时，立即终止并说明原因。识别结果无需单独向用户输出，但后续阶段必须按该值加载语言 prompt。

### Phase 2 加载全局上下文

按以下顺序读取全局上下文：

1. 按以下规则设置 `EXEC_SOURCE`：
   - 若 `${FEATURE_DIR}` 存在，设置 `EXEC_SOURCE=flux`。
   - 否则，若 prepare_test.sh 有返回 `EXEC_SOURCE`，则以该值返回的 `EXEC_SOURCE` 为准。
   - 以上条件均不满足时，`EXEC_SOURCE` 为空。
2. 如果 `${FEATURE_DIR}/spec.md` 存在，读取该文件，提取与本次单元测试有关的功能意图、验收条件、边界场景和约束。
3. **读取业务规约（biz_spec）**（可选）：如果用户提供了业务参考文档（PRD、TRD、设计文档、产品手册等均可），或在 prompt 中描述了业务背景/需求说明，按以下方式记录来源引用：
   - 飞书文档 URL（含 `larkoffice.com` 或 `feishu.cn`）：`biz_spec` 记为 `飞书文档 <URL> ，使用 lark-doc skill 的 docs read 命令获取内容`。
   - 其他 URL：`biz_spec` 记为 `网页 <URL> ，使用 WebFetch 获取`。
   - 本地文件路径：`biz_spec` 记为 `本地文件 <路径> ，使用 Read 工具读取`。
   - 用户在 prompt 中直接描述的业务背景/需求说明：`biz_spec` 记为 `用户在 prompt 中直接描述的业务规约：<原文内容>`。

   `biz_spec` 仅保存来源引用，不在此阶段加载全文。后续阶段（detect-bugs、gen-verify-fix-loop）在处理具体 TARGET 时按获取指令按需读取相关段落；若获取失败则跳过，不阻塞流程。

4. 在 `${PROJECT_ROOT}` 下检查 `AGENTS.md`、`CLAUDE.md`。若存在，只读取与单元测试有关的内容，例如命名约定、测试框架偏好、目录结构规则、mock 约定、执行命令和禁用事项。

不满足任一条件时不要臆造 `EXEC_SOURCE` 的值。未找到项目约定文件时，记录为"无项目级单测约定"，继续后续阶段。

### Phase 3 加载语言知识

读取 `${SKILL_ROOT}/assets/${LANG}/prompt.md`，并把其中规则作为语言级默认规则。重点提取：

- 最小执行单元定义。
- 最小生成和验证粒度。
- 目标函数或目标对象提取方法。
- 语言级补充文件过滤规则。
- 测试文件命名与组织约定。
- Mock 和断言框架选择。
- 编译、测试和覆盖率命令。
- 预检查要求。

语言 prompt 中引用的额外 reference 只在满足其触发条件时按需加载，不要一次性加载全部语言资料。

### Phase 4 输出上下文摘要

进入外层 `Step3 确定生成范围` 前，必须具备以下结果：

| 字段 | 要求                                                                                             |
| --- |------------------------------------------------------------------------------------------------|
| `LANG` | 已确定且受支持                                                                                        |
| `EXEC_SOURCE` | 已按 `${FEATURE_DIR}` 或 `~/.flux/config.json` 或 `prepare_test.sh` 返回值判断；值为 `flux`、`flux-web`、`PIPELINE` 或空 |
| `feature_context` | 来自 `${FEATURE_DIR}/spec.md` 的相关意图；无则为空                                                         |
| `biz_spec` | 业务规约来源引用（含地址和获取指令）；无则为空，后续阶段按需获取相关内容，获取失败时跳过不阻塞                                    |
| `project_test_conventions` | 来自 `AGENTS.md` / `CLAUDE.md` 的单测约定；无则为空                                                        |
| `language_test_rules` | 已完整加载 `${SKILL_ROOT}/assets/${LANG}/prompt.md`                                                 |

不要在此阶段输出 `TARGETS`、`diff_query`、`diff_index`、`target_method`、`candidate_funcs` 或测试风格样本，这些都是 `${SKILL_ROOT}/references/scope.md` 的产物。
