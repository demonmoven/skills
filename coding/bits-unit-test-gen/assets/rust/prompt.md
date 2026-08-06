# Rust 语言专属 Prompt

## 文档定位

本文件只承载 Rust 单测生成中对模型有增量价值的知识：Rust 项目的执行粒度、模块与 crate 决策、mock 约束、验证命令和 Rust 专属失败处理。通用单测原则、通用断言规范、边界值枚举、生成-验证-修复循环不在此重复。

优先级仍遵循全局规则：用户显式指令 > `AGENTS.md` / `CLAUDE.md` > 目标目录已有测试风格 > 本文件默认规则。

---

## 1. Rust 的处理粒度

Rust 的最小编译和验证单位是 **crate**（对应一个 `Cargo.toml` 定义的 package）。

对每个被选中的 crate，必须记录并在报告中沿用以下信息：

- crate 名称和路径（`Cargo.toml` 所在目录）
- 被选中的源文件、函数/方法、起始行
- 方法所属 `impl` 块的类型和 trait（如有）
- 对应测试模块或测试文件
- 实际执行的 `cargo test` 命令和工作目录

调度规则：

1. 按 crate 串行处理。
2. 同一 crate 内的多个目标函数一起生成/更新测试，统一维护 `use` 声明、辅助函数和 mock。
3. 当前 crate 写入完成后，以 crate 为单位验证和修复。
4. 当前 crate 收敛后再进入下一个 crate。

---

## 2. Workspace 与工作目录

生成测试前必须判断是否为 workspace 项目：

> **注意**：以下为语义参考。Agent 执行时应优先使用 Grep/Glob/Read 工具替代 shell 命令。

```bash
grep "\[workspace\]" Cargo.toml
find . -maxdepth 3 -name Cargo.toml
```

执行 `cargo test` 时：

- 必须在目标 crate 所属的 workspace 根目录或 crate 自身目录下运行。
- 使用 `-p <crate_name>` 指定目标 crate，或在 crate 目录下直接运行。
- 不要在 workspace 根目录执行不带 `-p` 的 `cargo test`，这会编译运行所有 crate 的测试，耗时且可能因无关 crate 编译失败而中断。

---

## 3. Rust 专属目标过滤

在全局文件过滤规则之外，Rust 还需要跳过：

- `fn main()` 入口函数
- `#[derive(...)]` 自动派生的 trait 实现
- 由 `build.rs` 或过程宏生成的代码（通常在 `OUT_DIR` 或 `target/` 中）
- Proto/Thrift 生成文件（`*.pb.rs`、`*.grpc.rs`）
- 行数少于 3 行、无分支/错误处理/依赖调用的简单 getter/setter
- `#[cfg(test)]` 模块自身（测试代码不是被测目标）
- FFI 的 `extern "C"` 桥接函数（仅做类型转换和转发的 wrapper）
- `/// # Examples` 文档测试（doc tests）— 属于文档维护范畴，不作为单测生成目标；验证时使用 `--lib` 跳过 doc tests

---

## 4. 写测试前的 Rust 专属学习

> **⚠️ 本节为索引入口**。具体执行步骤见第 14 节「预检查」。在生成任何测试代码之前，必须完成第 14 节的全部检查项。

---

## 5. 测试文件与模块约束

Rust 的单元测试有两种组织方式：

### 内联测试模块（推荐用于单元测试）

```rust
// 在源文件底部
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_foo_normal_case_bits_ut() {
        // ...
    }
}
```

### 集成测试（`tests/` 目录）

```
project/
├── src/
│   └── lib.rs
├── tests/
│   └── integration_test.rs
└── Cargo.toml
```

默认约定：

