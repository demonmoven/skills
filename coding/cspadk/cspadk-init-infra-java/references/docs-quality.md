# Quality Documentation Templates

## 1. quality/testing-strategy.md — Testing Strategy & Coverage Requirements

| Field | Content |
|-------|---------|
| Purpose | Define testing approach, tooling, coverage requirements, and best practices |
| Sections | 1) Testing Philosophy 2) Test Types (unit/integration/E2E) 3) Tooling (table: Tool / Version / Purpose) 4) Coverage Requirements (table: Module Type / Line / Branch) 5) Test Conventions 6) Running Tests |
| Content Guidelines | Testing philosophy must state the team's approach clearly. Test types must describe when to use each. Tooling table must list all test-related packages. Coverage requirements must specify thresholds by module type. Test conventions must cover naming, file location, and patterns. Running tests must include all common commands. |
| Quality Checks | Testing philosophy is stated; All test types are described; Tooling table is complete; Coverage thresholds are specified by module type; Test conventions are documented; Common test commands are provided |

### Java Tooling Table

```markdown
| Tool | Version | Purpose |
|------|---------|---------|
| JUnit 5 | 5.10+ | Test runner and assertions |
| Mockito | 5.x | Mocking framework |
| JaCoCo | 0.8.11+ | Code coverage |
| H2 | 2.x | In-memory database for repository tests |
| Spring Boot Test | 3.x | Integration test support |
| Testcontainers | 1.x | Container-based integration tests |
```

### Coverage Requirements Table

```markdown
| Module Type | Line Coverage | Branch Coverage |
|-------------|--------------|----------------|
| Core logic | >= 60% | >= 50% |
| Utility classes | >= 80% | >= 70% |
| Controller / Service | >= 60% | >= 50% |
| Repository / DAO | >= 50% | >= 40% |
```

### Test Conventions Example

```markdown
### Test Conventions

- **File naming**: `XxxTest.java` — test classes mirror source structure under `src/test/java/`
- **Grouping**: Use `@Nested` inner classes to group related test cases within a test class
- **Display names**: Use `@DisplayName` for descriptive, human-readable test names
- **Test naming**: Use `should<ExpectedBehavior>When<Condition>()` method naming pattern
- **Mocking**: Use `@Mock` and `@InjectMocks` with `@ExtendWith(MockitoExtension.class)` for dependency injection
- **Assertions**: Prefer JUnit 5 assertions (`assertEquals`, `assertThrows`) over AssertJ for consistency
- **Test isolation**: Each test must be independent; use `@BeforeEach` for setup, avoid shared mutable state
- **Spring context**: Only use `@SpringBootTest` when integration testing; prefer plain unit tests for service logic
```

### Running Tests Example

```markdown
### Running Tests

```bash
# Run all unit tests
mvn test

# Run unit + integration tests
mvn verify

# Run tests for a single module
mvn test -pl <module>

# Run a single test class
mvn test -Dtest=XxxTest

# Run a single test method
mvn test -Dtest=XxxTest#shouldReturnResultWhenInputValid

# Run tests with coverage report
mvn test jacoco:report

# Skip tests during build
mvn install -DskipTests

# Gradle equivalents
./gradlew test
./gradlew test --tests "com.example.XxxTest"
./gradlew test --tests "*.XxxTest.shouldReturnResultWhenInputValid"
./gradlew build -x test
```
```

---

## 2. quality/code-review-guide.md — Code Review Guide

| Field | Content |
|-------|---------|
| Purpose | Code review standards, checklist, and process |
| Sections | 1) Review Process (who/when/how) 2) Review Checklist (correctness, security, performance, maintainability) 3) Common Issues (with Java code examples) 4) Review Etiquette |
| Content Guidelines | Review process must define who reviews, when reviews happen, and how they are conducted. Checklist must be concise — no more than 15 items. Common issues must include Java code examples showing the anti-pattern and the fix. Review etiquette must cover tone, timeliness, and resolution of disagreements. |
| Quality Checks | Checklist has <= 15 items; Common issues include Java code examples; Review process is clearly defined; Review etiquette is documented |

### Review Process Example

