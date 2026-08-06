# 集成测试执行（TCE 模式）

在 TCE 内场环境下执行 Go 集成测试套件，解析结果并输出结构化报告。通过注入 `x-tt-env` header 实现泳道路由。

## 参数

| 参数 | 必填 | 默认值 | 说明 | 示例 |
|------|------|--------|------|------|
| `TEST_REPO_PATH` | 是 | — | 测试仓库绝对路径 | `/Users/xxx/cozeloop_api_test` |
| `TEST_SCOPE` | 是 | — | 测试范围（自然语言描述或结构化列表，每行 `TestFuncName ./dir` 或 `./...`） | "P0 用例"、`./...`、`TestCreate ./test_cases/PE/Create/P0` |
| `PSM` | 否 | — | 被测服务的 PSM 标识，用于上下文串联和日志关联（上层 fix-loop 编排时传入） | `stone.costudio.platform` |
| `TCE_LANE` | 否 | — | TCE 泳道名（提供时注入 x-tt-env header，使测试流量路由到指定泳道实例） | `boe_wtj_0306` |
| `ITERATION` | 否 | `1` | 当前迭代轮次（上层 fix-loop 编排时传入） | `3` |
| `OUTPUT_DIR` | 否 | `/tmp/costudio-test` | 输出基目录 | `/tmp/workspace/.costudio` |
| `BUSINESS_REPO_PATH` | 否 | — | 业务仓库绝对路径（单元测试执行目录） | `/Users/xxx/cozeloop` |
| `UNIT_TEST_SCOPE` | 否 | — | 单元测试范围（每行 `TestFuncName ./dir`），如果为空则跳过单元测试 | `TestConvertPO2DO ./domain/convertor` |
| `TEST_ENV` | 否 | `fornax_boe` | 测试环境名，决定配置文件路径 (`test_env/${TEST_ENV}.config.yaml`) 和 `go test` 环境变量。测试遇到 404 时自动回退为 `boe` 重试 | `fornax_boe`, `boe` |

**说明**：
- `PSM` 在测试执行阶段不直接传给 `go test`，但标识了被测服务。上层 fix-loop 通过 PSM 串联部署（`$BYTEDCLI tce deploy-lane --psm <PSM>`）和测试流程
- `TCE_LANE` 通常由上层部署阶段产出（部署时创建/复用泳道），单独使用 integration-test skill 时需要用户指定

## 执行步骤

### Step 1: 探索测试仓库，定位测试用例

根据用户提供的 `TEST_SCOPE` 自然语言描述，在 `TEST_REPO_PATH` 中探索并定位具体的测试目录和测试函数。

**探索策略**：

1. **先了解仓库结构**：用 Glob 查看测试仓库的目录结构
   ```
   Glob: $TEST_REPO_PATH/test_cases/*
   Glob: $TEST_REPO_PATH/test_cases/**/*_test.go
   ```

2. **根据用户描述匹配**：
   - 用户说"P0 用例" → 搜索路径中包含 `P0` 的测试目录
   - 用户说"CreateLabel 相关" → 搜索函数名或目录名包含 `CreateLabel` 的测试
   - 用户说"PE 目录下所有测试" → 定位 `test_cases/PE/` 下所有 `_test.go`
   - 用户说具体测试函数名 → 用 Grep 精确查找该函数所在文件和目录
   - 用户说"全部测试" → 使用 `./...`

3. **确认测试列表**：整理出要执行的测试条目，每条为以下格式之一：
   - `TestFuncName ./relative/dir/path` — 指定测试函数 + 目录
   - `./relative/dir/path` — 目录下所有测试
   - `./...` — 递归所有测试

4. **向用户确认**（如果范围模糊或匹配结果较多时）：列出找到的测试用例，请用户确认

### Step 2: 准备测试数据（TCE 泳道路由）

TCE 模式使用 `$TEST_ENV` 测试环境（默认 `fornax_boe`）。

如果提供了 `TCE_LANE`，需要将泳道 header 注入到测试配置中，使测试请求路由到指定泳道实例。

详细注入流程见本文档末尾的「TCE 测试数据准备」章节。

简要流程：
1. 读取 `$TEST_REPO_PATH/test_env/${TEST_ENV}.config.yaml`
2. 找到 `http_services` 列表，为每个 service 的 `headers` 添加 `x-tt-env: <TCE_LANE>`
3. 写回 YAML 文件
4. 用 `grep x-tt-env` 验证注入成功

### Step 3: 创建输出目录

