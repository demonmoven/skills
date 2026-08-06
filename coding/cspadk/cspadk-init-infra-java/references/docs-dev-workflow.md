# Development Documentation Templates

## 1. development/getting-started.md — Quick Start Guide

| Field | Content |
|-------|---------|
| Purpose | Help new contributors set up environment and make first contribution |
| Sections | 1) Prerequisites (tools + versions) 2) Environment Setup (copy-paste commands) 3) First Run 4) First Contribution |
| Content Guidelines | Every step must have a copy-paste command. Prerequisites must include version requirements. Common pitfalls must be documented with workarounds. |
| Quality Checks | Every step has a copy-paste command; Prerequisites include version requirements; Common pitfalls are documented |

### Java Example — Prerequisites

```markdown
### Prerequisites

| Tool | Version | Install | Verify |
|------|---------|---------|--------|
| JDK | >= 17 | Adoptium / SDKMAN | `java -version` (must be >= 17) |
| Maven | >= 3.9 | sdk install maven | `mvn --version` |
| Gradle | >= 8 | sdk install gradle | `gradle --version` |
| Git | >= 2.x | System package manager | `git --version` |
| IDE | IntelliJ IDEA (recommended) | JetBrains Toolbox | N/A |

> **Note**: The project requires JDK 17+. Using an older JDK will cause compilation
> and runtime failures. Maven and Gradle are alternatives — use whichever the
> project's build files indicate.

### Environment Setup

#### Maven

```bash
# 1. Clone the repository
git clone <repo-url> && cd <repo>

# 2. Resolve dependencies
mvn dependency:resolve

# 3. Compile the project
mvn compile

# 4. Start the application
mvn spring-boot:run
```

#### Gradle

```bash
# 1. Clone the repository
git clone <repo-url> && cd <repo>

# 2. Resolve dependencies
./gradlew dependencies

# 3. Build the project
./gradlew build

# 4. Start the application
./gradlew bootRun
```

### First Run

```bash
# Verify the setup
java -version       # >= 17
mvn --version       # Maven >= 3.9 (or gradle --version >= 8)
git --version       # >= 2.x

# Maven
mvn compile         # Compilation succeeds
mvn test            # Tests pass
mvn spring-boot:run # Application starts

# Gradle
./gradlew build     # Build + tests succeed
./gradlew bootRun   # Application starts
```

### First Contribution

```bash
# 1. Create a feature branch
git checkout -b feat/<ticket-id>-<description>

# 2. Make changes and write tests

# 3. Run checks before pushing
mvn verify                  # Maven: compile + test + integration test
mvn checkstyle:check        # Maven: style check (if configured)
# OR
./gradlew check             # Gradle: build + test + lint

# 4. Push and create MR
git push origin feat/<ticket-id>-<description>
```

### Common Pitfalls

| Pitfall | Cause | Fix |
|---------|-------|-----|
| JDK version mismatch | Project requires 17 but system has 11 | Install JDK 17+ and set `JAVA_HOME`; use `sdk install java <version>` via SDKMAN |
| Maven local repo cache issues | Corrupted or stale cached artifacts | `mvn dependency:purge-local-repository` then re-resolve |
| Internal repo not configured | Cannot pull internal dependencies | Add repository to `pom.xml` or `build.gradle`; check `settings.xml` for auth |
| Build fails after dependency change | Transitive dependency conflict | `mvn dependency:tree` to inspect; use `<dependencyManagement>` to pin versions |
| Gradle wrapper permission denied | `gradlew` not executable | `chmod +x gradlew` |
| `spring-boot:run` fails with port in use | Another process on same port | `lsof -i :<port>` then kill, or set `server.port` in `application.yml` |
```

---

## 2. development/workflow.md — Development Workflow Description

| Field | Content |
|-------|---------|
| Purpose | Describe development workflow: feature development, testing, review, and deployment |
| Sections | 1) Development Flow (numbered sequence) 2) Local Development (hot reload) 3) Debugging Guide 4) Testing Workflow (unit + integration) 5) Deployment Flow (includes rollback) |
| Content Guidelines | Development flow must be a clear numbered sequence from feature request to production. Local development must describe hot reload setup. Debugging guide must cover common debugging scenarios. Testing workflow must distinguish unit and integration tests. Deployment flow must include rollback procedure. |
| Quality Checks | Development flow is a numbered sequence; Hot reload configuration is described; Debugging scenarios are covered; Unit and integration tests are distinguished; Rollback procedure is included |

### Java Example — Development Workflow

