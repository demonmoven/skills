"""
ByteCloud Database Toolbox - 纯函数式 API

提供独立函数与 ToolboxClient 配合使用，用于与字节云 Database Workbench API 交互。
所有函数无状态，不修改传入参数。

核心设计原则：
1. 所有函数返回统一的字典格式 {success, message, data, context}
2. 函数无副作用（除写入缓存文件外）
3. AI Agent 可从返回值的 context 中获取已解析的参数，用于后续调用
"""

from typing import Any, Literal, Optional, List

# ── 导入 backends 以触发 @register（必须在 dispatch 之前）──
import backends  # noqa: F401

from common import (
    _ok, _error, _to_result, _pop_request_id, _build_context,
    _prepare, _is_redis,
    _parse_iso_to_timestamp, _detect_tz,
    _vregion_to_site,
    _instance_type_to_ds_type,
    _load_global_config, _save_global_config,
    _list_known_instances,
    _SITE_HOST,
)
from registry import dispatch
from client import ToolboxClient, create_client




# ──────────────────────────────────────────────
# 公开函数：元数据
# ──────────────────────────────────────────────

def list_instances(
    client: "ToolboxClient",
    db_type: Literal["ByteRDS", "ByteDoc", "ByteRedis"] = "ByteRDS",
    database: Optional[str] = None,
    psm: Optional[str] = None,
    instance_status: Optional[str] = None,
    page_number: int = 1,
    page_size: int = 10,
    favor: bool = False,
    owned: bool = False,
) -> dict[str, Any]:
    """查询数据库实例列表。须传过滤项（database/psm/favor/owned），查全量无意义。"""
    try:
        if not database and not psm and not favor and not owned:
            ctx = _build_context(vregion=client.vregion)
            return _error("请提供过滤条件：database（数据库名）、psm、favor=True（收藏）或 owned=True（自建）", context=ctx)
        _search = psm or database
        if db_type == "ByteRedis":
            ctx = _build_context(psm=_search, vregion=client.vregion, db_type="ByteRedis")
            _db_type = "ByteRedis"
        else:
            ctx = _build_context(vregion=client.vregion)
            _db_type = db_type

        params = {"instance_type": _db_type, "db_type": _db_type,
                  "psm": _search, "vregion": client.vregion}

        result = dispatch("list_instances", _db_type, client.vregion, client, params, ctx,
                          database=database, psm=psm,
                          instance_status=instance_status,
                          page_number=page_number, page_size=page_size,
                          favor=favor, owned=owned)
        if result is not None:
            return result
        return _error(f"{_db_type} 不支持 list_instances", context=ctx)
    except Exception as e:
        return _error(f"describe_instances失败: {str(e)}")


def list_tables(
    client: "ToolboxClient",
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    page_number: int = 1,
    page_size: int = 50,
    fetch_all: bool = False,
) -> dict[str, Any]:
    """列出数据库中的表。"""
    try:
        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("list_tables", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          fetch_all=fetch_all, page_number=page_number, page_size=page_size)
        if result is not None:
            return result
        if p["db_type"] == "ByteRedis":
            return _error(
                "ByteRedis 是 Key-Value 存储，没有表结构。"
                "请直接使用 execute_sql(client, sql='GET key', psm='...', db_type='ByteRedis') 执行 Redis 只读命令",
                context=ctx,
            )
        return _error(f"{p['db_type']} 不支持 list_tables", context=ctx)
    except Exception as e:
        return _error(f"list_tables失败: {str(e)}")


