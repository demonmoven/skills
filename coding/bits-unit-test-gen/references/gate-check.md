# 覆盖率统计与补充

## 目录

- [约定](#约定)
- [前置检查](#前置检查)
- [Phase 1 覆盖率统计检查](#phase-1-覆盖率统计检查)
- [Phase 2 快速补充单测](#phase-2-快速补充单测)
  - [规范要求](#规范要求)
  - [约束](#约束)
  - [工作流](#工作流)
- [输出](#输出)
- [覆盖率统计方法](#覆盖率统计方法)

本阶段对生成的单测进行覆盖率统计和可选补充。

**回溯禁令**：

- 禁止因本步骤的覆盖率结果返回修改 Step4 的 BUG_MAP。
- 禁止因本步骤的覆盖率目标在 Phase 2 中修改或删除 Step5 已生成的测试。
- 禁止修改业务源码（与 Step5 约束一致）。
- 覆盖率未达标是可接受的结束状态，不是需要用"一切手段"解决的阻塞。

## 约定

> ⚠️ 与 Step5 不同：本阶段以覆盖率模式运行单测和补充单测时，不按语言 prompt 所声明的最小调度单元（如 Go 的
> package）拆分执行，必须将所有 CHECK_TARGETS 合并为一次执行。

- 增量覆盖率定义：CHECK_TARGETS 中被测试覆盖的新增/变更行数 ÷ CHECK_TARGETS 中总新增/变更行数 × 100%；若为 non_diff 范围，则为 CHECK_TARGETS
  中被测试覆盖的可执行行数 ÷ CHECK_TARGETS 中总可执行行数 × 100%。
- 增量覆盖率统计范围为全部 CHECK_TARGETS；但判断是否达标和快速补充单测时，必须剔除以下 CHECK_TARGETS：
  - Step5 结论为 `未收敛` 或 `禁止修复` 的 TARGETS。
  - Step5 执行过程中因运行崩溃（如 stack overflow、确定性 panic）导致无法执行测试的包下的全部 TARGETS。
- 禁止使用包级或文件级整体覆盖率代替增量覆盖率。
- 禁止修改业务源码。
- 未通过验证的用例不计入覆盖率统计。

## 前置检查

### 1. 加载覆盖率配置

满足以下前提条件**之一**时，执行覆盖率加载：
1. `AGENTS.md`、`CLAUDE.md` 提到单测生成要以满足 CI 覆盖率卡口为目标。
2. 用户原始请求中明确要求生成单测需满足 CI 覆盖率卡口。
3. `EXEC_SOURCE` 值为 `flux` 或 `flux-web`。

所有条件都不满足时，跳过覆盖率加载，`cov_config` 设为 null。

执行以下命令加载覆盖率配置：

```bash
cd ${PROJECT_ROOT} && $HOME/.local/bin/utree coverage load-cov-config \
  --repo_path "${PROJECT_ROOT}" \
  --paths "<TARGETS 中所有 file_path，逗号分隔>" \
  [--preferred_ci_file "<AGENTS.md 或 CLAUDE.md 中声明的 CI 文件，无声明则不传此 flag>"]
```

- 若命令输出为 `null`，`cov_config` 设为 null。
- 若命令输出为 JSON，解析为 `cov_config`。
- 若命令不可用或执行报错，`cov_config` 设为 null。

### 2. 覆盖率规则过滤

前置条件：`cov_config` 不为 null 且 `cov_config.rules` 非空，并且满足以下任一条件：

1. `scope_type` 为 `diff`。
2. `scope_type` 为 `non_diff` 且指定的生成对象非单一函数或单一文件（如目录、包、模块或多个文件/函数）。

不满足前置条件则跳过本步骤，`CHECK_TARGETS` = `TARGETS`。

满足前置条件时：

1. 使用 Step3 输出的 TARGETS 组装 `${TMP_ROOT}/ut_filter_targets_input.json`：

```json
{
  "TARGETS": [<TARGETS 数组>],
  "cov_config": <cov_config 或 null>
}
```

2. 执行命令：

```bash
cd ${PROJECT_ROOT} && $HOME/.local/bin/utree coverage filter-by-rules \
  --targets_input_file ${TMP_ROOT}/ut_filter_targets_input.json \
  -o ${TMP_ROOT}/check_targets.json
```

- 命令输出文件 `${TMP_ROOT}/check_targets.json` 即为 `CHECK_TARGETS`（JSON 数组，结构与 TARGETS 相同）。
- 若命令输出为空数组或命令不可用或执行报错，`CHECK_TARGETS` = `TARGETS`。

后续 Phase 1、Phase 2 中所有覆盖率统计和补充操作均针对 `CHECK_TARGETS`，而非全部 `TARGETS`。

### 3. 确定检查模式

根据以下优先级确定 `CHECK_COV_MODE`：

1. 若 `cov_config` 不为 null 且 `cov_config.coverage_threshold` > 0，`CHECK_COV_MODE = enforce`。
2. 否则，若 `EXEC_SOURCE` 为 `flux` 或 `flux-web`，`CHECK_COV_MODE = report`。
3. 以上都不满足，`CHECK_COV_MODE = skip`。

若 `CHECK_COV_MODE = skip`，不执行任何覆盖率相关操作，直接结束本 Step。

## Phase 1 覆盖率统计检查

`CHECK_COV_MODE` 为 `enforce` 或 `report` 时执行。

1. 以覆盖率模式**一次性统一运行所有 CHECK_TARGETS 对应的测试**（禁止按语言最小调度单元拆分为多次执行），生成覆盖率数据文件。
2. 按「覆盖率统计方法」统计增量覆盖率。
3. 记录增量覆盖率。
4. 若 `CHECK_COV_MODE = report`，结束本 Step。
5. 若 `CHECK_COV_MODE = enforce`，判断增量覆盖率是否 >= `cov_config.coverage_threshold`：
  - 达标：门禁通过，结束本 Step。
  - 未达标：进入 Phase 2。

## Phase 2 快速补充单测

仅在 `CHECK_COV_MODE = enforce` 且 Phase 1 未达标时执行。最多执行 2 轮。

本阶段的职责补充用例使得未覆盖行被覆盖从而达成覆盖率要求，无需考虑缺陷分析、修复单测等。

### 规范要求

你需要重新回顾 Step2 中加载的项目单测约定和语言 prompt 规范（测试风格、断言库、命名、表驱动结构等），保证生成的用例满足规范要求。

"无需考虑缺陷分析"仅指跳过 BUG_MAP 挖掘和缺陷探测用例生成，不允许降低用例的断言质量和结构规范。

### 约束

**禁止事项**：

- 禁止修复运行失败的单测。
- 禁止修改先前阶段生成的单测。
- 禁止修改历史单测。
- 禁止修改业务源码。
- 禁止使用无意义断言（如仅断言 ShouldNotBeEmpty、ShouldNotBeNil、ShouldBeGreaterThan 0）代替具体期望值。每个用例必须基于代码逻辑推导出确定的正确期望值并断言。

**补充用例验证 - 硬性约束**：

补充的用例写入后，必须运行验证。若验证命令的退出码非零或存在新补充用例 FAIL（无论原因），立即执行以下操作：

1. 按照 Step5 Loop-C 的流程处置失败。
2. 若最后归类为 `禁止修复`：对于包级失败（即 panic、TestMain 失败等），删除此包下所有新补充的用例；对于用例级失败，移除此新补充用例。
3. 本轮视为无有效补充，直接进入下一轮或结束 Phase 2。

### 工作流

每轮执行：

1. 根据上次「覆盖率统计方法」输出的 `details`，定位 CHECK_TARGETS 中未被覆盖的新增/变更行（non_diff 范围下为未被覆盖的可执行行）。
2. 仅以覆盖这些未覆盖行为目标，**一次性统一生成所有补充单测**（禁止按语言最小调度单元逐个生成）。
3. **一次性统一验证所有补充的用例**（禁止按语言最小调度单元逐个验证）。按上述「补充用例验证」约束处理结果。
4. 若有用例保留，重新以覆盖率模式运行全部测试，按「覆盖率统计方法」重新统计增量覆盖率。
5. 若达标，门禁通过，结束本阶段。
6. 若仍未达标（或本轮无有效用例保留），进入下一轮。

2 轮后仍未达标，记录当前覆盖率，标记为未达标，结束本阶段。

## 输出

- `diff_coverage`：最终的增量单测覆盖率（百分比数值）。若调用了 `utree coverage parse` 命令，取其输出的 `all_target_coverage_rate`。
- `status`：覆盖率门禁检查状态，跳过/仅统计/未通过/通过。
- `unpasss_reason`：门禁检查状态为 `未通过` 时，总结的未通过原因。
- `uncovered_targets`：增量覆盖率未达标的 TARGETS 列表。从覆盖率统计结果（若调用了 `utree coverage parse` 则取输出的 `gate_targets_coverage.details`）中筛选增量覆盖率小于 `cov_config.coverage_threshold`（若 `cov_config` 为 null 则阈值取 0.8）的条目，每条包含：
  - `file_path`：源文件路径。
  - `symbol`：被测符号名称（函数/方法名）。
  - `total_lines`：总新增/变更行数（non_diff 范围下为总可执行行数）。
  - `hit_lines`：被测试覆盖的行数。
  - `coverage_rate`：该 TARGET 的覆盖率。
  - `uncovered_lines`：未覆盖的行号列表。

## 覆盖率统计方法

优先使用 `utree coverage parse` 命令，仅当命令返回「无法选择合适解析器」时才回退到手动统计。

### 1. 准备 targets_input_file

将本 Step 前置准备阶段段加载的 `cov_config` 和 `CHECK_TARGETS` 组装为 JSON 文件（如 `${TMP_ROOT}/ut_targets_input.json`）：

```json
{
  "TARGETS": [<CHECK_TARGETS 数组>],
  "cov_config": <cov_config 或 null>
}
```

### 2. 执行命令

```bash
cd ${PROJECT_ROOT} && $HOME/.local/bin/utree coverage parse \
  --lang <语言> \
  --coverage_file <覆盖率数据文件路径> \
  --targets_input_file ${TMP_ROOT}/ut_targets_input.json \
  --to_filter_targets "<file_path>#<locator>,..." \
  -o ${TMP_ROOT}/ut_coverage_output.json
```

参数说明：

- `--lang`：项目语言（go/java/python/js/ts/cpp/scala/kotlin）。
- `--coverage_file`：测试运行产出的覆盖率原始文件路径（如 Go 的 `coverage.out`）。
- `--targets_input_file`：上一步生成的 JSON 文件。
- `--to_filter_targets`：需要从门禁判定中剔除的 CHECK_TARGETS（Step5 结论为 `未收敛`/`禁止修复`，或因运行崩溃无法执行的包下的全部 TARGETS），格式为 `${file_path}#${locator}` 逗号分隔。
- `-o`：输出文件路径。

### 3. 解读输出

命令输出 JSON 结构：

```json
{
  "all_target_coverage_rate": 0.xx,
  "gate_targets_coverage": {
    "total_lines": N,
    "hit_lines": N,
    "coverage_rate": 0.xx,
    "details": [
      {
        "file_path": "path/to/file.go",
        "locator": "FuncName",
        "total_lines": 10,
        "hit_lines": 7,
        "coverage_rate": 0.7,
        "covered": true,
        "uncovered_lines": [15, 18, 22]
      }
    ]
  },
  "threshold": 0.xx,
  "gate_passed": true/false/null,
  "filtered_out_targets": ["file_path#locator", ...],
  "parser_used": "gotest"
}
```

- `all_target_coverage_rate`：全部 CHECK_TARGETS 的整体覆盖率。
- `gate_targets_coverage.coverage_rate`：剔除后用于门禁判定的增量覆盖率。
- `details`：逐 TARGET 的覆盖明细。
- `gate_passed`：门禁是否通过（当 `cov_config` 存在时自动判定；否则为 null）。若为 `false`，根据 `gate_targets_coverage.details[].uncovered_lines` 定位未覆盖行，在 Phase 2 中针对这些行补充单测。

### 4. Fallback：手动统计

若命令返回错误信息包含「无法选择合适解析器」，说明当前语言/覆盖率格式暂无内置 parser，此时手动解析覆盖率文件并按「约定」中的增量覆盖率定义计算。
