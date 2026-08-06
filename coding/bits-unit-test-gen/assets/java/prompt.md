# Java 语言专用提示词

## 函数提取方法

| 工具/方法   | 说明                                      |
|---------|-----------------------------------------|
| `grep`  | 快速定位方法签名 `public`/`protected`/`private` |
| IDE LSP | 精确解析类和方法结构                              |
| `javap` | 查看编译后的类方法签名                             |

## 过滤规则（Java 专用）

**跳过：**

- `public static void main` 入口方法
- 自动生成的代码：`*Generated*`、Proto 生成的文件
- Lombok 生成的 getter/setter/toString/equals/hashCode（标注了 `@Data`、`@Getter` 等注解的类）
- 少于 3 行的简单 getter/setter
- 接口中的 `default` 方法（除非包含复杂逻辑）

## 测试文件命名规范

| 项目   | 规范                                                      |
|------|---------------------------------------------------------|
| 测试文件 | `*Test.java`                                            |
| 位置   | 在 `src/test/java/` 下对应的包路径中                             |
| 包名   | 与被测类相同                                                  |
| 测试类  | `{ClassName}Test`（例如 `UserService` → `UserServiceTest`） |

---

## 处理粒度

Java 中的最小执行单元是**源码类**。将同一个类中选定的方法一起处理，以保持包设置、依赖注入、Mock 和测试类更新的一致性。

对于每个选定的类，跟踪其全限定类名、可选模块名、源文件路径、选定方法、起始行号以及重载方法的签名。在生成测试以及报告覆盖率、验证命令、失败信息和缺陷映射时直接使用这些信息。

---

## 执行调度

Java 以类为基本单位进行任务调度。调度策略：

1. **按类顺序处理**：逐个处理选定的类。
2. **Writer 按类生成**：Writer 处理类中选定的方法，生成一个测试类或追加到已有的测试类中。
3. **Fixer 按类验证**：Writer 完成后，调度 Fixer 运行特定的测试类，分类失败原因，并修复允许修复的测试问题。
4. **按类报告**：在最终响应中记录生成的测试文件、验证命令、状态、失败信息和缺陷映射。

### 验证修复轮次

Fixer 对每个测试类的验证修复循环最多限制为 **3 轮**。

---

## 预检查（编写测试前必须完成）

> **⚠️ 硬性前置条件**：在生成任何测试代码之前，本步骤的项目学习**必须**完成。不同项目在 DI 框架、Mock
> 策略和测试风格上差异很大。跳过此步骤几乎必然导致后续反复修复。

### 1. 环境检测

1. **检测构建工具**：检查项目使用的是 Maven（`pom.xml`）还是 Gradle（`build.gradle` / `build.gradle.kts`）
2. **检查是否为多模块项目**：查找多个 `pom.xml` 或 `build.gradle` 文件以确认是否为多模块项目
  - 在多模块项目中，编译和测试命令需要模块标识符（Maven 用 `-pl <module>`，Gradle 用 `:<module>`）
3. **检测 Java 版本**（推荐）：检查 `pom.xml` / `build.gradle` 中的 source/target 版本——影响可用的语法特性（var、records、sealed
   classes、text blocks 等）

### 2. 学习项目测试模式

学习目标类所在包（及相邻包）中已有测试的风格：

1. **扫描已有测试文件**（必须）：阅读对应测试目录中 1-2 个已有的 `*Test.java` 文件，学习：
  - **测试框架**：JUnit 5（`@Test` 来自 `org.junit.jupiter`）还是 JUnit 4（`@Test` 来自 `org.junit`）
  - **Mock 策略**：Mockito（`@Mock` + `@InjectMocks`）、PowerMock，还是手动 stub 类？
  - **断言风格**：JUnit 5 Assertions、AssertJ（`assertThat`），还是 Hamcrest matchers？
  - **DI 模式**：`@ExtendWith(MockitoExtension.class)`、`@SpringBootTest`，还是构造函数注入配合手动 Mock？
  - **命名规范**：测试方法的实际命名模式
  - **测试用例组织**：`@ParameterizedTest`、`@Nested` 类，还是扁平的 `@Test` 方法？
