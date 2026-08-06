# DevClaw Skills Center - AI Coding 指导文档

## 1) 仓库概览

- **仓库根目录**：`devclaw_skills_center/`
- **远端仓库**：`git@code.byted.org:stone/devclaw_skills_center.git`（origin）
- **定位**：DevClaw Skills 中心仓库，核心产出为 Agent Skills（支持 Claude Code / Trae），CLI 为辅助安装分发工具
- **技术栈**：TypeScript + esbuild 构建 + Commander CLI

## 2) 项目目录结构

`feat-dev/` 目录位于仓库根目录下，用于存放日常开发 feature 的 vibe 记录。

```
devclaw_skills_center/
├── .claude/
│   └── CLAUDE.md          # 本规范文件
├── feat-dev/              # 开发 feature vibe 记录（已 gitignore，仅本地）
│   └── 2026/              # 年
│       └── 04/            # 月
│           └── 03/        # 日
│               ├── 143025-xxx/   # 时分秒-极简概述
│               └── 160512-yyy/   # 时分秒-极简概述
├── bin/
│   └── xdev               # xdev CLI 启动器（bash 脚本）
├── marketplace/           # marketplace 根目录
│   ├── .claude-plugin/
│   │   └── marketplace.json    # CC marketplace 清单
│   └── plugins/
│       └── xdev/          # xdev plugin 本体
│           ├── .claude-plugin/
│           │   └── plugin.json # CC plugin manifest（Coco 也兼容此格式）
│           ├── .codex-plugin/
│           │   └── plugin.json # Codex plugin manifest
│           ├── .coco-plugin/
│           │   └── plugin.json # Coco / TRAE CLI plugin manifest
│           └── skills/    # Skills 目录（每个子目录为一个 skill）
│               ├── speckit/       # SDD 后端引擎 / Spec Kit（原 spkx / devclaw-sdd-be）
│               ├── sdd-fe/        # SDD 前端引擎（原 devclaw-sdd-fe）
│               ├── exec-plan/     # 执行计划引擎（原 devclaw-exec-plan）
│               ├── openspec/      # OpenSpec 工作流引擎（fission-ai/openspec 包装层）
│               ├── gstack/        # gstack 角色编排引擎（garrytan/gstack 33 个 action 全量迁移）
│               ├── fixloop/       # 修复循环 skill
│               ├── vv/            # Code review skill
│               └── ...            # 其他 skills
├── src/                   # 旧版 CLI 源码（install/list/uninstall）
│   ├── index.ts           # 入口，注册 install / list / uninstall 子命令
│   ├── config.ts          # 集中配置（白名单、bytedcli 仓库地址、skill 列表）
│   ├── install.ts         # install 命令：角色选择 + 目标选择 + skill 安装
│   ├── list.ts            # list 命令：列出可用 skills
│   └── clean-skills.ts    # uninstall 命令：清理已安装的 skills
├── dist/                  # 构建产物
├── build.js               # esbuild 构建脚本
├── Makefile               # make install 快捷安装
└── package.json
```

## 3) 每日需求目录管理规则（feat-dev）

### 日期目录
- 在 `feat-dev/` 下按 `YYYY/MM/DD/` 三级目录组织（如 `feat-dev/2026/04/03/`）。
- 月和日使用两位数字（补零），如 `04`、`03`。
- 每次接到需求时，若对应的年/月/日目录不存在则自动创建。

### 需求子目录命名
- 格式：`{HHMMSS}-{feature_description 极简概述}/`
- `HHMMSS` 为当前时间的时分秒（24小时制，补零），如 `143025`。
- 极简概述使用英文短横线连接的关键词（kebab-case），简明扼要。

示例：
```
feat-dev/2026/04/03/
├── 143025-cli-init-research/
├── 151200-skill-migration/
└── 160512-command-scaffold/
```

## 4) 需求产出文档规范

每个需求目录下根据需求类型输出对应文档：

| 场景 | 产出文件 | 说明 |
|------|----------|------|
| 纯调研 | `research.md` | 调研结论、技术选型、参考资料等 |
| 基于调研的开发 | `plan.md` | 引用/依赖已有的 `research.md`，列出开发计划 |
| 调研 + 开发计划一体 | `plan.md` | 文档内先写调研内容（Research 部分），紧接列出开发计划（Plan 部分） |

