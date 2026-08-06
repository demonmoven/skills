# AI Skills 安装指南

## 概述

当系统显示 "ai-skills not found" 时，需要安装 ai-skills CLI 工具以获取完整的技能管理能力。

## 安装条件

### 检测 ai-skills 是否已安装

```bash
which ai-skills
```

如果返回路径，则已安装；如果无输出或提示 "not found"，则需要安装。

### 检测命令是否可用

```bash
ai-skills --version
```

如果显示版本号，则安装正常；如果提示 "command not found"，则需要安装。

## 安装步骤

### 一键安装

使用以下命令安装 ai-skills：

```bash
curl -fsSL https://ai-skills.bytedance.net/scripts/cli.sh | bash
```

### 安装过程说明

1. **下载安装脚本**：从官方服务器获取安装脚本
2. **检测系统环境**：自动识别操作系统和架构
3. **下载二进制文件**：获取对应平台的 ai-skills 可执行文件
4. **配置环境变量**：将 ai-skills 添加到 PATH

### 安装后验证

```bash
# 检查安装路径
which ai-skills

# 检查版本
ai-skills --version

# 检查帮助信息
ai-skills --help
```

## 常见问题

### 1. 权限不足

**错误信息**：`Permission denied`

**解决方案**：

可直接在终端中执行（不要在沙箱中执行）：
```bash
# 使用 sudo 安装
curl -fsSL https://ai-skills.bytedance.net/scripts/cli.sh | sudo bash

# 或手动设置权限
chmod +x /usr/local/bin/ai-skills
```

### 2. 安装后命令不可用

**错误信息**：`command not found: ai-skills`

**解决方案**：

```bash
# 检查安装路径
ls -la /usr/local/bin/ai-skills

# 手动添加到 PATH
export PATH="/usr/local/bin:$PATH"

# 永久添加（添加到 shell 配置文件）
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### 3. 版本过旧

**解决方案**：

```bash
# 重新安装最新版本
curl -fsSL https://ai-skills.bytedance.net/scripts/cli.sh | bash

```

## 卸载

如需卸载 ai-skills：

```bash
# 删除二进制文件
rm -f /usr/local/bin/ai-skills

# 删除配置目录
rm -rf ~/.ai-skills
```

## 环境要求

| 项目 | 要求 |
|------|------|
| 操作系统 | macOS / Linux |
| Shell | bash / zsh |
| 网络访问 | 需要访问 ai-skills.bytedance.net |

## 相关链接

- 官方文档：https://ai-skills.bytedance.net/docs
- 技能市场：https://ai-skills.bytedance.net/skills
- 问题反馈：https://ai-skills.bytedance.net/issues