| 项目 | 默认规则 |
| --- | --- |
| 单元测试 | 源文件底部 `#[cfg(test)] mod tests` 块，优先追加到既有测试模块 |
| 集成测试 | `tests/` 目录下独立文件，测试 crate 的公开 API |
| 测试命名 | `test_{struct}_{method}_{scenario}_bits_ut` 或 `test_{func}_{scenario}_bits_ut`，除非项目已有明确风格 |
| use 声明 | 测试模块内 `use super::*;` 访问同模块私有成员；集成测试只能访问 pub 接口 |
| 可见性 | 内联测试模块可测试私有函数；集成测试只能测试公开 API |

---

## 6. Mock 策略

### mockall 使用约束

mockall 只用于不可控外部依赖（网络、数据库、文件系统、时间）。不要 mock：

- 目标函数本身
- 同模块内简单 helper
- 标准库纯函数（`str::len`、`Vec::push`、`Iterator` 方法等）
- 为了验证调用顺序而把内部逻辑全部 mock 掉

mockall 基本结构：

```rust
use mockall::{automock, predicate::*};

#[automock]
trait UserRepository {
    fn find_by_id(&self, id: i64) -> Result<Option<User>, DbError>;
}

#[cfg(test)]
mod tests {
    use super::*;
    use mockall::predicate;

    #[test]
    fn test_get_user_normal_case_bits_ut() {
        let mut mock_repo = MockUserRepository::new();
        mock_repo
            .expect_find_by_id()
            .with(predicate::eq(1001))
            .times(1)
            .returning(|_| Ok(Some(User { id: 1001, name: "张三".into() })));

        let service = UserService::new(Box::new(mock_repo));
        let user = service.get_user(1001).unwrap().unwrap();

        assert_eq!(user.name, "张三");
    }
}
```

### mockall 依赖声明

mockall 必须声明在 `[dev-dependencies]` 中，不应污染正式依赖。为此，业务代码中标注 `#[automock]` 时需配合条件编译：

**方式一：`cfg_attr` 条件标注（推荐，不侵入业务代码的正式编译）**

```rust
// 业务代码 — mockall 仅在 test 编译时生效
#[cfg_attr(test, mockall::automock)]
pub trait UserRepository: Send + Sync {
    fn find_by_id(&self, id: i64) -> Result<Option<User>, DbError>;
}
```

```toml
[dev-dependencies]
mockall = "0.13"
```

**方式二：`mock!` 宏在测试模块中定义（完全不侵入业务代码）**

```rust
// 业务代码 — 无任何 mockall 标注
pub trait UserRepository: Send + Sync {
    fn find_by_id(&self, id: i64) -> Result<Option<User>, DbError>;
}

// 测试模块
#[cfg(test)]
mod tests {
    use super::*;
    use mockall::mock;

    mock! {
        pub UserRepo {}
        impl UserRepository for UserRepo {
            fn find_by_id(&self, id: i64) -> Result<Option<User>, DbError>;
        }
    }

    #[test]
    fn test_example_bits_ut() {
        let mut mock = MockUserRepo::new();
        // ...
    }
}
```

**注意**：如果项目已有测试直接在业务代码中使用 `#[automock]`（即 mockall 在 `[dependencies]` 中），跟随已有风格即可，不必强行修改。

### Trait 注入（推荐默认方式）

Rust 惯用模式是通过泛型 + trait bound 或 trait object 注入依赖。应跟随被测代码已有的注入方式（泛型或 `Box<dyn Trait>`），不主动改写业务代码的依赖注入风格：

```rust
// 业务代码
pub struct UserService<R: UserRepository> {
    repo: R,
}

impl<R: UserRepository> UserService<R> {
    pub fn new(repo: R) -> Self {
        Self { repo }
    }

    pub fn get_user(&self, id: i64) -> Result<User, ServiceError> {
        // ...
    }
}
```

### 条件编译替换（简单场景）

对于无法或不需要引入 mockall 的场景，可使用 `#[cfg(test)]` 提供测试替代实现：

```rust
#[cfg(not(test))]
fn get_current_time() -> SystemTime {
    SystemTime::now()
}

#[cfg(test)]
fn get_current_time() -> SystemTime {
    // 返回固定时间用于测试
    SystemTime::UNIX_EPOCH + Duration::from_secs(1_700_000_000)
}
```