def get_table_info(
    client: "ToolboxClient",
    table: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict[str, Any]:
    """获取表结构（列名、类型、主键、注释等）。"""
    try:
        if not table:
            return _error("table 参数不能为空")

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("get_table_info", p["db_type"], p.get("vregion", ""),
                          client, p, ctx, table=table)
        if result is not None:
            return result
        if p["db_type"] == "ByteRedis":
            return _error(
                "ByteRedis 是 Key-Value 存储，没有表结构。"
                "请直接使用 execute_sql(client, sql='GET key', psm='...', db_type='ByteRedis') 执行 Redis 只读命令",
                context=ctx,
            )
        return _error(f"{p['db_type']} 不支持 get_table_info", context=ctx)
    except Exception as e:
        return _error(f"get_table_info失败: {str(e)}")


def search_cached_instances(
    keyword: Optional[str] = None,
    vregion: Optional[str] = None,
) -> dict[str, Any]:
    """搜索本地缓存的数据库列表。纯本地操作，不需要 client。"""
    all_cached = _list_known_instances()
    results = all_cached
    if vregion:
        results = [d for d in results if d["vregion"] == vregion]
    if keyword:
        kw = keyword.lower()
        results = [d for d in results if kw in d["database"].lower()]
    return _ok({
        "total": len(results),
        "instances": results,
    }, f"共 {len(results)} 个缓存数据库（总缓存 {len(all_cached)} 个）")


def set_global_config(**kwargs) -> dict[str, Any]:
    """设置全局配置（default_database, default_vregion, default_instance_type 等）。纯本地操作。"""
    config = _load_global_config()
    config.update(kwargs)
    _save_global_config(config)
    return _ok(config, "全局配置已更新")


def get_ticket_url(
    client: "ToolboxClient",
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict[str, Any]:
    """获取工单创建页面链接。"""
    global_cfg = _load_global_config()
    _db_type = db_type or global_cfg.get("default_instance_type", "ByteRDS")
    _db_type = _instance_type_to_ds_type(_db_type)
    if _is_redis(_db_type, psm):
        _database = psm
        _db_type = "ByteRedis"
    else:
        _database = database or (psm.split(".")[-1] if psm else None) or global_cfg.get("default_database")

    ctx = _build_context(database=_database, psm=psm, db_type=_db_type,
                         vregion=client.vregion, vdc=vdc)

    if not _database and not psm:
        return _error("缺少参数: database", {"missing": ["database"]}, context=ctx)

    _site = _vregion_to_site(client.vregion)
    _host = _SITE_HOST.get(_site, _SITE_HOST["cn"])

    params = {"database": _database, "psm": psm, "vregion": client.vregion}
    result = dispatch("get_ticket_url", _db_type, client.vregion,
                      client, params, ctx, host=_host)
    if result is not None:
        return result
    return _error(f"{_db_type} 不支持 get_ticket_url", context=ctx)


# ──────────────────────────────────────────────
# 公开函数：数据查询与分析
# ──────────────────────────────────────────────

def nl2sql(
    client: "ToolboxClient",
    query: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    tables: Optional[List[str]] = None,
) -> dict[str, Any]:
    """自然语言转 SQL（生成但不执行）。"""
    try:
        if not query:
            return _error("query 参数不能为空")

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("nl2sql", p["db_type"], p.get("vregion", ""),
                          client, p, ctx, query=query, tables=tables)
        if result is not None:
            return result
        if p["db_type"] == "ByteRedis":
            return _error(
                "ByteRedis 是 Key-Value 存储，不支持 SQL。"
                "请直接使用 execute_sql(client, sql='GET key', psm='...', db_type='ByteRedis') 执行 Redis 只读命令",
                context=ctx,
            )
        return _error(f"{p['db_type']} 不支持 nl2sql", context=ctx)
    except Exception as e:
        return _error(f"nl2sql失败: {str(e)}")


def execute_sql(
    client: "ToolboxClient",
    sql: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict[str, Any]:
    """执行 SQL 查询（仅支持 SELECT/SHOW TABLES/SHOW CREATE TABLE/EXPLAIN）。"""
    try:
        if not sql:
            return _error("sql 参数不能为空")

        if not _is_redis(db_type, psm) and psm and not database:
            database = psm.split(".")[-1]
        prep = _prepare(client, database=database, psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("execute_sql", p["db_type"], p.get("vregion", ""),
                          client, p, ctx, sql=sql)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 execute_sql", context=ctx)
    except Exception as e:
        return _error(f"execute_sql失败: {str(e)}")


def query_sql(
    client: "ToolboxClient",
    sql: str,
    database: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
):
    """执行 SELECT/SHOW 查询并返回 pandas DataFrame。需要安装 pandas。"""
    try:
        import pandas as pd
    except ImportError:
        return _error("query_sql 需要 pandas 库，请先执行: pip install pandas")

    result = execute_sql(client, sql=sql, database=database, db_type=db_type, vdc=vdc)
    if not result.get("success"):
        return result

    data = result.get("data") or {}
    columns = data.get("columns") or []
    rows = data.get("rows") or []
    records = [row["Cells"] for row in rows if isinstance(row, dict) and "Cells" in row]
    return pd.DataFrame(records, columns=columns)


# ──────────────────────────────────────────────
# 公开函数：慢查询诊断
# ──────────────────────────────────────────────

def describe_slow_logs(
    client: "ToolboxClient",
    start_time: str,
    end_time: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    page_number: int = 1,
    page_size: int = 10,
    order_by: str = "QueryTime",
    sort_by: str = "DESC",
    sql_template_id: Optional[str] = None,
) -> dict[str, Any]:
    """查询慢查询日志明细。start_time/end_time 使用 ISO 8601 格式，如 2024-01-01T00:00:00Z。"""
    try:
        if not start_time or not end_time:
            return _error("start_time 和 end_time 不能为空，请使用 ISO 8601 格式，如 2024-01-01T00:00:00Z")
        try:
            start_ts = _parse_iso_to_timestamp(start_time)
            end_ts = _parse_iso_to_timestamp(end_time)
            tz = _detect_tz(start_time)
        except ValueError as ve:
            return _error(str(ve))

        if not _is_redis(db_type, psm) and psm and not database:
            database = psm.split(".")[-1]
        prep = _prepare(client, database=database, psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("describe_slow_logs", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          start_ts=start_ts, end_ts=end_ts, tz=tz,
                          page_number=page_number, page_size=page_size,
                          order_by=order_by, sort_by=sort_by,
                          sql_template_id=sql_template_id)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 describe_slow_logs", context=ctx)
    except Exception as e:
        return _error(f"describe_slow_logs失败: {str(e)}")


def list_slow_query_advice(
    client: "ToolboxClient",
    summary_id: str,
    advice_type: str,
    group_by: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    order_by: str = "Benefit",
    page_number: int = 1,
    page_size: int = 10,
) -> dict[str, Any]:
    """获取慢查询优化建议。"""
    try:
        if not summary_id or not advice_type or not group_by:
            return _error("summary_id, advice_type, group_by 不能为空")

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("list_slow_query_advice", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          summary_id=summary_id, advice_type=advice_type, group_by=group_by,
                          order_by=order_by, page_number=page_number, page_size=page_size)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 list_slow_query_advice", context=ctx)
    except Exception as e:
        return _error(f"list_slow_query_advice失败: {str(e)}")


def slow_query_advice_task_history(
    client: "ToolboxClient",
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    page_number: int = 1,
    page_size: int = 10,
) -> dict[str, Any]:
    """查询慢查询诊断历史。BOE 环境不支持该接口。"""
    try:
        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("slow_query_advice_task_history", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          page_number=page_number, page_size=page_size)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 slow_query_advice_task_history", context=ctx)
    except Exception as e:
        return _error(f"slow_query_advice_task_history失败: {str(e)}")


def describe_aggregate_slow_logs(
    client: "ToolboxClient",
    start_time: str,
    end_time: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    page_number: int = 1,
    page_size: int = 10,
    order_by: str = "TotalQueryTime",
    sort_by: str = "DESC",
    users: Optional[List[str]] = None,
    source_ips: Optional[List[str]] = None,
    keywords: Optional[List[str]] = None,
    tables: Optional[List[str]] = None,
    sql_methods: Optional[List[str]] = None,
    min_query_time: Optional[float] = None,
    max_query_time: Optional[float] = None,
    group_ignored: Optional[List[str]] = None,
) -> dict[str, Any]:
    """查询慢查询聚合统计。start_time/end_time 使用 ISO 8601 格式。"""
    try:
        if not start_time or not end_time:
            return _error("start_time 和 end_time 不能为空，请使用 ISO 8601 格式，如 2024-01-01T00:00:00Z")
        try:
            start_ts = _parse_iso_to_timestamp(start_time)
            end_ts = _parse_iso_to_timestamp(end_time)
            tz = _detect_tz(start_time)
        except ValueError as ve:
            return _error(str(ve))

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        # 构建 SearchParam
        search_param: dict[str, Any] = {
            "GroupIgnored": group_ignored if group_ignored is not None else ["User", "SourceIP", "PSM"],
        }
        if users:
            search_param["Users"] = users
        if source_ips:
            search_param["SourceIPs"] = source_ips
        if keywords:
            search_param["Keywords"] = keywords
        if tables:
            search_param["Tables"] = tables
        if sql_methods:
            search_param["SqlMethods"] = sql_methods
        if min_query_time is not None:
            search_param["MinQueryTime"] = min_query_time
        if max_query_time is not None:
            search_param["MaxQueryTime"] = max_query_time

        result = dispatch("describe_aggregate_slow_logs", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          start_ts=start_ts, end_ts=end_ts, tz=tz,
                          page_number=page_number, page_size=page_size,
                          order_by=order_by, sort_by=sort_by,
                          search_param=search_param)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 describe_aggregate_slow_logs", context=ctx)
    except Exception as e:
        return _error(f"describe_aggregate_slow_logs失败: {str(e)}")


def slow_query_trend(
    client: "ToolboxClient",
    start_time: str,
    end_time: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    interval: int = 300,
    users: Optional[List[str]] = None,
    source_ips: Optional[List[str]] = None,
    min_query_time: Optional[float] = None,
    max_query_time: Optional[float] = None,
) -> dict[str, Any]:
    """查询慢查询时间序列趋势。start_time/end_time 使用 ISO 8601 格式。"""
    try:
        if not start_time or not end_time:
            return _error("start_time 和 end_time 不能为空，请使用 ISO 8601 格式，如 2024-01-01T00:00:00Z")
        try:
            start_ts = _parse_iso_to_timestamp(start_time)
            end_ts = _parse_iso_to_timestamp(end_time)
        except ValueError as ve:
            return _error(str(ve))

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        # 构建 SearchParam
        search_param: dict[str, Any] = {}
        if users:
            search_param["Users"] = users
        if source_ips:
            search_param["SourceIPs"] = source_ips
        if min_query_time is not None:
            search_param["MinQueryTime"] = min_query_time
        if max_query_time is not None:
            search_param["MaxQueryTime"] = max_query_time

        result = dispatch("slow_query_trend", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          start_ts=start_ts, end_ts=end_ts,
                          interval=interval, search_param=search_param)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 slow_query_trend", context=ctx)
    except Exception as e:
        return _error(f"slow_query_trend失败: {str(e)}")


def describe_full_sql_detail(
    client: "ToolboxClient",
    start_time: str,
    end_time: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    page_size: int = 50,
    users: Optional[List[str]] = None,
    source_ips: Optional[List[str]] = None,
    keywords: Optional[List[str]] = None,
    tables: Optional[List[str]] = None,
    sql_methods: Optional[List[str]] = None,
    min_exec_time: Optional[int] = None,
    max_exec_time: Optional[int] = None,
    context: Optional[str] = None,
) -> dict[str, Any]:
    """查询完整 SQL 历史详情。start_time/end_time 使用 ISO 8601 格式。"""
    try:
        if not start_time or not end_time:
            return _error("start_time 和 end_time 不能为空，请使用 ISO 8601 格式，如 2024-01-01T00:00:00+08:00")
        try:
            start_ts = _parse_iso_to_timestamp(start_time)
            end_ts = _parse_iso_to_timestamp(end_time)
        except ValueError as ve:
            return _error(str(ve))

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        # 构建 SearchParam
        search_param: dict[str, Any] = {}
        if users:
            search_param["Users"] = users
        if source_ips:
            search_param["SourceIPs"] = source_ips
        if keywords:
            search_param["KeyWords"] = keywords
        if tables:
            search_param["Tables"] = tables
        if sql_methods:
            search_param["SqlMethods"] = sql_methods
        if min_exec_time is not None:
            search_param["DuringDown"] = min_exec_time
        if max_exec_time is not None:
            search_param["DuringUp"] = max_exec_time

        result = dispatch("describe_full_sql_detail", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          start_ts=start_ts, end_ts=end_ts,
                          page_size=page_size, search_param=search_param,
                          context=context)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 describe_full_sql_detail", context=ctx)
    except Exception as e:
        return _error(f"describe_full_sql_detail失败: {str(e)}")


# ──────────────────────────────────────────────
# 公开函数：开发变更（工单）
# ──────────────────────────────────────────────

def create_dml_sql_change_ticket(client, sql_text, **kwargs):
    raise NotImplementedError("create_dml_sql_change_ticket 当前不可用：字节云平台暂不支持工单相关接口")

def create_ddl_sql_change_ticket(client, sql_text, **kwargs):
    raise NotImplementedError("create_ddl_sql_change_ticket 当前不可用：字节云平台暂不支持工单相关接口")

def describe_tickets(client, list_type, **kwargs):
    raise NotImplementedError("describe_tickets 当前不可用：字节云平台暂不支持该接口")

def describe_ticket_detail(client, ticket_id):
    raise NotImplementedError("describe_ticket_detail 当前不可用：字节云平台暂不支持工单相关接口")

def describe_workflow(client, ticket_id):
    raise NotImplementedError("describe_workflow 当前不可用：字节云平台暂不支持工单相关接口")


# ──────────────────────────────────────────────
# 公开函数：监控指标
# ──────────────────────────────────────────────

def get_metric_items(client, instance_type=None):
    raise NotImplementedError("get_metric_items 当前不可用：后端仅支持 MySQL 实例类型，ByteRDS 不支持")

def get_metric_data(client, metric_name, start_time, end_time, **kwargs):
    raise NotImplementedError("get_metric_data 当前不可用：后端仅支持 MySQL 实例类型，ByteRDS 不支持")


def table_write_analysis(
    client: "ToolboxClient",
    start_time: str,
    end_time: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    tables: Optional[List[str]] = None,
    order_by: str = "ExecCount",
    sort_by: Literal["ASC", "DESC"] = "ASC",
) -> dict[str, Any]:
    """写分析：按表维度聚合全量 SQL 统计。"""
    try:
        if not start_time or not end_time:
            return _error("start_time 和 end_time 不能为空，请使用 ISO 8601 格式，如 2024-01-01 08:00:00")
        try:
            start_ts = _parse_iso_to_timestamp(start_time)
            end_ts = _parse_iso_to_timestamp(end_time)
        except ValueError as ve:
            return _error(str(ve))

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("table_write_analysis", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          start_ts=start_ts, end_ts=end_ts,
                          tables=tables, order_by=order_by, sort_by=sort_by)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 table_write_analysis", context=ctx)
    except Exception as e:
        return _error(f"table_write_analysis失败: {str(e)}")


def describe_table_metric(
    client: "ToolboxClient",
    item_type: Literal["DML", "DDL"],
    db_name: str,
    table: str,
    start_time: str,
    end_time: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict[str, Any]:
    """获取表级别监控指标。"""
    try:
        if not start_time or not end_time:
            return _error("start_time 和 end_time 不能为空，请使用 ISO 8601 格式，如 2024-01-01 08:00:00")
        try:
            start_ts = _parse_iso_to_timestamp(start_time)
            end_ts = _parse_iso_to_timestamp(end_time)
        except ValueError as ve:
            return _error(str(ve))

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("describe_table_metric", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          start_ts=start_ts, end_ts=end_ts,
                          item_type=item_type, db_name=db_name, table=table)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 describe_table_metric", context=ctx)
    except Exception as e:
        return _error(f"describe_table_metric失败: {str(e)}")


def get_metric_data_predict(
    client: "ToolboxClient",
    metric_name: str,
    start_time: str,
    end_time: str,
    period: int = 60,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict[str, Any]:
    """获取监控数据预测。仅 BOE 可用（通过 register 精确匹配控制）。"""
    try:
        if not start_time or not end_time:
            return _error("start_time 和 end_time 不能为空，请使用 ISO 8601 格式")
        try:
            start_ts = _parse_iso_to_timestamp(start_time)
            end_ts = _parse_iso_to_timestamp(end_time)
        except ValueError as ve:
            return _error(str(ve))

        if end_ts <= start_ts:
            return _error("end_time 必须晚于 start_time")
        if end_ts - start_ts > 7 * 86400:
            return _error("时间跨度不能超过 7 天，请缩小查询范围")
        import time as _time
        now_ts = int(_time.time())
        if end_ts > now_ts:
            end_ts = now_ts
        if start_ts >= end_ts:
            start_ts = end_ts - 3600

        _database = database or (psm.split(".")[-1] if psm else None)
        prep = _prepare(client, database=_database, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("get_metric_data_predict", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          metric_name=metric_name, period=period,
                          start_ts=start_ts, end_ts=end_ts)
        if result is not None:
            return result
        return _error(f"当前 vregion ({p.get('vregion', '')}) 不支持 get_metric_data_predict（仅 BOE 可用）", context=ctx)
    except Exception as e:
        return _error(f"get_metric_data_predict失败: {str(e)}")


def describe_instance_nodes(
    client: "ToolboxClient",
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict[str, Any]:
    """查询实例节点列表。"""
    try:
        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("describe_instance_nodes", p["db_type"], p.get("vregion", ""),
                          client, p, ctx)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 describe_instance_nodes", context=ctx)
    except Exception as e:
        return _error(f"describe_instance_nodes失败: {str(e)}")


def describe_health_summary(
    client: "ToolboxClient",
    start_time: str,
    end_time: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    node_ids: Optional[List[str]] = None,
    diag_type: str = "ALL",
) -> dict[str, Any]:
    """查询实例健康概览。"""
    try:
        if not start_time or not end_time:
            return _error("start_time 和 end_time 不能为空，请使用 ISO 8601 格式")
        try:
            start_ts = _parse_iso_to_timestamp(start_time)
            end_ts = _parse_iso_to_timestamp(end_time)
        except ValueError as ve:
            return _error(str(ve))
        if end_ts <= start_ts:
            return _error("end_time 必须晚于 start_time")

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("describe_health_summary", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          start_ts=start_ts, end_ts=end_ts,
                          node_ids=node_ids, diag_type=diag_type)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 describe_health_summary", context=ctx)
    except Exception as e:
        return _error(f"describe_health_summary失败: {str(e)}")


# ──────────────────────────────────────────────
# 公开函数：会话与锁诊断
# ──────────────────────────────────────────────

def list_active_sessions(
    client: "ToolboxClient",
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    show_sleep: bool = False,
) -> dict[str, Any]:
    """查询实时连接/进程列表。"""
    try:
        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("list_active_sessions", p["db_type"], p.get("vregion", ""),
                          client, p, ctx, show_sleep=show_sleep)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 list_active_sessions", context=ctx)
    except Exception as e:
        return _error(f"list_active_sessions失败: {str(e)}")


def describe_deadlock(
    client: "ToolboxClient",
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict[str, Any]:
    """查询死锁信息。"""
    try:
        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("describe_deadlock", p["db_type"], p.get("vregion", ""),
                          client, p, ctx)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 describe_deadlock", context=ctx)
    except Exception as e:
        return _error(f"describe_deadlock失败: {str(e)}")


def list_transactions(
    client: "ToolboxClient",
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    page_number: int = 1,
    page_size: int = 10,
) -> dict[str, Any]:
    """查询事务和锁列表。"""
    try:
        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("list_transactions", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          page_number=page_number, page_size=page_size)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 list_transactions", context=ctx)
    except Exception as e:
        return _error(f"list_transactions失败: {str(e)}")


def export_transactions(
    client: "ToolboxClient",
    start_time: str,
    end_time: str,
    task_name: str = "",
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict[str, Any]:
    """创建事务导出任务。"""
    try:
        if not task_name:
            return _error("task_name 参数不能为空")
        if not start_time or not end_time:
            return _error("start_time 和 end_time 不能为空，请使用 ISO 8601 格式")
        try:
            start_ts = _parse_iso_to_timestamp(start_time)
            end_ts = _parse_iso_to_timestamp(end_time)
        except ValueError as ve:
            return _error(str(ve))

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("export_transactions", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          task_name=task_name, start_ts=start_ts, end_ts=end_ts)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 export_transactions", context=ctx)
    except Exception as e:
        return _error(f"export_transactions失败: {str(e)}")


def transaction_snapshots(
    client: "ToolboxClient",
    start_time: str,
    end_time: str,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    page_number: int = 1,
    page_size: int = 10,
) -> dict[str, Any]:
    """查询事务快照。"""
    try:
        if not start_time or not end_time:
            return _error("start_time 和 end_time 不能为空，请使用 ISO 8601 格式")
        try:
            start_ts = _parse_iso_to_timestamp(start_time)
            end_ts = _parse_iso_to_timestamp(end_time)
        except ValueError as ve:
            return _error(str(ve))

        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("transaction_snapshots", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          start_ts=start_ts, end_ts=end_ts,
                          page_number=page_number, page_size=page_size)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 transaction_snapshots", context=ctx)
    except Exception as e:
        return _error(f"transaction_snapshots失败: {str(e)}")


# ──────────────────────────────────────────────
# 错误日志与存储空间
# ──────────────────────────────────────────────

def describe_err_logs(client, start_time, end_time, **kwargs):
    raise NotImplementedError("describe_err_logs 当前不可用：后端仅支持 MySQL/VeDB/Postgres，ByteRDS 不支持")


def describe_table_space(
    client: "ToolboxClient",
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
    page_number: int = 1,
    page_size: int = 10,
    filter_database: Optional[str] = None,
    table_name: Optional[str] = None,
) -> dict[str, Any]:
    """查询表空间详情。"""
    try:
        prep = _prepare(client, database=database or (psm.split(".")[-1] if psm else None),
                        psm=psm, db_type=db_type, vdc=vdc)
        if not prep["ok"]:
            return prep["error"]
        p, ctx = prep["params"], prep["ctx"]

        result = dispatch("describe_table_space", p["db_type"], p.get("vregion", ""),
                          client, p, ctx,
                          page_number=page_number, page_size=page_size,
                          filter_database=filter_database, table_name=table_name)
        if result is not None:
            return result
        return _error(f"{p['db_type']} 不支持 describe_table_space", context=ctx)
    except Exception as e:
        return _error(f"describe_table_space失败: {str(e)}")


def describe_table_spaces(client, session_id, **kwargs):
    raise NotImplementedError("describe_table_spaces 当前不可用")


# ──────────────────────────────────────────────
# ByteRedis 独有公开函数
# ──────────────────────────────────────────────

def redis_list_big_keys(
    client,
    psm: Optional[str] = None,
    date: Optional[str] = None,
    begin: str = "00:00:00",
    end: str = "23:59:59",
    page: int = 1,
    page_size: int = 10,
    key_type: str = "string",
    db_type: Optional[str] = None,
) -> dict[str, Any]:
    """列出 Redis 大 Key。psm 传 PSM，date 传 YYYY-MM-DD。"""
    prep = _prepare(client, psm=psm, db_type=db_type or "ByteRedis")
    if not prep["ok"]:
        return prep["error"]
    p, ctx = prep["params"], prep["ctx"]
    if _vregion_to_site(client.vregion) == "boe":
        return _error("redis_list_big_keys 在 BOE 环境不可用", context=ctx)
    if not date:
        return _error("缺少 date 参数（格式 YYYY-MM-DD）", context=ctx)
    try:
        needs = ["psm"]
        missing = [n for n in needs if not p.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        raw = client.redis.list_big_keys(
            psm=p["psm"], date=date, begin=begin, end=end,
            page=page, size=page_size, key_type=key_type)
        return _ok(raw, "Redis 大 Key 查询完成", context=ctx)
    except Exception as e:
        return _error(f"Redis 大 Key 查询失败: {e}", context=ctx)
