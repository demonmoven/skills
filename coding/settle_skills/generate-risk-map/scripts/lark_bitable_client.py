"""
飞书多维表格 API 客户端模块

支持两种认证方式：
1. Aime 平台内置 JWT（通过环境变量 AIME_USER_CLOUD_JWT）
2. 飞书应用凭证 / 用户访问令牌（独立部署）

使用方式：
    from lark_bitable_client import LarkBitableClient
    client = LarkBitableClient()
    client.batch_add_records(app_token, table_id, records)
"""

import os
import sys
import json
import time
import urllib.request
import urllib.error
from typing import Any, Dict, List, Optional


class LarkBitableClient:
    """飞书多维表格 API 客户端"""

    BASE_URL = "https://open.feishu.cn/open-apis"

    def __init__(self):
        """初始化客户端，自动选择认证方式"""
        self.token = self._resolve_token()

    def _resolve_token(self) -> str:
        """解析访问令牌，优先级：AIME JWT > 用户令牌 > 应用令牌"""
        # 1. Aime 平台 JWT
        jwt = os.environ.get("AIME_USER_CLOUD_JWT")
        if jwt:
            return jwt

        # 2. 用户访问令牌
        user_token = os.environ.get("LARK_USER_ACCESS_TOKEN")
        if user_token:
            return user_token

        # 3. 应用凭证获取 tenant_access_token
        app_id = os.environ.get("LARK_APP_ID")
        app_secret = os.environ.get("LARK_APP_SECRET")
        if app_id and app_secret:
            return self._get_tenant_token(app_id, app_secret)

        raise RuntimeError(
            "未找到有效的飞书认证凭证。请设置以下环境变量之一：\n"
            "  - AIME_USER_CLOUD_JWT (Aime平台内自动注入)\n"
            "  - LARK_USER_ACCESS_TOKEN (用户访问令牌)\n"
            "  - LARK_APP_ID + LARK_APP_SECRET (应用凭证)"
        )

    def _get_tenant_token(self, app_id: str, app_secret: str) -> str:
        """通过应用凭证获取 tenant_access_token"""
        url = f"{self.BASE_URL}/auth/v3/tenant_access_token/internal"
        data = json.dumps({"app_id": app_id, "app_secret": app_secret}).encode()
        req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(req) as resp:
            result = json.loads(resp.read().decode())
        if result.get("code") != 0:
            raise RuntimeError(f"获取 tenant_access_token 失败: {result}")
        return result["tenant_access_token"]

    def _request(self, method: str, path: str, body: Optional[Dict] = None, retries: int = 3) -> Dict[str, Any]:
        """发送 HTTP 请求到飞书 API"""
        url = f"{self.BASE_URL}{path}"
        headers = {
            "Authorization": f"Bearer {self.token}",
            "Content-Type": "application/json; charset=utf-8",
        }

        data = json.dumps(body, ensure_ascii=False).encode("utf-8") if body else None

        for attempt in range(retries):
            try:
                req = urllib.request.Request(url, data=data, headers=headers, method=method)
                with urllib.request.urlopen(req, timeout=30) as resp:
                    result = json.loads(resp.read().decode("utf-8"))
                    if result.get("code") == 1254607:
                        # Rate limit - retry
                        time.sleep(2 ** attempt)
                        continue
                    return result
            except urllib.error.HTTPError as e:
                if e.code == 429:
                    time.sleep(2 ** attempt)
                    continue
                error_body = e.read().decode() if e.fp else str(e)
                raise RuntimeError(f"HTTP {e.code}: {error_body}")
            except urllib.error.URLError as e:
                if attempt == retries - 1:
                    raise RuntimeError(f"网络错误: {e}")
                time.sleep(2 ** attempt)

        raise RuntimeError(f"请求失败，已重试 {retries} 次")

    # ===== 多维表格操作 =====

    def create_app(self, name: str, folder_token: Optional[str] = None) -> Dict[str, Any]:
        """创建多维表格应用"""
        body: Dict[str, Any] = {"name": name}
        if folder_token:
            body["folder_token"] = folder_token
        return self._request("POST", "/bitable/v1/apps", body)

    def create_table(self, app_token: str, name: str, fields: List[Dict[str, Any]]) -> Dict[str, Any]:
        """创建数据表"""
        body = {
            "table": {
                "name": name,
                "default_view_name": "默认视图",
                "fields": fields,
            }
        }
        return self._request("POST", f"/bitable/v1/apps/{app_token}/tables", body)

    def batch_add_records(self, app_token: str, table_id: str, records: List[Dict[str, Any]]) -> Dict[str, Any]:
        """批量新增记录"""
        body = {"records": [{"fields": r["fields"]} for r in records]}
        return self._request("POST", f"/bitable/v1/apps/{app_token}/tables/{table_id}/records/batch_create", body)

    def batch_delete_records(self, app_token: str, table_id: str, record_ids: List[str]) -> Dict[str, Any]:
        """批量删除记录"""
        body = {"records": record_ids}
        return self._request("POST", f"/bitable/v1/apps/{app_token}/tables/{table_id}/records/batch_delete", body)

    def search_records(
        self,
        app_token: str,
        table_id: str,
        filter_obj: Optional[Dict] = None,
        sort: Optional[List[Dict]] = None,
        page_token: str = "",
        page_size: int = 500,
    ) -> Dict[str, Any]:
        """搜索记录"""
        body: Dict[str, Any] = {"page_size": page_size}
        if filter_obj:
            body["filter"] = filter_obj
        if sort:
            body["sort"] = sort
        if page_token:
            body["page_token"] = page_token
        return self._request("POST", f"/bitable/v1/apps/{app_token}/tables/{table_id}/records/search", body)

    def list_fields(self, app_token: str, table_id: str, page_token: str = "", page_size: int = 100) -> Dict[str, Any]:
        """获取字段列表"""
        params = f"?page_size={page_size}"
        if page_token:
            params += f"&page_token={page_token}"
        return self._request("GET", f"/bitable/v1/apps/{app_token}/tables/{table_id}/fields{params}")
