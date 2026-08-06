# Flux 环境报告生成说明

本文档适用于 flux 环境（即 `EXEC_SOURCE` 为 `flux` 或 `flux-web` 时）。在该环境下，收尾阶段需要**按序**执行以下步骤：

0. **前置准备** — 若 `$FEATURE_DIR` 不存在，调用脚本创建
1. **生成 Markdown 报告** — 写入文件供用户直接查看
2. **准备 html 报告物料** — 读取 `~/.flux/plugins/flux/core/skills/html-report-gen/references/ut.md` 并按照指引调用脚本写入结构化
   record，由平台渲染为 HTML
3. **生成 html 报告** — 若前置准备记录 `NEED_HTML_GEN=true`，且第二步正常执行，调用 html-report-gen skill 渲染生成 html
   报告文件

> 若 `EXEC_SOURCE` 不为 `flux` 或 `flux-web`，则不需要生成任何报告，跳过本文档全部步骤。

***

## 前置准备

若 `$FEATURE_DIR` 已存在，跳过本节，直接进入第一步。

若 `$FEATURE_DIR` 不存在，需要通过脚本创建：

1. 检查脚本 `~/.flux/plugins/flux/core/skills/test-plan/scripts/resolve_feature_dir.sh` 是否存在。若不存在，直接退出，跳过本文档全部后续步骤。
2. 执行 `~/.flux/plugins/flux/core/skills/test-plan/scripts/resolve_feature_dir.sh ${PROJECT_ROOT} <short-name>`，其中
   `<short-name>` 使用 2-4 个英文单词简要概括用户此次的请求。从脚本输出中解析得到创建的 `$FEATURE_DIR`。
3. 设置 `NEED_HTML_GEN=true`。

### 检查点

`$FEATURE_DIR` 已存在。

## 一、Markdown 报告

### 输出位置

按下面的 `模板格式` 写入 `${FEATURE_DIR}/unit-test/ut_test_report.md`，并在最终回复中提供该文件链接供用户查看。

### 字段展示规则

- `**执行状态**` 和 `**生成耗时**` 为**必填项**，不可省略。
- `单测增量覆盖率` 直接使用 Step6 输出的 `diff_coverage`。若 Step6 跳过，填写 `-`。
- `生成状态` 必须基于 Step5 / Step6 的真实结果填写：`成功`（即有测试产物或缺陷透出）或者`失败`（即没有任何测试产物或缺陷透出）。
- `生成耗时` 优先使用 `utree flush` 输出的 `generation_duration_min`。如果没有该变量，禁止重复执行 flush，填写 `未采集`
  ，不得估算或编造。
- `总体统计` 必须基于 Step5 的真实执行结果填写。其中 `执行通过用例数` 为 `生成用例` 中通过的用例数，而不是仓库全体用例。
- `缺陷明细` 必须基于 Step4 输出的 `BUG_MAP` 填写。
- `补测建议` 基于 Step6 输出的 `uncovered_targets`。按 `file_path` 分组，同一文件的多个 symbol 合并到一行的「补测动作」列中（每个
  symbol 一行，用 `<br>` 分隔）。`uncovered_lines` 中连续行号必须合并为区间（如 `15-18`），非连续行号用逗号分隔（如 `15-18,22`
  ）。**最多**输出 10 条条目，若文件数多于 10 个，则选择未覆盖行最多的 10 个文件。
- 统计字段必须来自真实执行结果、生成产物或 flush 输出；无法获得的字段用 `-` 占位，不得使用示例值。
- **当某个模块整体无内容时（如无跳过函数、无缺陷、无修复、无补测建议），不要展示该模块的表格，直接在模块标题下方显示「暂无」即可。
  ** 避免出现只有表头和 `-` 占位行的空表格。

### 格式模板

> **强制要求**：生成的 Markdown 报告**必须严格基于下方格式模板**进行编写，不得自行发挥或更改结构。模板中的标题层级、字段顺序、表格列定义均为固定格式，仅填充实际数据即可。

