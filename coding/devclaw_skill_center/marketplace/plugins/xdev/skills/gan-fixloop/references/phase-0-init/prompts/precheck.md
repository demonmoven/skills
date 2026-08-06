# Phase 0 / Precheck — 前置检查执行指南（PROFILE 适配）

> 主 context 直接 Read 本文件，按以下流程执行前置检查。
> 不启动 subagent。
> **本流程在 wizard（interactive-collect.md）完成参数收集之后执行，主 context 已经持有 PROFILE 变量。**

## 通用流程（所有 PROFILE 都执行）

```bash
mkdir -p $OUTPUT_DIR

# 清理上一次 fix-loop 的残留产出
if [ -d "$OUTPUT_DIR/craft" ]; then
  echo "清理上次 craft 产出: $OUTPUT_DIR/craft/"
  rm -rf "$OUTPUT_DIR/craft"
fi
for dir in "$OUTPUT_DIR"/iteration_[0-9]*; do
  if [ -d "$dir" ]; then
    echo "清理上次迭代数据: $dir/"
    rm -rf "$dir"
  fi
done

# 解析 BUSINESS_REPO → BUSINESS_REPO_PATH + GIT_REPO
# 解析 TEST_REPO → TEST_REPO_PATH
# 解析 IDL_REPO → IDL_REPO_PATH（仅 PROFILE=bytedance-tce 时）
# 详见 ../../_shared/parameter-defaults.md
```

## PROFILE 分支：认证 + 工具链检查

根据 wizard 中用户选择的 `PROFILE` 变量，Read 对应的 auth adapter 文件并执行：

| PROFILE | adapter 文件 | 行为 |
|---------|------|------|
| `bytedance-tce` | `adapters/auth-bytedance.md` | bytedcli 二进制检查 + sub-skills 检查 + 双站点认证 + AGW service-id 查找 |
| `none` | `adapters/auth-none.md` | 跳过所有 byted 相关检查 |

```bash
# 主 context 根据 PROFILE 读对应 adapter
if [ "$PROFILE" = "bytedance-tce" ]; then
  # Read adapters/auth-bytedance.md 并执行
  # 完成后主 context 持有: BYTEDCLI_SKILLS_DIR, AGW_SERVICE_ID, 双站点认证
  :
elif [ "$PROFILE" = "none" ]; then
  # Read adapters/auth-none.md（noop）
  echo "PROFILE=none: 跳过 bytedcli 检查"
fi
```

## 通用流程（所有 PROFILE 都执行）— 续

```bash
# 校验参数组合（仅 PROFILE=bytedance-tce 时有 SKIP_DEPLOY/TCE_LANE）
if [ "$PROFILE" = "bytedance-tce" ]; then
  if [ "$SKIP_DEPLOY" = "true" ] && [ -z "$TCE_LANE" ]; then
    echo "错误：SKIP_DEPLOY=true 必须同时指定 TCE_LANE 参数"
    exit 1
  fi
  
  # TEST_ENV 初始化
  TEST_ENV="${TEST_ENV:-fornax_boe}"
elif [ "$PROFILE" = "none" ]; then
  # PROFILE=none 时校验 BASE_URL
  if [ -z "$BASE_URL" ]; then
    echo "错误：PROFILE=none 必须指定 BASE_URL 参数"
    exit 1
  fi
  
  TEST_ENV="local"
fi

# 验证目录可达
ls $SPEC_DIR && ls $BUSINESS_REPO_PATH && ls $TEST_REPO_PATH
if [ "$PROFILE" = "bytedance-tce" ]; then
  ls $IDL_REPO_PATH
fi
```

## 完成后

主 context 应当持有以下变量供后续 phase 使用：

**通用变量**：
- `BUSINESS_REPO_PATH`、`TEST_REPO_PATH`
- `PROFILE`（来自 wizard）

**PROFILE=bytedance-tce 专用变量**：
- `IDL_REPO_PATH`、`PSM`、`IDL_BRANCH`
- `BYTEDCLI_SKILLS_DIR`
- `AGW_SERVICE_ID`（可能为空）
- `TEST_ENV` = `fornax_boe`
- bytedcli 双站点认证已通过

**PROFILE=none 专用变量**：
- `BASE_URL`
- `TEST_ENV` = `local`

进入 phase-1（如果 `GENERATE_TESTS=true`）或 phase-2（否则）。
