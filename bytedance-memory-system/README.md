# ByteDance Memory System

> 🧠 ByteDance Workspace + Memory + OpenViking 三层记忆架构管理工具

## 🚀 快速开始

```bash
# 安装
pip install openviking

# 配置
# 编辑 ~/.openviking/ov.conf 添加方舟 API 信息

# 启动
openviking-server

# 初始化 Workspace
cd ~/.openclaw/SKILLS/bytedance-memory-system
npm run init
```

## 📋 系统要求

- Python >= 3.10 (推荐 3.11)
- Node.js >= 18
- Git
- OpenViking >= 0.2.0

## 🎯 核心功能

- **三层记忆架构**: Workspace规范 + 手动核心记忆 + OpenViking自动记忆
- **8类记忆分类**: profile/preferences/entities/events/cases/patterns/tools/skills
- **自动记忆提取**: 基于 VLM 的对话内容分析
- **智能记忆召回**: 向量检索 + 多因子排序

## 📚 文档

- [QUICKSTART.md](QUICKSTART.md) - 5分钟快速上手指南
- [SKILL.md](SKILL.md) - 完整技术文档

## 🔧 可用命令

```bash
# Skill 脚本
npm run init      # 初始化 Workspace
npm run health    # 健康检查
npm run maintain  # 每周维护

# OpenViking CLI
ov ls viking://user/$(whoami)/           # 列出记忆
ov tree viking://user/$(whoami)/memories/ # 树形展示
ov read viking://user/$(whoami)/memories/profile.md  # 读取记忆
```

## ⚠️ 注意事项

- 需要配置方舟 API Key 和 Endpoint ID
- OpenViking 服务需要在本地运行 (默认端口 1933)
- 首次使用需要运行 `npm run init` 初始化 Workspace

## 📄 许可证

Apache-2.0

---

**版本**: v1.1.2  
**作者**: suying.1111  
**主页**: https://github.com/volcengine/OpenViking
