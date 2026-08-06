#!/usr/bin/env python3
"""
数据处理与飞书多维表格写入脚本

读取 DDA 和 HSQL 的 JSON 数据文件，进行风险分类和依赖表提取，
然后批量写入飞书多维表格。

使用方式：
    python3 scripts/import_to_bitable.py \
        --dda_file ./data/dda_data.json \
        --hsql_file ./data/hsql_data.json \
        --app_token <app_token> \
        --table_id <table_id>

    # 仅创建新表格（不写入已有表格）
    python3 scripts/import_to_bitable.py \
        --dda_file ./data/dda_data.json \
        --hsql_file ./data/hsql_data.json \
        --create_new \
        --app_name "追光清结算核对任务台账"
"""

import os
import sys
import json
import re
import argparse
from datetime import date
from typing import Any, Dict, List, Optional

# 确保可以导入同目录的模块
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from lark_bitable_client import LarkBitableClient


# ===== 风险分类关键词 =====

CONFIG_RISK_KEYWORDS = [
    "规则", "配置", "合约", "有效期", "过期",
    "fee_activity", "template_aggregation", "settle_rule",
    "charge_rule", "settle_code", "factor_match", "settle_template",
    "收费规则", "计费规则", "商户结算信息",
]

TIMELINESS_RISK_KEYWORDS = [
    "卡单", "延迟", "超时", "未终态", "处理中", "时效",
    "recovery", "首笔发现", "首笔监控", "首笔命中",
]

CONSISTENCY_RISK_KEYWORDS = [
    "幂等", "一致性", "vs", "VS", "entry汇总", "核对", "重复",
]

CORRECTNESS_RISK_KEYWORDS = [
    "金额", "退费", "资金流", "资金平衡", "不大于", "不能超过",
    "超限", "约束", "正确性", "合法性", "唯一性", "参数校验",
    "不能有", "资金明细", "必须", "只能",
]


# ===== 依赖表映射 =====

TABLE_NAME_MAPPING = [
    ("settle_order", "bytepay_settle_order"),
    ("settle_voucher", "bytepay_settle_voucher"),
    ("settle_entry", "bytepay_settle_entry"),
    ("settle_request", "settle_request"),
    ("settle_center", "settle_center_order"),
    ("charge_order", "bytepay_charge_order"),
    ("charge_detail_entry", "bytepay_charge_detail_entry"),
    ("charge_sharding_voucher", "bytepay_charge_sharding_voucher"),
    ("charge_voucher", "bytepay_charge_sharding_voucher"),
    ("settle_rule", "bytepay_settle_rule"),
    ("settle_code", "bytepay_settle_code"),
    ("settle_template", "bytepay_settle_template"),
    ("merchant_settle_info", "merchant_settle_info"),
    ("factor_match", "factor_match"),
    ("fee_activity_rule", "fee_activity_rule"),
    ("fee_activity_config", "fee_activity_config"),
    ("template_aggregation", "template_aggregation"),
    ("charge_rule_info", "charge_rule_info"),
    ("cycle_task", "cycle_task"),
    ("cycle_result", "cycle_result"),
    ("fund_clause", "fund_clause"),
    ("pay_order", "pay_order"),
    ("settle_sharding_set", "bytepay_settle_sharding_set"),
]


# ===== 分类和提取函数 =====

def classify_risk(name: Optional[str]) -> Optional[str]:
    """根据任务名称关键字分类风险点类别。
    优先级：配置类 > 时效性 > 一致性 > 业务正确性
    """
    if not name:
        return None
    text = name.lower()

    def contains_any(keywords: List[str]) -> bool:
        return any(kw.lower() in text for kw in keywords)

    if contains_any(CONFIG_RISK_KEYWORDS):
        return "配置类风险"
    if contains_any(TIMELINESS_RISK_KEYWORDS):
        return "时效性风险"
    if contains_any(CONSISTENCY_RISK_KEYWORDS):
        return "一致性风险"
    if contains_any(CORRECTNESS_RISK_KEYWORDS):
        return "业务正确性风险"
    return None


