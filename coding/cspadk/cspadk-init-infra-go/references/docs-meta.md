# Meta Documentation Templates

## 1. index.md — Documentation Center Homepage

| Field | Content |
|-------|---------|
| Purpose | Entry point for all documentation with organized links |
| Sections | 1) Welcome (one paragraph) 2) Quick Links 3) Documentation Index (by category) 4) External Resources |
| Content Guidelines | Welcome must be a single paragraph introducing the project documentation. Quick Links must provide shortcuts to the most frequently accessed documents. Documentation Index must organize all documents by category with relative path links. External Resources must list relevant external documentation URLs. |
| Quality Checks | Every doc file is linked in the index; All links use relative paths; Quick Links section exists; Welcome is a single paragraph |

### Example Structure

```markdown
# <Project Name> Documentation

Welcome to the <Project Name> documentation center. This site contains all technical documentation for the project, including architecture guides, development workflows, quality standards, and business context.

## Quick Links

- [Getting Started](development/getting-started.md) — Set up your environment and make your first contribution
- [Repository Constraints](constraints/repository-constraints.md) — Mandatory rules for this repository
- [Architecture Overview](architecture/overview.md) — How the system is structured
- [Testing Strategy](quality/testing-strategy.md) — Testing approach and coverage requirements

## Documentation Index

### Architecture & Design

- [Architecture Overview](architecture/overview.md) — Repository architecture, tech stack, and module dependencies
- [Module Index](architecture/module-index.md) — Module introductions and file directory index

### Constraints & Specifications

- [Repository Constraints](constraints/repository-constraints.md) — Dependency management, directory structure, CI/CD, security
- [Development Guidelines](constraints/development-guidelines.md) — Coding standards, branching, commits, releases
- [Constraint Validation](constraints/constraint-validation.md) — How to verify constraint compliance

### Development

- [Getting Started](development/getting-started.md) — Quick start guide for new contributors
- [Workflow](development/workflow.md) — Development workflow description

### Quality Assurance

- [Testing Strategy](quality/testing-strategy.md) — Testing approach, tooling, and coverage requirements
- [Code Review Guide](quality/code-review-guide.md) — Review standards and checklist

### Business Context

- [Background](business/background.md) — Business domain introduction
- [Processes](business/processes.md) — Core business process descriptions

## External Resources

- [Go Documentation](https://go.dev/doc/) — Go language reference
- [Kitex Documentation](https://www.cloudwego.io/docs/kitex/) — Kitex RPC framework reference
- [Hertz Documentation](https://www.cloudwego.io/docs/hertz/) — Hertz HTTP framework reference
- [golangci-lint](https://golangci-lint.run/) — Go lint tool reference
```

---

## 2. writing-guide.md — Documentation Writing Guide

| Field | Content |
|-------|---------|
| Purpose | Standards for writing and maintaining documentation |
| Sections | 1) Writing Principles 2) Formatting Standards (Markdown, headings, tables) 3) Language Policy 4) File Naming Convention (kebab-case) 5) Maintenance Guidelines |
| Content Guidelines | Writing principles must define the tone and style for all documentation. Formatting standards must specify Markdown conventions including heading levels, table formatting, and code blocks. Language policy must specify the default language and override rules. File naming must define the convention (kebab-case). Maintenance guidelines must describe how to keep documentation up to date. |
| Quality Checks | Language policy specifies default and override rules; File naming convention is documented; Formatting standards are defined; Maintenance guidelines are provided |

### Writing Principles Example

1. **Clarity over cleverness**: Write to be understood, not to impress
2. **Be concise**: Every sentence should earn its place
3. **Show, don't tell**: Use code examples and diagrams over prose descriptions
4. **Stay current**: Outdated documentation is worse than no documentation
5. **Link, don't duplicate**: Reference other docs instead of repeating content

### Formatting Standards Example

#### Headings

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

- Always specify the language for syntax highlighting (`go`, `bash`, `yaml`)
- Use inline code (`code`) for file paths, command names, and variable names
- Use fenced code blocks (```) for multi-line code examples

#### Links

- Use relative paths for internal documentation links
- Use full URLs for external resources
- Prefer reference-style links for repeated URLs

### Language Policy Example

- **Default language**: English (en)
- **Override rule**: Documentation may be written in the same language as the user's input when the skill is invoked. If the user writes in Chinese, produce documentation in Chinese. If unclear, default to English.
- **Code and commands**: Always in English regardless of document language
- **File names**: Always in English using kebab-case
- **Variable names and technical terms**: Always in English

### File Naming Convention Example

- Use **kebab-case** for all file names: `getting-started.md`, `testing-strategy.md`
- Do not use spaces, underscores, or camelCase in file names
- File names should be descriptive but concise (2-4 words)
- Use the pattern `<category>-<topic>.md` for clarity
- Examples:
  - `repository-constraints.md` (not `repoConstraints.md` or `repository_constraints.md`)
  - `getting-started.md` (not `GettingStarted.md`)
  - `code-review-guide.md` (not `code_review_guide.md`)

### Maintenance Guidelines Example

1. **Review on change**: When a module's code changes, verify its documentation is still accurate
2. **Annual audit**: Review all documentation at least once per year for accuracy
3. **Stale markers**: Add a "Last Updated" date to pages that change frequently
4. **Broken links**: Fix broken links immediately when discovered
5. **Template updates**: When updating a documentation template, notify module owners to update their docs accordingly
6. **Deletion policy**: Never delete documentation without a redirect or replacement. Mark as deprecated first.
