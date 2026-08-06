"""
ByteCloud DBW Client - JWT Token 鉴权
"""

import json
import re
import urllib.request
import urllib.error
import urllib.parse
from typing import Any, Dict, Optional

from common import _VREGION_TO_SITE, _SITE_HOST

# site → 域名（从 _SITE_HOST 提取，去掉 https:// 前缀）
_SITE_DOMAINS = {k: v.replace("https://", "") for k, v in _SITE_HOST.items()}


class DBWClient:
    def __init__(
        self,
        jwt_token: str,
        vregion: str,
        cookies: Optional[str] = None,
        extra_headers: Optional[Dict[str, str]] = None,
    ):
        self.jwt_token = jwt_token
        self.vregion = vregion
        self._site = _VREGION_TO_SITE.get(vregion, "cn")
        # 根据 site 自动选择站点域名
        domain = _SITE_DOMAINS.get(self._site, "cloud-boe.bytedance.net")
        self.api_url = f"https://{domain}/api/v1/dbw/api"
        self.action_url = f"https://{domain}/api/v1/dbw/action"
        self.cookies = cookies or ""
        self.extra_headers = extra_headers or {}

    @property
    def site(self) -> str:
        return self._site

    def _call_api(self, action: str, body_args: Dict[str, Any], extra_headers: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
        """发起 API 请求，使用 JWT token 鉴权"""
        query_params = {
            "Action": action,
            "Version": "2018-01-01",
            "Region": self._site,
            "Service": "dbw",
        }
        query_string = urllib.parse.urlencode(query_params)
        url = f"{self.api_url}/?{query_string}"

        body_dict = {k: v for k, v in body_args.items() if v is not None}
        body = json.dumps(body_dict)

        headers = {
            "Content-Type": "application/json; charset=UTF-8",
            "Accept": "application/json, text/plain, */*",
            "x-jwt-token": self.jwt_token,
            "x-top-region": self._site,
            "x-storage-language": "zh",
        }
        # 自动从请求体提取 instance-id 和 ds-type header
        # server 端 parseInstance 支持多种字段名：FollowInstanceID > InstanceID > InstanceId > Filters[].InstanceId
        follow_id = (body_args.get("FollowInstanceID", "")
                     or body_args.get("InstanceID", "")
                     or body_args.get("InstanceId", ""))
        instance_id = follow_id.split(".")[0] if "." in follow_id else follow_id
        if instance_id:
            headers["Instance-Id"] = instance_id
        # server 端 parseInstance 支持：InstanceType (DSType enum) > DSType
        ds_type = body_args.get("InstanceType", "") or body_args.get("DSType", "")
        if ds_type:
            headers["Ds-Type"] = ds_type
        # 合并额外 headers
        headers.update(self.extra_headers)
        if extra_headers:
            headers.update(extra_headers)

        # 添加 cookies（如果有）
        if self.cookies:
            headers["Cookie"] = self.cookies

        req = urllib.request.Request(url, data=body.encode("utf-8"), headers=headers, method="POST")
        request_id = None
        try:
            with urllib.request.urlopen(req, timeout=60) as resp:
                # 优先从响应头获取 RequestId，fallback 到 X-Tt-Logid
                request_id = resp.headers.get("X-Top-Request-Id") or resp.headers.get("X-Tt-Logid")
                result = json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            error_body = e.read().decode("utf-8")
            # 从错误响应中提取 RequestId（header 和 body 都尝试）
            request_id = e.headers.get("X-Top-Request-Id") or e.headers.get("X-Tt-Logid")
            err_meta = {}
            try:
                err_json = json.loads(error_body)
                err_meta = err_json.get("ResponseMetadata", {})
                if not request_id:
                    request_id = err_meta.get("RequestId")
            except (json.JSONDecodeError, AttributeError):
                pass
            rid_suffix = f" (RequestId: {request_id})" if request_id else ""
            error_code = err_meta.get("Error", {}).get("Code", "") if isinstance(err_meta.get("Error"), dict) else ""
            if e.code == 401:
                raise ValueError(f"鉴权失败(HTTP 401)，请重新登录 AI PaaS CLI{rid_suffix}")
            if e.code == 403 or error_code in ("Forbidden", "AccessDenied"):
                raise ValueError(f"权限不足，你没有该数据库的操作权限。(HTTP {e.code}, {error_code}){rid_suffix}")
            if error_code == "CreateSessionError":
                raise ValueError(f"无法连接数据库，可能原因：权限不足或数据库不存在。(HTTP {e.code}, {error_code}){rid_suffix}")
            if e.code in (502, 503, 504) or error_code == "InternalError":
                raise ValueError(f"服务端错误(HTTP {e.code})，可能原因：参数错误、查询数据量太大、权限不足、数据库不存在或服务暂时不可用。{rid_suffix}")
            raise ValueError(f"HTTP {e.code}: {error_body}{rid_suffix}")

        # 处理响应：从 body 补充 RequestId（header 优先）
        if "ResponseMetadata" in result:
            meta = result["ResponseMetadata"]
            if not request_id:
                request_id = meta.get("RequestId")
            if meta.get("Error"):
                raise ValueError(f"API Error: {meta['Error']} (RequestId: {request_id})")

        def _wrap_result(res):
            """确保返回 dict 并附带 _request_id"""
            if not isinstance(res, dict):
                res = {"_raw": res}
            if request_id:
                res["_request_id"] = request_id
            return res

        if "Result" in result:
            res = result["Result"]
            if isinstance(res, dict):
                return _wrap_result(res)
            return _wrap_result(res)
        elif "ResponseMetadata" in result:
            meta = result["ResponseMetadata"]
            if meta.get("Error"):
                raise ValueError(f"API Error: {meta['Error']}")

        return _wrap_result(result)

    def _pascal_case(self, s: str) -> str:
        components = s.split("_")
        return "".join(x.title() for x in components)

    def _convert_pascal_to_snake(self, data: Any) -> Any:
        if isinstance(data, dict):
            converted = {}
            for key, value in data.items():
                snake_key = re.sub(r"([A-Z])", r"_\1", key).lower().lstrip("_")
                converted[snake_key] = self._convert_pascal_to_snake(value)
            return converted
        elif isinstance(data, list):
            return [self._convert_pascal_to_snake(item) for item in data]
        return data

    def _call_stream_api(self, action: str, body_args: Dict[str, Any]) -> Dict[str, Any]:
        """发起流式 API 请求（SSE），用于 copilot 类接口"""
        query_params = {
            "Action": action,
            "Version": "2018-01-01",
            "Region": self._site,
            "Service": "dbw",
        }
        query_string = urllib.parse.urlencode(query_params)
        url = f"{self.action_url}/?{query_string}"

        # 不做 PascalCase 转换，保持原始 key
        body = json.dumps({k: v for k, v in body_args.items() if v is not None})

        headers = {
            "Content-Type": "application/json; charset=UTF-8",
            "Accept": "text/event-stream,application/json",
            "x-jwt-token": self.jwt_token,
            "x-top-region": self._site,
            "x-storage-language": "zh",
            "x-dbw-stream": "true",
        }
        # 自动从请求体提取 instance-id 和 ds-type header（与 _call_api 一致）
        follow_id = (body_args.get("FollowInstanceID", "")
                     or body_args.get("InstanceID", "")
                     or body_args.get("InstanceId", ""))
        instance_id = follow_id.split(".")[0] if "." in follow_id else follow_id
        if instance_id:
            headers["Instance-Id"] = instance_id
        ds_type = body_args.get("InstanceType", "") or body_args.get("DSType", "")
        if ds_type:
            headers["Ds-Type"] = ds_type
        headers.update(self.extra_headers)
        if self.cookies:
            headers["Cookie"] = self.cookies

        # urllib 默认不跟随 POST 307，需自定义 handler
        class _PostRedirect(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, req, fp, code, msg, hdrs, newurl):
                if code in (307, 308):
                    return urllib.request.Request(newurl, data=req.data,
                        headers=dict(req.header_items()), method="POST")
                return super().redirect_request(req, fp, code, msg, hdrs, newurl)

        opener = urllib.request.build_opener(_PostRedirect)
        req = urllib.request.Request(url, data=body.encode("utf-8"), headers=headers, method="POST")
        try:
            with opener.open(req, timeout=60) as resp:
                raw = resp.read().decode("utf-8")
        except urllib.error.HTTPError as e:
            error_body = e.read().decode("utf-8")
            request_id = e.headers.get("X-Top-Request-Id") or e.headers.get("X-Tt-Logid")
            rid_suffix = f" (RequestId: {request_id})" if request_id else ""
            if e.code == 401:
                raise ValueError(f"鉴权失败(HTTP 401)，请重新登录 AI PaaS CLI{rid_suffix}")
            if e.code == 403:
                raise ValueError(f"权限不足，你没有该数据库的操作权限。(HTTP 403){rid_suffix}")
            raise ValueError(f"HTTP {e.code}: {error_body}{rid_suffix}")

        # 优先尝试普通 JSON
        try:
            result = json.loads(raw)
            if "ResponseMetadata" in result and result["ResponseMetadata"].get("Error"):
                meta = result.get("ResponseMetadata") or {}
                rid = meta.get("RequestId")
                rid_suffix = f" (RequestId: {rid})" if rid else ""
                raise ValueError(f"API Error: {meta.get('Error')}{rid_suffix}")
            return result.get("Result", result)
        except json.JSONDecodeError:
            pass

        # fallback: 解析 SSE data: 行，合并所有事件
        events = []
        for line in raw.splitlines():
            if line.startswith("data:"):
                data_str = line[len("data:"):].strip()
                if not data_str:
                    continue
                try:
                    events.append(json.loads(data_str))
                except json.JSONDecodeError:
                    continue
        if events:
            # 合并所有事件的 key
            merged = {}
            for e in events:
                merged.update(e)
            return merged

        raise ValueError(f"无法解析响应: {raw[:300]}")

    # === API 方法 ===

    def describe_instances(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeInstances", args)

    def nl2sql(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """调用 GenerateSQLFromNL（/action 端点，需要 x-dbw-stream 头，SSE 流式响应）"""
        return self._call_stream_api("GenerateSQLFromNL", args)
    
    def execute_sql(self, args: Dict[str, Any]) -> Dict[str, Any]:
        result = self._call_api("ExecuteSQL", args)
        return result

    def list_tables(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("ListTables", args)

    def get_table_info(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("GetTableInfo", args)

    def create_dml_sql_change_ticket(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("CreateDmlSqlChangeTicket", args)

    def create_ddl_sql_change_ticket(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("CreateDdlSqlChangeTicket", args)

    def describe_tickets(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeTickets", args)

    def describe_ticket_detail(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeTicketDetail", args)

    def describe_workflow(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeWorkflow", args)

    def describe_slow_logs(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeSlowLogs", args)

    def describe_aggregate_slow_logs(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeAggregateSlowLogs", args)

    def describe_slow_log_time_series_stats(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeSlowLogTimeSeriesStats", args)

    def describe_full_sql_status(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeFullSqlStatus", args)

    def describe_full_sql_detail(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeFullSQLDetail", args)

    def describe_aggregation_sql_table(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeAggregationSQLTable", args)

    def list_slow_query_advice(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("ListSlowQueryAdvice", args)

    def slow_query_advice_task_history(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("SlowQueryAdviceTaskHistory", args)

    def get_metric_data(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("GetMetricData", args)

    def get_metric_items(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("GetMetricItems", args)

    def describe_table_metric(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeTableMetric", args)

    def get_metric_data_predict(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("GetMetricDataPredict", args)

    def kill_process(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("KillProcess", args)

    def describe_dialog_infos(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeDialogInfos", args)

    def describe_deadlock(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeDeadlock", args)

    def analyze_trx_and_lock(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("AnalyzeTrxAndLock", args)

    def describe_trx_and_locks(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeTrxAndLocks", args)

    def create_trx_export_task(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("CreateTrxExportTask", args)

    def describe_trx_snapshots(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeTrxSnapshots", args)

    def describe_err_logs(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeErrLogs", args)

    def describe_space_top(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeSpaceTop", args)

    def describe_table_space(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeTableSpace", args)

    def describe_table_spaces(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeTableSpaces", args)

    def describe_instance_nodes(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeInstanceNodes", args)

    def describe_health_summary(self, args: Dict[str, Any]) -> Dict[str, Any]:
        return self._call_api("DescribeHealthSummary", args)
