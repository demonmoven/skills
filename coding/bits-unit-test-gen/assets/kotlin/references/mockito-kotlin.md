# Mockito-Kotlin Reference

## 初始化

`@ExtendWith(MockitoExtension::class)` + `@Mock` + `@InjectMocks`（或手动构造被测对象避免注入歧义）

## 基本用法

```kotlin
@ExtendWith(MockitoExtension::class)
class UserServiceTest {
  @Mock private lateinit var userDAO: UserDAO
  @InjectMocks private lateinit var userService: UserService

  @Test
  fun testGetUser_正常获取用户_BitsUT() {
    whenever(userDAO.findById(1001L)).thenReturn(User(1001L, "张三"))
    val result = userService.getUser(1001L)
    assertEquals("张三", result.name)
    verify(userDAO).findById(1001L)
  }
}
```

## any() 与 Kotlin 非空参数（关键）

Mockito 的 `any()` 内部返回 `null`，传给 Kotlin 非空参数触发 NPE。

- **必须使用 `org.mockito.kotlin.any()`**（reified 版本）
- nullable 参数用 `anyOrNull<T>()`
- `eq()` 同理：使用 `org.mockito.kotlin.eq()`
- 任一参数使用 matcher 时，所有参数都必须用 matcher

```kotlin
// ❌ org.mockito.any() → null → NPE
// ❌ matcher 与字面量混用 → InvalidUseOfMatchersException

// ✅
whenever(dao.save(any<User>(), any())).thenReturn(true)
whenever(dao.save(any(), eq(3))).thenReturn(true)
```

`NPE` 出现在 `whenever(...)` 行本身 → 几乎必然是 `any()` import 错误。

## Spy

- stub spy **必须用 `doReturn(...).whenever(spy).method()`**
- 不要用 `whenever(spy.method()).thenReturn(...)` — 会先执行真实方法

## 协程

suspend 函数直接用 `whenever(...).thenReturn(...)` 在 `runTest` 中（Mockito 4.x+ 原生支持 suspend）。

```kotlin
@Test
fun testFetchData_正常返回_BitsUT() = runTest {
  whenever(apiClient.fetchData("key")).thenReturn(DataResult("value"))
  val result = service.fetchData("key")
  assertEquals("value", result.data)
}
```

## Final Class Mock

Kotlin 类默认 `final`，需 inline mock maker：
- 检查 `src/test/resources/mockito-extensions/org.mockito.plugins.MockMaker`（内容 `mock-maker-inline`）
- 或 `mockito-inline` / `mockito-kotlin` 5.x+
- 出现 `Cannot mock/spy ... final class` 时检查上述配置，不要将被测类改为 `open`

## 常见错误

- `NPE`：检查 `any()`/`eq()` import 及 nullable/non-nullable 匹配
- `Wanted but not invoked`：参数匹配器与真实调用不一致
- `Cannot mock/spy`：final/inline class/object，检查 mock maker 配置
