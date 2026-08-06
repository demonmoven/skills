# Test Case 示例

## 组件级别示例

**文件名**: `phase-4-case-1-python-syntax-highlight.yaml`

```yaml
web:
  auth:
    email: admin@example.com
    password: admin123
    strategy: login_then_register
  urls:
    - url: http://localhost:8082/components-preview?component=Code%20Evaluator%2F%E5%9F%BA%E7%A1%80%E7%BB%84%E4%BB%B6%2FCodeEvaluatorEditor%2FPython%20Example
      description: CodeEvaluatorEditor 组件预览页面 - Python 示例

验收场景:

- CodeEvaluatorEditor 代码编辑器组件
  - case: Python 代码语法高亮
    - pc: 用户在 Python Example 预览页面
      - ts: 1. 查看代码编辑器 2. 输入 Python 代码
        - ✅ 编辑器显示 Python 代码
        - ✅ def、return 等关键字显示不同颜色
        - ✅ 字符串、注释显示不同颜色
        - ✅ 支持代码自动补全
```

## 组件禁用状态示例（含 ❌）

**文件名**: `phase-4-case-3-disabled-state.yaml`

```yaml
web:
  auth:
    email: admin@example.com
    password: admin123
    strategy: login_then_register
  urls:
    - url: http://localhost:8082/components-preview?component=Code%20Evaluator%2F%E5%9F%BA%E7%A1%80%E7%BB%84%E4%BB%B6%2FCodeEvaluatorEditor%2FDisabled%20(Readonly)
      description: CodeEvaluatorEditor 组件预览页面 - 禁用（只读）状态

验收场景:

- CodeEvaluatorEditor 代码编辑器组件
  - case: 编辑器禁用状态
    - pc: 用户在 Disabled 预览页面
      - ts: 1. 尝试点击编辑器 2. 尝试输入代码
        - ✅ 编辑器显示只读状态
        - ❌ 无法编辑代码内容
        - ❌ 无法获取输入焦点
```

## 用户故事级别示例

**文件名**: `phase-11-case-6-try-run.yaml`

```yaml
web:
  auth:
    email: admin@example.com
    password: admin123
    strategy: login_then_register
  urls:
    - url: http://localhost:8082/console/enterprise/1/space/1/evaluation/evaluators/create?type=2
      description: Code 评估器配置页面

验收场景:

- Code 评估器配置界面
  - case: 试运行功能
    - pc: 用户编辑了执行函数体和测试数据
      - ts: 1. 点击 "试运行" 按钮
        - ✅ 系统使用测试数据执行函数体
        - ✅ 配置区域下方显示试运行结果
        - ✅ 结果包含：总条数、成功条数、失败条数
        - ✅ 结果包含：状态（loading/success/fail）、得分、原因
```

## 多 URL 示例

**文件名**: `phase-10-case-2-template-switching.yaml`

```yaml
web:
  auth:
    email: admin@example.com
    password: admin123
    strategy: login_then_register
  urls:
    - url: http://localhost:8082/console/enterprise/1/space/1/evaluation/evaluators/create?type=2
      description: Code 评估器创建页面
    - url: http://localhost:8082/console/enterprise/1/space/1/evaluation/evaluators/create?type=1
      description: LLM 评估器创建页面

验收场景:

- 评估器模板选择
  - case: Code 模板切换
    - pc: 用户在 Code 评估器创建页面
      - ts: 1. 点击模板选择下拉框 2. 选择不同模板
        - ✅ 模板列表正确显示
        - ✅ 切换后代码区域自动更新
        - ✅ 测试数据区域同步更新
```
