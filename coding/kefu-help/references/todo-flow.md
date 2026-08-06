# 进度追踪规范

## 概述

kefu_help 使用 TodoWrite 工具进行任务进度追踪，确保复杂任务有序执行。

## 任务状态

| 状态 | 说明 |
|------|------|
| pending | 待处理，尚未开始 |
| in_progress | 进行中，正在执行 |
| completed | 已完成，成功结束 |

## 任务优先级

| 优先级 | 说明 |
|--------|------|
| high | 高优先级，必须完成 |
| medium | 中优先级，重要但不紧急 |
| low | 低优先级，可以延后 |

## 任务结构

```json
{
  "id": "unique-task-id",
  "content": "任务描述",
  "status": "pending|in_progress|completed",
  "priority": "high|medium|low",
  "created_at": "ISO 8601 时间戳"
}
```

## 工作流程

### 1. 任务规划

在开始复杂任务前，先创建任务列表：

```json
{
  "todos": [
    {"id": "1", "content": "分析需求", "status": "pending", "priority": "high"},
    {"id": "2", "content": "设计方案", "status": "pending", "priority": "high"},
    {"id": "3", "content": "实现功能", "status": "pending", "priority": "high"},
    {"id": "4", "content": "编写测试", "status": "pending", "priority": "medium"},
    {"id": "5", "content": "代码审查", "status": "pending", "priority": "medium"}
  ]
}
```

### 2. 任务执行

执行任务时遵循以下规则：

1. **单一进行中**: 同时只有一个任务处于 in_progress 状态
2. **完成即更新**: 任务完成后立即标记为 completed
3. **顺序执行**: 按优先级和依赖关系执行任务

### 3. 任务完成

任务完成后提供摘要：

```json
{
  "todos": [
    {"id": "1", "content": "分析需求", "status": "completed", "priority": "high"},
    {"id": "2", "content": "设计方案", "status": "completed", "priority": "high"},
    {"id": "3", "content": "实现功能", "status": "completed", "priority": "high"},
    {"id": "4", "content": "编写测试", "status": "completed", "priority": "medium"},
    {"id": "5", "content": "代码审查", "status": "completed", "priority": "medium"}
  ],
  "summary": "所有任务已完成，功能实现并通过测试"
}
```

## 最佳实践

### 1. 任务粒度

- 每个任务应该是可独立完成的单元
- 避免过于笼统的任务描述
- 复杂任务拆分为多个子任务

### 2. 优先级设置

- 核心功能设为 high
- 优化和文档设为 medium
- 可选功能设为 low

### 3. 状态更新

- 开始任务前标记为 in_progress
- 完成任务后立即标记为 completed
- 遇到阻塞时记录原因

### 4. 进度可见

- 定期更新任务状态
- 提供清晰的进度反馈
- 标注已完成的工作

## 示例场景

### 场景 1: 新功能开发

```json
{
  "todos": [
    {"id": "1", "content": "阅读需求文档", "status": "completed", "priority": "high"},
    {"id": "2", "content": "设计数据模型", "status": "completed", "priority": "high"},
    {"id": "3", "content": "实现 API 接口", "status": "in_progress", "priority": "high"},
    {"id": "4", "content": "编写单元测试", "status": "pending", "priority": "medium"},
    {"id": "5", "content": "更新文档", "status": "pending", "priority": "low"}
  ]
}
```

### 场景 2: Bug 修复

```json
{
  "todos": [
    {"id": "1", "content": "复现问题", "status": "completed", "priority": "high"},
    {"id": "2", "content": "定位根因", "status": "completed", "priority": "high"},
    {"id": "3", "content": "编写修复代码", "status": "in_progress", "priority": "high"},
    {"id": "4", "content": "验证修复", "status": "pending", "priority": "high"},
    {"id": "5", "content": "添加回归测试", "status": "pending", "priority": "medium"}
  ]
}
```

### 场景 3: 代码重构

```json
{
  "todos": [
    {"id": "1", "content": "分析现有代码", "status": "completed", "priority": "high"},
    {"id": "2", "content": "设计重构方案", "status": "completed", "priority": "high"},
    {"id": "3", "content": "提取公共方法", "status": "completed", "priority": "medium"},
    {"id": "4", "content": "更新调用方", "status": "in_progress", "priority": "medium"},
    {"id": "5", "content": "运行测试验证", "status": "pending", "priority": "high"}
  ]
}
```

## 任务依赖

### 串行依赖

任务之间有依赖关系，必须按顺序执行：

```
任务A → 任务B → 任务C
```

### 并行执行

无依赖的任务可以并行执行：

```
任务A ─┬→ 任务D
任务B ─┤
任务C ─┘
```

### 条件执行

根据条件决定是否执行：

```
任务A → [条件判断] → 任务B (条件满足)
                    → 任务C (条件不满足)
```

## 进度报告

### 每日报告

```
今日完成:
- [x] 任务1: 分析需求
- [x] 任务2: 设计方案
- [ ] 任务3: 实现功能 (进行中)

明日计划:
- [ ] 完成任务3
- [ ] 开始任务4
```

### 周报

```
本周进度:
- 已完成: 8 个任务
- 进行中: 2 个任务
- 待处理: 3 个任务

完成率: 61.5%
```

## 故障处理

### 任务阻塞

当任务遇到阻塞时：

1. 标记阻塞原因
2. 寻找替代方案
3. 必要时调整优先级

### 任务取消

当任务不再需要时：

1. 从列表中移除
2. 记录取消原因
3. 更新依赖任务

### 任务拆分

当任务过于复杂时：

1. 分析任务组成部分
2. 拆分为多个子任务
3. 重新设置优先级
