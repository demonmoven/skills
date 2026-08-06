# Cursor Plugin Support Design

## 背景

当前仓库仅支持 Claude Code 插件格式（`.claude-plugin/`）。需要新增 Cursor 插件支持，使同一套 skills 可以同时分发到两个平台。

## 方案

**并行配置**：在现有 `.claude-plugin/` 旁新增 `.cursor-plugin/` 配置，skills 目录和 symlinks 完全复用，不改动现有文件。

## 设计细节

### 1. 新增文件清单

共新增 5 个配置文件：

```
gdpa_skills/
├── .cursor-plugin/
│   └── marketplace.json              # 新增
├── plugins/
│   ├── backend/.cursor-plugin/
│   │   └── plugin.json               # 新增
│   ├── backend-knowledge/.cursor-plugin/
│   │   └── plugin.json               # 新增
│   ├── backend-tools/.cursor-plugin/
│   │   └── plugin.json               # 新增
│   └── gdp/.cursor-plugin/
│       └── plugin.json               # 新增
```

### 2. marketplace.json

位于 `.cursor-plugin/marketplace.json`，遵循 Cursor 格式约定：

```json
{
  "name": "gdpa-skills",
  "owner": {
    "name": "chenglinfeng",
    "email": "chenglinfeng@bytedance.com"
  },
  "metadata": {
    "description": "GDPA SKILLS",
    "version": "2.0.0",
    "pluginRoot": "plugins"
  },
  "plugins": [
    {
      "name": "backend",
      "source": "./plugins/backend",
      "description": "后端开发全栈技能集。涵盖 GDP 框架、RAL 资源访问、Kitex RPC、Hertz HTTP、Overpass 代码生成、存储/配置/监控 SDK、TikTok 开发规范、GDPA 工具集等。"
    },
    {
      "name": "backend-knowledge",
      "source": "./plugins/backend-knowledge",
      "description": "后端知识文档技能集。涵盖 GDP 框架、RAL、Kitex、Hertz、Overpass、存储/配置/监控 SDK、TikTok 开发规范、集成测试、可观测性等框架文档和规范指南。"
    },
    {
      "name": "backend-tools",
      "source": "./plugins/backend-tools",
      "description": "后端开发工具集。涵盖 GDPA CLI、BAM API 查询与测试、Overpass 代码生成、Repotalk 代码分析、Argos 日志查询、RDS/TCC 查询、SCM 版本管理等工具。"
    },
    {
      "name": "gdp",
      "source": "./plugins/gdp",
      "description": "GDP 服务开发技能集。涵盖 GDP 框架核心知识和 RAL 资源访问层。"
    }
  ]
}
```

与 Claude Code 版本的差异：
- 新增 `metadata.pluginRoot: "plugins"`（Cursor 约定）
- 每个 plugin 去掉 `strict` 和 `skills` 数组（Cursor 靠目录自动发现）

### 3. plugin.json

Cursor 的 plugin.json 是纯元信息，不列 skills 路径。组件靠目录自动发现。

4 个 plugin 的配置：

| plugin | name | displayName | description |
|--------|------|-------------|-------------|
| backend | `backend` | `GDPA Backend` | 后端开发全栈技能集 |
| backend-knowledge | `backend-knowledge` | `GDPA Backend Knowledge` | 后端知识文档技能集 |
| backend-tools | `backend-tools` | `GDPA Backend Tools` | 后端开发工具集 |
| gdp | `gdp` | `GDPA GDP` | GDP 服务开发技能集 |

示例（backend）：

```json
{
  "name": "backend",
  "displayName": "GDPA Backend",
  "version": "1.0.0",
  "description": "后端开发全栈技能集",
  "author": {
    "name": "chenglinfeng",
    "email": "chenglinfeng@bytedance.com"
  },
  "keywords": ["gdpa", "backend", "gdp", "tiktok"]
}
```

### 4. backend-knowledge 嵌套目录

当前 `plugins/backend-knowledge/skills/` 使用嵌套子目录（`framework-knowledge/`、`sdk-knowledge/`、`guideline-knowledge/`）。

策略：**先保持原样**。Cursor 的 skill 发现机制大概率支持递归子目录（每个子目录下有 SKILL.md 即可识别）。若实测不支持，再补一层扁平 symlinks，成本很低。

### 5. Skills 兼容性

现有 SKILL.md 的 frontmatter 格式已兼容 Cursor：
- `name` 和 `description` 两个平台共用
- Claude Code 的 `user-invocable: false` 字段会被 Cursor 忽略（不报错）
- Cursor 默认允许 agent 自动调用 skills（等效于 knowledge skill 的行为）

不需要修改任何 SKILL.md 文件。

### 6. 文档更新

**README.md**：
- 安装说明区域新增 Cursor 安装方式
- 项目结构树补充 `.cursor-plugin/` 目录

**CLAUDE.md**：
- "Adding a new skill" checklist 补充 Cursor 维护步骤：
  - 新增 skill 时无需改动 Cursor 配置（自动发现）
  - 新增 plugin 时需同步更新 `.cursor-plugin/marketplace.json` 和对应 `.cursor-plugin/plugin.json`

## 平台格式对比

| 维度 | Claude Code | Cursor |
|------|------------|--------|
| 配置目录 | `.claude-plugin/` | `.cursor-plugin/` |
| 组件发现 | marketplace.json 显式列出 skills 路径 | 自动发现 `skills/` 目录 |
| plugin.json | 包含 `skills` 数组 | 纯元信息，无 skills 列表 |
| marketplace.json | 有 `strict`、`skills` 字段 | 有 `pluginRoot`，无 `strict`/`skills` |
| SKILL.md | `user-invocable` 字段 | `disable-model-invocation` 字段 |
| 额外组件 | 无 | rules (.mdc), agents, commands, hooks |

## 不做的事

- 不修改现有 `.claude-plugin/` 配置
- 不修改任何 SKILL.md 文件
- 不生成 `.mdc` rules 文件
- 不添加 agents、commands、hooks 等 Cursor 额外组件
- 不写自动同步脚本（YAGNI）
