---
name: upgrade-idl
description: 在本项目中需要执行新增/ 修改/ 删除接口时使用。
---
# IDL 升级 Skill

当在本项目中需要执行新增/ 修改/ 删除接口时，使用此 Skill 完成IDL变更、代码生成和验证流程。

## 项目结构

- IDL 路径：`idl/thrift/coze/loop/{module_name}/`
- 模块：foundation、prompt、evaluation、llm、data、observability、apis等

## 执行步骤

当用户请求进行接口变更时，**必须**按以下步骤执行：

### 1. IDL 代码变更
- 修改对应模块的 `.thrift` 文件
- 路径格式：`idl/thrift/coze/loop/{module_name}/xxx.thrift`
- 确保 Thrift 语法正确

### 2. 执行代码生成和验证脚本
- **必须**使用 Bash 工具执行：`.claude/skills/upgrade-idl/scripts/run.sh`
- **等待**脚本完成，不要中断
- **检查**输出中是否有 "Error" 或失败信息
- 如果失败，**报告**错误并停止

## 使用示例

- 为 prompt 模块新增一个接口
- 修改 foundation 模块的某个接口参数
- 删除 evaluation 模块的废弃接口
- 批量更新多个模块的 IDL 定义