### Mock 反模式（禁止）

- ❌ Mock 简单工具函数（如 `format!`、`Vec::len`、`str::contains`）→ ✅ 直接调用，不需要 mock
- ❌ 对 `struct` 的私有字段通过 unsafe 方式强制访问 → ✅ 通过公共方法间接覆盖
- ❌ Mock 所有依赖使测试变成"验证调用顺序" → ✅ 仅 mock 不可控的外部依赖
- ❌ 使用 `unsafe` 进行不安全的内存操作来绕过可见性 → ✅ 使用 `#[cfg(test)]` 模块的 `use super::*`
- ❌ Mock 返回值的所有权/生命周期与 trait 签名不一致 → ✅ Mock 返回值必须严格符合 trait 定义的类型和生命周期

---

## 7. 验证命令

在目标 crate 所在目录或 workspace 根目录执行。

单 crate 验证：

```bash
cargo test -p <crate_name> -- --nocapture
```

仅运行特定测试函数：

```bash
cargo test -p <crate_name> <test_name> -- --nocapture
```

运行特定模块的测试：

```bash
cargo test -p <crate_name> <module_path>::tests -- --nocapture
```

常用附加选项：

```bash
# 只运行 lib target 的单元测试（跳过集成测试和 doc tests）
cargo test -p <crate_name> --lib

# 只运行某个集成测试文件
cargo test -p <crate_name> --test <test_file_name>

# 串行执行（有全局状态或文件系统竞争时）
cargo test -p <crate_name> -- --test-threads=1

# 带特定 feature 编译
cargo test -p <crate_name> --features <feature_name>
```

覆盖率（使用 cargo-llvm-cov）：

```bash
cargo llvm-cov --package <crate_name> --lcov --output-path ${TMP_ROOT}/lcov.info
cargo llvm-cov report --package <crate_name>
```

若新增了依赖声明（`Cargo.toml` 修改），Cargo 会自动拉取，无需额外操作。但如有 `Cargo.lock` 冲突（如 git merge 后），运行以下命令重新生成锁文件：

```bash
cargo generate-lockfile
```

若仅需更新某个依赖的锁定版本：

```bash
cargo update -p <dependency_name>
```

---

## 8. Rust 专属失败信号

以下信号只能结合上下文判定，不能机械归因为业务缺陷：

| 失败信号 | 判定要点 |
| --- | --- |
| `thread 'xxx' panicked at 'index out of bounds'` | 正常业务输入导致才可能是缺陷；测试故意构造非法空切片不算 |
| `thread 'xxx' panicked at 'called Option::unwrap() on a None value'` | None 由函数内部正常流程产生才可能是缺陷；测试故意传入导致 None 的参数不算 |
| `thread 'xxx' panicked at 'called Result::unwrap() on an Err value'` | 先检查 mock 返回值是否正确；正常流程也可能触发时才计入缺陷 |
| `thread 'xxx' panicked at 'attempt to divide by zero'` | 高概率是真实缺陷，需验证除数来源 |
| `thread 'xxx' panicked at 'integer overflow'`（debug 模式） | 需判断溢出是否在正常业务输入范围内 |
| 断言失败且期望值符合正确语义 | 高概率是逻辑缺陷，不得直接把期望改成实际错误结果 |

Rust 常见测试问题修复方向：

- `cannot find value/function/type in this scope`：优先检查 `use super::*;` 是否存在、模块路径是否正确、可见性是否满足（`pub`/`pub(crate)`）。
- `unresolved import`：检查 `Cargo.toml` 中是否声明了依赖，以及 `use` 路径是否正确。
- `trait bound not satisfied`：检查 mock 对象是否实现了所需 trait，泛型约束是否匹配。
- `lifetime does not live long enough`：检查测试中引用的生命周期，mock 返回值是否需要 owned 类型而非引用。
- `cannot borrow as mutable`：检查是否需要 `&mut self` 或 `RefCell`/`Mutex` 包装来满足借用规则。
- `mismatched types`：检查 mock 返回值类型是否与 trait 方法签名完全一致，注意 `Box<dyn Error>` vs 具体错误类型。
- mock 不生效（mockall）：确认 trait 标注了 `#[automock]` 或使用了 `mock!` 宏，确认 `expect_*` 设置在调用前。

