---
name: csp-infra-mr-ci-fix
description: Fetch CI check failures and MR comments for current branch's MR via bytedcli and auto-fix the errors
version: 1.3.1
metadata:
  patterns:
    - tool-wrapper
    - pipeline
  domain: csp-infra
  i18n_level: 0
  prompt_version: "1.3.1"
  agent_support:
    - claude-code
    - cursor
  language:
    - zh-CN
    - en
---

# csp-infra-mr-ci-fix

Automatically fetch CI check failures and MR review comments for the current branch's MR and attempt to fix the errors. This skill streamlines the debug-fix loop by combining bytedcli codebase checks capabilities with intelligent error analysis and auto-repair.

## When to Use

- When you push a branch and want to check if CI passes before requesting review
- When CI fails on your MR and you need to diagnose and fix the errors
- When iterating on code changes and want quick CI feedback loop
- When you want to check and address MR review comments
- Trigger phrases: "fix ci errors", "check mr ci status", "what's wrong with my mr", "ci failed fix it", "check mr comments", "address review feedback"

## Input

| Input | Required | Example |
|-------|----------|---------|
| auto_fix | No (default: true) | Set to `false` to only analyze without making changes |
| check_run_id | No | Specific check run ID to focus on (skips MR lookup) |

## Workflow

### Stage 1: MR Detection

1. Get current branch name from git: `git branch --show-current`
2. Fetch MR info for current branch: `bytedcli codebase mr get` (auto-detects from current repo)
3. Extract MR ID and source commit SHA from response
4. If no MR found, prompt user to create one or provide MR ID manually

### Stage 2: CI Status Analysis

1. Get MR's CI check runs: `bytedcli codebase checks mr <mr_id>`
2. Parse output to identify:
   - Failed check runs (status: completed, conclusion: failure)
   - Running check runs (status: in_progress, queued)
   - Successful check runs (status: completed, conclusion: success)
3. If all checks pass, report success and exit
4. If checks still running, wait for completion using the **CI Polling Loop** (see Stage 6)

#### CI Status Summary Table

After parsing, present a unified status table to the user:

```
| # | Check Name | Status | Conclusion |
|---|-----------|--------|------------|
| 1 | ...       | ✅ PASS / ❌ FAIL / ⏳ RUNNING | ... |
```

If any items show FAIL, list the failing items and proceed to Stage 3.

### Stage 2.5: MR Comments Review

After CI checks pass (or as a separate step), fetch and process MR review comments:

1. **Fetch MR comments**:
   ```bash
   # List all comments
   bytedcli codebase mr comment list <mr_id>

   # Get detailed JSON output for parsing
   bytedcli codebase mr comment list <mr_id> -j | jq '.'
   ```

2. **Parse comment structure**:
   - `threads[].Comments[].Content` - Comment text
   - `threads[].Positions[].Path` - File path the comment refers to
   - `threads[].Positions[].StartLine/EndLine` - Line range
   - `threads[].Status` - "open" (unresolved) or "resolved"

3. **Identify actionable comments**:
   - Focus on comments with `status: "open"`
   - Extract file paths and line numbers from `Positions`
   - Categorize by type:
     - **Code suggestions**: Request to modify code
     - **Questions**: Need clarification/answer
     - **Blocking issues**: Must fix before merge

4. **Process each comment**:
   - Read the referenced file at the specified lines
   - Understand the reviewer's intent
   - Apply the suggested fix or prepare a response

5. **After fixing, reply and resolve comment threads**:
   ```bash
   # Step 1: Reply to a comment thread (creates draft)
   bytedcli codebase mr comment reply <mr_id> --thread-id <thread_id> -b "Fixed in commit xxx"

   # Step 2: Publish all draft comments
   bytedcli codebase mr comment publish <mr_id>

   # Step 3: Resolve the thread if issue is fully addressed
   bytedcli codebase mr comment resolve --id <thread_id>
   ```

