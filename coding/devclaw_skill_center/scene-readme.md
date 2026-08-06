# Scene 条件分发机制

## 概述

Skills 的 `.md` 文件支持按**场景（Scene）**条件分发不同的 prompt 内容。在执行 `install` 命令时，CLI 会根据用户指定的 `--scene` 参数解析所有 `.md` 文件中的条件表达式，仅保留对应场景的 prompt 内容写入目标目录。

---

## CLI 指令用法

```bash
devclaw-skills-center install [options]
```

### `--scene` 参数

| 参数 | 说明 |
|------|------|
| `--scene <scene>` | 指定安装场景，可选值：`main`（默认）、`devClaw` |

- **非必填**，不传时默认为 `main`。
- 不需要交互式选择，直接通过命令行传入。

### 示例

```bash
# 默认 main 场景
devclaw-skills-center install --trae --fe

# 指定 devClaw 场景
devclaw-skills-center install --cc --fe --scene devClaw

# 完整参数组合
devclaw-skills-center install --trae --fullstack --scene devClaw
```

### 安装行为差异

| 场景 | Claude Code (--cc) | Trae (--trae) |
|------|-------------------|---------------|
| `main` | **符号链接**（symlink），不修改源文件 | 文件复制，解析 `.md` 中的条件表达式 |
| `devClaw` | 文件复制，解析 `.md` 中的条件表达式 | 文件复制，解析 `.md` 中的条件表达式 |

> `main` + `--cc` 时使用 symlink 是因为 `main` 场景保留的内容即为源文件默认内容（else 分支或最后一个分支），无需修改文件。

---

## Skills MD 文件条件表达式写法

### 语法格式

```
<if scene="scene_name_1">
scene_name_1 的 prompt 内容
<else if scene="scene_name_2">
scene_name_2 的 prompt 内容
<else>
默认 prompt 内容
<end>
```

### 标签说明

| 标签 | 必填 | 说明 |
|------|------|------|
| `<if scene="...">` | ✅ | 条件块开始，指定第一个场景名称 |
| `<else if scene="...">` | ❌ | 可选，追加更多场景分支（可多个） |
| `<else>` | ❌ | 可选，默认分支（无匹配时的兜底） |
| `<end>` | ✅ | 条件块结束标记 |

### 规则

1. 每个条件块必须以 `<if scene="...">` 开头，以 `<end>` 结尾。
2. `<else if scene="...">` 和 `<else>` 是可选的，可以只有 `<if>...<end>`。
3. 一个 `.md` 文件中可以包含**多个**条件块，也可以完全不包含条件块（原样输出）。
4. 条件块外的普通文本不受影响，原样保留。

### 当前支持的 scene 值

| scene 值 | 说明 |
|----------|------|
| `main` | 主场景（默认值） |
| `devClaw` | DevClaw 场景 |

如需扩展新场景，修改 `src/config.ts` 中的 `Scene` 类型定义即可。

---

## 场景匹配规则

安装时，解析器按以下优先级为每个条件块选择 prompt：

### `main` 场景

1. 精确匹配 `<if scene="main">` 或 `<else if scene="main">` 分支
2. 匹配 `<else>` 分支
3. 取最后一个分支的内容（兜底）

### `devClaw` 场景

1. 精确匹配 `<if scene="devClaw">` 或 `<else if scene="devClaw">` 分支
2. 匹配 `<else>` 分支
3. 无匹配时输出空内容

---

## 完整示例

### MD 源文件内容

```markdown
# 开发流程

按照以下流程进行开发：

<if scene="devClaw">
1. 在 DevClaw 平台上创建工作空间
2. 使用 DevClaw CLI 拉取项目模板
3. 按照 DevClaw 规范编写代码
<else>
1. 在本地创建项目目录
2. 使用标准脚手架初始化项目
3. 按照团队规范编写代码
<end>

完成开发后提交代码。
```

### `--scene main` 安装后的内容

```markdown
# 开发流程

按照以下流程进行开发：

1. 在本地创建项目目录
2. 使用标准脚手架初始化项目
3. 按照团队规范编写代码

完成开发后提交代码。
```

### `--scene devClaw` 安装后的内容

```markdown
# 开发流程

按照以下流程进行开发：

1. 在 DevClaw 平台上创建工作空间
2. 使用 DevClaw CLI 拉取项目模板
3. 按照 DevClaw 规范编写代码

完成开发后提交代码。
```

---

## 多分支示例

```markdown
<if scene="devClaw">
使用 DevClaw 专属工具链进行调试。
<else if scene="main">
使用标准调试工具进行调试。
<else>
请根据项目环境选择合适的调试方式。
<end>
```

| scene | 输出内容 |
|-------|---------|
| `main` | 使用标准调试工具进行调试。 |
| `devClaw` | 使用 DevClaw 专属工具链进行调试。 |

---

## 相关代码

| 文件 | 说明 |
|------|------|
| `src/config.ts` | `Scene` 类型定义（`'main' \| 'devClaw'`） |
| `src/scene-parser.ts` | 条件表达式解析器（`resolveSceneConditionals`） |
| `src/install.ts` | `--scene` 参数注册 & 安装时调用解析器 |
