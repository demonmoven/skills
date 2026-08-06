"""DBW 后端处理函数 - 通过 DBW API 处理 ByteRDS/ByteDoc 请求（国内 vregion）"""

import sys
from typing import Any, List, Optional

from registry import register
from common import (
    _ok, _error, _to_result, _pop_request_id, _build_context,
    _truncate_list,
    _ts_to_iso, _detect_tz, _parse_iso_to_timestamp,
    _vregion_to_site, _vregion_to_region_id,
    _instance_type_to_ds_type,
    _load_region_config, _save_region_config,
    _MYSQL_TYPES, _CN_TZ, _SITE_HOST, _status_text,
    _DBW_SUPPORT_MONGO_FUNCS,
)


# DBW 支持的 ByteRDS vregion（国内 + BOE）
# 注意：新增国内 vregion 时需在此处手动添加，否则 dispatch 找不到 handler。
# i18n vregion 由 byterds.py 的 _BYTERDS_VREGIONS 覆盖，两边互不重叠。
# ChinaSinf-North（华北2）已迁移至 ByteRDS 直连后端（byterds.py）。
_DBW_VREGIONS = {
    "China-North", "China-North5",
    "China-East", "China-Fintech", "China-BOE",
}
# cn-only 子集（不含 China-BOE），用于仅限 cn 的函数注册
_DBW_CN_VREGIONS = _DBW_VREGIONS - {"China-BOE"}
# 全量 SQL 子集（不含 China-East），China-East 不支持全量 SQL 相关功能
_DBW_FULLSQL_VREGIONS = _DBW_VREGIONS - {"China-East"}
# ByteDoc 仍走 DBW，需保留 ChinaSinf-North
_DBW_BYTEDOC_VREGIONS = _DBW_VREGIONS | {"ChinaSinf-North"}


def _region(p: dict) -> str:
    """从 params 中的 vregion 推导 RegionId。"""
    return _vregion_to_region_id(p.get("vregion", ""))


# ──────────────────────────────────────────────
# 内部辅助
# ──────────────────────────────────────────────

def _check_full_sql_enabled(client, instance_id: str, instance_type: str, vregion: str) -> bool:
    """内部：检查实例是否开启了全量 SQL 分析。"""
    try:
        result = client.describe_full_sql_status({
            "FollowInstanceID": instance_id + "." + vregion,
            "DSType": instance_type,
            "VRegion": vregion,
            "RegionId": _vregion_to_region_id(vregion),
        })
        _pop_request_id(result)
        if isinstance(result, dict):
            return (result.get("SqlAnalysisFunStatus") == "RUN"
                    and result.get("DetailStatus") == "RUN")
    except Exception as e:
        print(f"[warn] check_full_sql_enabled failed: {e}", file=sys.stderr)
    return False


def _get_primary_node_id(
    client, instance_id: str, instance_type: str, vregion: str,
) -> Optional[str]:
    """内部辅助：获取主节点 NodeId。失败返回 None。"""
    try:
        result = client.describe_instance_nodes({
            "DSType": instance_type,
            "InstanceId": instance_id,
            "VRegion": vregion,
            "RegionId": _vregion_to_region_id(vregion),
        })
        if isinstance(result, dict):
            nodes = result.get("NodesInfo") or []
            for n in nodes:
                if n.get("NodeType") == "Primary":
                    return n.get("NodeId", "")
            # 没找到 Primary，取第一个
            if nodes:
                return nodes[0].get("NodeId", "")
    except Exception as e:
        print(f"[warn] get_primary_node_id failed: {e}", file=sys.stderr)
    return None


def _check_dbw_support(func_name: str, instance_type: str, ctx) -> Optional[dict]:
    """检查 DBW 后端是否支持该 instance_type + func_name 组合。
    不支持返回 error dict，支持返回 None。"""
    # MySQL 类型全部支持
    if instance_type in _MYSQL_TYPES:
        return None
    # ByteDoc/Mongo 白名单检查
    if instance_type in ("ByteDoc", "Mongo"):
        if func_name not in _DBW_SUPPORT_MONGO_FUNCS:
            return _error(f"ByteDoc 不支持 {func_name}", context=ctx)
        return None
    # 其他类型不走 DBW
    return _error(f"{instance_type} 不支持 {func_name}", context=ctx)


# ──────────────────────────────────────────────
# 实例解析（resolve_instance）
# ──────────────────────────────────────────────

