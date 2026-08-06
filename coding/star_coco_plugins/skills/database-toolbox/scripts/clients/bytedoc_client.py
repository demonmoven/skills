"""ByteDoc Client - ByteDoc Cloud API（Cloud Native Mongo + Classic 原生接口）"""

import json
import urllib.request
import urllib.error
import urllib.parse
from typing import Any, Dict, Optional

from common import _vregion_to_site, _SITE_HOST


class ByteDocClient:
    def __init__(self, jwt_token: str, vregion: str = "China-BOE"):
        self._site = _vregion_to_site(vregion)
        self._jwt_token = jwt_token
        host = _SITE_HOST.get(self._site, _SITE_HOST["cn"])
        # Cloud Native Mongo API
        self._base_url = f"{host}/api/v1/bytedoc_cloud/services"
        # Classic ByteDoc API
        self._classic_base_url = f"{host}/api/v1/bytedoc_cloud/bytedoc/api/v1"
        # Capacity Manager API
        self._capacity_base_url = f"{host}/api/v1/bytedoc_cloud/capacity_manager/v1"

    def _headers(self) -> Dict[str, str]:
        return {
            "Accept": "application/json, text/plain, */*",
            "Accept-Language": "zh",
            "x-jwt-token": self._jwt_token,
            "x-bcgw-tenant-id": "bytedance",
            "x-bcgw-vregion": self._site.upper(),
        }

    def _do_request(self, url: str, method: str = "GET",
                    body: Optional[Dict[str, Any]] = None,
                    timeout: int = 60, label: str = "ByteDoc API",
                    extra_headers: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
        """通用 HTTP 请求"""
        headers = self._headers()
        if extra_headers:
            headers.update(extra_headers)
        data = None
        if body is not None:
            headers["Content-Type"] = "application/json"
            data = json.dumps(body).encode()
        req = urllib.request.Request(url, headers=headers, data=data, method=method)
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                return json.loads(resp.read().decode())
        except urllib.error.HTTPError as e:
            err_body = e.read().decode()
            if e.code == 401:
                raise ValueError("鉴权失败(HTTP 401)，请重新登录 AI PaaS CLI")
            if e.code == 403:
                raise ValueError("权限不足(HTTP 403)")
            raise ValueError(f"{label} 失败 (HTTP {e.code}): {err_body[:300]}")

    def _request(self, action: str, params: Dict[str, str],
                  method: str = "GET", body: Optional[Dict[str, Any]] = None,
                  timeout: int = 60) -> Dict[str, Any]:
        """发送 HTTP 请求到 ByteDoc Cloud Native API"""
        qs = urllib.parse.urlencode({k: v for k, v in params.items() if v is not None})
        url = f"{self._base_url}/{action}?{qs}"
        return self._do_request(url, method=method, body=body, timeout=timeout,
                                label=f"ByteDoc API {action}")

    def _classic_request(self, database: str, endpoint: str,
                          params: Optional[Dict[str, str]] = None,
                          method: str = "GET",
                          body: Optional[Dict[str, Any]] = None,
                          timeout: int = 60) -> Dict[str, Any]:
        """发送 HTTP 请求到 ByteDoc Classic API"""
        url = f"{self._classic_base_url}/database/{database}/{endpoint}/"
        if params:
            qs = urllib.parse.urlencode({k: v for k, v in params.items() if v is not None})
            url = f"{url}?{qs}"
        return self._do_request(url, method=method, body=body, timeout=timeout,
                                label=f"ByteDoc Classic {endpoint}")

    # ── Cloud Native Mongo API ──

    def describe_slow_logs(
        self,
        instance_id: str,
        pod_name: str,
        region: str,
        psm: str,
        start_time: str,
        end_time: str,
        limit: int = 10,
        sort: str = "DESC",
    ) -> Dict[str, Any]:
        """查询 ByteDoc 慢日志明细"""
        return self._request("DescribeSlowLogs", {
            "instanceId": instance_id,
            "podName": pod_name,
            "region": region,
            "psm": psm,
            "queryStartTime": start_time,
            "queryEndTime": end_time,
            "limit": str(limit),
            "sort": sort,
        })

    def instance_collection_info(
        self,
        instance_id: str,
        database: str,
        psm: str,
        region: str,
    ) -> Dict[str, Any]:
        """查询 Mongo 实例的 collection 磁盘详情"""
        return self._request(
            "InstanceCollectionInfo",
            params={"psm": psm, "region": region},
            method="POST",
            body={"instance_id": instance_id, "database": database},
        )

    # ── Classic ByteDoc API ──

    def classic_usage_info(self, database: str) -> Dict[str, Any]:
        """获取 Classic ByteDoc 数据库总览（含 cluster_name = cluster_id）"""
        return self._classic_request(database, "usage_info")

    def classic_static_info(self, database: str) -> Dict[str, Any]:
        """获取 Classic ByteDoc 数据库静态信息"""
        return self._classic_request(database, "static_info")

    def classic_get_collections(self, database: str) -> Dict[str, Any]:
        """列出 Classic ByteDoc 数据库的所有 collection"""
        return self._classic_request(database, "get_collections")

    def classic_get_collection_indexes(self, database: str, collection: str) -> Dict[str, Any]:
        """获取 Classic ByteDoc collection 的索引信息"""
        return self._classic_request(database, "get_collection_indexes",
                                      params={"collection": collection})

    def classic_web_query(self, database: str, collection: str, query: str) -> Dict[str, Any]:
        """执行 Classic ByteDoc 查询"""
        return self._classic_request(database, "web_query",
                                      method="POST",
                                      body={"collection": collection, "query": query})

    def classic_slow_query_overview(self, database: str,
                                     start_ts: int, end_ts: int,
                                     millis: int = 100) -> Dict[str, Any]:
        """查询 Classic ByteDoc 慢日志聚合"""
        return self._classic_request(database, "slow_query_overview",
                                      params={"start_ts": str(start_ts),
                                              "end_ts": str(end_ts),
                                              "millis": str(millis)})

    # Capacity Manager API 网关要求 x-bcgw-vregion 带 Native 后缀
    _CAPACITY_VREGION = {"cn": "CNNative", "boe": "BOENative"}

    def classic_capacity(self, cluster_id: str, database: str) -> Dict[str, Any]:
        """获取 Classic ByteDoc collection 磁盘容量信息"""
        url = (f"{self._capacity_base_url}/cluster_collection_info"
               f"/clusters/{cluster_id}/databases/{database}")
        vregion = self._CAPACITY_VREGION.get(self._site, self._site.upper())
        return self._do_request(url, label="ByteDoc Capacity API",
                                extra_headers={"x-bcgw-vregion": vregion})
