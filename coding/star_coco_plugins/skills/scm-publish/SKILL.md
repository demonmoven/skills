---
name: scm-publish
description: This skill should be used when the user asks to "publish to SCM", "trigger SCM build", "compile on SCM", "SCM 编译", or needs to create a build version on ByteDance SCM system. Provides guidance for SCM repository publishing workflow.
tags:
  - byte-skill
---

# SCM Publish

Assist users in publishing code to ByteDance SCM (Source Code Management) system and monitoring build status.

## When to Use

- User wants to create a build version on SCM
- User needs to compile code through CI/CD pipeline
- User mentions "SCM publish", "SCM compile", "SCM build", or "SCM 编译"

## When NOT to Use

- User wants to run local tests or local compilation (`go test`, `go build`)
- User is debugging locally and doesn't need CI/CD
- User is working on code that isn't ready for SCM build

## Required Information

Before executing, ensure the following information is available:

1. **Repository name** (required): Full SCM repository path, e.g., `code_forge/pipeline/api`
2. **Branch name** (optional): Target branch to build; if not provided, infer from `git branch --show-current`
3. **Build type** (optional): `offline` (default), `online`, or `test`

If repository name is missing, ask the user to provide it.

## Authentication

This skill requires a personal JWT token. The token is obtained via `byte-cli login` (bytedance-authentication plugin).

### Obtaining JWT Token

Before executing the build tool, ensure a valid JWT token is available:

```bash
# Try to get JWT from cached cookies
byte-cli login --control-plane cn get-jwt
```

If the output shows `"success": true`, extract the `data.jwt` field. If `"success": false`, trigger the QR code login flow per the bytedance-jwt skill.

**The JWT token serves two purposes:**
1. Authentication for SCM API requests (via `x-jwt-token` header)
2. User identity extraction — the tool auto-detects the username from the JWT payload, replacing the hardcoded `root` default

## Execution Steps
0. **install cli**
   - Run `curl -LsSf https://astral.sh/uv/install.sh | sh`
   - Run `uv tool install --force "git+https://code.byted.org/bytedance/byte-skill.git#subdirectory=byte-cli" --index https://bytedpypi.byted.org/simple`

1. **Obtain JWT token**
   - Run `byte-cli login --control-plane cn get-jwt`
   - If success, extract `data.jwt` as the JWT token
   - If failed, follow bytedance-jwt skill to complete QR code login, then retry

2. **Gather and validate parameters**
   - Repository name (required from user)
   - Branch name: if not provided, run `git branch --show-current` to infer

2. **Check for uncommitted changes** (when inferring branch from local git)
   - Run `git status --porcelain` to detect uncommitted changes
   - If uncommitted changes exist, warn the user:
     - SCM builds from remote repository, local uncommitted changes will NOT be included
     - Ask if user wants to commit and push first, or proceed anyway
   - Also check if local branch is ahead of remote (`git status` shows "Your branch is ahead")

3. **Confirm with user before executing**
   - Show repository name
   - Show branch name (indicate if inferred from local)
   - Show build type
   - If there were warnings (uncommitted changes, unpushed commits), remind user

5. **Navigate to tool directory and execute**
   ```bash
   cd ${CLAUDE_PLUGIN_ROOT}/skills/scm-publish/scripts/scm-publish-tool
   uv run python main.py --jwt <jwt_token> --repo <repo> --branch <branch> --type <type>
   ```

6. **Monitor and report progress**
   - Inform user that build has been created
   - Report polling status periodically
   - When complete, summarize the result clearly

## Tool Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `--jwt` | No | `SCM_JWT_TOKEN` env var | Personal JWT token for authentication |
| `--repo` | Yes | - | Repository name (e.g., `code_forge/pipeline/api`) |
| `--branch` | Yes | - | Branch name to build |
| `--type` | No | `offline` | Build type: `online`, `offline`, `test` |
| `--user` | No | auto from JWT | Creator user (auto-detected from JWT payload) |
| `--version` | No | - | Query existing version instead of creating new |

## Handling Results

### On Success
Report to user:
- Repository and branch
- Version number created
- Build log URL for reference

### On Failure
Help user understand the issue:
- Show failed step and build number
- Point to the saved log file location
- Suggest checking the build log for error details
- Offer to read and analyze the log file if user wants

### On Timeout
Inform user:
- Build may still be in progress
- Provide SCM dashboard URL to check status manually

## Examples

### Example 1: User provides full information
**User**: "Publish code_forge/pipeline/api on branch feature/auth to SCM"

**Action**:
1. Obtain JWT token via `byte-cli login --control-plane cn get-jwt`
2. Execute with provided parameters:
```bash
cd ${CLAUDE_PLUGIN_ROOT}/skills/scm-publish/scripts/scm-publish-tool
uv run python main.py --jwt <jwt_token> --repo code_forge/pipeline/api --branch feature/auth
```

### Example 2: User provides only repo, has uncommitted changes
**User**: "SCM compile code_forge/pipeline/worker"

**Action**:
1. Infer branch from `git branch --show-current` → `feature/new-api`
2. Run `git status --porcelain` → detects uncommitted changes
3. Warn user:
   "Current branch is `feature/new-api`. However, I detected uncommitted changes in your working directory:
   - Modified: `src/handler.go`
   - New file: `src/utils.go`

   ⚠️ SCM builds from the remote repository. These local changes will NOT be included in the build.

   Would you like to:
   1. Commit and push these changes first, then build
   2. Proceed with the build anyway (without local changes)"
4. Execute after user confirms their choice

### Example 3: User provides only repo, clean working directory
**User**: "SCM compile code_forge/pipeline/worker"

**Action**:
1. Infer branch from `git branch --show-current`
2. Run `git status --porcelain` → no uncommitted changes
3. Confirm with user: "I'll build `code_forge/pipeline/worker` on branch `master` (current branch). Proceed?"
4. Execute after confirmation

### Example 4: User asks without specifics
**User**: "Help me publish to SCM"

**Action**: Ask for required information:
"To publish to SCM, I need the repository name. What's the full SCM repository path? (e.g., `code_forge/pipeline/api`)"

### Example 5: Build fails
**User**: Gets build failure

**Action**:
1. Report the failure clearly
2. Show log file location
3. Offer: "Would you like me to read the build log and help analyze the error?"

## Notes

- **SCM builds from remote repository**: Local uncommitted changes or unpushed commits will NOT be included in the build
- Build process typically takes 1-5 minutes
- The tool polls every 30 seconds, up to 60 attempts (30 minutes max)
- Build logs are saved to `${CLAUDE_PLUGIN_ROOT}/skills/scm-publish/scripts/scm-publish-tool/build-logs/`
- For dependency issues, run `uv sync` in the tool directory first
- The `--user` flag is auto-detected from the JWT token payload; no need to specify manually
