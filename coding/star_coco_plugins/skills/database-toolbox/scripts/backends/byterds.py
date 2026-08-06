"""ByteRDS 直连后端 - 通过 RDS Platform API 直接访问（不经过 DBW）

覆盖 _BYTERDS_VREGIONS 中的所有 vregion（DBW 未覆盖的国内 + 全部 i18n）。
使用 (instance_type, vregion) 精确匹配注册，与 DBW 后端互不重叠。

设计原则：
- 后端函数通过 client.byterds 访问 ByteRDSClient，由 _ensure_vdc 切换机房
- ByteRDSClient 构造时已从 vregion 推导出正确的 host，后端通过 set_region 切换 vdc
"""

from typing import Any

from registry import register
from common import _ok, _error, _truncate_list, _load_region_config, _save_region_config, _CN_TZ
from clients.byterds_client import _VREGION_TO_REGION, _VREGION_TO_VDCS, _VDC_REGION, _VDC_API_OVERRIDE

# 走 ByteRDS 直连的 vregion（DBW 不覆盖的部分）
_BYTERDS_VREGIONS: set[str] = {
    # 国内站 cn — DBW 未覆盖的 vregion
    "ChinaSinf-North",  # 华北2：从 DBW 迁移至直连
    "China-North3", "China-North6",
    "China-Pay", "China-Pay2", "China-HKPay",
    "China-Aggregation", "China-Enterprise", "Aliyun_NC2",
    # 国内站 boe — DBW 未覆盖的 vregion
    "China-BOE2", "ChinaSinf-BOE", "China-InfBOE",
    # i18n-bd
    "Asia-SouthEastBD", "Europe-WestBD", "Singapore-SaaS", "Asia-SaaS",
    "US-EE", "Singapore-Common", "US-EastBD", "US-TTP3", "Australia-SouthEastBD",
    # i18n-tt
    "Singapore-Central", "EasternEuro-TT", "I18N-Game", "Europe-Central",
    "US-East", "US-West", "Australia-SouthEast",
}


def _ensure_vdc(client, params):
    """从传入参数或缓存加载 vdc 并切换 client，确保后续 API 请求使用正确机房。"""
    # 优先使用传入的 vdc 参数
    vdc = params.get("vdc")
    if vdc:
        client.byterds.set_region(vdc)
        return
    # 退化到缓存
    vregion = params.get("vregion", "")
    database = params.get("database", "")
    if not vregion or not database:
        return
    region_cfg = _load_region_config(vregion)
    cached = region_cfg.get("instances", {}).get(database, {})
    cached_vdc = cached.get("vdc") or cached.get("region")  # 兼容旧缓存
    if cached_vdc:
        client.byterds.set_region(cached_vdc)


