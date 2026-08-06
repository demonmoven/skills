---
name: upgrade-config
description: 在本项目中需要修改服务运行时配置文件时使用。
---
# 配置变更 Skill

当在本项目中需要修改服务运行时配置文件时，使用此 Skill 完成配置变更流程。

## 重要原则

- **向前兼容**：所有配置变更都必须保证向前兼容
- **不兼容变更**：如果需要进行不兼容配置变更，请人工确认后再进行
- **模块专注**：每次变更应专注于单个模块，不要改动其他模块的配置
- **保持一致**：同时修改多个部署方式的配置文件，确保一致性

## 配置文件位置

项目支持多种部署方式，变更配置时需要同时变更对应的多个位置，保证一致性：

```json
{
  "Docker": "release/deployment/docker-compose/conf",
  "Kubernetes": "release/deployment/helm-chart/umbrella/conf"
}
```

## 模块配置文件命名规则

各模块配置文件均在上述目录下，文件命名规则：
- **通用模块**：文件名为模块名，例如 `foundation.yaml`、`prompt.yaml`
- **LLM 模块特殊说明**：
  - `model_runtime_config.yaml` - LLM 模块的运行时配置文件（开发时需要变更）
  - `model_config.yaml` - 模型列表配置（一般不需要变更）

## 执行步骤

当用户请求进行配置变更时，**必须**按以下步骤执行：

### 1. 确定变更的模块和配置项

- 明确要修改的模块名称
- 确认具体需要变更的配置项
- 如果是 LLM 模块，确认是修改运行时配置还是模型列表配置

### 2. 修改配置文件

- 在 Docker 部署配置目录修改：`release/deployment/docker-compose/conf/{module}.yaml`
- 在 Kubernetes 部署配置目录修改：`release/deployment/helm-chart/umbrella/conf/{module}.yaml`
- 确保 **2 个位置** 的配置修改完全一致
- 保持向前兼容，不要删除已有配置项

### 3. 验证配置文件格式

- **必须**使用 Bash 工具执行：`.claude/skills/upgrade-config/scripts/run.sh`
- **等待**脚本完成，不要中断
- **检查**输出中是否有 YAML 格式错误
- 如果失败，**报告**错误并停止

## 使用示例

- 为 prompt 模块新增配置项
- 修改 foundation 模块的某个配置值
- 调整 LLM 模块的运行时配置
- 更新某模块的超时时间配置

## 注意事项

- 变更前确认当前开发的模块范围
- 同时修改 2 个位置的配置文件（Docker + Kubernetes）
- 保持向前兼容，避免破坏性变更
- LLM 模块注意区分运行时配置和模型列表配置
- 验证 YAML 文件格式正确性