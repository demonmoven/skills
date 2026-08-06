# CSPADK Anti-Degradation Skill

A **generic** anti-degradation scanner for security, architecture, complexity, and test discipline. Repository-agnostic and adaptable to different tech stacks.

## Quick Start

Invoke this skill to scan any codebase:
- "Run anti-degradation scan on this repository"
- "Check for security and architecture issues"
- "Generate quality report for this project"

## Features

| Category | Checks |
|----------|--------|
| **Security** | Command injection, path traversal, sensitive data, input validation |
| **Architecture** | God objects, layering, type duplication, dead code, circular deps |
| **Complexity** | File size, function length, nesting depth, duplication, cyclomatic complexity |
| **Testing** | Coverage, organization, test quality |

## Output

Reports are saved to `docs/anti-degradation/` by default with format:
```
{project-name}-diagnostic-report-{date}.md
```

## Rule Summary

| Rule ID | Category | Severity | Description |
|---------|----------|----------|-------------|
| SEC-001 | Security | CRITICAL | Command injection prevention |
| SEC-002 | Security | HIGH | Path traversal prevention |
| SEC-003 | Security | MEDIUM | Sensitive data exposure |
| SEC-004 | Security | MEDIUM | Input validation gaps |
| ARCH-001 | Architecture | HIGH | God object prevention |
| ARCH-002 | Architecture | MEDIUM | Layering integrity |
| ARCH-003 | Architecture | MEDIUM | Type duplication |
| ARCH-004 | Architecture | LOW | Dead code elimination |
| ARCH-005 | Architecture | MEDIUM | Circular dependencies |
| COMP-001 | Complexity | MEDIUM | File size limits |
| COMP-002 | Complexity | MEDIUM | Function length limits |
| COMP-003 | Complexity | LOW | Nesting depth limits |
| COMP-004 | Complexity | MEDIUM | Code duplication |
| COMP-005 | Complexity | MEDIUM | Cyclomatic complexity |
| TEST-001 | Testing | MEDIUM | Test coverage |
| TEST-002 | Testing | LOW | Test organization |
| TEST-003 | Testing | LOW | Test quality |

## Tech Stack Support

| Stack | Security Patterns | Architecture Patterns |
|-------|-------------------|----------------------|
| TypeScript/JavaScript | execSync, eval, prototype pollution | Classes, modules, imports |
| Go | exec.Command, os.Open, filepath.Join | Packages, interfaces, imports |
| Java | Runtime.exec, FileInputStream | Classes, packages, imports |
| Python | os.system, subprocess, eval | Modules, classes, imports |

## Reference

- [Anti-Degradation Rules](./anti-degradation-rules.md) - Complete rule definitions with examples
