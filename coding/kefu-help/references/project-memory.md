# 项目记忆管理指南

## 概述

项目记忆系统用于存储决策记录、踩坑经验和最佳实践，实现知识的积累和复用。

## 记忆存储

### 存储位置

```
.kefu/
├── memory/
│   └── archive.json    # 项目记忆存档
├── logs/               # 对话日志
└── tasks/              # 任务进度
```

### archive.json 结构

```json
{
  "project": "my-project",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "decisions": [
    {
      "id": "dec-001",
      "title": "选择 PostgreSQL 作为主数据库",
      "context": "需要支持复杂查询和事务",
      "decision": "使用 PostgreSQL",
      "alternatives": ["MySQL", "MongoDB"],
      "rationale": "PostgreSQL 提供更好的 JSON 支持和扩展性",
      "created_at": "2024-01-05T14:00:00Z"
    }
  ],
  "lessons": [
    {
      "id": "les-001",
      "title": "避免在循环中执行数据库查询",
      "problem": "性能问题：N+1 查询",
      "solution": "使用批量查询或 JOIN",
      "created_at": "2024-01-10T09:00:00Z"
    }
  ],
  "patterns": [
    {
      "id": "pat-001",
      "name": "Repository Pattern",
      "description": "数据访问层抽象",
      "usage": "用于解耦业务逻辑和数据访问",
      "files": ["src/repository/user.go", "src/repository/order.go"]
    }
  ]
}
```

## 记忆类型

### 1. 决策记录 (Decisions)

记录重要的技术决策：

```json
{
  "id": "dec-xxx",
  "title": "决策标题",
  "context": "决策背景",
  "decision": "做出的决定",
  "alternatives": ["备选方案1", "备选方案2"],
  "rationale": "选择理由",
  "consequences": "预期影响",
  "created_at": "ISO 8601 时间戳"
}
```

### 2. 踩坑经验 (Lessons)

记录遇到的问题和解决方案：

```json
{
  "id": "les-xxx",
  "title": "问题标题",
  "problem": "问题描述",
  "solution": "解决方案",
  "prevention": "预防措施",
  "created_at": "ISO 8601 时间戳"
}
```

### 3. 设计模式 (Patterns)

记录项目中的设计模式：

```json
{
  "id": "pat-xxx",
  "name": "模式名称",
  "description": "模式描述",
  "usage": "使用场景",
  "files": ["相关文件列表"]
}
```

## CLI 命令

### 查看记忆

```bash
# 查看所有记忆
kefu config get memory

# 查看决策记录
kefu config get memory.decisions

# 查看踩坑经验
kefu config get memory.lessons

# 查看设计模式
kefu config get memory.patterns
```

### 添加记忆

```bash
# 添加决策
kefu config set memory.add_decision '{"title": "...", "decision": "..."}'

# 添加经验
kefu config set memory.add_lesson '{"title": "...", "solution": "..."}'

# 添加模式
kefu config set memory.add_pattern '{"name": "...", "description": "..."}'
```

### 清除记忆

```bash
# 清除所有记忆
kefu config set memory.clear true

# 清除特定类型
kefu config set memory.clear_decisions true
kefu config set memory.clear_lessons true
kefu config set memory.clear_patterns true
```

## 使用场景

### 场景 1: 记录技术选型

当需要做出重要技术决策时：

1. 分析需求和约束
2. 列出备选方案
3. 评估各方案优劣
4. 做出决策并记录
5. 存储到 archive.json

### 场景 2: 记录踩坑经验

当遇到并解决问题时：

1. 记录问题现象
2. 分析根本原因
3. 记录解决方案
4. 总结预防措施
5. 存储到 archive.json

### 场景 3: 复用设计模式

当发现可复用的模式时：

1. 识别模式特征
2. 提取通用结构
3. 记录使用场景
4. 关联相关文件
5. 存储到 archive.json

## 最佳实践

### 1. 及时记录

- 决策后立即记录
- 解决问题后立即总结
- 发现模式后立即提取

### 2. 结构化存储

- 使用统一的 JSON 格式
- 包含完整的上下文信息
- 添加时间戳便于追溯

### 3. 定期回顾

- 每周回顾决策记录
- 每月总结踩坑经验
- 每季度更新模式库

### 4. 团队共享

- 将 archive.json 纳入版本控制
- 定期同步团队记忆
- 新成员入职时介绍项目记忆

## Git 集成

### 版本控制

```bash
# 将记忆纳入版本控制
git add .kefu/memory/archive.json
git commit -m "docs: update project memory"
```

### 变更追踪

```bash
# 查看记忆变更历史
git log --oneline .kefu/memory/archive.json

# 查看特定决策的变更
git log -p -- .kefu/memory/archive.json
```

## 隐私和安全

### 敏感信息

不要在记忆中存储：
- 密码和密钥
- 个人身份信息
- 商业机密

### 访问控制

```bash
# 设置记忆文件权限
chmod 600 .kefu/memory/archive.json
```

## 故障排除

### 记忆丢失

```bash
# 从 Git 恢复
git checkout HEAD -- .kefu/memory/archive.json
```

### 格式错误

```bash
# 验证 JSON 格式
cat .kefu/memory/archive.json | jq .

# 修复格式
jq '.' .kefu/memory/archive.json > temp.json && mv temp.json .kefu/memory/archive.json
```

### 记忆过大

```bash
# 归档旧记忆
mv .kefu/memory/archive.json .kefu/memory/archive-2023.json

# 创建新的记忆文件
echo '{"project": "my-project", "decisions": [], "lessons": [], "patterns": []}' > .kefu/memory/archive.json
```
