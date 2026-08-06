# Java CI Pipeline Configuration

## 1. Codebase CI Pipeline

### 1.1 Standard Pipeline Structure

Define in `.codebase/pipelines/java-ci.yaml`:

```yaml
name: java-ci
trigger:
  change:
    branches: [main]
  push:
    branches: [main]

jobs:
  lint:
    name: Lint
    image: hub.byted.org/codebase/ci_java17
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Checkstyle
        commands:
          - mvn checkstyle:check
      - name: SpotBugs
        commands:
          - mvn spotbugs:check

  test:
    name: Test
    image: hub.byted.org/codebase/ci_java17
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Run Tests
        commands:
          - mvn test
      - name: JaCoCo Report
        commands:
          - mvn jacoco:report

  build:
    name: Build
    image: hub.byted.org/codebase/ci_java17
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Build
        commands:
          - mvn package -DskipTests
```

### 1.2 Key Configuration Requirements

- **Trigger**: MR to main branch and push to main branch
- **Job Order**: lint → test → build (add `depends` if sequential execution needed)
- **Dependency Cache**: Recommend configuring Maven local repository cache
- **Coverage**: Test stage must output coverage report via JaCoCo

### 1.3 With Cache (Recommended)

```yaml
name: java-ci
trigger:
  change:
    branches: [main]
    paths: &paths
      - "src/**/*.java"
      - "pom.xml"
    patchset-paths: *paths

jobs:
  lint:
    name: Lint
    image: hub.byted.org/codebase/ci_java17
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache Maven
        uses: actions/cache
        inputs:
          path: ~/.m2/repository
          key: maven-${{ hashFiles('pom.xml') }}
      - name: Checkstyle
        commands:
          - mvn checkstyle:check
      - name: SpotBugs
        commands:
          - mvn spotbugs:check

  test:
    name: Test
    image: hub.byted.org/codebase/ci_java17
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache Maven
        uses: actions/cache
        inputs:
          path: ~/.m2/repository
          key: maven-${{ hashFiles('pom.xml') }}
      - name: Run Tests
        commands:
          - mvn test
      - name: JaCoCo Report
        commands:
          - mvn jacoco:report

  build:
    name: Build
    image: hub.byted.org/codebase/ci_java17
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache Maven
        uses: actions/cache
        inputs:
          path: ~/.m2/repository
          key: maven-${{ hashFiles('pom.xml') }}
      - name: Build
        commands:
          - mvn package -DskipTests
```

### 1.4 CI Images Reference

| Image | Base OS | Status |
|-------|---------|--------|
| hub.byted.org/codebase/ci_java17 | Debian 11 | ✅ Recommended (aligns with JDK 17+ requirement) |
| hub.byted.org/codebase/ci_java11 | Debian 10 | ⚠️ Legacy only (not compliant with JDK 17+ requirement) |
| hub.byted.org/codebase/ci_java8 | - | ⚠️ Legacy only |
| hub.byted.org/codebase/ci_java8_oracle | - | ⚠️ Legacy only |

**Note**: CI images come with internal Maven mirror pre-configured.

## 1.5 Gradle CI Pipeline Variant

For projects using Gradle (`build.gradle` or `build.gradle.kts`):

```yaml
name: java-ci-gradle
trigger:
  change:
    branches: [main]
    paths: &paths
      - "src/**/*.java"
      - "build.gradle"
      - "build.gradle.kts"
    patchset-paths: *paths
  push:
    branches: [main]

jobs:
  lint:
    name: Lint
    image: hub.byted.org/codebase/ci_java17
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache Gradle
        uses: actions/cache
        inputs:
          path: .gradle/
          key: gradle-${{ hashFiles('build.gradle', 'build.gradle.kts') }}
      - name: Checkstyle
        commands:
          - ./gradlew checkstyleMain checkstyleTest
      - name: SpotBugs
        commands:
          - ./gradlew spotbugsMain

  test:
    name: Test
    image: hub.byted.org/codebase/ci_java17
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache Gradle
        uses: actions/cache
        inputs:
          path: .gradle/
          key: gradle-${{ hashFiles('build.gradle', 'build.gradle.kts') }}
      - name: Run Tests
        commands:
          - ./gradlew test jacocoTestReport
      - name: JaCoCo Report
        uses: actions/artifact
        inputs:
          name: coverage-report
          path: build/reports/jacoco/test/jacocoTestReport.xml

  build:
    name: Build
    image: hub.byted.org/codebase/ci_java17
    steps:
      - name: Checkout
        uses: actions/checkout
      - name: Cache Gradle
        uses: actions/cache
        inputs:
          path: .gradle/
          key: gradle-${{ hashFiles('build.gradle', 'build.gradle.kts') }}
      - name: Build
        commands:
          - ./gradlew build -x test
```

**Key differences from Maven variant**:
- Uses `./gradlew` wrapper instead of `mvn` commands
- Cache path: `.gradle/` instead of `~/.m2/repository`
- Checkstyle via `./gradlew checkstyleMain checkstyleTest`
- JaCoCo via `./gradlew jacocoTestReport`
- Build with `./gradlew build -x test`

## 2. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| CI Platform | Codebase Pipeline | GitHub Actions | Non-ByteDance projects |
| JDK Image | ci_java17 | ci_java11 | Legacy projects requiring JDK 11 |
| Build Tool | Maven | Gradle | Polyglot projects or Kotlin DSL preference |
| Dependency Cache | Maven local repo cache | Gradle cache | Match build tool |

Priority: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 3. Auto-Completion Procedure

### CI Pipeline Auto-Completion

1. Check for existing YAML files under `.codebase/pipelines/` → CI exists
2. If CI exists, validate it includes lint/test/build stages → report any gaps
3. If no CI → detect build tool:
   - Check for `pom.xml` → Maven project
   - Check for `build.gradle` or `build.gradle.kts` → Gradle project
4. Generate appropriate pipeline YAML from template:
   - Maven: Use Section 1.3 template with ci_java17 image
   - Gradle: Use Section 1.5 template with ci_java17 image
5. Validate YAML syntax
6. Acceptance criteria:
   - [ ] `.codebase/pipelines/` contains Java CI configuration file
   - [ ] Pipeline includes lint, test, build stages
   - [ ] CI image is ci_java17 (or ci_java11 for legacy projects)
   - [ ] YAML syntax is valid

## 2. Verification Checklist

- [ ] `.codebase/pipelines/` contains Java CI configuration file
- [ ] Pipeline includes lint, test, build stages
- [ ] Test stage configured with JaCoCo coverage output
- [ ] MR trigger condition configured
- [ ] Using correct CI image (`ci_java17` recommended, `ci_java11` for legacy)
