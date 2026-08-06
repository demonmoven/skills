---
name: upgrade-sql
description: 在本项目中需要进行数据库表变更（MySQL/ClickHouse）时使用。
---
# 数据库表变更 Skill

当在本项目中需要进行数据库表变更（MySQL/ClickHouse）时，使用此 Skill 完成表定义修改和代码生成流程。

## 重要原则

- **向前兼容**：所有变更都必须保证向前兼容
- **不兼容变更**：如果需要进行不兼容变更，请人工确认后再进行
- **模块专注**：每次变更应专注于单个模块，不要改动其他模块的数据库表
- **只增不改**：新增字段时，不要修改或删除历史字段

## 数据库表定义位置

项目支持多种部署方式，变更数据库表时需要同时变更对应的多个位置，保证一致性：

```json
{
  "MySQL": {
    "Docker": "release/deployment/docker-compose/bootstrap/mysql-init/init-sql",
    "Kubernetes": "release/deployment/helm-chart/charts/app/bootstrap/init/mysql/init-sql"
  },
  "ClickHouse": {
    "Docker": "release/deployment/docker-compose/bootstrap/clickhouse-init/init-sql",
    "Kubernetes": "release/deployment/helm-chart/charts/app/bootstrap/init/clickhouse/init-sql"
  }
}
```

## 执行步骤

当用户请求进行数据库表变更时，**必须**按以下步骤执行：

### 1. 修改数据表定义

**新增数据表：**
- 在上述对应目录（MySQL/ClickHouse 的 Docker 和 Kubernetes）下新增文件
- 文件名为新增数据表名称
- 确保在 **4 个位置** 都创建相同的表定义

**修改已有数据表（新增字段）：**
- 在上述对应目录的现有文件中修改
- **只能新增字段**，不要修改或删除历史字段
- 确保在 **4 个位置** 都进行相同的修改

### 2. 检查是否需要修改 generate.go

- 路径：`backend/script/gorm_gen/generate.go`
- 如果新增了数据表，需要在该文件中添加对应的表配置
- 注意软删标记的设置（如 `deleted_at` 字段）

### 3. 执行代码生成脚本

- **必须**使用 Bash 工具执行：`.claude/skills/upgrade-sql/scripts/run.sh`
- **等待**脚本完成，不要中断
- **检查**输出中是否有错误信息
- 如果失败，**报告**错误并停止

## 使用示例

- 为某模块新增一张数据表
- 在已有数据表中新增字段
- 修改表的索引定义
- 新增 ClickHouse 分析表

## 注意事项

- 变更前确认当前开发的模块范围
- 同时修改 4 个位置的表定义文件（MySQL Docker/K8s + ClickHouse Docker/K8s）
- 保持向前兼容，避免破坏性变更
- 新增表时记得更新 generate.go