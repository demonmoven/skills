# 自动生成代码与豁免规则（Go Exclusions）

---

## 1. 自动生成代码目录（跳过审查）

| 目录/模式 | 说明 |
|-----------|------|
| `loop_gen/` | 由 IDL 工具自动生成的代码 |
| `kitex_gen/` | 由 Kitex 框架自动生成的 RPC 代码 |
| `**/*.gen.go` | 任何以 .gen.go 结尾的文件（如 GORM Gen 生成代码） |

这些代码通过工具从 IDL 定义自动生成，不能直接修改，只能通过修改 IDL 后重新生成。

---

## 2. 基础设施与配置豁免

以下文件属于基础设施即代码 (IaC) 或本地开发配置，不属于业务代码审查范围：
- `release/deployment/` 下的所有文件 (docker-compose, minio-init 等)
- `.claude/` 下的 IDE 配置
- `.vscode/`, `.idea/` 等编辑器配置
- 测试专用的 override 文件 (如 `docker-compose.override.yml`)

---

## 3. Linter 自动检查项豁免

以下问题应由 CI/Linter 自动处理，不需要人工审查：
- 变量/函数命名风格（驼峰/下划线）
- Import 排序
- 简单的 JSON Tag 缺失
- 简单的格式化问题