2. **发现测试辅助类/工厂类**（推荐）：搜索可复用的测试资产
   ```bash
   grep -rn "class.*TestBase\|class.*TestHelper\|class.*TestFactory\|@TestConfiguration" --include="*.java" <target_test_dir>
   ```
  - 如果存在 `TestBase`、`TestHelper`、`TestFactory`、`TestFixture` 类，优先复用
3. **阅读项目约定**（推荐）：检查 `PROJECT_ROOT` 下的 `AGENTS.md`、`CLAUDE.md`
  - 提取单元测试相关要求（命名规范、Mock 框架、目录结构等）

### 3. 上下文分析

对于每个目标方法，收集充分的上下文信息：

1. **第一层（必须）**：阅读目标方法源码，理解方法签名、参数/返回值类型定义、类级别依赖（`@Autowired` / 构造函数注入的字段）
2. **第二层（推荐）**：阅读注入依赖的接口定义（以确定 Mock 策略和返回类型）
3. **第三层（按需）**：当第二层信息不足时，阅读间接依赖、DTO/Entity 类定义或配置类

---

## Java 单元测试标准

### 测试方法签名

- 使用 JUnit 5 的 `@Test` 注解标记测试方法
- 方法必须为 `void` 返回类型、无参数、非 `static`
- 方法命名遵循 `test<Method>_<scenario>` 或 `should<Expected>_when<Condition>` 格式
- 推荐方法访问修饰符为 `package-private`（即不加修饰符），无需 `public`

### 测试类结构

- 测试类无需继承任何基类（JUnit 5）
- 使用 `@BeforeEach` / `@AfterEach` 替代 JUnit 4 的 `@Before` / `@After`
- 使用 `@BeforeAll` / `@AfterAll` 管理类级别资源（方法必须为 `static`）
- `@ExtendWith(MockitoExtension.class)` 启用 Mockito 注解支持

### 测试隔离原则

- 每个 `@Test` 方法必须独立，不依赖其他测试的执行顺序
- 禁止通过实例变量在测试方法间传递状态（除非在 `@BeforeEach` 中重新初始化）
- Mock 对象在每个测试方法执行前自动重置（Mockito 的默认行为）
- 禁止使用 `@TestMethodOrder` 强制测试顺序来满足依赖关系

### 断言标准

- 优先使用 JUnit 5 的 `Assertions`：`assertEquals(expected, actual)` —— 注意参数顺序：**期望值在前，实际值在后**
- 或使用 AssertJ 的流式断言：`assertThat(actual).isEqualTo(expected)`（跟随已有测试）
- 浮点数比较使用 `assertEquals(expected, actual, delta)` 或 `assertThat(actual).isCloseTo(expected, within(delta))`
- 集合断言使用 `assertThat(list).hasSize(3).contains("a", "b")`
- 异常断言使用 `assertThrows(ExceptionType.class, () -> { ... })`
- 优先使用精确值比较（`assertEquals`/`isEqualTo`），但在断言复杂对象的字段之前验证其非空时，`assertNotNull` 是合理的前置断言
- 复杂对象比较优先使用 `assertThat(actual).usingRecursiveComparison().isEqualTo(expected)`

### 异常测试要求

- 可能抛出异常的方法必须覆盖异常路径
- 使用 `assertThrows` 断言异常类型并验证异常消息：
  ```java
  Exception ex = assertThrows(IllegalArgumentException.class, () -> service.process(null));
  assertThat(ex.getMessage()).contains("不能为空");
  ```
- 禁止使用 JUnit 4 的 `@Test(expected = ...)` 语法

### null/空值处理

- 引用类型参数需要覆盖 `null` 输入场景
- `String` 参数需要覆盖空字符串 `""` 和 `null` 两种场景
- `List`/`Map`/`Set` 参数需要覆盖 `null` 和空集合 `Collections.emptyList()` 两种场景
- `Optional` 返回值需要覆盖 `Optional.empty()` 场景
- 数值参数需要覆盖 `0`、负数、边界值（如 `Integer.MAX_VALUE`）场景

### 访问控制

- 测试类与被测类在相同包路径下，可以访问 `package-private` 方法
- `private` 方法不直接测试；通过其公共方法间接覆盖
- 禁止使用反射绕过访问控制来测试 `private` 方法（极端情况除外）

---

## 验证方法

> 在预检查阶段确定构建工具（Maven 或 Gradle），然后使用下面对应的命令。

### Maven

