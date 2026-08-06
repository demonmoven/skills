#!/usr/bin/env bash

if [ -n "$ZSH_VERSION" ]; then
    exec bash "$0" "$@"
fi

set -e

UTREE_TARGET="$HOME/.local/bin/utree"
UTREE_EXISTS_BEFORE="false"
UTREE_BEGIN_EXIT_CODE=""
UTREE_BEGIN_STDERR=""
if [[ -f "$UTREE_TARGET" && -x "$UTREE_TARGET" ]]; then
    UTREE_EXISTS_BEFORE="true"
fi

json_escape() {
    local value="${1:-}"
    value="${value//\\/\\\\}"
    value="${value//\"/\\\"}"
    value="${value//$'\n'/\\n}"
    value="${value//$'\r'/\\r}"
    value="${value//$'\t'/\\t}"
    printf '%s' "$value"
}

current_time_ms() {
    if command -v python3 >/dev/null 2>&1; then
        python3 -c 'import time; print(int(time.time() * 1000))'
        return 0
    fi

    local millis
    millis="$(date +%s%3N 2>/dev/null || true)"
    if [[ "$millis" =~ ^[0-9]+$ ]]; then
        printf '%s' "$millis"
        return 0
    fi

    printf '%s000' "$(date +%s)"
}

get_cpu_arch() {
    local machine
    machine="$(uname -m 2>/dev/null || true)"

    case "$machine" in
        x86_64|i386)
            printf 'intel'
            ;;
        arm64|aarch64)
            printf 'arm64'
            ;;
        "")
            printf 'unknown'
            ;;
        *)
            printf '%s' "$machine"
            ;;
    esac
}

get_skill_version() {
    local skill_file="${SKILL_ROOT:-}/SKILL.md"
    if [[ ! -r "$skill_file" ]]; then
        printf ''
        return 0
    fi

    awk '
        /^metadata:[[:space:]]*$/ { in_metadata = 1; next }
        in_metadata && /^[^[:space:]]/ { exit }
        in_metadata && /^[[:space:]]+version:[[:space:]]*/ {
            value = $0
            sub(/^[[:space:]]+version:[[:space:]]*/, "", value)
            gsub(/^["'"'"']|["'"'"']$/, "", value)
            print value
            exit
        }
    ' "$skill_file"
}

get_event_user() {
    local user=""
    if [[ -n "${USER_CLOUD_JWT:-}" ]]; then
        local payload encoded_payload payload_len padding decoded_payload
        encoded_payload="${USER_CLOUD_JWT#*.}"
        encoded_payload="${encoded_payload%%.*}"
        payload="$(printf '%s' "$encoded_payload" | tr '_-' '/+')"
        payload_len=${#payload}
        padding=$(( (4 - payload_len % 4) % 4 ))
        if (( padding > 0 )); then
            payload+="$(printf '%*s' "$padding" '' | tr ' ' '=')"
        fi
        decoded_payload="$(printf '%s' "$payload" | base64 -d 2>/dev/null || printf '%s' "$payload" | base64 -D 2>/dev/null || true)"
        user="$(printf '%s' "$decoded_payload" | sed -n 's/.*"username"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
    fi
    if [[ -z "$user" ]]; then
        local git_email
        git_email="$(git config user.email 2>/dev/null || true)"
        user="${git_email%@*}"
        user="${user%__dcar}"
    fi

    printf '%s' "${user:-${USER:-}}"
}

get_event_context_attrs() {
    local agent="${AGENT_SOURCE:-}"
    if [[ -z "$agent" || "$agent" == "unknown" ]]; then
        agent="${AI_AGENT:-unknown}"
    fi
    agent="$(printf '%s' "$agent" | tr '[:upper:]' '[:lower:]')"

    local model="${MODEL_SOURCE:-unknown}"
    model="$(printf '%s' "$model" | tr '[:upper:]' '[:lower:]')"

    printf '"agent":"%s","model":"%s","exec_source":"%s","exec_session_id":"%s","ci_task_source":"%s","tmp_root":"%s","bits_ut_sid":"%s","pwd":"%s","ut_agent_assignment_id":"%s","eval_task_id":"%s","eval_subtask_id":"%s"' \
        "$(json_escape "$agent")" \
        "$(json_escape "$model")" \
        "$(json_escape "${EXEC_SOURCE:-}")" \
        "$(json_escape "${EXEC_SESSION_ID:-}")" \
        "$(json_escape "${UT_AGENT_TRIGGER_SOURCE:-}")" \
        "$(json_escape "${TMP_ROOT:-}")" \
        "$(json_escape "${BITS_UT_SID:-}")" \
        "$(json_escape "$(pwd)")" \
        "$(json_escape "${UT_AGENT_ASSIGNMENT_ID:-}")" \
        "$(json_escape "${EVAL_TASK_ID:-}")" \
        "$(json_escape "${EVAL_SUBTASK_ID:-}")"
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --repo-path)
                if [[ -z "${2:-}" ]]; then
                    REPO_PATH=""
                    shift
                    continue
                fi
                REPO_PATH="$2"
                shift 2
                ;;
            *)
                echo "Error: unknown argument: $1"
                exit 1
                ;;
        esac
    done

    REPO_PATH="${REPO_PATH:-}"
}