6. **Complete workflow example**:
   ```bash
   # Fetch comments and parse
   bytedcli codebase mr comment list 40 -j | jq '.data.threads[] | select(.Status == "open")'

   # Apply fixes to code...

   # Reply to each resolved issue
   bytedcli codebase mr comment reply 40 --thread-id 770103851377875 -b "已删除重复内容，修复提交: abc123"
   bytedcli codebase mr comment reply 40 --thread-id 770104470054849 -b "已修复，修复提交: def456"

   # Publish all replies
   bytedcli codebase mr comment publish 40

   # Resolve threads
   bytedcli codebase mr comment resolve --id 770103851377875
   bytedcli codebase mr comment resolve --id 770104470054849
   ```

### Stage 3: Error Diagnosis

For each failed check run:
1. Get detailed check info: `bytedcli codebase checks get --id <check_run_id>`
2. Fetch logs to temp file: `bytedcli codebase checks log --check-run-id <id> > /tmp/ci-<id>.log`
3. Search logs for error patterns:
   - Lint errors: `eslint`, `tsc`, `golangci-lint`, `flake8`, `pylint`
   - Test failures: `FAIL`, `Error:`, `panic:`, `AssertionError`
   - Build errors: `Cannot find module`, `undefined reference`, `compilation failed`
   - Dependency errors: `npm ERR!`, `go mod`, `pip install failed`
4. Categorize errors by type and affected files

### Stage 4: Auto-Fix

Based on error type, apply appropriate fixes:

**Lint Errors**:
- Run `npm run lint --fix` or equivalent
- For TypeScript: fix type errors in identified files
- For Go: run `gofmt -w` and fix import issues

**Test Failures**:
- Analyze test output to identify failing tests
- Check if test is outdated vs. code changes
- Update test expectations or fix code logic

**Build Errors**:
- Check missing dependencies: `npm install`, `go mod tidy`
- Fix import paths and module references

**Common Patterns**:
- Type errors: Add type annotations or fix type mismatches
- Import errors: Update import paths, add missing imports
- Formatting: Apply auto-formatters (prettier, gofmt)

### Stage 5: Verification

1. Run local checks if available: `npm run lint`, `npm run test`, `go test ./...`
2. If local checks pass, stage and commit changes
3. Push to trigger new CI run
4. **Ask user whether to start the CI polling loop**: After the first round of fixes is pushed, present the following prompt:
   - "Fixes have been pushed. Would you like to start CI polling to automatically monitor and fix remaining issues? (Yes — start polling / No — stop here)"
5. If the user declines, output the manual monitoring command and exit
6. If the user accepts, proceed to Stage 6

### Stage 6: CI Polling Loop

After pushing fixes, automatically poll CI status and loop until all checks pass. This stage is only entered when the user confirms in Stage 5.

#### Polling Strategy

1. **Poll interval**: Use 30–60 second intervals. Do not poll too aggressively.
2. **Maximum iterations**: 20 iterations (≈10–20 minutes). If CI has not resolved after this, stop and report.
3. **Per-iteration flow**:
   a. Run `bytedcli codebase checks mr <mr_id>` to get current CI status
   b. Parse results: categorize checks as passed / failed / running
   c. If all checks pass → report success and exit the loop
   d. If new failures found since last check → proceed to Stage 3 (diagnosis) → Stage 4 (fix) → Stage 5 (verify + push)
   e. If checks still running with no new failures → wait and poll again
   f. Output a brief status update each iteration:

   ```
   [Poll N/20] ✅ Passed: X | ❌ Failed: Y | ⏳ Running: Z | ...
   ```

#### Handling Failures Found During Polling

When a new failed check is detected:
1. Fetch logs: `bytedcli codebase checks log --check-run-id <id> > /tmp/ci-<id>.log`
2. Search for error patterns in the log file
3. Apply fix (Stage 4)
4. Run local verification (Stage 5 step 1)
5. Commit and push
6. Reset poll counter and continue monitoring