@register(("ByteRDS", _DBW_VREGIONS), "resolve_instance")
@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "resolve_instance")
def dbw_resolve_instance(client, params, ctx, **kwargs):
    """通过 DBW DescribeInstances 验证实例存在 + 模糊匹配。"""
    database = kwargs.get("database", "") or params.get("database", "")
    instance_type = params.get("instance_type", "ByteRDS")
    vregion = params.get("vregion", "")
    region = _vregion_to_region_id(vregion)

    try:
        ds_type = _instance_type_to_ds_type(instance_type)
        result = client.describe_instances({
            "InstanceId": database,
            "DSType": ds_type,
            "PageNumber": 1,
            "PageSize": 10,
            "RegionId": region,
        })
        instances = result.get("Instances", [])
    except ValueError as e:
        return _error(str(e))

    if not instances:
        return _error(
            f"数据库 '{database}' 在 {vregion} 中不存在（类型: {instance_type}）。直接将此结果告知用户，请用户确认名称、地域和类型是否正确，不要做任何额外搜索或重试"
        )

    # 精确匹配：先按 InstanceName，再按 InstanceId
    exact = [i for i in instances if i.get("InstanceName") == database]
    if not exact:
        exact = [i for i in instances if i.get("InstanceId") == database]

    # 同名实例可能在多个机房（VDC），用 vdc 消歧
    if len(exact) > 1:
        target_vdc = kwargs.get("vdc", "")
        if target_vdc:
            exact = [i for i in exact if i.get("RegionId") == target_vdc]
        if len(exact) != 1:
            all_exact = [i for i in instances if i.get("InstanceName") == database]
            candidates = [
                {"id": i.get("InstanceId", ""), "name": i.get("InstanceName", ""),
                 "type": i.get("InstanceType", ""), "status": i.get("InstanceStatus", ""),
                 "vdc": i.get("RegionId", "")}
                for i in (exact if exact else all_exact)
            ]
            return _error(
                f"'{database}' 在多个机房中存在，请指定机房（vdc）",
                {"fuzzy_matches": candidates}
            )

    if exact:
        matched = exact[0]
    elif len(instances) == 1:
        matched = instances[0]
    else:
        candidates = [
            {"id": i.get("InstanceId", ""), "name": i.get("InstanceName", ""),
             "type": i.get("InstanceType", ""), "status": i.get("InstanceStatus", ""),
             "vdc": i.get("RegionId", "")}
            for i in instances
        ]
        return _error(
            f"'{database}' 匹配到 {len(candidates)} 个数据库，请指定精确名称",
            {"fuzzy_matches": candidates}
        )

    resolved_id = matched.get("InstanceId", database)
    resolved_type = matched.get("InstanceType") or instance_type
    resolved_name = matched.get("InstanceName") or None
    resolved_region = matched.get("RegionId", "")

    # 统一用 InstanceName 解析 database
    if resolved_type in _MYSQL_TYPES or resolved_type in ("Mongo", "ByteDoc"):
        resolved_db = resolved_name
    else:
        resolved_db = None

    # 写入缓存（用 vregion 作为文件名）
    region_cfg = _load_region_config(vregion)
    if "instances" not in region_cfg:
        region_cfg["instances"] = {}
    cache_key = resolved_name or resolved_id
    cache_entry = {"instance_type": resolved_type}
    if resolved_name and resolved_name != resolved_id:
        cache_entry["real_instance_id"] = resolved_id
    if resolved_region:
        cache_entry["vdc"] = resolved_region
    region_cfg["instances"][cache_key] = cache_entry
    if resolved_name and resolved_name != resolved_id:
        region_cfg["instances"][resolved_id] = {"instance_type": resolved_type}
    _save_region_config(vregion, region_cfg)

    return {
        "success": True,
        "instance_id": resolved_id,
        "instance_type": resolved_type,
        "database": resolved_db,
        "vdc": resolved_region,
    }


# ──────────────────────────────────────────────
# 元数据
# ──────────────────────────────────────────────

