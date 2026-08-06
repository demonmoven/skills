#!/usr/bin/env bash
# validate-gate-policies.sh — 校验 gate-policies.json 结构完整性
#
# 用法: validate-gate-policies.sh <gate_policies_json_path>
# 退出码: 0 = 有效, 1 = 无效

set -euo pipefail

POLICIES_PATH="${1:?用法: validate-gate-policies.sh <gate_policies_json_path>}"

# 检查文件是否存在
if [[ ! -f "$POLICIES_PATH" ]]; then
    echo "ERROR: 文件不存在: $POLICIES_PATH"
    exit 1
fi

# 检查文件是否为有效 JSON
if ! python3 -c "import json; json.load(open('$POLICIES_PATH'))" 2>/dev/null; then
    echo "ERROR: 无效的 JSON 格式: $POLICIES_PATH"
    exit 1
fi

# 校验结构和业务规则
if ! VALIDATION=$(python3 - "$POLICIES_PATH" << 'PYEOF'
import json, sys

try:
    with open(sys.argv[1], encoding="utf-8") as f:
        data = json.load(f)
except Exception as e:
    print(f"ERROR: 无法解析 JSON: {e}")
    sys.exit(1)

errors = []

# 检查顶层字段
required_top = ["levels", "verdict_rules", "scoring", "subsystem_metadata"]
for field in required_top:
    if field not in data:
        errors.append(f"缺少顶层字段: {field}")

# 检查 levels
levels = data.get("levels", {})
required_levels = ["pr", "nightly", "release"]
for level in required_levels:
    if level not in levels:
        errors.append(f"缺少门禁级别: levels.{level}")
        continue
    level_data = levels[level]
    # 检查必需字段
    for field in ["subsystems", "thresholds", "timeout_minutes"]:
        if field not in level_data:
            errors.append(f"缺少字段: levels.{level}.{field}")
    # 检查 thresholds
    thresholds = level_data.get("thresholds", {})
    for field in ["min_score", "max_critical"]:
        if field not in thresholds:
            errors.append(f"缺少字段: levels.{level}.thresholds.{field}")
    # 检查 min_score 范围
    min_score = thresholds.get("min_score", 0)
    if not isinstance(min_score, int) or min_score < 0 or min_score > 100:
        errors.append(f"levels.{level}.thresholds.min_score 必须为 0-100 的整数")
    # 检查 subsystems 是有效值
    valid_subsystems = {"rule_checks", "structural_tests", "task_evals", "trace_grading", "regression_gates"}
    for sub in level_data.get("subsystems", []):
        if sub not in valid_subsystems:
            errors.append(f"levels.{level}.subsystems 包含无效子系统: {sub}")

# 检查 scoring
scoring = data.get("scoring", {})
for mode in ["gate_mode", "llm_mode"]:
    if mode not in scoring:
        errors.append(f"缺少字段: scoring.{mode}")
        continue
    mode_data = scoring[mode]
    if "base" not in mode_data:
        errors.append(f"缺少字段: scoring.{mode}.base")
    if "weights" not in mode_data:
        errors.append(f"缺少字段: scoring.{mode}.weights")
    else:
        weights = mode_data["weights"]
        for sev in ["critical", "high", "medium", "low"]:
            if sev not in weights:
                errors.append(f"缺少字段: scoring.{mode}.weights.{sev}")

# 检查 subsystem_metadata
metadata = data.get("subsystem_metadata", {})
expected_prefixes = {
    "rule_checks": "RC-",
    "structural_tests": "ST-",
    "task_evals": "TE-",
    "trace_grading": "TG-",
    "regression_gates": "RG-",
}
for sub, prefix in expected_prefixes.items():
    if sub not in metadata:
        errors.append(f"缺少字段: subsystem_metadata.{sub}")
    elif metadata[sub].get("id_prefix") != prefix:
        errors.append(f"subsystem_metadata.{sub}.id_prefix 应为 {prefix}")

# 检查阈值递增: pr < nightly < release
if all(l in levels for l in required_levels):
    scores = [levels[l].get("thresholds", {}).get("min_score", 0) for l in required_levels]
    if not (scores[0] <= scores[1] <= scores[2]):
        errors.append(f"阈值应递增: pr({scores[0]}) <= nightly({scores[1]}) <= release({scores[2]})")

if errors:
    for e in errors:
        print(f"ERROR: {e}")
    sys.exit(1)
else:
    print("OK: gate-policies.json 校验通过")
    sys.exit(0)
PYEOF
); then
    echo "$VALIDATION"
    exit 1
fi

echo "$VALIDATION"
