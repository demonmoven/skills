"""ByteRDS Direct Client - 通过 RDS Platform HTTP API 直接调用（参考 bytedcli 源码）

用于不走 DBW 的 vregion（如非字节云平台的直连场景）。
jwt_token 由 ToolboxClient 传入，不自行获取。

设计原则：
- 构造时传入 vregion，内部自动推导 host、region、headers
- 公开方法不暴露 region 参数，调用方无需关心 RDS Platform 的 region 体系
- region 映射参考 bytedcli: defaultRegionForSite() + bytecloudHostForSite()
"""

import json
import sys
import urllib.request
import urllib.error
import urllib.parse
from typing import Any, Dict, List, Optional

from common import _vregion_to_site, _SITE_HOST

# API region → x-bcgw-vregion header 映射（仅列出 bcgw-vregion ≠ region 的例外）
# 大部分 region 的 bcgw-vregion 等于 region 本身，这里只覆盖不一致的
_REGION_BCGW_VREGION = {
    "sinf": "cn-beijing",            # huabei2 的 API region=sinf，但 bcgw-vregion=cn-beijing
    "boe2": "boe",                   # boe site 统一用 boe
    "ChinaSinf-BOE": "boe",          # boe site 统一用 boe
    "China-InfBOE": "boe",           # boe site 统一用 boe
}

# vregion → RDS Platform search API 的 region 参数（默认 vdc 对应的 region）
# 大部分 vdc 的 region = vdc 码本身，少数例外见 _VDC_REGION
_VREGION_TO_REGION = {
    # 国内站 cn — 每个 vregion 取默认 vdc 对应的 region
    "China-North": "cn",
    "ChinaSinf-North": "sinf",
    "China-East": "China-East",       # 默认 vdc=ce，但 ce 的 region=China-East
    "China-Fintech": "sdqd",
    "China-North3": "China-North3",   # 无独立 vdc，region=vregion 名
    "China-North5": "China-North5",
    "China-North6": "China-North6",
    "China-Pay": "China-Pay",
    "China-Pay2": "China-Pay2",
    "China-HKPay": "China-HKPay",
    "China-Aggregation": "agsdqd",
    "China-Enterprise": "lftobiaas",
    "Aliyun_NC2": "galinc2",
    # 国内站 boe
    "China-BOE": "boe",
    "China-BOE2": "boe2",
    "ChinaSinf-BOE": "ChinaSinf-BOE",
    "China-InfBOE": "China-InfBOE",
    # US-BOE 走独立 API，暂不在此映射
    # i18n-bd → vdc（机房）码，每个 vregion 取默认机房
    "Asia-SouthEastBD": "bdsgdt",
    "Europe-WestBD": "bddedt",
    "Singapore-SaaS": "sgsaas1larkidc1",
    "Asia-SaaS": "jpsaas",
    "US-EE": "awsva",
    "Singapore-Common": "sgcomm1",
    "US-EastBD": "useast14a",
    "US-TTP3": "useast15a",
    "Australia-SouthEastBD": "syd2a",
    # i18n-tt → vdc（机房）码，从 cloud.tiktok-row.net 页面抓取
    "Singapore-Central": "sg2",
    "EasternEuro-TT": "ycru",
    "I18N-Game": "awssggm",
    "Europe-Central": "awsfr",
    "US-East": "maliva",
    "US-West": "uswest2",
    "Australia-SouthEast": "syd1a",
}

# vregion → 全部 vdc（机房）列表，默认使用第一个 vdc
_VREGION_TO_VDCS = {
    # 国内站 cn
    "China-North": ["cn", "tjdt"],
    "ChinaSinf-North": ["huabei2", "multicloud"],
    "China-East": ["ce", "pddt"],
    "China-Fintech": ["sdqd", "hkcj", "xjhd"],
    "China-Aggregation": ["agsdqd", "aggdsz", "aghbwh", "agcq", "bjlgy"],
    "China-Enterprise": ["lftobiaas"],
    "Aliyun_NC2": ["galinc2"],
    # 国内站 boe
    "China-BOE": ["boe"],
    "China-BOE2": ["boe2"],
    # i18n-bd
    "Asia-SouthEastBD": ["bdsgdt", "my5a", "nonttap"],
    "Europe-WestBD": ["bddedt", "be2a"],
    "Singapore-SaaS": ["sgsaas1larkidc1"],
    "Asia-SaaS": ["jpsaas"],
    "US-EE": ["awsva"],
    "Singapore-Common": ["sgcomm1"],
    "US-EastBD": ["useast14a"],
    "US-TTP3": ["useast15a"],
    "Australia-SouthEastBD": ["syd2a"],
    # i18n-tt
    "Singapore-Central": ["sg2", "sgdt", "alisg"],
    "EasternEuro-TT": ["ycru"],
    "I18N-Game": ["awssggm"],
    "Europe-Central": ["awsfr", "fr1a"],
    "US-East": ["maliva", "useastdt", "awsvagm"],
    "US-West": ["uswest2"],
    "Australia-SouthEast": ["syd1a"],
}