#### Loop Exit Conditions

The polling loop exits when:
- All CI checks pass (success)
- Maximum iterations reached (report remaining issues)
- User interrupts (always allow graceful exit)
- No progress after 3 consecutive polls with same failures (stuck)

#### Output on Exit

```
## CI Polling Result
- Total polling iterations: N
- Final status: ALL PASSED / PARTIAL / STUCK
- Passed checks: X/Y
- Remaining failures (if any):
  - <check_name>: <brief error description>
- Manual check: bytedcli codebase checks mr <mr_id>
```

## Output

A structured report including:

```markdown
## CI Check Summary
- Total checks: N
- Passed: X
- Failed: Y
- Running: Z

## Failed Checks Analysis
### <check_name> (ID: xxx)
- **Error Type**: lint / test / build / dependency
- **Affected Files**: file1.ts, file2.go
- **Root Cause**: Brief description
- **Fix Applied**: What was changed

## MR Comments Review (if applicable)
### Thread: <thread_id>
- **File**: path/to/file.ts (lines 10-20)
- **Author**: reviewer_name
- **Comment**: Reviewer's suggestion
- **Action Taken**: Fix applied / Response provided

## Changes Made
- file1.ts: Fixed type error in function X
- file2.go: Added missing import

## Next Steps
- CI polling loop is active — monitoring for new check results
- Manual check: `bytedcli codebase checks mr <mr_id>`
- Reply to any outstanding comments
```

## Common Pitfalls

1. **Large log files overwhelm context**: Always redirect logs to temp files and search with grep/rg instead of loading entire logs into context. The `bytedcli codebase checks log` output can be thousands of lines.
2. **Fixing wrong error first**: CI failures often cascade. Always fix lint/build errors before test failures - tests may fail due to type errors in the test setup itself.
3. **Assuming CI is deterministic**: Flaky tests or external service issues can cause intermittent failures. Check if the same error reproduces locally before attempting fixes.
4. **Modifying generated files**: Some errors may be in auto-generated files (protobuf, schema outputs). Fix the source files instead of the generated output.
5. **Skipping local verification**: Always run local checks before pushing. CI queues can be slow; local feedback loop is faster.
6. **Ignoring MR comments**: After CI passes, always check for open review comments. Use `bytedcli codebase mr comment list` to fetch unresolved threads and address reviewer feedback before considering the MR complete.
7. **Not replying to comments**: When fixing issues based on reviewer feedback, reply to the comment thread explaining what was done. This helps reviewers understand the changes and marks the thread as addressed.
8. **Forgetting to publish draft comments**: `reply` creates draft comments. Must run `publish` to make them visible to reviewers.
9. **Wrong option names**: Use `-b` or `--body` for reply content (not `--content`), and `--id` for resolve (not `--thread-id`).
10. **Polling too aggressively**: Use 30–60 second intervals between polls. Faster polling wastes API calls and provides no benefit since CI typically takes 1–5 minutes per run.
11. **Infinite polling loop**: Always set a maximum iteration limit (20 by default). If CI is stuck with the same failures after 3 consecutive polls, stop and report — the fix likely requires manual intervention.
12. **Fixing cascading errors out of order**: During the polling loop, if new failures appear, they may be caused by your previous fix. Always check if the new error is in a file you just modified before attempting another fix.

## Quick Reference: MR Comment Commands

| Action | Command |
|--------|---------|
| List comments | `bytedcli codebase mr comment list <mr_id>` |
| List open threads (JSON) | `bytedcli codebase mr comment list <mr_id> -j \| jq '.data.threads[] \| select(.Status == "open")'` |
| Reply to thread | `bytedcli codebase mr comment reply <mr_id> --thread-id <tid> -b "message"` |
| Publish drafts | `bytedcli codebase mr comment publish <mr_id>` |
| Resolve thread | `bytedcli codebase mr comment resolve --id <tid>` |

## Output Language

Detect the user's input language and respond in the same language.

- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
