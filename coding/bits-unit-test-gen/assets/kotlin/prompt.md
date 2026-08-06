# Kotlin 语言专属 Prompt

本文件内容包括：

- Kotlin 目标提取与处理粒度
- Kotlin 语义特有的过滤规则
- 测试文件组织的默认基线
- 与具体 Mock 框架无关的通用约束
- 协程测试的最小共识
- 其他必要单测知识

优先级：用户显式指令 > `AGENTS.md` / `CLAUDE.md` > 目标目录已有测试风格 > 本文件默认规则 > 按需加载的 reference 默认规则。

## 预加载规则

写测试前，先识别项目类型、构建工具、测试 source set 和 mock 框架，然后只加载相关 reference：

- Android 项目：加载 `android.md`
- KMP 项目：加载 `kmp.md`
- Gradle 项目：加载 `gradle.md`
- Maven 项目：加载 `maven.md`
- 已有测试使用 MockK 或无既有测试：加载 `mockk.md`
- 已有测试使用 Mockito-Kotlin / Mockito：加载 `mockito-kotlin.md`
- 验证失败或环境异常时：加载 `troubleshooting.md`

多个 reference 可同时加载。不要预设项目一定是 Gradle、JVM、MockK 或 Android；必须先读构建文件和邻近测试再决定。

### 组合规则

- **构建工具**（互斥）：`gradle.md` 与 `maven.md` 不会同时出现
- **平台**：按检测结果组合加载
  - 纯 Android（无 `kotlin {}` 多 target）→ `android.md`
  - KMP 无 Android target → `kmp.md`
  - KMP 含 `androidTarget()` → `kmp.md` + `android.md`

构建工具与平台跨组可自由组合。

### 识别信号

- `build.gradle(.kts)` / `settings.gradle(.kts)` → `gradle.md`
- `pom.xml`（目标 module 无 Gradle 文件）→ `maven.md`
- `com.android.application` / `com.android.library` / `android {}` → `android.md`
- `kotlin {}` + 多 target / 存在 `commonMain`、`commonTest` → `kmp.md`
- `@MockK`、`every {}`、`coEvery {}` → `mockk.md`
- `@Mock`、`whenever`、`org.mockito.kotlin` → `mockito-kotlin.md`
- 目标模块无既有测试文件 → `mockk.md`（默认）

## 工作单元与调度策略

- 目标选择：函数/方法
- 生成与修复：声明容器（类/object/companion → 该类；顶层/扩展函数 → 该文件）
- 验证：测试类
- 调度：按声明容器串行

同一声明容器内的多个目标函数一起生成，统一维护 import、fixture、mock。当前声明容器收敛后再进入下一个。

记录信息：`container_kind`（class/object/companion/file）、`container_name`（类为完全限定名，文件级为路径；JVM 顶层声明必要时补
facade 名如 `FooKt`）、所属模块、源文件路径、目标函数及起始行、测试文件、测试命令和工作目录。

## Kotlin 目标提取

- 顶层函数、扩展函数、`object` / `companion object` 方法均可作为独立测试目标
- 扩展函数必须记录 `receiver_type`
- 重载函数必须结合参数列表和起始行区分
- 局部函数默认不直接测试，通过外层函数间接覆盖
- 额外记录：`receiver_type`（扩展函数）、`signature`（重载）、`is_suspend`

## Kotlin 目标过滤

跳过以下声明：

- `fun main()`
- `data class` 自动生成方法（`copy`/`toString`/`hashCode`/`equals`/`componentN`）
- `enum class` 的 `values()`/`valueOf()`/`entries`
- `object`/`companion object` 中仅做常量声明的部分
- 行数 < 3 且无分支的简单属性访问器
- `sealed class/interface` 本身（测试其子类）
- `@Composable` 函数（除非用户显式要求）
- 自动生成代码（`*Generated*`、Proto/Thrift 生成文件）

## 预检查

每个声明容器写测试前：