---

## 9. 异步测试

Rust 异步测试需要 async runtime。根据项目使用的 runtime 选择：

### tokio（最常见）

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_fetch_user_success_bits_ut() {
        let mut mock_repo = MockUserRepository::new();
        mock_repo
            .expect_find_by_id()
            .returning(|_| Ok(Some(User { id: 1, name: "test".into() })));

        let service = UserService::new(Box::new(mock_repo));
        let result = service.fetch_user(1).await;

        assert!(result.is_ok());
        assert_eq!(result.unwrap().name, "test");
    }
}
```

### async-std

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[async_std::test]
    async fn test_fetch_user_success_bits_ut() {
        // ...
    }
}
```

在 `Cargo.toml` 中需确保测试依赖包含对应的 runtime：

```toml
[dev-dependencies]
tokio = { version = "1", features = ["macros", "rt-multi-thread"] }
```

---

## 10. 所有权与借用测试注意事项

Rust 的所有权系统对测试代码有以下影响：

- 被测函数消耗（`move`）参数时，每个测试用例需要独立构造输入值，不能跨 case 复用
- 返回引用的函数需要确保测试中被引用的数据活得足够长
- 使用 `Clone` trait 在需要时复制测试数据
- mock 对象若需要在多次调用间共享状态，使用 `Arc<Mutex<...>>` 或 mockall 的 `returning` 闭包
- 测试 `Send + Sync` 约束：异步代码中的 mock 对象需要满足 `Send + Sync`

---

## 11. Error 处理测试

Rust 的 `Result<T, E>` 模式要求充分覆盖错误路径：

### 测试 Ok 路径

```rust
#[test]
fn test_parse_config_valid_input_bits_ut() {
    let result = parse_config("key=value");
    assert!(result.is_ok());
    let config = result.unwrap();
    assert_eq!(config.key, "key");
    assert_eq!(config.value, "value");
}
```

### 测试 Err 路径

```rust
#[test]
fn test_parse_config_invalid_format_bits_ut() {
    let result = parse_config("invalid");
    assert!(result.is_err());
    assert!(matches!(result.unwrap_err(), ParseError::InvalidFormat(_)));
}
```

### 使用 `?` 操作符简化 Ok 路径测试

```rust
#[test]
fn test_parse_config_valid_input_bits_ut() -> Result<(), Box<dyn std::error::Error>> {
    let config = parse_config("key=value")?;
    assert_eq!(config.key, "key");
    assert_eq!(config.value, "value");
    Ok(())
}
```

### panic 测试

```rust
#[test]
#[should_panic(expected = "index out of bounds")]
fn test_get_item_out_of_bounds_bits_ut() {
    let list = ItemList::new(vec![]);
    list.get(10); // 应该 panic
}
```

---

## 12. unsafe 代码测试

包含 `unsafe` 块的函数需要额外关注内存安全：

### 测试策略

- **边界条件强化**：`unsafe` 代码通常处理裸指针、手动内存管理或 FFI，需重点覆盖空指针、越界、对齐等边界场景
- **使用安全包装验证**：优先通过安全 API 间接测试 `unsafe` 实现，确保公开接口的安全约定被满足
- **Miri 验证**（推荐）：对包含 `unsafe` 的代码使用 Miri 检测未定义行为（UB）

### Miri 使用

```bash
# 安装 Miri（需要 nightly toolchain）
rustup +nightly component add miri

# 运行特定测试
cargo +nightly miri test -p <crate_name> <test_name>
```

