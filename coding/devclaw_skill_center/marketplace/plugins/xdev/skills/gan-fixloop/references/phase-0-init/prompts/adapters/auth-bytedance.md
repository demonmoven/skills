# Phase 0 / Auth / Adapter: bytedance

> PROFILE=bytedance-tce 的认证 adapter。**bytedcli 双站点认证 + AGW service-id 查找**。
>
> 由 `precheck.md` 在 PROFILE=bytedance-tce 时调用。

## 适用场景

PROFILE=bytedance-tce 时使用。需要 bytedcli 工具链准备就绪 + 双站点认证 + AGW service-id 缓存。

## 流程

### Step 1: 发现 bytedcli skills 目录

```bash
BYTEDCLI_SKILLS_DIR=""
for dir in .*/ "$HOME"/.*/ ; do
  if [ -f "${dir}skills/bytedance-tce/SKILL.md" ]; then
    BYTEDCLI_SKILLS_DIR="${dir}skills"
    break
  fi
done
if [ -z "$BYTEDCLI_SKILLS_DIR" ]; then
  echo "警告：未找到 bytedcli skills 目录"
  echo ""
  echo "请安装 bytedcli sub-skills（任选一种方式）："
  echo "  1. 一键安装（推荐）: xdex install --be --cc"
  echo "  2. 手动逐个装:"
  echo "     npx skills add git@code.byted.org:byteapi/bytedcli.git --skill bytedance-tce -a claude-code -y"
  echo "     npx skills add git@code.byted.org:byteapi/bytedcli.git --skill bytedance-agw -a claude-code -y"
  echo "     npx skills add git@code.byted.org:byteapi/bytedcli.git --skill bytedance-auth -a claude-code -y"
  echo "     npx skills add git@code.byted.org:byteapi/bytedcli.git --skill bytedance-log -a claude-code -y"
  echo ""
  echo "PROFILE=bytedance-tce 必须有 bytedcli sub-skills 才能继续。"
  exit 1
fi
```

### Step 2: bytedcli 二进制检查

```bash
if ! command -v bytedcli >/dev/null 2>&1; then
  echo "警告：未找到 bytedcli 二进制"
  echo ""
  echo "请安装 bytedcli（任选一种方式）："
  echo "  1. 全局安装（推荐）:"
  echo "     NPM_CONFIG_REGISTRY=http://bnpm.byted.org npm install -g @bytedance-dev/bytedcli@latest"
  echo "  2. npx 即用即走（每次启动较慢）:"
  echo "     无需安装，但所有 bytedcli 命令前面要加 'npx -y @bytedance-dev/bytedcli@latest'"
  echo ""
  exit 1
fi
```

### Step 3: 双站点认证检查

```bash
# 参考 $BYTEDCLI_SKILLS_DIR/bytedance-auth/SKILL.md 获取认证命令用法
# 检查 prod 和 BOE 两个站点的认证状态

# 检查 prod 站点
bytedcli auth status 2>&1
if [ $? -ne 0 ]; then
  echo "prod 站点未认证，触发登录..."
  bytedcli auth login
  if [ $? -ne 0 ]; then
    echo "ERROR: bytedcli prod 站点登录失败,终止"
    exit 1
  fi
fi

# 检查 BOE 站点
BYTEDCLI_CLOUD_SITE=boe bytedcli auth status 2>&1
if [ $? -ne 0 ]; then
  echo "BOE 站点未认证，触发登录..."
  BYTEDCLI_CLOUD_SITE=boe bytedcli auth login
  if [ $? -ne 0 ]; then
    echo "ERROR: bytedcli BOE 站点登录失败,终止"
    exit 1
  fi
fi
```

### Step 4: AGW service-id 查找

```bash
# 参考 $BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md 获取 service 搜索命令用法
# 用 PSM 作为关键字搜索

AGW_SERVICE_ID=""
# 执行搜索命令，解析 JSON 提取第一个非空 service_id
result=$(bytedcli --json --site $BYTEDCLI_SITE agw service search --keyword "$PSM" 2>/dev/null)
if [ -n "$result" ]; then
  AGW_SERVICE_ID=$(echo "$result" | jq -r '.data[0].service_id // empty' 2>/dev/null)
fi

if [ -z "$AGW_SERVICE_ID" ]; then
  echo "提示：未找到 PSM=$PSM 对应的 AGW service-id，将跳过 AGW IDL 同步步骤"
fi
```

## 产出

主 context 持有：
- `BYTEDCLI_SKILLS_DIR`（已校验存在）
- bytedcli 双站点认证已通过
- `AGW_SERVICE_ID`（可能为空）

## 失败处理

- bytedcli 二进制未安装 → **阻断主流程**，提示安装命令
- bytedcli sub-skills 未安装 → **阻断主流程**，提示安装命令
- 双站点认证失败 → **阻断主流程**
- AGW service-id 未找到 → 仅 warn，phase-2 stage-1 的 AGW 同步段会跳过
