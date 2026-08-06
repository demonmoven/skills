# MockK Reference

## 初始化

JUnit 5：`@ExtendWith(MockKExtension::class)` + `@MockK` + `@InjectMockKs`

JUnit 4：`@get:Rule val mockkRule = MockKRule(this)` 或 `@Before` 中 `MockKAnnotations.init(this)`

清理：`mockkObject`/`mockkStatic` 必须在 `@AfterEach` 中 `unmockkAll()`，或使用作用域写法自动还原。

## 基本用法

```kotlin
@ExtendWith(MockKExtension::class)
class UserServiceTest {
  @MockK private lateinit var userDAO: UserDAO
  @InjectMockKs private lateinit var userService: UserService

  @Test
  fun testGetUser_正常获取用户_BitsUT() {
    every { userDAO.findById(1001L) } returns User(1001L, "张三")
    val result = userService.getUser(1001L)
    assertEquals("张三", result.name)
    verify { userDAO.findById(1001L) }
  }
}
```

## 协程

suspend 函数必须用 `coEvery`/`coAnswers`/`coVerify`，不要使用普通 `every`/`verify`。

```kotlin
@Test
fun testFetch_正常返回_BitsUT() = runTest {
  coEvery { apiClient.fetchData("key") } returns DataResult("value")
  val result = service.fetchData("key")
  assertEquals("value", result.data)
  coVerify { apiClient.fetchData("key") }
}
```

## Kotlin 特有

- Kotlin 类默认 `final`；出现 transform 失败时检查 MockK 版本/JDK/agent
- mock 顶层/扩展函数：`mockkStatic("<FacadeOrKtClass>")`
- mock `object`/`companion object`：`mockkObject(...)`

## Spy

- `spyk(RealClass())`：未 stub 方法走真实逻辑
- `@SpyK` 可作为依赖注入到 `@InjectMockKs`；被测对象不能同时标 `@SpyK` 和 `@InjectMockKs`

## Relaxed Mock

- `@MockK(relaxed = true)`：未配置调用返回默认值（0/空字符串/空集合）
- `relaxUnitFun = true`：仅放行 `Unit` 返回函数，更精确

## Argument Capture

```kotlin
val slot = slot<Long>()
every { userDAO.findById(capture(slot)) } returns User(1001L, "张三")
userService.getUser(1001L)
assertEquals(1001L, slot.captured)
```

多次捕获用 `mutableListOf<T>()` + `capture(list)`。`capture` 可与字面量或 `any()` 混用。

## 常见错误

- `no answer found`：未配置返回值或参数匹配不一致
- `Failed to transform`：检查 MockK 版本、JDK、依赖
- `@InjectMockKs` 失败：构造参数类型与 `@MockK` 字段不一致；必要时手动构造
