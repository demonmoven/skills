# bytedcli 通用调用方式

## 执行

```bash
# 方式 1：直接用 npx 运行最新版
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest --help
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest <command> [options]

# 方式 2：先全局安装，再直接调用 bytedcli
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npm install -g @bytedance-dev/bytedcli@latest
bytedcli --help
bytedcli <command> [options]
```

- 使用 `npx` 时，把后文示例里的 `bytedcli` 替换成 `NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest`
- 已全局安装时，直接按后文示例执行 `bytedcli ...`

## 站点切换

通过全局参数 `--site` 或环境变量 `BYTEDCLI_CLOUD_SITE` 切换 ByteCloud 站点：

| 站点值 | 说明 | SSO | 备注 |
|--------|------|-----|------|
| `cn` | 国内生产（默认） | `sso.bytedance.com` | |
| `i18n-bd` | ByteIntl 国际站 | `sso.bytedance.com` | 通常复用 cn 登录态 |
| `i18n-tt` | TikTok 国际站 | `sso.tiktok-intl.com` | 需单独登录 |
| `eu-ttp` | EU TTP 站 | `sso.tiktok-intl.com` | 需单独登录 |
| `boe` | BOE 测试 | `test-sso.bytedance.net` | |

> `--site i18n-bd` 是 ByteIntl 国际站的规范站点值（`i18n` 也可用作别名）。

**重要：认证隔离按 SSO 环境生效。`i18n-tt`、`eu-ttp`（TikTok SSO）需单独 `auth login`；`cn`、`i18n-bd`（ByteDance SSO）通常共享登录态。** 切换站点前先检查认证状态：

```bash
# 检查 i18n-tt 站点认证
BYTEDCLI_CLOUD_SITE=i18n-tt bytedcli auth status

# 登录 i18n-tt 站点（使用 TikTok SSO）
BYTEDCLI_CLOUD_SITE=i18n-tt bytedcli auth login
```

站点切换示例：

```bash
# BOE
BYTEDCLI_CLOUD_SITE=boe bytedcli <command> [options]
bytedcli --site boe <command> [options]

# i18n-tt（TikTok 国际站）
BYTEDCLI_CLOUD_SITE=i18n-tt bytedcli <command> [options]
bytedcli --site i18n-tt <command> [options]

# i18n-bd（ByteIntl 国际站）
BYTEDCLI_CLOUD_SITE=i18n-bd bytedcli <command> [options]
bytedcli --site i18n-bd <command> [options]

# eu-ttp（EU TTP 站）
BYTEDCLI_CLOUD_SITE=eu-ttp bytedcli <command> [options]
bytedcli --site eu-ttp <command> [options]
```

## JSON 输出

```bash
bytedcli --json <command> [options]
```

注意：`--json` 是全局参数，必须放在 `<command>` 前面，例如 `--json auth status`，不能写成 `auth status --json`。

调试 HTTP 请求时，可使用以下全局参数：

```bash
bytedcli --http-debug <command> [options]
bytedcli --http-print HBhbmt <command> [options]
bytedcli --http-trace-file /tmp/bytedcli.http.log --http-body-limit 4096 <command> [options]
```

其中 `--http-print <parts>` 的 flag 含义如下：
- `H`：request headers
- `B`：request body
- `h`：response headers
- `b`：response body
- `m`：meta
- `t`：time

输出结构：

```json
{
  "status": "success|error",
  "data": {"...": "..."},
  "error": "error message",
  "context": {
    "execution_time_ms": 100,
    "timestamp": "2026-01-20T14:15:57.472335"
  }
}
```

## 版本保鲜

全局安装后，bytedcli 会在命令执行时自动检查并升级到最新版本。如需手动检查：

```bash
bytedcli self update --check
bytedcli self update
```

Agent 在消费 JSON 输出时，应检查 `_upgrade` 字段。如果 `cli_stale` 或 `skills_stale` 为 `true`，说明当前版本已过期，升级将在下次命令执行时自动完成。
