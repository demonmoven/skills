"""共享工具函数 - 常量、配置管理、结果封装、参数解析"""

import json
import os
from datetime import datetime, timezone, timedelta
from typing import Any, Optional

# 所有 MySQL 兼容类型（ByteRDS + 多云 MySQL）
_MYSQL_TYPES = {"ByteRDS", "VeDBMySQL", "MySQL", "MySQLSharding"}

_CN_TZ = timezone(timedelta(hours=8))

# ──────────────────────────────────────────────
# 配置管理
# ──────────────────────────────────────────────

_CONFIG_DIR: str = os.path.normpath(
    os.path.join(os.path.dirname(os.path.abspath(__file__)), os.pardir)
)
_GLOBAL_CONFIG_PATH: str = os.path.join(_CONFIG_DIR, ".config.json")

_SKILL_HEADERS = {"X-Bytecloud-Dbw-Skill": "database-toolbox"}

# vregion → site 映射（site 用于域名和 x-top-region header）
_VREGION_TO_SITE: dict[str, str] = {
    "China-North": "cn",
    "ChinaSinf-North": "cn",
    "China-North3": "cn",
    "China-North5": "cn",
    "China-North6": "cn",
    "China-East": "cn",
    "China-Fintech": "cn",
    "China-Pay": "cn",
    "China-Pay2": "cn",
    "China-HKPay": "cn",
    "China-Aggregation": "cn",
    "China-Enterprise": "cn",
    "Aliyun_NC2": "cn",
    "China-BOE": "boe",
    "China-BOE2": "boe",
    "ChinaSinf-BOE": "boe",
    "China-InfBOE": "boe",
    "US-BOE": "boei18n",
    "Asia-SouthEastBD": "i18n-bd",
    "Europe-WestBD": "i18n-bd",
    "Singapore-SaaS": "i18n-bd",
    "Asia-SaaS": "i18n-bd",
    "US-EE": "i18n-bd",
    "Singapore-Common": "i18n-bd",
    "US-EastBD": "i18n-bd",
    "US-TTP3": "i18n-bd",
    "Australia-SouthEastBD": "i18n-bd",
    "Singapore-Central": "i18n-tt",
    "EasternEuro-TT": "i18n-tt",
    "I18N-Game": "i18n-tt",
    "Europe-Central": "i18n-tt",
    "US-East": "i18n-tt",
    "US-West": "i18n-tt",
    "Australia-SouthEast": "i18n-tt",
}

# vregion → RegionId 特殊映射（未列出的 vregion 直接用自身作为 RegionId）
_VREGION_REGION_ID: dict[str, str] = {
    "China-North": "cn",
    "China-Fintech": "sdqd",
    "China-BOE": "boe",
}

# RegionId → vregion 短别名映射（仅存放 region != vregion 的条目）
# region 值与 vregion 相同的情况由 _resolve_vregion 自动处理
_REGION_TO_VREGION: dict[str, str] = {
    "cn": "China-North",
    "sdqd": "China-Fintech",
    "boe": "China-BOE",
    "i18n-bd": "Asia-SouthEastBD",
    "i18n-tt": "Singapore-Central",
}

# site → 域名
_SITE_HOST = {
    "cn": "https://cloud.bytedance.net",
    "boe": "https://cloud-boe.bytedance.net",
    "boei18n": "https://cloud-boei18n.bytedance.net",
    "i18n-tt": "https://cloud.tiktok-row.net",
    "tx-ttp": "https://cloud.tiktok-usts.net",
    "eu-ttp": "https://cloud-eu.tiktok-row.net",
    "i18n-bd": "https://cloud.byteintl.net",
}

_TICKET_STATUS_MAP = {
    "TicketUndo": "未开始",
    "TicketPreCheck": "预检查中",
    "TicketPreCheckError": "预检查失败",
    "TicketExamine": "审批中",
    "TicketCancel": "已取消",
    "TicketReject": "已拒绝",
    "TicketWaitExecute": "等待执行",
    "TicketExecute": "执行中",
    "TicketFinished": "已完成",
    "TicketError": "执行失败",
}

# DBW 支持 ByteDoc/Mongo 的函数白名单
_DBW_SUPPORT_MONGO_FUNCS: set[str] = {
    "list_tables", "get_table_info", "execute_sql", "nl2sql",
    "list_active_sessions", "describe_instance_nodes",
}

# ──────────────────────────────────────────────
# 映射函数
# ──────────────────────────────────────────────

