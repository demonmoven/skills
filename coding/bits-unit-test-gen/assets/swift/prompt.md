# Swift 语言专属 Prompt

本文件内容包括：

- Swift 目标提取与处理粒度
- Swift 语义特有的过滤规则
- 测试文件组织的默认基线
- Mock 策略与协议注入
- 异步/并发测试约束
- 其他必要单测知识

优先级：用户显式指令 > `AGENTS.md` / `CLAUDE.md` > 目标目录已有测试风格 > 本文件默认规则。

***

## 工作单元与调度策略

- 目标选择：函数/方法
- 生成与修复：声明容器（class/struct/enum/actor → 该类型；顶层函数/扩展 → 该文件）
- 验证：测试类（XCTestCase 子类）或测试结构体（Swift Testing `@Suite`）
- 调度：按声明容器串行

同一声明容器内的多个目标函数一起生成，统一维护 import、fixture、mock。当前声明容器收敛后再进入下一个。

记录信息：`container_kind`（class/struct/enum/actor/extension/file）、`container_name`（类型为完全限定名，文件级为路径）、所属 target/module、源文件路径、目标函数及起始行、测试文件、测试命令和工作目录。

***

## Swift 目标提取

- 顶层函数、扩展方法、`static`/`class` 方法均可作为独立测试目标
- 重载函数必须结合参数列表和起始行区分
- 计算属性（`var ... { get { } }`）若含有分支逻辑或依赖调用，可作为测试目标
- `actor` 的方法需标记 `is_async`（actor 隔离导致所有外部调用都为 async）
- 额外记录：`is_async`、`is_throwing`、`access_level`

***

## Swift 目标过滤

跳过以下声明：

- `@main` 入口结构体/类的 `static func main()`
- 自动合成方法（`Codable` 的 `init(from:)`/`encode(to:)`、`Equatable` 的 `==`、`Hashable` 的 `hash(into:)`）
- `enum` 的 `allCases`（`CaseIterable` 自动合成）
- 行数 < 3 且无分支的简单存储属性访问
- `@resultBuilder` 的 `buildBlock`/`buildOptional` 等方法
- 自动生成代码（Proto/Thrift 生成文件、`*.pb.swift`、`*.grpc.swift`）
- `@objc dynamic` 的纯桥接 wrapper（仅转发到 Swift 实现）
- SwiftUI `body` 属性（除非用户显式要求）
- `PreviewProvider` / `#Preview` 相关代码

***

## 预检查（编写测试前必须完成）

> **⚠️ 硬性前置条件**：在生成任何测试代码之前，本步骤的项目学习**必须**完成。Swift 项目在构建系统、测试框架和 mock 策略上差异很大。跳过此步骤几乎必然导致后续反复修复。

### 1. 环境检测

1. **检测构建系统**：检查 `Package.swift`（SPM）或 `*.xcodeproj`/`*.xcworkspace`（Xcode）
2. **检测测试框架**：
   - `import XCTest` → XCTest（传统框架）
   - `import Testing` → Swift Testing（Swift 5.9+/Xcode 16+）
3. **检测 Swift 版本**：检查 `Package.swift` 中的 `swift-tools-version` 或项目的 `SWIFT_VERSION` 设置
4. **检测平台**：iOS/macOS/tvOS/watchOS/Linux — 影响可用 API 和测试执行方式
5. **检测依赖管理**：SPM、CocoaPods（`Podfile`）、Carthage（`Cartfile`）
6. **确认测试 target**：SPM 项目检查 `Package.swift` 中 `.testTarget`；Xcode 项目检查测试 target 配置

### 2. 学习项目测试模式

学习目标文件所在模块中已有测试的风格：

1. **扫描已有测试文件**（必须）：阅读 1-2 个已有 `*Tests.swift` 或 `*Test.swift` 文件，学习：
   - **测试框架**：XCTest（`class ... : XCTestCase`）还是 Swift Testing（`@Test`、`@Suite`）？
   - **Mock 策略**：协议注入 + 手写 Mock、Mockingbird、Cuckoo，还是无 Mock？
   - **断言风格**：`XCTAssertEqual`、`#expect`（Swift Testing）、还是第三方库如 Nimble？
   - **命名规范**：测试方法的实际命名模式
   - **异步测试**：`async throws` 测试方法还是 `XCTestExpectation`？
