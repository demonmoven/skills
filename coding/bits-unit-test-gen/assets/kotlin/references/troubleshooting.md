# Kotlin Unit Test Troubleshooting

只修测试文件、测试依赖和测试配置；不修改生产代码迁就测试。

## 常见失败信号

| 信号                                     | 优先检查                             | 修复方向                              |
|----------------------------------------|----------------------------------|-----------------------------------|
| `NPE`/`KotlinNPE`                      | mock 返回 null 给非空签名；测试向非空参数传 null | 修正 stub/输入                        |
| `ClassCastException`                   | mock 返回值/泛型类型不匹配                 | 修正 stub 类型                        |
| `UninitializedPropertyAccessException` | `lateinit` 未初始化；mock 注入失败        | 检查 `@InjectMockKs`/`@InjectMocks` |
| `Unresolved reference`                 | import/依赖/source set/模块边界        | 修正 import 或补 test 依赖              |
| `NoClassDefFoundError`                 | test 依赖缺失或 scope 错误              | 补全 `testImplementation`           |
| 协程超时                                   | `runBlocking`/真实 delay/未结束协程     | 改 `runTest`，替换 dispatcher         |

## MockK

- `no answer found`：参数匹配不一致
- `Failed to transform`：版本/JDK/agent 问题
- `Verification failed`：检查 `any()` vs 具体值、nullable

## Mockito-Kotlin

- `Wanted but not invoked`：参数匹配器不一致
- `NPE`：`any()` import 错误（见 mockito-kotlin.md）
- `Cannot mock/spy`：final/object，检查 mock maker

## 协程

- 不要用 `runBlocking` 测 suspend
- `Dispatchers.Main` 需 rule/setMain 替换
- `UncompletedCoroutinesError`：未完成协程/无限 collect/未关闭 channel → mock 挂起依赖、对无限 flow 用 `take(n)`
  、确保协程在测试作用域内可终止

## 原则

- 先验证测试构造正确性，再判定业务缺陷
- 不要改期望值来"修复"测试
- 构建工具/平台选择错误时回到对应 reference
