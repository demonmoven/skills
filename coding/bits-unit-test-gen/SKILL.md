---
name: bits-unit-test-gen
description: 强制工作流 — 非可选指导。为 Go、JS/TS、Python、Java、Kotlin、Swift、C++ 和 Rust 项目生成、修复和维护单元测试。当此 Skill 被调用时，你必须按以下定义逐步顺序执行。不得跳过、重排、合并或概括步骤。不得将此视为参考资料。这就是你的执行计划。触发词：“写单测”、“生成单测”、“补充单测”、“修复单测”、“提升单测覆盖率”。当用户选中或提及一个单元测试函数/测试文件，并要求对其进行修复、更新、保鲜、补充用例或任何修改时，也必须触发此 Skill。不负责 API 接口测试、GUI/UI 测试、E2E 测试。
allowed-tools:
  - Read
  - Write
  - Edit
  - Grep
  - Bash
  - Glob
metadata:
  version: 2.2.2
---

# bits-unit-test-gen

## 元信息
| 变量 | 作用                                                                                                                                           |
|------|----------------------------------------------------------------------------------------------------------------------------------------------|
| `SKILL_ROOT` | 当前 skill 根目录，即包含本 SKILL.md 的目录的绝对路径。本文件及其引用文档（`assets/`、`references/`、`scripts/`）中的所有 Skill 内部路径必须从 `SKILL_ROOT` 解析。 |
| `PROJECT_ROOT` | 用户当前项目根目录的绝对路径。                                                                                                                              |
| `LANG` | 语言标识符，获取方式见 `${SKILL_ROOT}/references/knowledge-context.md`。                                                                                |
| `TMP_ROOT` | 本次生成任务的临时工作目录，绝对路径，获取方式见外层生成流程 `Step1 测试准备`                                                                                                    |
| `EXEC_SOURCE` | 执行来源，获取方式见 `${SKILL_ROOT}/references/knowledge-context.md`。                                                |

## 生成流程
你要严格按 bits-unit-test-gen 执行，不允许把它当参考。
要求：
1. 先给出 Step1~Step7 的执行状态。
2. 在完成 Step3 前不要写测试。
3. 在完成 Step4 前不要生成用例，先输出 TARGETS 和 BUG_MAP。
4. Step5 必须按”生成 -> 验证 -> 修复”循环汇报，每轮给出测试命令、失败类型和处理结果。
5. 最终总结里单独列出：范围、缺陷分析、生成的用例、验证结果。
如果你要跳步，先说明原因并征求确认。

### 强制流程跟踪
> 在执行生成流程之前，**必须**先使用 TodoWrite 工具（或者类似的任务管理工具）将要执行的任务按步骤全部写入 task list，参考以下模版
```
Task List 模板（进入流程时一次性写入）：
1. Step1 测试准备
2. Step2 加载语言知识&全局上下文
3. Step3 确定生成范围
4. Step4 分析缺陷
5. Step5 生成、验证、修复
6. Step6 覆盖率统计与补充
7. Step7 输出报告
```

### Step1 测试准备
**运行 prepare_test.sh**，完成必要的工具准备和变量获取
```bash
AGENT_SOURCE=<agent_name> MODEL_SOURCE=<model_name> SKILL_ROOT=${SKILL_ROOT} \
  bash ${SKILL_ROOT}/scripts/prepare_test.sh --repo-path "$PROJECT_ROOT"
```
执行上述命令后，输出的 `BITS_TMP_ROOT=<path>` 为 `TMP_ROOT` 的绝对路径。

### Step2 加载语言知识&全局上下文
**加载 `${SKILL_ROOT}/references/knowledge-context.md`**，获取语言特定的单测知识和全局上下文。
#### Step2 检查点
- `LANG` 已确定且非空
- `EXEC_SOURCE` 已按 `${FEATURE_DIR}` 或 `~/.flux/config.json` 或 `prepare_test.sh` 返回值判断

### Step3 确定生成范围
**加载 `${SKILL_ROOT}/references/scope.md`**，确定生成范围、diff 上下文。
#### Step3 检查点
- `TARGETS` 已确定且非空。
- `cov_config` 已加载；无配置时为 null

### Step4 分析缺陷
按照 `${SKILL_ROOT}/references/detect-bugs.md` 中的说明，基于代码事实检测缺陷并输出 BUG_MAP。BUG_MAP 的准入判定完全由 detect-bugs.md 的门禁决定，不受任何下游步骤的可执行性、通过率或覆盖率目标的约束。检出的缺陷应映射到单测场景，保证后续生成的单测能够失败，并且能够定位到相关缺陷。
#### Step4 检查点
- `BUG_MAP` 已确定。


### Step5 生成、验证、修复
**加载 `${SKILL_ROOT}/references/gen-verify-fix-loop.md`**，按其中的 `Loop-A -> Loop-B -> Loop-C -> Loop-D` 内部循环生成、验证和修复单测；该内部循环整体属于外层 `Step5`。

> **关键区分**：Step5 的终态允许缺陷探测用例保持失败（结论为"部分失败但已实锤 BUG_MAP 缺陷"）。不得为了达到"全部通过"而清空 BUG_MAP 或把实锤缺陷重新归因为测试问题。
> 缺陷探测用例不会计入最终的覆盖率统计，**切勿**为了通过覆盖率要求而剔除缺陷。

### Step6 覆盖率统计与补充
**加载 `${SKILL_ROOT}/references/gate-check.md`**，按其流程执行覆盖率统计和可选补充。
#### Step6 检查点
- 覆盖率统计与补充**跳过**或**已完成**（通过、补充后通过、或未达标已记录，并输出覆盖率结果）

### Step7 输出报告
**加载 `${SKILL_ROOT}/references/report.md`**，生成单测报告。

#### Step7 检查点
- `utree flush` 已执行。
- 若 `EXEC_SOURCE=flux` 或 `EXEC_SOURCE=flux-web`，已按 `${SKILL_ROOT}/assets/templates/flux_report.md` 的要求生成报告。
- 若 `EXEC_SOURCE=PIPELINE`，已按 `${SKILL_ROOT}/assets/artifacts/defects.md` 中的要求写入缺陷文件。