2. **发现测试辅助工具**（推荐）：搜索可复用测试资产
   ```bash
   grep -rn "class.*TestCase\|protocol.*Mock\|struct.*Mock\|class.*Mock\|func make.*\|factory" --include="*.swift" <test_dir>
   ```
   - 如果存在 `TestHelper`、`MockFactory`、`XCTestCase` 基类扩展，优先复用
3. **阅读项目约定**（推荐）：检查 `PROJECT_ROOT` 下的 `AGENTS.md`、`CLAUDE.md`

### 3. 上下文分析

对于每个目标方法，收集充分的上下文信息：

1. **第一层（必须）**：阅读目标方法源码，理解方法签名、参数/返回值类型、依赖（通过构造函数注入或属性注入的协议）
2. **第二层（推荐）**：阅读依赖协议定义（确定 Mock 策略和返回类型）
3. **第三层（按需）**：当第二层信息不足时，阅读间接依赖、Model/DTO 定义或配置

***

## 默认风格

| 项目      | 默认约定                                                                      |
| ------- | ------------------------------------------------------------------------- |
| 注释语言   | 中文                                                                                          |
| 命名      | `test{Method}_{scenarioDesc}_BitsUT`（XCTest）或 `{method}_{scenarioDesc}_BitsUT`（Swift Testing） |
| 文件      | `<TypeName>Tests.swift`，放在对应测试 target 下，优先追加既有测试文件                        |
| 断言      | 沿用项目已有；无参考时 XCTest 用 `XCTAssert*`，Swift Testing 用 `#expect`               |
| Mock    | 沿用项目已有；无参考时使用协议注入 + 手写 Mock 类                                             |
| 异步      | `async throws` 原生支持（XCTest/Swift Testing 均支持）                             |
| Import  | 系统框架 → 第三方 → 项目内部，`@testable import` 单独一行                                 |
| Access  | 使用 `@testable import` 访问 `internal` 成员                                    |

### 代码语言硬约束

> **禁止在生成的代码中除注释以外的任何位置使用中文。** 方法名、变量名、字符串字面量（含 `@Test`/`@Suite` 的 display name）、枚举 case 名等均必须使用英文。注释（`//`、`/* */`、`/// doc comment`）允许使用中文。

***

## 测试文件命名规范

| 项目    | 规范                                                |
| ----- | ------------------------------------------------- |
| 测试文件  | `<TypeName>Tests.swift`                           |
| 位置    | SPM: `Tests/<TestTarget>/`；Xcode: 对应测试 target 目录下 |
| 测试类   | `final class <TypeName>Tests: XCTestCase`（XCTest） |
| 测试结构体 | `@Suite struct <TypeName>Tests`（Swift Testing）    |

***

## Mock 策略

### 协议注入（推荐默认方式）

Swift 没有 Java/Kotlin 那样通用的 Mock 框架生态。推荐方式：

1. 被测代码通过协议声明依赖
2. 测试中创建协议的 Mock 实现
3. 通过构造函数注入 Mock 对象

```swift
// 协议定义
protocol UserRepository {
    func findById(_ id: Int) async throws -> User?
}

// Mock 实现
final class MockUserRepository: UserRepository {
    var findByIdResult: User?
    var findByIdError: Error?
    var findByIdCallCount = 0
    var findByIdReceivedId: Int?

    func findById(_ id: Int) async throws -> User? {
        findByIdCallCount += 1
        findByIdReceivedId = id
        if let error = findByIdError { throw error }
        return findByIdResult
    }
}
```

### Mock 通用约束

- 只 mock 不可控外部依赖（网络、数据库、文件系统、时间、UserDefaults）
- `struct` 是值类型，无法被 mock；使用真实实例或协议抽象
- `final class` 无法被子类化 mock；必须通过协议注入
- `actor` 方法从外部调用时自动 async；mock actor 时需要注意隔离域
- 不要 mock 标准库纯函数和值类型操作
- 不要 mock 被测函数本身

