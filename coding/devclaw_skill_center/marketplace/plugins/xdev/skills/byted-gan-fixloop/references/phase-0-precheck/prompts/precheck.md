# Phase 0 / Precheck — 前置检查执行指南

> 主 context 直接 Read 本文件，按以下流程执行前置检查。
> 不启动 subagent。

## 流程

```bash
mkdir -p $OUTPUT_DIR

# 清理上一次 fix-loop 的残留产出（仅在前置检查时执行一次）
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
# 解析 TEST_REPO → TEST_REPO_PATH（详见 ../../_shared/parameter-defaults.md）

# 发现 bytedcli skills 目录（兼容 claude/codex/trae 等所有 agent）
BYTEDCLI_SKILLS_DIR=""
for dir in .*/ "$HOME"/.*/ ; do
  if [ -f "${dir}skills/bytedance-tce/SKILL.md" ]; then
    BYTEDCLI_SKILLS_DIR="${dir}skills"
    break
  fi
done
if [ -z "$BYTEDCLI_SKILLS_DIR" ]; then
  echo "警告：未找到 bytedcli skills 目录，bytedcli 命令需手动查阅 --help"
fi

# bytedcli 认证检查
# 参考 $BYTEDCLI_SKILLS_DIR/bytedance-auth/SKILL.md 获取认证命令用法
# 先 --help 确认参数，然后检查 prod 和 BOE 两个站点的认证状态
# 如果任一站点未认证，执行登录；登录失败则输出错误信息并终止流程

# AGW service-id 查找（仅执行一次，缓存结果）
# 参考 $BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md 获取 service 搜索命令用法
# 用 PSM 作为关键字搜索，提取第一个非空的 service_id
AGW_SERVICE_ID=""
# ... 执行搜索命令，解析 JSON 结果 ...

if [ -z "$AGW_SERVICE_ID" ]; then
  echo "提示：未找到 PSM=$PSM 对应的 AGW service-id，将跳过 AGW IDL 同步步骤"
fi

# 校验参数组合
if [ "$SKIP_DEPLOY" = "true" ] && [ -z "$TCE_LANE" ]; then
  echo "错误：SKIP_DEPLOY=true 必须同时指定 TCE_LANE 参数，否则测试请求无法路由到目标泳道"
  # 终止流程
fi

# TEST_ENV 初始化
TEST_ENV = "fornax_boe"   # 默认值，测试阶段遇到 404 时会自动切换为 boe

ls $SPEC_DIR && ls $BUSINESS_REPO_PATH && ls $TEST_REPO_PATH && ls $IDL_REPO_PATH
```

## 完成后

主 context 应当持有以下变量供后续 phase 使用：
- `BUSINESS_REPO_PATH`、`TEST_REPO_PATH`、`IDL_REPO_PATH`
- `BYTEDCLI_SKILLS_DIR`（可能为空，下游 stage 容错处理）
- `AGW_SERVICE_ID`（可能为空，phase-2 stage-1.5 据此跳过）
- `TEST_ENV` = `fornax_boe`
- bytedcli 双站点认证已通过

进入 phase-1（如果 `GENERATE_TESTS=true`）或 phase-2（否则）。
