# Android Kotlin Reference

## 适用范围

默认只处理本地 JVM 单测（`src/test/kotlin`）；用户显式要求才进入 instrumented test（`src/androidTest/kotlin`）。Android 项目中的纯 JVM module 按普通 Kotlin/JVM + `gradle.md` 处理。

## 任务选择

- 本地 JVM 单测：`test<Variant>UnitTest`（如 `testDebugUnitTest`）
- Instrumented test：`connected<Variant>AndroidTest`
- 有 flavor 时 variant = `flavor + buildType`（如 `testQaDebugUnitTest`）
- 无法确定 variant 时查看 CI/README 或 `./gradlew tasks`

## 测试过滤

- 本地 JVM 单测支持 `--tests` 过滤
- Instrumented test 不支持 `--tests` 时退回 variant 任务级验证

## 平台特有陷阱与解法

### Main Dispatcher 替换

`Dispatchers.Main` 在本地 JVM 测试中不存在，必须替换。优先复用项目已有 rule/extension，无参考时使用以下模板：

JUnit 4：

```kotlin
@OptIn(ExperimentalCoroutinesApi::class)
class MainDispatcherRule(
  private val dispatcher: TestDispatcher = UnconfinedTestDispatcher()
) : TestWatcher() {
  override fun starting(description: Description) { Dispatchers.setMain(dispatcher) }
  override fun finished(description: Description) { Dispatchers.resetMain() }
}
// 使用：@get:Rule val mainDispatcherRule = MainDispatcherRule()
```

JUnit 5：

```kotlin
@OptIn(ExperimentalCoroutinesApi::class)
class MainDispatcherExtension(
  private val dispatcher: TestDispatcher = UnconfinedTestDispatcher()
) : BeforeEachCallback, AfterEachCallback {
  override fun beforeEach(context: ExtensionContext?) { Dispatchers.setMain(dispatcher) }
  override fun afterEach(context: ExtensionContext?) { Dispatchers.resetMain() }
}
// 使用：@ExtendWith(MainDispatcherExtension::class)
```

## 注意事项

- Robolectric、Truth、AndroidX Test 仅在项目已有时沿用
- 依赖 `Dispatchers.Main` 时优先参考邻近测试的 dispatcher 替换方式