Miri 可检测：use-after-free、越界访问、未对齐指针解引用、数据竞争、违反 `&` 和 `&mut` 的别名规则等。

### 注意事项

- 不要在测试中为了触达 `unsafe` 内部逻辑而编写 `unsafe` 测试代码 — 通过安全接口间接覆盖
- 若被测函数是 `pub unsafe fn`，测试中调用时需包裹 `unsafe {}` 块，并在注释中说明为何该调用在测试上下文中是安全的
- 如果项目未使用 nightly toolchain 或 CI 不支持 Miri，不强制要求 Miri 验证

---

## 13. 属性测试（Property-based Testing）

当项目已使用 `proptest` 或 `quickcheck` 时，应跟随其风格。

### proptest 基本用法

```rust
#[cfg(test)]
mod tests {
    use super::*;
    use proptest::prelude::*;

    proptest! {
        #[test]
        fn test_encode_decode_roundtrip_bits_ut(input in "\\PC*") {
            let encoded = encode(&input);
            let decoded = decode(&encoded).unwrap();
            prop_assert_eq!(decoded, input);
        }
    }

    proptest! {
        #[test]
        fn test_sort_preserves_length_bits_ut(mut vec in prop::collection::vec(any::<i32>(), 0..100)) {
            let original_len = vec.len();
            my_sort(&mut vec);
            prop_assert_eq!(vec.len(), original_len);
        }
    }
}
```

### 适用场景

- 函数具有明确的数学不变式（如编码/解码往返一致性、排序后有序性）
- 输入空间大且边界值难以手工枚举
- 序列化/反序列化、解析器等需要验证广泛输入的场景

### 何时不使用

- 项目 `[dev-dependencies]` 中无 `proptest` / `quickcheck` 且用户未要求 — 不主动引入
- 函数行为依赖复杂外部状态（mock 难以与属性测试结合）
- 简单函数用确定性用例即可充分覆盖

---

## 14. 预检查（编写测试前必须完成）

> **⚠️ 硬性前置条件**：在生成任何测试代码之前，本步骤的项目学习**必须**完成。Rust 项目在构建配置、依赖管理和测试模式上差异很大。跳过此步骤几乎必然导致编译失败和反复修复。

### 1. 环境检测

1. **检测项目结构**：检查 `Cargo.toml` 确认是单 crate 还是 workspace
2. **检测 Rust 版本**：检查 `rust-toolchain.toml` 或 `Cargo.toml` 中的 `rust-version` — 影响可用特性（async、let-else、`#[expect]` 等）
3. **检测异步 runtime**：确认 `tokio`、`async-std` 或其他 — 影响异步测试宏选择
4. **检测已有测试依赖**：检查 `[dev-dependencies]` 中的测试相关 crate（`mockall`、`rstest`、`proptest`、`pretty_assertions`、`test-case` 等）
5. **确认 feature flags**：某些代码路径可能在特定 feature 下才编译，测试时需带对应 feature

### 2. 学习项目测试模式

学习目标 crate 中已有测试的风格：

1. **扫描已有测试**（必须）：阅读 1-2 个已有测试模块或测试文件，学习：
   - **测试组织**：内联 `#[cfg(test)] mod tests` 还是 `tests/` 目录？
   - **Mock 策略**：`mockall`、手写 mock struct、`#[cfg(test)]` 条件编译替换？
   - **断言风格**：标准库 `assert_eq!` 还是 `pretty_assertions`、`claims` 等？
   - **测试辅助**：是否使用 `rstest`（fixtures + parametrize）、`test-case`、`proptest`？
   - **命名规范**：测试函数的实际命名模式
   - **错误处理**：测试返回 `Result<(), E>` 还是使用 `unwrap()`？
2. **发现可复用资产**（推荐）：
   ```bash
   grep -rn "mod tests\|#\[test\]\|fn setup\|fn fixture\|TestBuilder\|Helper\|mock" --include="*.rs" <target_dir>
   ```
   - 如果存在 test helper 模块或 builder pattern，优先复用
