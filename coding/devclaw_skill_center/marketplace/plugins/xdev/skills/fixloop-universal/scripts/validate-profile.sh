#!/bin/bash
# validate-profile.sh - Validate a fixloop-profile.yaml
# Usage: bash validate-profile.sh <path-to-profile>
#
# NOTE: This is a basic smoke-test that checks whether expected field names
# appear somewhere in the YAML file. It does NOT validate YAML hierarchy,
# nesting depth, or value types. A field like "start_command" will be
# detected regardless of whether it sits under "dev_server" or elsewhere.
# For full structural validation, use a proper YAML schema validator.

set -euo pipefail

PROFILE="${1:?Usage: validate-profile.sh <path-to-profile.yaml>}"

if [ ! -f "$PROFILE" ]; then
  echo "ERROR: Profile not found: $PROFILE"
  exit 1
fi

echo "Validating fixloop profile: $PROFILE"
echo "========================================"

ERRORS=0
WARNINGS=0

# Helper: check if a YAML key name appears in the file.
# This is a flat text search -- it cannot verify nesting or hierarchy.
# A match means the key name is present *somewhere* in the file, which is
# sufficient for a smoke test but not for structural validation.
check_required() {
  local key="$1"
  local desc="$2"
  if ! grep -qE "^\s*${key}:" "$PROFILE"; then
    echo "  ERROR: Required field '$key' not found anywhere in profile ($desc)"
    ERRORS=$((ERRORS + 1))
  fi
}

check_optional() {
  local key="$1"
  local desc="$2"
  if ! grep -qE "^\s*${key}:" "$PROFILE"; then
    echo "  WARN:  Optional field '$key' not found in profile ($desc)"
    WARNINGS=$((WARNINGS + 1))
  fi
}

echo ""
echo "[1/8] Project metadata..."
check_required "name" "project name"
check_required "app_path" "main app directory"
check_required "framework" "UI framework (react/vue/...)"
check_required "build_tool" "build tool (vite/webpack/...)"

echo ""
echo "[2/8] Setup commands..."
check_required "install_command" "dependency install command"
check_required "start_command" "dev server start command"
check_required "port_injection" "how to inject port into dev server"
check_optional "build_command" "pre-dev build command"
check_optional "mock_server" "mock server config"

echo ""
echo "[3/8] Authentication..."
check_required "strategy" "auth strategy (none/form_login/cookie_inject/...)"

echo ""
echo "[4/8] Testing config..."
check_required "cases_dir" "E2E test cases directory"
check_optional "specs_dir" "spec documents directory"
check_optional "max_concurrency" "parallel Chrome instances"

echo ""
echo "[5/8] Fix loop config..."
check_optional "max_iterations" "max fix iterations"
check_optional "enabled_agents" "code review agents"
check_optional "lint_command" "lint command for fix agent"
check_optional "typecheck_command" "typecheck command for fix agent"

echo ""
echo "[6/8] Browser config (v2)..."
check_optional "mode" "browser launch mode (system_chrome/cdp_connect/bundled/headless)"
# Validate browser.mode enum
BROWSER_MODE=$(grep -E '^\s*mode:\s*' "$PROFILE" | head -1 | sed 's/.*mode:\s*//' | tr -d '[:space:]' | tr -d '"' | tr -d "'")
if [ -n "$BROWSER_MODE" ]; then
  case "$BROWSER_MODE" in
    system_chrome|cdp_connect|bundled|headless) ;;
    *)
      echo "  WARN:  browser.mode '$BROWSER_MODE' is not one of: system_chrome, cdp_connect, bundled, headless"
      WARNINGS=$((WARNINGS + 1))
      ;;
  esac
  # If mode is cdp_connect, cdp_endpoint is required
  if [ "$BROWSER_MODE" = "cdp_connect" ]; then
    if ! grep -qE "^\s*cdp_endpoint:" "$PROFILE"; then
      echo "  WARN:  browser.mode is 'cdp_connect' but cdp_endpoint is not set"
      WARNINGS=$((WARNINGS + 1))
    fi
  fi
fi

echo ""
echo "[7/8] Observation config (v2)..."
check_optional "observation" "observation/recording config"

echo ""
echo "[8/8] Exploration config (v2)..."
check_optional "exploration" "post-test exploration config"
# Validate exploration.strategy enum
EXPLORE_STRATEGY=$(grep -E '^\s*strategy:\s*' "$PROFILE" | head -1 | sed 's/.*strategy:\s*//' | tr -d '[:space:]' | tr -d '"' | tr -d "'")
if [ -n "$EXPLORE_STRATEGY" ]; then
  case "$EXPLORE_STRATEGY" in
    a11y_driven|boundary|chaos) ;;
    *)
      echo "  WARN:  exploration.strategy '$EXPLORE_STRATEGY' is not one of: a11y_driven, boundary, chaos"
      WARNINGS=$((WARNINGS + 1))
      ;;
  esac
fi

echo ""
echo "========================================"
echo "Errors:   $ERRORS"
echo "Warnings: $WARNINGS"

if [ "$ERRORS" -gt 0 ]; then
  echo "RESULT: FAIL - Fix the errors above before running fixloop"
  exit 1
else
  echo "RESULT: PASS - Profile is valid (review warnings if any)"
  exit 0
fi
