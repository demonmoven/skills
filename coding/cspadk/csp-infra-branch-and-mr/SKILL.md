---
name: csp-infra-branch-and-mr
description: Create a new branch from main, commit all changes, push, and open a Codebase MR via bytedcli
version: 1.0.0
metadata:
  patterns:
    - pipeline
    - tool-wrapper
  domain: csp-infra
  i18n_level: 0
  prompt_version: "1.0.0"
  agent_support:
    - claude-code
    - cursor
  language:
    - zh-CN
    - en
---

# csp-infra-branch-and-mr

One-shot skill to create a new branch from the remote main branch, commit all changes, push, and open a Codebase MR via bytedcli. Automatically detects whether the default branch is `main` or `master` — no hard-coding.

## When to Use

- User says "commit and open an MR", "create a branch and MR", "push and create a merge request"
- User has uncommitted changes and wants to walk through branch → commit → push → MR in one go
- User explicitly wants to branch off the main branch (not the current branch)

Trigger phrases: "commit and open MR", "branch off main and MR", "提代码开 MR", "建分支提 MR"

## Input

| Input | Required | Example |
|-------|----------|---------|
| Change description / commit message intent | Yes | "add --json output mode to CLI commands" |
| Target branch name (optional; auto-generated from changes if omitted) | No | "feat/cli-json-mode" |
| MR title (optional; defaults to commit message) | No | "feat(cli): add --json output mode" |

## Workflow

### Step 1 — Dynamically detect the main branch

Query the remote repository for its default branch — never hard-code `main` or `master`:

```bash
git fetch origin
git remote show origin | grep 'HEAD branch' | awk '{print $NF}'
```

Store the result as `MAIN_BRANCH`. If the command fails (permissions, network), fall back to trying `origin/main` then `origin/master` in order, using whichever fetch succeeds.

### Step 2 — Generate branch name and create branch

If the user did not provide a branch name, infer a semantic name from the changes, formatted as `feat/<slug>` or `fix/<slug>`.

```bash
git checkout -b <branch-name> origin/<MAIN_BRANCH>
```

### Step 3 — Review pending changes

```bash
git status
git diff --stat
```

Confirm all pending files are expected. Warn the user and exclude sensitive files (`.env`, `credentials`, files containing tokens).

### Step 4 — Commit changes

```bash
git add <file1> <file2> ...
git commit -m "<commit-message>"
```

- Specify files individually for `git add` — avoid `git add -A` or `git add .` to prevent accidentally staging sensitive files.
- Follow Conventional Commits format for the commit message.

### Step 5 — Push branch

```bash
git push -u origin <branch-name>
```

### Step 6 — Create Codebase MR

```bash
bytedcli codebase mr create \
  --head <branch-name> \
  --base <MAIN_BRANCH> \
  --title "<mr-title>" \
  --body "<mr-description>"
```

The MR description should include:
- **Summary**: 1–3 bullet points of the core changes
- **Test plan**: A verification checklist

## Output

- New branch created from the remote main branch
- All changes committed and pushed
- Codebase MR URL (output by `bytedcli codebase mr create`)

## Common Pitfalls

1. **Hard-coding the main branch name.** Different repos use `main` or `master`. Always detect via `git remote show`; otherwise `git checkout -b` will fail when the remote branch doesn't exist.
2. **Using `git add -A` and staging sensitive files.** The working tree may contain `.env`, `credentials.json`, or other files that should not be committed. Always add files individually.
3. **Wrong bytedcli flag names.** The flags are `--base` (not `--target-branch`) and `--body` (not `--description`). Always refer to `bytedcli codebase mr create --help` for the authoritative list.

## Output Language

Detect the user's input language and respond in the same language.

- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