should_skip_event_report() {
    [[ -n "${UT_AGENT_AUTH_HEADER:-}" ]]
}

ensure_tmp_root() {
    if [[ -z "${TMP_ROOT:-}" ]]; then
        TMP_ROOT=$(mktemp -d)
    fi
    export TMP_ROOT
}

ensure_bits_ut_sid() {
    if [[ -n "${BITS_UT_SID:-}" ]]; then
        export BITS_UT_SID
        return 0
    fi

    if command -v uuidgen >/dev/null 2>&1; then
        BITS_UT_SID="$(uuidgen | tr '[:upper:]' '[:lower:]')"
    elif [[ -r /proc/sys/kernel/random/uuid ]]; then
        BITS_UT_SID="$(cat /proc/sys/kernel/random/uuid)"
    fi

    export BITS_UT_SID
}

parse_utree_cli_version() {
    local version_output="${1:-}"

    awk '
        /^[[:space:]]*cli version[[:space:]]*:/ {
            value = $0
            sub(/^[^:]*:[[:space:]]*/, "", value)
            gsub(/[[:space:]]+$/, "", value)
            if (value ~ /^[0-9]+(\.[0-9]+)*$/) {
                print value
            }
            exit
        }
    ' <<<"$version_output"
}

version_lte() {
    local version="$1"
    local threshold="$2"

    awk -v version="$version" -v threshold="$threshold" '
        BEGIN {
            version_len = split(version, version_parts, ".")
            threshold_len = split(threshold, threshold_parts, ".")
            max_len = version_len > threshold_len ? version_len : threshold_len

            for (i = 1; i <= max_len; i++) {
                version_part = i in version_parts ? version_parts[i] + 0 : 0
                threshold_part = i in threshold_parts ? threshold_parts[i] + 0 : 0

                if (version_part < threshold_part) {
                    exit 0
                }
                if (version_part > threshold_part) {
                    exit 1
                }
            }

            exit 0
        }
    '
}

update_old_utree_after_begin_failure() {
    local target="$1"
    local version_output cli_version
    local update_threshold="1.0.0.51"

    if [[ "${UTREE_BEGIN_EXIT_CODE:-0}" == "0" ]]; then
        return 0
    fi

    if ! version_output=$("$target" version 2>&1); then
        echo "Warning: failed to check utree version after begin failure; skipping utree update"
        return 0
    fi

    cli_version="$(parse_utree_cli_version "$version_output")"
    if [[ -z "$cli_version" ]]; then
        echo "Warning: failed to parse utree cli version after begin failure; skipping utree update"
        return 0
    fi

    if ! version_lte "$cli_version" "$update_threshold"; then
        echo "Info: utree cli version $cli_version is newer than $update_threshold; skipping utree update"
        return 0
    fi

    echo "Info: utree cli version $cli_version is not newer than $update_threshold; updating utree..."
    if ! "$target" update; then
        echo "Warning: utree update failed; continuing with bootstrap"
        return 0
    fi

    echo "Info: utree update succeeded; retrying utree begin"
    run_begin "$target"
}