#### local-run

```bash
mvn test -pl <module> -Dtest=<TestClass>
```

#### 运行测试

```bash
mvn test -pl <module> -Dtest=<TestClass>
```

#### 覆盖率检查

```bash
mvn test -pl <module> -Dtest=<TestClass> -Djacoco.skip=false
mvn jacoco:report -pl <module>
```

> 单模块项目省略 `-pl <module>`。

### Gradle

#### local-run

```bash
./gradlew :<module>:test --tests "<fully.qualified.TestClass>"
```

#### 运行测试

```bash
./gradlew :<module>:test --tests "<fully.qualified.TestClass>"
```

#### 覆盖率检查

```bash
./gradlew :<module>:test --tests "<fully.qualified.TestClass>" jacocoTestReport
```

> 单模块项目省略 `:<module>:` 前缀（使用 `:test` 等）。

---

## 特殊修复规则

**失败分类：**

> 完整的缺陷判定规则参见 `references/test-fixer/AGENT.md` 中的"失败分类流程"章节。此处仅列出 Java 专用的补充内容。

**Java 专用缺陷信号（必须结合上下文判断；不能直接判定为缺陷）：**

- `java.lang.NullPointerException`（非 Mock 注入问题）→ 可能缺少空值防护，**但仅当 null 源自方法内部逻辑时才算缺陷**；测试故意传入
  null 参数触发的不算
- `java.lang.ArrayIndexOutOfBoundsException` → 可能缺少数组边界检查，**但仅当输入来自正常业务场景时才算缺陷**
- `java.lang.StringIndexOutOfBoundsException` → 可能是字符串索引越界，**同上**
- `java.lang.ArithmeticException: / by zero` → 可能缺少除零防护，**大概率是真实缺陷**
- `java.lang.ClassCastException` → 可能缺少类型检查，**需确认正常流程中是否可能出现类型不匹配**
- `java.util.ConcurrentModificationException` → 可能是并发安全问题，**大概率是真实缺陷**
- `java.lang.StackOverflowError` → 可能是递归终止条件缺陷，**大概率是真实缺陷**
- 断言失败且期望值符合方法的正确语义 → 逻辑缺陷，**大概率是真实缺陷**

- 如果编译报 `cannot find symbol`，检查 import 语句以及依赖是否在 `pom.xml` / `build.gradle` 中声明
- 如果出现 `NullPointerException` 而非预期行为，检查 Mock 对象是否正确注入（`@InjectMocks` + `@Mock`）
- 如果 Mockito 报 `Unnecessary stubbings detected`，使用 `@MockitoSettings(strictness = Strictness.LENIENT)` 或移除未使用的
  stub
- 如果出现 `org.mockito.exceptions.misusing.MissingMethodInvocationException`，检查是否在 Mock final 类/方法（需要
  `mockito-inline`）
- 如果 Spring 上下文加载失败，检查是否缺少必要的 `@MockBean` 声明

---

## 格式化

遵循项目的格式化工具配置。

如果项目没有统一的格式化工具，推荐使用 Google Java Format：

```bash
google-java-format -i <file>
```

---

## 代码风格

风格由优先级决定：用户指令 > AGENTS.md > 同目录已有测试 > 以下默认值。

| 项目        | 规范                                                     |
|-----------|--------------------------------------------------------|
| 注释语言      | 中文                                                     |
| 命名        | `test<Method>_<scenario>`（JUnit 5 `@Test`，English only）|
| 文件        | `<ClassName>Test.java`，位于 `src/test/java/` 对应包路径下      |
| 断言        | `org.junit.jupiter.api.Assertions` 或 `AssertJ`（跟随已有测试） |
| Mock      | `Mockito`（或跟随已有测试）                                     |
| 测试用例组织    | `@ParameterizedTest` + `@MethodSource` / `@CsvSource`  |
| import 顺序 | 静态导入 → 标准库 → 第三方 → 项目内部包，用空行分隔                         |

---

## Mockito 用法

### 基本结构

使用 `@ExtendWith` + `@Mock` + `@InjectMocks` 组合：

```java

@ExtendWith(MockitoExtension.class)
class UserServiceTest {

  @Mock
  private UserDAO userDAO;

  @Mock
  private CacheClient cacheClient;

  @InjectMocks
  private UserService userService;

  @Test
  void testGetUser_normalFetch() {
    // 正常获取用户
    // 1. 设置 mock
    // 2. 调用目标方法
    // 3. 断言结果
    // 4. 验证调用（可选）
  }
}
```

