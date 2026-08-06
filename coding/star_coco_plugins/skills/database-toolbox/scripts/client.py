"""ToolboxClient + create_client 工厂"""

import sys
from typing import Optional

from clients.dbw_client import DBWClient
from common import (
    _load_global_config, _resolve_vregion, _vregion_to_site,
    _SKILL_HEADERS,
)
from jwt_utils import get_user_jwt_token


def _get_jwt_token(site: Optional[str] = None) -> Optional[str]:
    try:
        token = get_user_jwt_token(site)
        if token:
            return token
    except Exception as e:
        print(f"[warn] get_jwt_token failed: {e}", file=sys.stderr)
    return None


class ToolboxClient:
    """复合客户端。__getattr__ 透传到 DBWClient，.redis/.bytedoc/.byterds 懒加载。"""

    def __init__(self, jwt_token, vregion, extra_headers):
        self.jwt_token = jwt_token
        self.vregion = vregion
        self.extra_headers = extra_headers
        self._dbw = DBWClient(jwt_token, vregion,
                              extra_headers=extra_headers)
        self._redis = None
        self._bytedoc = None
        self._byterds = None

    @property
    def redis(self):
        if self._redis is None:
            from clients.redis_client import RedisClient
            self._redis = RedisClient(
                jwt_token=self.jwt_token,
                vregion=self.vregion,
            )
        return self._redis

    @property
    def bytedoc(self):
        if self._bytedoc is None:
            from clients.bytedoc_client import ByteDocClient
            self._bytedoc = ByteDocClient(
                jwt_token=self.jwt_token,
                vregion=self.vregion,
            )
        return self._bytedoc

    @property
    def byterds(self):
        if self._byterds is None:
            from clients.byterds_client import ByteRDSClient
            self._byterds = ByteRDSClient(
                jwt_token=self.jwt_token,
                vregion=self.vregion,
            )
        return self._byterds

    def __getattr__(self, name):
        # 设计意图：DBW 是默认后端，未在 ToolboxClient 上定义的属性（如 describe_instances、
        # execute_sql 等 DBW API 方法）自动透传到 DBWClient。
        # 副作用：拼写错误不会立即报 AttributeError，而是尝试在 DBWClient 上查找。
        return getattr(self._dbw, name)


def create_client(
    vregion: Optional[str] = None,
) -> ToolboxClient:
    """创建 ToolboxClient，自动获取 JWT token。

    Args:
        vregion: 虚拟地域（如 China-North, China-East, China-BOE 等）。
                 也支持传入 RegionId（如 cn, boe, sdqd），会自动映射为对应的 vregion。
                 不传则从 .config.json 读 default_vregion。
    """
    global_cfg = _load_global_config()
    _vregion = _resolve_vregion(vregion) if vregion else None
    _vregion = _vregion or global_cfg.get("default_vregion")
    if not _vregion:
        raise ValueError("全局配置没有 default_vregion，缺少 vregion 信息，向用户询问")

    _site = _vregion_to_site(_vregion)
    jwt_token = _get_jwt_token(site=_site)

    return ToolboxClient(
        jwt_token=jwt_token,
        vregion=_vregion,
        extra_headers=_SKILL_HEADERS,
    )
