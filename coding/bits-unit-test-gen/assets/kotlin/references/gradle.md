# Kotlin Gradle Reference

## 模块定位

- 根目录执行，通过 Gradle project path 指定模块（`:module`、`:parent:child`）
- 路径来源于 `settings.gradle(.kts)` 的 `include(...)`，不要用文件系统路径猜测

## 测试命令

```bash
# 单类
./gradlew :<module>:test --tests "<fully.qualified.TestClass>"
# 单方法
./gradlew :<module>:test --tests "<fully.qualified.TestClass.testMethodName>"
# 覆盖率
./gradlew :<module>:test --tests "<TestClass>" :<module>:jacocoTestReport
```

单模块项目省略 `:<module>:` 前缀。`jacocoTestReport` 不存在时通过 `./gradlew tasks --group=verification` 查找实际任务名。

## Android / KMP

- Android 本地单测：`test<Variant>UnitTest`
- KMP：使用 target 任务（`jvmTest`、`jsTest` 等）
- 先读 `android.md` 或 `kmp.md` 再落命令

## 反引号方法名

反引号方法名在 `--tests` 中直接使用空格：

```bash
./gradlew :<module>:test --tests "<TestClass.获取用户 正常返回 BitsUT>"
```

过滤失败时退回类级过滤。

## 嵌套测试类

JUnit 5 `@Nested` 内部类用 `$` 分隔：

```bash
./gradlew :<module>:test --tests "com.example.OuterTest\$NestedInner.testMethod"
```

若过滤命中 0 个测试，优先检查是否误用了 `.` 分隔嵌套类名。

## 调试

失败时按需添加 `--stacktrace` 或 `--info`；`--rerun-tasks` 跳过缓存。不要默认添加。
