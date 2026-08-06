# setup-browser-cookies

> 移植自 [gstack/setup-browser-cookies](https://github.com/garrytan/gstack/blob/main/setup-browser-cookies/SKILL.md) v0.15.1.0
> 上游 frontmatter / preamble / telemetry / proactive 配置已删除，原工作流逻辑保留。
> 中间产物统一落到 `{PRIMARY_REPO}/docs/xdev/gstack/{FEATURE_NAME}/setup-browser-cookies.md`。
>
> ⚠️ **前置依赖**：需要 gstack 上游的 `browse` 二进制。在 devclaw 环境中若 `~/.claude/skills/gstack/browse/dist/browse` 不存在，需要先从 garrytan/gstack 仓库 clone 并运行 `./setup` 构建。

## 参数

- `ARG`（可选）：要直接导入的 cookie 域名（如 `github.com`）。可为空——若为空则打开交互式 picker UI 让用户手动选择要导入哪些域。

## 入参校验

读取并执行 `<skill_dir>/prompts/resolve_action_target.md`，传入：
- `ACTION_NAME = "setup-browser-cookies"`
- `ARG = {用户传入的参数}`

执行完成后获得 `PRIMARY_REPO`、`PRIMARY_BRANCH`、`FEATURE_NAME`、`SPECS_DIR`、`IS_FEATURE_BRANCH`。

> 若本 action 是被 `/gstack run` 派发（pipeline 模式），调用方已预先设置 `SPECS_DIR` 等环境变量，本步骤检测到则跳过。

## 哲学约束

执行本 action 前，读取 `<skill_dir>/resources/ethos.md` 并将其内容作为隐含约束。关键原则（针对本 action）：

- **User Sovereignty**：cookie 是敏感数据，会携带真实登录态、session token、个人身份。整个导入流程必须经过用户主动点击 picker UI 或显式传入 `--domain`，绝不能在用户不知情的情况下读取用户的 Chrome/Brave/Edge/Comet cookie 数据库。
- **Security Awareness**：在 macOS 上首次导入会触发 Keychain 对话框——必须提醒用户这是预期行为；picker UI 只显示 domain 和数量，不暴露 cookie value。

---

## 上游工作流（已清理 gstack 私有基础设施）

# Setup Browser Cookies

Import logged-in sessions from your real Chromium browser into the headless browse session.

## CDP mode check

First, check if browse is already connected to the user's real browser:
```bash
$B status 2>/dev/null | grep -q "Mode: cdp" && echo "CDP_MODE=true" || echo "CDP_MODE=false"
```
If `CDP_MODE=true`: tell the user "Not needed — you're connected to your real browser via CDP. Your cookies and sessions are already available." and stop. No cookie import needed.

## How it works

1. Find the browse binary
2. Run `cookie-import-browser` to detect installed browsers and open the picker UI
3. User selects which cookie domains to import in their browser
4. Cookies are decrypted and loaded into the Playwright session

## Steps

### 1. Find the browse binary

## SETUP (run this check BEFORE any browse command)

```bash
_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
B=""
[ -n "$_ROOT" ] && [ -x "$_ROOT/.claude/skills/gstack/browse/dist/browse" ] && B="$_ROOT/.claude/skills/gstack/browse/dist/browse"
[ -z "$B" ] && B=~/.claude/skills/gstack/browse/dist/browse
if [ -x "$B" ]; then
  echo "READY: $B"
else
  echo "NEEDS_SETUP"
fi
```

If `NEEDS_SETUP`:
1. Tell the user: "gstack browse needs a one-time build (~10 seconds). OK to proceed?" Then STOP and wait.
2. Run: `cd <SKILL_DIR> && ./setup`
3. If `bun` is not installed:
   ```bash
   if ! command -v bun >/dev/null 2>&1; then
     BUN_VERSION="1.3.10"
     BUN_INSTALL_SHA="bab8acfb046aac8c72407bdcce903957665d655d7acaa3e11c7c4616beae68dd"
     tmpfile=$(mktemp)
     curl -fsSL "https://bun.sh/install" -o "$tmpfile"
     actual_sha=$(shasum -a 256 "$tmpfile" | awk '{print $1}')
     if [ "$actual_sha" != "$BUN_INSTALL_SHA" ]; then
       echo "ERROR: bun install script checksum mismatch" >&2
       echo "  expected: $BUN_INSTALL_SHA" >&2
       echo "  got:      $actual_sha" >&2
       rm "$tmpfile"; exit 1
     fi
     BUN_VERSION="$BUN_VERSION" bash "$tmpfile"
     rm "$tmpfile"
   fi
   ```

### 2. Open the cookie picker

```bash
$B cookie-import-browser
```

This auto-detects installed Chromium browsers and opens
an interactive picker UI in your default browser where you can:
- Switch between installed browsers
- Search domains
- Click "+" to import a domain's cookies
- Click trash to remove imported cookies

Tell the user: **"Cookie picker opened — select the domains you want to import in your browser, then tell me when you're done."**

### 3. Direct import (alternative)

If the user specifies a domain directly (e.g., `/setup-browser-cookies github.com`), skip the UI:

```bash
$B cookie-import-browser comet --domain github.com
```

Replace `comet` with the appropriate browser if specified.

### 4. Verify

After the user confirms they're done:

```bash
$B cookies
```

Show the user a summary of imported cookies (domain counts).

## Notes

- On macOS, the first import per browser may trigger a Keychain dialog — click "Allow" / "Always Allow"
- On Linux, `v11` cookies may require `secret-tool`/libsecret access; `v10` cookies use Chromium's standard fallback key
- Cookie picker is served on the same port as the browse server (no extra process)
- Only domain names and cookie counts are shown in the UI — no cookie values are exposed
- The browse session persists cookies between commands, so imported cookies work immediately

---

## 产物落地

本 action 的关键输出必须写入 `{SPECS_DIR}/setup-browser-cookies.md`：

- 若文件已存在则**在文件末尾追加**新一轮的输出（用 `## 第 N 次执行 ({YYYY-MM-DD HH:MM:SS+TZ})` 作为分隔标题）
- 若文件不存在则创建并写入完整结构
- 该 action 产生的导入 cookie 域名列表、数量摘要等附件统一放到 `{SPECS_DIR}/artifacts/`，并在产物文件中以相对路径引用（**不要写入任何 cookie value**）

执行完成后，输出一行简短摘要：

```
✓ setup-browser-cookies 完成。报告已写入 {SPECS_DIR}/setup-browser-cookies.md
```