```bash
mkdir -p $OUTPUT_DIR/iteration_$ITERATION
> $OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl
```

### Step 4: 逐条执行测试

对 Step 1 确定的每一条测试，在 `TEST_REPO_PATH` 目录下构造并执行 `go test` 命令。

**命令格式**：

```bash
cd $TEST_REPO_PATH

# 如果格式为 "TestName ./dir"（精确匹配测试函数）
env $(env | grep ^OTEL_ | sed 's/=.*/-u &/' | tr '\n' ' ') TEST_ENV=$TEST_ENV \
  go test -json -v -count=1 -p=1 -run "^TestName$" ./dir 2>&1

# 如果格式为 "./dir"（目录下所有测试）
env $(env | grep ^OTEL_ | sed 's/=.*/-u &/' | tr '\n' ' ') TEST_ENV=$TEST_ENV \
  go test -json -v -count=1 -p=1 ./dir 2>&1
```

**关键参数说明**：
- `-json`：机器可读的 JSON 输出格式
- `-v`：详细模式
- `-count=1`：禁用测试缓存
- `-p=1`：顺序执行（不并行）
- `-run "^TestName$"`：精确匹配测试函数名（仅在指定了测试名时使用）

**环境变量**：
- 过滤掉所有 `OTEL_*` 环境变量，避免 OpenTelemetry 干扰测试
- 设置 `TEST_ENV=$TEST_ENV`（TCE 模式固定值）

### Step 5: 解析 go test -json 输出

`go test -json` 输出为每行一个 JSON 对象。详细解析规则见 `references/go-test-json-format.md`。

**解析算法**：

1. 逐行读取 JSON 输出
2. 如果 `Action == "fail"` 且有 `FailedBuild` 字段 → 编译失败，记录为 `build_fail`
3. 忽略没有 `Test` 字段的行（包级别事件）
4. 按 `Package/Test` 为 key 跟踪每个测试用例：
   - `Action == "run"` → 开始跟踪
   - `Action == "output"` → 累积输出内容
   - `Action == "pass"` → 通过，计数 +1
   - `Action == "fail"` → 失败，**立即写入 failed_cases.jsonl**（status=`fail`）
   - `Action == "skip"` → 跳过，计数 +1，**立即写入 failed_cases.jsonl**（status=`skip`）

**写入 failed_cases.jsonl**（JSONL 格式，每行一个 JSON）：
```json
{"package":"github.com/test/pkg","test":"TestLogin","status":"fail","message":"expected 200, got 500\n...","elapsed":1.234,"start_time":"2026-03-09T10:00:00Z","end_time":"2026-03-09T10:00:01Z","log_ids":["20260309100000C91A145A63CB5F0B9D80"]}
```

**LogID 提取**：解析 `Action == "output"` 事件时，从 `Output` 内容中用正则提取 LogID（来自 HTTP 响应头 `X-Tt-Logid`），写入 `log_ids` 字段。详见 `references/go-test-json-format.md`。

**推荐方式**：用 Python 脚本管道解析 go test -json 输出，实现流式写入 JSONL + 统计：

```bash
go test -json -v -count=1 -p=1 [-run "^TestName$"] ./dir 2>&1 | python3 -c "
import sys, json, re
LOGID_PATTERN = re.compile(r'(?:X-Tt-Logid|x-tt-logid|logid|LogID)[=:\s]+([a-fA-F0-9]{20,})')
cases = {}
stats = {'pass': 0, 'fail': 0, 'skip': 0}
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    try:
        event = json.loads(line)
    except json.JSONDecodeError:
        continue
    action = event.get('Action', '')
    test = event.get('Test', '')
    pkg = event.get('Package', '')
    if action == 'fail' and event.get('FailedBuild'):
        stats['fail'] += 1
        fc = {'package': pkg, 'test': '', 'status': 'build_fail', 'message': f'Package build failed: {event[\"FailedBuild\"]}', 'elapsed': 0, 'log_ids': []}
        print(json.dumps(fc, ensure_ascii=False), file=open(sys.argv[1], 'a') if len(sys.argv) > 1 else sys.stderr)
        continue
    if not test:
        continue
    key = f'{pkg}/{test}'
    if action == 'run':
        cases[key] = {'package': pkg, 'test': test, 'outputs': [], 'start_time': event.get('Time', ''), 'log_ids': []}
    elif action == 'output':
        if key in cases:
            output_text = event.get('Output', '')
            cases[key]['outputs'].append(output_text)
            cases[key]['log_ids'].extend(LOGID_PATTERN.findall(output_text))
    elif action in ('pass', 'fail', 'skip'):
        stats[action] += 1
        if action in ('fail', 'skip') and key in cases:
            c = cases[key]
            fc = {'package': c['package'], 'test': c['test'], 'status': action, 'message': ''.join(c['outputs']), 'elapsed': event.get('Elapsed', 0), 'start_time': c.get('start_time', ''), 'end_time': event.get('Time', ''), 'log_ids': list(set(c.get('log_ids', [])))}
            print(json.dumps(fc, ensure_ascii=False), file=open(sys.argv[1], 'a') if len(sys.argv) > 1 else sys.stderr)
        if key in cases:
            del cases[key]
print(json.dumps({'total_tests': sum(stats.values()), 'passed_cases': stats['pass'], 'failed_cases': stats['fail'], 'skipped_cases': stats['skip']}), file=sys.stderr)
" "$OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl"
```

