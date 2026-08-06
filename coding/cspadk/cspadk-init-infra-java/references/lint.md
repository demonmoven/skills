# Java Lint Rules Configuration

## 1. Checkstyle

CSP Java projects use Checkstyle as the unified code style checker.

### 1.1 Standard Configuration (checkstyle.xml)

Key rules:
- Code indentation: 4 spaces
- Line length limit: 120 characters
- Import order: third-party → JDK → project internal
- No wildcard imports
- Classes/methods must have Javadoc
- Opening brace on same line (K&R style)

### 1.2 Maven Integration

```xml
<plugin>
  <groupId>org.apache.maven.plugins</groupId>
  <artifactId>maven-checkstyle-plugin</artifactId>
  <version>3.3.1</version>
  <configuration>
    <configLocation>checkstyle.xml</configLocation>
    <failOnViolation>true</failOnViolation>
  </configuration>
</plugin>
```

## 2. SpotBugs

For static analysis to detect potential bugs.

### 2.1 Maven Integration

```xml
<plugin>
  <groupId>com.github.spotbugs</groupId>
  <artifactId>spotbugs-maven-plugin</artifactId>
  <version>4.8.3.0</version>
  <configuration>
    <effort>Max</effort>
    <threshold>Low</threshold>
  </configuration>
</plugin>
```

## 3. PMD (Optional)

For code complexity and best practices checking.

## 4. checkstyle.xml Template

Complete CSP standard Checkstyle rules configuration:

```xml
<?xml version="1.0"?>
<!DOCTYPE module PUBLIC
    "-//Checkstyle//DTD Checkstyle Configuration 1.3//EN"
    "https://checkstyle.org/dtds/configuration_1_3.dtd">
<module name="Checker">
  <property name="charset" value="UTF-8"/>
  <property name="severity" value="warning"/>

  <module name="FileTabCharacter">
    <property name="eachLine" value="true"/>
  </module>

  <module name="TreeWalker">
    <module name="Indentation">
      <property name="basicOffset" value="4"/>
    </module>
    <module name="LineLength">
      <property name="max" value="120"/>
    </module>
    <module name="ImportOrder">
      <property name="groups" value="/^java\./,/^javax\./,/^com\.bytedance\./,/^org\./,*"/>
      <property name="ordered" value="true"/>
      <property name="separated" value="true"/>
      <property name="option" value="top"/>
    </module>
    <module name="AvoidStarImport"/>
    <module name="LeftCurly">
      <property name="option" value="eol"/>
    </module>
    <module name="RightCurly">
      <property name="option" value="same"/>
    </module>
    <module name="JavadocMethod">
      <property name="accessModifiers" value="public,protected"/>
    </module>
    <module name="JavadocType">
      <property name="scope" value="public"/>
    </module>
  </module>
</module>
```

## 5. Tech Stack Recommendations

| Category | Recommended | Alternative | When to Use Alternative |
|----------|-------------|-------------|------------------------|
| Code Style | Checkstyle 10.x | PMD | When rule complexity analysis is needed |
| Static Analysis | SpotBugs 4.8+ | FindSecBugs | When security-focused analysis is needed |
| Build Integration (Maven) | maven-checkstyle-plugin 3.3.1 | — | — |
| Build Integration (Gradle) | Spotless plugin | Checkstyle Gradle plugin | When formatting-focused lint is preferred |

Priority: Keep current repo's existing choice. Only recommend alternatives when capability is missing.

## 6. Auto-Completion Procedure

### Checkstyle Auto-Completion

1. Check for `checkstyle.xml` or `src/main/resources/checkstyle.xml` → config exists
2. Check for Maven checkstyle plugin in `pom.xml` → plugin configured
3. Check for Gradle Spotless or Checkstyle plugin in `build.gradle` → plugin configured
4. If config exists but no plugin → add plugin to build file
5. If no config → full auto-completion:
   - Create `checkstyle.xml` with CSP standard rules (see Section 4 template)
   - Add Maven checkstyle plugin to `pom.xml` (version 3.3.1, `failOnViolation=true`)
   - Or add Gradle Checkstyle plugin configuration to `build.gradle`
   - Run: `mvn checkstyle:check` or `./gradlew checkstyleMain` → must exit 0
6. Acceptance criteria:
   - [ ] checkstyle.xml exists with valid XML syntax
   - [ ] Maven or Gradle checkstyle plugin configured in build file
   - [ ] `mvn checkstyle:check` or `./gradlew checkstyleMain` exits 0