def _byterds_resolve_instance(client, params, ctx, **kwargs):
    """通过 ByteRDS direct client 验证实例存在并写入缓存。遍历所有 vdc 搜索。"""
    database = kwargs.get("database", "") or params.get("database", "")
    vregion = params.get("vregion", "")

    try:
        results = client.byterds.search_databases_all_vdcs(
            keyword=database, page=1, size=10,
        )
    except Exception as e:
        return _error(f"ByteRDS 搜索实例失败: {e}")

    if not isinstance(results, list) or not results:
        return _error(
            f"数据库 '{database}' 在 {vregion} 中不存在（ByteRDS direct）。"
            f"请用户确认名称和地域是否正确，不要做任何额外搜索或重试"
        )

    # 精确匹配 dbname
    exact = [r for r in results if r.get("dbname") == database or r.get("db_name") == database]

    # 同名数据库可能存在于多个机房（VDC），用 vdc 消歧
    if len(exact) > 1:
        target_vdc = kwargs.get("vdc", "")
        if not target_vdc:
            cached = _load_region_config(vregion).get("instances", {}).get(database, {})
            target_vdc = cached.get("vdc") or cached.get("region", "")
        if target_vdc:
            exact = [r for r in exact if r.get("_vdc") == target_vdc]
        if len(exact) != 1:
            all_exact = [r for r in results if r.get("dbname") == database or r.get("db_name") == database]
            candidates = [
                {"name": database, "vdc": r.get("_vdc", ""), "type": "ByteRDS",
                 "owner": r.get("owner", ""), "department": r.get("department", "")}
                for r in (exact if exact else all_exact)
            ]
            return _error(
                f"'{database}' 在多个机房中存在（{', '.join(c['vdc'] for c in candidates)}），请指定机房（vdc）",
                {"fuzzy_matches": candidates}
            )

    matched = exact[0] if exact else (results[0] if len(results) == 1 else None)

    if not matched:
        candidates = [
            {"name": r.get("dbname", r.get("db_name", "")), "vdc": r.get("_vdc", ""),
             "type": "ByteRDS", "status": r.get("status", "")}
            for r in results
        ]
        return _error(
            f"'{database}' 匹配到 {len(candidates)} 个数据库，请指定精确名称",
            {"fuzzy_matches": candidates}
        )

    resolved_name = matched.get("dbname") or matched.get("db_name") or database
    found_vdc = matched.get("_vdc", "")

    # 写入缓存
    region_cfg = _load_region_config(vregion)
    if "instances" not in region_cfg:
        region_cfg["instances"] = {}
    cache_entry = {"instance_type": "ByteRDS"}
    if found_vdc:
        cache_entry["vdc"] = found_vdc
    region_cfg["instances"][resolved_name] = cache_entry
    _save_region_config(vregion, region_cfg)

    # 切换 client 到找到的机房
    if found_vdc:
        client.byterds.set_region(found_vdc)

    return {
        "success": True,
        "instance_id": resolved_name,
        "instance_type": "ByteRDS",
        "database": resolved_name,
        "vdc": found_vdc,
    }


def _byterds_list_instances(client, params, ctx, **kwargs):
    """通过 RDS Platform search API 搜索数据库，遍历所有 vdc"""
    try:
        keyword = kwargs.get("database") or kwargs.get("psm") or ""
        page = kwargs.get("page_number", 1)
        size = kwargs.get("page_size", 20)

        if not keyword:
            return _error("ByteRDS 直连模式需要提供 database 关键词进行搜索", context=ctx)

        results = client.byterds.search_databases_all_vdcs(
            keyword=keyword, page=page, size=size,
        )

        if not isinstance(results, list):
            return _error(f"搜索结果格式异常: {type(results)}", context=ctx)

        normalized = []
        for db in results:
            normalized.append({
                "id": db.get("dbname", "") or db.get("db_name", ""),
                "name": db.get("dbname", "") or db.get("db_name", ""),
                "status": db.get("status", ""),
                "type": "ByteRDS",
                "region": db.get("region", ""),
                "vdc": db.get("_vdc", ""),
                "department": db.get("department", ""),
                "owners": [db.get("owner", "")] if db.get("owner") else [],
                "psm_list": db.get("psms", []),
            })
        return _ok({
            "total": len(normalized),
            "instances": normalized,
        }, f"共 {len(normalized)} 个数据库", context=ctx)
    except Exception as e:
        return _error(f"ByteRDS 搜索数据库失败: {e}", context=ctx)


def _byterds_list_tables(client, params, ctx, **kwargs):
    """通过 RDS Platform API 获取表列表"""
    try:
        dbname = params.get("database", "")

        if not dbname:
            return _error("缺少 database 参数", context=ctx)

        _ensure_vdc(client, params)
        results = client.byterds.list_tables(dbname=dbname)

        if not isinstance(results, list):
            return _error(f"表列表格式异常: {type(results)}", context=ctx)

        tables = [t.get("name", "") for t in results if t.get("name")]
        return _ok({
            "total": len(tables),
            "instance_id": dbname,
            "tables": tables,
        }, f"共 {len(tables)} 张表", context=ctx)
    except Exception as e:
        return _error(f"ByteRDS 获取表列表失败: {e}", context=ctx)