def _instance_type_to_ds_type(instance_type: str) -> str:
    """instance_type → ds_type 映射（ds_type 只有 ByteRDS / ByteDoc / ByteRedis 三个值）"""
    if instance_type in _MYSQL_TYPES:
        return "ByteRDS"
    if instance_type in ("Mongo", "ByteDoc"):
        return "ByteDoc"
    return instance_type  # ByteRedis 等直接透传


def _vregion_to_site(vregion: str) -> str:
    """vregion → site（cn/boe），用于域名选择和 header。"""
    return _VREGION_TO_SITE.get(vregion, "cn")


def _vregion_to_region_id(vregion: str) -> str:
    """vregion → RegionId，用于 API 请求体。China-North 特殊映射为 'cn'，其余直接用 vregion。"""
    return _VREGION_REGION_ID.get(vregion, vregion)


def _resolve_vregion(value: str) -> Optional[str]:
    """智能解析 vregion：先按 vregion 精确匹配，没有再按 region 反查。
    返回 None 表示无法识别。"""
    if value in _VREGION_TO_SITE:
        return value
    return _REGION_TO_VREGION.get(value)


# ──────────────────────────────────────────────
# 配置文件操作
# ──────────────────────────────────────────────

def _region_config_path(vregion: str) -> str:
    new_path = os.path.join(_CONFIG_DIR, f".config.{vregion}.json")
    if os.path.isfile(new_path):
        return new_path
    # 兼容旧格式
    old_path = os.path.join(_CONFIG_DIR, f".config.bytecloud.{vregion}.json")
    if os.path.isfile(old_path):
        return old_path
    return new_path


def _load_json(path: str) -> dict:
    if not os.path.isfile(path):
        return {}
    try:
        with open(path, encoding="utf-8") as f:
            return json.load(f)
    except (json.JSONDecodeError, OSError):
        return {}


def _save_json(path: str, data: dict) -> None:
    try:
        with open(path, "w", encoding="utf-8") as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
    except OSError:
        pass


def _load_global_config() -> dict:
    return _load_json(_GLOBAL_CONFIG_PATH)


def _save_global_config(config: dict) -> None:
    _save_json(_GLOBAL_CONFIG_PATH, config)


_CACHE_TTL_DAYS = 7

def _load_region_config(vregion: str) -> dict:
    data = _load_json(_region_config_path(vregion))
    updated_at = data.get("_updated_at")
    if updated_at:
        try:
            age = datetime.now(timezone.utc) - datetime.fromisoformat(updated_at)
            if age > timedelta(days=_CACHE_TTL_DAYS):
                return {}
        except (ValueError, TypeError):
            pass
    return data


def _save_region_config(vregion: str, config: dict) -> None:
    # 始终写入新格式路径，附加更新时间戳
    config["_updated_at"] = datetime.now(timezone.utc).isoformat()
    _save_json(os.path.join(_CONFIG_DIR, f".config.{vregion}.json"), config)


def _list_known_instances() -> list[dict[str, str]]:
    """扫描所有 region 缓存文件，返回 [{"database": ..., "vregion": ...}, ...]"""
    result = []
    import glob as _glob
    # 扫描新格式和旧格式
    for pattern in (".config.*.json", ".config.bytecloud.*.json"):
        for path in _glob.glob(os.path.join(_CONFIG_DIR, pattern)):
            filename = os.path.basename(path)
            # 跳过全局配置
            if filename == ".config.json":
                continue
            # 解析 vregion
            if filename.startswith(".config.bytecloud."):
                vregion = filename[len(".config.bytecloud."):-len(".json")]
            else:
                vregion = filename[len(".config."):-len(".json")]
            cfg = _load_json(path)
            instances = cfg.get("instances") or {}
            for inst_name in instances:
                result.append({"database": inst_name, "vregion": vregion})
    # 去重
    seen = set()
    deduped = []
    for item in result:
        key = (item["database"], item["vregion"])
        if key not in seen:
            seen.add(key)
            deduped.append(item)
    return deduped


# ──────────────────────────────────────────────
# 结果封装
# ──────────────────────────────────────────────

def _ok(
    data: Any = None,
    message: str = "成功",
    request_id: Optional[str] = None,
    context: Optional[dict] = None,
) -> dict[str, Any]:
    result = {"success": True, "message": message}
    if request_id:
        result["request_id"] = request_id
    if context:
        result["context"] = context
    if data is not None:
        result["data"] = data
    return result


def _error(
    message: str,
    error_detail: Any = None,
    request_id: Optional[str] = None,
    context: Optional[dict] = None,
) -> dict[str, Any]:
    result = {"success": False, "message": message}
    if request_id:
        result["request_id"] = request_id
    if context:
        result["context"] = context
    if error_detail:
        result["error"] = error_detail
    return result


