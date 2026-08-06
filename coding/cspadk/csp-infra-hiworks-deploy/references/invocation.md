# Unified `bytedcli` invocation

This document provides a standard way to run `cspadk` / `bytedcli` commands with the correct registry, site selection, and authentication.

## Run

```bash
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest --help
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest <command> [options]
```

## Authentication (required)
Before running any command that calls ByteCloud / BITS / HiWorks APIs, make sure you are authenticated for the target site.
```bash
# Check auth status (default site)
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest auth status
# Login if needed (default site)
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest auth login
```
For non-default sites, set `BYTEDCLI_CLOUD_SITE` (or use `--site` on `cspadk` commands):
```bash
# Example: i18n-tt (TikTok SSO)
BYTEDCLI_CLOUD_SITE=i18n-tt NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest auth status
BYTEDCLI_CLOUD_SITE=i18n-tt NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest auth login
```

If you provide `--meego` and expect the CLI to auto-fetch information from Meego, you also need to authenticate Meego:

```bash
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest meego status
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest meego login
```

## Switch ByteCloud site

Switch ByteCloud sites with the global flag `--site` or environment variable `BYTEDCLI_CLOUD_SITE`:

| Site value | Description | SSO | Notes |
|-----------|-------------|-----|------|
| `prod` | Mainland production (default) | `sso.bytedance.com` | |
| `i18n-tt` | TikTok international | `sso.tiktok-intl.com` | Requires separate login |
| `i18n-bd` | ByteIntl international | `sso.bytedance.com` | Usually shares session with `prod` |
| `boe` | BOE test | `test-sso.bytedance.net` | |
| `boei18n` | BOE international test | `sso.bytedance.com` | Usually shares session with `prod` |

> `--site i18n` is normalized to `i18n-bd`.

**Important: authentication isolation follows the SSO environment.** `i18n-tt` (TikTok SSO) requires a dedicated `auth login`. `prod`, `i18n-bd`, and `boei18n` (ByteDance SSO) usually share the login session. Before switching sites, check your auth status:

```bash
# Check authentication for i18n-tt
BYTEDCLI_CLOUD_SITE=i18n-tt NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest auth status

# Login to i18n-tt (TikTok SSO)
BYTEDCLI_CLOUD_SITE=i18n-tt NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest auth login
```

Examples:

```bash
# BOE
BYTEDCLI_CLOUD_SITE=boe NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest <command> [options]
NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest --site boe <command> [options]

# i18n-tt (TikTok international)
BYTEDCLI_CLOUD_SITE=i18n-tt NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest <command> [options]
NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest --site i18n-tt <command> [options]

# i18n-bd (ByteIntl international)
BYTEDCLI_CLOUD_SITE=i18n-bd NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest <command> [options]
NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest --site i18n-bd <command> [options]

# BOEi18n
BYTEDCLI_CLOUD_SITE=boei18n NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest <command> [options]
NPM_CONFIG_REGISTRY=http://bnpm.byted.org \
  npx -y @bytedance-dev/bytedcli@latest --site boei18n <command> [options]
```

## JSON output

```bash
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest --json <command> [options]
```

Note: `--json` is a global flag and must appear before `<command>`, for example `--json auth status` (not `auth status --json`).

## HTTP debugging

```bash
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest --http-debug <command> [options]
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest --http-print HBhbmt <command> [options]
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @cspadk/cli@latest --http-trace-file /tmp/cli.http.log --http-body-limit 4096 <command> [options]
```

The `--http-print <parts>` flag accepts:

- `H`: request headers
- `B`: request body
- `h`: response headers
- `b`: response body
- `m`: meta
- `t`: time

Output shape:

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
