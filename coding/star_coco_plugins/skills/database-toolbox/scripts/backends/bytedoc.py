"""ByteDoc 后端处理函数 — 统一接管 db_type=ByteDoc

按 instance_type 内部分流：
- Mongo（Cloud Native）→ 委托 DBW handler 函数（纯函数调用，不走 dispatch）
- Classic（ByteDoc）  → 调用 ByteDoc Classic 原生 API
"""

import re
from datetime import datetime, timezone

from registry import register
from common import _ok, _error, _truncate_list, _vregion_to_region_id
from backends.dbw import (
    _DBW_BYTEDOC_VREGIONS,
    dbw_list_tables,
    dbw_get_table_info,
    dbw_execute_sql,
    dbw_nl2sql,
    dbw_describe_instance_nodes,
    dbw_list_active_sessions,
)


def _is_classic(params: dict) -> bool:
    """判断是否为 Classic ByteDoc（非 Mongo）"""
    return params.get("instance_type") != "Mongo"


def _get_bytedoc_instance_meta(client, instance_id: str, vregion: str) -> dict:
    """从 describe_instances 获取 ByteDoc 实例元信息。
    返回 {"psm_db": str, "region_id": str, "instance_type": str} 或抛 ValueError。
    """
    ds_type = "ByteDoc"
    region = _vregion_to_region_id(vregion)
    result = client.describe_instances({
        "InstanceId": instance_id,
        "DSType": ds_type,
        "PageNumber": 1,
        "PageSize": 10,
        "RegionId": region,
    })
    instances = result.get("Instances", [])
    matched = None
    for inst in instances:
        if inst.get("InstanceId") == instance_id:
            matched = inst
            break
    if not matched and len(instances) == 1:
        matched = instances[0]
    if not matched:
        raise ValueError(f"未找到 ByteDoc 实例 {instance_id}")
    psm_db = matched.get("PsmDb", "")
    region_id = matched.get("RegionId", "")
    instance_type = matched.get("InstanceType", "")
    if not psm_db:
        raise ValueError(f"ByteDoc 实例 {instance_id} 缺少 PsmDb 字段")
    return {"psm_db": psm_db, "region_id": region_id, "instance_type": instance_type}


def _parse_mongo_command(sql: str):
    """解析 MongoDB shell 命令格式 db.collection.operation(...)。
    返回 (collection, query_body) 或 None。
    """
    m = re.match(r'^db\.(\w+)\.(\w+)\((.+)\)\s*;?\s*$', sql.strip(), re.DOTALL)
    if m:
        return m.group(1), m.group(2) + "(" + m.group(3) + ")"
    return None


# ──────────────────────────────────────────────
# 新增 6 个 ByteDoc handler（Mongo 委托 DBW，Classic 调原生 API）
# ──────────────────────────────────────────────

@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "list_tables")
def _bytedoc_list_tables(client, params, ctx, **kwargs):
    """ByteDoc list_tables：Mongo 委托 DBW，Classic 用原生 get_collections"""
    if not _is_classic(params):
        return dbw_list_tables(client, params, ctx, **kwargs)
    try:
        database = params.get("database")
        if not database:
            return _error("缺少 database 参数", context=ctx)
        raw = client.bytedoc.classic_get_collections(database)
        data = raw.get("data") or raw.get("result") or []
        if isinstance(data, dict):
            data = data.get("collections") or data.get("data") or []
        tables = []
        for item in data:
            name = item.get("name", "") if isinstance(item, dict) else str(item)
            if name:
                tables.append(name)
        return _ok({
            "total": len(tables),
            "tables": tables,
        }, f"共 {len(tables)} 个 collection", context=ctx)
    except Exception as e:
        return _error(f"list_tables失败: {e}", context=ctx)


@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "get_table_info")
def _bytedoc_get_table_info(client, params, ctx, **kwargs):
    """ByteDoc get_table_info：Mongo 委托 DBW，Classic 用原生 get_collection_indexes"""
    if not _is_classic(params):
        return dbw_get_table_info(client, params, ctx, **kwargs)
    try:
        database = params.get("database")
        table = kwargs.get("table", "")
        if not database:
            return _error("缺少 database 参数", context=ctx)
        if not table:
            return _error("缺少 table（collection）参数", context=ctx)
        raw = client.bytedoc.classic_get_collection_indexes(database, table)
        data = raw.get("data") or raw.get("result") or []
        indexes = []
        if isinstance(data, list):
            indexes = data
        elif isinstance(data, dict):
            indexes = data.get("indexes") or data.get("data") or []
        return _ok({
            "collection": table,
            "indexes": indexes,
        }, f"collection {table} 的索引信息", context=ctx)
    except Exception as e:
        return _error(f"get_table_info失败: {e}", context=ctx)