def _truncate_list(data: dict, *, limit: int = 20, label: str = "data") -> dict:
    """对一组列表做统一截断 + 写文件降级，防止撑爆上下文。

    data: dict {key: list} — 所有列表按同一 limit 截断，全部写入 artifact。
    返回 dict: truncated, returned_count, artifact_path + 各 key 的截断后列表。
    不含 total（调用方自行提供真实总数）。
    写文件失败（sandbox）→ 不截断，全量返回。
    """
    max_len = max((len(v) for v in data.values()), default=0)
    if max_len <= limit:
        return {"truncated": False, "returned_count": max_len, "artifact_path": None, **data}

    artifact_path = None
    try:
        import json as _json, tempfile
        tmp = tempfile.NamedTemporaryFile(
            prefix=f"{label}_", suffix=".json", dir="/tmp", delete=False, mode="w")
        _json.dump(data, tmp, ensure_ascii=False, indent=2)
        tmp.close()
        artifact_path = tmp.name
    except Exception:
        pass

    if artifact_path is None:
        return {"truncated": False, "returned_count": max_len, "artifact_path": None, **data}

    return {
        "truncated": True,
        "returned_count": limit,
        "artifact_path": artifact_path,
        **{k: v[:limit] for k, v in data.items()},
    }


def _pop_request_id(data: Any) -> Optional[str]:
    if isinstance(data, dict):
        return data.pop("_request_id", None)
    return None


def _to_result(
    data: Any,
    message: str = "操作成功",
    context: Optional[dict] = None,
) -> dict[str, Any]:
    if data is None:
        return _error("返回数据为空", context=context)
    if isinstance(data, dict):
        rid = _pop_request_id(data)
        if "_raw" in data and data["_raw"] is None:
            return _error("返回数据为空（API 返回 null）", request_id=rid, context=context)
        if "error" in data or "Error" in data:
            error_info = data.get("error") or data.get("Error")
            return _error(f"API错误: {error_info}", data, request_id=rid, context=context)
        return _ok(data, message, request_id=rid, context=context)
    return _ok(data, message, context=context)


def _build_context(
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vregion: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict:
    """构建 context 快照（只含 Agent 可见字段）"""
    ctx = {}
    if vregion:
        ctx["vregion"] = vregion
    if vdc:
        ctx["vdc"] = vdc
    if psm:
        ctx["psm"] = psm
    if database:
        ctx["database"] = database
    if db_type:
        ctx["db_type"] = db_type
    return ctx


def _status_text(status: str) -> str:
    return _TICKET_STATUS_MAP.get(status, status)


# ──────────────────────────────────────────────
# 时间工具
# ──────────────────────────────────────────────

def _ts_to_iso(ts: int, tz: timezone = _CN_TZ) -> str:
    """Unix 时间戳转 ISO 8601 字符串，使用指定时区。"""
    if not ts:
        return ""
    return datetime.fromtimestamp(ts, tz=tz).strftime("%Y-%m-%d %H:%M:%S")


def _detect_tz(value: str) -> timezone:
    """从 ISO 8601 字符串中检测时区，无时区默认北京时间 +08:00。"""
    if value.endswith("Z"):
        return timezone.utc
    for fmt in ("%Y-%m-%dT%H:%M:%S%z",):
        try:
            dt = datetime.strptime(value, fmt)
            if dt.tzinfo is not None:
                return dt.tzinfo
        except ValueError:
            pass
    return _CN_TZ


def _parse_iso_to_timestamp(value: str) -> int:
    """将 ISO 8601 字符串转为 Unix 时间戳（秒）。支持多种格式：
    - 2024-01-01T00:00:00Z          (UTC)
    - 2024-01-01T08:00:00+08:00     (带时区)
    - 2024-01-01 00:00:00           (无时区，默认北京时间 +08:00)
    - 2024-01-01                    (无时区，默认北京时间 +08:00)
    """
    for fmt in (
        "%Y-%m-%dT%H:%M:%S%z",
        "%Y-%m-%dT%H:%M:%SZ",
        "%Y-%m-%d %H:%M:%S",
        "%Y-%m-%dT%H:%M:%S",
        "%Y-%m-%d",
    ):
        try:
            dt = datetime.strptime(value, fmt)
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=_CN_TZ)
            return int(dt.timestamp())
        except ValueError:
            continue
    raise ValueError(f"无法解析时间格式: {value}，请使用 ISO 8601 格式，如 2024-01-01T00:00:00+08:00")


# ──────────────────────────────────────────────
# 参数解析
# ──────────────────────────────────────────────

def _is_redis(db_type: Optional[str], psm: Optional[str]) -> bool:
    return db_type == "ByteRedis" or (bool(psm) and ".redis." in psm)