1. 确认构建工具、模块结构和测试 source set，加载对应 reference
2. 阅读邻近 1-2 个 `*Test.kt`，确认：测试框架（JUnit 5/4）、Mock 策略、断言库、协程测试、命名风格
3. 搜索可复用测试资产（`TestHelper`、`BaseTest`、`testModule`、dispatcher rule 等）
4. 无参考时回退到本文默认约定

## 默认风格

| 项目      | 默认约定                                                         |
|---------|--------------------------------------------------------------|
| 注释语言   | 中文                                                           |
| 命名      | `test{Method}_{scenarioDesc}_BitsUT` 或反引号 `` `{method} {scenarioDesc} BitsUT` `` |
| 文件      | `<ClassName>Test.kt`，跟随目标 source set，优先追加既有测试文件              |
| 断言      | 沿用项目已有；无参考时用 JUnit Assertions                                |
| Mock    | 沿用项目已有；无参考时默认 MockK                                          |
| 协程      | `kotlinx.coroutines.test.runTest`                            |
| Import  | 标准库 → 第三方 → 内部，空行分隔                                          |
| Package | 与被测代码一致                                                      |

### 代码语言硬约束

> **禁止在生成的代码中除注释以外的任何位置使用中文。** 方法名、变量名、字符串字面量（含 `@DisplayName` 参数）、枚举值名等均必须使用英文。注释（`//`、`/* */`、`/** doc */`）允许使用中文。

## Mock 通用约束

- 只 mock 不可控外部依赖（RPC、DB、缓存、时间、网络、文件系统）
- `inline fun` 无法被 mock（编译期内联）；需 mock 其内部调用的依赖
- `value class` 运行时被擦除，mock 框架无法代理；使用真实实例
- 不要给非空签名返回 `null`
- 仅对不关心返回值的辅助依赖使用 relaxed/lenient 模式
- 具体 API 从对应 mock reference 获取

## 协程测试基线

- 默认使用 `runTest`；不要默认用 `runBlocking`
- 需要控制时间时注入 `TestDispatcher`（精确控制用 `StandardTestDispatcher`，立即执行用 `UnconfinedTestDispatcher`）
- `delay`/`withTimeout` 逻辑使用虚拟时间推进

Flow 测试：

- `StateFlow`/`SharedFlow` 永不 complete，禁止直接 `toList()`；用 `.value` 或 `take(n).toList()`/Turbine
- cold flow 无限流同样需要 `take(n)` 或 `first()`
- `launch { flow.collect {} }` 必须在测试结束前取消
- 被测代码内部启动新协程时，使用 `backgroundScope.launch` 避免 `UncompletedCoroutinesError`
- 邻近测试用 Turbine 时沿用；未使用时不主动引入

## 验证规则

- 默认以测试类为单位验证；过大时可降级为单方法
- 命令选择：读取对应构建工具 reference（`gradle.md`/`maven.md`）
- Android/KMP 先读对应平台 reference 再落命令

## 失败判定

以下信号不能机械归因为业务缺陷：`NPE`、`ClassCastException`、`UninitializedPropertyAccessException`、断言失败。

优先检查：mock 返回值是否匹配签名、测试是否向非空参数传 `null`、source set/模块边界问题、框架初始化错误。

细节从 `troubleshooting.md` 获取。

## Reference Index

| Document             | Purpose                                 |
|----------------------|-----------------------------------------|
| `android.md`         | 本地 JVM 单测、variant、Main dispatcher 替换    |
| `kmp.md`             | source set、target task、`kotlin.test`    |
| `gradle.md`          | module path、测试命令、覆盖率                    |
| `maven.md`           | 模块定位、Surefire 过滤                        |
| `mockk.md`           | MockK API、协程 mock、object/static mock    |
| `mockito-kotlin.md`  | Mockito-Kotlin API、any() NPE、final mock |
| `troubleshooting.md` | 失败信号与修复策略                               |