### Mock 反模式（禁止）

- ❌ Mock 简单值类型操作（如 `Array.map`、`String.count`）→ ✅ 直接使用真实值
- ❌ 通过 runtime 黑魔法 mock `private` 方法 → ✅ 通过公共方法间接覆盖
- ❌ Mock 所有依赖使测试变成"验证调用顺序" → ✅ 仅 mock 不可控的外部依赖
- ❌ 使用 `@objc` + method swizzling 进行 mock → ✅ 使用协议注入
- ❌ Mock 返回值类型与协议签名不匹配 → ✅ Mock 返回值必须严格符合协议定义

***

## 异步/并发测试

### async/await 测试

XCTest 和 Swift Testing 均原生支持 `async throws` 测试方法：

```swift
// XCTest
func testFetchUser_fetchSuccess_BitsUT() async throws {
    let mockRepo = MockUserRepository()
    mockRepo.findByIdResult = User(id: 1, name: "John")
    let service = UserService(repository: mockRepo)

    let user = try await service.fetchUser(id: 1)

    XCTAssertEqual(user.name, "John")
    XCTAssertEqual(mockRepo.findByIdCallCount, 1)
}
```

```swift
// Swift Testing
@Test func fetchUser_fetchSuccess_BitsUT() async throws {
    let mockRepo = MockUserRepository()
    mockRepo.findByIdResult = User(id: 1, name: "John")
    let service = UserService(repository: mockRepo)

    let user = try await service.fetchUser(id: 1)

    #expect(user.name == "John")
    #expect(mockRepo.findByIdCallCount == 1)
}
```

### Actor 测试

- `actor` 的所有方法从外部调用均需 `await`
- 测试 actor 内部状态需要通过 actor 的公共方法或属性访问
- 不要尝试绕过 actor 隔离

### Combine/异步流测试

- `Publisher` 使用 `XCTestExpectation` + `sink` 或 `values` 属性收集
- `AsyncSequence` 使用 `for await` + `Task` 并设置超时
- 永不终止的流需要用 `prefix(_:)` 或 `Task.cancel()` 限制

***

## XCTest 标准

### 测试方法签名

- 方法必须以 `test` 开头，无参数，返回 `Void`
- 可选 `throws` 和 `async`：`func testXxx() async throws`
- 测试类必须继承 `XCTestCase`，推荐 `final class`

### 生命周期

- `setUp()` / `setUpWithError()` — 每个测试方法执行前调用
- `tearDown()` / `tearDownWithError()` — 每个测试方法执行后调用
- `override class func setUp()` — 测试类初始化（一次）
- Xcode 13.2+（Swift 5.5+）支持 `setUp() async throws`

### 断言

| 用途     | API                                               |
| ------ | ------------------------------------------------- |
| 相等     | `XCTAssertEqual(actual, expected)`                |
| 不相等    | `XCTAssertNotEqual(actual, expected)`             |
| 布尔     | `XCTAssertTrue(expr)` / `XCTAssertFalse(expr)`    |
| nil 检查 | `XCTAssertNil(expr)` / `XCTAssertNotNil(expr)`    |
| 异常     | `XCTAssertThrowsError(try expr) { error in ... }` |
| 无异常    | `XCTAssertNoThrow(try expr)`                      |
| 浮点近似   | `XCTAssertEqual(a, b, accuracy: 0.001)`           |
| 大于/小于  | `XCTAssertGreaterThan(a, b)`                      |

***

## Swift Testing 标准（Swift 5.9+（包引入）/ Xcode 16+（内置））

### 基本结构

```swift
import Testing

@Suite("UserService Tests")
struct UserServiceTests {
    let mockRepo = MockUserRepository()
    let svc: UserService

    init() {
        svc = UserService(repository: mockRepo)
    }

    @Test("fetch user success")
    func fetchUser_fetchSuccess_BitsUT() async throws {
        mockRepo.findByIdResult = User(id: 1, name: "John")
        let user = try await svc.fetchUser(id: 1)
        #expect(user.name == "John")
    }

    @Test("throws error when user not found")
    func fetchUser_notFound_BitsUT() async {
        mockRepo.findByIdResult = nil
        await #expect(throws: UserError.self) {
            try await svc.fetchUser(id: 999)
        }
    }
}
```

