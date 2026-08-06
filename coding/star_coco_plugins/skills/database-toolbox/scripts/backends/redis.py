"""ByteRedis 后端处理函数（Cache Platform HTTP API）"""

from registry import register
from common import _ok, _error

_REDIS_TICKET_TYPES = ["扩容", "缩容", "执行命令", "删Key"]


@register("ByteRedis", "list_instances")
def _redis_list_instances(client, params, ctx, **kwargs):
    """Cache Platform service/front_list API"""
    try:
        psm = kwargs.get("psm") or params.get("psm")
        favor = kwargs.get("favor", False)
        page = kwargs.get("page", 1)
        page_size = kwargs.get("page_size", 20)
        if psm and not favor:
            raw = client.redis.search_services(keyword=psm, page=page, size=page_size)
        else:
            raw = client.redis.list_starred_services(page=page, size=page_size)
        return _ok(raw, "查询 Redis 服务成功", context=ctx)
    except Exception as e:
        return _error(f"查询 Redis 服务失败: {e}", context=ctx)


@register("ByteRedis", "execute_sql")
def _redis_execute_sql(client, params, ctx, **kwargs):
    """Cache Platform execute_command API，sql 解析为 command + args"""
    try:
        sql = kwargs.get("sql", "")
        parts = sql.strip().split(None, 1)
        command = parts[0].upper() if parts else ""
        args = parts[1] if len(parts) > 1 else ""

        if not command:
            return _error("Redis 命令不能为空", context=ctx)

        # 动态获取服务端支持的命令白名单
        supported = client.redis.list_commands()
        if supported and command not in supported:
            return _error(
                f"Cache Platform 不支持命令 {command}（仅支持只读查询），"
                f"支持的命令：GET, MGET, HGET, HGETALL, HMGET, LRANGE, SMEMBERS, "
                f"ZRANGE, ZRANGEBYSCORE, EXISTS, TTL, TYPE 等（共 {len(supported)} 个）",
                context=ctx)

        # needs: 构造请求所需参数
        needs = ["psm"]
        missing = [n for n in needs if not params.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        raw = client.redis.execute_command(
            psm=params["psm"], command=parts[0], args=args)
        return _ok(raw, f"Redis 命令执行成功: {command}", context=ctx)
    except Exception as e:
        return _error(f"Redis 命令执行失败: {e}", context=ctx)


@register("ByteRedis", "describe_slow_logs")
def _redis_describe_slow_logs(client, params, ctx, **kwargs):
    """Cache Platform slow_log API（内部自动解析集群 + 轮询结果）"""
    try:
        # needs: 构造请求所需参数
        needs = ["psm"]
        missing = [n for n in needs if not params.get(n)]
        if missing:
            return _error(f"缺少参数: {', '.join(missing)}", context=ctx)

        raw = client.redis.slow_log(psm=params["psm"])
        return _ok(raw, "Redis 慢日志查询完成", context=ctx)
    except Exception as e:
        return _error(f"Redis 慢日志查询失败: {e}", context=ctx)


@register("ByteRedis", "get_ticket_url")
def _redis_get_ticket_url(client, params, ctx, **kwargs):
    """ByteRedis 工单链接：通过 search_services 查 service_id 拼接 URL"""
    host = kwargs["host"]
    psm = params.get("psm") or params.get("database")
    if not psm or len(psm.split(".")) < 3:
        return _error(
            f"ByteRedis 需要 PSM 格式参数（如 toutiao.redis.explorer），"
            f"当前值 '{psm}' 不是有效的 PSM",
            context=ctx,
        )
    try:
        svc = client.redis.search_services(psm, page=1, size=10)
        results = svc.get("results", []) if isinstance(svc, dict) else []
        exact = next((s for s in results if s.get("psm") == psm), None)
        matched = exact or (results[0] if results else None)
        if not matched or not matched.get("id"):
            return _error(f"未找到 PSM '{psm}' 对应的 Redis 服务", context=ctx)
        service_id = matched["id"]
    except Exception as e:
        return _error(f"查询 Redis service ID 失败: {e}", context=ctx)

    url = f"{host}/cache/redis-classic/detail/{psm}/service?id={service_id}"
    return _ok({
        "url": url,
        "psm": psm,
        "service_id": service_id,
        "guide": "请在浏览器中打开链接，在服务详情页发起工单",
        "ticket_types": _REDIS_TICKET_TYPES,
    }, "获取工单链接成功", context=ctx)
