# Code Patterns Guide

本文件记录本仓库中已建立的代码模式和惯例。
新代码应遵循这些模式以保持一致性。

## 命名惯例

<!-- HARNESS-GEN: 从 Phase 1 代码模式采样中提取命名惯例，
     列出文件命名、函数命名、变量命名的实际规律。
     如果 Phase 1 未能确定，标记为 Knowledge Gap。 -->

| 类别 | 规则 | 示例 |
|------|------|------|
| 文件命名 | <!-- HARNESS-GEN: 从代码模式采样中提取文件命名规律 --> | |
| 函数命名 | <!-- HARNESS-GEN: 从代码模式采样中提取函数命名规律 --> | |
| 变量命名 | <!-- HARNESS-GEN: 从代码模式采样中提取变量命名规律 --> | |
| 类型/接口命名 | <!-- HARNESS-GEN: 从代码模式采样中提取类型命名规律 --> | |
| 常量命名 | <!-- HARNESS-GEN: 从代码模式采样中提取常量命名规律 --> | |

## 错误处理模式

<!-- HARNESS-GEN: 从 Phase 1 代码模式采样中提取典型的错误处理方式。
     分析入口文件和核心业务文件中的 try/catch、Result/Error 类型、
     error boundary、panic recovery 等模式。
     如果 Phase 1 未能确定，标记为 Knowledge Gap。 -->

## 常见开发模式

### 新增功能的标准流程

<!-- HARNESS-GEN: 从代码模式采样中提取新增功能的标准流程。
     根据仓库类型，可能是：
     - "新增 API endpoint" 的步骤
     - "新增 React 组件" 的步骤
     - "新增 DI service" 的步骤
     - "新增 CLI 命令" 的步骤
     列出从创建文件到注册/导出的完整步骤。 -->

### 模块间通信模式

<!-- HARNESS-GEN: 从 Phase 1 依赖图分析中提取模块间的通信方式。
     例如：直接 import、事件总线、DI 注入、RPC 调用等。 -->

### 状态管理模式

<!-- HARNESS-GEN: 从代码模式采样中提取状态管理方式。
     如果 Phase 1 未检测到明确的状态管理模式，标记为 Knowledge Gap。 -->

## 测试编写模式

<!-- HARNESS-GEN: 从 Phase 1 测试模式分析中提取：
     - 测试文件的组织方式（co-located vs 独立 test 目录）
     - 测试命名惯例
     - Mock/Stub 策略（使用的 mock 库、fixture 文件位置、in-memory 替代方式）
     - 常用的测试辅助函数或 test utilities
     - 集成测试 vs 单元测试的区分标准
     如果 Phase 1 未能确定，标记为 Knowledge Gap。 -->