```
# 单元测试生成结果汇总
**生成状态**：<成功|失败>；**生成耗时**：`<generation_duration_min 或 未采集>`
---
## 总体统计
**单测增量覆盖率**：`x.x%`；**命中函数数**：<count>；**生成用例数**：<count>；**执行通过用例数**：<count>；
**修复编译失败包**：<count>；**修复执行失败用例数**：<count>；**发现缺陷数**：<count>
---
## 生成明细
| 文件名 | 执行成功数/生成用例数 | 生成后增量覆盖率 |
|:-------|:--------------------:|:----------------:|
| [<test_file>](file:///<absolute_path_to_test_file>) | <passed>/<generated> | <percent 或 -> |
---
## 用例修复明细
| 修复类型 | 修复对象| 包含用例数 | 修复后增量覆盖率 |
|:-------|:-------:|:---------:|:-------------:|
| <编译失败|执行失败|测试基础设施> | [<object>](file:///<absolute_path>) | <count> | <percent 或 -> |
---
## 缺陷明细
| 函数 | 场景 | 类型 | 问题 | 修复建议 |
|:-----|:-----|:-----|:-----|:---------|
| [<function>](file:///<absolute_path_to_source>#L<start>-L<end>) | <scenario> | <type> | <problem> | <suggestion> |
## 跳过函数明细
| 函数 | 跳过原因 |
|:-----|:-------|
| [<function>](file:///<absolute_path_to_source>#L<start>-L<end>) | <reason> |
---
## 补测建议
| 对象 | 标题 | 优先级 | 补测动作 |
|:-----|:-----|:------:|:---------|
| [<file_path>](file:///<absolute_path>) | <basename> 单测未覆盖行补测建议 | P1 | <symbol_1> 未覆盖行: a-b,c-d<br><symbol_2> 未覆盖行: e,f-g |
```

### 检查点

确认 `${FEATURE_DIR}/unit-test/ut_test_report.md` 已生成。若该文件不存在，说明第一步未正常完成，重新返回执行第一步，直到该文件生成后进入第二步。

### 写入 BUG_MAP 原始数据

将上下文中的 `BUG_MAP` JSON 写入 `${FEATURE_DIR}/unit-test/bug_map.json`。若 `BUG_MAP` 为空数组，则写入 `{"BUG_MAP": []}`。

#### 检查点

确认 `${FEATURE_DIR}/unit-test/bug_map.json` 已写入。

## 二、准备 html 报告物料

> **前置检查**：先确认文件 `~/.flux/plugins/flux/core/skills/html-report-gen/references/ut.md` 存在。若该文件不存在，跳过本节全部步骤。

读取 `~/.flux/plugins/flux/core/skills/html-report-gen/references/ut.md`，按照其中定义的
record\_type、字段映射和写入方式，准备并写入报告物料。

### 检查点

已调用脚本写数据。

## 三、生成 html 报告

> **前置检查**：仅当 `NEED_HTML_GEN` 为 `true` 且第二步正常执行时才需要执行本步，否则跳过本节全部步骤。

1. 若 `${FEATURE_DIR}/unit-test/ut_test_report.md` 中的 `## 补测建议` 不为空，则逐一将每个条目转换为以下数据格式后追加写入到
   `${FEATURE_DIR}/report.jsonl`：

```json
{
  "record_type": "retest_item",
  "id": "UT-01",
  "source": "bits-unit-test-gen",
  "payload": {
    "title": "<标题>",
    "severity": "<优先级>",
    "sources": [
      "bits-unit-test-gen"
    ],
    "target": "<对象>",
    "action": "<补测动作>"
  }
}
```

2. 读取 `~/.flux/plugins/core/skills/html-report-gen/SKILL.md`，调用 html-report-gen skill 来渲染生成 html 报告文件。

### 检查点

html 报告文件已生成。

***

## 禁止事项

- 不得为了补全报告字段而重新运行测试、覆盖率或 `utree flush`；报告只能汇总本次流程已经执行过的结果。
- 不得在报告中暴露 `BUG_MAP`、`CANDIDATE_BUGS` 等内部变量名；缺陷信息必须转写为用户可理解的描述。