def _resolve_params(
    client,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict[str, Any]:
    """参数补全。database 和 psm 独立处理，互不覆盖。

    返回值：
    - vregion: 地域信息
    - database: 数据库名称（ByteRDS/ByteDoc 用）
    - psm: PSM（ByteRedis 用）
    - instance_id: 内部系统 ID（API 请求用，Agent 不可见）
    - instance_type, db_type: 类型信息
    """
    global_cfg = _load_global_config()

    _vregion = client.vregion
    _psm = psm
    _instance_type = None
    _db_type = db_type

    # Redis: 不需要 database、instance_id、缓存查找
    if _is_redis(db_type, psm):
        _database = database
        _instance_id = None
        _instance_type = "ByteRedis"
    else:
        _database = database or global_cfg.get("default_database")
        _instance_id = _database  # 初始值 = database，多云由缓存 real_instance_id 覆盖

        # 缓存查找
        if _database and not _vregion:
            default_vregion = global_cfg.get("default_vregion")
            if default_vregion:
                region_cfg = _load_region_config(default_vregion)
                if _database in (region_cfg.get("instances") or {}):
                    _vregion = default_vregion

        if _database and _vregion:
            region_cfg = _load_region_config(_vregion)
            cached = (region_cfg.get("instances") or {}).get(_database, {})
            if cached.get("instance_type"):
                _instance_type = cached["instance_type"]
            if cached.get("real_instance_id"):
                _instance_id = cached["real_instance_id"]

        if not _instance_type:
            _instance_type = _db_type or global_cfg.get("default_instance_type", "ByteRDS")
    if not _db_type:
        _db_type = _instance_type_to_ds_type(_instance_type)

    result = {
        "vregion": _vregion,
        "database": _database,
        "psm": _psm,
        "instance_id": _instance_id,
        "instance_type": _instance_type,
        "db_type": _db_type,
    }
    if vdc:
        result["vdc"] = vdc
    return result


def _prepare(
    client,
    database: Optional[str] = None,
    psm: Optional[str] = None,
    db_type: Optional[str] = None,
    vdc: Optional[str] = None,
) -> dict[str, Any]:
    """参数解析 + 实例解析。返回带真实 instance_type 的 params 和 ctx。
    类型支持检查由 dispatch 负责，needs 由调用方在函数体中显式校验。
    实例解析通过 dispatch("resolve_instance", ...) 路由到对应后端。
    vdc: 可选，同名数据库在多个机房时用于消歧。
    """
    from registry import dispatch

    params = _resolve_params(client, database=database, psm=psm, db_type=db_type,
                             vdc=vdc)
    ctx = _build_context(database=params["database"], psm=params["psm"],
                         db_type=params.get("db_type"), vregion=params.get("vregion"),
                         vdc=vdc)

    if params["db_type"] == "ByteRedis":
        # Redis: type 立刻已知，验证 psm 前置条件
        if not params.get("psm"):
            return {"ok": False, "error": _error("缺少 psm", context=ctx)}
        if len(params["psm"].split(".")) < 3:
            return {"ok": False, "error": _error(
                f"PSM 格式无效（需要 a.b.c 格式）: {params['psm']}", context=ctx
            )}
        return {"ok": True, "params": params, "ctx": ctx}

    # Non-Redis: 验证 database → resolve 得到真实 instance_type
    if not params.get("database"):
        return {"ok": False, "error": _error("缺少 database", context=ctx)}
    if not params.get("vregion"):
        return {"ok": False, "error": _error(
            "缺少 vregion，请在 create_client 中指定", context=ctx
        )}
    resolved = dispatch(
        "resolve_instance", params["db_type"], params["vregion"],
        client, params, ctx,
        database=params["database"], vdc=vdc,
    )

    if resolved is None:
        return {"ok": False, "error": _error(
            f"{params['db_type']} 不支持 resolve_instance（vregion={params['vregion']}）", context=ctx
        )}
    if not resolved.get("success"):
        resolved["context"] = ctx
        return {"ok": False, "error": resolved}
    params["instance_id"] = resolved["instance_id"]
    params["instance_type"] = resolved["instance_type"]
    if resolved.get("database"):
        params["database"] = resolved["database"]
    params["db_type"] = _instance_type_to_ds_type(params["instance_type"])
    # resolve 可能确定了 vdc（从缓存或匹配结果）
    if not vdc and resolved.get("vdc"):
        params["vdc"] = resolved["vdc"]
    ctx = _build_context(database=params["database"],
                         db_type=params["db_type"],
                         vregion=params.get("vregion"),
                         vdc=params.get("vdc"))

    return {"ok": True, "params": params, "ctx": ctx}
