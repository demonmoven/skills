"""ByteRedis Client - 通过 Cache Platform HTTP API 调用（参考 bytedcli 源码实现）"""
import json
import time
import urllib.request
import urllib.error
import urllib.parse

from common import _vregion_to_site, _SITE_HOST


# site → x-bcgw-vregion 映射
_SITE_VREGIONS = {
    "boe": "boe",
    "boei18n": "boei18n",
}


class RedisClient:
    def __init__(self, jwt_token, vregion="China-North"):
        self._site = _vregion_to_site(vregion)
        self._jwt_token = jwt_token
        self._host = _SITE_HOST.get(self._site, _SITE_HOST["cn"])
        self._base_url = f"{self._host}/api/v1/cache"
        self._supported_commands = None  # 懒加载，从服务端获取

    def _headers(self):
        return {
            "Accept": "application/json, text/plain, */*",
            "Accept-Language": "zh",
            "Content-Type": "application/json",
            "Origin": self._host,
            "Referer": f"{self._host}/",
            "x-jwt-token": self._jwt_token,
            "x-bcgw-tenant-id": "bytedance",
            "x-bcgw-vregion": _SITE_VREGIONS.get(self._site, "online"),
        }

    def _get(self, url, timeout=60):
        req = urllib.request.Request(url, headers=self._headers())
        try:
            resp = urllib.request.urlopen(req, timeout=timeout)
            return self._unwrap(url, json.loads(resp.read().decode()))
        except urllib.error.HTTPError as e:
            body = e.read().decode()
            raise ValueError(f"Cache API GET {url} 失败 ({e.code}): {body[:300]}")

    def _post(self, url, body, timeout=60):
        data = json.dumps(body).encode()
        req = urllib.request.Request(url, data=data, headers=self._headers(), method="POST")
        try:
            resp = urllib.request.urlopen(req, timeout=timeout)
            return self._unwrap(url, json.loads(resp.read().decode()))
        except urllib.error.HTTPError as e:
            body_str = e.read().decode()
            raise ValueError(f"Cache API POST {url} 失败 ({e.code}): {body_str[:300]}")

    def _unwrap(self, url, payload):
        """解析 Cache API 标准响应 {code, success, msg, data}"""
        if not isinstance(payload, dict):
            raise ValueError(f"Cache API 响应格式异常: {str(payload)[:200]}")
        code = payload.get("code")
        success = payload.get("success")
        msg = payload.get("msg") or payload.get("message") or ""
        if (isinstance(code, int) and code not in (200, 0)) or success is False:
            raise ValueError(f"Cache API 错误: {msg}")
        return payload.get("data")

    # ── 服务查询 ──

    def list_starred_services(self, page=1, size=20):
        params = urllib.parse.urlencode({"page": page, "page_size": size, "favor": "1"})
        return self._get(f"{self._base_url}/service/front_list?{params}")

    def search_services(self, keyword, page=1, size=20):
        params = urllib.parse.urlencode({
            "page": page, "page_size": size, "favor": "0", "search": keyword,
        })
        return self._get(f"{self._base_url}/service/front_list?{params}")

    def get_service(self, psm=None, service_id=None):
        """通过 PSM 或 service_id 获取服务详情（含 clusters 列表）"""
        if service_id:
            return self._get(f"{self._base_url}/service/{service_id}")
        if not psm:
            raise ValueError("需要提供 psm 或 service_id")
        # PSM → service_id：搜索后精确匹配
        data = self.search_services(psm, page=1, size=20)
        results = data.get("results", []) if isinstance(data, dict) else []
        exact = next((s for s in results if s.get("psm") == psm or s.get("consul_psm") == psm), None)
        svc = exact or (results[0] if results else None)
        if not svc or not svc.get("id"):
            raise ValueError(f"未找到 PSM '{psm}' 对应的服务")
        return self._get(f"{self._base_url}/service/{svc['id']}")

    def _resolve_cluster_id(self, psm, idc=None):
        """PSM → cluster_id（选第一个集群，或按 idc 筛选）"""
        svc = self.get_service(psm=psm)
        clusters = svc.get("clusters", []) if isinstance(svc, dict) else []
        if not clusters:
            raise ValueError(f"PSM '{psm}' 没有可用集群")
        if idc:
            selected = next((c for c in clusters if c.get("idc") == idc), None)
            if not selected:
                available = [c.get("idc") for c in clusters]
                raise ValueError(f"PSM '{psm}' 无 idc='{idc}' 的集群，可用: {available}")
        else:
            selected = clusters[0]
        cid = selected.get("id")
        if not cid:
            raise ValueError(f"PSM '{psm}' 集群无有效 ID")
        return cid, clusters

    # ── 命令执行 ──

    def list_commands(self):
        """获取服务端支持的命令白名单"""
        if self._supported_commands is None:
            data = self._get(f"{self._base_url}/command")
            self._supported_commands = set(data) if isinstance(data, list) else set()
        return self._supported_commands

    def execute_command(self, psm, command, args="", cluster_id=None, idc=None):
        if not cluster_id:
            cluster_id, _ = self._resolve_cluster_id(psm, idc)
        body = {"command": command, "args": args}
        return self._post(f"{self._base_url}/cluster/{cluster_id}/execute_command", body)

    # ── 慢日志 ──

    def _get_cluster_endpoints(self, cluster_id):
        """获取集群实例的 endpoint 列表"""
        data = self._get(f"{self._base_url}/cluster/{cluster_id}")
        insts = data.get("insts", []) if isinstance(data, dict) else []
        endpoints = []
        for inst in insts:
            addr = inst.get("addr")
            if addr:
                endpoints.append(addr)
            elif inst.get("ip") and inst.get("port"):
                endpoints.append(f"{inst['ip']}:{inst['port']}")
        return endpoints

    def _slow_log_for_cluster(self, cluster_id, poll_interval=2, max_wait=15):
        """对单个 cluster 启动慢日志任务并轮询结果"""
        endpoints = self._get_cluster_endpoints(cluster_id)
        if not endpoints:
            raise ValueError(f"集群 {cluster_id} 无可用实例 endpoint")

        start_data = self._post(
            f"{self._base_url}/cluster/{cluster_id}/slow_log",
            {"endpoints": endpoints},
        )
        task = start_data.get("task", {}) if isinstance(start_data, dict) else {}
        task_id = task.get("id")
        if not task_id:
            raise ValueError("慢日志任务启动失败，未获得 task_id")

        elapsed, result = 0, None
        while elapsed < max_wait:
            time.sleep(poll_interval)
            elapsed += poll_interval
            params = urllib.parse.urlencode({"task_id": task_id})
            result = self._get(f"{self._base_url}/cluster/{cluster_id}/slow_log?{params}")
            if isinstance(result, dict):
                state = result.get("state", "")
                if state in ("done", "finished", "completed", "success"):
                    return result
                results = result.get("results", {})
                if isinstance(results, dict) and results.get("slow_log"):
                    return result

        return result if isinstance(result, dict) else {"state": "timeout", "task_id": task_id}

    def slow_log(self, psm=None, cluster_id=None, idc=None,
                 poll_interval=2, max_wait=15):
        """启动慢日志任务并轮询结果。自动遍历所有 cluster 直到成功。"""
        if cluster_id:
            return self._slow_log_for_cluster(cluster_id, poll_interval, max_wait)

        _, clusters = self._resolve_cluster_id(psm, idc)
        last_err = None
        for cluster in clusters:
            cid = cluster.get("id")
            if not cid:
                continue
            try:
                return self._slow_log_for_cluster(cid, poll_interval, max_wait)
            except Exception as e:
                last_err = e
                continue

        raise ValueError(f"所有集群均不可用（最后错误: {last_err}）")

    # ── 大 Key ──

    def list_big_keys(self, psm, date, begin="00:00:00", end="23:59:59",
                      page=1, size=10, key_type="string"):
        params = urllib.parse.urlencode({
            "psm": psm, "date": date, "begin": begin, "end": end,
            "page": page, "page_size": size, "type": key_type,
        })
        return self._get(f"{self._base_url}/key/get_big_keys/v2?{params}")
