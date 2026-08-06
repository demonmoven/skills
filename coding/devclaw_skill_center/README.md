# xdev CLI

xdev 是 XDev Plugin 的统一安装与启动工具,支持 5 个 coding agent: **Trae CN (IDE)** / **Trae (IDE)** / **Coco / TRAE CLI** / **Claude Code** / **Codex**。

在项目目录下运行 `xdev`，即可自动同步最新 plugin 并启动对应的 coding agent。

> ⚠️ **命名提醒**：`--coco` 启动的是 **Coco / TRAE CLI**（命令行 agent），与 `--trae` / `--trae-cn` 启动的 **Trae IDE**（GUI 桌面应用）是两个不同的产品。详见下文「[Trae IDE 与 TRAE CLI 的区别](#trae-ide-与-trae-cli-的区别)」小节。

## 安装

### 推荐:从 bnpm 直接安装(普通用户)

```bash
npm install -g @byted/xdex --registry https://bnpm.byted.org
```

装完后:

```bash
which xdev
# 应当输出 $(npm config get prefix)/bin/xdev

xdev --version
# 应当输出当前 @byted/xdex 的版本号
```

之后所有项目目录直接跑 `xdev` 即可。升级:

```bash
npm install -g @byted/xdex@latest --registry https://bnpm.byted.org
```

xdev 启动时会**自动检查并升级**(同步检查 bnpm latest,有新版本就 spawn `npm install -g` 然后 exit + 提示重跑命令)。24h cache 节流,网络失败静默。可通过环境变量 `XDEV_NO_UPDATE=1` 关闭。

### 备选:用 npx 直接运行(无需安装,临时使用)

不想全局装的话,也可以直接用 npx:

```bash
NPM_CONFIG_REGISTRY=https://bnpm.byted.org npx @byted/xdex <args>
```

或显式 `--registry`:

```bash
npx --registry https://bnpm.byted.org @byted/xdex <args>
```

例如:

```bash
NPM_CONFIG_REGISTRY=https://bnpm.byted.org npx @byted/xdex --cdx           # 直接启 codex
NPM_CONFIG_REGISTRY=https://bnpm.byted.org npx @byted/xdex --version       # 看版本
NPM_CONFIG_REGISTRY=https://bnpm.byted.org npx @byted/xdex                 # 交互启动
```

> ⚠️ 注意:
>
> 1. **首次较慢**:npx 会下载完整 npm 包(~13MB tarball,含 marketplace 18 个 skill),首次大约 5-15 秒;之后 npx cache 命中,启动很快。
> 2. **建议关闭自动升级**:npx 模式下 xdev 的"启动自检 + 自动 install -g"机制没有意义(npx 不是全局装的),建议设环境变量关掉:
>     ```bash
>     XDEV_NO_UPDATE=1 NPM_CONFIG_REGISTRY=https://bnpm.byted.org npx @byted/xdex --cdx
>     ```
>     可以加到 shell function / alias 里避免每次输入。
> 3. **适用场景**:CI 流水线、临时机器、不想 globally 装包的环境、想测试某个特定版本(`npx @byted/xdex@0.0.4`)。
> 4. **如果你长期用 xdev**,推荐还是用上面的"从 bnpm 直接安装"方式 — 启动更快,自动升级 work,体验更好。

### 备选:从源码安装(开发者)

```bash
git clone git@code.byted.org:stone/devclaw_skills_center.git
cd devclaw_skills_center
make install
```

`make install` 内部走 `npm install` + `npm run build` + `npm link`，把 `xdev` 命令软链接到**当前 npm 全局 prefix** 的 `bin/` 目录(npm 包名 `@byted/xdex`)：

| 你的 node 来源 | xdev 实际路径 |
|---------------|--------------|
| Homebrew (Apple Silicon) | `/opt/homebrew/bin/xdev` |
| Homebrew (Intel) | `/usr/local/bin/xdev` |
| nvm | `~/.nvm/versions/node/<version>/bin/xdev` |
| asdf | `~/.asdf/installs/nodejs/<version>/bin/xdev` |

> 不需要 `sudo`。如果你之前装过旧 bash 版本的 `xdev`（在 `/usr/local/bin/xdev`），`make uninstall` 会顺手清掉那个残留。

安装完成后验证：

```bash
which xdev
# 应当输出 $(npm config get prefix)/bin/xdev
```

## 前置依赖

- **Claude Code CLI**（如需使用 `--cc`）：`npm install -g @anthropic-ai/claude-code`
- **Codex CLI**（如需使用 `--cdx`）：`npm install -g @openai/codex`
- **Coco / TRAE CLI**（如需使用 `--coco`）：
  ```bash
  sh -c "$(curl -L https://code.byted.org/api/tos-proxy/download/adopt_coco.sh)" \
    && export PATH=~/.local/bin:$PATH
  ```
  > 注意：`coco` 默认装在 `~/.local/bin/`，确保你的 shell PATH 包含此目录，否则 xdev spawn `coco` 时会 ENOENT。
- **Git**(可选):仅在你**从源码安装**(`make install`)或**手动 publish 新版本**时需要。普通用户走 `npm install -g` 安装则**完全不需要 git**

## 使用

在你的**项目目录**下运行：

```bash
# 交互式选择 coding agent（菜单顺序：trae-cn → trae → coco → cc → cdx）
xdev

# 直接指定 Trae CN (IDE)
xdev --trae-cn

# 直接指定 Trae (IDE)
xdev --trae

# 直接指定 Coco / TRAE CLI
xdev --coco

# 直接指定 Claude Code
xdev --cc

# 直接指定 Codex
xdev --cdx
```

### 每次运行时的自动行为

1. **同步检查 bnpm 上是否有 `@byted/xdex` 新版本**(24h cache 节流);如果有,自动 `npm install -g @byted/xdex@<latest>` 然后 exit + 提示重跑命令
2. 进入 agent 选择(命令行参数已指定则跳过)
3. 按所选 agent 把 plugin 注入到对应位置
4. spawn 对应 agent

**marketplace 内容直接从 npm 包内读**(`<npm-prefix>/lib/node_modules/@byted/xdex/marketplace/`),没有 `~/.xdev/marketplace/` 缓存,**完全不依赖 git**。Plugin/skill 内容随 npm 版本一起更新——每次自动升级到 latest 都拿到最新 marketplace 内容。

### 升级 xdev

xdev 启动时会**自动**检查并升级。手动升级:

```bash
npm install -g @byted/xdex@latest --registry https://bnpm.byted.org
```

关闭自动升级:

```bash
export XDEV_NO_UPDATE=1
```

### 开发模式(working on xdev itself)

不需要任何 flag 或环境变量。`make install` 走 `npm link`,把 `xdev` bin 软链接到源码 repo 的 `dist/index.js`。`PLUGIN_SRC_DIR` 通过 `import.meta.url` 推算,自动指向 `<repo>/marketplace/plugins/xdev/`,所以**改 SKILL.md → 重 build → 跑 xdev 立即看到效果**。

## 参数说明

| 参数 | 说明 |
|------|------|
| `--trae-cn` | 使用 Trae IDE 中国版 |
| `--trae` | 使用 Trae IDE 国际版 |
| `--coco` | 使用 Coco / TRAE CLI |
| `--cc` | 使用 Claude Code CLI |
| `--cdx` | 使用 Codex CLI |

未被识别的参数会透传给对应的 coding agent。例如：

```bash
# 透传 --model 参数给 claude
xdev --cc --model sonnet
```

## 工作原理

```
xdev [--trae-cn|--trae|--coco|--cc|--cdx]
 │
 ├─ 0. 自检 + 自动升级(同步,可能 exit)
 │     ├─ 24h cache 节流;XDEV_NO_UPDATE=1 关闭
 │     ├─ curl bnpm dist-tags.latest
 │     └─ 若 latest > 本地: npm install -g @byted/xdex@<latest> + exit + 提示重跑
 │
 ├─ 1. 选择 coding agent(命令行参数指定则跳过交互)
 │
 ├─ 2. 安装 plugin 到当前项目(按 agent 分别处理,marketplace 直接从 npm 包内读)
 │     ├─ CC：
 │     │   - 清空 ~/.claude/plugins/cache/xdev-official/
 │     │   - claude plugin marketplace add ~/.xdev/marketplace
 │     │   - claude plugin install xdev@xdev-official --scope project
 │     │   - 写入项目 .claude/settings.json
 │     │
 │     ├─ CDX（项目级 plugin,散件机制）：
 │     │   - findProjectRoot:git rev-parse --show-toplevel,找不到 fallback cwd
 │     │   - 拒绝写入 $HOME(防止污染用户 home 目录)
 │     │   - 重建 <project-root>/.codex/skills/xdev/(幂等,只动这一个 plugin 子目录)
 │     │   - cp .codex-plugin/plugin.json + skills/<sub>/SKILL.md (嵌套到 skills/ 中间层)
 │     │   - codex 看到 .codex-plugin/plugin.json 后把目录识别为 plugin,自动赋予 'xdev:' namespace
 │     │   - 每个 skill 在 codex 里渲染为 'xdev:<skill-name>'
 │     │   - codex 启动时自动 walk parent dirs 找 .codex/skills/,无需强制 spawn cwd
 │     │
 │     └─ COCO（项目级）：
 │         - findProjectRoot：git rev-parse --show-toplevel，找不到 fallback cwd
 │         - 拒绝写入 $HOME（防止污染用户 home 目录）
 │         - 读取 <project-root>/.coco/coco.yaml（不存在则新建空文档）
 │         - 用 yaml round-trip API merge：保留用户原有的 model / permission_mode / 注释 / 字段顺序
 │         - upsert marketplaces[xdev-official] + plugins[xdev]（按 name 去重）
 │         - 写回 <project-root>/.coco/coco.yaml
 │
 └─ 4. 启动 agent
       ├─ TRAE-CN: trae-cn .
       ├─ TRAE:    trae .
       ├─ COCO:    coco --yolo  (cwd 强制设为项目根，绕过子目录不向上查找的限制)
       ├─ CC:      claude --dangerously-skip-permissions
       └─ CDX:     codex --dangerously-bypass-approvals-and-sandbox
```

- **Claude Code**：通过 marketplace 正规安装到项目级，写入 `.claude/settings.json`，下次直接运行 `claude` 也能加载。**cc CLI 自己处理合并**，不会破坏你 `.claude/settings.json` 中其他 enabledPlugins / permissions / env 字段
- **Codex CLI**：项目级散件机制,cp skills 到 `<project-root>/.codex/skills/xdev/<skill-name>/SKILL.md`。codex 启动时自动扫描 + walk parent dirs,**只重建 `xdev/` 这一个 namespace 子目录**,不会破坏你 `.codex/skills/` 下其他 namespace 或自己写的 skill
- **Coco / TRAE CLI**：项目级安装，写入 `<project-root>/.coco/coco.yaml`。**用 yaml round-trip 合并**，保留你已有的 yaml 注释、字段顺序、自定义 marketplaces / plugins 项

三种方式都**不影响**你已安装的其他 plugin，也**不会破坏**你已有的 cc/cdx/coco 项目配置。

> **注意**：Codex 会在你的项目里创建 `.agents/` 目录，建议加到 `.gitignore`。

## 目录结构

```
~/.xdev/
└── marketplace/                    # 直接打在 npm 包里(`<npm-prefix>/lib/node_modules/@byted/xdex/marketplace/`)
    ├── .claude-plugin/
    │   └── marketplace.json        # marketplace 清单（CC / Coco 共用）
    └── plugins/
        └── xdev/                   # xdev plugin 本体
            ├── .claude-plugin/
            │   └── plugin.json     # CC plugin manifest（Coco 也兼容此格式）
            ├── .codex-plugin/
            │   └── plugin.json     # CDX plugin manifest
            ├── .coco-plugin/
            │   └── plugin.json     # Coco / TRAE CLI plugin manifest
            └── skills/
                ├── speckit/        # SDD 后端引擎 / Spec Kit
                ├── sdd-fe/         # SDD 前端引擎
                ├── exec-plan/      # 执行计划引擎
                ├── openspec/       # OpenSpec 工作流引擎（fission-ai/openspec 包装层）
                ├── gstack/         # gstack 角色编排引擎（garrytan/gstack 33 个 action 全量）
                ├── fixloop/        # 修复循环
                ├── vv/             # Code review
                └── ...

项目目录/
├── .claude/
│   └── settings.json                # CC 项目级 plugin 配置(写入 enabledPlugins,cc CLI 自己合并)
├── .codex/                          # CDX 项目级 plugin
│   └── skills/
│       └── xdev/                    # plugin 根目录,codex 用此 plugin name 作 namespace
│           ├── .codex-plugin/
│           │   └── plugin.json      # plugin manifest(必须存在,触发 namespace)
│           └── skills/               # 注意是 skills/ 子层,不是直接子目录
│               ├── exec-plan/SKILL.md   # → 'xdev:exec-plan'
│               ├── speckit/SKILL.md     # → 'xdev:speckit'
│               ├── fixloop/SKILL.md     # → 'xdev:fixloop'
│               └── ...
└── .coco/                           # Coco / TRAE CLI 项目级配置
    └── coco.yaml                    # marketplaces[xdev-official] + plugins[xdev](yaml merge)
```

## Trae IDE 与 TRAE CLI 的区别

xdev 接入了 5 个 coding agent，其中 3 个名字都"姓 Trae"，但**指向的是两个完全不同的产品**：

| flag | 启动二进制 | 产品类型 | 是什么 |
|------|-----------|----------|--------|
| `--trae-cn` | `trae-cn` | **Trae IDE**（GUI 桌面应用） | Trae 编辑器中国版 |
| `--trae` | `trae` | **Trae IDE**（GUI 桌面应用） | Trae 编辑器国际版 |
| `--coco` | `coco`（别名 `traecli` / `trae-agent` / `ta`） | **Coco / TRAE CLI**（命令行 agent） | Codebase 团队的 CLI agent，已品牌升级为 TRAE CLI |

**记忆口诀**：
- 想用 **IDE**（图形界面）→ `--trae` 或 `--trae-cn`
- 想用 **CLI**（命令行 agent，对标 claude code / codex / gemini cli）→ `--coco`

**为什么 "Coco" 又叫 "TRAE CLI"**：根据 [飞书 wiki - TRAE CLI 使用文档](https://bytedance.larkoffice.com/wiki/Er28whaTUiRgMekHLvDcISWmnXc)，原 "Coco CLI" 在 2026 年 3 月正式品牌升级为 "TRAE CLI"，归入 TRAE 品牌体系。**功能不变**，原有 `coco` 命令持续支持，新增 `traecli` 别名。xdev 沿用 `--coco` flag 名以避免与 `--trae`（IDE）混淆。

**安装位置也完全不同**：
- Trae IDE 的项目级 skills：`<cwd>/.trae/skills/`（xdev 直接 cp 文件，加 `xdev-` 前缀）
- Coco / TRAE CLI 的项目级 plugin：`<project-root>/.coco/`（xdev 写 yaml）—— 注意是 `.coco/` 不是 `.trae/`，与 Trae IDE 的 `.trae/` 完全隔离（oncall 确认 coco 和 trae 配置目前还没合并）

## Codex 项目级 plugin 注意事项

- xdev 把 plugin 完整结构写到 `<project-root>/.codex/skills/xdev/`,包含 `.codex-plugin/plugin.json`(plugin manifest) + `skills/<sub>/SKILL.md`(嵌套到 skills/ 中间层)。codex 看到 plugin manifest 后把目录识别为 plugin,赋予 **`xdev:` namespace 前缀**,所以你在 codex 里看到的 skill 名是 `xdev:exec-plan`、`xdev:speckit`、`xdev:fixloop` 等。
- **没有 `.codex-plugin/plugin.json` 就没有 namespace**!如果只放 SKILL.md 散件,所有 skill 会平铺为裸名(`exec-plan`、`speckit` 等),会与用户其他 skill 名字冲突。
- **codex 自动 walk parent directories**——你在子目录跑 `xdev --cdx` 也能加载,xdev 不强制切换 spawn cwd(与 launchCoco 的"严格 cwd"形成对比)。
- xdev 完全重建 `<project-root>/.codex/skills/xdev/` 这一个 plugin 子目录(rmSync + cp),**不会动**你 `.codex/skills/` 下的其他 plugin 或自己写的 skill。
- 注意:**codex 0.118.0 实测发现 `<repo>/.agents/plugins/marketplace.json` 这条 OpenAI 官方文档推荐的 repo-scoped marketplace 路径根本不被 codex 加载**。xdev 走的是 `.codex/skills/<plugin>/` 完整 plugin 嵌入路径(实测唯一可行的项目级 plugin namespace 方案)。
- xdev 没用 codex 的 hooks (`<project-root>/.codex/hooks.json`) 或 mcp_servers (`<project-root>/.codex/config.toml`) 等其他散件资产。如果未来 xdev plugin 需要,扩展点已经在 `installToCodex` 函数注释里预留。
- **Trust 约束**:hooks 和 mcp_servers 受 codex "trusted project" 安全模型限制——首次进入项目时 codex 会询问 trust。**skills 不受这个约束**(skills 走自动扫描 + frontmatter,不需要 trust)。
- `xdev clean` **不会**清理项目级 `<project-root>/.codex/skills/xdev/`(项目源文件,不是缓存),但**会清理**旧版的用户级残留(`~/.codex/plugins/cache/xdev-official/` + `~/.codex/config.toml` 中的 `[plugins."xdev@xdev-official"]` 段)。要彻底卸载某项目里的 xdev,手动 `rm -rf <repo>/.codex/skills/xdev/` 即可。
- **怎么在 codex 里调用 xdev skill**:**不要**用 `/xdev:exec-plan ...`(codex 没有 cc/coco 风格的 slash command 协议)。用以下两种之一:
  - `@xdev:exec-plan <参数>`(显式调用)
  - 或者用自然语言:"按 xdev 的 exec-plan 流程帮我..."(codex 根据 SKILL.md frontmatter description 自动触发)
- 第一次跑完后**重启 codex** 一次,确保 codex 重新扫描 skill 目录。后续 codex 会自动检测变更。

## Coco 项目级 plugin 注意事项

- xdev 在 `<project-root>/.coco/coco.yaml` 写 marketplaces + plugins 段。**项目根**通过 `git rev-parse --show-toplevel` 解析；不在 git 仓库内时退回到当前目录。在 `$HOME` 跑 `xdev --coco` 会被拒绝，避免污染用户 home。
- xdev spawn coco 时 **强制 cwd = 项目根**（即使你在子目录跑 `xdev --coco`）。原因：coco 0.120.16 的项目级配置只在 cwd 严格匹配 `.coco/coco.yaml` 所在目录时加载，**不会向上查找父目录**。
- `.coco/coco.yaml` 是项目源文件，建议根据团队习惯加入 `.gitignore` 或共享到 git。
- xdev 用 yaml round-trip API 合并：你已有的 `model` / `permission_mode` / 自定义 marketplaces 和 plugins / 注释 / 字段顺序 / 缩进 **全部保留**。
- 如果 `.coco/coco.yaml` 解析失败（用户写法不合法），xdev 会**报错而不是覆盖**——你的文件是神圣的。
- `xdev clean` **不会**清理项目级 `.coco/coco.yaml`（它是项目源文件，不是缓存）。要彻底卸载某个项目里的 xdev，手动从 `.coco/coco.yaml` 删除 `xdev-official` marketplace 段和 `xdev` plugin 段即可。

## lark-cli MCP Server: 让 Claude Desktop Chat / Cowork 用上飞书

xdev 内置了一个 lark-cli MCP Server,让 **Claude Desktop App 的 Chat 和 Cowork 模式**能调用飞书能力(查日历、发消息、读写文档、电子表格、多维表格、邮件、任务、知识库等)。

> **背景**:Claude Desktop App 的 Chat 模式跑在云端沙箱里、Cowork 模式跑在本机 Linux VM 里,**两者都看不到 macOS host 的 lark-cli 二进制**。Code 模式(=Claude Code)倒是能直接用 lark-cli skills,但 Chat / Cowork 不行。 唯一的桥梁是 **MCP Connector** —— MCP Server 进程跑在 macOS host 上,Chat/Cowork 通过 JSON-RPC 调用它。xdev 把这套流程封装成了一个子命令。

### 一键安装

```bash
xdev lark-mcp mac setup
```

幂等的 5 步流程,已完成的步骤会自动跳过:

| Step | 做什么 | 怎么做 |
|------|--------|--------|
| 1/5 | 检查 lark-cli | 没装就 `npm install -g @larksuite/cli` |
| 2/5 | 飞书应用配置 | 没配就引导你跑 `lark-cli config init --new`(在飞书中打开链接完成应用创建) |
| 3/5 | 用户授权 | 没授就引导你跑 `lark-cli auth login`(在飞书中确认 OAuth)。可加 `--skip-auth` 跳过 |
| 4/5 | 部署 MCP Server | 把 `dist/lark-mcp-server.mjs` 拷到 `~/.xdev/lark-mcp-server/index.mjs` |
| 5/5 | 注册到 Claude Desktop | 把 `mcpServers.lark-cli` 写入 `~/Library/Application Support/Claude/claude_desktop_config.json`(JSON merge,不覆盖已有字段) |

如果 Step 2 / Step 3 中断,按提示在终端跑对应命令完成后,**重新跑 `xdev lark-mcp mac setup`**(已完成的步骤会跳过)。

### 重启 Claude Desktop App(必须)

setup / start / stop 完成后,**必须完全退出并重新打开 Claude Desktop App** 才能让 MCP 配置生效:

- **完全退出**:Cmd+Q,**不是关窗口**(关窗口 App 还在后台跑)
- 重新打开 App
- Desktop 启动时会读 `claude_desktop_config.json` → 自动 spawn MCP Server 子进程

> 为什么要重启:Claude Desktop **不会热加载** `claude_desktop_config.json`。MCP Server 进程的生命周期由 Desktop 全程托管 —— 启动时 spawn,退出时 kill,你**不需要也不应该手动启停 MCP Server 进程**。

### 验证

打开 Chat 或 Cowork 模式,试一句:

```
帮我看一下今天的飞书日程
```

Claude 应该会调用 `lark_cli` tool,触发 host 上的 `lark-cli calendar +agenda`,返回真实日程。

### 看状态

```bash
xdev lark-mcp mac status
```

输出 6 个组件的状态:

```
  lark-cli MCP Server Status
  ──────────────────────────────────

  lark-cli        ✅ v1.0.6 (/opt/homebrew/bin/lark-cli)
  应用配置        ✅ cli_xxxxxxxxxxxxxxxx
  用户授权        ✅ 你的名字
  MCP Server      ✅ /Users/xxx/.xdev/lark-mcp-server/index.mjs
  Desktop 配置    ✅ mcpServers.lark-cli 已注册
  Desktop 进程    ✅ 运行中 (pid xxxx)

  ✅ 全部就绪。如有问题请重启 Claude Desktop App。
```

任何一项 ❌ / ⚠️,跑 `xdev lark-mcp mac setup` 一键修复。

### 命令一览

```bash
xdev lark-mcp mac setup [--skip-install] [--skip-auth]   # 一键安装(幂等)
xdev lark-mcp mac start                                  # 注册到 Claude Desktop 配置(setup 后已自动 start)
xdev lark-mcp mac stop                                   # 从 Claude Desktop 配置中移除
xdev lark-mcp mac status                                 # 查看所有组件状态

xdev lark-mcp mac --help          # 看完整流程说明
xdev lark-mcp mac setup --help    # 看 5 步详解
xdev lark-mcp mac stop --help     # 看 stop 的实际语义
```

> **`start` / `stop` 不是真的启停进程**!MCP Server 进程的生命周期完全由 Claude Desktop 托管。`start` 实际是把 `mcpServers.lark-cli` 写入 config,`stop` 是从 config 中删掉。**两者执行后都必须重启 Desktop App 才生效**。

### 工具暴露(给 Claude 用)

MCP Server 当前暴露 1 个通用 tool `lark_cli`:

| 字段 | 类型 | 说明 |
|------|------|------|
| `command` | string(必填) | lark-cli 子命令及参数(不含 `lark-cli` 前缀),例如 `"calendar +agenda"`、`"docs +fetch --doc <token>"` |
| `as_identity` | enum(可选) | 身份类型:`user` / `bot` / `auto`(默认) |

Claude 根据 tool description 中的 8 个常用示例自己拼命令。需要更多语法,Claude 可以调 `schema <service.resource.method>` 查 lark-cli API 详情,或调 `help` 查所有命令。

### 故障排查

| 现象 | 原因 | 修复 |
|------|------|------|
| Chat/Cowork 看不到 `lark_cli` tool | Desktop 没重启 | Cmd+Q 后重新打开 |
| `xdev lark-mcp mac status` 全 ✅ 但 Desktop 中仍调不到 | Desktop 没重启 / 启动时 MCP server spawn 失败 | 1. 重启 Desktop 2. 看 Desktop 的 MCP 日志(Settings → Developer) |
| `lark-cli` 报权限不足 | scope 没授 | 按错误提示运行 `lark-cli auth login --scope "<missing_scope>"` |
| Step 2 卡住等链接 | `lark-cli config init --new` 在等用户操作 | 复制终端里的链接到飞书打开,完成后回到终端 |
| `~/.xdev/lark-mcp-server/` 突然不见 | 跑了 `xdev clean` | clean 已经被白名单保护,不会删 lark-mcp-server。如果真不见了,重新跑 `xdev lark-mcp mac setup` |
| 配置文件其他字段被覆盖 | JSON merge bug | 报 issue。当前实现用 `JSON.parse + 字段 merge + JSON.stringify`,只动 `mcpServers.lark-cli` 这一个字段 |

### 与 xdev 其他命令的关系

| xdev 命令 | 对 lark-mcp 的影响 |
|-----------|-------------------|
| `xdev clean` | **保留** `~/.xdev/lark-mcp-server/`(白名单),只清其他缓存 |
| `xdev uninstall` | **删除** `~/.xdev/lark-mcp-server/` + 同步从 `claude_desktop_config.json` 移除 `mcpServers.lark-cli` + 卸载 npm 全局包 |
| `xdev --cc` / `--coco` 等启动命令 | 完全不动 lark-mcp 相关文件 |
| xdev 自动升级(版本检查) | 升级会更新 `dist/lark-mcp-server.mjs`(npm 包内的 bundle),但**不会自动重新部署**到 `~/.xdev/lark-mcp-server/`。要更新部署,跑一次 `xdev lark-mcp mac setup` |

> **升级 xdev 后**:旧的 MCP Server 仍能跑(代码自包含),但如果想用新版本 MCP Server,跑一次 `xdev lark-mcp mac setup` 重新部署即可。

## 卸载

### 推荐:`xdev uninstall` 子命令

不管是 `npm install -g` 装的还是 `npx` 跑的,都用同一条命令彻底卸载:

```bash
xdev uninstall
```

或者(npx 模式 / 没有全局装 xdev 的情况):

```bash
NPM_CONFIG_REGISTRY=https://bnpm.byted.org npx @byted/xdex uninstall
```

`xdev uninstall` 分两步:

1. **清缓存** —— 等价于 `xdev clean`,删除 `~/.xdev/`、`~/.claude/plugins/cache/xdev-official/`、`~/.codex/plugins/cache/xdev-official/`、以及 `~/.codex/config.toml` 里残留的 `[mcp_servers.xdev-official]` / `[mcp_servers.xdev]` 段(pre-project-scope 时代的遗物)
2. **卸 npm 全局包** —— 跑 `npm uninstall -g @byted/xdex`。如果 xdev 是 npx 模式装的(没有全局),npm 会报 not installed,这是预期的,不算失败

> 自删除安全:`npm uninstall -g` 会把当前正在运行的 `dist/index.js` 文件从磁盘上删掉,但 Node 已经把它 mmap 到内存里,进程还能继续把"卸载完成"的提示打完才退出 —— 跟 `brew uninstall` / `gh extension uninstall` 是同一个套路。

### `xdev uninstall` **不会**碰的项目级文件

下面这些是**项目源文件**(可能在你的 git 里),不属于"xdev 自己的状态",`xdev uninstall` 一律不动。需要的话你自己从每个项目里删:

- `<项目根>/.codex/skills/xdev/` —— codex 项目级 plugin 安装
- `<项目根>/.coco/coco.yaml` 里的 xdev marketplace / plugin 段 —— coco 项目级 plugin
- `<项目根>/.trae/skills/xdev-*` —— trae IDE skill 安装
- `<项目根>/.claude/settings.json` 里的 `enabledPlugins` 中 `xdev@xdev-official` 那一项 —— cc plugin 启用记录

`~/.trace/`(SSO token + device id)同样不会清。

### 备选:开发者源码安装的卸载

如果你是用 `git clone + make install` 装的(开发者模式):

```bash
cd /path/to/devclaw_skills_center && make uninstall
```

`make uninstall` 做三件事:

1. `npm unlink -g @byted/xdex` —— 解掉 `npm link` 的软链(顺手 unlink 旧的 `@byted/devclaw-skills-center` 残留)
2. `sudo rm -f /usr/local/bin/xdev` —— 清理**旧 bash 版本** xdev 残留(仅迁移老用户用,sudo 会要密码)
3. `rm -rf ~/.xdev/` —— 清 marketplace 同步缓存

`make uninstall` 跟 `xdev uninstall` 的区别:`make uninstall` 走 `npm unlink`(对应 `npm link` 的开发者安装),`xdev uninstall` 走 `npm uninstall -g`(对应 `npm install -g` 的普通用户安装)。

## 故障排查

### `xdev` 报 `-bash: /usr/local/bin/xdev: No such file or directory`

如果你之前装过旧 bash 版本的 `xdev`（位于 `/usr/local/bin/xdev`），或者在同一个 terminal 里跑了 `make uninstall` + `make install`，bash 会缓存旧路径并继续尝试 exec 它。

> bash 在 PATH 搜索一次找到命令后会把"命令名 → 完整路径"缓存在内部 hash table，下次同名命令**不再做 PATH 查找**。这是 bash 设计，不是 xdev 的 bug。完整路径（而不是 `command not found`）出现在错误信息里就是 hash 缓存的特征。

任选一条修复：

```bash
hash -r           # 清空全部 bash 命令缓存
hash -d xdev      # 只清 xdev 这一项
exec bash         # 用新进程替换当前 shell
```

或者直接开一个新的 terminal 窗口。

### `xdev` 报 `command not found`

说明 npm 全局 prefix 的 `bin/` 不在你的 PATH 里。运行 `npm config get prefix` 看到 prefix 路径后，把 `<prefix>/bin` 加到你的 PATH（在 `~/.zshrc` 或 `~/.bashrc` 里）。

```bash
echo "export PATH=\"$(npm config get prefix)/bin:\$PATH\"" >> ~/.zshrc
source ~/.zshrc
```