def _byterds_get_table_info(client, params, ctx, **kwargs):
    """通过 RDS Platform API 获取表结构"""
    try:
        dbname = params.get("database", "")
        table = kwargs.get("table", "")

        if not dbname:
            return _error("缺少 database 参数", context=ctx)
        if not table:
            return _error("缺少 table 参数", context=ctx)

        _ensure_vdc(client, params)
        result = client.byterds.get_table_schema(
            dbname=dbname, table=table,
        )

        if not isinstance(result, dict):
            return _ok(result, f"表 {table} 结构获取成功", context=ctx)

        # RDS Platform 返回格式与 DBW 不同，做适配
        columns = result.get("columns", [])
        normalized_columns = []
        for col in columns:
            if isinstance(col, dict):
                normalized_columns.append({
                    "name": col.get("name", col.get("Field", "")),
                    "type": col.get("type", col.get("Type", "")),
                    "nullable": col.get("nullable", col.get("Null", "YES") == "YES"),
                    "primary_key": col.get("primary_key", col.get("Key", "") == "PRI"),
                    "default": col.get("default", col.get("Default")),
                    "comment": col.get("comment", col.get("Comment", "")),
                })
            elif isinstance(col, str):
                normalized_columns.append({"name": col, "type": ""})

        return _ok({
            "name": table,
            "definition": result.get("create_table", result.get("definition", "")),
            "columns": normalized_columns,
        }, f"表 {table} 结构获取成功", context=ctx)
    except Exception as e:
        return _error(f"ByteRDS 获取表结构失败: {e}", context=ctx)


def _byterds_execute_sql(client, params, ctx, **kwargs):
    """通过 RDS Platform API 执行 SQL"""
    try:
        dbname = params.get("database", "")
        sql = kwargs.get("sql", "")

        if not dbname:
            return _error("缺少 database 参数", context=ctx)
        if not sql:
            return _error("缺少 sql 参数", context=ctx)

        _ensure_vdc(client, params)
        result = client.byterds.run_sql(
            dbname=dbname, sql=sql,
        )

        if not isinstance(result, dict):
            return _ok(result, "查询成功", context=ctx)

        # RDS Platform 返回 {amount, data: [{...}], row_order: [...], cost_time_ms}
        amount = result.get("amount", 0)
        data_rows = result.get("data", [])
        row_order = result.get("row_order", [])
        cost_time = result.get("cost_time_ms", 0)

        # 转换为 DBW 兼容格式
        columns = row_order if row_order else (list(data_rows[0].keys()) if data_rows else [])
        rows = []
        for row in data_rows:
            cells = [str(row.get(col, "")) for col in columns]
            rows.append({"Cells": cells})

        return _ok({
            "command_str": sql,
            "state": "Success",
            "row_count": amount,
            "columns": columns,
            "rows": rows,
        }, f"查询成功，{amount} 行", context=ctx)
    except Exception as e:
        return _error(f"ByteRDS SQL 执行失败: {e}", context=ctx)