```markdown
### Review Process

1. **Who**: At least one peer reviewer + one senior reviewer for architecture changes
2. **When**: Review requested within 1 business day of MR creation
3. **How**: Use merge request comments; approve only when all comments are resolved
4. **Resolution**: Author addresses comments; reviewer re-reviews after changes
5. **Merge**: Squash merge after all approvals and CI passes
```

### Review Checklist Example

```markdown
### Review Checklist (15 items)

**Correctness**
- [ ] Code does what the MR description says
- [ ] Edge cases and null inputs are handled
- [ ] Error handling is appropriate and exceptions are specific

**Security**
- [ ] No hardcoded secrets or credentials
- [ ] Input validation is in place (especially on API boundaries)
- [ ] Dependencies do not introduce known vulnerabilities

**Performance**
- [ ] No N+1 queries or unnecessary loops
- [ ] Proper use of collections (avoid O(n) lookups where O(1) is available)
- [ ] Thread safety is ensured for shared mutable state

**Maintainability**
- [ ] Code is readable and self-documenting
- [ ] Methods are single-responsibility and not excessively long
- [ ] Types are explicit; no raw types or unchecked casts
- [ ] Tests cover the new/changed code
- [ ] No dead code or commented-out code

**Compatibility**
- [ ] Changes are backward-compatible or properly versioned
```

### Common Issues Example

```markdown
### Common Issues with Java Code Examples

#### 1. NullPointerException — Prefer Optional over null checks

```java
// Anti-pattern — null check is error-prone and unclear
public String getUserName(User user) {
    if (user != null) {
        return user.getName();
    }
    return "unknown";
}

// Fix — use Optional for explicit null-safety
public String getUserName(User user) {
    return Optional.ofNullable(user)
            .map(User::getName)
            .orElse("unknown");
}
```

#### 2. Resource Leaks — Use try-with-resources

```java
// Anti-pattern — resource may leak if exception occurs
public String readFile(String path) throws IOException {
    BufferedReader reader = new BufferedReader(new FileReader(path));
    return reader.readLine(); // reader never closed if exception thrown
}

// Fix — try-with-resources guarantees cleanup
public String readFile(String path) throws IOException {
    try (BufferedReader reader = new BufferedReader(new FileReader(path))) {
        return reader.readLine();
    }
}
```

#### 3. Thread Safety — Use thread-safe collections for shared state

```java
// Anti-pattern — HashMap is not thread-safe
public class RequestCache {
    private final Map<String, Response> cache = new HashMap<>(); // shared mutable state
}

// Fix — use ConcurrentHashMap for thread-safe access
public class RequestCache {
    private final Map<String, Response> cache = new ConcurrentHashMap<>();
}
```

#### 4. Immutable Objects — Use final fields and defensive copies

```java
// Anti-pattern — mutable fields expose internal state
public class Config {
    private List<String> servers; // not final, not defensive copy
    public List<String> getServers() { return servers; } // caller can mutate
}

// Fix — final fields + defensive copies
public class Config {
    private final List<String> servers;

    public Config(List<String> servers) {
        this.servers = List.copyOf(servers); // defensive copy on input
    }

    public List<String> getServers() {
        return servers; // already unmodifiable via List.copyOf
    }
}
```

#### 5. Exception Handling — Catch specific exceptions, not Exception

```java
// Anti-pattern — catching Exception hides real failures
try {
    fileService.process(path);
    databaseService.save(record);
} catch (Exception e) {
    log.error("Something went wrong", e); // swallows IOException, SQLException, etc.
}

// Fix — catch specific exceptions separately
try {
    fileService.process(path);
} catch (FileNotFoundException e) {
    log.warn("File not found: {}", path, e);
    throw new BusinessException("File not found", e);
}

try {
    databaseService.save(record);
} catch (SQLException e) {
    log.error("Database error saving record", e);
    throw new BusinessException("Database error", e);
}
```
```

### Review Etiquette Example

```markdown
### Review Etiquette

- **Be constructive**: Frame feedback as suggestions, not commands
- **Explain why**: Provide rationale, not just "change this"
- **Be timely**: Review within 1 business day of request
- **Keep it focused**: Comment on code quality, not personal style preferences
- **Resolve disagreements**: Discuss in comments or synchronous call; escalate to tech lead if needed
- **Acknowledge good work**: Call out well-written code and clever solutions
```
