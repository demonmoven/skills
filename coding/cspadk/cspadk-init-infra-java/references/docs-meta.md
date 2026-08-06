# Meta Documentation Templates

## 1. index.md — Documentation Center Homepage

| Field | Content |
|-------|---------|
| Purpose | Entry point for all documentation with organized links |
| Sections | 1) Welcome (one paragraph) 2) Quick Links 3) Documentation Index (by category) 4) External Resources |
| Content Guidelines | Welcome paragraph must be concise. Quick Links must list the 3-5 most accessed docs. Documentation Index must cover all categories (Architecture, Constraints, Development, Quality, Business, Modules). External Resources must include relevant internal links. |
| Quality Checks | Every doc file is linked; Relative paths used; Quick links section exists; All categories covered |

### Example Structure

```markdown
# <Project Name> Documentation

Welcome to the <Project Name> documentation center. This site contains all technical documentation for the project, including architecture guides, development workflows, quality standards, and business context.

## Quick Links

- [Getting Started](development/getting-started.md) — Set up your environment and make your first contribution
- [Repository Constraints](constraints/repository-constraints.md) — Mandatory rules for this repository
- [Architecture Overview](architecture/overview.md) — How the system is structured
- [Testing Strategy](quality/testing-strategy.md) — Testing approach and coverage requirements
- [Module Index](architecture/module-index.md) — Module introductions and file directory index

## Documentation Index

### Architecture

- [Architecture Overview](architecture/overview.md) — Repository architecture, tech stack, and module dependencies
- [Module Index](architecture/module-index.md) — Module introductions and file directory index

### Constraints

- [Repository Constraints](constraints/repository-constraints.md) — Dependency management, directory structure, CI/CD, security
- [Development Guidelines](constraints/development-guidelines.md) — Coding standards, branching, commits, releases
- [Constraint Validation](constraints/constraint-validation.md) — How to verify constraint compliance

### Development

- [Getting Started](development/getting-started.md) — Quick start guide for new contributors
- [Workflow](development/workflow.md) — Development workflow description

### Quality

- [Testing Strategy](quality/testing-strategy.md) — Testing approach, tooling, and coverage requirements
- [Code Review Guide](quality/code-review-guide.md) — Review standards and checklist

### Business

- [Background](business/background.md) — Business domain introduction
- [Processes](business/processes.md) — Core business process descriptions

### Modules

- [Module A](modules/module-a.md) — Module A description and API reference
- [Module B](modules/module-b.md) — Module B description and API reference

## External Resources

- [Spring Boot Documentation](https://spring.io/projects/spring-boot) — Spring Boot framework reference
- [Maven Documentation](https://maven.apache.org/guides/) — Maven build tool reference
- [JUnit 5 Documentation](https://junit.org/junit5/docs/current/user-guide/) — JUnit 5 testing framework reference
```

---

## 2. writing-guide.md — Documentation Writing Guide

| Field | Content |
|-------|---------|
| Purpose | Standards for writing and maintaining documentation |
| Sections | 1) Writing Principles 2) Formatting Standards (Markdown, headings, tables) 3) Language Policy 4) File Naming Convention (kebab-case) 5) Maintenance Guidelines |
| Content Guidelines | Writing principles should emphasize clarity, consistency, and conciseness. Formatting standards should cover Markdown conventions (ATX headings, table formatting, code blocks). Language policy should specify: default language is English; Chinese may be used for business-domain docs when team primarily communicates in Chinese; code comments and variable names must always be in English. File naming must be kebab-case. Maintenance guidelines should cover review frequency and update triggers. |
| Quality Checks | Language policy specifies default and override rules; File naming convention documented; Writing principles cover clarity and consistency |

### Writing Principles Example

```markdown
### Writing Principles

1. **Clarity over cleverness**: Write to be understood, not to impress
2. **Consistency**: Use the same terminology and formatting conventions throughout all documentation
3. **Conciseness**: Every sentence should earn its place; remove unnecessary words
4. **Show, don't tell**: Use code examples and diagrams over prose descriptions
5. **Stay current**: Outdated documentation is worse than no documentation
6. **Link, don't duplicate**: Reference other docs instead of repeating content
```

### Formatting Standards Example

```markdown
### Formatting Standards

#### Headings

- Use ATX-style headings (`#`, `##`, `###`) — do not use Setext-style (underlined) headings
- Use `#` for the page title (only one per file)
- Use `##` for major sections
- Use `###` for subsections
- Do not skip heading levels (e.g., `#` to `###`)

#### Tables

- Always include header rows
- Align columns for readability
- Use tables for structured data (config params, entities, checklists)
- Use prose for narrative explanations

#### Code Blocks

- Always specify the language for syntax highlighting
- Use inline code (`code`) for file paths, command names, and variable names
- Use fenced code blocks (```) for multi-line code examples
```

### Language Policy Example

```markdown
### Language Policy

- **Default language**: English (en)
- **Override rule**: Chinese may be used for business-domain documentation when the team primarily communicates in Chinese (e.g., business process descriptions, domain glossaries)
- **Code comments**: Always in English regardless of document language
- **Variable names and technical terms**: Always in English regardless of document language
- **File names**: Always in English using kebab-case
- **When in doubt**: Default to English
```

### File Naming Convention Example

```markdown
### File Naming Convention

- Use **kebab-case** for all file names: `getting-started.md`, `testing-strategy.md`
- Do not use spaces, underscores, or camelCase in file names
- File names should be descriptive but concise (2-4 words)
- Use the pattern `<category>-<topic>.md` for clarity
- Examples:
  - `repository-constraints.md` (not `repoConstraints.md` or `repository_constraints.md`)
  - `getting-started.md` (not `GettingStarted.md`)
  - `code-review-guide.md` (not `code_review_guide.md`)
```

### Maintenance Guidelines Example

```markdown
### Maintenance Guidelines

1. **Review on change**: When a module's code changes, verify its documentation is still accurate
2. **Review frequency**: Review all documentation at least once per quarter for accuracy
3. **Update triggers**: Update documentation when APIs change, new modules are added, dependencies are upgraded, or architecture decisions are made
4. **Stale markers**: Add a "Last Updated" date to pages that change frequently
5. **Broken links**: Fix broken links immediately when discovered
6. **Template updates**: When updating a documentation template, notify module owners to update their docs accordingly
7. **Deletion policy**: Never delete documentation without a redirect or replacement. Mark as deprecated first.
```
