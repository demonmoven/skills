# Java Testing Framework Configuration

## 1. Test Organization

### 1.1 Directory Structure

```
src/
├── main/java/com/bytedance/csp/
│   ├── controller/
│   ├── service/
│   └── repository/
└── test/java/com/bytedance/csp/
    ├── controller/
    │   └── XxxControllerTest.java
    ├── service/
    │   └── XxxServiceTest.java
    └── repository/
        └── XxxRepositoryTest.java
```

- Test files mirror source files, located under `src/test/java/`
- Test classes named `XxxTest.java`

### 1.2 JUnit 5 Basic Configuration

```xml
<dependency>
  <groupId>org.junit.jupiter</groupId>
  <artifactId>junit-jupiter</artifactId>
  <scope>test</scope>
</dependency>
```

```java
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class ExampleTest {
    @Test
    void testAddition() {
        assertEquals(2, 1 + 1);
    }
}
```

### 1.3 Helper Libraries

| Library | Purpose | When to Use |
|---------|---------|-------------|
| `org.mockito:mockito-core` | Mock objects | Required for isolating external dependencies |
| `org.mockito:mockito-junit-jupiter` | JUnit 5 integration | Recommended |
| `com.h2database:h2` | In-memory database | Repository layer testing |
| `org.springframework.boot:spring-boot-starter-test` | Spring Boot testing | Spring Boot projects |

## 2. Coverage Configuration (JaCoCo)

```xml
<plugin>
  <groupId>org.jacoco</groupId>
  <artifactId>jacoco-maven-plugin</artifactId>
  <version>0.8.11</version>
  <executions>
    <execution>
      <goals><goal>prepare-agent</goal></goals>
    </execution>
    <execution>
      <id>report</id>
      <phase>test</phase>
      <goals><goal>report</goal></goals>
    </execution>
  </executions>
</plugin>
```

- **Minimum Coverage**: Core logic ≥ 60%, utility classes ≥ 80%

## 3. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| Test Runner | JUnit 5 | TestNG | Legacy projects already using TestNG |
| Assertion | JUnit 5 built-in | AssertJ | When fluent assertion style is preferred |
| Mocking | Mockito 5.x | EasyMock | Never (Mockito is standard) |
| Coverage | JaCoCo | Cobertura | Never (Cobertura is unmaintained) |
| In-memory DB | H2 | Derby | Never (H2 is more feature-complete) |

Priority: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 4. Auto-Completion Procedure

### JUnit 5 Auto-Completion

1. Check for `src/test/java/` directory → test directory exists
2. Check for JUnit 5 dependency in `pom.xml` or `build.gradle` → framework configured
3. Check for test files matching `*Test.java` → tests written
4. If directory exists but no framework → add JUnit 5 + Mockito dependencies to build file
5. If framework exists but no tests → generate smoke test
6. If no test directory → full auto-completion:
   - Create `src/test/java/` directory structure mirroring main
   - Add JUnit 5 + Mockito dependencies to build configuration
   - Add JaCoCo plugin configuration to build file
   - Create smoke test file (`ApplicationSmokeTest.java`)
   - Run: `mvn test` or `gradle test` → must exit 0
7. Acceptance criteria:
   - [ ] `src/test/java/` exists with correct directory structure
   - [ ] JUnit 5 and Mockito dependencies configured
   - [ ] JaCoCo plugin configured in build file
   - [ ] `mvn test` or `gradle test` exits 0
   - [ ] Coverage report can be generated
