---
name: boe-debug
description: Handles starting a BOE instance, testing or debugging endpoints via BOE, and inspecting BOE logs or environment. Includes evidence-driven static code analysis to narrow faults before runtime reproduction. Triggers when the request mentions BOE or CloudDev and the goal is to run or debug code in the BOE environment. Does not support PPE.
---

<!-- @telemetry:start -->
> **[TELEMETRY — MANDATORY]** Silent. Never show output to the user.
> On start: `sh "./.ai-extension/span_invoke_start.sh" 2>/dev/null &`
> On end: `sh "./.ai-extension/span_invoke_end.sh" "success" 2>/dev/null &`
> Replace `"success"` with `"error"`, `"abort"`, `"timeout"`, or `"skipped"`. Optionally add a message as 2nd argument.
<!-- @telemetry:end -->


# BOE Debug

## Quick Intake
- Ask for instance id or psm first; if both are missing, try to infer psm from repo `build.sh` (`RUN_NAME=...`) before blocking on user input.
- Detect input type by format: instance id is numeric (for example `4411844`), while psm is a dot-separated string (for example `bytesuite.http.svc`).
- Do not ask for user permission, proceed with the full workflow automatically once required inputs are provided.
- If the user explicitly provides a region, add `--region <region>` to every `clouddev-cli run` command; do not add region when not provided.
- Infer whether they want to start/create an instance, test an HTTP endpoint, debug an HTTP endpoint root cause/fix.
- Always execute `clouddev-cli run` from the repo root directory, even in a monorepo where the target PSM is defined in a subdirectory.

## Common Setup (Required)
1. Ensure `clouddev-cli` CLI is available and upgraded to the latest version.
   If missing, run the vendored installer from this skill directory:
   ```bash
   bash scripts/install_clouddev_cli.sh
   ```
   If already installed, run `clouddev-cli upgrade`.
   If installation or upgrade fails, show the full shell output and let the user handle the download.
2. If user did not provide `psm`, try to discover it from the repo.
   Check module root `build.sh` files for patterns like `RUN_NAME="bytesuite.http.svc"` and use that value as `psm`.

## Network Access (Required)
This skill requires internal network access. Run commands outside the sandbox and request escalation when needed so internal services are reachable. When requesting escalation, set prefix rules for `clouddev-cli` and other commands so the user only approves once. Do not add extra confirmation steps; proceed immediately after escalation is granted. If escalation is denied, stop and report that the workflow cannot continue.

## Login Handling
If login is required during `clouddev-cli` commands, show the full shell output to the user without truncation.
If the UI collapses or truncates the terminal output, ask the user to log in by either running `clouddev-cli` in their terminal or logging in via the CloudDev plugin.

## IDE Identification and Flag (Required)
IDE identification is mandatory before running `clouddev-cli run`.
Pick IDE as one of: `jetbrains`, `vscode`, `trae`.
Use explicit user input first; otherwise infer from context (logs, paths, commands, plugin names, UI wording).
`--ide <ide>` is mandatory on every `clouddev-cli run` command.

## User Token for curl
If a `curl` request needs a user token, fetch it with:

```bash
USER_TOKEN="$(clouddev-cli get-token)"
[ -n "$USER_TOKEN" ] || { echo "Failed to get user token"; exit 1; }
```

Then use that token in the `curl` request, for example:

```bash
curl ... -H "x-jwt-token: $USER_TOKEN"
```

## Mode Selection (Required)
- If the user wants to start or create a BOE instance only, use the Start-only workflow.
- If the user wants to test endpoint behavior (bug check), use the Endpoint Test workflow.
- If the user wants to debug an HTTP endpoint root cause/fix, use the Debug workflow.

## Start Procedure (Shared)
1. Run `clouddev-cli run --sync` with either `--id <id>` or `--psm <psm>` based on user input.
   If using `--id <id>`, do not set `--env`.
   If using `--psm <psm>` and user did not provide env, run `clouddev-cli get-user`, build env as `boe_debug_{user}`, and add `--env <boe_debug_{user}>`.
   If `clouddev-cli get-user` returns empty, generate a random string as `{user}` and continue.
   Do not use `--no-code-sync` flag with `clouddev-cli run`.
2. Wait for `clouddev-cli run --sync` to return, and confirm the terminal output indicates the instance is running.
3. Run `clouddev-cli detail <id>` and read `main_service_address_v6` (if `running`) or check `failure_message` (if `ready`).

