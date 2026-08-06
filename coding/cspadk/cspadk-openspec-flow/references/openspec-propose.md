---
name: openspec-propose
description: Propose a new change with all artifacts generated in one step. Use when the user wants to quickly describe what they want to build and get a complete proposal with design, specs, and tasks ready for implementation.
license: MIT
compatibility: Requires openspec CLI.
metadata:
  author: openspec
  version: "1.0"
  generatedBy: "1.2.0"
---

Propose a new change - create the change and generate all artifacts in one step.

I'll create a change with artifacts (generated in dependency order):

- proposal.md (what & why)
- specs/ (behavioral specifications — one spec per capability from proposal)
- design.md (how)
- tasks.md (implementation steps)

Dependency chain: proposal → specs + design → tasks. Specs MUST be generated before tasks since tasks references spec scenarios.

When ready to implement, run /opsx:apply

---

**Input**: The user's request should include a change name (kebab-case) OR a description of what they want to build.

**Steps**

1. **If no clear input provided, ask what they want to build**

   Use the **AskUserQuestion tool** (open-ended, no preset options) to ask:
   > "What change do you want to work on? Describe what you want to build or fix."

   From their description, derive a kebab-case name (e.g., "add user authentication" → `add-user-auth`).

   **IMPORTANT**: Do NOT proceed without understanding what the user wants to build.

2. **Create the change directory**
   ```bash
   openspec new change "<name>"
   ```
   This creates a scaffolded change at `openspec/changes/<name>/` with `.openspec.yaml`.

3. **Get the artifact build order**
   ```bash
   openspec status --change "<name>" --json
   ```
   Parse the JSON to get:
   - `applyRequires`: array of artifact IDs needed before implementation (e.g., `["tasks"]`)
   - `artifacts`: list of all artifacts with their status and dependencies

4. **Create artifacts in sequence until apply-ready**

   Use the **TodoWrite tool** to track progress through the artifacts.

   Loop through artifacts in dependency order. The schema defines this order:

   1. **proposal** (no dependencies) → generate first
   2. **specs** (requires proposal) → generate after proposal is done; creates `specs/<capability>/spec.md` per capability from proposal's "New Capabilities" and "Modified Capabilities" sections
   3. **design** (requires proposal) → generate after proposal is done (can run in parallel with specs)
   4. **tasks** (requires specs AND design) → generate last, after both specs and design are done

   **CRITICAL**: The `specs` artifact is NOT optional. It MUST be generated between proposal and tasks. Without specs, the tasks artifact cannot reference spec scenarios and the schema dependency chain is broken.

   a. **For each artifact that is `ready` (dependencies satisfied)**:
      - Get instructions:
        ```bash
        openspec instructions <artifact-id> --change "<name>" --json
        ```
      - The instructions JSON includes:
        - `context`: Project background (constraints for you - do NOT include in output)
        - `rules`: Artifact-specific rules (constraints for you - do NOT include in output)
        - `template`: The structure to use for your output file
        - `instruction`: Schema-specific guidance for this artifact type
        - `outputPath`: Where to write the artifact
        - `dependencies`: Completed artifacts to read for context
      - Read any completed dependency files for context
      - Create the artifact file using `template` as the structure
      - Apply `context` and `rules` as constraints - but do NOT copy them into the file
      - Show brief progress: "Created <artifact-id>"

   b. **Continue until all `applyRequires` artifacts are complete**
      - After creating each artifact, re-run `openspec status --change "<name>" --json`
      - Check if every artifact ID in `applyRequires` has `status: "done"` in the artifacts array
      - Stop when all `applyRequires` artifacts are done

   c. **If an artifact requires user input** (unclear context):
      - Use **AskUserQuestion tool** to clarify
      - Then continue with creation

5. **Read back generated artifacts before reporting or syncing paths**

   After generation reaches apply-ready:
   - Read all generated files under `openspec/changes/<name>/`
   - Do not assume only flat root files exist; include nested generated files under `openspec/changes/<name>/specs/` when the schema creates them
   - Treat the files actually present on disk as the source of truth for later workflow sync
   - Determine the canonical paths that downstream workflow steps should record in requirement status
   - Verify the generated paths exist before returning them to the caller

6. **Show final status**
   ```bash
   openspec status --change "<name>"
   ```

7. **Validate artifact quality**

   After all artifacts are created, validate against quality requirements:

   **proposal.md validation**:
   - Has Problem Analysis table with >= 1 issue per module
   - Has Module Decomposition table with effort estimates (S/M/L)
   - Has Success Criteria section with checkboxes
   - Has Capabilities section listing "New Capabilities" and/or "Modified Capabilities"

   **specs validation**:
   - One spec file exists per capability listed in proposal's Capabilities section (under `specs/<capability>/spec.md`)
   - Each spec file has at least one requirement section (ADDED / MODIFIED / REMOVED / RENAMED)
   - Each requirement has `### Requirement: <name>` header
   - Each requirement has at least one scenario with `#### Scenario: <name>` header (exactly 4 hashtags)
   - Scenarios use WHEN/THEN format
   - ADDED requirements use SHALL/MUST for normative language
   - MODIFIED requirements include full updated content (not partial)
   - REMOVED requirements include Reason and Migration

   **design.md validation**:
   - Has Implementation Details section for each affected file
   - Each file subsection has change diffs using +/- line prefixes (NEW files show only `+` lines, DELETED files show only `-` lines)
   - Each file subsection includes key responsibilities and flow
   - Has Detection Rules table (if infrastructure changes)
   - Has Exact Commands section
   - Has Quality Checks for each affected file

   **tasks.md validation**:
   - Has task groups matching each file from design.md
   - Each task group has Tests/Implementation/Verification subsections
   - Has Final Verification section
   - Test tasks reference scenarios from specs when applicable

   **If validation fails**: Use AskUserQuestion to ask:
   > "The [artifact] is missing [X, Y]. Would you like me to enhance these sections?"

   Do not proceed to reporting until quality gate passes.

**Output**

After completing all artifacts, summarize:
- Change name and location
- List of artifacts created with brief descriptions
- The canonical generated file paths discovered from reading back `openspec/changes/<name>/`, including nested `specs/` outputs when present
- What's ready: "All artifacts created! Ready for implementation."
- Prompt: "Run `/opsx:apply` or ask me to implement to start working on the tasks."

**Artifact Creation Guidelines**

- Follow the `instruction` field from `openspec instructions` for each artifact type
- The schema defines what each artifact should contain - follow it
- Read dependency artifacts for context before creating new ones
- Use `template` as the structure for your output file - fill in its sections
- **IMPORTANT**: `context` and `rules` are constraints for YOU, not content for the file
  - Do NOT copy `<context>`, `<rules>`, `<project_context>` blocks into the artifact
  - These guide what you write, but should never appear in the output

**Guardrails**
- Create ALL artifacts needed for implementation (as defined by schema's `apply.requires`)
- Always read dependency artifacts before creating a new one
- If context is critically unclear, ask the user - but prefer making reasonable decisions to keep momentum
- If a change with that name already exists, ask if user wants to continue it or create a new one
- Verify each artifact file exists after writing before proceeding to next
- Before handing control back to workflow code, enumerate and read the generated change files so downstream status sync uses discovered paths, not assumed ones