def extract_dep_tables(name: Optional[str]) -> List[str]:
    """从任务名称中提取相关依赖表名称，去重保持顺序。"""
    if not name:
        return []
    text = name.lower()
    found: List[str] = []
    for kw, table_name in TABLE_NAME_MAPPING:
        if kw.lower() in text and table_name not in found:
            found.append(table_name)
    return found


def derive_date_from_code(code: Optional[str]) -> Optional[str]:
    """根据 HSQL 任务编码前 6 位 (YYMMDD) 推导日期。"""
    if not code:
        return None
    m = re.match(r"(\d{6})", str(code))
    if not m:
        return None
    prefix = m.group(1)
    yy, mm, dd = int(prefix[0:2]), int(prefix[2:4]), int(prefix[4:6])
    try:
        d = date(2000 + yy, mm, dd)
    except ValueError:
        return None
    return d.strftime("%Y-%m-%d")


# ===== 记录构建 =====

def build_dda_record(row: Dict[str, Any], fop_domain: str) -> Dict[str, Any]:
    """构造 DDA 任务记录"""
    name = row.get("name") or ""
    code = row.get("code") or ""
    status = row.get("status") or ""

    fields: Dict[str, Any] = {}
    fields["任务编码"] = code
    fields["任务名称"] = {
        "text": name,
        "link": f"https://{fop_domain}/boss/check-core/dda/config-new?OrganizationId=ORG240313190058294165643265",
    }

    risk = classify_risk(name)
    if risk:
        fields["风险点类别"] = risk

    if status in ("生效", "失效"):
        fields["运行状态"] = status

    fields["核对方式"] = "实时核对"

    dep_tables = extract_dep_tables(name)
    if dep_tables:
        fields["相关依赖表名称"] = ", ".join(dep_tables)

    if row.get("createTime"):
        fields["任务创建时间"] = row["createTime"]
    if row.get("modifyTime"):
        fields["任务修改时间"] = row["modifyTime"]
    if row.get("modifier"):
        fields["最后修改人"] = row["modifier"]

    return {"record_id": "", "fields": fields}


def build_hsql_record(row: Dict[str, Any], fop_domain: str) -> Dict[str, Any]:
    """构造 HSQL 任务记录"""
    name = row.get("name") or ""
    code = row.get("code") or ""
    status = row.get("status") or ""
    creator = row.get("creator") or ""

    fields: Dict[str, Any] = {}
    fields["任务编码"] = code
    fields["任务名称"] = {
        "text": name,
        "link": f"https://{fop_domain}/render/check-core-lowcode/task-execute-list?TaskCode={code}",
    }

    risk = classify_risk(name)
    if risk:
        fields["风险点类别"] = risk

    if status in ("启动", "禁用"):
        fields["运行状态"] = status

    fields["核对方式"] = "离线核对"

    dep_tables = extract_dep_tables(name)
    if dep_tables:
        fields["相关依赖表名称"] = ", ".join(dep_tables)

    date_str = derive_date_from_code(code)
    if date_str:
        fields["任务创建时间"] = date_str
        fields["任务修改时间"] = date_str

    if creator:
        fields["最后修改人"] = creator

    return {"record_id": "", "fields": fields}


# ===== 表格结构定义 =====

TABLE_FIELDS = [
    {"field_name": "任务编码", "type": 1},  # Text
    {"field_name": "任务名称", "type": 15},  # URL
    {"field_name": "风险点类别", "type": 3, "property": {  # SingleSelect
        "options": [
            {"name": "配置类风险"},
            {"name": "时效性风险"},
            {"name": "一致性风险"},
            {"name": "业务正确性风险"},
        ]
    }},
    {"field_name": "归属子域", "type": 4, "property": {"options": []}},  # MultiSelect
    {"field_name": "商户类型", "type": 4, "property": {"options": []}},  # MultiSelect
    {"field_name": "业务分类", "type": 4, "property": {"options": []}},  # MultiSelect
    {"field_name": "业务子类", "type": 4, "property": {"options": []}},  # MultiSelect
    {"field_name": "运行状态", "type": 3, "property": {  # SingleSelect
        "options": [
            {"name": "生效"},
            {"name": "失效"},
            {"name": "启动"},
            {"name": "禁用"},
        ]
    }},
    {"field_name": "核对方式", "type": 3, "property": {  # SingleSelect
        "options": [
            {"name": "实时核对"},
            {"name": "离线核对"},
        ]
    }},
    {"field_name": "相关依赖表名称", "type": 1},  # Text
    {"field_name": "任务创建时间", "type": 1},  # Text
    {"field_name": "任务修改时间", "type": 1},  # Text
    {"field_name": "最后修改人", "type": 1},  # Text
]


