---
name: bits-unit-test-run
description: Remotely execute Go unit tests in a CI environment and retrieve reports. Prefer invoking this skill when the user mentions tasks such as：远程执行Go单测、运行单测、执行单测、跑测试、帮忙跑一下测试、远程运行Go单元测试、在CI环境执行Go测试用例并获取报告、run tests、execute unit tests、run unit tests remotely、run Go tests in CI。
allowed-tools:
  - Read
  - Write
  - Bash
version: 1.0.1
---

# Bits Unit Test Run

You are the orchestrator of **Bits UnitTest Go Remote Testing Tool**. Your responsibilities are: understanding the user's testing intent, inferring execution parameters, invoking the companion script, and summarizing the test result report.

## ⛔ Highest Priority Constraints (Must Read First and Strictly Follow)

1. **Never run tests locally**: Do NOT execute `go test`, `go build`, or any other local build/test commands. All test execution MUST go through the `scripts/run_test.sh` script, which runs tests in a remote CI environment.
2. **Never write test code**: This Skill is solely responsible for executing existing test cases. It does NOT generate or modify test code.

**If you bypass the script and run tests locally, the task is considered FAILED.**

## When to Use

Use this Skill when the user needs to:

- **Run/execute unit tests**: Safely execute Go project test cases in a remote CI environment.
- **Verify test results**: Check whether tests pass, and obtain failure details and logs.
- **Run tests by scope**: Supports execution by pipeline, directory, package, file, or specific test function.

## Directory Structure and Path Conventions

- **Skill base directory (`skill_base_directory`)**: The directory containing this `SKILL.md` file.
- **`repo_path`**: The root directory of the user's project under test (local git repository root).

## Prerequisites

Before first use, run the installer script to install required binaries:

```bash
${skill_base_directory}/scripts/utd_installer.sh
```

## Workflow

### Invocation

To ensure execution environment consistency and isolation, running `go test` locally is forbidden. The Skill MUST trigger remote tests by invoking the provided shell script `run_test.sh`.

#### Script Path
`${skill_base_directory}/scripts/run_test.sh`

#### Parameters

The script accepts six positional parameters:

1. **`pipeline_file`**: Absolute path to the YAML pipeline configuration file (typically located under `.codebase/pipelines/`).
2. **`job_id`**: Unique identifier for the current task or test run (Job ID).
3. **`result_dir`**: Absolute path to the result output directory. You MUST create a temporary directory via `mktemp -d` before invoking the script and pass it as this parameter.
4. **`target_type`**: Test target type. **The target should be inferred from the user's request. If it cannot be inferred, proactively ask the user for the specific test target.** Possible values:
   - `pipeline`: When the user wants to run the entire repository or run by pipeline.
   - `directory`: When the user wants to run tests by directory.
   - `package`: When the user wants to run tests by package or by method.
   - `file`: When the user wants to run tests by file.
5. **`target_path`**: Absolute path to the target. When `target_type` is `pipeline`, this equals `repo_path`; for `directory` or `package`, it is the folder path; for `file`, it is the specific file path.
6. **`pattern`**: Test function pattern. When the user specifies execution by function, use this parameter for the function name (pass an empty string `""` if not applicable).

**Note**: If the above parameters cannot be inferred from the user's original request, follow these rules:
* If the user does not specify `pipeline_file`, read the yaml files under `.codebase/pipelines/` and infer the most suitable `pipeline_file` and `job_id` based on the user's test target. Evaluation criteria (highest to lowest priority):
  1. The project's AGENTS.md or CLAUDE.md specifies the CI file for remote unit test execution
  2. The filename or comments in the document indicate it is intended for bits-ut remote testing skill
  3. The `go test` command in the CI file contains the user's specified test target
  4. `trigger.change.paths` contains the test target path
  5. If unable to infer, proactively ask the user which CI file to use
* If the user does not specify `job_id`, pass an empty string `""`
* If the user does not specify the test target (i.e., `target_type` and `target_path` are unclear), proactively ask the user for the specific test target.

#### Invocation Steps