### 断言宏

| 用途   | API                                            |
| ---- | ---------------------------------------------- |
| 通用断言 | `#expect(condition)`                           |
| 相等   | `#expect(actual == expected)`                  |
| 异常   | `#expect(throws: ErrorType.self) { try expr }` |
| 无异常  | `#expect(throws: Never.self) { try expr }`     |
| 不可达      | `Issue.record("should not reach here")`                |

### 参数化测试

```swift
@Test("addition", arguments: [
    (1, 2, 3),
    (5, -3, 2),
    (-1, -2, -3),
    (0, 0, 0),
])
func testAdd_BitsUT(a: Int, b: Int, expected: Int) {
    #expect(add(a, b) == expected)
}
```

***

## 验证方法

> 在预检查阶段确定构建系统（SPM 或 Xcode），然后使用对应命令。

### Swift Package Manager

#### 运行测试

```bash
swift test --filter <TestTarget>.<TestClass>
```

#### 运行单个测试方法

```bash
swift test --filter <TestTarget>.<TestClass>/test<MethodName>
```

#### 覆盖率检查

```bash
swift test --enable-code-coverage --filter <TestTarget>.<TestClass>
# macOS
llvm-cov report .build/debug/<PackageName>PackageTests.xctest/Contents/MacOS/<PackageName>PackageTests -instr-profile=.build/debug/codecov/default.profdata
# Linux
llvm-cov report .build/debug/<PackageName>PackageTests -instr-profile=.build/debug/codecov/default.profdata
```

### Xcode (xcodebuild)

#### 运行测试

```bash
xcodebuild test -scheme <Scheme> -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:<TestTarget>/<TestClass>
```

#### 覆盖率检查

```bash
xcodebuild test -scheme <Scheme> -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:<TestTarget>/<TestClass> -enableCodeCoverage YES
xcrun xccov view --report <DerivedData>/Logs/Test/*.xcresult
```

***

## 特殊修复规则

**失败分类：**

> 完整的缺陷判定规则参见 `references/test-fixer/AGENT.md` 中的"失败分类流程"章节。此处仅列出 Swift 专用的补充内容。

**Swift 专用缺陷信号（必须结合上下文判断；不能直接判定为缺陷）：**

- `Fatal error: Unexpectedly found nil while unwrapping an Optional value` → 可能缺少 Optional 安全解包，**但仅当 nil 源自方法内部逻辑时才算缺陷**；测试故意传入导致 nil 的参数不算
- `Fatal error: Index out of range` → 可能缺少数组边界检查，**但仅当输入来自正常业务场景时才算缺陷**
- `Fatal error: Division by zero` → 可能缺少除零防护，**大概率是真实缺陷**
- 断言失败且期望值符合方法的正确语义 → 逻辑缺陷，**大概率是真实缺陷**
- `EXC_BAD_ACCESS` → 内存访问错误，**大概率是真实缺陷**

**常见测试问题修复方向：**

- 如果编译报 `cannot find type/value in scope`，检查 `@testable import` 是否正确、目标 access level 是否足够
- 如果报 `Module 'XXX' has no member named 'YYY'`，检查被测代码是否在正确的 target 中、是否标记了正确的 access level
- 如果 Mock 对象调用报 `actor-isolated` 错误，需要在 Mock 方法上标记 `nonisolated` 或使用 `await`
- 如果 `async` 测试报 `Expression is 'async' but is not marked with 'await'`，确保测试方法声明为 `async`
- 如果 `XCTAssertThrowsError` 未触发，检查被测方法签名是否为 `throws` — 非 `throws` 方法无法抛出错误
- 如果 SPM 测试报 `no tests found`，确认测试 target 在 `Package.swift` 中正确声明且依赖了被测 target

***

## Optional 处理测试

Swift 特有的 Optional 类型要求额外关注：

