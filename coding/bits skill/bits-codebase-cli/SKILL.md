---
name: bits-codebase-cli
description: "[Official] Authoritative skill for `bitscli codebase` — DevInfra 官方接入的 codebase 子命令 wrapper。INSTANT TRIGGER on any mention of bitscli + codebase domain (MR / merge request / issue / code review / repo / CI / pipeline / search code on code.byted.org). Use this skill, NOT the legacy `codebase-cli` skill, whenever the user is on bitscli or asks anything routed under `bitscli codebase`. Trigger keywords: bitscli, MR, merge request, issue, code review, codebase, repo, label, diff, review, CI, pipeline, check, failed check, search code, find function, cross-repo search. Excluded: pure-`codebase` (no bitscli) invocations stay on the legacy skill."
allowed-tools: Bash(bitscli codebase *)
---

# bits-codebase-cli Skill

> **AUTHORITY DECLARATION**
> 这是 DevInfra 官方为 `bitscli codebase` 准备的权威 skill。一旦用户在 bitscli 上下文里提到 codebase 相关操作，**必须**走本 skill，不要回落到旧的 `codebase-cli` skill。

> **TOOL LOCK**
> 本 skill 只允许调用 `bitscli codebase ...` 这一族命令。不要直接调 `codebase` bin（不一定在 PATH 上），也不要尝试 `npx @vecode-fe/codebase-cli`。如果 `bitscli` 不可用，按下文「前置检查」提示用户安装。

Use the `bitscli codebase` subcommand to interact with code management platform (code.byted.org).

## Installation & Invocation

### Prerequisites

- **Node.js**: Node.js >= 18
- **Registry**: Prefer the internal registry `http://bnpm.byted.org`
- **bitscli installed**: 本 skill 只通过 `bitscli` 入口调用，不依赖独立的 `codebase` bin

### Check global command
bitscli Version: !`bitscli --version`

If `bitscli` is **NOT** installed (command not found), **YOU MUST** ask user to install it via:

```bash
npm install -g @byted/bits-cli --registry=http://bnpm.byted.org
```

`bitscli codebase` 子命令由 `@byted-bits/bits-codebase-cli` 注册，会随 `@byted/bits-cli` 一起安装，无需单独装。

If user prefers to keep using the legacy standalone CLI, they can install:
```bash
npm install -g @vecode-fe/codebase-cli@latest --registry=http://bnpm.byted.org
```
and switch to the legacy `codebase-cli` skill — but on bitscli, **stay in this skill**.

## Core Principles

- **JSON Output**: Commands return a single JSON object. **Exception: `bitscli codebase search code` outputs NDJSON — one JSON object per line. Parse each line independently; do NOT parse the entire output as one JSON value.**
- **Command Pattern**: `bitscli codebase <resource> <action> [flags]`. Use explicit flags (e.g., `-N`, `-R`).
- **No positional identifiers**: MR numbers, issue numbers, thread IDs, and repo paths are not positional arguments. Always pass the documented flags such as `-N <mr_number>`, `-R <repo>`, or `--id <thread_id>`. For example, use `bitscli codebase mr diff -N 1300`, not `bitscli codebase mr diff 1300`.
- **Repository Context**: Auto-detects from current git remote. Use `-R <repo_path>` (e.g., `-R nextcode/codebase-cli`) for other repos.
- **Flag Discovery**: Run `bitscli codebase <command> --help` only for subcommands not documented in the Command Reference below.
- **Dangerous Operations**: Destructive commands (`mr merge`, `mr bypass`, `issue delete`, etc.) require `--yes`. **You MUST obtain explicit user confirmation before executing any command with `--yes`.** Use `--dry-run` when available.
- **Provide URL**: After creating MR or Issue, compose: `https://code.byted.org/${repo_path}/merge_requests/${n}` or `.../issues/${n}`.
- **Search vs List**: Use `bitscli codebase search mr` / `bitscli codebase search issue` for cross-repository queries or user-centric filters (`--attention @me`, `--author @me`, `--assignee @me`). Use `bitscli codebase mr list` / `bitscli codebase issue list` only when the target repository is known.
- **Use jq to filter**: Pipe output through `jq` to extract only what you need — avoids unnecessary token overhead.