def _byterds_describe_aggregate_slow_logs(client, params, ctx, **kwargs):
    """通过 RDS Platform slow_log API 查询慢日志汇总，返回格式对齐 DBW describe_aggregate_slow_logs"""
    try:
        dbname = params.get("database", "")
        if not dbname:
            return _error("缺少 database 参数", context=ctx)

        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        if not start_ts or not end_ts:
            return _error("缺少 start_time / end_time 参数", context=ctx)

        order_by = kwargs.get("order_by", "TotalQueryTime")
        sort_by = kwargs.get("sort_by", "DESC")

        search_param = kwargs.get("search_param", {})
        min_qt = search_param.get("MinQueryTime", 0.1)
        sql_methods = search_param.get("SqlMethods", [])
        action = sql_methods[0].lower() if sql_methods else None
        tables = search_param.get("Tables", [])
        table = tables[0] if tables else None

        _ensure_vdc(client, params)
        result = client.byterds.slow_log(
            dbname=dbname,
            start=start_ts,
            end=end_ts,
            query_time=min_qt,
            action=action,
            table=table,
        )

        if not isinstance(result, dict):
            return _ok(result, "慢查询汇总查询完成", context=ctx)

        # 从 abstracts 提取总查询数
        abstracts = result.get("abstracts", [])
        total_count = 0
        for a in abstracts:
            if a.get("name") == "query_count":
                total_count = a.get("value", 0)

        # 将 detail 映射为 DBW 格式的 logs
        # RDS Platform API 的 metrics[] 始终为空，实际统计在扁平字段上：
        #   avg_query_time, max_query_time, count, example_time
        detail = result.get("detail", [])
        logs = []
        for item in detail:
            fingerprint = item.get("fingerprint", "")
            count = item.get("count", 0)
            avg_qt = float(item.get("avg_query_time", 0))
            max_qt = float(item.get("max_query_time", 0))
            count_ratio = round(count / total_count * 100, 2) if total_count else 0

            logs.append({
                "sql_template": fingerprint,
                "db": item.get("db", dbname),
                "user": item.get("user", ""),
                "source_ip": item.get("host", ""),
                "execute_count": count,
                "execute_count_ratio": count_ratio,
                "query_time_stats": {
                    "Average": avg_qt,
                    "Max": max_qt,
                    "Total": round(avg_qt * count, 6),
                },
                "first_appear_time": item.get("ts_min", ""),
                "last_appear_time": item.get("ts_max", ""),
                "sql_fingerprint": fingerprint,
                "sql_method": fingerprint.split()[0].upper() if fingerprint else "",
                "table": "",
                "example_sql": item.get("example", ""),
            })

        # 排序（RDS API 返回全量数据，需在本地排序）
        sort_key_map = {
            "TotalQueryTime": lambda x: x["query_time_stats"]["Total"],
            "AverageQueryTime": lambda x: x["query_time_stats"]["Average"],
            "MaxQueryTime": lambda x: x["query_time_stats"]["Max"],
            "ExecuteCount": lambda x: x["execute_count"],
        }
        sort_fn = sort_key_map.get(order_by, sort_key_map["TotalQueryTime"])
        logs.sort(key=sort_fn, reverse=(sort_by == "DESC"))

        total = len(logs)
        tr = _truncate_list({"logs": logs}, limit=50, label="slow_agg")
        msg = f"共 {total} 条聚合慢查询"
        if tr["truncated"]:
            msg += f"，当前返回 Top {tr['returned_count']}。完整结果已写入 {tr['artifact_path']}"
        return _ok({
            "total": total,
            "returned_count": tr["returned_count"],
            "truncated": tr["truncated"],
            "artifact_path": tr["artifact_path"],
            "logs": tr["logs"],
        }, msg, context=ctx)
    except Exception as e:
        return _error(f"ByteRDS 慢查询汇总失败: {e}", context=ctx)


def _byterds_describe_health_summary(client, params, ctx, **kwargs):
    """通过 RDS Platform diagnostic API 触发健康检查并返回详细报告

    流程: trigger → 轮询 query_record 等完成 → query_report 拿详情
    """
    import time

    try:
        dbname = params.get("database", "")
        if not dbname:
            return _error("缺少 database 参数", context=ctx)

        start_ts = kwargs.get("start_ts")
        end_ts = kwargs.get("end_ts")
        if not start_ts or not end_ts:
            return _error("缺少 start_time / end_time 参数", context=ctx)

        _ensure_vdc(client, params)

        # 1. 触发健康检查
        try:
            client.byterds.trigger_diagnostic(
                dbname=dbname, start_ts=start_ts, end_ts=end_ts,
            )
        except Exception as e:
            return _error(f"触发健康检查失败: {e}", context=ctx)

        # 2. 轮询 query_record 等待完成（最多 60 秒）
        record_id = None
        for _ in range(10):
            data = client.byterds.query_diagnostic_records(dbname=dbname)
            records = data.get("records", []) if isinstance(data, dict) else []
            if records:
                latest = records[0]
                if latest.get("status") == "success":
                    record_id = latest.get("record_id")
                    break
            time.sleep(3)

        if not record_id:
            return _error("健康检查超时，请稍后重试", context=ctx)

        # 3. 查询详细报告
        report = client.byterds.query_diagnostic_report(
            dbname=dbname, record_id=record_id,
        )

        if not isinstance(report, dict):
            return _ok(report, "健康检查完成", context=ctx)

        # 提取关键指标，转为与 DBW 一致的 metrics[] 数组格式
        common = report.get("common", [{}])
        raw = common[0] if common else {}

        metrics_list = [
            {"name": "CPU利用率", "avg": raw.get("cpu", 0), "unit": "%"},
            {"name": "内存利用率", "avg": raw.get("mem", 0), "unit": "%"},
            {"name": "IOPS", "avg": raw.get("iops", 0), "unit": ""},
        ]
        for key, name in [("active_connections", "活跃连接数"), ("qps", "QPS"), ("tps", "TPS")]:
            val = raw.get(key)
            if isinstance(val, dict):
                metrics_list.append({
                    "name": name,
                    "avg": val.get("avg"), "max": val.get("max"), "min": val.get("min"),
                    "unit": "",
                })
            elif isinstance(val, (int, float)):
                metrics_list.append({"name": name, "avg": val, "unit": ""})

        return _ok({
            "instance_id": dbname,
            "record_id": record_id,
            "metrics": metrics_list,
            "diagnostic": {
                "suggests": report.get("suggests", []),
                "slave_status": report.get("slave_status", []),
                "instance": report.get("instance", []),
                "proxy_info": report.get("proxy_info", []),
                "slow_logs": report.get("slow_logs", []),
                "components": report.get("components", {}),
                "auth": report.get("auth", {}),
            },
        }, "健康检查完成", context=ctx)
    except Exception as e:
        return _error(f"ByteRDS 健康检查失败: {e}", context=ctx)


