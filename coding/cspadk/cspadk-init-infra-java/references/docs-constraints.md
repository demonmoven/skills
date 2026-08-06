# Constraints Documentation Templates

## 1. constraints/repository-constraints.md — Repository Constraint Specifications

| Field | Content |
|-------|---------|
| Purpose | Document mandatory low-level constraints: dependency management, directory structure, CI/CD, and security |
| Sections | 1) Dependency Management 2) Directory Structure Rules 3) CI/CD Requirements 4) Security Constraints 5) Repository-Specific Rules |
| Content Guidelines | Every constraint must be prefixed with MUST / MUST NOT / SHOULD. Each constraint must include a rationale explaining why it exists. Each constraint must provide a validation command that can be run to check compliance. |
| Quality Checks | Every constraint has a MUST/MUST NOT/SHOULD prefix; Rationale is provided for each constraint; Validation command is provided for each constraint |

### Java-Specific Constraint Examples

```markdown
### Dependency Management

- **MUST** use JDK 17 or higher. Rationale: JDK 17 is the current LTS release and is required for modern Java features, security patches, and internal platform compatibility. Validation: `java -version`
- **MUST** use BOM (Bill of Materials) or `<dependencyManagement>` in `pom.xml` for version unification. Rationale: Centralized version management prevents dependency conflicts and ensures consistent versions across modules. Validation: `grep -r '<dependencyManagement>' pom.xml || grep -r '<bom>' pom.xml`
- **MUST** configure internal Maven repository (maven.bytedance.net) in `settings.xml` or `pom.xml`. Rationale: Internal artifacts and patched libraries are only available through the corporate Maven repository. Validation: `grep 'maven.bytedance.net' ~/.m2/settings.xml pom.xml`
- **MUST NOT** use SNAPSHOT dependencies in release builds. Rationale: SNAPSHOT versions are mutable and can cause non-reproducible builds, leading to unpredictable behavior in production. Validation: `grep -r 'SNAPSHOT' pom.xml`
- **SHOULD** use `maven-enforcer-plugin` to enforce JDK version and dependency convergence. Rationale: Automated enforcement prevents accidental use of incompatible JDK versions or conflicting transitive dependencies. Validation: `grep 'maven-enforcer-plugin' pom.xml`

### Directory Structure Rules

- **MUST** follow the standard Maven directory layout (`src/main/java`, `src/main/resources`, `src/test/java`). Rationale: Standard layout ensures compatibility with Maven plugins, IDEs, and team expectations. Validation: `test -d src/main/java && test -d src/test/java && echo PASS || echo FAIL`
- **MUST NOT** commit IDE configuration files (`.idea/`, `*.iml`, `.vscode/`, `.classpath`, `.project`). Rationale: IDE files are user-specific and cause merge conflicts; they should be generated locally or managed by IDE. Validation: `git ls-files | grep -E '\.idea/|\.iml$|\.vscode/|\.classpath|\.project' && echo FAIL || echo PASS`
- **MUST** include a `.gitignore` that excludes IDE files, build output (`target/`), and OS files (`.DS_Store`). Rationale: Prevents accidentally committing generated or user-specific files. Validation: `grep -E 'target/|\.idea/|\.iml' .gitignore`

### CI/CD Requirements

- **MUST** include compile, test, and package stages in CI pipeline. Rationale: Prevents merging code that fails compilation or tests. Validation: `grep -E 'compile|test|package' .gitlab-ci.yml Jenkinsfile .github/workflows/*.yml`
- **MUST** cache `.m2/repository` in CI to reduce build times. Rationale: Downloading dependencies on every CI run is slow and wastes network resources. Validation: `grep -r '\.m2' .gitlab-ci.yml Jenkinsfile .github/workflows/*.yml`
- **SHOULD** run `mvn spotbugs:check` and `mvn checkstyle:check` in CI. Rationale: Automated static analysis catches bugs and style violations before code review. Validation: `grep -E 'spotbugs|checkstyle' .gitlab-ci.yml Jenkinsfile .github/workflows/*.yml`

### Security Constraints

- **MUST NOT** commit secrets, tokens, API keys, or database credentials to the repository. Rationale: Prevents credential leakage through source control. Validation: `git log --all --diff-filter=A -- '*.env' '*.key' '*.pem' '*.jks' '*.p12'`
- **MUST** use environment variables or a secrets manager for all sensitive configuration. Rationale: Separates secrets from code and enables per-environment configuration. Validation: `grep -r 'password\s*=\s*\"' src/main/resources/ && echo FAIL || echo PASS`
- **MUST NOT** use hardcoded database URLs, API endpoints, or service addresses. Rationale: Configuration must be externalized for environment-specific deployments. Validation: `grep -rE '(jdbc|http)://[a-zA-Z0-9.-]+\.(com|net|org|io)' src/main/ && echo FAIL || echo PASS`

### Repository-Specific Rules

- **MUST** include `LICENSE` and `NOTICE` files in the repository root. Rationale: Legal compliance requires clear licensing information. Validation: `test -f LICENSE && test -f NOTICE && echo PASS || echo FAIL`
- **SHOULD** use multi-module Maven structure for projects with more than one deployable artifact. Rationale: Multi-module builds enforce consistent versions and enable selective builds. Validation: `grep '<modules>' pom.xml`
- **MUST** declare all direct dependencies explicitly; do not rely on transitive dependencies. Rationale: Transitive dependencies can change without notice, breaking the build. Validation: `mvn dependency:analyze | grep 'Used undeclared dependencies'`
```

---

## 2. constraints/development-guidelines.md — Development & Version Control Guidelines

| Field | Content |
|-------|---------|
| Purpose | Coding standards, branching strategy, commit conventions, and release process |
| Sections | 1) Coding Standards (with examples) 2) Branching Strategy (naming format) 3) Commit Conventions (message template) 4) Release Process (includes rollback) |
| Content Guidelines | Coding standards must include concrete code examples. Branch naming must define the exact format with examples. Commit conventions must provide a message template. Release process must include rollback procedure. |
| Quality Checks | Coding standards have code examples; Branch naming format is defined; Commit message template is provided; Rollback procedure is documented |

### Java Coding Standards Example

```markdown
### Java Coding Standards

- Use `Optional` instead of returning `null` for methods that may not have a result
- Prefer composition over inheritance for sharing behavior between classes
- Use SLF4J for logging; never use `System.out.println` or `System.err.println` in production code
- Use `final` for variables, parameters, and fields that are not reassigned
- Prefer immutable objects; use `record` types (JDK 17+) for data carriers

Example:

```java
// ✅ Preferred — Optional for nullable returns
public Optional<User> findUser(String id) {
    return userRepository.findById(id);
}

// ❌ Avoid — null returns require callers to remember null checks
public User findUser(String id) {
    return userRepository.findById(id); // may return null
}
```

```java
// ✅ Preferred — composition with interface delegation
public class OrderProcessor {
    private final PaymentService paymentService;
    private final InventoryService inventoryService;

    public OrderProcessor(PaymentService paymentService, InventoryService inventoryService) {
        this.paymentService = paymentService;
        this.inventoryService = inventoryService;
    }
}

// ❌ Avoid — inheritance for code reuse
public class OrderProcessor extends PaymentService {
    // couples OrderProcessor to PaymentService implementation
}
```

```java
// ✅ Preferred — SLF4J logging
private static final Logger log = LoggerFactory.getLogger(OrderService.class);

public void processOrder(Order order) {
    log.info("Processing order id={}", order.getId());
}

// ❌ Avoid — System.out in production
public void processOrder(Order order) {
    System.out.println("Processing order " + order.getId());
}
```

```java
// ✅ Preferred — record for immutable data carrier (JDK 17+)
public record UserDto(String id, String name, String email) {}

// ❌ Avoid — mutable DTO with getters and setters
public class UserDto {
    private String id;
    private String name;
    private String email;
    // getters and setters...
}
```
```

### Branch Naming Example

```markdown
### Branch Naming Convention

Format: `<type>/<ticket-id>-<short-description>`

Types: `feat/`, `fix/`, `chore/`, `refactor/`, `docs/`, `test/`, `ci/`

Examples:
- `feat/cspadk-123-add-skill-pull`
- `fix/cspadk-456-null-pointer-exception`
- `chore/cspadk-789-upgrade-spring-boot`
- `refactor/cspadk-101-consolidate-dto`
- `docs/cspadk-202-api-reference`
```

### Commit Message Template

```markdown
### Commit Conventions

Format:

```
<type>(<scope>): <subject>

<body>

Co-Authored-By: ...
```

Types: `feat`, `fix`, `chore`, `refactor`, `docs`, `test`, `ci`

Scopes: module or package name (e.g., `api`, `core`, `infra`, `client`)

Example:

```
feat(api): add user lookup endpoint

Implement GET /api/v1/users/{id} with Optional return type.
Caches results in Redis with 5-minute TTL.

Co-Authored-By: Developer <dev@example.com>
```
```

### Release Process Example

```markdown
### Release Process

1. Create release branch from `main`: `release/v<version>`
2. Run full test suite: `mvn clean verify`
3. Update version in all `pom.xml` files: `mvn versions:set -DnewVersion=<version>`
4. Commit version bump: `git commit -m "chore: bump version to <version>"`
5. Merge release branch to `main` via MR
6. Tag the merge commit: `git tag v<version>`
7. Deploy artifact: `mvn deploy -P release`

### Rollback Procedure

1. Identify the problematic commit or release tag
2. Create a revert branch from `main`: `fix/revert-v<version>`
3. Run `git revert <commit-sha>` on the revert branch
4. Bump version for hotfix: `mvn versions:set -DnewVersion=<next-patch>`
5. Merge revert MR with fast-track review
6. Deploy hotfix artifact: `mvn deploy -P release`
7. Verify rollback in staging before production deploy
```

---

## 3. constraints/constraint-validation.md — Constraint Validation Guide

| Field | Content |
|-------|---------|
| Purpose | Checklist and commands to validate constraint compliance |
| Sections | 1) Automated Checks (exact commands) 2) Manual Checklist 3) CI Validation 4) Remediation Guide |
| Content Guidelines | Automated checks must provide exact copy-paste commands. Manual checklist must cover constraints not easily automated. CI validation must describe how the pipeline enforces constraints. Remediation guide must provide step-by-step fixes for common violations. |
| Quality Checks | Every constraint has a validation step; Exact commands are provided; Remediation guide covers common violations |

### Automated Checks Example

```markdown
### Dependency Management Validation

| Check | Command | Expected Result |
|-------|---------|-----------------|
| JDK 17+ installed | `java -version 2>&1 \| grep -oP 'version "17\|18\|19\|20\|21'` | Match found |
| BOM or dependencyManagement configured | `grep -r '<dependencyManagement>' pom.xml \|\| grep -r '<bom>' pom.xml` | Match found |
| Internal Maven repo configured | `grep 'maven.bytedance.net' ~/.m2/settings.xml pom.xml` | Match found |
| No SNAPSHOT in release | `grep -r 'SNAPSHOT' pom.xml && echo FAIL \|\| echo PASS` | PASS |
| Enforcer plugin configured | `grep 'maven-enforcer-plugin' pom.xml` | Match found |

### Code Quality Validation

| Check | Command | Expected Result |
|-------|---------|-----------------|
| Code style passes | `mvn checkstyle:check` | BUILD SUCCESS |
| Static analysis passes | `mvn spotbugs:check` | BUILD SUCCESS |
| All tests pass | `mvn test` | BUILD SUCCESS |
| Dependency constraints enforced | `mvn enforcer:enforce` | BUILD SUCCESS |
| No undeclared dependencies | `mvn dependency:analyze` | No "Used undeclared" warnings |
| Gradle equivalent (if applicable) | `./gradlew check` | BUILD SUCCESS |

### Directory Structure Validation

| Check | Command | Expected Result |
|-------|---------|-----------------|
| Maven layout exists | `test -d src/main/java && test -d src/test/java && echo PASS \|\| echo FAIL` | PASS |
| No IDE files tracked | `git ls-files \| grep -E '\.idea/\|\.iml$\|\.vscode/\|\.classpath\|\.project' \|\| echo PASS` | PASS |
| .gitignore covers target/ and IDE | `grep -E 'target/\|\.idea/\|\.iml' .gitignore` | Match found |

### Security Validation

| Check | Command | Expected Result |
|-------|---------|-----------------|
| No committed secrets | `git log --all --diff-filter=A -- '*.env' '*.key' '*.pem' '*.jks' '*.p12' \|\| echo PASS` | PASS |
| No hardcoded passwords | `grep -r 'password\s*=\s*\"' src/main/resources/ \|\| echo PASS` | PASS |
| No hardcoded URLs | `grep -rE '(jdbc\|http)://[a-zA-Z0-9.-]+\.(com\|net\|org\|io)' src/main/ \|\| echo PASS` | PASS |
```

### Manual Checklist Example

```markdown
### Manual Checklist

- [ ] No hardcoded secrets, API keys, or database credentials in source code
- [ ] CI pipeline includes compile, test, checkstyle, and spotbugs stages
- [ ] Environment variables or secrets manager used for all sensitive configuration
- [ ] Branch naming follows `<type>/<ticket-id>-<description>` format
- [ ] Commit messages follow the conventional commit template
- [ ] All direct dependencies are explicitly declared in pom.xml
- [ ] JDK version is 17 or higher in local and CI environments
- [ ] Internal Maven repository (maven.bytedance.net) is configured
- [ ] No SNAPSHOT dependencies in release builds
- [ ] .gitignore excludes target/, IDE files, and OS files
- [ ] LICENSE and NOTICE files are present
```

### CI Validation Example

```markdown
### CI Validation

The CI pipeline enforces constraints at merge request time:

1. **Compile Stage**: Runs `mvn compile` — fails on compilation errors
2. **Checkstyle Stage**: Runs `mvn checkstyle:check` — fails on code style violations
3. **SpotBugs Stage**: Runs `mvn spotbugs:check` — fails on static analysis findings
4. **Test Stage**: Runs `mvn test` — fails on test failures
5. **Enforcer Stage**: Runs `mvn enforcer:enforce` — fails on dependency constraint violations
6. **Package Stage**: Runs `mvn package -DskipTests` — fails on packaging errors
7. **Security Scan Stage**: Runs dependency vulnerability check — fails on known vulnerabilities
```

### Remediation Guide Example

```markdown
### Common Violations and Remediation

#### SNAPSHOT dependency found in pom.xml

1. Identify the SNAPSHOT dependency: `grep -n 'SNAPSHOT' pom.xml`
2. Replace the SNAPSHOT version with a released version
3. If no released version exists, contact the library maintainers for a release
4. Run `mvn dependency:resolve` to verify the dependency resolves
5. Verify: `grep -r 'SNAPSHOT' pom.xml || echo PASS`

#### IDE files committed to the repository

1. Remove tracked IDE files: `git rm -r --cached .idea/ *.iml .classpath .project`
2. Add them to `.gitignore`:
   ```
   echo '.idea/' >> .gitignore
   echo '*.iml' >> .gitignore
   echo '.classpath' >> .gitignore
   echo '.project' >> .gitignore
   ```
3. Commit the changes: `git commit -m "chore: remove IDE files and update .gitignore"`
4. Verify: `git ls-files | grep -E '\.idea/|\.iml$|\.classpath|\.project' || echo PASS`

#### checkstyle:check fails

1. Run checkstyle with verbose output: `mvn checkstyle:check -Dcheckstyle.consoleOutput=true`
2. Review the violations in the console output or `target/checkstyle-result.xml`
3. Fix each violation according to the project's `checkstyle.xml` rules
4. Common fixes:
   - Missing Javadoc: add `/** */` comments to public classes and methods
   - Line too long: break lines or configure `lineLength` in `checkstyle.xml`
   - Import order: run `mvn checkstyle:check` and reorder imports as indicated
5. Verify: `mvn checkstyle:check`

#### spotbugs:check fails

1. Run SpotBugs with report: `mvn spotbugs:spotbugs`
2. Open the report at `target/spotbugsXml.xml` or `target/site/spotbugs.html`
3. Fix each bug according to the SpotBugs message:
   - NP_NULL_ON_SOME_PATH: add null checks or use `Optional`
   - RCN_REDUNDANT_NULLCHECK: remove redundant null checks
   - DM_NUMBER_CTOR: use `valueOf()` instead of constructor for wrapper types
4. Verify: `mvn spotbugs:check`

#### enforcer:enforce fails

1. Run enforcer with verbose output: `mvn enforcer:enforce -X`
2. Common failures:
   - JDK version mismatch: install or configure JDK 17+ and set `JAVA_HOME`
   - Dependency convergence: add `<dependencyManagement>` entries to resolve version conflicts
   - Banned dependencies: remove the banned dependency or add an exclusion
3. Verify: `mvn enforcer:enforce`

#### Hardcoded password in configuration

1. Identify the hardcoded value: `grep -rn 'password\s*=\s*"' src/main/resources/`
2. Replace with a placeholder: `password=${DB_PASSWORD}`
3. Add the environment variable to the deployment configuration
4. Add the property to `application.yml` with a default for local development:
   ```yaml
   spring:
     datasource:
       password: ${DB_PASSWORD:dev-password}
   ```
5. Verify: `grep -r 'password\s*=\s*"' src/main/resources/ || echo PASS`
```
