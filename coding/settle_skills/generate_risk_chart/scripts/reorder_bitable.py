#!/usr/bin/env python3
"""
飞书多维表格记录排序脚本

将"禁用"和"失效"状态的任务排列到表格末尾。
由于飞书 API 不支持直接设置视图排序规则，本脚本通过：
1. 获取所有记录
2. 按运行状态排序（生效/启动优先，禁用/失效靠后）
3. 删除所有原记录
4. 按排序后的顺序重新插入

使用方式：
    # 试运行（仅打印统计，不做修改）
    python3 scripts/reorder_bitable.py --app_token <token> --table_id <id>

    # 实际执行
    python3 scripts/reorder_bitable.py --app_token <token> --table_id <id> --execute

⚠ 注意：执行此操作会改变所有记录的 record_id，请确保无外部引用依赖。
"""

import os
import sys
import json
import time
import argparse
from typing import Any, Dict, List, Optional

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from lark_bitable_client import LarkBitableClient


# ===== 字段值转换 =====

READ_ONLY_UI_TYPES = {"Formula", "Lookup", "AutoNumber", "CreatedTime", "ModifiedTime", "CreatedUser", "ModifiedUser"}
READ_ONLY_TYPES = {19, 20, 1001, 1002, 1003, 1004, 1005}


def is_read_only_field(meta: Dict[str, Any]) -> bool:
    return meta.get("type") in READ_ONLY_TYPES or meta.get("ui_type") in READ_ONLY_UI_TYPES


def convert_value_for_write(value: Any, meta: Dict[str, Any]) -> Any:
    """将字段值从读取格式转换为写入格式"""
    if value is None:
        return None

    ui_type = meta.get("ui_type")

    if ui_type in {"Text", "Barcode", "Email"}:
        if isinstance(value, list):
            return "".join(str(seg.get("text", "")) if isinstance(seg, dict) else str(seg) for seg in value)
        return str(value) if value else ""

    if ui_type == "Url":
        return value

    if ui_type in {"Number", "Progress", "Currency", "Rating", "SingleSelect", "MultiSelect", "DateTime", "Checkbox"}:
        return value

    if ui_type == "User":
        if isinstance(value, list):
            return [{"id": u["id"]} for u in value if isinstance(u, dict) and "id" in u]
        return None

    if ui_type == "Attachment":
        if isinstance(value, list):
            return [{"file_token": f["file_token"]} for f in value if isinstance(f, dict) and "file_token" in f]
        return None

    return value


# ===== 主流程 =====

def main():
    parser = argparse.ArgumentParser(description="重排多维表格记录（禁用/失效置底）")
    parser.add_argument("--app_token", required=True)
    parser.add_argument("--table_id", required=True)
    parser.add_argument("--execute", action="store_true", help="实际执行删除和重插入")
    args = parser.parse_args()

    client = LarkBitableClient()
    app_token = args.app_token
    table_id = args.table_id

    # 1. 获取字段元数据
    print("获取字段元数据...")
    fields_resp = client.list_fields(app_token, table_id)
    fields = fields_resp.get("data", {}).get("items", []) or []
    meta_by_name = {f["field_name"]: f for f in fields if "field_name" in f}
    print(f"  字段数: {len(fields)}")

    # 2. 获取全部记录
    print("获取全部记录...")
    all_records: List[Dict[str, Any]] = []
    page_token = ""
    while True:
        resp = client.search_records(app_token, table_id, page_token=page_token)
        data = resp.get("data", {}) or {}
        items = data.get("items", []) or []
        all_records.extend(items)
        if not data.get("has_more"):
            break
        page_token = data.get("page_token", "")

    total = len(all_records)
    print(f"  记录总数: {total}")

    # 3. 分组排序
    active_statuses = {"生效", "启动"}
    inactive_statuses = {"失效", "禁用"}

    active, inactive, others = [], [], []
    for item in all_records:
        status = (item.get("fields") or {}).get("运行状态")
        if status in active_statuses:
            active.append(item)
        elif status in inactive_statuses:
            inactive.append(item)
        else:
            others.append(item)

    print(f"  活跃状态 (生效/启动): {len(active)}")
    print(f"  停用状态 (失效/禁用): {len(inactive)}")
    print(f"  其他: {len(others)}")

    sorted_items = active + others + inactive
    print(f"  排序后总数: {len(sorted_items)}")

    # 4. 构造写入负载
    new_records: List[Dict[str, Any]] = []
    for item in sorted_items:
        fields_read = item.get("fields", {}) or {}
        fields_write: Dict[str, Any] = {}
        for name, value in fields_read.items():
            meta = meta_by_name.get(name)
            if not meta or is_read_only_field(meta):
                continue
            fields_write[name] = convert_value_for_write(value, meta)
        new_records.append({"record_id": "", "fields": fields_write})

    if not args.execute:
        print("\n[试运行模式] 未加 --execute 参数，不做实际修改。")
        return

    # 5. 删除原记录
    record_ids = [r["record_id"] for r in all_records if r.get("record_id")]
    print(f"\n删除 {len(record_ids)} 条原记录...")
    for i in range(0, len(record_ids), 100):
        batch = record_ids[i: i + 100]
        try:
            client.batch_delete_records(app_token, table_id, batch)
        except Exception as e:
            print(f"  ✗ 删除批次 {i}-{i + len(batch)} 失败: {e}")
    print("  删除完成")

    # 6. 重新插入
    print(f"重新插入 {len(new_records)} 条记录（按排序后顺序）...")
    for i in range(0, len(new_records), 450):
        batch = new_records[i: i + 450]
        try:
            client.batch_add_records(app_token, table_id, batch)
        except Exception as e:
            print(f"  ✗ 插入批次 {i}-{i + len(batch)} 失败: {e}")
    print("  插入完成")

    print(f"\n✓ 排序完成！活跃任务在前，禁用/失效任务在后。")


if __name__ == "__main__":
    main()