# ──────────────────────────────────────────────
# 辅助：通过 topo API 获取主库 IP/Port
# ──────────────────────────────────────────────

def _get_master_endpoint(client, dbname):
    """从 topo API 获取主库 (ip, port)，失败返回 (None, None)"""
    try:
        topo = client.byterds.topo(dbname=dbname)
        if isinstance(topo, list):
            for node in topo:
                if node.get("role") == "master":
                    return node.get("ip"), node.get("port")
        return None, None
    except Exception:
        return None, None


# ──────────────────────────────────────────────
# list_active_sessions — 通过 information_schema.processlist
# ──────────────────────────────────────────────

def _byterds_list_active_sessions(client, params, ctx, **kwargs):
    """通过 SQL 查询 processlist，返回格式对齐 DBW list_active_sessions"""
    try:
        dbname = params.get("database", "")
        if not dbname:
            return _error("缺少 database 参数", context=ctx)

        show_sleep = kwargs.get("show_sleep", False)

        _ensure_vdc(client, params)
        sql = "SELECT * FROM information_schema.processlist ORDER BY TIME DESC LIMIT 100"
        result = client.byterds.run_sql(dbname=dbname, sql=sql)

        if not isinstance(result, dict) or result.get("err_code"):
            err = result.get("err_msg", str(result)) if isinstance(result, dict) else str(result)
            return _error(f"查询 processlist 失败: {err}", context=ctx)

        records = result.get("data", [])

        sessions = []
        for rec in records:
            command = rec.get("COMMAND", "")
            if not show_sleep and command == "Sleep":
                continue
            sessions.append({
                "process_id": rec.get("ID", ""),
                "user": rec.get("USER", ""),
                "host": rec.get("HOST", ""),
                "db": rec.get("DB", ""),
                "command": command,
                "time": int(rec.get("TIME", 0)),
                "state": rec.get("STATE", ""),
                "info": rec.get("INFO", ""),
            })

        sessions.sort(key=lambda s: s.get("time", 0), reverse=True)
        return _ok({
            "total": len(sessions),
            "sessions": sessions,
        }, f"共 {len(sessions)} 个实时会话", context=ctx)
    except Exception as e:
        return _error(f"list_active_sessions 失败: {e}", context=ctx)


# ──────────────────────────────────────────────
# describe_table_space — 通过 information_schema.tables
# ──────────────────────────────────────────────