# vdc → region 映射（仅列出 region ≠ vdc 的例外情况）
# 大部分 vdc 的 region 就是 vdc 码本身，这里只覆盖不一致的
_VDC_REGION = {
    "ce": "China-East",              # China-East 的 ce 机房，region 用 vregion 名
    "multicloud": "ChinaSinf-North", # ChinaSinf-North 的 multicloud 机房，region 用 vregion 名
    "huabei2": "sinf",               # ChinaSinf-North 的 huabei2 机房，API region 是 sinf
}

# 特殊 vdc 的 API 覆盖配置
# 这些 vdc 不走标准的 {host}/api/v1/rds 路径，需要不同的 host/path/region
# 例如 nonttap（Asia-SouthEastBD 的非TT机房）走火山模式的 rds-sinf 接口
_VDC_API_OVERRIDE = {
    "nonttap": {
        "host": "https://cloud-i18n.sinf.net",
        "base_path": "/api/v1/rds-sinf",
        "region": "sinf-my",
    },
}


class ByteRDSClient:
    """ByteRDS 直连客户端 - 通过 RDS Platform API 直接访问。

    构造时传入 vregion，内部自动推导 host / region / headers。
    公开方法不接受 region 参数 —— RDS Platform 的 region 由 vregion 唯一确定，
    调用方不需要（也不应该）传入 DBW 的 region。
    """

    def __init__(self, jwt_token: str, vregion: str):
        self._jwt_token = jwt_token
        self._vregion = vregion
        self._site = _vregion_to_site(vregion)
        self._host = _SITE_HOST.get(self._site, _SITE_HOST["cn"])
        self._base_url = f"{self._host}/api/v1/rds"
        self._region = _VREGION_TO_REGION.get(vregion, "cn")

    def _headers(self, region_override: Optional[str] = None) -> Dict[str, str]:
        """构建请求 headers（参考 bytedcli getRdsHeaders）"""
        region = region_override or self._region
        bcgw_vregion = _REGION_BCGW_VREGION.get(region, region)
        headers = {
            "Accept": "application/json, text/plain, */*",
            "Accept-Language": "zh",
            "x-jwt-token": self._jwt_token,
            "Origin": self._host,
            "Referer": f"{self._host}/",
            "x-bcgw-vregion": bcgw_vregion,
            "x-bcgw-region": region,
        }
        # prod / boe / boei18n / i18n-bd 需要 tenant-id
        if self._site in ("cn", "boe", "boei18n", "i18n-bd"):
            headers["x-bcgw-tenant-id"] = "bytedance"
        return headers

    def _get(self, url: str, timeout: int = 60, region_override: Optional[str] = None) -> Any:
        """GET 请求"""
        headers = self._headers(region_override)
        req = urllib.request.Request(url, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                data = json.loads(resp.read().decode())
        except urllib.error.HTTPError as e:
            body = e.read().decode()
            if e.code == 401:
                raise ValueError(f"鉴权失败(HTTP 401)，请重新登录 AI PaaS CLI")
            if e.code == 403:
                raise ValueError(f"权限不足(HTTP 403)")
            raise ValueError(f"RDS API GET 失败 (HTTP {e.code}): {body[:300]}")
        self._ensure_ok(url, data)
        return data.get("data")

    def _post(self, url: str, body: Any, timeout: int = 60, region_override: Optional[str] = None) -> Any:
        """POST 请求"""
        headers = {**self._headers(region_override), "Content-Type": "application/json"}
        data_bytes = json.dumps(body).encode()
        req = urllib.request.Request(url, data=data_bytes, headers=headers, method="POST")
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                data = json.loads(resp.read().decode())
        except urllib.error.HTTPError as e:
            body_str = e.read().decode()
            if e.code == 401:
                raise ValueError(f"鉴权失败(HTTP 401)，请重新登录 AI PaaS CLI")
            if e.code == 403:
                raise ValueError(f"权限不足(HTTP 403)")
            raise ValueError(f"RDS API POST 失败 (HTTP {e.code}): {body_str[:300]}")
        self._ensure_ok(url, data)
        return data.get("data")

    def _ensure_ok(self, url: str, data: Any):
        """检查 RDS API 响应码"""
        if not isinstance(data, dict):
            raise ValueError(f"RDS API 响应格式异常: {str(data)[:200]}")
        code = data.get("code", 0)
        msg = data.get("msg", "")
        if code != 0:
            raise ValueError(f"RDS API 错误 (code={code}): {msg}")

    # ── 内部辅助 ──

    def _resolve_vdc_params(self, vdc: str) -> tuple:
        """根据 vdc 返回 (base_url, region)，不修改实例状态。"""
        override = _VDC_API_OVERRIDE.get(vdc)
        if override:
            return f"{override['host']}{override['base_path']}", override["region"]
        region = _VDC_REGION.get(vdc, vdc)
        return f"{self._host}/api/v1/rds", region

    # ── 公开方法 ──

    def set_region(self, vdc: str):
        """切换当前使用的 vdc（机房），后续请求将使用新 vdc。

        对于特殊 vdc（如 nonttap），会同时切换 API base_url 和 region。
        """
        self._base_url, self._region = self._resolve_vdc_params(vdc)

    def search_databases(
        self,
        keyword: str,
        page: int = 1,
        size: int = 20,
    ) -> List[Dict[str, Any]]:
        """搜索数据库列表（仅在当前 vdc 中搜索）

        API: GET /api/v1/rds/api/v2/search/?region=&keyword=&no=&size=
        返回: RdsDatabase[] 数组
        """
        params = urllib.parse.urlencode({
            "region": self._region,
            "keyword": keyword,
            "no": page,
            "size": size,
        })
        url = f"{self._base_url}/api/v2/search/?{params}"
        return self._get(url)

    def _search_databases_in_vdc(
        self,
        vdc: str,
        keyword: str,
        page: int = 1,
        size: int = 20,
    ) -> List[Dict[str, Any]]:
        """在指定 vdc 中搜索数据库，不修改实例状态。"""
        base_url, region = self._resolve_vdc_params(vdc)
        params = urllib.parse.urlencode({
            "region": region,
            "keyword": keyword,
            "no": page,
            "size": size,
        })
        url = f"{base_url}/api/v2/search/?{params}"
        return self._get(url, region_override=region)

    def search_databases_all_vdcs(
        self,
        keyword: str,
        page: int = 1,
        size: int = 20,
    ) -> List[Dict[str, Any]]:
        """在当前 vregion 的所有 vdc 中搜索数据库，合并去重。

        每条结果附带 _vdc 字段标识来源机房。
        仅有一个 vdc 时等同于 search_databases。
        """
        vdcs = _VREGION_TO_VDCS.get(self._vregion, [self._region])
        if len(vdcs) == 1:
            vdc = vdcs[0]
            self.set_region(vdc)
            results = self.search_databases(keyword, page, size)
            if isinstance(results, list):
                for db in results:
                    db["_vdc"] = vdc
            return results

        seen: set[tuple] = set()
        merged: List[Dict[str, Any]] = []
        for vdc in vdcs:
            try:
                results = self._search_databases_in_vdc(vdc, keyword, page, size)
            except Exception as e:
                print(f"[warn] search_databases in vdc {vdc} failed: {e}", file=sys.stderr)
                continue
            if isinstance(results, list):
                for db in results:
                    name = db.get("dbname") or db.get("db_name") or ""
                    key = (name, vdc)
                    if name and key not in seen:
                        seen.add(key)
                        db["_vdc"] = vdc
                        merged.append(db)
        return merged

    def list_tables(
        self,
        dbname: str,
    ) -> List[Dict[str, Any]]:
        """获取数据库的表列表

        API: GET /api/v1/rds/api/v2/dbs/{dbname}/tables/?region=&all=1
        返回: RdsTable[] 数组 [{name, comment, mtime}, ...]
        """
        params = urllib.parse.urlencode({
            "region": self._region,
            "all": "1",
        })
        url = f"{self._base_url}/api/v2/dbs/{dbname}/tables/?{params}"
        return self._get(url)

    def get_table_schema(
        self,
        dbname: str,
        table: str,
    ) -> Dict[str, Any]:
        """获取表结构

        API: GET /api/v1/rds/api/v2/dbs/{dbname}/tables/{table}/?region=
        返回: 表结构信息（字段、索引等）
        """
        params = urllib.parse.urlencode({"region": self._region})
        url = f"{self._base_url}/api/v2/dbs/{dbname}/tables/{table}/?{params}"
        return self._get(url)

    def _resolve_dbpsm(self, dbname: str) -> str:
        """查询数据库 PSM 列表，优先匹配 _read 后缀，fallback 为第一个 PSM。"""
        try:
            results = self.search_databases(keyword=dbname, page=1, size=5)
            if isinstance(results, list):
                for db in results:
                    if (db.get("dbname") or db.get("db_name")) == dbname:
                        psms = db.get("psms") or []
                        if psms:
                            read_psms = [p for p in psms if p.endswith("_read")]
                            return read_psms[0] if read_psms else psms[0]
        except Exception:
            pass
        return f"toutiao.mysql.{dbname}_read"

    def run_sql(
        self,
        dbname: str,
        sql: str,
        dbpsm: Optional[str] = None,
    ) -> Dict[str, Any]:
        """执行 SQL 查询

        API: POST /api/v1/rds/api/v2/dbs/{dbname}/run_sql/?region=
        Body: {dql: sql, dbpsm: ...}
        返回: {amount, data: [{...}], row_order: [...], cost_time_ms}
        """
        _dbpsm = dbpsm or self._resolve_dbpsm(dbname)
        params = urllib.parse.urlencode({"region": self._region})
        url = f"{self._base_url}/api/v2/dbs/{dbname}/run_sql/?{params}"
        body = {
            "dql": sql,
            "dbpsm": _dbpsm,
        }
        return self._post(url, body)

    def slow_log(
        self,
        dbname: str,
        start: int,
        end: int,
        action: Optional[str] = None,
        table: Optional[str] = None,
        query_time: float = 0.1,
    ) -> Any:
        """查询慢日志

        API: GET /api/v1/rds/api/v2/dbs/{dbname}/slow_log/?region=&start=&end=&action=&table=&query_time=
        参数:
          - action: 选填，枚举 select/insert/update/delete，不传则全部
          - table: 选填，按表名过滤
          - query_time: 最小查询时间阈值（秒），默认 0.1
        """
        query = {
            "region": self._region,
            "start": start,
            "end": end,
            "query_time": query_time,
        }
        if action:
            query["action"] = action
        if table:
            query["table"] = table
        params = urllib.parse.urlencode(query)
        url = f"{self._base_url}/api/v2/dbs/{dbname}/slow_log/?{params}"
        return self._get(url)

    def topo(self, dbname: str, need_detail: int = 0) -> Any:
        """查询实例拓扑（主从节点列表）

        API: GET /api/v1/rds/platform/v1/topo/{dbname}/?region=&need_detail=
        返回: [{ip, port, role, slaves: [...], ...}]
        """
        query = urllib.parse.urlencode({
            "region": self._region,
            "need_detail": need_detail,
        })
        url = f"{self._base_url}/platform/v1/topo/{dbname}/?{query}"
        return self._get(url)

    def engine_status(self, dbname: str, ip: str, port: int) -> Any:
        """查询 InnoDB 引擎状态（等价于 SHOW ENGINE INNODB STATUS）

        API: GET /api/v1/rds/api/v2/dbs/{dbname}/instances/{ip}/engine/?region=&port=
        返回: {Type, Name, Status} — Status 字段为完整 InnoDB Monitor Output
        """
        query = urllib.parse.urlencode({
            "region": self._region,
            "port": port,
        })
        url = f"{self._base_url}/api/v2/dbs/{dbname}/instances/{ip}/engine/?{query}"
        return self._get(url)

    def trigger_diagnostic(
        self,
        dbname: str,
        start_ts: int,
        end_ts: int,
        service: str = "all",
    ) -> Any:
        """触发健康检查

        API: POST /api/v1/rds/api/v2/dbs/{dbname}/diagnostic/record/?region=
        Body: {"service": "all", "start_ts": ..., "end_ts": ...}
        """
        params = urllib.parse.urlencode({"region": self._region})
        url = f"{self._base_url}/api/v2/dbs/{dbname}/diagnostic/record/?{params}"
        return self._post(url, {"service": service, "start_ts": start_ts, "end_ts": end_ts})

    def query_diagnostic_records(
        self,
        dbname: str,
        page: int = 0,
        size: int = 6,
    ) -> Any:
        """查询健康检查记录

        API: GET /api/v1/rds/api/v2/dbs/{dbname}/diagnostic/query_record/?region=&page=&size=
        """
        params = urllib.parse.urlencode({
            "region": self._region,
            "page": page,
            "size": size,
        })
        url = f"{self._base_url}/api/v2/dbs/{dbname}/diagnostic/query_record/?{params}"
        return self._get(url)

    def query_diagnostic_report(
        self,
        dbname: str,
        record_id: str,
    ) -> Any:
        """查询健康检查详细报告

        API: GET /api/v1/rds/api/v2/dbs/{dbname}/diagnostic/query_report/?record_id=&db_name=&region=
        """
        params = urllib.parse.urlencode({
            "record_id": record_id,
            "db_name": dbname,
            "region": self._region,
        })
        url = f"{self._base_url}/api/v2/dbs/{dbname}/diagnostic/query_report/?{params}"
        return self._get(url)
