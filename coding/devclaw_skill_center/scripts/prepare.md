# 环境准备说明

本文档用于在另一台机器上准备 Claude Code + LiteLLM 代理环境。

默认前提：

- 当前仓库已经同步到目标机器。
- 以下命令默认在本文件所在目录执行，即 `devclaw_skills_center/scripts/`。
- `scripts/.claude/settings.json`、`scripts/.litellm/litellm-config.yaml`、`scripts/.litellm/litellm_custom_callbacks.py`、`scripts/bin/cca`、`scripts/bin/lll` 已存在。

交互说明：

- 本文档是人工执行手册，不是可直接自动运行的脚本。
- 涉及 `sudo` 的命令，需要当前操作者在终端里亲自执行，并按系统提示输入密码确认。
- 涉及 `api_key` 的步骤，需要当前操作者在终端里手动输入；输入完成后，再由后续命令把该值替换进本机的 `~/.litellm/litellm-config.yaml`。
- 不要把用户输入的真实 `api_key` 回写到仓库里的模板文件。

先进入目录：

```bash
cd /path/to/devclaw_skills_center/scripts
pwd
ls -la
```

建议先确认文件都在：

```bash
test -f ./prepare.md
test -f ./.claude/settings.json
test -f ./.litellm/litellm-config.yaml
test -f ./.litellm/litellm_custom_callbacks.py
test -f ./bin/cca
test -f ./bin/lll
```

## 1. 安装并更新 LiteLLM

先检查 `python3` 和 `uv`：

```bash
if command -v python3 >/dev/null 2>&1; then
  python3 --version
else
  echo "python3 未安装，请先补齐 Python 3 环境"
fi

if command -v uv >/dev/null 2>&1; then
  uv --version
else
  echo "uv 未安装，开始安装"
  curl -LsSf https://astral.sh/uv/install.sh | sh
fi
```

安装脚本执行后，当前 shell 可能还没刷新 PATH。手动补一遍：

```bash
export PATH="$HOME/.local/bin:$PATH"
hash -r
uv --version
```

安装或升级 LiteLLM Proxy：

```bash
uv tool install --upgrade "litellm[proxy]"
hash -r
command -v litellm
litellm --version
```

## 2. 安装 Claude Code

```bash
curl -fsSL https://claude.ai/install.sh | bash
hash -r
command -v claude
claude --version
```

如果 `claude` 仍然找不到，先开一个新 shell，再执行一次：

```bash
export PATH="$HOME/.local/bin:$PATH"
hash -r
command -v claude
```

## 3. 安装 `cca`

`cca` 是本机 wrapper，作用是执行：

```bash
claude --dangerously-skip-permissions
```

把仓库内的 wrapper 安装到 `/usr/local/bin/cca`：

以下命令需要当前操作者手动执行，并在系统弹出提示时自行完成 `sudo` 密码输入：

```bash
sudo install -m 755 ./bin/cca /usr/local/bin/cca
type cca
sed -n '1,20p' /usr/local/bin/cca
```

## 4. 复制 `.claude/settings.json`

先备份目标机器已有配置，再覆盖：

```bash
mkdir -p "$HOME/.claude"

if [ -f "$HOME/.claude/settings.json" ]; then
  cp "$HOME/.claude/settings.json" "$HOME/.claude/settings.json.bak.$(date +%Y%m%d-%H%M%S)"
fi

cp ./.claude/settings.json "$HOME/.claude/settings.json"
ls -l "$HOME/.claude/settings.json"
grep -n 'ANTHROPIC_BASE_URL' "$HOME/.claude/settings.json"
```

## 5. 安装 `lll`

`lll` 是本机 LiteLLM wrapper，目标机器上会固定读取：

- `$HOME/.litellm/litellm-config.yaml`
- 端口 `4000`

安装到 `/usr/local/bin/lll`：

以下命令需要当前操作者手动执行，并在系统弹出提示时自行完成 `sudo` 密码输入：

```bash
sudo install -m 755 ./bin/lll /usr/local/bin/lll
type lll
sed -n '1,80p' /usr/local/bin/lll
```

## 6. 复制 LiteLLM 配置和 callback

先备份，再复制：

```bash
mkdir -p "$HOME/.litellm"

if [ -f "$HOME/.litellm/litellm-config.yaml" ]; then
  cp "$HOME/.litellm/litellm-config.yaml" "$HOME/.litellm/litellm-config.yaml.bak.$(date +%Y%m%d-%H%M%S)"
fi

if [ -f "$HOME/.litellm/litellm_custom_callbacks.py" ]; then
  cp "$HOME/.litellm/litellm_custom_callbacks.py" "$HOME/.litellm/litellm_custom_callbacks.py.bak.$(date +%Y%m%d-%H%M%S)"
fi

cp ./.litellm/litellm-config.yaml "$HOME/.litellm/litellm-config.yaml"
cp ./.litellm/litellm_custom_callbacks.py "$HOME/.litellm/litellm_custom_callbacks.py"

ls -l "$HOME/.litellm/litellm-config.yaml"
ls -l "$HOME/.litellm/litellm_custom_callbacks.py"
grep -n 'litellm_custom_callbacks' "$HOME/.litellm/litellm-config.yaml"
```

注意：当前仓库里的 `litellm-config.yaml` 已经把真实 `api_key` 替换成占位符 `__LITELLM_API_KEY__`。

## 7. 输入 `api_key` 并写入配置

以下步骤需要当前操作者在终端里手动输入真实 `api_key`。输入完成后，后续命令会把 `~/.litellm/litellm-config.yaml` 里的 `__LITELLM_API_KEY__` 占位符替换成刚输入的值：

```bash
read -r -s -p "请输入 LiteLLM api_key: " LITELLM_API_KEY
echo

if [ -z "$LITELLM_API_KEY" ]; then
  echo "api_key 不能为空" >&2
  exit 1
fi

tmp_file="$(mktemp)"
escaped_api_key="$(printf '%s' "$LITELLM_API_KEY" | sed -e 's/[\\/&]/\\&/g')"

sed "s/__LITELLM_API_KEY__/${escaped_api_key}/g" \
  "$HOME/.litellm/litellm-config.yaml" > "$tmp_file"

mv "$tmp_file" "$HOME/.litellm/litellm-config.yaml"
unset LITELLM_API_KEY escaped_api_key tmp_file

grep -n '__LITELLM_API_KEY__\|api_key:' "$HOME/.litellm/litellm-config.yaml"
```

如果输出里还残留 `__LITELLM_API_KEY__`，说明替换没有成功，不要继续启动服务。

## 8. 验收

先启动 LiteLLM：

```bash
lll
```

另开一个终端，检查命令都已经就绪：

```bash
type litellm
type claude
type cca
type lll
```

如果 `lll` 已正常启动，再执行：

```bash
cca
```

如果 Claude Code 能通过 `http://0.0.0.0:4000` 正常走通请求，这套准备就完成了。