def _byterds_describe_table_space(client, params, ctx, **kwargs):
    """通过 SQL 查询表空间，返回格式对齐 DBW describe_table_space"""
    try:
        dbname = params.get("database", "")
        if not dbname:
            return _error("缺少 database 参数", context=ctx)

        filter_database = kwargs.get("filter_database") or dbname

        _ensure_vdc(client, params)
        sql = (
            "SELECT table_name, table_rows, data_length, index_length, data_free, "
            "engine, table_collation, create_time, update_time "
            f"FROM information_schema.tables WHERE table_schema='{filter_database}' "
            "ORDER BY data_length DESC LIMIT 100"
        )
        result = client.byterds.run_sql(dbname=dbname, sql=sql)

        if not isinstance(result, dict) or result.get("err_code"):
            err = result.get("err_msg", str(result)) if isinstance(result, dict) else str(result)
            return _error(f"查询表空间失败: {err}", context=ctx)

        records = result.get("data", [])

        tables = []
        for rec in records:
            # 兼容大小写字段名（不同 vregion 返回格式不一致）
            def _get(key: str, default=0):
                for k in (key, key.upper(), key.lower()):
                    if k in rec:
                        return rec[k]
                return default

            data_len = int(_get("data_length"))
            idx_len = int(_get("index_length"))
            data_free = int(_get("data_free"))
            total = data_len + idx_len
            tables.append({
                "table_name": _get("table_name", ""),
                "table_rows": int(_get("table_rows")),
                "data_length": data_len,
                "index_length": idx_len,
                "data_free": data_free,
                "total_size": total,
                "total_size_mb": round(total / 1024 / 1024, 2),
                "fragmentation_ratio": round(data_free / total * 100, 2) if total > 0 else 0,
                "engine": _get("engine", ""),
            })

        return _ok({
            "database": filter_database,
            "total": len(tables),
            "tables": tables,
        }, f"共 {len(tables)} 张表", context=ctx)
    except Exception as e:
        return _error(f"describe_table_space 失败: {e}", context=ctx)


# ──────────────────────────────────────────────
# describe_instance_nodes — 通过 topo API
# ──────────────────────────────────────────────

def _byterds_describe_instance_nodes(client, params, ctx, **kwargs):
    """通过 topo API 获取节点列表，返回格式对齐 DBW describe_instance_nodes"""
    try:
        dbname = params.get("database", "")
        if not dbname:
            return _error("缺少 database 参数", context=ctx)

        _ensure_vdc(client, params)
        topo = client.byterds.topo(dbname=dbname)

        if not isinstance(topo, list):
            return _error(f"topo 返回格式异常: {type(topo)}", context=ctx)

        nodes = []

        def _collect(node_list):
            for n in node_list:
                nodes.append({
                    "node_id": f"{n.get('ip', '')}:{n.get('port', '')}",
                    "node_type": {"master": "Primary", "slave": "Secondary"}.get(
                        n.get("role", ""), n.get("role", "")),
                    "ip": n.get("ip", ""),
                    "port": n.get("port", 0),
                    "idc": n.get("idc", ""),
                    "mysql_version": n.get("mysql_version_topo", ""),
                })
                _collect(n.get("slaves", []))

        _collect(topo)
        return _ok({
            "instance_id": dbname,
            "nodes": nodes,
        }, f"共 {len(nodes)} 个节点", context=ctx)
    except Exception as e:
        return _error(f"describe_instance_nodes 失败: {e}", context=ctx)


# ──────────────────────────────────────────────
# describe_deadlock — 通过 topo + engine API 解析 InnoDB Status
# ──────────────────────────────────────────────

def _parse_innodb_section(status_text, section_name):
    """从 InnoDB Monitor Output 中提取指定段落"""
    lines = status_text.split("\n")
    result_lines = []
    in_section = False
    for line in lines:
        if section_name in line and line.strip().startswith("-"):
            in_section = True
            continue
        if in_section and line.strip().startswith("---") and line.strip().endswith("---"):
            break
        if in_section:
            result_lines.append(line)
    return "\n".join(result_lines).strip()