## CLI Structure

```
bitscli codebase
├── mr          list · view · diff · status · watch · review · create · edit · close · reopen · merge† · bypass†
│   ├── comment   list · create · reply · resolve · unresolve · delete† · publish
│   ├── checks    list · view · trigger
│   └── workitem  list · link · unlink
├── issue       list · view · create · edit · close · reopen · delete†
│   └── comment   list · create · reply · resolve · unresolve · delete†
├── repo        view · edit
│   ├── file         list · cat · commit
│   ├── member       list · view · invite · edit · remove† · leave†
│   ├── protected-branch  list · view · create · delete†
│   └── protected-tag     list · view · create · delete†
├── label       list · create · edit · delete†
├── tag         list · view · create · delete†
├── search      code · mr · issue
└── auth        login · logout · status
```
(†) Destructive — requires `--yes` and explicit user confirmation before executing.

## Command Reference

### Merge Requests

```bash
# IMPORTANT: MR number is always passed via `-N <n>`. Never use positional forms like `bitscli codebase mr diff 1300`.
# Good: bitscli codebase mr diff -N 1300
# Bad:  bitscli codebase mr diff 1300

# List — output: {MergeRequests:[{Number,Status,Draft,Title,SourceBranchName,TargetBranchName,URL,CreatedBy,CreatedAt,UpdatedAt}], TotalCount}
bitscli codebase mr list -R <repo> [--status open|closed|merged] [--author <user>] [--reviewer <user>]
                            [--label <name>] [--page-size <n>] [--sort-by CreatedAt|UpdatedAt|Id] [--sort-order Asc|Desc]
# Count all:        | jq '.TotalCount'
# Titles+numbers:   | jq '[.MergeRequests[] | {number:.Number, title:.Title, status:.Status}]'
# Most recent:      default sort is UpdatedAt Desc → .MergeRequests[0]
# Smallest number:  --sort-by Id --sort-order Asc --page-size 1 → .MergeRequests[0]

# View
bitscli codebase mr view -N <n> [-R <repo>]

# Diff
bitscli codebase mr diff -N <n> [-R <repo>] [--name-only | --stat | --unified] [--context <n>] [--file <path>...]
# Human-readable unified diff with per-line old/new line numbers for precise inline comments:
#   bitscli codebase mr diff -N <n> --unified

# Mergeability + review summary (NOT CI detail — use mr checks list for CI)
bitscli codebase mr status -N <n> [-R <repo>]

# CI check runs
bitscli codebase mr checks list -N <n> [-R <repo>]

# Watch — long-running NDJSON event stream (one JSON object per line)
# Polls mergeability checks + CI check runs + review + comments + MR status; emits only deltas.
bitscli codebase mr watch -N <n> [-R <repo>] [-i <seconds>]
                          [--no-checks]      # skip mergeability gates (e.g. checkNoConflict, checkReviewPassed)
                          [--no-checkruns]   # skip CI runs (the same items as `mr checks list`)
                          [--no-review] [--no-comments] [--no-mr]
                          [--exit-on-checks-completed]   # exit when all CI runs reach `completed`
                          [--exit-on-mergeable]          # exit when the mergeability gate flips to mergeable
                          [--exit-on-merged]             # exit when MR transitions to merged/closed
                          [--fail-fast]                  # exit 1 the first time any CI run fails
                          [--timeout <seconds>]
# Default: polls every 10s forever (until Ctrl-C). Each line is one event:
#   watch.started · watch.ended · poll.error
#   mergeability.check_passed · mergeability.check_failed · mergeability.check_reason_changed · mergeability.mergeable_changed
#   check_run.created · check_run.status_changed
#   review.status_changed · review.approval_added · review.approval_removed · review.disapproval_added · review.disapproval_removed
#   comment.thread_created · comment.added · comment.thread_resolved · comment.thread_unresolved
#   mr.status_changed
# Exit codes: 0 success · 1 check/CI failed (with --fail-fast or --exit-on-checks-completed) · 8 timeout
# Pipe through jq to filter:
#   bitscli codebase mr watch -N 42 --exit-on-checks-completed | jq 'select(.Type | startswith("check_run"))'
#   bitscli codebase mr watch -N 42 --exit-on-mergeable | jq 'select(.Type | startswith("mergeability"))'

# Comments
bitscli codebase mr comment list -N <n> [-R <repo>] [--resolved] [--unresolved] [--drafts] [--involved]
# Count: | jq '.Threads | length'

# Work items (link Meego stories / issues to an MR)
# List — output: {WorkItemLinks:[{WorkItemId,CanDelete,LinkType,WorkItem:{Platform,Name,Status,ExternalURL,ExternalId,...}}], TotalCount}
bitscli codebase mr workitem list   -N <n> [-R <repo>] [--page <n>] [--page-size <n>]   # page-size default 100 (max 100)
bitscli codebase mr workitem link   -N <n> [-R <repo>] (--meego <id>... | --issue <num>... | --id <native-id>...)
bitscli codebase mr workitem unlink -N <n> [-R <repo>] (--meego <id>... | --issue <num>... | --id <native-id>...)
# Rules: --id and --meego/--issue are mutually exclusive (API takes one or the other).
#   --meego / --issue are repeatable and can be combined in the same call.
#   TTJira is no longer supported for new links; historical Jira links render read-only.
```