@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "execute_sql")
def _bytedoc_execute_sql(client, params, ctx, **kwargs):
    """ByteDoc execute_sql：Mongo 委托 DBW，Classic 用原生 API"""
    if not _is_classic(params):
        return dbw_execute_sql(client, params, ctx, **kwargs)
    try:
        database = params.get("database")
        sql = kwargs.get("sql", "").strip()
        if not database:
            return _error("缺少 database 参数", context=ctx)
        if not sql:
            return _error("缺少 sql 参数", context=ctx)

        # show collections → 转用 get_collections API
        if sql.lower() in ("show collections", "show tables"):
            raw = client.bytedoc.classic_get_collections(database)
            data = raw.get("data") or []
            tables = [str(item) for item in data if item] if isinstance(data, list) else []
            return _ok({
                "columns": ["collection"],
                "rows": [{"Cells": [t]} for t in tables],
                "row_count": len(tables),
            }, f"共 {len(tables)} 个 collection", context=ctx)

        # db.collection.operation(...) → web_query API
        parsed = _parse_mongo_command(sql)
        if not parsed:
            return _error(
                "Classic ByteDoc 需要 MongoDB 查询格式，"
                "如: db.myCollection.find({\"key\": \"value\"}) 或 show collections",
                context=ctx,
            )
        collection, query = parsed
        raw = client.bytedoc.classic_web_query(database, collection, query)
        data = raw.get("data") or raw.get("result")
        return _ok(data, "查询成功", context=ctx)
    except Exception as e:
        return _error(f"execute_sql失败: {e}", context=ctx)


@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "nl2sql")
def _bytedoc_nl2sql(client, params, ctx, **kwargs):
    """ByteDoc nl2sql：Mongo 委托 DBW，Classic 不支持"""
    if not _is_classic(params):
        return dbw_nl2sql(client, params, ctx, **kwargs)
    return _error("Classic ByteDoc 暂不支持 nl2sql", context=ctx)


@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "describe_instance_nodes")
def _bytedoc_describe_instance_nodes(client, params, ctx, **kwargs):
    """ByteDoc describe_instance_nodes：Mongo 委托 DBW，Classic 用 static_info"""
    if not _is_classic(params):
        return dbw_describe_instance_nodes(client, params, ctx, **kwargs)
    try:
        database = params.get("database")
        if not database:
            return _error("缺少 database 参数", context=ctx)
        raw = client.bytedoc.classic_static_info(database)
        data = raw.get("data") or raw.get("result") or {}
        return _ok(data, "获取 Classic ByteDoc 静态信息成功", context=ctx)
    except Exception as e:
        return _error(f"describe_instance_nodes失败: {e}", context=ctx)


@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "list_active_sessions")
def _bytedoc_list_active_sessions(client, params, ctx, **kwargs):
    """ByteDoc list_active_sessions：统一委托 DBW"""
    return dbw_list_active_sessions(client, params, ctx, **kwargs)


# ──────────────────────────────────────────────
# 已有 handler（更新：加 Classic 分支）
# ──────────────────────────────────────────────