- 返回 `Optional` 的方法必须覆盖 `nil` 和有值两种场景
- 强制解包 `!` 的代码路径需要覆盖 nil 场景验证是否会 crash
- `guard let` / `if let` 分支需要覆盖 else 分支
- `Optional` 链式调用 `a?.b?.c` 中任意节点为 nil 的场景

***

## 访问控制

- 使用 `@testable import ModuleName` 可访问 `internal` 成员
- `private` 和 `fileprivate` 方法不直接测试；通过公共/internal 方法间接覆盖
- `@testable import` 仅在 debug 构建下生效；确保测试配置正确
- 禁止修改被测代码的 access level 仅为了测试

***

## 格式化

```bash
swift-format format -i <file>
```

或使用 SwiftLint：

```bash
swiftlint lint --fix <file>
```

遵循项目已有格式化工具配置。

***

## 代码风格

风格由优先级决定：用户指令 > AGENTS.md > 同目录已有测试 > 以下默认值。

| 项目        | 规范                                                    |
| --------- | ----------------------------------------------------- |
| 注释语言      | 中文                                                    |
| 命名        | `test{Method}_{scenarioDesc}_BitsUT`（XCTest）           |
| 文件        | `<TypeName>Tests.swift`，放在对应测试 target 下               |
| 断言        | `XCTAssert*`（XCTest）或 `#expect`（Swift Testing），跟随已有测试 |
| Mock      | 协议注入 + 手写 Mock（或跟随已有测试）                               |
| 测试用例组织    | `// MARK: -` 分组或 Swift Testing `@Suite` 嵌套            |
| import 顺序 | 系统框架 → 第三方 → `@testable import` 项目模块，空行分隔             |

***

## 示例

### 示例 1：XCTest 基本结构

目标方法：

```swift
class Calculator {
    func divide(_ a: Double, by b: Double) throws -> Double {
        guard b != 0 else { throw CalculatorError.divisionByZero }
        return a / b
    }
}
```

测试代码：

```swift
import XCTest
@testable import MyApp

final class CalculatorTests: XCTestCase {
    private var sut: Calculator!

    override func setUp() {
        super.setUp()
        sut = Calculator()
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func testDivide_normalDivision_BitsUT() throws {
        let result = try sut.divide(10, by: 2)
        XCTAssertEqual(result, 5.0, accuracy: 0.001)
    }

    func testDivide_divideByZeroThrowsError_BitsUT() {
        XCTAssertThrowsError(try sut.divide(10, by: 0)) { error in
            XCTAssertEqual(error as? CalculatorError, .divisionByZero)
        }
    }

    func testDivide_negativeDivision_BitsUT() throws {
        let result = try sut.divide(-10, by: 2)
        XCTAssertEqual(result, -5.0, accuracy: 0.001)
    }

    func testDivide_zeroDividedByNonZero_BitsUT() throws {
        let result = try sut.divide(0, by: 5)
        XCTAssertEqual(result, 0.0, accuracy: 0.001)
    }
}
```

### 示例 2：协议注入 + Mock

目标方法：

```swift
protocol UserRepository {
    func findById(_ id: Int) async throws -> User?
}

class UserService {
    private let repository: UserRepository

    init(repository: UserRepository) {
        self.repository = repository
    }

    func getUser(id: Int) async throws -> User {
        guard id > 0 else { throw UserError.invalidId }
        guard let user = try await repository.findById(id) else {
            throw UserError.notFound(id: id)
        }
        return user
    }
}
```

测试代码：