def _byterds_describe_deadlock(client, params, ctx, **kwargs):
    """通过 engine API 获取 InnoDB status 并解析死锁段"""
    try:
        dbname = params.get("database", "")
        if not dbname:
            return _error("缺少 database 参数", context=ctx)

        _ensure_vdc(client, params)
        master_ip, master_port = _get_master_endpoint(client, dbname)
        if not master_ip:
            return _error("无法获取主库 IP（topo API 失败）", context=ctx)

        result = client.byterds.engine_status(
            dbname=dbname, ip=master_ip, port=master_port,
        )

        status_text = ""
        if isinstance(result, dict):
            status_text = result.get("Status", "")
        elif isinstance(result, str):
            status_text = result

        if not status_text:
            return _error("InnoDB status 为空", context=ctx)

        # 提取 LATEST DETECTED DEADLOCK 段
        deadlock_text = ""
        lines = status_text.split("\n")
        in_deadlock = False
        dl_lines = []
        for line in lines:
            if "LATEST DETECTED DEADLOCK" in line:
                in_deadlock = True
                continue
            if in_deadlock:
                if line.strip().startswith("-------"):
                    break
                dl_lines.append(line)
        deadlock_text = "\n".join(dl_lines).strip()

        if not deadlock_text:
            return _ok({
                "has_deadlock": False,
                "deadlock_info": "未检测到最近的死锁",
                "master": f"{master_ip}:{master_port}",
            }, "未检测到死锁", context=ctx)

        return _ok({
            "has_deadlock": True,
            "deadlock_info": deadlock_text,
            "master": f"{master_ip}:{master_port}",
        }, "检测到死锁信息", context=ctx)
    except Exception as e:
        return _error(f"describe_deadlock 失败: {e}", context=ctx)


# ──────────────────────────────────────────────
# list_transactions — 通过 topo + engine API 解析事务段
# ──────────────────────────────────────────────

def _byterds_list_transactions(client, params, ctx, **kwargs):
    """通过 engine API 获取 InnoDB status 并解析事务段"""
    try:
        dbname = params.get("database", "")
        if not dbname:
            return _error("缺少 database 参数", context=ctx)

        _ensure_vdc(client, params)
        master_ip, master_port = _get_master_endpoint(client, dbname)
        if not master_ip:
            return _error("无法获取主库 IP（topo API 失败）", context=ctx)

        result = client.byterds.engine_status(
            dbname=dbname, ip=master_ip, port=master_port,
        )

        status_text = ""
        if isinstance(result, dict):
            status_text = result.get("Status", "")
        elif isinstance(result, str):
            status_text = result

        if not status_text:
            return _error("InnoDB status 为空", context=ctx)

        # 提取 TRANSACTIONS 段
        trx_text = _parse_innodb_section(status_text, "TRANSACTIONS")

        # 解析事务列表
        transactions = []
        current_trx = None
        for line in trx_text.split("\n"):
            if line.startswith("---TRANSACTION"):
                if current_trx:
                    transactions.append(current_trx)
                parts = line.split(",")
                trx_id = parts[0].replace("---TRANSACTION", "").strip()
                state = parts[1].strip() if len(parts) > 1 else ""
                current_trx = {
                    "trx_id": trx_id,
                    "state": state,
                    "lock_structs": 0,
                    "row_locks": 0,
                    "detail": "",
                }
            elif current_trx:
                if "lock struct" in line:
                    import re
                    m = re.search(r"(\d+) lock struct", line)
                    if m:
                        current_trx["lock_structs"] = int(m.group(1))
                    m = re.search(r"(\d+) row lock", line)
                    if m:
                        current_trx["row_locks"] = int(m.group(1))
                if line.strip():
                    current_trx["detail"] = (current_trx["detail"] + "\n" + line).strip()

        if current_trx:
            transactions.append(current_trx)

        # 按锁数量降序，有锁的事务排前面
        transactions.sort(key=lambda t: (t["lock_structs"], t["row_locks"]), reverse=True)

        total = len(transactions)
        tr = _truncate_list({"transactions": transactions}, limit=50, label="transactions")
        msg = f"共 {total} 个事务"
        if tr["truncated"]:
            msg += f"，当前返回前 {tr['returned_count']} 个。完整结果已写入 {tr['artifact_path']}"
        return _ok({
            "total": total,
            "returned_count": tr["returned_count"],
            "truncated": tr["truncated"],
            "artifact_path": tr["artifact_path"],
            "master": f"{master_ip}:{master_port}",
            "transactions": tr["transactions"],
        }, msg, context=ctx)
    except Exception as e:
        return _error(f"list_transactions 失败: {e}", context=ctx)


# ──────────────────────────────────────────────
# 工单链接（wildcard，覆盖所有 vregion）
# ──────────────────────────────────────────────