3. **阅读项目约定**（推荐）：检查 `PROJECT_ROOT` 下的 `AGENTS.md`、`CLAUDE.md`
   - 提取单测相关要求

### 3. 上下文分析

对于每个目标函数，收集充分的上下文信息：

1. **第一层（必须）**：阅读目标函数源码，理解函数签名、泛型约束、生命周期参数、所属 `impl` 块的类型
2. **第二层（推荐）**：阅读依赖的 trait 定义（确定 mock 策略和返回类型）、错误类型定义
3. **第三层（按需）**：当第二层信息不足时，阅读间接依赖、struct/enum 定义、feature-gated 代码

---

## 15. Rust 单测标准

### 测试函数签名

- 测试函数必须标注 `#[test]`（同步）或 `#[tokio::test]` / `#[async_std::test]`（异步）
- 函数无参数，返回 `()` 或 `Result<(), E>`
- 测试模块必须标注 `#[cfg(test)]`，确保不会编译进生产代码

### 测试隔离原则

- 每个测试函数必须独立，不依赖执行顺序或其他测试的副作用
- 禁止通过 `static mut` 或全局变量在测试间传递状态
- 需要共享初始化逻辑时，使用 helper 函数或 `rstest` fixtures
- `once_cell` / `lazy_static` 全局状态如果被测试修改，需使用 `Mutex` 保护或独立进程运行
- 需要串行执行的测试（如操作文件系统、全局状态），若项目已有 `serial_test` crate，使用 `#[serial]` 标注；否则可通过 `-- --test-threads=1` 运行
- 耗时较长或依赖外部环境的测试使用 `#[ignore]` 标注，避免影响正常 `cargo test` 执行速度

### 断言标准

- `assert_eq!(actual, expected)` — 相等比较（要求实现 `PartialEq` + `Debug`）
- `assert_ne!(actual, expected)` — 不等比较
- `assert!(condition)` — 布尔条件
- `assert!(matches!(expr, pattern))` — 模式匹配（enum variant 验证）；断言失败时默认只显示 `false`，建议添加自定义消息：`assert!(matches!(result, Pattern::Variant(_)), "unexpected: {:?}", result)`
- 浮点比较使用 `assert!((actual - expected).abs() < epsilon)`，其中 `epsilon` 的选取：
  - epsilon 类型应与被测浮点类型一致（`f32` 场景用 `f32::EPSILON`，`f64` 场景用 `f64::EPSILON`）
  - 简单赋值/无计算损失场景可用对应类型的 `EPSILON`
  - 涉及多次浮点运算时应使用更宽松的容差（如 `1e-6` for f32、`1e-10` for f64），根据精度需求调整
  - 项目已使用 `approx` crate 时，优先使用 `assert_abs_diff_eq!` 或 `assert_relative_eq!`
- 优先使用精确值比较，不要仅断言 `is_some()` / `is_ok()` 而忽略具体值

### 边界值与特殊值覆盖

- `Option<T>` 参数需覆盖 `None` 和 `Some(...)` 场景
- `String` / `&str` 参数需覆盖空字符串 `""` 场景
- `Vec<T>` / slice 参数需覆盖空集合 `vec![]` / `&[]` 场景
- 数值参数需覆盖 `0`、负数、边界值（`i32::MAX`、`i32::MIN`、`usize::MAX`）场景
- `Result<T, E>` 返回值需覆盖 `Ok` 和 `Err` 场景

---

## 16. 格式化

```bash
cargo fmt -p <crate_name>
```

若 `cargo fmt -p` 不被支持（旧版 rustfmt），可在 crate 目录下直接运行 `cargo fmt`。

---

## 17. 代码风格

风格由优先级决定：用户指令 > AGENTS.md > 同目录已有测试 > 以下默认值。