run_begin() {
    local target="$1"
    local -a begin_args
    begin_args=(begin)
    local stderr_file begin_exit_code

    if [[ -n "$REPO_PATH" ]]; then
        begin_args+=(--repo-path "$REPO_PATH")
    fi

    if [[ -n "${TMP_ROOT:-}" && -d "$TMP_ROOT" ]]; then
        stderr_file="$TMP_ROOT/utree_begin.stderr"
    else
        stderr_file="$(mktemp 2>/dev/null || true)"
    fi

    if [[ -n "$stderr_file" ]]; then
        : >"$stderr_file" 2>/dev/null || stderr_file=""
    fi

    if [[ -n "$stderr_file" ]]; then
        if AGENT_SOURCE="${AGENT_SOURCE:-unknown}" MODEL_SOURCE="${MODEL_SOURCE:-unknown}" \
            SKILL_ROOT="${SKILL_ROOT:-}" \
            TMP_ROOT="${TMP_ROOT:-}" \
            BITS_UT_SID="${BITS_UT_SID:-}" \
            "$target" "${begin_args[@]}" 2>"$stderr_file"; then
            begin_exit_code=0
        else
            begin_exit_code=$?
        fi
    else
        if AGENT_SOURCE="${AGENT_SOURCE:-unknown}" MODEL_SOURCE="${MODEL_SOURCE:-unknown}" \
            SKILL_ROOT="${SKILL_ROOT:-}" \
            TMP_ROOT="${TMP_ROOT:-}" \
            BITS_UT_SID="${BITS_UT_SID:-}" \
            "$target" "${begin_args[@]}"; then
            begin_exit_code=0
        else
            begin_exit_code=$?
        fi
    fi

    UTREE_BEGIN_EXIT_CODE="$begin_exit_code"
    if [[ -n "$stderr_file" && -r "$stderr_file" ]]; then
        UTREE_BEGIN_STDERR="$(cat "$stderr_file" 2>/dev/null || true)"
    else
        UTREE_BEGIN_STDERR=""
    fi

    if [[ -n "${UTREE_BEGIN_EXIT_CODE_FILE:-}" ]]; then
        printf '%s' "$UTREE_BEGIN_EXIT_CODE" >"$UTREE_BEGIN_EXIT_CODE_FILE" 2>/dev/null || true
    fi
    if [[ -n "${UTREE_BEGIN_STDERR_FILE:-}" ]]; then
        printf '%s' "$UTREE_BEGIN_STDERR" >"$UTREE_BEGIN_STDERR_FILE" 2>/dev/null || true
    fi

    if [[ -n "$UTREE_BEGIN_STDERR" ]]; then
        printf '%s\n' "$UTREE_BEGIN_STDERR" >&2
    fi

    if (( begin_exit_code != 0 )); then
        echo "Warning: utree begin failed; continuing with bootstrap"
    fi
}

ensure_utree_installed_impl() {
    local target="$UTREE_TARGET"
    local cache_dir

    # 检查目标二进制是否存在且有效
    if [[ -f "$target" && -x "$target" ]]; then
        echo "Info: utree already exists at $target"
        run_begin "$target"
        update_old_utree_after_begin_failure "$target"
        return 0
    fi

    # 根据系统和架构确定二进制名称
    local binary_name
    if [[ "$OSTYPE" == "darwin"* ]]; then
        binary_name="utree_darwin_arm64"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        binary_name="utree_linux_amd64"
    else
        echo "Error: unsupported OS ($OSTYPE)"
        return 1
    fi

    echo "Info: fetching latest utree version..."
    local api_url="https://scm.byted.org/api/v2/versions/latest_version/?repo_id=544895&status=build_ok&type=online"
    local tar_url
    local api_response
    api_response=$(curl -s "$api_url")
    if [[ "$OSTYPE" == "darwin"* ]]; then
        tar_url=$(echo "$api_response" | grep -o '"tar_url_aarch64":"[^"]*"' | head -1 | sed 's/"tar_url_aarch64":"//;s/"//')
    else
        tar_url=$(echo "$api_response" | grep -o '"tar_url":"[^"]*"' | head -1 | sed 's/"tar_url":"//;s/"//')
    fi

    if [[ -z "$tar_url" ]]; then
        echo "Error: failed to get tar_url from API response"
        return 1
    fi

    if ! cache_dir=$(mktemp -d); then
        echo "Error: failed to create temporary directory for utree package"
        return 1
    fi

    echo "Info: downloading utree from $tar_url..."
    if ! LC_ALL=C curl -sL "$tar_url" | LC_ALL=C tar -xz -C "$cache_dir"; then
        echo "Error: failed to download or extract utree package"
        rm -rf "$cache_dir"
        return 1
    fi

    # 找到下载的二进制并复制到脚本目录，重命名为 utree
    local downloaded_binary="$cache_dir/$binary_name"
    if [[ ! -f "$downloaded_binary" ]]; then
        # 尝试在 cache_dir 的子目录中查找
        downloaded_binary=$(find "$cache_dir" -name "$binary_name" -type f 2>/dev/null | head -1)
    fi

    if [[ -z "$downloaded_binary" || ! -f "$downloaded_binary" ]]; then
        echo "Error: $binary_name not found in downloaded package"
        rm -rf "$cache_dir"
        return 1
    fi

    if ! mkdir -p "$(dirname "$target")"; then
        echo "Error: failed to create utree target directory $(dirname "$target")"
        rm -rf "$cache_dir"
        return 1
    fi
    if command -v install >/dev/null 2>&1; then
        if ! install -m 0755 "$downloaded_binary" "$target"; then
            echo "Error: failed to install utree to $target"
            rm -rf "$cache_dir"
            return 1
        fi
    else
        if ! chmod +x "$downloaded_binary"; then
            echo "Error: failed to mark downloaded utree as executable"
            rm -rf "$cache_dir"
            return 1
        fi
        if ! cp -f "$downloaded_binary" "$target"; then
            echo "Error: failed to copy utree to $target"
            rm -rf "$cache_dir"
            return 1
        fi
        if ! chmod 0755 "$target"; then
            echo "Error: failed to set executable permissions on $target"
            rm -rf "$cache_dir"
            return 1
        fi
    fi
    rm -rf "$cache_dir"

    run_begin "$target"

    echo "Info: utree installed to $target"
}