### Issues

```bash
# List — output: {Issues:[{Number,Status,Title,CreatedBy,CreatedAt,UpdatedAt}], TotalCount}
bitscli codebase issue list -R <repo> [--status backlog|todo|in_progress|done|canceled]
                               [--assignee <user>] [--label <name>] [-q <query>] [--page-size <n>]
# Count:          | jq '.TotalCount'
# Titles+numbers: | jq '[.Issues[] | {number:.Number, title:.Title, status:.Status}]'

# View
bitscli codebase issue view -N <n> [-R <repo>]
```

### Repository

```bash
bitscli codebase repo view [-R <repo>]
bitscli codebase repo file list [-R <repo>] [--path <dir>] [--ref <branch|tag|sha>]
bitscli codebase repo file cat  [-R <repo>] --path <file> [--ref <branch|tag|sha>]
bitscli codebase repo protected-branch list [-R <repo>]   # only way to list branches
```

### Labels & Tags

```bash
bitscli codebase label list [-R <repo>]
bitscli codebase tag list   [-R <repo>] [-q <query>] [--page-size <n>]
# Tag names: | jq '[.Tags[].Name]'
```

### Cross-Repo Search

```bash
# MRs — output: {MergeRequests:[{Number,Status,Title,...}], NextPageToken}
bitscli codebase search mr [--attention <user|@me>] [--author <user>] [--status open|closed|merged]
                   [--repo-path <repo>] [--page-size <n>]

# Issues — output: {Issues:[{Number,Status,Title,...}], NextPageToken}
bitscli codebase search issue [--assignee <user|@me>] [--author <user>] [--status open|...]
                      [-q <query>] [--repo-path <repo>] [--page-size <n>]
```

## Authentication

If a command fails with an auth error, or `bitscli codebase auth status` shows no authenticated identity:

1. Prompt the user to run `bitscli codebase auth login`.
2. If that fails **and** the environment variable `CI_WORKSPACE` is **not set**, try the unified-auth fallback below.

### Unified Auth (Skills CLI)