| 项目 | 规范 |
| --- | --- |
| 注释语言 | 跟随项目已有注释语言，无参考时默认中文 |
| 命名 | `test_{struct}_{method}_{scenario}_bits_ut` 或 `test_{func}_{scenario}_bits_ut`（English only） |
| 位置 | 源文件底部 `#[cfg(test)] mod tests` 块，优先追加既有测试模块 |
| 断言 | 标准库 `assert_eq!` / `assert!`（或跟随已有测试） |
| Mock | trait 注入 + `mockall`（或跟随已有测试） |
| 测试用例组织 | 按被测函数分组，复杂场景使用 `rstest` 参数化 |
| use 声明 | `use super::*;` + 按需引入外部 crate，标准库 → 第三方 → 项目内部 |

---

## 18. 示例

### 示例 1：基本内联测试

目标函数：

```rust
pub fn divide(a: f64, b: f64) -> Result<f64, MathError> {
    if b == 0.0 {
        return Err(MathError::DivisionByZero);
    }
    Ok(a / b)
}
```

测试代码：

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_divide_normal_division_bits_ut() {
        let result = divide(10.0, 2.0).unwrap();
        assert!((result - 5.0).abs() < f64::EPSILON);
    }

    #[test]
    fn test_divide_division_by_zero_bits_ut() {
        let result = divide(10.0, 0.0);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), MathError::DivisionByZero));
    }

    #[test]
    fn test_divide_negative_numbers_bits_ut() {
        let result = divide(-10.0, 2.0).unwrap();
        assert!((result - (-5.0)).abs() < f64::EPSILON);
    }

    #[test]
    fn test_divide_zero_dividend_bits_ut() {
        let result = divide(0.0, 5.0).unwrap();
        assert!((result - 0.0).abs() < f64::EPSILON);
    }
}
```

### 示例 2：Trait 注入 + mockall

目标方法：

```rust
// mockall 仅在 test 编译时生效，mockall 声明在 [dev-dependencies] 中
#[cfg_attr(test, mockall::automock)]
pub trait UserRepository: Send + Sync {
    fn find_by_id(&self, id: i64) -> Result<Option<User>, DbError>;
}

pub struct UserService {
    repo: Box<dyn UserRepository>,
}

impl UserService {
    pub fn new(repo: Box<dyn UserRepository>) -> Self {
        Self { repo }
    }

    pub fn get_user(&self, id: i64) -> Result<User, ServiceError> {
        if id <= 0 {
            return Err(ServiceError::InvalidId);
        }
        match self.repo.find_by_id(id)? {
            Some(user) => Ok(user),
            None => Err(ServiceError::NotFound(id)),
        }
    }
}
```

测试代码：

```rust
#[cfg(test)]
mod tests {
    use super::*;
    use mockall::predicate;

    fn setup_service(mock: MockUserRepository) -> UserService {
        UserService::new(Box::new(mock))
    }

    #[test]
    fn test_get_user_normal_case_bits_ut() {
        let mut mock_repo = MockUserRepository::new();
        mock_repo
            .expect_find_by_id()
            .with(predicate::eq(1001))
            .times(1)
            .returning(|_| Ok(Some(User { id: 1001, name: "张三".into() })));

        let service = setup_service(mock_repo);
        let user = service.get_user(1001).unwrap();

        assert_eq!(user.id, 1001);
        assert_eq!(user.name, "张三");
    }

