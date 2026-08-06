# Stage 2 / Test / Adapter: none

> PROFILE=none 的测试 adapter。**直接用 BASE_URL 替换测试 config 中的 endpoint**，跑 go test。
>
> 主 context 直接 Read 本文件，按以下流程执行。

## 适用场景

PROFILE=none 时使用。服务已经部署在 BASE_URL（来自 phase-0 wizard 收集），不需要 TCE 泳道路由，直接用 BASE_URL 当测试入口。

## 输入

- `BASE_URL`：服务地址（来自 phase-0 wizard）
- `TEST_REPO_PATH`：测试仓库路径
- `TEST_SCOPE`：测试范围（来自 craft 的 test_dirs.txt 或 fallback `./...`）
- `BUSINESS_REPO_PATH`：业务仓库路径（用于单元测试）
- `UNIT_TEST_SCOPE`：单元测试范围
- `ITERATION`、`OUTPUT_DIR`：迭代上下文

## 与 bytedance-tce adapter 的核心差异

| 维度 | bytedance-tce | none |
|------|--------|------|
| 测试入口注入方式 | 在 `test_env/${TEST_ENV}.config.yaml` 注入 `x-tt-env: $TCE_LANE` header | **直接修改 test config 把所有 endpoint 替换为 BASE_URL** |
| 配置文件路径 | `test_env/fornax_boe.config.yaml` 或 `boe.config.yaml` | `test_env/local.config.yaml` 或用户指定 |
| TEST_ENV 环境变量 | `fornax_boe`（默认）/ `boe` | `local` 或不设置 |
| 404 回退 | 有（fornax_boe → boe） | 无 |
| LogID 提取 | 仍提取（用 X-Tt-Logid 等正则） | 仍提取（如有的话） |

## 流程

### Step 1: 探索测试仓库

与 bytedance-tce adapter 相同：根据 `TEST_SCOPE` 在 `TEST_REPO_PATH` 中定位测试函数和目录。

### Step 2: 准备测试配置（直接 BASE_URL 注入）

```bash
# 找 config 文件（按优先级）
CONFIG_CANDIDATES=(
  "$TEST_REPO_PATH/test_env/local.config.yaml"
  "$TEST_REPO_PATH/test_env/dev.config.yaml"
  "$TEST_REPO_PATH/config/test.yaml"
  "$TEST_REPO_PATH/test/config.yaml"
)

CONFIG_FILE=""
for c in "${CONFIG_CANDIDATES[@]}"; do
  if [ -f "$c" ]; then
    CONFIG_FILE="$c"
    break
  fi
done

if [ -z "$CONFIG_FILE" ]; then
  echo "ERROR: 找不到测试 config 文件。预期路径之一: ${CONFIG_CANDIDATES[*]}"
  echo "请手动修改 test config 把 host 设置为 $BASE_URL"
  exit 1
fi

# 用 Python 把 config 中的 host 全部替换为 BASE_URL
python3 -c "
import yaml
import sys

base_url = sys.argv[1]
config_file = sys.argv[2]

with open(config_file, 'r') as f:
    config = yaml.safe_load(f)

# http_services 列表（与 bytedance-tce 同 schema）
for svc in config.get('http_services', []):
    if isinstance(svc, dict):
        svc['host'] = base_url

with open(config_file, 'w') as f:
    yaml.dump(config, f, default_flow_style=False, allow_unicode=True)

print(f'已将所有 http_services 的 host 设置为 {base_url}')
" "$BASE_URL" "$CONFIG_FILE"
```

### Step 3: 创建输出目录

```bash
mkdir -p $OUTPUT_DIR/iteration_$ITERATION
> $OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl
```

### Step 4: 执行测试

与 bytedance-tce adapter 相同（不过 TEST_ENV 用 `local` 或不设置）：

```bash
cd $TEST_REPO_PATH

env $(env | grep ^OTEL_ | sed 's/=.*/-u &/' | tr '\n' ' ') TEST_ENV=local \
  go test -json -v -count=1 -p=1 -run "^TestName$" ./dir 2>&1 | python3 ...
```

### Step 5: 解析 + 写 failed_cases.jsonl + test_stats.json

与 bytedance-tce adapter 相同，参考 `../../stage-1-deploy/lib/` 或共享的 `_shared/go-test-json-format.md`。

### Step 6: 恢复 config 文件

```bash
cd $TEST_REPO_PATH && git checkout -- "$CONFIG_FILE"
```

### Step 7: 单元测试（可选）

与 bytedance-tce adapter 完全相同（单元测试本来就不依赖部署环境）。

## 错误处理

- BASE_URL 不可达 → 已在 stage-1-deploy/none.md 中校验过，到这里应该 OK
- 测试 config 文件找不到 → 报错让用户手动准备
- 测试用例失败 → 正常写入 failed_cases.jsonl，由 stage-3 分析

## 跳过条件

无（PROFILE=none 时本 adapter 必须执行）。

## 注意事项

- 不依赖 `x-tt-env` header（none profile 没有泳道概念）
- 不依赖 `fornax_boe` 测试环境名（none profile 用 `local`）
- 不需要 404 回退逻辑（无环境差异）
- 仍然解析 LogID（如果服务返回了 X-Tt-Logid 等 header），但 stage-3 的 log-query adapter 是 none 时不会查询日志
