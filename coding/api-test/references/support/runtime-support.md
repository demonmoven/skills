# 运行支撑

本文档是 `api-test` 的运行支撑单一源，集中维护 CLI / Skill 的安装、更新、调用方式、登录鉴权、站点切换和通用错误处理。流程文件与参考文件只引用本文件，禁止重复维护安装或鉴权命令。

> 约定：所有 npm 类命令必须指定内网 registry `https://bnpm.byted.org`；JWT 为敏感信息，不得回显或落盘。

## 目录

- [运行支撑](#运行支撑)
  - [目录](#目录)
  - [bytedcli 通用调用方式](#bytedcli-通用调用方式)
  - [统一前置依赖](#统一前置依赖)
    - [bytedcli 安装与更新](#bytedcli-安装与更新)
    - [agentbuddy 安装与 JWT 获取](#agentbuddy-安装与-jwt-获取)
  - [测试主链路 CLI](#测试主链路-cli)
  - [诊断与环境辅助 CLI / Skill](#诊断与环境辅助-cli--skill)
  - [站点切换](#站点切换)
  - [JSON 输出与调试](#json-输出与调试)
  - [版本保鲜](#版本保鲜)
  - [通用错误处理](#通用错误处理)
  - [无权限处理](#无权限处理)
  - [登录鉴权失败处理](#登录鉴权失败处理)

## bytedcli 通用调用方式

```bash
# 方式 1：直接用 npx 运行最新版
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest --help
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest <command> [options]

# 方式 2：先全局安装，再直接调用 bytedcli
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npm install -g @bytedance-dev/bytedcli@latest
bytedcli --help
bytedcli <command> [options]
```

- 使用 `npx` 时，把后文示例里的 `bytedcli` 替换成 `NPM_CONFIG_REGISTRY=http://bnpm.byted.org npx -y @bytedance-dev/bytedcli@latest`。
- 已全局安装时，直接按后文示例执行 `bytedcli ...`。

## 统一前置依赖

### bytedcli 安装与更新

```bash
# 安装 / 更新统一研发 CLI（bytedcli）
NPM_CONFIG_REGISTRY=http://bnpm.byted.org npm install -g @bytedance-dev/bytedcli@latest
bytedcli self update            # 仅检查用 bytedcli self update --check
```

### agentbuddy 安装与 JWT 获取

`agentbuddy` 为 `skills` npm 包的替代品，功能完全对齐。自 2026-06-18 起旧包 `skills` 无法获取 JWT，必须使用 `agentbuddy`。

```bash
# 安装 agentbuddy（skills 包替代品）
npm_config_registry="https://bnpm.byted.org" npm install -g agentbuddy
# 获取字节云 JWT（bitscli / bits_env_cli 等鉴权所需），<JWT-VRegion> 见 refs/data-reference.md 的 VRegion 映射表
npm_config_registry="https://bnpm.byted.org" npx -y agentbuddy@latest get-jwt --region <JWT-VRegion>
```

- 必须保留 `npm_config_registry=https://bnpm.byted.org/` 前缀，否则 npx 会从公网 registry 拉取，无法获取内部包。
- 除非用户明确要求，勿直接回显 JWT Token。

## 测试主链路 CLI

`bitscli api-test` 是接口测试主命令族，所有测试命令模板见 [cli-reference.md](../refs/cli-reference.md)。

```bash
# 更新 bitscli
bitscli update
# 若 command not found，先安装（内网 npm），再 bitscli update
npm install -g @byted/bits-cli --registry https://bnpm.byted.org
# 更新 api-test 子命令到最新版本
bitscli plugin install api-test
```

## 诊断与环境辅助 CLI / Skill

在 `flows/doctor-flow.md` 证据链诊断与 `flows/env-precheck-flow.md` 环境预检中按需使用；若提示 `command not found` 或版本过旧，按下表安装/更新后重试。

| 关联 Skill | 底层 CLI | 安装 | 更新 |
| :--- | :--- | :--- | :--- |
| `bam` | `bitscli` | 见上方「测试主链路 CLI」 | `bitscli update` |
| `bits-env` | `bits_env_cli`（`bitscli env`） | `bash scripts/install_env.sh && export PATH=~/.local/share/bits-env:$PATH` | 重新执行安装脚本覆盖 |
| `codebase-cli`、`bits-code-guard` | `codebase` | `npm install -g @vecode-fe/codebase-cli@latest --registry=http://bnpm.byted.org`（需 Node ≥ 20.12） | 重新执行安装命令获取 `@latest` |
| `argos` | `argos` | `sh -c "$(curl -L https://argos.byted.org/cli/install.sh)" && export PATH=~/.local/bin:$PATH` | `argos update` |
| `bits-env`、`boe-debug`、`ppe-debug` 等官方 Skill | 经 Skill 调用底层能力 | `bytedcli self skill install -s <skill> -a <agent>`（全局加 `-g`，`<agent>` 如 `claude-code`/`trae`/`trae-cn`） | 随 `bytedcli self update` 与平台 Skill 同步更新 |

> DevInfra 官方各 Skill 依赖对应平台权限（BAM、Bits、PPE/BOE、Argos/Slardar 等），缺权限时按 Skill 自身提示补齐。

## 站点切换

通过全局参数 `--site` 或环境变量 `BYTEDCLI_CLOUD_SITE` 切换 ByteCloud 站点：

| 站点值 | 说明 | SSO | 备注 |
|--------|------|-----|------|
| `cn` | 国内生产（默认） | `sso.bytedance.com` | |
| `i18n-bd` | ByteIntl 国际站 | `sso.bytedance.com` | 通常复用 cn 登录态 |
| `i18n-tt` | TikTok 国际站 | `sso.tiktok-intl.com` | 需单独登录 |
| `eu-ttp` | EU TTP 站 | `sso.tiktok-intl.com` | 需单独登录 |
| `boe` | BOE 测试 | `test-sso.bytedance.net` | |

`--site i18n-bd` 是 ByteIntl 国际站的规范站点值（`i18n` 也可用作别名）。

```bash
# 检查并登录 i18n-tt 站点（使用 TikTok SSO）
BYTEDCLI_CLOUD_SITE=i18n-tt bytedcli auth status
BYTEDCLI_CLOUD_SITE=i18n-tt bytedcli auth login
```

## JSON 输出与调试

```bash
bytedcli --json <command> [options]
```

`--json` 是全局参数，必须放在 `<command>` 前面，例如 `--json auth status`，不能写成 `auth status --json`。

```bash
bytedcli --http-debug <command> [options]
bytedcli --http-print HBhbmt <command> [options]
bytedcli --http-trace-file /tmp/bytedcli.http.log --http-body-limit 4096 <command> [options]
```

`--http-print <parts>` 的 flag 含义：`H` request headers，`B` request body，`h` response headers，`b` response body，`m` meta，`t` time。

## 版本保鲜

```bash
bytedcli self update --check
bytedcli self update
```

Agent 在消费 JSON 输出时，应检查 `_upgrade` 字段。如果 `cli_stale` 或 `skills_stale` 为 `true`，说明当前版本已过期，升级将在下次命令执行时自动完成。

## 通用错误处理

| 情况 | 处理方式 |
|------|----------|
| 区域不可用 | 返回明确的错误提示，并建议用户选择可用区域 |
| 超时处理 | 返回超时错误，建议调整 `request_timeout` 参数 |
| 服务未找到 | 返回清晰的错误信息，建议检查 PSM 拼写 |
| IDL 版本不存在 | 检查项目文件（`AGENTS.md` / `CLAUDE.md`）中是否有备选分支信息，若无则提示用户确认正确的 IDL 分支名 |
| 参数格式错误 | 提供具体的格式说明和示例 |

## 无权限处理

请求返回后，若 `has_permission` 为 `false` 且 `permission_link` 不为空，说明当前用户无该接口的测试权限：

- 向用户展示权限审批链接 `permission_link`，提示用户点击申请权限。
- 终止后续测试流程，不再执行结果分析与总结。
- 无权限返回示例见 [cli-reference.md](../refs/cli-reference.md#返回示例) 末尾。

## 登录鉴权失败处理

CLI 鉴权失败时，按各 CLI 提示执行登录，**不要回显或落盘 JWT**：

```bash
bytedcli auth login
bytedcli --json auth status
codebase auth login
agentbuddy login
```

- 部分命令会自动按 `BYTEDCLI_USER_CLOUD_JWT -> AIME_USER_CLOUD_JWT` 或 `BYTEDCLI_USER_CODE_JWT -> AIME_USER_CODE_JWT` 回退；这些环境变量也不可用时再重新登录。
- 目标站点已切换但仍 401：认证隔离按 SSO 环境生效，`i18n-tt` 与 ByteDance SSO 站点不共享登录态，需为目标站点单独登录，例如 `BYTEDCLI_CLOUD_SITE=i18n-tt bytedcli auth login`。
- 禁止复用从历史流量推荐参数里拿到的 JWT 来做环境查询认证。
