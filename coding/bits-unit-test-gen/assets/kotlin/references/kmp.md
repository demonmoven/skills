# Kotlin Multiplatform Reference

## 适用范围

适用于使用 Kotlin Multiplatform（`kotlin {}` 多 target）的项目。`commonTest` 是共享测试 source set，不可直接执行；必须选一个实际 target 承载验证。`commonTest` 使用 `kotlin.test`（`@Test`、`assertEquals`、`assertFailsWith`），不要 import `org.junit.*`。target-specific 测试可额外使用平台测试库。

## 任务选择

- `commonMain` 代码：优先选项目最常用 target；无线索时用 `jvmTest`
- `jvmMain` → `jvmTest`；`jsMain` → `jsTest`；iOS → 对应 simulator target

```bash
./gradlew :<module>:jvmTest --tests "<TestClass>"
./gradlew :<module>:jsTest
./gradlew :<module>:iosSimulatorArm64Test
```

单模块项目省略 `:<module>:` 前缀。

## 测试过滤

- JVM target 支持 `--tests`
- JS/Native/iOS 不一定支持；不支持时退回 target 任务级验证，不要伪造过滤参数

## 平台特有陷阱与解法

- `commonTest` 中只能使用 `kotlin.test` 断言，不能依赖 JUnit API
- `expect` 声明本身无实现，测试应针对 `actual`
- 已有 expect/actual 测试模式时沿用

## 注意事项

- JVM-only 测试依赖不要加到 `commonTest`
- `internal` 在跨模块/自定义 source set 时可能受 friend paths 影响