@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "describe_slow_logs")
def _bytedoc_describe_slow_logs(client, params, ctx, **kwargs):
    """ByteDoc 慢日志：Mongo 用 Cloud Native API，Classic 用 slow_query_overview"""
    try:
        needs = ["instance_id", "vregion"]
        missing = [n for n in needs if not params.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        if not start_ts or not end_ts:
            return _error("缺少 start_time / end_time", context=ctx)

        meta = _get_bytedoc_instance_meta(client, params["instance_id"], params["vregion"])

        if meta["instance_type"] == "Mongo":
            # Cloud Native Mongo: 用原有 ByteDoc Cloud API
            page_size = kwargs.get("page_size", 10)
            sort_by = kwargs.get("sort_by", "DESC")
            start_iso = datetime.fromtimestamp(start_ts, tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
            end_iso = datetime.fromtimestamp(end_ts, tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
            pod_name = f"{params['instance_id']}-0"

            raw = client.bytedoc.describe_slow_logs(
                instance_id=params["instance_id"],
                pod_name=pod_name,
                region=meta["region_id"],
                psm=meta["psm_db"],
                start_time=start_iso,
                end_time=end_iso,
                limit=page_size,
                sort=sort_by,
            )
            data = raw.get("data", {})
            slow_logs = data.get("Datas") or []
            logs = []
            for log in slow_logs:
                logs.append({
                    "sql": log.get("SQL", ""),
                    "db": log.get("DBName", ""),
                    "user": log.get("UserName", ""),
                    "time": log.get("StartTime", ""),
                    "query_time": log.get("Duration", 0),
                    "ip": log.get("ClientIP", ""),
                    "rows_scanned": log.get("FileScan", 0),
                    "index_scan": log.get("IndexScan", 0),
                    "rows_sent": log.get("Return", 0),
                })
            return _ok({
                "total": data.get("Total", 0),
                "logs": logs,
            }, f"共 {len(logs)} 条慢查询", context=ctx)
        else:
            # Classic ByteDoc: 用 slow_query_overview
            database = params.get("database")
            if not database:
                return _error("缺少 database 参数", context=ctx)
            millis = kwargs.get("millis", 100)
            raw = client.bytedoc.classic_slow_query_overview(
                database, int(start_ts), int(end_ts), millis=millis,
            )
            data = raw.get("data") or raw.get("result") or []
            logs = []
            if isinstance(data, list):
                for item in data:
                    logs.append({
                        "sql": item.get("query_pattern") or item.get("sql", ""),
                        "db": item.get("db", database),
                        "collection": item.get("collection", ""),
                        "count": item.get("count", 0),
                        "avg_time_ms": item.get("avg_millis") or item.get("avg_time_ms", 0),
                        "max_time_ms": item.get("max_millis") or item.get("max_time_ms", 0),
                    })
            total = len(logs)
            tr = _truncate_list({"logs": logs}, limit=100, label="slow_logs")
            msg = f"共 {total} 条慢查询聚合"
            if tr["truncated"]:
                msg += f"，当前返回前 {tr['returned_count']} 条。完整结果已写入 {tr['artifact_path']}"
            return _ok({
                "total": total,
                "returned_count": tr["returned_count"],
                "truncated": tr["truncated"],
                "artifact_path": tr["artifact_path"],
                "logs": tr["logs"],
            }, msg, context=ctx)

    except Exception as e:
        return _error(f"describe_slow_logs失败: {e}", context=ctx)


@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "describe_table_space")
def _bytedoc_describe_table_space(client, params, ctx, **kwargs):
    """ByteDoc collection 磁盘详情：Mongo 用 Cloud Native API，Classic 用 capacity_manager"""
    try:
        needs = ["instance_id", "vregion"]
        missing = [n for n in needs if not params.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        meta = _get_bytedoc_instance_meta(client, params["instance_id"], params["vregion"])

        if meta["instance_type"] == "Mongo":
            # Cloud Native Mongo: 用原有 InstanceCollectionInfo API
            database = params.get("database") or params["instance_id"]
            raw = client.bytedoc.instance_collection_info(
                instance_id=params["instance_id"],
                database=database,
                psm=meta["psm_db"],
                region=meta["region_id"],
            )
            collections = raw.get("data") or []
            tables = []
            for c in collections:
                tables.append({
                    "collection": c.get("collection_name", ""),
                    "size": c.get("collection_size", "0 B"),
                    "documents": c.get("total_document", 0),
                    "avg_doc_size": c.get("average_document_size", "0 B"),
                    "index_count": c.get("index_count", 0),
                    "last_day_incr_size_gb": c.get("last_day_incr_size_gb", "0"),
                    "last_day_incr_count": c.get("last_day_incr_count", 0),
                    "large_document_count": c.get("large_document_count", 0),
                })
            return _ok({
                "total": len(tables),
                "tables": tables,
            }, f"共 {len(tables)} 个 collection", context=ctx)
        else:
            # Classic ByteDoc: 先 usage_info 拿 cluster_id，再 classic_capacity 拿 collection 级详情
            database = params.get("database")
            if not database:
                return _error("缺少 database 参数", context=ctx)
            usage = client.bytedoc.classic_usage_info(database)
            usage_data = usage.get("data") or {}
            cluster_id = usage_data.get("cluster_name", "")
            if not cluster_id:
                return _error("无法获取 cluster_id（usage_info 缺少 cluster_name）", context=ctx)
            cap = client.bytedoc.classic_capacity(cluster_id, database)
            cap_data = cap.get("data") or []
            collections = []
            if isinstance(cap_data, list):
                for item in cap_data:
                    collections.append({
                        "collection": item.get("collection_name", ""),
                        "size": item.get("collection_size", "0 B"),
                        "documents": item.get("total_document", 0),
                        "avg_doc_size": item.get("average_document_size", "0 B"),
                        "index_count": item.get("index_count", 0),
                        "last_day_incr_size_gb": item.get("last_day_incr_size_gb", "0"),
                        "last_day_incr_count": item.get("last_day_incr_count", 0),
                        "large_document_count": item.get("large_document_count", 0),
                    })
            return _ok({
                "total": len(collections),
                "tables": collections,
            }, f"共 {len(collections)} 个 collection", context=ctx)

    except Exception as e:
        return _error(f"describe_table_space失败: {e}", context=ctx)


@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "get_ticket_url")
def _bytedoc_get_ticket_url(client, params, ctx, **kwargs):
    """ByteDoc 工单链接：根据 InstanceType 区分 Mongo / ByteDoc(classic)"""
    host = kwargs["host"]
    database = params.get("database")
    vregion = params.get("vregion") or client.vregion
    try:
        meta = _get_bytedoc_instance_meta(client, database, vregion)
    except Exception as e:
        return _error(f"获取 ByteDoc 实例信息失败: {e}", context=ctx)

    if meta["instance_type"] == "Mongo":
        url = f"{host}/bytedoc/cloud_native/volcDetail/{meta['psm_db']}/{meta['region_id']}"
        guide = "请在浏览器中打开链接，可查看实例详情、修改参数配置或登录数据库工作台"
        ticket_types = []
    else:
        url = f"{host}/bytedoc/classic/database/{database}/overview"
        guide = "请在浏览器中打开链接，可创建人工工单或进行集群规格升级"
        ticket_types = ["人工工单", "集群规格升级"]

    return _ok({
        "url": url,
        "database": database,
        "region": meta["region_id"],
        "guide": guide,
        "ticket_types": ticket_types,
    }, "获取工单链接成功", context=ctx)
