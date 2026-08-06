#!/usr/bin/env bash
# validate-summary-json.sh — 校验 summary.json 结构完整性
#
# 用法: validate-summary-json.sh <summary_json_path>
# 退出码: 0 = 有效, 1 = 无效

set -euo pipefail

SUMMARY_PATH="${1:?用法: validate-summary-json.sh <summary_json_path>}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCHEMA_PATH="$SCRIPT_DIR/../references/summary.schema.json"

# 检查文件是否存在
if [[ ! -f "$SUMMARY_PATH" ]]; then
    echo "ERROR: 文件不存在: $SUMMARY_PATH"
    exit 1
fi

# 检查文件是否为有效 JSON
if ! python3 -c "import json; json.load(open('$SUMMARY_PATH'))" 2>/dev/null; then
    echo "ERROR: 无效的 JSON 格式: $SUMMARY_PATH"
    exit 1
fi

# 校验 schema 和业务规则
if ! VALIDATION=$(python3 - "$SUMMARY_PATH" "$SCHEMA_PATH" << 'PYEOF'
import json, re, sys

try:
    with open(sys.argv[1], encoding="utf-8") as f:
        data = json.load(f)
except Exception as e:
    print(f"ERROR: 无法解析 JSON: {e}")
    sys.exit(1)

try:
    with open(sys.argv[2], encoding="utf-8") as f:
        schema = json.load(f)
except Exception as e:
    print(f"ERROR: 无法解析 schema: {e}")
    sys.exit(1)

errors = []

def validate_schema(schema_node, value, path):
    schema_type = schema_node.get("type")
    if schema_type == "object":
        if not isinstance(value, dict):
            errors.append(f"字段类型错误: {path} 应为 object")
            return
        required = schema_node.get("required", [])
        for field in required:
            if field not in value:
                errors.append(f"缺少必填字段: {path}.{field}" if path else f"缺少必填字段: {field}")
        properties = schema_node.get("properties", {})
        additional = schema_node.get("additionalProperties", True)
        for key, item in value.items():
            child_path = f"{path}.{key}" if path else key
            if key in properties:
                validate_schema(properties[key], item, child_path)
            elif not additional:
                errors.append(f"存在未定义字段: {child_path}")
        return
    if schema_type == "array":
        if not isinstance(value, list):
            errors.append(f"字段类型错误: {path} 应为 array")
            return
        item_schema = schema_node.get("items")
        if item_schema is not None:
            for index, item in enumerate(value):
                validate_schema(resolve_ref(item_schema), item, f"{path}[{index}]")
        return
    if schema_type == "string":
        if not isinstance(value, str):
            errors.append(f"字段类型错误: {path} 应为 string")
            return
        if "minLength" in schema_node and len(value) < schema_node["minLength"]:
            errors.append(f"字段长度错误: {path} 长度不能小于 {schema_node['minLength']}")
        if "enum" in schema_node and value not in schema_node["enum"]:
            errors.append(f"字段枚举错误: {path}={value} 不在允许值 {schema_node['enum']} 中")
        if "pattern" in schema_node and re.match(schema_node["pattern"], value) is None:
            errors.append(f"字段格式错误: {path} 不匹配模式 {schema_node['pattern']}")
        return
    if schema_type == "integer":
        if not isinstance(value, int) or isinstance(value, bool):
            errors.append(f"字段类型错误: {path} 应为 integer")
            return
        if "minimum" in schema_node and value < schema_node["minimum"]:
            errors.append(f"字段取值错误: {path} 不能小于 {schema_node['minimum']}")
        if "maximum" in schema_node and value > schema_node["maximum"]:
            errors.append(f"字段取值错误: {path} 不能大于 {schema_node['maximum']}")
        return
    if schema_type == "number":
        if not isinstance(value, (int, float)) or isinstance(value, bool):
            errors.append(f"字段类型错误: {path} 应为 number")
            return
        if "minimum" in schema_node and value < schema_node["minimum"]:
            errors.append(f"字段取值错误: {path} 不能小于 {schema_node['minimum']}")
        if "maximum" in schema_node and value > schema_node["maximum"]:
            errors.append(f"字段取值错误: {path} 不能大于 {schema_node['maximum']}")
        return
    if schema_type == "boolean":
        if not isinstance(value, bool):
            errors.append(f"字段类型错误: {path} 应为 boolean")
        return

def resolve_ref(schema_node):
    ref = schema_node.get("$ref")
    if not ref:
        return schema_node
    if not ref.startswith("#/$defs/"):
        errors.append(f"不支持的 schema 引用: {ref}")
        return {}
    key = ref.split("/")[-1]
    return schema.get("$defs", {}).get(key, {})

validate_schema(schema, data, "")

subsystem_prefix_map = {
    "rule_checks": "RC-",
    "structural_tests": "ST-",
    "task_evals": "TE-",
    "trace_grading": "TG-",
    "regression_gates": "RG-",
}

findings_count = data.get("findings_count")
findings = data.get("findings", [])
if isinstance(findings_count, dict) and isinstance(findings, list):
    actual_count = {"critical": 0, "high": 0, "medium": 0, "low": 0}
    expected_prefix = subsystem_prefix_map.get(data.get("subsystem"))
    for index, finding in enumerate(findings):
        if not isinstance(finding, dict):
            continue
        severity = finding.get("severity")
        if severity in actual_count:
            actual_count[severity] += 1
        finding_id = finding.get("id")
        if expected_prefix and isinstance(finding_id, str) and not finding_id.startswith(expected_prefix):
            errors.append(f"字段格式错误: findings[{index}].id 必须以 {expected_prefix} 开头")
    for severity, count in actual_count.items():
        if findings_count.get(severity) != count:
            errors.append(f"findings_count.{severity}={findings_count.get(severity)} 与 findings 实际数量 {count} 不一致")

if errors:
    for e in errors:
        print(f"ERROR: {e}")
    sys.exit(1)
else:
    print("OK: summary.json 校验通过")
    sys.exit(0)
PYEOF
); then
    echo "$VALIDATION"
    exit 1
fi

echo "$VALIDATION"
# 如果 python 返回非 0，本脚本也返回非 0
