# 说明
本阶段主要用于生成与展示单元测试生成结果。

## 执行阶段
### Report-A 失败时生成 FAILED_REASON
在执行 utree flush 之前，判断本次运行是否产生了有效的生成测试用例。如果没有生成有效用例，请总结FAILED_REASON，要求512个字符以内。
如果有生成有效的用例，FAILED_REASON留空。

### Report-B 执行 flush[强制要求]
flush 用于单测总结的本地落盘，必须执行。
```bash
FAILED_REASON="<computed_failed_reason_or_empty>" AGENT_SOURCE=<agent_name> MODEL_SOURCE=<model_name> SKILL_ROOT=${SKILL_ROOT} TMP_ROOT=${TMP_ROOT} \
$HOME/.local/bin/utree flush --repo-path ${PROJECT_ROOT}
```

### Report-C 总结生成结果报告内容：总结生成的用例、和发现的缺陷数据

报告格式如下：

```
### 测试已完成
完成 <N> 个用例生成，通过率 <percent>（<passed>/<total>），发现 <p0_count> 个 P0 阻塞缺陷、<p1_count> 个 P1 阻塞缺陷、<p2_count> 个 P2 需关注问题，建议修复后补充单测。
🔗 [查看完整测试报告](file:///<absolute_path_to_test_report.html>)

#### 🔴 高优缺陷概览
- [<Priority>] BUG-<序号> — `<函数名>` <缺陷描述>（<源码位置>）
- ...（最多展示 5 条，更多请查看完整测试报告）

#### 🟡 补测建议概览
- `<symbol>` <补测建议描述>（未覆盖行: <合并区间>）
- ...（最多展示 5 条，更多请查看完整测试报告）

#### 产物
- 测试报告：[test_report.html](file:///<absolute_path_to_test_report.html>)

#### 下一步建议
<基于缺陷和覆盖率情况给出的 1-2 句可执行建议>
```

#### 字段填写规则

- 总结行中的数据必须来自真实执行结果，不得估算。
- 若某优先级缺陷数为 0，省略该优先级的计数（如无 P0 则不展示「0 个 P0 阻塞缺陷」）。
- 若无缺陷发现，省略「高优缺陷概览」整个模块。
- 高优缺陷概览最多展示 5 条，按优先级排序（P0 > P1 > P2）。若超过 5 条，展示前 5 条并附提示「更多请查看完整测试报告」。
- **仅报告缺陷**需与测试验证过的缺陷**分区展示**：
  - 标注为「静态分析发现（难以构造测试验证）」。
  - 展示：缺陷等级、触发场景、缺陷描述、源码位置（可点击链接）、无法构造测试的原因、修复建议。
  - **不展示**：复现缺陷的单测代码位置（因为没有对应测试）。
  - 提示用户该缺陷基于高置信静态分析，建议人工 review 确认。
- 缺陷格式为 `[P0] BUG-01 — \`FuncName\` 描述`，源码位置用 `file:line` 形式。每条缺陷应包含缺陷等级、触发场景、缺陷描述、函数及源码位置、修复建议。源码位置和复现单测位置展示为可点击跳转的链接。
- 「补测建议概览」基于 Step6 输出的 `uncovered_targets`。若 Step6 跳过或 `uncovered_targets` 为空，省略该模块。最多展示 5 条，每条格式为 `` `symbol` 补测建议描述（未覆盖行: a-b,c-d）``，`uncovered_lines` 连续行号合并为区间。若超过 5 条，附提示「更多请查看完整测试报告」。
- 「查看完整测试报告」链接指向 flux 环境生成的报告文件（若有），否则省略此行。
- 「产物」仅输出测试报告，超链接至 `test_report.html`。
- 「下一步建议」根据缺陷严重度和覆盖率情况给出可操作建议（如运行 flux-bugfix、补充覆盖率等）。

## 全局约束
- 不允许跳过 Report-B 中的 flush 指令。
- 禁止在总结中提示BUG_MAP等不容易给人理解的变量，如果未发现缺陷，则不需要提示缺陷相关内容。
- 当且仅当 `EXEC_SOURCE` 为 `flux` 或 `flux-web` 时，需要根据 `${SKILL_ROOT}/assets/templates/flux_report.md` 的要求来生成报告。其他情况不需要写文件。
- 当且仅当 `${EXEC_SOURCE}` 为 `PIPELINE` 时，需要根据 `${SKILL_ROOT}/assets/artifacts/defects.md` 中的要求写入缺陷文件。