# 🚀 快速开始指南

> 30 秒内上手 ByteDance Memory System

## ✅ 安装前检查

确保系统满足以下要求：

```bash
# Python >= 3.10
python3 --version  # 需要 >= 3.10

# Node.js >= 18
node --version

# Git
git --version
```

## 🎯 一键安装

```bash
# 下载并运行安装脚本
curl -fsSL https://raw.githubusercontent.com/volcengine/OpenViking/main/examples/openclaw-plugin/install.sh | bash
```

安装过程中会提示：
- **Mode**: 选择 `local` (单机) 或 `remote` (团队)
- **API Key**: 输入方舟 API Key
- **Endpoint IDs**: 输入 Embedding 和 VLM 的 Endpoint ID

## 🚀 启动服务

```bash
# 启动 OpenViking 服务
openviking-server

# 或者后台运行
nohup openviking-server > /tmp/openviking.log 2>&1 &
```

## ✅ 验证安装

```bash
# 检查服务状态
curl http://127.0.0.1:1933/health

# 输出: {"status":"ok","version":"0.2.6"}
```

## 🎮 基本使用

### 1. 初始化 Workspace

```bash
cd ~/.openclaw/SKILLS/bytedance-memory-system
npm run init
```

### 2. 健康检查

```bash
npm run health
```

### 3. 每周维护

```bash
npm run maintain
```

### 4. 使用 OpenViking CLI

```bash
# 列出记忆
ov ls viking://user/$(whoami)/

# 树形展示记忆
ov tree viking://user/$(whoami)/memories/

# 读取具体记忆
ov read viking://user/$(whoami)/memories/profile.md

# 查看摘要
ov abstract viking://user/$(whoami)/memories/

# 查看概述
ov overview viking://user/$(whoami)/memories/
```

## 🔧 手动安装（备用）

如果一键安装脚本失败，可以手动安装：

```bash
# 1. 确保 Python >= 3.10
python3 --version

# 2. 安装 OpenViking
pip install openviking

# 3. 配置 API Key
mkdir -p ~/.openviking
cat > ~/.openviking/ov.conf << 'EOF'
{
  "server": {
    "host": "127.0.0.1",
    "port": 1933
  },
  "embedding": {
    "provider": "volcengine",
    "api_key": "YOUR_API_KEY",
    "model": "YOUR_ENDPOINT_ID",
    "api_base": "https://ark.ap-southeast-1.byteintl.net/api/v3",
    "dimension": 1024,
    "input": "multimodal"
  },
  "vlm": {
    "provider": "volcengine",
    "api_key": "YOUR_API_KEY",
    "model": "YOUR_ENDPOINT_ID",
    "api_base": "https://ark.ap-southeast-1.byteintl.net/api/v3",
    "temperature": 0.1,
    "max_retries": 3
  }
}
EOF

# 4. 启动服务
openviking-server
```

## ❓ 常见问题

### Q: 服务启动失败
```bash
# 检查日志
tail -f /tmp/openviking.log

# 检查端口占用
lsof -i :1933
```

### Q: API Key 错误
- 确保方舟 API Key 有效且未过期
- 确保 Endpoint ID 正确（以 `ep-` 开头）
- 确保 `api_base` 与控制台区域匹配

### Q: ov 命令找不到
```bash
# 检查安装位置
which ov

# 或直接使用 Python 模块
python3 -m openviking.cli --help
```

## 📚 相关资源

- [OpenViking GitHub](https://github.com/volcengine/OpenViking)
- [完整 SKILL 文档](./SKILL.md)
- [安装记录](./INSTALL_LOG.md)

---

**安装完成时间**: 2026-03-13 12:05 PM (Asia/Shanghai)