### Step 6: 写入统计文件

所有测试执行完毕后，汇总各条测试的统计数据，写入 `test_stats.json`：

```json
{"total_tests": 10, "passed_cases": 7, "failed_cases": 2, "skipped_cases": 1}
```

**多次执行的汇总方式**：每次 Python 管道脚本输出的 stderr 统计是该次执行的局部统计。全局汇总逻辑：将所有局部统计的 pass/fail/skip 分别累加，写入最终的 `test_stats.json`。可以在每次执行时将 stderr 重定向到临时文件（如 `2> /tmp/stats_N.json`），最后用脚本汇总；或者在主 context 中维护全局计数器，每次执行后累加。

### Step 7: 恢复测试数据

如果 Step 2 修改了配置文件，**必须恢复**（无论测试成功与否）：

```bash
cd $TEST_REPO_PATH && git checkout -- test_env/${TEST_ENV}.config.yaml
```

### Step 5.5: 404 回退检测

所有测试执行完毕后，检查 `$OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl` 中是否存在 404 相关错误。

**检测逻辑**：

```bash
# 检查失败用例中是否包含 404 错误
grep -i -E '(status.*(code)?.*404|404.*(not found|error)|expected.*200.*got.*404|got.*404)' \
  $OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl
```

**回退条件**（全部满足时触发）：
- 检测到 404 相关错误
- 当前 `TEST_ENV != "boe"`（避免无限循环，回退只尝试一次）

**回退步骤**：

1. 恢复配置文件：`cd $TEST_REPO_PATH && git checkout -- test_env/${TEST_ENV}.config.yaml`
2. 切换 `TEST_ENV = "boe"`
3. 清空本轮输出：`> $OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl`
4. 重新执行 Step 2（注入泳道 header 到 `test_env/${TEST_ENV}.config.yaml`）
5. 重新执行 Step 4-5（逐条执行测试 + 解析结果）
6. 返回结果中标注 `TEST_ENV` 已切换为 `boe`，通知主 context 后续迭代沿用此值

**不触发回退时**：继续正常流程。

### Step 7.5: 执行单元测试（可选）

> 当 `UNIT_TEST_SCOPE` 和 `BUSINESS_REPO_PATH` 均已提供时执行此步骤，否则跳过。

单元测试在**业务仓库**（非测试仓库）中执行，不需要 TCE 泳道路由和 `TEST_ENV`。

**执行方式**：对 `UNIT_TEST_SCOPE` 中的每一条测试，在 `BUSINESS_REPO_PATH` 目录下执行：

```bash
cd $BUSINESS_REPO_PATH

# 如果格式为 "TestName ./dir"
go test -json -v -count=1 -p=1 -run "^TestName$" ./dir 2>&1

# 如果格式为 "./dir"
go test -json -v -count=1 -p=1 ./dir 2>&1
```

**解析方式**：与 Step 5 相同——使用同样的 Python 管道脚本解析 `go test -json` 输出，将失败用例**追加**到同一个 `$OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl` 文件。

**统计合并**：单元测试的 pass/fail/skip 计数需要累加到 Step 6 的总统计中，最终写入同一个 `test_stats.json`。

### Step 8: 报告结果

打印测试结果摘要，并返回以下信息：
- **PSM**（如果提供了）: 被测服务标识
- **总测试数、通过数、失败数、跳过数**（含 E2E + 单元测试）
- **失败用例详情**：如果失败数 > 0，列出 failed_cases.jsonl 的前几条
- **TESTS_PASSED**：当 `failed_cases == 0` 时为 true
- **OUTPUT_PATH**：实际输出目录路径（如 `/tmp/costudio-test/iteration_1/`）
- **输出文件位置**：`failed_cases.jsonl` 和 `test_stats.json` 的完整路径