@register(("ByteRDS", _DBW_VREGIONS), "list_instances")
@register(("ByteDoc", _DBW_BYTEDOC_VREGIONS), "list_instances")
def dbw_list_instances(client, params, ctx, **kwargs):
    """DBW DescribeInstances"""
    try:
        db_type = params.get("db_type", "ByteRDS")
        database = kwargs.get("database")
        psm = kwargs.get("psm")
        instance_status = kwargs.get("instance_status")
        page_number = kwargs.get("page_number", 1)
        page_size = kwargs.get("page_size", 10)
        favor = kwargs.get("favor", False)
        owned = kwargs.get("owned", False)

        _search_keyword = database or psm
        if db_type == "ByteDoc" and not _search_keyword and not favor:
            return _error("ByteDoc 必须传 database 或 favor=True，否则无法查询实例列表", context=ctx)

        _region = _vregion_to_region_id(client.vregion)

        args = {
            "DSType": db_type,
            "PageNumber": page_number,
            "PageSize": page_size,
            "RegionId": _region,
            "QueryInstanceFilter": {
                "Favor": favor,
                "Owned": owned,
            },
        }
        if _search_keyword:
            args["InstanceId"] = _search_keyword
        if instance_status:
            args["InstanceStatus"] = instance_status

        result = client.describe_instances(args)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            instances = result.get("Instances", [])
            normalized = []
            for inst in instances:
                spec = inst.get("InstanceSpec") or {}
                normalized.append({
                    "id": inst.get("InstanceId", ""),
                    "name": inst.get("InstanceName", ""),
                    "status": inst.get("InstanceStatus", ""),
                    "type": inst.get("InstanceType", ""),
                    "version": inst.get("DBEngineVersion", ""),
                    "region": inst.get("RegionId", ""),
                    "zone": inst.get("Zone", ""),
                    "endpoint": inst.get("InternalAddress", ""),
                    "port": inst.get("Port", 3306),
                    "cpu": spec.get("CpuNum", 0),
                    "memory": spec.get("MemInGiB", 0),
                    "storage": spec.get("Storage", 0),
                    "create_time": inst.get("CreateTime", ""),
                    "access_source": inst.get("AccessSource", ""),
                    "dept": inst.get("Dept", ""),
                    "owners": inst.get("Owners", []),
                    "dbas": inst.get("Dbas", []),
                    "psm_list": inst.get("PsmList", []),
                })
            return _ok({
                "total": result.get("Total", 0),
                "instances": normalized,
            }, f"共 {len(normalized)} 个实例", request_id=rid, context=ctx)

        return _to_result(result, context=ctx)
    except Exception as e:
        return _error(f"describe_instances失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "list_tables")
def dbw_list_tables(client, params, ctx, **kwargs):
    """DBW ListTables"""
    try:
        p = params
        fetch_all = kwargs.get("fetch_all", False)
        page_number = kwargs.get("page_number", 1)
        page_size = kwargs.get("page_size", 50)

        _page = page_number or 1
        _size = page_size or 50
        all_tables: list[str] = []
        total = 0
        rid = None

        while True:
            needs = ["database", "instance_id", "instance_type"]
            missing = [n for n in needs if not p.get(n)]
            if missing:
                return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

            req = {
                "InstanceID": p["instance_id"],
                "InstanceType": p["instance_type"],
                "Database": p["database"],
                "PageNumber": _page,
                "PageSize": _size,
            }
            if _region(p):
                req["Region"] = _region(p)

            result = client.list_tables(req)
            rid = _pop_request_id(result)

            if not isinstance(result, dict):
                return _to_result(result, context=ctx)

            total = result.get("Total", 0)
            items = result.get("Items", [])
            all_tables.extend(items)

            if not fetch_all or len(all_tables) >= total or not items:
                break
            _page += 1

        return _ok({
            "total": len(all_tables),
            "instance_id": p["instance_id"],
            "tables": all_tables,
        }, f"共 {len(all_tables)} 张表", request_id=rid, context=ctx)
    except Exception as e:
        return _error(f"list_tables失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "get_table_info")
def dbw_get_table_info(client, params, ctx, **kwargs):
    """DBW GetTableInfo"""
    try:
        p = params
        table = kwargs.get("table", "")

        needs = ["database", "instance_id", "instance_type"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "InstanceID": p["instance_id"],
            "InstanceType": p["instance_type"],
            "Database": p["database"],
            "Table": table,
        }

        result = client.get_table_info(req)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            table_meta = result.get("TableMeta", result)
            columns = table_meta.get("Columns", [])
            normalized_columns = []
            for col in columns:
                normalized_columns.append({
                    "name": col.get("Name", ""),
                    "type": col.get("Type", ""),
                    "length": col.get("Length", ""),
                    "nullable": col.get("AllowBeNull", True),
                    "primary_key": col.get("IsPrimaryKey", False),
                    "auto_increment": col.get("IsAutoIncrement", False),
                    "default": col.get("DefaultValue"),
                    "comment": col.get("Comment", ""),
                })
            return _ok({
                "name": table_meta.get("Name", table),
                "engine": table_meta.get("Engine", ""),
                "charset": table_meta.get("CharacterSet", ""),
                "definition": table_meta.get("Definition", ""),
                "columns": normalized_columns,
            }, f"表 {table} 结构获取成功", request_id=rid, context=ctx)

        return _to_result(result, context=ctx)
    except Exception as e:
        return _error(f"get_table_info失败: {str(e)}")


# ──────────────────────────────────────────────
# 数据查询与分析
# ──────────────────────────────────────────────

@register(("ByteRDS", _DBW_VREGIONS), "nl2sql")
def dbw_nl2sql(client, params, ctx, **kwargs):
    """DBW GenerateSQLFromNL"""
    try:
        p = params
        query = kwargs.get("query", "")
        tables = kwargs.get("tables")

        p["tables"] = tables
        needs = ["database", "instance_id", "instance_type", "vregion"]
        if p["instance_type"] in ("Mongo", "ByteDoc"):
            needs.append("tables")
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "InstanceID": p["instance_id"],
            "InstanceType": p["instance_type"],
            "Database": p["database"],
            "Query": query,
            "Region": _region(p) or "cn",
            "RegionId": _region(p) or "cn",
        }
        if tables:
            req["Tables"] = tables

        result = client.nl2sql(req)
        rid = _pop_request_id(result)
        if isinstance(result, dict):
            sql = result.get("sql") or result.get("SQL") or ""
            if sql:
                return _ok({"query": query, "sql": sql}, "SQL生成成功", request_id=rid, context=ctx)
        return _to_result(result, "SQL生成成功", context=ctx)
    except Exception as e:
        return _error(f"nl2sql失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "execute_sql")
def dbw_execute_sql(client, params, ctx, **kwargs):
    """DBW ExecuteSQL"""
    try:
        p = params
        sql = kwargs.get("sql", "")

        needs = ["database", "instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "InstanceID": p["instance_id"],
            "InstanceType": p["instance_type"],
            "Database": p["database"],
            "Commands": sql,
            "Region": _region(p),
            "TimeOutSeconds": 60,
        }

        result = client.execute_sql(req)
        rid = _pop_request_id(result)

        if not isinstance(result, dict):
            return _to_result(result, "执行完成", context=ctx)

        # 从多种响应格式中提取第一条结果
        first = None
        for key in ("Results", "results"):
            items = result.get(key)
            if isinstance(items, list) and items:
                first = items[0]
                break
        if first is None:
            first = result

        def _get(pascal: str, snake: str, default=None):
            return first.get(pascal, first.get(snake, default))

        state = _get("State", "state", "")
        if state in ("Success", "success"):
            return _ok({
                "command_str": _get("CommandStr", "command_str", sql),
                "state": state,
                "row_count": _get("RowCount", "row_count", 0),
                "columns": _get("ColumnNames", "column_names", []),
                "rows": _get("Rows", "rows", []),
            }, "查询成功", request_id=rid, context=ctx)
        else:
            reason = _get("ReasonDetail", "reason_detail", "SQL执行失败")
            return _error(reason, {"state": state, "reason_detail": reason},
                          request_id=rid, context=ctx)
    except Exception as e:
        return _error(f"execute_sql失败: {str(e)}")


# ──────────────────────────────────────────────
# 慢查询诊断
# ──────────────────────────────────────────────

@register(("ByteRDS", _DBW_VREGIONS), "describe_slow_logs")
def dbw_describe_slow_logs(client, params, ctx, **kwargs):
    """DBW DescribeSlowLogs"""
    try:
        p = params
        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        tz = kwargs.get("tz", _CN_TZ)
        page_number = kwargs.get("page_number", 1)
        page_size = kwargs.get("page_size", 10)
        order_by = kwargs.get("order_by", "QueryTime")
        sort_by = kwargs.get("sort_by", "DESC")
        sql_template_id = kwargs.get("sql_template_id")

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        args = {
            "InstanceId": p["instance_id"],
            "DSType": p["instance_type"],
            "StartTime": start_ts,
            "EndTime": end_ts,
            "PageNumber": page_number,
            "PageSize": page_size,
            "OrderBy": order_by,
            "SortBy": sort_by,
            "RegionId": _region(p),
        }
        if sql_template_id:
            args["SearchParam"] = {"SQLTemplateID": sql_template_id}

        result = client.describe_slow_logs(args)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            slow_logs = result.get("SlowLogs", [])
            logs = []
            for log in slow_logs:
                logs.append({
                    "sql": log.get("SQLText", ""),
                    "template": log.get("SQLTemplate", ""),
                    "query_time": log.get("QueryTime", 0),
                    "lock_time": log.get("LockTime", 0),
                    "rows_scanned": log.get("RowsExamined", 0),
                    "rows_sent": log.get("RowsSent", 0),
                    "time": _ts_to_iso(log.get("Timestamp", 0), tz),
                    "timestamp": log.get("Timestamp", 0),
                    "user": log.get("User", ""),
                    "ip": log.get("SourceIP", ""),
                    "db": log.get("DB", ""),
                })
            return _ok({
                "total": result.get("Total", 0),
                "logs": logs,
            }, f"共 {len(logs)} 条慢查询", request_id=rid, context=ctx)

        return _to_result(result, context=ctx)
    except Exception as e:
        return _error(f"describe_slow_logs失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "describe_aggregate_slow_logs")
def dbw_describe_aggregate_slow_logs(client, params, ctx, **kwargs):
    """DBW DescribeAggregateSlowLogs"""
    try:
        p = params
        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        tz = kwargs.get("tz", _CN_TZ)
        page_number = kwargs.get("page_number", 1)
        page_size = kwargs.get("page_size", 10)
        order_by = kwargs.get("order_by", "TotalQueryTime")
        sort_by = kwargs.get("sort_by", "DESC")
        search_param = kwargs.get("search_param", {})

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        args = {
            "InstanceId": p["instance_id"],
            "DSType": p["instance_type"],
            "StartTime": start_ts,
            "EndTime": end_ts,
            "PageNumber": page_number,
            "PageSize": page_size,
            "OrderBy": order_by,
            "SortBy": sort_by,
            "RegionId": _region(p),
            "SearchParam": search_param,
        }

        result = client.describe_aggregate_slow_logs(args)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            slow_logs = result.get("AggregateSlowLogs", [])
            logs = []
            for log in slow_logs:
                logs.append({
                    "sql_template_id": log.get("SQLTemplateID", ""),
                    "sql_template": log.get("SQLTemplate", ""),
                    "db": log.get("DB", ""),
                    "user": log.get("User", ""),
                    "source_ip": log.get("SourceIP", ""),
                    "execute_count": log.get("ExecuteCount", 0),
                    "execute_count_ratio": log.get("ExecuteCountRatio", 0),
                    "query_time_ratio": log.get("QueryTimeRatio", 0),
                    "lock_time_ratio": log.get("LockTimeRatio", 0),
                    "rows_sent_ratio": log.get("RowsSentRatio", 0),
                    "rows_examined_ratio": log.get("RowsExaminedRatio", 0),
                    "query_time_stats": log.get("QueryTimeStats", {}),
                    "lock_time_stats": log.get("LockTimeStats", {}),
                    "rows_sent_stats": log.get("RowsSentStats", {}),
                    "rows_examined_stats": log.get("RowsExaminedStats", {}),
                    "first_appear_time": _ts_to_iso(log.get("FirstAppearTime", 0), tz),
                    "first_appear_timestamp": log.get("FirstAppearTime", 0),
                    "last_appear_time": _ts_to_iso(log.get("LastAppearTime", 0), tz),
                    "last_appear_timestamp": log.get("LastAppearTime", 0),
                    "sql_fingerprint": log.get("SqlFingerprint", ""),
                    "sql_method": log.get("SqlMethod", ""),
                    "table": log.get("Table", ""),
                })
            return _ok({
                "total": result.get("Total", 0),
                "logs": logs,
            }, f"共 {len(logs)} 条聚合慢查询", request_id=rid, context=ctx)

        return _to_result(result, context=ctx)
    except Exception as e:
        return _error(f"describe_aggregate_slow_logs失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "slow_query_trend")
def dbw_slow_query_trend(client, params, ctx, **kwargs):
    """DBW DescribeSlowLogTimeSeriesStats"""
    try:
        p = params
        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        interval = kwargs.get("interval", 300)
        search_param = kwargs.get("search_param", {})

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        args = {
            "InstanceId": p["instance_id"],
            "DSType": p["instance_type"],
            "StartTime": start_ts,
            "EndTime": end_ts,
            "Interval": interval,
            "RegionId": _region(p),
        }
        if search_param:
            args["SearchParam"] = search_param

        result = client.describe_slow_log_time_series_stats(args)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            count_stats = result.get("SlowLogCountStats", [])
            cpu_stats = result.get("CpuUsageStats", [])
            tr = _truncate_list({
                "slow_log_count_stats": count_stats,
                "cpu_usage_stats": cpu_stats,
            }, limit=200, label="slow_trend")
            total = len(count_stats)
            msg = f"获取 {total} 个时间点的慢查询统计"
            if tr["truncated"]:
                msg += f"，当前返回前 {tr['returned_count']} 个。完整结果已写入 {tr['artifact_path']}"
            return _ok({
                "total": total,
                "returned_count": tr["returned_count"],
                "truncated": tr["truncated"],
                "artifact_path": tr["artifact_path"],
                "interval": result.get("Interval", interval),
                "slow_log_count_stats": tr["slow_log_count_stats"],
                "cpu_usage_stats": tr["cpu_usage_stats"],
            }, msg, request_id=rid, context=ctx)

        return _to_result(result, context=ctx)
    except Exception as e:
        return _error(f"slow_query_trend失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "list_slow_query_advice")
def dbw_list_slow_query_advice(client, params, ctx, **kwargs):
    """DBW ListSlowQueryAdvice"""
    try:
        p = params
        summary_id = kwargs.get("summary_id", "")
        advice_type = kwargs.get("advice_type", "")
        group_by = kwargs.get("group_by", "")
        order_by = kwargs.get("order_by", "Benefit")
        page_number = kwargs.get("page_number", 1)
        page_size = kwargs.get("page_size", 10)

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        args = {
            "InstanceId": p["instance_id"],
            "InstanceType": p["instance_type"],
            "RegionId": _region(p),
            "SummaryId": summary_id,
            "AdviceType": advice_type,
            "GroupBy": group_by,
            "OrderBy": order_by,
            "PageNumber": page_number,
            "PageSize": page_size,
        }

        result = client.list_slow_query_advice(args)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            advices = result.get("advices", [])
            normalized = []
            for a in advices:
                normalized.append({
                    "table": a.get("table_name", ""),
                    "sql": a.get("sql_module", ""),
                    "advice": a.get("advice", ""),
                    "level": a.get("advice_level", ""),
                    "benefit": a.get("benefit", 0),
                    "speed_up": a.get("speed_up", 0),
                    "estimated_time_after": a.get("query_time_avg_after", 0),
                })
            return _ok({
                "total": result.get("total", 0),
                "advices": normalized,
            }, f"共 {len(normalized)} 条建议", request_id=rid, context=ctx)

        return _to_result(result, context=ctx)
    except Exception as e:
        return _error(f"list_slow_query_advice失败: {str(e)}")


@register(("ByteRDS", _DBW_CN_VREGIONS), "slow_query_advice_task_history")
def dbw_slow_query_advice_task_history(client, params, ctx, **kwargs):
    """DBW SlowQueryAdviceTaskHistory"""
    try:
        p = params
        page_number = kwargs.get("page_number", 1)
        page_size = kwargs.get("page_size", 10)

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        args = {
            "InstanceId": p["instance_id"],
            "InstanceType": p["instance_type"],
            "RegionId": _region(p),
            "PageNumber": page_number,
            "PageSize": page_size,
        }

        result = client.slow_query_advice_task_history(args)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            res_list = result.get("res_list") or []
            tasks = []
            for item in res_list:
                status = item.get("status", "")
                tasks.append({
                    "id": item.get("summary_id", ""),
                    "db": item.get("db", ""),
                    "date": item.get("date", ""),
                    "status": status,
                    "status_text": "诊断中" if status == "INIT" else ("已完成" if status == "SUCCESS" else "异常"),
                    "slow_query_count": item.get("slow_query_num", 0),
                    "index_advice_count": item.get("advice_index_num", 0),
                    "rewrite_advice_count": item.get("advice_rewrite_num", 0),
                })
            return _ok({
                "total": result.get("total", 0),
                "tasks": tasks,
            }, f"共 {len(tasks)} 个诊断任务", request_id=rid, context=ctx)

        return _to_result(result, context=ctx)
    except Exception as e:
        return _error(f"slow_query_advice_task_history失败: {str(e)}")


# ──────────────────────────────────────────────
# 全量 SQL
# ──────────────────────────────────────────────

@register(("ByteRDS", _DBW_FULLSQL_VREGIONS), "describe_full_sql_detail")
def dbw_describe_full_sql_detail(client, params, ctx, **kwargs):
    """DBW DescribeFullSQLDetail"""
    try:
        p = params
        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        page_size = kwargs.get("page_size", 50)
        search_param = kwargs.get("search_param", {})
        context_cursor = kwargs.get("context")

        # 检查全量 SQL 是否开启
        if not _check_full_sql_enabled(client, p["instance_id"], p["instance_type"], p["vregion"]):
            hint = ""
            if p["vregion"] == "China-East":
                hint = "（China-East 地域当前不支持全量 SQL 功能）"
            return _error(f"该实例未开启全量 SQL 分析功能，无法查询 SQL 详情{hint}", context=ctx)

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        args = {
            "FollowInstanceID": p["instance_id"] + "." + p["vregion"],
            "InstanceType": p["instance_type"],
            "StartTime": start_ts,
            "EndTime": end_ts,
            "PageSize": page_size,
            "RegionId": _region(p),
            "SearchParam": search_param,  # 必须传，即使为空，否则 API 不返回数据
        }
        if context_cursor:
            args["Context"] = context_cursor

        result = client.describe_full_sql_detail(args)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            sql_list = result.get("DescribeFullSQLDetailRows", [])
            normalized = []
            for sql in sql_list:
                normalized.append({
                    "db_name": sql.get("DBName", ""),
                    "process_id": sql.get("ProcessID", "") or sql.get("SessionID", ""),
                    "sql_type": sql.get("SqlType", ""),
                    "query_string": sql.get("QueryString", ""),
                    "start_timestamp": sql.get("StartTimestamp", ""),
                    "end_timestamp": sql.get("EndTimestamp", ""),
                    "exec_time": sql.get("ExecTime", 0),
                    "cpu_time": sql.get("CpuTime", 0),
                    "row_lock_wait_time": sql.get("RowlockWaitTime", 0),
                    "rows_examined": sql.get("RowsExamined", 0),
                    "rows_sent": sql.get("RowsSent", 0),
                    "user_name": sql.get("UserName", ""),
                    "client_ip": sql.get("ClientIp", ""),
                    "sql_fingerprint": sql.get("SqlFingerprint", ""),
                    "sql_table": sql.get("SqlTable", ""),
                    "node_id": sql.get("NodeId", ""),
                    "sql_template": sql.get("SqlTemplate", ""),
                })
            return _ok({
                "total": result.get("Total", 0),
                "list_over": result.get("ListOver", False),
                "context": result.get("Context", ""),
                "sql_list": normalized,
            }, f"共 {len(normalized)} 条 SQL 详情", request_id=rid, context=ctx)

        return _to_result(result, context=ctx)
    except Exception as e:
        return _error(f"describe_full_sql_detail失败: {str(e)}")


@register(("ByteRDS", _DBW_FULLSQL_VREGIONS), "table_write_analysis")
def dbw_table_write_analysis(client, params, ctx, **kwargs):
    """DBW DescribeAggregationSQLTable"""
    try:
        p = params
        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        tables = kwargs.get("tables")
        order_by = kwargs.get("order_by", "ExecCount")
        sort_by = kwargs.get("sort_by", "ASC")

        # 依赖全量 SQL 功能
        if not _check_full_sql_enabled(client, p["instance_id"], p["instance_type"], p["vregion"]):
            hint = ""
            if p["vregion"] == "China-East":
                hint = "（China-East 地域当前不支持全量 SQL 功能）"
            return _error(f"该实例未开启全量 SQL 分析功能，写分析不可用{hint}", context=ctx)

        req: dict[str, Any] = {
            "InstanceId": p["instance_id"] + "." + p["vregion"],
            "InstanceType": p["instance_type"],
            "StartTime": start_ts,
            "EndTime": end_ts,
            "Tables": tables or [],
            "OrderBy": order_by,
            "SortBy": sort_by,
            "VRegion": p["vregion"],
            "RegionId": _region(p),
        }
        result = client.describe_aggregation_sql_table(req)
        rid = _pop_request_id(result)
        return _to_result(result, "写分析查询成功", context=ctx)
    except Exception as e:
        return _error(f"table_write_analysis失败: {str(e)}")


@register(("ByteRDS", _DBW_FULLSQL_VREGIONS), "describe_table_metric")
def dbw_describe_table_metric(client, params, ctx, **kwargs):
    """DBW DescribeTableMetric"""
    try:
        p = params
        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        item_type = kwargs.get("item_type", "DML")
        db_name = kwargs.get("db_name", "")
        table = kwargs.get("table", "")

        # 依赖全量 SQL 功能
        if not _check_full_sql_enabled(client, p["instance_id"], p["instance_type"], p["vregion"]):
            hint = ""
            if p["vregion"] == "China-East":
                hint = "（China-East 地域当前不支持全量 SQL 功能）"
            return _error(f"该实例未开启全量 SQL 分析功能，表监控不可用{hint}", context=ctx)

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "InstanceId": p["instance_id"] + "." + p["vregion"],
            "InstanceType": p["instance_type"],
            "ItemType": item_type,
            "DbName": db_name,
            "Table": table,
            "StartTime": start_ts,
            "EndTime": end_ts,
            "VRegion": p["vregion"],
        }
        result = client.describe_table_metric(req)
        rid = _pop_request_id(result)
        return _to_result(result, "获取表监控成功", context=ctx)
    except Exception as e:
        return _error(f"describe_table_metric失败: {str(e)}")


@register(("ByteRDS", "China-BOE"), "get_metric_data_predict")
def dbw_get_metric_data_predict(client, params, ctx, **kwargs):
    """DBW GetMetricDataPredict"""
    try:
        p = params
        metric_name = kwargs.get("metric_name", "")
        period = kwargs.get("period", 60)
        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "InstanceID": p["instance_id"],
            "InstanceType": p["instance_type"],
            "MetricName": metric_name,
            "Period": period,
            "StartTime": start_ts,
            "EndTime": end_ts,
            "Filters": [{"InstanceId": p["instance_id"]}],
            "RegionId": _region(p),
        }
        result = client.get_metric_data_predict(req)
        rid = _pop_request_id(result)
        return _to_result(result, "获取监控预测数据成功", context=ctx)
    except Exception as e:
        return _error(f"get_metric_data_predict失败: {str(e)}")


# ──────────────────────────────────────────────
# 实例节点与健康
# ──────────────────────────────────────────────

@register(("ByteRDS", _DBW_VREGIONS), "describe_instance_nodes")
def dbw_describe_instance_nodes(client, params, ctx, **kwargs):
    """DBW DescribeInstanceNodes"""
    try:
        p = params

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "DSType": p["instance_type"],
            "InstanceId": p["instance_id"],
            "VRegion": p["vregion"],
            "RegionId": _region(p),
        }
        result = client.describe_instance_nodes(req)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            nodes_raw = result.get("NodesInfo") or []
            nodes = []
            for n in nodes_raw:
                nodes.append({
                    "node_id": n.get("NodeId", ""),
                    "node_type": n.get("NodeType", ""),
                    "cpu": n.get("CpuNum", 0),
                    "memory_gib": n.get("MemInGiB", 0),
                })
            return _ok({
                "instance_id": p["instance_id"],
                "nodes": nodes,
            }, f"共 {len(nodes)} 个节点", request_id=rid, context=ctx)

        return _to_result(result, "查询节点列表成功", context=ctx)
    except Exception as e:
        return _error(f"describe_instance_nodes失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "describe_health_summary")
def dbw_describe_health_summary(client, params, ctx, **kwargs):
    """DBW DescribeHealthSummary"""
    try:
        p = params
        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        node_ids = kwargs.get("node_ids")
        diag_type = kwargs.get("diag_type", "ALL")

        # node_ids 不可为空，未传则自动获取主节点
        _node_ids = node_ids
        if not _node_ids:
            primary = _get_primary_node_id(client, p["instance_id"], p["instance_type"], p["vregion"])
            if not primary:
                return _error("无法获取实例节点信息，请手动指定 node_ids（可先调用 describe_instance_nodes 获取）", context=ctx)
            _node_ids = [primary]

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "RegionId": _region(p),
            "InstanceType": p["instance_type"],
            "InstanceId": p["instance_id"],
            "StartTime": start_ts,
            "EndTime": end_ts,
            "NodeIds": _node_ids,
            "DiagType": diag_type,
            "VRegion": p["vregion"],
        }
        result = client.describe_health_summary(req)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            stats_raw = result.get("ResourceStats") or []
            metrics = []
            for s in stats_raw:
                metrics.append({
                    "name": s.get("Name", ""),
                    "avg": s.get("Avg"),
                    "max": s.get("Max"),
                    "min": s.get("Min"),
                    "unit": s.get("Unit", ""),
                    "mom": s.get("MoM"),
                    "yoy": s.get("YoY"),
                })
            return _ok({
                "instance_id": p["instance_id"],
                "node_ids": _node_ids,
                "metrics": metrics,
            }, f"共 {len(metrics)} 项监控指标", request_id=rid, context=ctx)

        return _to_result(result, "查询健康概览成功", context=ctx)
    except Exception as e:
        return _error(f"describe_health_summary失败: {str(e)}")


# ──────────────────────────────────────────────
# 会话与锁诊断
# ──────────────────────────────────────────────

@register(("ByteRDS", _DBW_VREGIONS), "list_active_sessions")
def dbw_list_active_sessions(client, params, ctx, **kwargs):
    """DBW DescribeDialogInfos"""
    try:
        p = params
        show_sleep = kwargs.get("show_sleep", False)

        needs = ["instance_id", "instance_type"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "InstanceId": p["instance_id"],
            "DSType": p["instance_type"],
            "Component": "DBEngine",
            "PageNumber": 1,
            "PageSize": 50,
            "QueryFilter": {
                "ShowSleepConnection": show_sleep,
            },
            "VRegion": p["vregion"],
            "RegionId": _region(p),
        }
        result = client.describe_dialog_infos(req)
        rid = _pop_request_id(result)

        if isinstance(result, dict):
            details = result.get("Details") or {}
            dialog_list = details.get("DialogDetails") or []
            total = details.get("Total", len(dialog_list))
            sessions = []
            for d in dialog_list:
                sessions.append({
                    "process_id": d.get("ProcessID", ""),
                    "user": d.get("User", ""),
                    "host": d.get("Host", ""),
                    "db": d.get("DB", ""),
                    "command": d.get("Command", ""),
                    "time": d.get("Time", 0),
                    "state": d.get("State", ""),
                    "info": d.get("Info", ""),
                    "blocking_pid": d.get("BlockingPid", ""),
                })
            # 按执行时间降序排列
            sessions.sort(key=lambda s: int(s.get("time", 0)), reverse=True)
            return _ok({
                "total": total,
                "sessions": sessions,
            }, f"共 {total} 个实时会话", request_id=rid, context=ctx)

        return _to_result(result, "查询实时会话成功", context=ctx)
    except Exception as e:
        return _error(f"list_active_sessions失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "describe_deadlock")
def dbw_describe_deadlock(client, params, ctx, **kwargs):
    """DBW DescribeDeadlock"""
    try:
        p = params

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "InstanceId": p["instance_id"],
            "DSType": p["instance_type"],
            "RegionId": _region(p),
        }
        result = client.describe_deadlock(req)
        return _to_result(result, "查询死锁信息成功", context=ctx)
    except Exception as e:
        return _error(f"describe_deadlock失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "list_transactions")
def dbw_list_transactions(client, params, ctx, **kwargs):
    """DBW DescribeTrxAndLocks"""
    try:
        p = params
        page_number = kwargs.get("page_number", 1)
        page_size = kwargs.get("page_size", 10)

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "InstanceId": p["instance_id"],
            "DSType": p["instance_type"],
            "PageNumber": page_number,
            "PageSize": page_size,
            "QueryFilter": {},
            "VRegion": p["vregion"],
            "RegionId": _region(p),
        }
        result = client.describe_trx_and_locks(req)
        return _to_result(result, "查询事务和锁列表成功", context=ctx)
    except Exception as e:
        return _error(f"list_transactions失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "export_transactions")
def dbw_export_transactions(client, params, ctx, **kwargs):
    """DBW CreateTrxExportTask"""
    try:
        p = params
        task_name = kwargs.get("task_name", "")
        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "TaskName": task_name,
            "RegionId": _region(p),
            "InstanceId": p["instance_id"],
            "InstanceType": p["instance_type"],
            "StartTime": start_ts * 1000,  # 后端 int64 毫秒
            "EndTime": end_ts * 1000,
        }
        result = client.create_trx_export_task(req)
        return _to_result(result, "创建事务导出任务成功", context=ctx)
    except Exception as e:
        return _error(f"export_transactions失败: {str(e)}")


@register(("ByteRDS", _DBW_VREGIONS), "transaction_snapshots")
def dbw_transaction_snapshots(client, params, ctx, **kwargs):
    """DBW DescribeTrxSnapshots"""
    try:
        p = params
        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        page_number = kwargs.get("page_number", 1)
        page_size = kwargs.get("page_size", 10)

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "InstanceId": p["instance_id"],
            "InstanceType": p["instance_type"],
            "StartTime": start_ts,
            "EndTime": end_ts,
            "PageNumber": page_number,
            "PageSize": page_size,
            "RegionId": _region(p),
        }
        result = client.describe_trx_snapshots(req)
        return _to_result(result, "查询事务快照成功", context=ctx)
    except Exception as e:
        return _error(f"transaction_snapshots失败: {str(e)}")


# ──────────────────────────────────────────────
# 存储空间
# ──────────────────────────────────────────────

@register(("ByteRDS", _DBW_VREGIONS), "describe_table_space")
def dbw_describe_table_space(client, params, ctx, **kwargs):
    """DBW DescribeTableSpace"""
    try:
        p = params
        page_number = kwargs.get("page_number", 1)
        page_size = kwargs.get("page_size", 10)
        filter_database = kwargs.get("filter_database")
        table_name = kwargs.get("table_name")

        needs = ["instance_id", "instance_type", "vregion"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        req = {
            "InstanceId": p["instance_id"],
            "InstanceType": p["instance_type"],
            "RegionId": _region(p),
            "PageNumber": page_number,
            "PageSize": page_size,
        }
        if filter_database:
            req["Database"] = filter_database
        if table_name:
            req["TableName"] = table_name

        result = client.describe_table_space(req)
        return _to_result(result, "查询表空间详情成功", context=ctx)
    except Exception as e:
        return _error(f"describe_table_space失败: {str(e)}")




