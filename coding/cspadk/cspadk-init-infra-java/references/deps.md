# Java Dependency Management Best Practices

## 1. Build Tool Selection

CSP Java projects support both Maven and Gradle, with Maven as the preferred option.

### 1.1 Maven (pom.xml)

```xml
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.bytedance.csp</groupId>
  <artifactId>csp-xxx-service</artifactId>
  <version>1.0.0-SNAPSHOT</version>

  <properties>
    <java.version>17</java.version>
    <maven.compiler.source>17</maven.compiler.source>
    <maven.compiler.target>17</maven.compiler.target>
    <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
  </properties>
</project>
```

### 1.2 Gradle (build.gradle)

```groovy
plugins {
    id 'java'
    id 'jacoco'
}

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}
```

## 2. Dependency Version Management

### 2.1 Unified Version Management

- Maven uses `<dependencyManagement>` or BOM for unified version management
- Gradle uses version catalogs (`gradle/libs.versions.toml`) or platform dependencies

### 2.2 Internal Repository Source

```xml
<repositories>
  <repository>
    <id>bytedance-maven</id>
    <url>https://maven.bytedance.net/repository/maven-public/</url>
  </repository>
</repositories>
```

### 2.3 Common Internal Dependencies

| Dependency | Purpose | Version Management |
|------------|---------|-------------------|
| `com.bytedance:kitex` | RPC Framework | Via BOM |
| `com.bytedance:hertz` | HTTP Framework | Via BOM |
| `com.bytedance:csp-common` | CSP Common Library | Via BOM |

## 3. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| Build Tool | Maven | Gradle | Polyglot projects or Kotlin DSL preference |
| JDK Version | 17 (LTS) | 21 (LTS) | New greenfield projects |
| Dependency Management | BOM (Spring Boot) | `<dependencyManagement>` | Non-Spring-Boot projects |
| Internal Repo | maven.bytedance.net | Artifactory proxy | When maven.bytedance.net is unreachable |

Priority: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 4. Auto-Completion Procedure

### Dependency Management Auto-Completion

1. Check for `pom.xml` → Maven project exists
2. Check for `build.gradle` or `build.gradle.kts` → Gradle project exists
3. If Maven project:
   - Verify JDK >= 17 in `<properties>`
   - Verify `<dependencyManagement>` or BOM exists
   - Verify internal repository configured
   - If any missing → add to pom.xml
4. If Gradle project:
   - Verify `sourceCompatibility = JavaVersion.VERSION_17`
   - Verify version catalog or platform dependencies
   - Verify internal repository configured
   - If any missing → add to build.gradle
5. If neither → full auto-completion:
   - Create `pom.xml` from template (groupId=com.bytedance.csp, JDK 17)
   - Configure internal repository
   - Add Spring Boot BOM or `<dependencyManagement>`
   - Run: `mvn dependency:resolve` → must exit 0
6. Acceptance criteria:
   - [ ] Build file exists with valid XML/Groovy syntax
   - [ ] JDK >= 17 configured
   - [ ] Dependency versions unified (BOM or dependencyManagement)
   - [ ] Internal repository configured
   - [ ] `mvn dependency:resolve` or `gradle dependencies` exits 0
