---
name: byted-bnpm-setup
description: 配置本机字节内场 bnpm/npm 发布环境的固定流程技能。Use when user asks to 配置 bnpm、配置字节内网 npm 源、setup byted bnpm、登录字节内场 npm、检查 ~/.npmrc、切换 registry 到 https://bnpm.byted.org，或排查内网 npm 发包前的本机环境。
---

# byted-bnpm-setup

按固定流程在本机配置字节内场 bnpm/npm 环境，目标是让机器具备访问和发布字节内场 npm 包的基础能力。

## 目标信息

- 内场 registry：`https://bnpm.byted.org`
- 文档口径：内部仅支持 **SSO 登录**，不支持账号密码登录
- 推荐登录命令：`npx @bytedance-dev/bnpm@latest login --auth-type=sso`
- 推荐 npm 版本：`npm@8`（如果当前版本不是 8，也可以先尝试登录，异常时再回退到 npm@8）

## 执行步骤

按顺序执行，不要跳步。

### 1. 检查当前环境

先执行：

```bash
node -v
npm -v
npm config get registry
```

再检查用户级配置文件：

```bash
cat ~/.npmrc
```

如果用户给了具体仓库路径，再额外检查项目内是否有 `.npmrc`：

```bash
find <repo_root> -maxdepth 2 -name .npmrc
```

要求：
- 明确区分用户级 `~/.npmrc` 和项目级 `.npmrc`
- 如果项目内已有 `.npmrc`，后续结论里必须提醒“项目配置可能覆盖全局配置”

### 2. 配置用户级 registry

如果 `~/.npmrc` 里还没配置内场源，则追加或设置：

```bash
npm config set registry https://bnpm.byted.org
```

要求：
- 不要粗暴覆盖整个 `~/.npmrc`
- 保留用户已有配置，例如 `prefix=...`
- 修改后再次验证：

```bash
npm config get registry
npm ping --registry=https://bnpm.byted.org
```

预期：
- `npm config get registry` 返回 `https://bnpm.byted.org/`
- `npm ping` 返回 `PONG`

### 3. 检查 npm 版本兼容性

根据文档，优先判断当前 npm 是否为 8：

- 如果是 `npm@8`：直接继续登录
- 如果不是 `npm@8`：也可以先尝试 `npx @bytedance-dev/bnpm@latest login --auth-type=sso`
- 如果登录流程异常，再明确建议切回 `npm@8`

不要默认替用户降级 npm，除非用户明确授权。

### 4. 执行 SSO 登录

在 registry 已切到内场源后执行：

```bash
npx @bytedance-dev/bnpm@latest login --auth-type=sso
```

要求：
- 明确说明：这是 **SSO 登录**，通常需要人工浏览器确认
- 如果命令进入等待态、打开浏览器、或要求人工确认，不要无限卡住
- 需要用户介入时，明确告诉用户“请完成浏览器 SSO 登录”
- 不要输出或泄露 `~/.npmrc` 中可能出现的 token 内容

### 5. 验证登录是否成功

登录后执行：

```bash
npm whoami --registry=https://bnpm.byted.org
```

如果成功，记录当前用户名。

如果失败：
- 明确是“未登录成功”还是“registry 错误”还是“网络/权限问题”

### 6. 面向发版前的补充检查（可选）

如果用户是为了某个具体仓库发版，再额外检查：

```bash
cd <repo_root>
ls package.json package-lock.json
npm ls --depth=0
```

重点判断：
- 是否已有依赖安装
- 是否缺少构建依赖（如 `esbuild`、`typescript`）
- 是否存在 `prepublishOnly` / `publishConfig` / `.npmrc`

注意：
- 这里只做环境和依赖判断，不默认真实发布
- 若用户只要求“配置 bnpm”，可不做这一步

## 输出要求

最终至少返回：

- 当前 Node 版本
- 当前 npm 版本
- 用户级 registry 是否已切到 `https://bnpm.byted.org`
- `npm ping` 是否通过
- 登录是否完成
- 若未完成，具体卡在哪一步
- 若指定了仓库，是否还缺依赖/构建环境

## 常见结论模板

### 成功场景

- Node：`vXX`
- npm：`X.Y.Z`
- 用户级 registry：已切换到 `https://bnpm.byted.org`
- `npm ping`：通过
- `npm whoami`：成功，当前账号为 `xxx`
- 结论：本机已具备基础 bnpm 使用能力

### 需要用户配合场景

- registry 已配置
- `npm ping` 通过
- 但 `npx @bytedance-dev/bnpm@latest login --auth-type=sso` 需要浏览器确认
- 结论：请用户先完成一次 SSO 登录，再继续后续发版操作

## 安全与边界

- 不要泄露 `~/.npmrc` 中的 token
- 不要在未确认的情况下覆盖整个 `~/.npmrc`
- 不要默认真实执行 `npm publish`
- 如果文档建议与本机现状冲突，先保守汇报，再请求用户确认

## 衔接下一步

配置完成后，用下面格式在总结末尾提示用户：

- 如果已完成登录：`如果你要，我可以继续顺手检查目标仓库的发版依赖是否齐全，或者直接帮你验证 publish 流程。`
- 如果还没完成登录：`如果你要，我可以继续盯着登录后的下一步，把仓库发版前的依赖和脚本问题一并检查掉。`