## Start-only Workflow (Run/Create Instance)
Follow Common Setup first. Do not add logs or change code unless the user explicitly asks.
If the user asks to run an instance or create an instance, use `clouddev-cli run` directly; it will create the instance when it does not already exist.
1. Run the Start Procedure (Shared).

2. If status is `running`, report success and `main_service_address_v6`, then ask if they want to run a request or stop.
3. If status is `ready` and `failure_message` is non-empty, treat the start as failed. Run `clouddev-cli tail <id> -n 1000` to inspect logs and determine whether it is a user-side issue (for example, build script or repo errors) or a server-side failure. Report evidence and ask how to proceed.

## Endpoint Test Workflow (HTTP Probe / Bug Check)
Follow Common Setup first.
1. Start Procedure (Shared) (mandatory).
2. Build the request from user input (method/path/headers/body); include expected behavior if provided.
3. Execute the request locally from the agent terminal (typically with `curl`) against `main_service_address_v6`, not inside the BOE server instance.
4. Compare actual vs expected behavior and report whether there is evidence of a bug.
5. Do not add logs or change code in this workflow. If user asks for root cause/fix, switch to the Debug workflow.

## Source Code Analysis

Evidence-driven static analysis techniques for narrowing down the fault area before adding logs. Apply these during the "Analyze the code path statically" step of the Debug Workflow.

### Anti-Patterns to Avoid

| Anti-Pattern | Correct Approach |
|---|---|
| Guessing file paths from naming conventions | Search or extract from error output / user context |
| Re-searching the same pattern that returned 0 results | Switch to a different signal |
| Reading file headers (lines 1-20) instead of suspect location | Read 100-200 lines centered on the suspect area |
| Claiming code structure without reading it | Read the file to verify before claiming |
| Reading 10-20 lines at a time | Read 100-200 lines for sufficient context |
| Exploring tangential code after finding the suspect area | Stop analysis and proceed to log instrumentation |

### Core Principle: RETRIEVE, not GENERATE

NEVER make claims about code without direct evidence from reading the actual source. Use information from error output and code you have actually read. Do not infer paths, function names, or behavior from training knowledge or naming conventions.

### Multi-Signal Extraction

Extract ALL identifiers from the problem description as search signals, in priority order:

1. **Error message keywords** — field names from assertion failures, exact error tokens (HIGHEST priority)
2. **HTTP-specific signals** — route patterns, handler names, HTTP status codes, middleware identifiers
3. **Function names** — may contain typos, don't trust blindly
4. **Field/variable names** — usually exact, good fallback
5. **Type names** — struct, interface, class names
6. **File path hints** — if mentioned directly

Search strategy: try PRIMARY signal first. If 0 results, IMMEDIATELY try FALLBACK signals. Do not waste iterations on variations of the same failed signal.

### Progressive Search Relaxation

When initial search returns 0 results:

1. **Drop common suffixes** — Type, Impl, Handler, Helper, Utils, V2, Wrapper
2. **Use common prefix** — `processUserDataHelper` → try `processUserData`
3. **Switch to alternative signals** — field names, type names, error strings
4. After 2 failed searches on same signal, switch strategy. NEVER repeat the same pattern.

### Evidence-Gated File Reading

NEVER read a file unless you have evidence it's relevant:
- Valid: file appeared in search results, error output, or is imported by a confirmed-relevant file
- Invalid: filename sounds related, following naming conventions, guessing based on directory structure

### Function Call Tracing

When tracing the code path for the failing endpoint:
- If the handler calls other functions, READ THEM before concluding where to instrument
- The bug is often in a CALLED function, not the handler itself
- Do NOT hypothesize about the fault without reading the full call chain

### External Type Recognition

When searching for type definitions yields no results:
- Check import paths — `kitex_gen/`, `pb/`, `proto/`, `gen/` indicate external/generated types
- Do NOT keep searching for definitions that are in external packages
- Instead, understand the type by HOW it's used in the current repo

### Analysis Termination

Stop static analysis and proceed to log instrumentation when you have:
- Located the suspect code area where the error likely occurs
- Read the relevant code around that location
- Determined where to place targeted logs for maximum diagnostic value

### Fix Discipline

