# Kotlin Maven Reference

## 模块定位

根目录执行，通过 `-pl <module>` 指定目标模块。`<module>` 与 `pom.xml` 中 `<modules>` 声明一致。

## 测试命令

```bash
# 单类
mvn test -pl <module> -Dtest=<TestClass>
# 单方法
mvn test -pl <module> -Dtest=<TestClass>#<testMethod>
# 覆盖率
mvn test -pl <module> -Dtest=<TestClass> -Djacoco.skip=false
mvn jacoco:report -pl <module>
```

单模块项目省略 `-pl <module>`。

## 注意

- 单测和集成测试分 Surefire/Failsafe 时，沿用已有单测插件路径
- Kotlin 编译报错先检查 `kotlin-maven-plugin` 配置
- 项目已有 profile、wrapper 或企业脚本时优先沿用
