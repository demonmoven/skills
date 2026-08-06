---
name: auto-test-case
description: 网页模块自动化测试。探索用例、执行测试、录制请求。
---

# 网页模块自动化测试

## 前置依赖

### 安装 CLI

```bash
bash -c "$(curl -fsSL https://tosv.byted.org/obj/ies-cs-rtf/flou/install.sh)"
flou-cli --version
```

## 常量定义
- **RESULT_DIR**: `${PROJECT_DIR}/.flou/test`

## 核心流程

**测试用例生成流程** - [FLOW-test-case-generation.md](references/FLOW-test-case-generation.md)

该流程包含以下阶段：
1. **需求分析** - 挖掘测试需求，生成问题清单
2. **用户澄清** - 确认用例来源、测试方式、预期结果、产物格式
3. **用例生成** - 根据来源类型生成标准化测试用例
4. **用例审查** - 用户审查用例，通过后才能执行测试
5. **测试执行** - 根据测试方式执行浏览器测试/接口测试/组合测试
6. **报告生成** - 生成标准化测试报告
7. **用户确认** - 用户确认测试结果
8. **归档** - 整理产物并归档到记忆

## 场景路由

### 探索模块
- **场景**: 探索网页模块的元素定位方法
- **路由**: [references/explore.md](references/explore.md)
- **触发条件**: 用户需要探索网页模块的元素定位

### 测试模块
- **场景**: 基于探索结果执行自动化测试
- **路由**: [references/test.md](references/test.md)
- **触发条件**: 用户需要执行自动化测试

### 请求录制模块
- **场景**: 录制 Web 项目请求并分析生成接口文档
- **路由**: [references/record-req.md](references/record-req.md)
- **触发条件**: 用户需要录制 Web 请求并生成接口文档

### RPC测试模块
- **场景**: 完成RPC接口测试
- **路由**: [references/rpc-test.md](references/rpc-test.md)
- **触发条件**: 用户需要进行RPC接口测试

## 注意事项
- 优先使用 text 标记定位元素
- 不建议使用 class 标记可能有重复
- **执行测试前必须完成用户澄清阶段**，确认以下信息：
  - 用例来源（手动描述/探索发现/已有用例/接口文档）
  - 测试方式（接口测试/浏览器测试/都要）
  - 预期结果（操作成功/数据验证/截图验证/自定义）
  - 产物格式（标准报告/详细报告/简洁报告/自定义）