- **Fix scope:** determine whether the bug is in handler/production code only, test code only, or both. Only modify what is necessary.
- **Focused changes:** make ONLY the changes necessary to solve the specific problem. Do not refactor unrelated code, rename variables, reformat files, or add unnecessary logging.
- **Batch fix:** for repetitive errors (same pattern in multiple places), read with enough context to see all occurrences and apply ALL fixes together, not one per iteration.
- **Signature mismatch:** when fixing argument count errors, search for ALL call sites and fix them in ONE pass.
- **Test preservation:** NEVER remove test assertions to fix compilation — UPDATE them to use correct fields/methods. The test INTENT usually remains valid even when struct fields change.

## Debug Workflow (HTTP Endpoints)
**Mandatory process:** Do not fix the bug before reproducing it. Do not diagnose or propose fixes based only on static code or prior knowledge. Always execute the steps labeled "Analyze the code path statically", "Add targeted logs", "Start Procedure", "Reproduce the request", and "Fetch and analyze logs" before concluding. If any step is blocked, stop and ask the user to unblock it.

1. Locate the code path for the failing endpoint.
   Do not fix the bug yet. Reproducing the bug is required.
   If needed, use `clouddev-cli detail` output (psm, repo info, name) to infer likely repo and path.

2. Analyze the code path statically (mandatory).
   Apply the techniques from the Source Code Analysis section above to narrow down the fault area.
   Use Multi-Signal Extraction to identify search targets, trace function calls through the handler chain, and read code with evidence-gated discipline.
   The goal is to determine exactly where to place targeted logs for maximum diagnostic value. Do not fix the bug at this step.

3. Add targeted logs informed by analysis (mandatory).
   Based on findings from step 2, insert log statements around the identified suspect areas and their dependencies (inputs, upstream calls, error paths, and outputs).
   Confirm code changes are applied before starting the instance. Do not run `clouddev-cli run` while still waiting on user permission.
   If the user declines, stop and ask whether they want to proceed without log instrumentation. Do not diagnose without logs.

4. Start Procedure (Shared) (mandatory).

5. Reproduce the request (mandatory).
   Craft a request based on the code and user context.
   Example: `curl -sS "http://[<main_service_address_v6>]/ping"` (adjust method, headers, body, and path). Read the response directly.

6. Fetch and analyze logs (mandatory).
   Run `clouddev-cli tail <id> -n 200` and read it directly.
   If no relevant lines appear, fetch more lines or broaden the patterns.

7. Diagnose and fix with discipline (after mandatory steps).
   Implement the fix in code after identifying the root cause. Apply the Fix Discipline principles from the Source Code Analysis section: determine fix scope, make focused changes, batch-fix repetitive errors, and preserve test intent.
   Re-run the Start Procedure, then re-run the request and logs to verify the fix resolves the issue.
   Do not loop on speculative fixes. Make at most one fix attempt per confirmed root-cause hypothesis; if it fails and no new evidence appears, stop and ask the user for more context instead of continuing to retry.
   After verification, remove the added debug logs.
   After log removal, don't verify again, provide a final summary directly (root cause, code changes, and verification evidence).

## Unexpected Situations
- Keep raw outputs when `clouddev-cli` returns non-standard responses and ask the user how to proceed.
- If no id/psm is provided and no prior instance context is available, ask for id or psm.
- If `clouddev-cli` is killed by the terminal (for example output includes `killed`), treat the local binary as likely broken. Remove it (`rm "$(command -v clouddev-cli)"`) and reinstall from this skill directory:
  ```bash
  bash scripts/install_clouddev_cli.sh
  ```
- If a `clouddev-cli` subcommand or flag used in this skill is reported as unrecognized/unknown by the CLI, run `clouddev-cli upgrade` and retry the original command.
- Use `clouddev-cli --help` anytime command usage is unclear.
- If status becomes `ready` and `failure_message` is non-empty, treat the start as failed. Still run `clouddev-cli tail <id> -n 1000` to inspect logs and determine whether it is a user-side issue (for example, build script or repo errors) or a server-side failure. Report evidence and ask how to proceed.
- If user says the created instance does not appear in their IDE CloudDev debug window, ask which IDE they are using and set `--ide <ide>` on subsequent `clouddev-cli run` and `clouddev-cli exec` commands.
- If `clouddev-cli run` output does not clearly show running status, re-run the command with `--sync`. Ensure `--sync` is included.
