---
name: writing-skills
description: Use when creating new skills, editing existing skills, or documenting reusable patterns for the gdpa_skills project. Provides structure, conventions, and quality guidelines for skill authoring.
---

# Writing Skills

## Overview

Skills are reusable reference guides for proven techniques, patterns, or tools. This skill defines conventions for creating skills in the gdpa_skills project.

**Core principle:** Write skills that help future Claude instances find and apply effective approaches. Focus on searchability and clarity.

## When to Use

- Creating a new skill for a technique or pattern
- Documenting tool usage or API patterns
- Organizing reusable knowledge across projects

**Don't use for:**
- One-off solutions
- Project-specific conventions (use CLAUDE.md instead)
- Standard practices well-documented elsewhere

## Directory Structure

```
gdpa_skills/
  category-name/           # Group related skills
    skill-name/
      SKILL.md             # Main reference (required)
      supporting-file.*    # Only if needed
```

## SKILL.md Structure

### Frontmatter (Required)

```yaml
---
name: skill-name-with-hyphens
description: Use when [specific triggering conditions]. [Context about the skill].
---
```

**Rules:**
- `name`: Letters, numbers, hyphens only (no special chars)
- `description`: Start with "Use when...", describe triggers not process
- Max 1024 characters total

### Sections

```markdown
# Skill Name

## Overview
Core principle in 1-2 sentences.

## When to Use
- Bullet list with symptoms and use cases
- When NOT to use

## Instructions
Step-by-step guidance or core patterns.

## Examples
Concrete, runnable examples with comments.

## Related Resources
Links to documentation, tools, references.
```

## Writing Guidelines

### 1. Optimize for Search (CSO)

Future Claude needs to FIND your skill:

- **Description triggers:** Specific symptoms, error messages, use cases
- **Keywords:** Include error messages, tool names, symptoms
- **Clear naming:** Use verb-first active voice (`creating-skills` not `skill-creation`)

### 2. Be Concise

- Target < 500 words for most skills
- Use tables for quick reference
- One excellent example beats many mediocre ones

### 3. Code Examples

```go
// ✅ GOOD: Complete, well-commented, from real scenario
func ExampleFunction(ctx context.Context) {
    // Why: GDP requires context-aware logging
    logs.Info(ctx, "handling request")
}

// ❌ BAD: Generic template without context
func DoSomething() {
    // TODO: implement
}
```

### 4. Description Anti-Patterns

```yaml
# ❌ BAD: Summarizes workflow - Claude may skip reading full skill
description: Use for TDD - write test first, watch it fail, write minimal code

# ❌ BAD: Too vague
description: For testing

# ✅ GOOD: Just triggering conditions
description: Use when implementing features or bugfixes, before writing implementation code
```

## Examples

### Example 1: Creating a New Skill

```bash
# 1. Create directory
mkdir -p gdpa_skills/my-category/my-skill

# 2. Create SKILL.md with proper structure
# (see template above)

# 3. Verify skill can be discovered
# Test that description triggers match expected use cases
```

### Example 2: Skill Frontmatter

```yaml
---
name: gdp-logging
description: Use when implementing logging in GDP services. Covers context-aware logging, structured fields, and log levels.
---
```

### Example 3: Minimal Technique Skill

```markdown
---
name: condition-based-waiting
description: Use when tests have race conditions, timing dependencies, or pass/fail inconsistently.
---

# Condition-Based Waiting

## Overview
Replace fixed delays with condition checks that poll until ready.

## Pattern

// ❌ BAD: Fixed delay
time.Sleep(5 * time.Second)

// ✅ GOOD: Condition-based
waitFor(func() bool { return service.IsReady() }, 10*time.Second)
```

## Checklist

Before committing a new skill:

- [ ] Name uses only letters, numbers, hyphens
- [ ] Description starts with "Use when..."
- [ ] Description describes triggers, not workflow
- [ ] Overview states core principle clearly
- [ ] Examples are concrete and runnable
- [ ] Content is < 500 words (unless heavy reference)
- [ ] Keywords cover error messages and symptoms

## Related Resources

- [Anthropic Skill Authoring Best Practices](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/tutorials#skill-files-teaching-claude-new-capabilities)
- Existing skills in this repo for reference patterns