### 需求原文结构化整理（固定步骤）
- 每次接到用户的需求描述后，**必须**将用户原始输入的需求内容进行语言和表达上的结构化整理。
- 整理后的内容放在 `research.md` 或 `plan.md` 文档的**最开头位置**，作为独立章节，标题为 `## 需求描述（整理）`。
- 整理要求：保留用户原始意图，但在语言表达上做清晰化、结构化处理（分点、分层、去冗余、补充隐含信息），使需求描述更易于理解和回顾。

### 文档编写要求
- 使用中文撰写，代码/API/术语保留英文。
- 结构清晰，使用分级标题、列表、表格、代码块。
- 是否进入实际编码开发，由用户指令决定，不主动执行。

## 5) 工作流程（接到需求时）

1. 确认当前日期和时间，检查 `feat-dev/{YYYY}/{MM}/{DD}/` 是否存在，不存在则创建。
2. 以当前时间 `HHMMSS` 作为前缀，创建新需求目录：`{HHMMSS}-{极简概述}/`。
3. 根据需求类型输出 `research.md` 和/或 `plan.md`。
4. 等待用户后续指令决定是否进入开发阶段。

## 6) 核心概念

### 角色（Role）

安装时需选择角色，决定安装哪些 skills：

| 角色 | 值 | 本仓库 Skills | bytedcli Skills |
|------|-----|--------------|-----------------|
| 前端 | `fe` | 仅白名单内的 skills | 不安装 |
| 后端 | `be` | 排除白名单内的 skills | 安装 |
| 全栈 | `fullstack` | 全部安装 | 安装 |

前端白名单定义在 `src/config.ts` 的 `FE_SKILL_WHITELIST` 中，当前包含：
- `sdd-fe`
- `x-ui-workflow`

### 目标（Target）

安装目标决定 skills 的安装位置和方式：

| 目标 | 值 | 安装目录 | 安装方式 |
|------|-----|---------|---------|
| Claude Code | `cc` | `<cwd>/.claude/skills/` | 符号链接（symlink） |
| Trae | `trae` | `<cwd>/.trae/skills/` | 递归文件拷贝 |

### bytedcli Skills

除本仓库的 skills 外，还会通过 `npx -y skills add` 从 bytedcli 仓库安装额外的 skill 域：
- `bytedance-tce`、`bytedance-log`、`bytedance-agw`、`bytedance-auth`、`bytedance-tools`

安装命令格式：
```bash
npx -y skills add git@code.byted.org:byteapi/bytedcli.git --skill <skill-name> -a <agent-name> -y
```

其中 `<agent-name>` 根据目标映射：`cc` → `claude-code`，`trae` → `trae`

## 7) 构建与开发

```bash
npm run build      # esbuild 构建 + npm link
npm run dev         # ts-node 开发模式
npm run lint        # ESLint 检查
npm run clean       # 清理 dist/
```

构建产物为单个 ESM bundle `dist/index.js`，通过 esbuild 打包，外部化 node builtins 和 dependencies。

### 发布到 bnpm

包名: `@byted/xdex`(注意:历史曾经叫 `@byted/devclaw-skills-center`,从 0.0.34 后已彻底改名重置版本号到 0.0.1,**旧包不再维护**)。

#### 单步发布命令

```bash
npm publish --registry https://bnpm.byted.org/
```

`prepublishOnly` 钩子会自动触发:
1. 先跑 `npm test`(vitest)
2. 再执行 `pre-publish.sh`,内部:`npm run build` 构建 → 自动 bump patch 版本号(修改 `package.json`)
3. 然后 npm 打包并发布到 bnpm registry

注意事项:
- 发布前无需手动 build 或改版本号,`prepublishOnly` 会自动处理
- 发布后 `package.json` 中的版本号会被 bump 到下一个 patch(为下次发布预留)
- `package.json` 的 `files` 字段控制发布内容:`dist`、`marketplace`、`agents`、`README.md`
- registry 已在全局 `~/.npmrc` 配置为 `https://bnpm.byted.org/`,但显式 `--registry` 更安全
- 改名后 `bin` 字段只有 `xdev`(`devclaw-skills-center` bin 已删除)
- esbuild build 时通过 `define: { XDEV_VERSION: ... }` 把 `package.json` 的 version 注入到 dist 里,运行时 `xdev --version` 自动跟随 package 版本(无需手改硬编码字符串)

#### 完整发布工作流(6 步 SOP)

> 推荐流程:确保版本来源是 main 上一个干净的 commit,发布与代码可追溯。