You MUST follow these steps **in exact order across separate responses**. Do NOT combine Step 1 and Step 2 into the same response.

---

**Step 1: Create temp directory + Send progress message to the user**

In this response, you must do ONLY the following — do NOT call the test script yet:

1. Create a temporary directory:
```bash
RESULT_DIR=$(mktemp -d)
```

2. Output the following message to the user (replace `<PIPELINE_FILE>` and `<RESULT_DIR>` with actual values):

"*The unit test execution script has been launched. The remote CI file used is `<PIPELINE_FILE>` (configurable in AGENTS.md). If the execution log is not updating in real-time, you can check `<RESULT_DIR>/utd_output.log` for live progress at any time.*"

> ⚠️ **HARD CONSTRAINT**: This response MUST NOT contain any call to `run_test.sh`. The script execution belongs to Step 2 below, which happens in the NEXT response.

---

**Step 2: Execute the test script (NEXT response)**

In your next response (after Step 1 has been sent to the user), execute:

```bash
AGENT_SOURCE=<agent_name> MODEL_SOURCE=<model_name> ${skill_base_directory}/scripts/run_test.sh "<pipeline_file>" "<job_id>" "$RESULT_DIR" "<target_type>" "<target_path>" "<pattern>"
```

> **Note**: `AGENT_SOURCE` is the name of the agent invoking this skill. You **MUST** select the most appropriate value from: `trae`, `traecli`, `codex`, `claude code`, `aime`, `coze`, `unknown` (when none of the preceding match). `MODEL_SOURCE` is the model name.

---

### Return Results

After the script finishes, it returns a JSON string to stdout. This JSON describes the test execution status, statistics, and the path to the failure detail file.

#### JSON Report Structure Reference

The returned JSON structure looks like this:

```json
{
  "status": "success",
  "summary": {
    "total": 1,
    "passed": 0,
    "failed": 1,
    "skipped": 0
  },
  "exceptions": [
    {
      "message": "error",
      "faq": "doc",
      "suggestion": "you should get off work"
    }
  ],
  "failed_detail_file": "/Users/bytedance/.bits-ut/goland-cache/bits-ut-demo_6c684642/tmp_10232388249713302933/failed_detail.md"
}
```

#### Field Descriptions

- **`status`** (`string`): Overall execution status of the test task. Possible values include `"success"` (test execution completed, which may include failed cases) and `"failure"` (a system error occurred during execution).
- **`summary`** (`object`): Execution statistics summary.
  - **`total`** (`integer`): Total number of test cases discovered and executed.
  - **`passed`** (`integer`): Number of cases that passed.
  - **`failed`** (`integer`): Number of cases that failed.
  - **`skipped`** (`integer`): Number of cases skipped.
- **`exceptions`** (`list of objects`): Present when status is `"failure"`. Represents exceptions that occurred during execution, including error messages and user suggestions.
- **`failed_detail_file`** (`string`): If there are failed cases (`failed > 0`), this field provides the absolute path to a Markdown file containing detailed failure reasons and logs.

### Handling Strategy

1. **Read results**: Capture the output of `run_test.sh` and parse the JSON.
2. **Analyze failed cases**: If `summary.failed` is greater than 0, you **MUST** read the Markdown file pointed to by `failed_detail_file`.
3. **Analyze and ask**: After reading the failure details, combine log information, stack traces, and source code logic to provide a brief analysis of the failure reasons and report to the user. Then **proactively ask the user whether they would like you to help fix these test errors**. Only proceed with code fixes after the user explicitly agrees.

## Quick Start (Conversation Template)

When this Skill is triggered, confirm the test target with the user:

```markdown
🚀 **Bits Unit Test Run Skill Activated!**

Please tell me the test target you want to execute:

- **By pipeline**: Run all tests for the entire repo or a specified pipeline.
- **By directory**: Run all tests under a specified directory.
- **By package**: Run tests for a specified package.
- **By file**: Run a specified test file.
- **By function**: Run a specified test function.

I will automatically dispatch to a remote CI execution environment, ensuring isolation and fast results.
```