ensure_utree_installed() {
    local start_ms end_ms duration_ms output exit_code status error_message begin_code_file begin_stderr_file
    start_ms="$(current_time_ms)"

    begin_code_file="${TMP_ROOT:-}/utree_begin.exit_code"
    begin_stderr_file="${TMP_ROOT:-}/utree_begin.captured_stderr"
    : >"$begin_code_file" 2>/dev/null || begin_code_file=""
    : >"$begin_stderr_file" 2>/dev/null || begin_stderr_file=""
    UTREE_BEGIN_EXIT_CODE_FILE="$begin_code_file"
    UTREE_BEGIN_STDERR_FILE="$begin_stderr_file"

    set +e
    output="$(ensure_utree_installed_impl 2>&1)"
    exit_code=$?
    set -e

    if [[ -n "$begin_code_file" && -r "$begin_code_file" ]]; then
        UTREE_BEGIN_EXIT_CODE="$(cat "$begin_code_file" 2>/dev/null || true)"
        rm -f "$begin_code_file"
    fi
    if [[ -n "$begin_stderr_file" && -r "$begin_stderr_file" ]]; then
        UTREE_BEGIN_STDERR="$(cat "$begin_stderr_file" 2>/dev/null || true)"
        rm -f "$begin_stderr_file"
    fi
    unset UTREE_BEGIN_EXIT_CODE_FILE UTREE_BEGIN_STDERR_FILE

    end_ms="$(current_time_ms)"
    duration_ms=$(( end_ms - start_ms ))
    if (( duration_ms < 0 )); then
        duration_ms=0
    fi

    if [[ -n "$output" ]]; then
        printf '%s\n' "$output"
    fi

    if (( exit_code == 0 )); then
        if [[ "$UTREE_EXISTS_BEFORE" == "true" ]]; then
            status="already_exists"
        else
            status="installed"
        fi
        error_message=""
    else
        status="failed"
        error_message="$output"
    fi

    return 0
}

parse_args "$@"
ensure_tmp_root
ensure_bits_ut_sid
ensure_utree_installed &
if [ -n "${GEN_BUG_OUTPUT_FILE_PATH}" ]; then
    echo "EXEC_SOURCE=PIPELINE"
    echo "GEN_BUG_OUTPUT_FILE_PATH=${GEN_BUG_OUTPUT_FILE_PATH}"
elif [[ -r "$HOME/.flux/config.json" ]]; then
    _install_mode="$(sed -n 's/.*"install_mode"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$HOME/.flux/config.json" | head -1)"
    if [[ "$_install_mode" == "web" ]]; then
        echo "EXEC_SOURCE=flux-web"
    fi
    unset _install_mode
fi
echo "BITS_TMP_ROOT=${TMP_ROOT}"
