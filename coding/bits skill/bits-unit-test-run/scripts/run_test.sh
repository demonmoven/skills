#!/usr/bin/env bash

set -e

# ==============================================================================
# Module: Argument Parsing
# ==============================================================================

if [ "$#" -lt 3 ]; then
    echo "Usage: $0 <pipeline_file> <job_id> <result_dir> [target_type] [target_path] [pattern]"
    echo "  pipeline_file: Absolute path to the pipeline configuration file"
    echo "  job_id:        Job identifier in pipeline_file"
    echo "  result_dir:    Absolute path to the result output directory (created by caller via mktemp -d)"
    echo "  target_type:   pipeline | directory | package | file (inferred from user request)"
    echo "  target_path:   Absolute path to the target (same as repo_path for pipeline, folder path for directory/package, file path for file)"
    echo "  pattern:       Function name pattern when executing by specific function"
    exit 1
fi

PIPELINE_FILE="$1"
JOB_ID="$2"
RESULT_DIR="$3"
TARGET_TYPE="${4:-pipeline}"
TARGET_PATH="${5:-}"
PATTERN="${6:-}"

# Validate pipeline_file is an absolute path
if [[ "$PIPELINE_FILE" != /* ]]; then
    echo "Error: pipeline_file must be an absolute path ($PIPELINE_FILE)"
    exit 1
fi

# Infer REPO_PATH (two levels up from pipeline_file)
REPO_PATH=$(dirname $(dirname "$(dirname "$PIPELINE_FILE")"))
if [ -z "$TARGET_PATH" ]; then
    TARGET_PATH="$REPO_PATH"
fi

# Validate target_path is an absolute path
if [[ "$TARGET_PATH" != /* ]]; then
    echo "Error: target_path must be an absolute path ($TARGET_PATH)"
    exit 1
fi

UTD_PATH="${HOME}/.bits-ut/utd"

if [ ! -x "$UTD_PATH" ]; then
    # Output standard error JSON when environment is not ready
    cat <<EOF
{
  "status": "error",
  "summary": {
    "total": 0,
    "passed": 0,
    "failed": 0,
    "skipped": 0
  },
  "error_message": "utd not installed"
}
EOF
    exit 1
fi

# ==============================================================================
# Module: Test Execution
# ==============================================================================
# Validate result_dir is an absolute path
if [[ "$RESULT_DIR" != /* ]]; then
    echo "Error: result_dir must be an absolute path ($RESULT_DIR)"
    exit 1
fi
mkdir -p "$RESULT_DIR"

STATUS_FILE="${RESULT_DIR}/status.json"

# Build common arguments
BASE_ARGS="remote_test --pipeline_file=${PIPELINE_FILE} --job_id=${JOB_ID} --result_dir=${RESULT_DIR} --status_file=${STATUS_FILE} --save_test_log_mode=1 --agent --update"

# Build command based on target_type
CMD_ARGS="$BASE_ARGS"

case "$TARGET_TYPE" in
    package)
        if [ -n "$PATTERN" ]; then
            PATTERN_ARG="--pattern=\"^\Q${PATTERN}\E$\""
        else
            PATTERN_ARG=""
        fi

        CMD_ARGS="$CMD_ARGS --test_kind=package --package_path=\"$TARGET_PATH\" $PATTERN_ARG --working_directory=\"$TARGET_PATH\""
        ;;
    directory)
        CMD_ARGS="$CMD_ARGS --test_kind=directory --directory=\"$TARGET_PATH\" --working_directory=\"$TARGET_PATH\""
        ;;
    pipeline)
        CMD_ARGS="$CMD_ARGS --test_kind=pipeline --directory=\"$REPO_PATH\" --working_directory=\"$REPO_PATH\""
        ;;
    file)
        # For file type, target_path is the file path; working_directory is its parent directory
        WORKING_DIR=$(dirname "$TARGET_PATH")
        CMD_ARGS="$CMD_ARGS --test_kind=file --files=\"$TARGET_PATH\" --working_directory=\"$WORKING_DIR\""
        ;;
    *)
        echo "Unknown target_type: $TARGET_TYPE"
        exit 1
        ;;
esac

# Execute and redirect output to log file
cd "$REPO_PATH"
eval "\"$UTD_PATH\" $CMD_ARGS" > "$RESULT_DIR/utd_output.log" 2>&1

# ==============================================================================
# Module: Output Results
# ==============================================================================
REPORT_FILE="${RESULT_DIR}/agent_report.json"
if [ -f "$REPORT_FILE" ]; then
    cat "$REPORT_FILE"
else
    # Report file not generated due to an exception
    cat <<EOF
{
  "status": "error",
  "summary": {
    "total": 0,
    "passed": 0,
    "failed": 0,
    "skipped": 0
  },
  "error_message": "agent_report.json file not generated"
}
EOF
fi
