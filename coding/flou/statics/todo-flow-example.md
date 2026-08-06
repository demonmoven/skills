---
task-id: TASK-{date}-{task-name}
summary: {user_input}
status: 执行中
create-time: {timestamp}
---

# Todo-Flow

## 流程引用

| Flow ID | 流程名称 | 流程文件 | 状态 | 启动时间 |
|---------|---------|---------|------|---------|
| 0 | Flou 工作流程 | interactive_flow.md | 执行中 | {timestamp} |

**当前活跃流程**: Flow 0 (Flou 工作流程)

---

## 任务信息

- **任务ID**: TASK-{date}-{task-name}
- **创建时间**: {timestamp}
- **项目路径**: {project_path}
- **用户原始输入**: {user_input}

---

## Flow 0: Flou 工作流程

**流程文件**: [interactive_flow.md](interactive_flow.md)

### 当前进度

| 阶段 | 状态 | 完成时间 |
|------|------|---------|
| 初始化 | ✅ 完成 | {timestamp} |
| 澄清 | 🔄 进行中 | - |
| 执行 | ⏳ 待执行 | - |
| 小结 | ⏳ 待执行 | - |
| 归档 | ⏳ 待执行 | - |

### 阶段详情

#### 初始化
- [x] 创建任务ID
- [x] 创建任务目录
- [x] 初始化 todo-flow 文件

#### 澄清
- [ ] 检索项目记忆
- [ ] 澄清任务流程
- [ ] 选择任务流程
- [ ] 用户确认继续

**当前节点**: 澄清 - 检索项目记忆
**下一步**: 检索项目记忆并澄清任务需求

---

## 归档记录

待补充

## 项目记忆摘要

待补充