```swift
import XCTest
@testable import MyApp

final class MockUserRepository: UserRepository {
    var findByIdResult: User?
    var findByIdError: Error?
    var findByIdCallCount = 0
    var findByIdReceivedId: Int?

    func findById(_ id: Int) async throws -> User? {
        findByIdCallCount += 1
        findByIdReceivedId = id
        if let error = findByIdError { throw error }
        return findByIdResult
    }
}

final class UserServiceTests: XCTestCase {
    private var mockRepo: MockUserRepository!
    private var sut: UserService!

    override func setUp() {
        super.setUp()
        mockRepo = MockUserRepository()
        sut = UserService(repository: mockRepo)
    }

    func testGetUser_fetchSuccess_BitsUT() async throws {
        mockRepo.findByIdResult = User(id: 1, name: "John")

        let user = try await sut.getUser(id: 1)

        XCTAssertEqual(user.name, "John")
        XCTAssertEqual(mockRepo.findByIdCallCount, 1)
        XCTAssertEqual(mockRepo.findByIdReceivedId, 1)
    }

    func testGetUser_userNotFoundThrowsError_BitsUT() async {
        mockRepo.findByIdResult = nil

        do {
            _ = try await sut.getUser(id: 999)
            XCTFail("Should throw an error")
        } catch {
            XCTAssertEqual(error as? UserError, .notFound(id: 999))
        }
    }

    func testGetUser_invalidIdThrowsError_BitsUT() async {
        do {
            _ = try await sut.getUser(id: -1)
            XCTFail("Should throw an error")
        } catch {
            XCTAssertEqual(error as? UserError, .invalidId)
        }
    }

    func testGetUser_repoFailureThrowsError_BitsUT() async {
        mockRepo.findByIdError = NSError(domain: "DB", code: -1)

        do {
            _ = try await sut.getUser(id: 1)
            XCTFail("Should throw an error")
        } catch {
            XCTAssertFalse(error is UserError)
        }
    }
}
```

### 示例 3：Swift Testing 参数化测试

```swift
import Testing
@testable import MyApp

@Suite("Calculator Tests")
struct CalculatorTests {
    let sut = Calculator()

    @Test("division calculation", arguments: [
        (10.0, 2.0, 5.0),
        (-10.0, 2.0, -5.0),
        (0.0, 5.0, 0.0),
        (7.0, 3.0, 2.3333),
    ])
    func divide_normalCalculation_BitsUT(a: Double, b: Double, expected: Double) throws {
        let result = try sut.divide(a, by: b)
        #expect(abs(result - expected) < 0.001)
    }

    @Test("divide by zero throws error")
    func divide_divideByZero_BitsUT() {
        #expect(throws: CalculatorError.divisionByZero) {
            try sut.divide(10, by: 0)
        }
    }
}
```

***

## 常见陷阱与修复

| 陷阱                         | 原因                                    | 修复方法                                     |
| -------------------------- | ------------------------------------- | ---------------------------------------- |
| `@testable import` 编译失败    | 测试 target 未将被测 target 作为依赖            | 在 `Package.swift` 或 Xcode 中添加 target 依赖  |
| XCTest 方法未被执行              | 方法名不以 `test` 开头或有参数                   | 确保方法签名为 `func testXxx()` 无参数             |
| `async` 测试中 Mock 未被调用      | 忘记 `await` 导致 actor 隔离问题              | 确保所有 async 方法调用都有 `await`                |
| 强制解包 crash 导致测试中断          | 使用 `!` 解包了 nil 值                      | 在 setUp 中使用 `guard` 或改用 Optional binding |
| `XCTAssertEqual` 对自定义类型失败  | 类型未实现 `Equatable`                     | 实现 `Equatable` 或逐字段断言                    |
| Swift Testing `@Test` 未被发现 | Xcode 版本过低或 swift-tools-version < 5.9 | 确认环境支持 Swift Testing，或回退使用 XCTest        |
| SPM 测试报 `dependency cycle` | 测试 target 和被测 target 存在循环依赖           | 重新组织 target 依赖结构                         |
| 测试中修改了 singleton 状态影响其他测试  | 全局/静态状态未在 tearDown 中恢复                | 在 `tearDown` 中重置状态，或使用依赖注入替代 singleton   |

***

## 上下文发现命令

```bash
grep -rn "class.*XCTestCase\|@Test\|@Suite\|import Testing\|import XCTest" --include="*.swift" <test_dir>
cat Package.swift 2>/dev/null
find . -name "*Tests.swift" -o -name "*Test.swift" | head -20
grep -rn "protocol.*Mock\|class.*Mock\|struct.*Mock" --include="*.swift" . | head -10
grep -rn "@testable import" --include="*.swift" . | head -10
```

