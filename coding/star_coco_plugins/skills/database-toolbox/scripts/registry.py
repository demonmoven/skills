"""后端分发引擎 - Strategy Pattern with Tuple Keys"""

import functools
from typing import Any, Optional

# 分发表：((db_type_tuple), func_name) → handler
_DISPATCH_TABLE: dict[tuple[tuple[str, ...], str], callable] = {}


def register(db_type_spec, func_name: str):
    """注册后端处理函数的装饰器。

    db_type_spec:
      - str: 所有 vregion 通配，如 "ByteRedis" → key (("ByteRedis",), func_name)
      - (str, str): 精确匹配单个 vregion，如 ("ByteRDS", "China-North")
      - (str, set|list): 批量精确匹配，如 ("ByteRDS", {"China-North", "China-BOE"})
    """
    def decorator(fn):
        @functools.wraps(fn)
        def wrapper(*args, **kwargs):
            return fn(*args, **kwargs)
        if isinstance(db_type_spec, str):
            _DISPATCH_TABLE[((db_type_spec,), func_name)] = wrapper
        elif isinstance(db_type_spec, tuple) and len(db_type_spec) == 2:
            itype, vregion_spec = db_type_spec
            if isinstance(vregion_spec, (set, list, frozenset)):
                for vr in vregion_spec:
                    _DISPATCH_TABLE[((itype, vr), func_name)] = wrapper
            else:
                _DISPATCH_TABLE[(db_type_spec, func_name)] = wrapper
        return wrapper
    return decorator


def dispatch(func_name: str, db_type: str, vregion: str, client, params, ctx, **kwargs) -> Optional[dict[str, Any]]:
    """分发到注册的后端。

    匹配优先级：
    1. 精确匹配 (db_type, vregion) + func
    2. 通配匹配 (db_type,) + func
    3. 未注册 → 返回 None（由调用方处理默认逻辑或报错）
    """
    # 1. Exact match (db_type, vregion) + func
    handler = _DISPATCH_TABLE.get(((db_type, vregion), func_name))
    # 2. Wildcard match (db_type,) + func
    if not handler:
        handler = _DISPATCH_TABLE.get(((db_type,), func_name))
    if handler:
        return handler(client, params, ctx, **kwargs)
    return None


def is_registered(db_type: str, func_name: str, vregion: str = "") -> bool:
    """检查某个 (db_type, func_name) 组合是否有注册 handler。"""
    if vregion and ((db_type, vregion), func_name) in _DISPATCH_TABLE:
        return True
    return ((db_type,), func_name) in _DISPATCH_TABLE
