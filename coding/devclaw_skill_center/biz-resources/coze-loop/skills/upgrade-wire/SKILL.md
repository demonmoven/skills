---
name: upgrade-wire
description: 在本项目中需要修改依赖注入配置时使用。
---
# 依赖注入 Skill

当在本项目中需要修改依赖注入（Dependency Injection）配置时，使用此 Skill 完成 Wire 配置修改和代码生成流程。

## 重要原则

- **修改 wire.go**：只修改 `wire.go` 文件来定义依赖注入
- **自动生成**：禁止手动修改 `wire_gen.go` 文件，必须通过 `wire` 命令生成
- **模块隔离**：每个模块有自己独立的 wire 配置
- **保持一致**：确保依赖注入配置与实际代码结构一致

## Wire 文件位置

项目中的 wire 配置文件分布在各个模块的 `application` 目录下：

```
backend/
├── modules/
│   ├── prompt/application/
│   │   ├── wire.go          # 手动编辑
│   │   └── wire_gen.go      # 自动生成，禁止修改
│   ├── evaluation/application/
│   │   ├── wire.go
│   │   └── wire_gen.go
│   ├── foundation/application/
│   ├── llm/application/
│   ├── data/application/
│   └── observability/application/
└── api/handler/coze/loop/apis/
    ├── wire.go
    └── wire_gen.go
```

## Wire 文件结构

`wire.go` 文件通常包含：
- `//go:build wireinject` 构建标签
- Wire 提供者（Providers）定义
- 初始化函数（Injectors）定义
- 使用 `wire.Build()` 声明依赖关系

## 执行步骤

当用户请求修改依赖注入时，**必须**按以下步骤执行：

### 1. 修改 wire.go 文件

- 定位到对应模块的 `wire.go` 文件
- 根据需求修改：
  - 添加新的 Provider 函数
  - 修改 `wire.Build()` 中的依赖列表
  - 更新 Injector 函数签名
- 确保 import 声明正确

### 2. 执行 Wire 代码生成

- **必须**使用 Bash 工具执行：`.claude/skills/upgrade-wire/scripts/run.sh`
- 脚本会自动查找所有 `wire.go` 文件并生成对应的 `wire_gen.go`
- **等待**脚本完成，不要中断
- **检查**输出中是否有错误信息
- 如果失败，**报告**错误并停止

### 3. 验证生成的代码

- 检查 `wire_gen.go` 文件是否正确生成
- 确认没有编译错误
- 验证依赖注入逻辑符合预期

## 使用示例

- 为某模块新增服务依赖
- 修改现有服务的初始化逻辑
- 添加新的 Repository 或 Service 到依赖图
- 重构模块的依赖关系

## 注意事项

- **绝对禁止**手动修改 `wire_gen.go` 文件
- 修改 `wire.go` 后必须执行 wire 命令重新生成
- 确保 wire 工具已安装（`go install github.com/google/wire/cmd/wire@latest`）
- 依赖注入失败时，检查循环依赖和类型匹配问题
- Provider 函数的返回值类型必须与使用方的参数类型匹配