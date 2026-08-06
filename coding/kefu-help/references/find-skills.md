# 技能搜索与安装指南

## 概述

kefu_help 通过 `find-skill` 机制实现能力扩展，可以搜索、安装和管理外部技能。

## ByteDance Skill Hub

### 默认技能源

kefu_help 默认使用 ByteDance 内部 Skill Hub：

```
https://ai-skills.bytedance.net/liumengjie-ai/skills/bytedance-find-skills
```

### 技能分类

| 分类 | 说明 | 示例 |
|------|------|------|
| 语言特定 | 针对特定编程语言的技能 | python-expert, go-guru |
| 框架特定 | 针对特定框架的技能 | react-master, spring-pro |
| 领域特定 | 针对特定领域的技能 | devops-toolkit, security-scanner |
| 工具特定 | 针对特定工具的技能 | docker-helper, k8s-manager |

### 技能元数据

每个技能包含以下元数据：

```json
{
  "id": "bytedance-find-skills",
  "name": "ByteDance Find Skills",
  "description": "ByteDance 内部技能搜索工具",
  "version": "latest",
  "author": "liumengjie-ai",
  "tags": ["bytedance", "skills", "search"],
  "source": "https://ai-skills.bytedance.net/liumengjie-ai/skills/bytedance-find-skills"
}
```

## CLI 命令

### 搜索技能

```bash
# 基础搜索
kefu find-skills search <query>

# 示例
kefu find-skills search python
kefu find-skills search react hooks
kefu find-skills search security
```

### 安装技能

```bash
# 安装最新版本
kefu find-skills install <skill-id>

# 安装指定版本
kefu find-skills install <skill-id> --version 1.2.0

# 示例
kefu find-skills install bytedance-find-skills
```

### 同步默认技能

```bash
# 从 ByteDance Skill Hub 同步默认技能
kefu find-skills sync
```

### 列出已安装技能

```bash
kefu find-skills list
```

### 更新技能

```bash
# 更新所有技能
kefu find-skills update

# 更新指定技能
kefu find-skills update <skill-id>
```

## 自动更新机制

### 异步更新

每次执行 kefu_help 命令时，系统会自动在后台异步更新 `bytedance-find-skills` 技能：

- 更新频率限制：5 分钟内不重复更新
- 更新方式：后台异步执行，不阻塞主命令
- 排除命令：`find-skills` 和 `sync` 命令不触发自动更新

### 手动更新

```bash
# 手动触发更新
kefu find-skills sync
```

## 技能源配置

### 环境变量

```bash
# 设置自定义 Skill Hub URL
export KEFU_SKILL_HUB="https://ai-skills.bytedance.net/liumengjie-ai/skills/bytedance-find-skills"
```

### 配置文件

```bash
# 查看当前配置
kefu config get skill_hub_url

# 设置 Skill Hub URL
kefu config set skill_hub_url https://ai-skills.bytedance.net/liumengjie-ai/skills/bytedance-find-skills
```

## 技能版本锁定

### skills-lock.json

位置: `~/.kefu/skills-lock.json`

```json
[
  {
    "id": "bytedance-find-skills",
    "name": "ByteDance Find Skills",
    "version": "latest",
    "author": "liumengjie-ai",
    "source": "https://ai-skills.bytedance.net/liumengjie-ai/skills/bytedance-find-skills",
    "installed_at": "2024-01-15T10:30:00Z"
  }
]
```

## 技能安装方式

### npx 安装

技能通过 npx 命令安装：

```bash
npx @tiktok-fe/skills add liumengjie-ai/skills --skill bytedance-find-skills --source local
```

### 安装目录

技能安装到以下目录：

```
~/.kefu/skills/
└── bytedance-find-skills/
    ├── SKILL.md
    └── ...
```

## 技能开发

### 技能结构

```
my-skill/
├── SKILL.md          # 技能定义文件
├── references/       # 参考文档
├── statics/          # 流程定义
└── metadata.json     # 元数据
```

### SKILL.md 模板

```markdown
# My Skill

## 身份定义
描述技能的身份和定位

## 核心能力
列出技能的核心能力

## 使用方式
说明如何使用该技能
```

### 发布技能

1. 准备技能文件
2. 编写 metadata.json
3. 提交到 ByteDance Skill Hub

## 最佳实践

### 1. 按需安装

只安装需要的技能，避免过度膨胀。

### 2. 版本控制

使用 `skills-lock.json` 锁定版本，确保团队一致性。

### 3. 定期更新

定期更新技能以获取最新功能和安全修复。

### 4. 本地缓存

对于常用技能，可以缓存到本地加速安装。

## 故障排除

### 安装失败

```bash
# 检查网络连接
kefu config get skill_hub_url

# 清除缓存重试
rm -rf ~/.kefu/cache/
kefu find-skills sync
```

### 版本冲突

```bash
# 查看已安装版本
kefu find-skills list

# 强制重新安装
rm -rf ~/.kefu/skills/bytedance-find-skills
kefu find-skills sync
```

### npx 命令失败

```bash
# 检查 npm/npx 是否可用
which npx

# 安装 npm（如果未安装）
brew install node
```

### 自动更新不工作

```bash
# 检查技能是否已安装
kefu find-skills list

# 手动触发同步
kefu find-skills sync
```