```markdown
### Development Flow

1. **Create Feature Branch**: `git checkout -b feat/<ticket-id>-<description>`
2. **Implement Feature**: Write code and tests following TDD when applicable
3. **Run Local Checks**:
   - Maven: `mvn verify && mvn checkstyle:check`
   - Gradle: `./gradlew check`
4. **Push and Create MR**: Push branch and create merge request
5. **Code Review**: Address review feedback
6. **CI Validation**: Ensure all pipeline stages pass (build, test, integration test)
7. **Merge**: Squash merge to main after approval
8. **Deploy**: Follow deployment flow for staging and production

### Local Development

#### Hot Reload with Spring Boot DevTools (Maven)

```bash
# Add DevTools dependency (if not already present)
# In pom.xml:
#   <dependency>
#     <groupId>org.springframework.boot</groupId>
#     <artifactId>spring-boot-devtools</artifactId>
#     <scope>runtime</scope>
#     <optional>true</optional>
#   </dependency>

# Start with automatic restart on classpath changes
mvn spring-boot:run
```

#### Hot Reload with Spring Boot DevTools (Gradle)

```bash
# Add DevTools dependency (if not already present)
# In build.gradle:
#   developmentOnly 'org.springframework.boot:spring-boot-devtools'

# Start with continuous build in a separate terminal
./gradlew bootRun
# In another terminal, keep building on changes:
./gradlew classes -t
```

#### IDE-Based Hot Reload

- **IntelliJ IDEA**: Enable "Build project automatically" in Settings >
  Compiler, then enable "compiler.automake.allow.when.app.running" in
  Registry. Spring Boot DevTools will auto-restart on class changes.

### Debugging Guide

#### Remote Debugging

```bash
# Maven: start with JDWP agent
mvn spring-boot:run -Dspring-boot.run.jvmArguments="-Xdebug -Xrunjdwp:transport=dt_socket,server=y,suspend=n,address=5005"

# Gradle: start with JDWP agent
./gradlew bootRun --debug-jvm
```

Then attach your IDE remote debugger to port 5005.

#### Common Debugging Scenarios

| Scenario | Tool | Steps |
|----------|------|-------|
| Application startup failure | IDE debugger + logs | Check `application.yml` for config errors; set breakpoint in `main()` |
| API endpoint returning unexpected response | Postman / curl + IDE debugger | Set breakpoint in controller method; inspect request/response |
| Dependency injection error | Stack trace analysis | Check `@Component`, `@Service`, `@Autowired` annotations and component scan config |
| Database query issue | IDE debugger + DB client | Enable SQL logging (`spring.jpa.show-sql=true`); inspect query and parameters |
| Performance bottleneck | JVM profiler (JProfiler / VisualVM) | Profile CPU and memory; check for N+1 queries, excessive object creation |
| Memory leak | Heap dump analysis | `jmap -dump:format=b,file=heap.bin <pid>`; analyze with Eclipse MAT |

### Testing Workflow

1. **Write Unit Tests**: Test individual classes and methods in isolation using
   JUnit 5 + Mockito
   ```bash
   # Maven: run unit tests only
   mvn test

   # Gradle: run unit tests only
   ./gradlew test
   ```

2. **Write Integration Tests**: Test across multiple layers using
   `@SpringBootTest` with embedded databases or mock servers
   ```bash
   # Maven: run integration tests (typically failsafe plugin)
   mvn verify

   # Gradle: run integration tests
   ./gradlew integrationTest
   ```

3. **Run Single Module Tests**:
   ```bash
   # Maven: test a single module
   mvn test -pl <module-name>

   # Gradle: test a single module
   ./gradlew :<module-name>:test
   ```

4. **Run a Single Test Class**:
   ```bash
   # Maven
   mvn test -Dtest=MyTestClass -pl <module-name>

   # Gradle
   ./gradlew :<module-name>:test --tests "com.example.MyTestClass"
   ```

5. **Check Coverage**: Configure JaCoCo and verify thresholds
   ```bash
   # Maven
   mvn jacoco:report

   # Gradle
   ./gradlew jacocoTestReport
   ```

6. **Fix Failures**: Debug and fix all failures before pushing

### Deployment Flow

1. Code is merged to `main` via approved MR
2. CI pipeline runs full validation (compile, unit test, integration test, checkstyle)
3. Build deployable artifact:
   ```bash
   # Maven: package without running tests (CI already validated)
   mvn package -DskipTests

   # Gradle: build jar
   ./gradlew bootJar
   ```
4. Deploy to staging:
   ```bash
   java -jar target/app.jar --spring.profiles.active=staging
   ```
5. QA verification in staging environment
6. Production deployment via manual promotion:
   ```bash
   java -jar target/app.jar --spring.profiles.active=prod
   ```
7. Post-deployment smoke test

#### Rollback Procedure

1. Identify the deployment that introduced the issue
2. Revert the merge commit on `main`
3. Rebuild the previous known-good version:
   ```bash
   # Maven
   mvn package -DskipTests

   # Gradle
   ./gradlew bootJar
   ```
4. Deploy the previous artifact to staging and verify
5. Promote the verified rollback artifact to production
6. Document the incident and root cause
```