_RDS_TICKET_TYPES = [
    "DML 工单（UPDATE/DELETE/INSERT）",
    "DDL（新建/修改表）",
    "清表工单（RENAME/TRUNCATE/DROP）",
    "参数修改（针对单个实例）",
    "规格升降配",
]


@register("ByteRDS", "get_ticket_url")
def _byterds_get_ticket_url(client, params, ctx, **kwargs):
    """ByteRDS 工单链接 — 区分字节云/火山 URL 模式"""
    database = params.get("database")
    vregion = params.get("vregion") or client.vregion
    host = kwargs["host"]

    # 1. 查缓存获取 vdc + instance_type
    region_cfg = _load_region_config(vregion)
    cached = region_cfg.get("instances", {}).get(database, {})
    vdc = cached.get("vdc")
    cached_type = cached.get("instance_type", "ByteRDS")

    # 2. 无缓存则搜索
    if not vdc:
        try:
            results = client.byterds.search_databases_all_vdcs(database)
            if isinstance(results, list) and results:
                exact = [r for r in results
                         if r.get("dbname") == database or r.get("db_name") == database]
                matched = exact[0] if exact else results[0]
                vdc = matched.get("_vdc", "")
                if vdc:
                    if "instances" not in region_cfg:
                        region_cfg["instances"] = {}
                    entry = region_cfg["instances"].get(database, {})
                    entry["vdc"] = vdc
                    region_cfg["instances"][database] = entry
                    _save_region_config(vregion, region_cfg)
        except Exception:
            pass

    # 3. 仍无结果则 fallback 到默认 vdc（从 _VREGION_TO_VDCS 取第一个）
    if not vdc:
        vdcs = _VREGION_TO_VDCS.get(vregion)
        if vdcs:
            vdc = vdcs[0]

    # 4. 检查 override（特殊 vdc 使用不同 host）
    override = _VDC_API_OVERRIDE.get(vdc) if vdc else None
    if override:
        host = override["host"]

    # 5. vdc → API region（URL 路径用 API region）
    if vdc:
        api_region = _VDC_REGION.get(vdc, vdc)
    else:
        api_region = _VREGION_TO_REGION.get(vregion, "cn")

    # 6. 构造 URL — 根据 cached instance_type 区分模式
    if cached_type == "VeDBMySQL":
        # 火山模式
        url = f"{host}/rds/volc/detail/{api_region}/{database}"
    else:
        # 字节云模式（ByteRDS/MySQL/MySQLSharding 等）
        url = f"{host}/rds/detail/db/{api_region}/{database}/overview"
    url += f"?x-bc-vregion={vregion}"
    if vdc:
        url += f"&x-bc-vdc={vdc}"

    return _ok({
        "url": url,
        "database": database,
        "vdc": vdc,
        "vregion": vregion,
        "guide": "请在浏览器中打开链接，点击页面右上角的「发起工单」按钮创建工单",
        "ticket_types": _RDS_TICKET_TYPES,
    }, "获取工单链接成功", context=ctx)


# ──────────────────────────────────────────────
# 批量注册：为 _BYTERDS_VREGIONS 注册精确匹配 handler
# DBW 不支持这些 vregion，两个后端各管各的，互不重叠
# ──────────────────────────────────────────────

_BYTERDS_FUNC_MAP = {
    "resolve_instance": _byterds_resolve_instance,
    "list_instances": _byterds_list_instances,
    "list_tables": _byterds_list_tables,
    "get_table_info": _byterds_get_table_info,
    "execute_sql": _byterds_execute_sql,
    "describe_aggregate_slow_logs": _byterds_describe_aggregate_slow_logs,
    "describe_health_summary": _byterds_describe_health_summary,
    "list_active_sessions": _byterds_list_active_sessions,
    "describe_table_space": _byterds_describe_table_space,
    "describe_instance_nodes": _byterds_describe_instance_nodes,
    "describe_deadlock": _byterds_describe_deadlock,
    "list_transactions": _byterds_list_transactions,
}

for _fn_name, _fn in _BYTERDS_FUNC_MAP.items():
    register(("ByteRDS", _BYTERDS_VREGIONS), _fn_name)(_fn)
