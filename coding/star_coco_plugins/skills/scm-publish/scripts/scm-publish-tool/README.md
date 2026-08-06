# SCM Publish Tool

CLI tool for automating SCM repository publishing workflow for ByteDance internal CI/CD.

## Features

- Create SCM build versions via SCM OpenAPI
- Personal JWT authentication (no shared Bearer token)
- Auto-detect username from JWT payload
- Poll build status with automatic retry
- Retrieve and format build logs on failure
- Support for branch-based and version-based builds

## Installation

```bash
cd scripts/scm-publish-tool
uv sync
```

## Usage

### Create a new build from branch

```bash
uv run python main.py --jwt <jwt_token> --repo code_forge/pipeline/worker --branch feature/auth --type offline
```

### Query existing version status

```bash
uv run python main.py --jwt <jwt_token> --repo code_forge/pipeline/worker --version 3.0.4.4209
```

### Parameters

| Parameter | Short | Required | Default | Description |
|-----------|-------|----------|---------|-------------|
| `--jwt` | - | No | `SCM_JWT_TOKEN` env var | Personal JWT token |
| `--repo` | `-r` | Yes | - | Repository name (e.g., `code_forge/pipeline/worker`) |
| `--branch` | `-b` | Yes* | - | Branch name to build |
| `--version` | - | Yes* | - | Version number to query |
| `--type` | `-t` | No | `offline` | Build type: `online`, `offline`, or `test` |
| `--user` | - | No | auto from JWT | Creator user (auto-detected from JWT payload) |
| `--verbose` | `-v` | No | `false` | Enable verbose logging |

*Either `--branch` or `--version` is required (mutually exclusive).

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `SCM_JWT_TOKEN` | No | Personal JWT token (alternative to `--jwt` flag) |
| `SCM_LOG_OUTPUT_DIR` | No | Directory for build log output (default: `./build-logs`) |

## Build Log Output

When a build fails, logs are saved to:
- `{SCM_LOG_OUTPUT_DIR}/{version}-{build_num}.log` - Filtered log
- `{SCM_LOG_OUTPUT_DIR}/{version}-{build_num}-unfiltered.log` - Full log

Default location: `./build-logs/` relative to tool directory.

## Integration with Claude Code

This tool is designed to be used as a Claude Code skill. The skill definition is located at `skills/scm-publish/SKILL.md`.

Navigate to tool directory and run:
```bash
cd ${CLAUDE_PLUGIN_ROOT}/scripts/scm-publish-tool
uv run python main.py --jwt <jwt_token> --repo <repo> --branch <branch>
```