## 输出文件

| 文件 | 格式 | 说明 |
|------|------|------|
| `iteration_N/failed_cases.jsonl` | JSONL | 每行一个失败用例 JSON |
| `iteration_N/test_stats.json` | JSON | 测试统计摘要 |

## 注意事项

- **TCE 模式使用 `TEST_ENV=$TEST_ENV`**（默认 `fornax_boe`，遇到 404 自动回退为 `boe`），配置文件为 `test_env/${TEST_ENV}.config.yaml`
- `go test` 返回非零退出码是正常的（有测试失败时会返回 1），不要因此中止执行
- 测试用例**逐条执行**，一条失败不影响后续执行
- 失败用例必须在检测到时**立即写入** JSONL 文件，不要等所有测试完成
- 环境变量 `OTEL_*` 必须过滤掉，避免 OpenTelemetry 干扰测试
- 配置文件修改后必须通过 `git checkout` 恢复，即使测试失败也要恢复
- **单元测试**在 `BUSINESS_REPO_PATH` 中执行，不需要 `TEST_ENV` 和泳道 header 注入
- 单元测试的失败用例追加到同一个 `failed_cases.jsonl`，统计合并到同一个 `test_stats.json`

## References

- go test -json 输出格式和解析规则：`go-test-json-format.md`

## TCE 测试数据准备

### 目的

在 TCE（内场）模式下，测试请求需要通过 BOE 网关路由到指定泳道实例。通过在测试配置文件的 HTTP 服务 headers 中注入 `x-tt-env: <泳道名>` 实现流量路由。

### 配置文件位置

```
$TEST_REPO_PATH/test_env/${TEST_ENV}.config.yaml
```

### 配置文件结构

```yaml
http_services:
  - name: service_a
    host: "https://example.com"
    headers:
      Cookie: "session_key=xxx"
      Authorization: "Bearer xxx"
      # 需要注入: x-tt-env: <泳道名>
  - name: service_b
    host: "https://example2.com"
    headers:
      Cookie: "session_key=yyy"
      # 需要注入: x-tt-env: <泳道名>
```

### 注入流程

#### 方法 1: 使用 Python（推荐）

```bash
python3 -c "
import yaml, sys

lane = sys.argv[1]
config_file = sys.argv[2]

with open(config_file, 'r') as f:
    config = yaml.safe_load(f)

modified = False
for svc in config.get('http_services', []):
    if isinstance(svc, dict):
        if 'headers' not in svc or not isinstance(svc['headers'], dict):
            svc['headers'] = {}
        svc['headers']['x-tt-env'] = lane
        modified = True

if modified:
    with open(config_file, 'w') as f:
        yaml.dump(config, f, default_flow_style=False, allow_unicode=True)
    print(f'已注入泳道 header x-tt-env={lane} 到配置文件: {config_file}')
else:
    print('未找到需要注入 header 的 http_services，跳过')
" "$TCE_LANE" "$TEST_REPO_PATH/test_env/${TEST_ENV}.config.yaml"
```

#### 方法 2: 使用 yq

```bash
yq -i '.http_services[].headers."x-tt-env" = "'$TCE_LANE'"' \
  $TEST_REPO_PATH/test_env/${TEST_ENV}.config.yaml
```

#### 方法 3: 使用 Claude Code Edit 工具

直接用 Read 工具读取 YAML 文件，然后用 Edit 工具为每个 service 的 headers 部分添加 `x-tt-env: <lane>` 行。

### 恢复

测试执行完毕后，**必须恢复**配置文件到原始状态：

```bash
cd $TEST_REPO_PATH && git checkout -- test_env/${TEST_ENV}.config.yaml
```

恢复步骤在测试完成后执行（无论测试是否成功），使用 `git checkout` 确保配置文件恢复为 Git 仓库中的版本。

### 注意事项

- 配置文件名为 `${TEST_ENV}.config.yaml`（默认 `fornax_boe`，遇到 404 回退为 `boe`）
- `x-tt-env` header 的值就是泳道名（如 `boe_wtj_0306`）
- 如果 `http_services` 下的某个 service 没有 `headers` 字段，需要创建一个空 map 再添加
- 注入后请确认文件内容正确（可用 `grep x-tt-env` 快速验证）
