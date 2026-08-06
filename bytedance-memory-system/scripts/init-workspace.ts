#!/usr/bin/env node
/**
 * @fileoverview Workspace 三层记忆架构初始化脚本
 * @author suying.1111
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const WORKSPACE_DIR = process.env.WORKSPACE_DIR || process.cwd();

const DIRECTORIES_MD = `# DIRECTORIES.md - 目录使用规范

## 核心规则

| 类别 | 文件 | 清理策略 |
|------|------|----------|
| 🔴 **永不清理** | 核心配置见下方列表 | 禁止删除 |
| 🟡 **按期清理** | logs/, memory/, .learnings/, state/ | 超期删除/归档 |
| 🟢 **随时清理** | __pycache__/, *.tmp, *.bak, *.zip | 自动清理 |

### 🔴 核心配置文件（永不清理）

SOUL.md, USER.md, IDENTITY.md, AGENTS.md, MEMORY.md, EVOLUTION.md, TOOLS.md, HEARTBEAT.md, DIRECTORIES.md

**其他 *.md 文件**：按普通文件处理，清理前需确认

---

## 清理期限速查

| 目录 | 保留期限 | 操作 |
|------|----------|------|
| logs/*.json | 7 天 | 删除 |
| logs/*.log | 3 天 | 删除 |
| memory/YYYY-MM-DD.md | 7 天 | 归档 |
| memory/archive/ | 90 天 | 压缩/删除 |
| .learnings/ 已解决条目 | 30 天 | 移除 |
| *.zip | 7 天 | 删除 |

---

## 健康指标

| 指标 | 健康 | 危险 |
|------|------|------|
| logs/ 文件数 | < 10 | > 30 |
| memory/ 大小 | < 100KB | > 500KB |

---

## 清理权限

**可自动执行**：缓存、临时文件、过期日志

**需用户确认**：删除 .md 文件、skills/、state/、批量 >10 文件`;

const HEARTBEAT_MD = `# HEARTBEAT.md - 定期维护任务

## 任务清单

### 每日执行

**检查轮换** (2-4次/天)

轮流检查以下项目：
- [ ] **提及** - 飞书/社交媒体通知？
- [ ] **项目** - 检查 git 状态、待办事项
- [ ] **Channel 健康** - Feishu/Gateway 是否在线
- [ ] **AI厂商动态** - 监控OpenAI、Anthropic、Google动态

在 memory/heartbeat-state.json 中跟踪检查时间。

## 何时通知用户

**主动通知**：
- 发现有趣的内容
- 超过 6 小时未交流

**保持静默 (HEARTBEAT_OK)**：
- 深夜（23:00-08:00）除非紧急
- 用户明显忙碌
- 自上次检查以来没有新内容
- 距离上次检查 <30 分钟`;

const EVOLUTION_MD = `# EVOLUTION.md - 配置变更记录

## 变更历史

### ${new Date().toISOString().split('T')[0]}
- 初始化 Workspace 三层记忆架构
- 创建 DIRECTORIES.md、HEARTBEAT.md
- 配置 OpenViking 自动记忆系统`;

const LEARNINGS_MD = `# LEARNINGS.md - 经验教训

## 已解决

## 待解决`;

const PROJECTS_MD = `# 项目状态跟踪

## 活跃项目

## 待办项目`;

function ensureDir(dir) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
    console.log(`✅ 创建目录: ${dir}`);
  }
}

function writeFile(filePath, content) {
  if (!fs.existsSync(filePath)) {
    fs.writeFileSync(filePath, content);
    console.log(`✅ 创建文件: ${filePath}`);
  } else {
    console.log(`⏭️  文件已存在: ${filePath}`);
  }
}

function initWorkspace() {
  console.log('🚀 初始化 Workspace 三层记忆架构...\n');

  // 创建目录
  const dirs = [
    'memory',
    '.learnings',
    'state',
    'logs',
    'tmp'
  ];

  dirs.forEach(dir => ensureDir(path.join(WORKSPACE_DIR, dir)));

  // 创建文件
  writeFile(path.join(WORKSPACE_DIR, 'DIRECTORIES.md'), DIRECTORIES_MD);
  writeFile(path.join(WORKSPACE_DIR, 'HEARTBEAT.md'), HEARTBEAT_MD);
  writeFile(path.join(WORKSPACE_DIR, 'EVOLUTION.md'), EVOLUTION_MD);
  writeFile(path.join(WORKSPACE_DIR, '.learnings/LEARNINGS.md'), LEARNINGS_MD);
  writeFile(path.join(WORKSPACE_DIR, 'memory/projects.md'), PROJECTS_MD);

  // 创建今日日志
  const today = new Date().toISOString().split('T')[0];
  const dailyLog = `# ${today} 日志

## 今日操作
- 初始化 Workspace 三层记忆架构

## 备注`;
  writeFile(path.join(WORKSPACE_DIR, `memory/${today}.md`), dailyLog);

  // Git 提交
  try {
    execSync('git add -A', { cwd: WORKSPACE_DIR });
    execSync('git commit -m "feat: 初始化 Workspace 三层记忆架构"', { cwd: WORKSPACE_DIR });
    console.log('\n✅ Git 提交成功');
  } catch (e) {
    console.log('\n⚠️ Git 提交失败（可能已提交或无变更）');
  }

  console.log('\n🎉 初始化完成！');
  console.log('\n目录结构:');
  console.log('  🔴 核心配置: AGENTS.md, SOUL.md, MEMORY.md, ...');
  console.log('  🟡 按期清理: memory/, .learnings/, state/, logs/');
  console.log('  🟢 随时清理: tmp/');
}

initWorkspace();