### Mock 方法返回值

```java

@Test
void testGetUser_normalFetch() {
  // 正常获取用户
  User expectedUser = new User(1001L, "张三", "zhangsan@example.com");
  when(userDAO.findById(1001L)).thenReturn(Optional.of(expectedUser));

  User result = userService.getUser(1001L);

  assertEquals("张三", result.getName());
  assertEquals("zhangsan@example.com", result.getEmail());
  verify(userDAO).findById(1001L);
}

@Test
void testGetUser_notFoundThrowsException() {
  // 用户不存在抛出异常
  when(userDAO.findById(999L)).thenReturn(Optional.empty());

  Exception ex = assertThrows(UserNotFoundException.class,
    () -> userService.getUser(999L));

  assertThat(ex.getMessage()).contains("用户不存在");
}
```

### Mock 方法抛出异常

```java

@Test
void testGetUser_dbQueryFailed() {
  // 数据库查询失败
  when(userDAO.findById(anyLong()))
    .thenThrow(new RuntimeException("connection refused"));

  assertThrows(ServiceException.class,
    () -> userService.getUser(1001L));
}
```

### 条件 Mock（根据参数返回不同结果）

```java

@Test
void testBatchGetUsers_differentIdsReturnDifferentResults() {
  // 不同ID返回不同结果
  when(userDAO.findById(1L)).thenReturn(Optional.of(new User(1L, "UserA")));
  when(userDAO.findById(2L)).thenReturn(Optional.of(new User(2L, "UserB")));
  when(userDAO.findById(999L)).thenReturn(Optional.empty());

  List<User> users = userService.batchGetUsers(List.of(1L, 2L, 999L));

  assertThat(users).hasSize(2);
  assertThat(users).extracting(User::getName).containsExactly("UserA", "UserB");
}
```

### Mock void 方法

```java

@Test
void testDeleteUser_normalDelete() {
  // 正常删除
  doNothing().when(userDAO).deleteById(1001L);

  userService.deleteUser(1001L);

  verify(userDAO).deleteById(1001L);
}

@Test
void testDeleteUser_deleteFailedThrowsException() {
  // 删除失败抛出异常
  doThrow(new RuntimeException("delete failed"))
    .when(userDAO).deleteById(anyLong());

  assertThrows(ServiceException.class,
    () -> userService.deleteUser(1001L));
}
```

### Mock 静态方法

```java

@Test
void testGenerateOrderNo_normalGeneration() {
  // 正常生成订单号
  try (MockedStatic<LocalDateTime> mockedTime = mockStatic(LocalDateTime.class)) {
    LocalDateTime fixedTime = LocalDateTime.of(2024, 1, 15, 10, 30, 0);
    mockedTime.when(LocalDateTime::now).thenReturn(fixedTime);

    String orderNo = orderService.generateOrderNo();

    assertThat(orderNo).startsWith("20240115");
  }
}
```

> `mockStatic` 必须在 `try-with-resources` 中使用以确保恢复。静态 Mock 泄漏到其他测试会导致级联失败。

### Mock 反模式（禁止）

- ❌ Mock 简单工具方法（如 `String.format`、`Collections.sort`）→ ✅ 直接调用，无需 Mock
- ❌ 通过反射 Mock 被测类的私有方法 → ✅ 通过公共方法间接覆盖
- ❌ Mock 所有依赖使测试变成"验证调用顺序" → ✅ 仅 Mock 不可控的外部依赖（DB/RPC/HTTP）
- ❌ 单元测试使用 `@SpringBootTest` → ✅ 使用 `@ExtendWith(MockitoExtension.class)` 进行轻量级单元测试
- ❌ Mock 返回值类型与真实签名不匹配 → ✅ Mock 返回值必须与方法签名完全匹配
- ❌ Mock `equals`/`hashCode`/`toString` → ✅ 这些方法应使用真实实现

---

## 示例

### 示例 1：参数化测试（推荐模式）

目标方法：

```java
public int add(int a, int b) {
  return a + b;
}
```

测试代码：