```bash
# 1. 切到 main 并强制同步最新(高风险动作,会丢本地未提交改动)
git fetch origin --prune
git checkout main
git reset --hard origin/main

# 2. 检查 npm 身份与 registry
npm config get registry
npm whoami
# 确保是字节内网 bnpm 凭证

# 3. 项目根 npm publish(会自动跑 prepublishOnly → npm test → pre-publish.sh)
npm publish --registry https://bnpm.byted.org/
# 如果发布失败立即停止后续步骤,并汇报失败原因
# 验证发布: npm view @byted/xdex --registry https://bnpm.byted.org/

# 4. 创建 publish 分支,把版本号变更 commit 上
git checkout -b "publish/$(date +%Y%m%d-%H%M%S)"
git status --short  # 应该看到 package.json 的 version 被 pre-publish.sh bump 了
git add package.json package-lock.json
git commit -m "chore: publish xdex"

# 5. 推送 publish 分支到远端
git push -u origin HEAD

# 6. 拼接 MR 链接(用 URL encode 替换分支名里的 /)
echo "https://code.byted.org/stone/devclaw_skills_center/merge_requests/new?target_branch=main&source_branch=$(git rev-parse --abbrev-ref HEAD | sed 's|/|%2F|g')"
# 把 MR 链接发出去合并回 main
```

如果只是验证能否发布(不真发):

```bash
npm publish --dry-run --registry https://bnpm.byted.org/
```

#### 安全约束

- **不要**在 `git reset --hard origin/main` 前丢失本地未提交工作 — 先 `git stash` 或确认无未提交变更
- **不要**在没确认 npm 身份和 registry 时 publish — 可能误发到公网或错误 scope
- 历史发布记录:本仓库 main 上的 `chore: publish xdex` / `chore: publish devclaw-skills-center` commit + `publish/{YYYYMMDD-HHMMSS}` 命名规范的临时分支

## 8) xdev CLI 使用

xdev 是统一的 Plugin 加载入口，支持 Trae CN / Trae（IDE）、Coco / TRAE CLI、Claude Code、Codex CLI 5 个 coding agent：

```bash
# 在项目目录下运行，交互选择 agent
xdev

# 直接指定 agent（菜单顺序：trae-cn → trae → coco → cc → cdx）
xdev --trae-cn                     # Trae CN (IDE)
xdev --trae                        # Trae (IDE)
xdev --coco                        # Coco / TRAE CLI
xdev --cc                          # Claude Code
xdev --cdx                         # Codex CLI

# 开发调试模式（从本地仓库同步 plugin，跳过版本检查）
xdev --coco
```

xdev 会将 plugin 安装到 `~/.xdev/plugin/`，启动时按 agent 加载：
- CC：通过 `claude plugin marketplace add / install --scope project`，cc CLI 自己合并 `.claude/settings.json`
- CDX:项目级 plugin,装 plugin 完整结构到 `<project-root>/.codex/skills/xdev/{.codex-plugin/plugin.json,skills/<name>/SKILL.md}`,codex 看到 .codex-plugin/plugin.json 后把目录识别为 plugin,自动用 `xdev` 作为 namespace 前缀,渲染为 `xdev:<name>`
- COCO：项目级，写入 `<project-root>/.coco/coco.yaml`（用 yaml round-trip 合并，保留用户已有注释和字段）；spawn coco 时强制 cwd = 项目根
- TRAE / TRAE CN：直接 cp skills 到 `<cwd>/.trae/skills/xdev-*`（IDE，不走 plugin 机制）

> ⚠️ **命名提醒**：`--coco` 是 Coco / TRAE **CLI**（命令行 agent），`--trae` / `--trae-cn` 是 Trae **IDE**（GUI），是两个完全不同的产品。详见 `README.md` 的「Trae IDE 与 TRAE CLI 的区别」小节。

### 旧版 CLI（standalone skills 安装）

```bash
# 交互式安装（会依次提示选择角色和目标）
npm install -g @byted/xdex --registry https://bnpm.byted.org

# 命令行参数直接指定
npx ... install --fe --cc          # 前端 + Claude Code
npx ... install --be --trae        # 后端 + Trae
npx ... install --fullstack --cc   # 全栈 + Claude Code
```

## 9) 编码约定

- 使用 TypeScript，严格类型
- 使用 `chalk` 做终端彩色输出
- 使用 `commander` 做命令行解析
- 配置项集中在 `src/config.ts`
- 不使用注释，代码自文档化
- 交互选择使用自实现的 `promptSelect` 泛型函数（基于 `readline` 原生 API）
- 安装逻辑中 Claude Code 使用 symlink，Trae 使用文件拷贝（因 Trae 不支持 symlink）