    #[test]
    fn test_get_user_not_found_bits_ut() {
        let mut mock_repo = MockUserRepository::new();
        mock_repo
            .expect_find_by_id()
            .with(predicate::eq(999))
            .times(1)
            .returning(|_| Ok(None));

        let service = setup_service(mock_repo);
        let result = service.get_user(999);

        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ServiceError::NotFound(999)));
    }

    #[test]
    fn test_get_user_invalid_id_zero_bits_ut() {
        let mock_repo = MockUserRepository::new();
        let service = setup_service(mock_repo);

        let result = service.get_user(0);

        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ServiceError::InvalidId));
    }

    #[test]
    fn test_get_user_invalid_id_negative_bits_ut() {
        let mock_repo = MockUserRepository::new();
        let service = setup_service(mock_repo);

        let result = service.get_user(-1);

        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ServiceError::InvalidId));
    }

    #[test]
    fn test_get_user_db_error_bits_ut() {
        let mut mock_repo = MockUserRepository::new();
        mock_repo
            .expect_find_by_id()
            .returning(|_| Err(DbError::ConnectionFailed));

        let service = setup_service(mock_repo);
        let result = service.get_user(1);

        assert!(result.is_err());
    }
}
```

### 示例 3：rstest 参数化测试

```rust
#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    #[rstest]
    #[case(1, 2, 3, "two positive numbers")]
    #[case(5, -3, 2, "positive plus negative")]
    #[case(-1, -2, -3, "two negative numbers")]
    #[case(10, 0, 10, "add zero")]
    #[case(0, 0, 0, "both zero")]
    fn test_add_bits_ut(#[case] a: i32, #[case] b: i32, #[case] expected: i32, #[case] _desc: &str) {
        assert_eq!(add(a, b), expected);
    }
}
```

---

## 19. 常见陷阱与修复

| 陷阱 | 原因 | 修复方法 |
| --- | --- | --- |
| `cannot find value in this scope` | 测试模块缺少 `use super::*;` 或目标函数非 pub | 添加 `use super::*;` 或检查可见性 |
| `trait bound not satisfied` | mock 对象未实现所需 trait 或泛型约束不匹配 | 确认 `#[automock]` 标注正确，检查 trait 上的 `Send + Sync` 约束 |
| `cannot borrow as mutable more than once` | 测试中多次可变借用同一对象 | 重新组织代码，分开借用作用域，或使用 `RefCell` |
| `lifetime may not live long enough` | mock 返回引用而非 owned 类型 | 使 trait 方法返回 owned 值，或使用 `'static` 引用 |
| `unresolved import` | `Cargo.toml` 中未添加 dev-dependency | 在 `[dev-dependencies]` 中添加所需 crate |
| mockall `expect_*` 未被调用 | mock 设置后未被业务代码路径触发 | 检查 mock 是否正确注入到被测代码中 |
| `test result: FAILED` 但无 panic 信息 | 测试返回 `Result<(), E>` 且返回了 `Err` | 检查 `?` 操作符对应的调用是否返回了错误 |
| `dead_code` 警告干扰编译 | 测试专用的 helper 函数被标记为未使用 | 在测试 helper 上添加 `#[allow(dead_code)]` 或确保在测试中引用 |
| 异步测试 hang 住 | 缺少 tokio runtime 或 runtime 配置不当 | 确认使用 `#[tokio::test]` 且 `tokio` dev-dependency 包含 `macros` + `rt` feature |
| `multiple applicable items in scope` | `use super::*` 引入了与外部 crate 或 prelude 同名的函数/宏/类型 | 使用具名引入替代通配符导入（如 `use super::{specific_fn, SpecificType}`），或通过路径限定消歧义 |

---

## 20. 上下文发现命令

> **注意**：以下为语义参考。Agent 执行时应优先使用对应工具（Grep/Glob/Read）替代 shell 命令。

```bash
# 查找已有测试（Agent 优先用 Grep 工具）
grep -rn "#\[test\]\|#\[tokio::test\]\|#\[async_std::test\]\|mod tests" --include="*.rs" <target_dir>
# 查找 mock 使用（Agent 优先用 Grep 工具）
grep -rn "mockall\|#\[automock\]\|mock!\|MockAll" --include="*.rs" .
# 读取 Cargo.toml（Agent 优先用 Read 工具）
cat Cargo.toml
# 查找集成测试文件（Agent 优先用 Glob 工具）
find . -path "*/tests/*.rs" | head -20
# 查找 dev-dependencies（Agent 优先用 Grep 工具）
grep -rn "\[dev-dependencies\]" --include="Cargo.toml" -A 10 .
```