```java

@ParameterizedTest(name = "{0}: add({1}, {2}) = {3}")
@MethodSource("addTestCases")
void testAdd_parameterized(String name, int a, int b, int expected) {
  assertEquals(expected, calculator.add(a, b));
}

static Stream<Arguments> addTestCases() {
  return Stream.of(
    Arguments.of("two positive numbers", 1, 2, 3),
    Arguments.of("positive plus negative", 5, -3, 2),
    Arguments.of("two negative numbers", -1, -2, -3),
    Arguments.of("add zero", 10, 0, 10),
    Arguments.of("both zero", 0, 0, 0),
    Arguments.of("large number addition", Integer.MAX_VALUE - 1, 1, Integer.MAX_VALUE)
  );
}
```

### 示例 2：@Nested 类组织（一个方法的多种场景）

目标方法：

```java
public User getUser(Long id) {
  if (id == null || id <= 0) {
    throw new IllegalArgumentException("用户ID不合法");
  }
  return userDAO.findById(id)
    .orElseThrow(() -> new UserNotFoundException("用户不存在: " + id));
}
```

测试代码：

```java

@ExtendWith(MockitoExtension.class)
class UserServiceTest {

  @Mock
  private UserDAO userDAO;

  @InjectMocks
  private UserService userService;

  @Nested
  class GetUser {

    @Test
    void normalFetch() {
      // 正常获取用户
      User expected = new User(1001L, "张三");
      when(userDAO.findById(1001L)).thenReturn(Optional.of(expected));

      User result = userService.getUser(1001L);

      assertEquals("张三", result.getName());
      verify(userDAO).findById(1001L);
    }

    @Test
    void notFoundThrowsException() {
      // 用户不存在时抛出异常
      when(userDAO.findById(999L)).thenReturn(Optional.empty());

      assertThrows(UserNotFoundException.class,
        () -> userService.getUser(999L));
    }

    @Test
    void nullIdThrowsIllegalArgument() {
      // ID为null时抛出参数异常
      assertThrows(IllegalArgumentException.class,
        () -> userService.getUser(null));
    }

    @Test
    void negativeIdThrowsIllegalArgument() {
      // ID为负数时抛出参数异常
      assertThrows(IllegalArgumentException.class,
        () -> userService.getUser(-1L));
    }
  }
}
```

---

## 常见陷阱与修复

| 陷阱                               | 原因                                               | 修复方法                                                            |
|----------------------------------|--------------------------------------------------|-----------------------------------------------------------------|
| `Unnecessary stubbings detected` | 设置了 stub 但测试中未实际调用                               | 移除未使用的 stub，或使用 `@MockitoSettings(strictness = LENIENT)`        |
| `Cannot mock final class/method` | Mockito 默认不支持 final 类/方法                         | 添加 `mockito-inline` 依赖（Mockito 5+ 默认支持）                         |
| `@InjectMocks` 注入失败              | 被测类的构造函数参数与 `@Mock` 字段类型不匹配                      | 手动构造被测对象，通过构造函数传入 Mock 对象                                       |
| Mock 对象上的 `NullPointerException` | Mock 方法未设置返回值 stub，默认返回 null                     | 为所有被调用的 Mock 方法设置返回值                                            |
| 浮点数比较 `assertEquals` 失败          | 浮点精度问题                                           | 使用三参数版本 `assertEquals(expected, actual, delta)`                 |
| 自定义对象比较 `assertEquals` 失败        | 对象未重写 `equals`/`hashCode`                        | 使用 AssertJ 的 `usingRecursiveComparison()` 或逐字段断言                |
| 静态方法无法 Mock                      | Mockito 要求 `mockStatic` 在 try-with-resources 中使用 | 使用 `try (MockedStatic<T> mocked = mockStatic(T.class)) { ... }` |
| Spring 集成测试上下文加载缓慢               | `@SpringBootTest` 启动完整上下文                        | 单元测试改用 `@ExtendWith(MockitoExtension.class)`                    |

---

## 上下文发现命令

```bash
grep -rn "public.*class\|public.*interface" <package_path>
cat pom.xml build.gradle 2>/dev/null
find . -name "*Test.java" | head -20
grep -rn "@Mock\|@InjectMocks\|@MockBean" --include="*Test.java" .
grep -rn "import.*assert\|import.*Mock" --include="*Test.java" . | head -10
```