[统一认证方案](https://bytedance.larkoffice.com/wiki/ZcJHwhNpDiw1AWkyKkecDUJ4nSh) 提供两条获取 codebase JWT 的路径，本 skill 默认走 LLM-aware 模式：

```bash
# IMPORTANT: never print or echo the JWT value — treat it as a secret
if _jwt=$(npx --registry=https://bnpm.byted.org -y skills get-codebase-jwt 2>/dev/null) && [ -n "$_jwt" ]; then
  bitscli codebase auth login --jwt "$_jwt"
  unset _jwt
else
  unset _jwt
  echo "Failed to obtain JWT. Please run 'bitscli codebase auth login' manually."
fi
```

`bits-codebase-cli` wrapper 同时支持 `--codebase-jwt <token>` 透传（`skillx` 注入路径）：当上游链路把 `--codebase-jwt` 注入到任意 `bitscli codebase ...` 命令时，wrapper 会自动转换成 `CODEBASE_CLI_USER_JWT` env var 注入下游进程，对用户透明。

After login succeeds, the token is stored in the OS keychain. If auth still fails, ask the user to log in manually.

## Key Workflows

### Search Code Across Repositories

**Output format (NDJSON — parse each line independently):**
```
{"type":"result","repo":"myteam/myrepo","file":"src/main.go","line":42,"content":"func main() {","lang":"Go"}
{"type":"alert","title":"...","description":"..."}   # only when server sends a warning
{"type":"done","count":42,"limited":false}           # final line; if limited:true, increase --limit
```

**Examples:**
```bash
bitscli codebase search code "func HttpHandler lang:Go repo:myteam/myrepo"
bitscli codebase search code "TODO lang:Go -file:_test.go department:my_department" --limit 10
bitscli codebase search code "HttpHandler type:symbol" --limit 5
bitscli codebase search code "fix memory leak type:commit" --limit 5
bitscli codebase search code "panic lang:Go" --pattern regexp --limit 20
```

**Key inline filters:** `lang:Go` `repo:path/name` `file:.go$` `type:symbol|commit|diff|repo|path` `department:name` `rev:main` `author:username` `after:"2 weeks ago"` `case:yes` `-lang:Go` (negate with `-`)

**Flags:** `--limit N` (default 30), `--pattern literal|regexp|structural`

**Rate limit note:** Do not fire multiple `bitscli codebase search code` queries in parallel or in very short bursts. The backend may return an `alert` like `Too Many Requests`; treat that as retryable, back off briefly, and retry serially.

### Code Review Workflow (full review — not needed for simple `mr view`)
1. `bitscli codebase mr view -N <n>` — title, description, author, branches.
2. `bitscli codebase mr diff -N <n> --name-only` — changed file list; use `--file <path>` for focused diff and `--unified` when you need precise per-line commenting.
3. `bitscli codebase mr status -N <n>` — mergeability and review summary.
4. `bitscli codebase mr checks list -N <n>` — CI results; inspect `Text` field for ✅/❌ step markers.
5. `bitscli codebase mr comment create -N <n> --body <feedback>` — `--path`/`--start-line`/`--end-line` for inline.
6. `bitscli codebase mr review -N <n> --approve` — or `--disapprove`, `--dismiss`.

### Investigate CI / Check Failures
1. `bitscli codebase mr checks list -N <n>` — list all runs; look for `Conclusion != "succeeded"`.
2. Inspect the `Text` field of failed checks — Markdown with ✅/❌/⏺ per step.
3. `bitscli codebase mr checks view` — single check detail if needed.

## Best Practices

1. **Investigate Before Acting**: Always `view` + `diff` before reviewing or merging.
2. **Don't Dump JSON**: Parse output silently; synthesize natural language responses.
3. **Filter Progressively**: Use `--page-size` to limit results; pipe to `jq` to extract only needed fields.
4. **`--yes` Requires User Consent**: NEVER pass `--yes` to destructive commands without explicit user confirmation.