# ===== 主流程 =====

def main():
    parser = argparse.ArgumentParser(description="追光清结算任务 → 飞书多维表格")
    parser.add_argument("--dda_file", required=True, help="DDA 数据 JSON 文件路径")
    parser.add_argument("--hsql_file", required=True, help="HSQL 数据 JSON 文件路径")
    parser.add_argument("--app_token", help="目标多维表格 app_token（已有表格时）")
    parser.add_argument("--table_id", help="目标数据表 table_id（已有表格时）")
    parser.add_argument("--create_new", action="store_true", help="创建新的多维表格应用")
    parser.add_argument("--app_name", default="追光清结算核对任务台账", help="新建表格应用名称")
    parser.add_argument("--batch_size", type=int, default=450, help="批量写入大小")
    parser.add_argument("--fop_domain", default="fop.bytedance.net", help="FOP 平台域名")
    args = parser.parse_args()

    # 加载数据
    print(f"加载 DDA 数据: {args.dda_file}")
    with open(args.dda_file, "r", encoding="utf-8") as f:
        dda_rows = json.load(f)
    print(f"加载 HSQL 数据: {args.hsql_file}")
    with open(args.hsql_file, "r", encoding="utf-8") as f:
        hsql_rows = json.load(f)

    print(f"DDA 记录数: {len(dda_rows)}, HSQL 记录数: {len(hsql_rows)}")

    # 构建写入记录
    records: List[Dict[str, Any]] = []
    for row in dda_rows:
        records.append(build_dda_record(row, args.fop_domain))
    for row in hsql_rows:
        records.append(build_hsql_record(row, args.fop_domain))

    print(f"合计待写入记录: {len(records)}")

    # 初始化客户端
    client = LarkBitableClient()

    # 创建新表格或使用已有表格
    if args.create_new:
        print(f"\n创建新多维表格应用: {args.app_name}")
        resp = client.create_app(args.app_name)
        app_data = resp.get("data", {}).get("app", {})
        app_token = app_data.get("app_token")
        print(f"  app_token: {app_token}")
        print(f"  URL: {app_data.get('url')}")

        print(f"创建数据表...")
        table_resp = client.create_table(app_token, "核对任务台账", TABLE_FIELDS)
        table_id = table_resp.get("data", {}).get("table_id")
        print(f"  table_id: {table_id}")
    else:
        if not args.app_token or not args.table_id:
            print("错误: 请提供 --app_token 和 --table_id，或使用 --create_new 创建新表格")
            sys.exit(1)
        app_token = args.app_token
        table_id = args.table_id

    # 分批写入
    print(f"\n开始写入记录到表格 (app={app_token}, table={table_id})...")
    total = len(records)
    batch_idx = 1
    for start in range(0, total, args.batch_size):
        end = min(start + args.batch_size, total)
        batch = records[start:end]
        print(f"  批次 {batch_idx}: 记录 {start + 1}-{end}")
        try:
            resp = client.batch_add_records(app_token, table_id, batch)
            if resp.get("code") not in (None, 0):
                print(f"    ⚠ 批次写入异常: code={resp.get('code')}, msg={resp.get('msg')}")
        except Exception as e:
            print(f"    ✗ 批次写入失败: {e}")
        batch_idx += 1

    print(f"\n✓ 写入完成！共 {total} 条记录已导入飞书多维表格。")
    if args.create_new:
        print(f"  表格链接: https://bytedance.larkoffice.com/base/{app_token}")


if __name__ == "__main__":
    main()
