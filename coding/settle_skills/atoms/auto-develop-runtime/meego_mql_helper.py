# -*- coding: utf-8 -*-
"""
Meego MQL 辅助 atom(auto-develop Step 1 复用,2026-06-03 新增)。

定位:
1. **MQL 反查兜底**:飞书 PRD 内 `<readonly-block type="meego">` 嵌入卡片只有 name 没有 URL,
   必须 MQL 反查;同时把 `business`(需求业务线 cascade)一并 select 回来,避开
   后续 `workitem get --rich` 对浏览器 cookie 的依赖
2. **Cascade 拍平**:Meego 的 `_business` 字段返回多级嵌套,本 atom 把它拍平成
   "支付 / 渠道对接层 / 合众-清结算" 这种人类可读的 path,塞进 fields["业务线"],
   供 resolver `_read_business_line` / `_pick_meego_by_business_line` 直接消费
3. **路由层缓存协议**:把 MQL 反查命中的每个 story 按 URL 索引缓存,后续
   `meego_workitem_getter(real_url)` 第一时间命中,无需再发 CLI 调用,
   避开 goapi cookie 缺失场景

使用方式(在路由层 `meego_workitem_getter` 实现里):

    from meego_mql_helper import MqlGetter

    mql_getter = MqlGetter()  # 整个路由生命周期共享一个实例,保留缓存

    def meego_workitem_getter(probe: str) -> dict:
        if probe.startswith("mql://"):
            return mql_getter.lookup_by_name(probe)
        # 优先吃缓存
        cached = mql_getter.get_cached(probe)
        if cached is not None:
            return cached
        # 兜底走 bytedcli meego workitem get --url <probe> --rich
        return _run_workitem_get_rich(probe)
"""

import json
import os
import re
import subprocess
import urllib.parse
from typing import Any, Dict, Optional


# 业务线字段在 Meego 内部 key(label = "需求业务线",type = "_business" cascade)
_BUSINESS_FIELD_KEY = "business"

# 业务线在 resolver 端的中文别名 key(_read_business_line 会读 fields["业务线"])
_BUSINESS_FIELD_ALIAS = "业务线"

# story URL 模板
_URL_TEMPLATE = "https://meego.larkoffice.com/{project_key}/story/detail/{story_id}"


def _env_with_npm_bin() -> Dict[str, str]:
    env = os.environ.copy()
    env["PATH"] = os.path.expanduser("~/.npm-global/bin") + ":" + env.get("PATH", "")
    return env


def _flatten_cascade(node: Optional[Dict[str, Any]]) -> Optional[str]:
    """Cascade tree → ' / '.join(labels);取每层第一个子节点的 path。"""
    if not node:
        return None
    labels = []
    cur = node
    while cur:
        if cur.get("label"):
            labels.append(cur["label"])
        children = cur.get("children") or []
        cur = children[0] if children else None
    return " / ".join(labels) if labels else None


def _extract_field_value(field: Dict[str, Any]) -> Any:
    """从 MQL moql_field_list 单个 field 提取业务可读值。"""
    v = field.get("value") or {}
    for k in ("long_value", "string_value", "double_value", "bool_value"):
        if k in v and v[k] is not None:
            return v[k]
    casc = v.get("cascade_key_label_value")
    if casc:
        return _flatten_cascade(casc)
    for k in ("option_value", "user_value"):
        if k in v and v[k]:
            return v[k].get("label") or v[k].get("name") or str(v[k])
    return None


def _strip_card_prefix(name: str) -> str:
    """飞书 PRD 卡片 name 常带 '[需求]' / '[Bug]' 等前缀,Meego 真实标题没有。"""
    return re.sub(r"^\s*\[[^\]]{1,8}\]\s*", "", name).strip()


def _make_url(project_key: str, story_id: Any) -> str:
    return _URL_TEMPLATE.format(project_key=project_key, story_id=story_id)


class MqlGetter:
    """
    路由生命周期内共享的 MQL 反查器 + URL → story 缓存。

    线程安全:无锁;auto-develop 路由是单线程编排,无需并发保护。
    """

    def __init__(self):
        self._cache: Dict[str, Dict[str, Any]] = {}

    def get_cached(self, url: str) -> Optional[Dict[str, Any]]:
        """命中即返,未命中返 None(由调用方决定是否回退到 workitem get)。"""
        return self._cache.get(url)

    def lookup_by_name(self, probe: str) -> Dict[str, Any]:
        """
        解析 `mql://<project_key>?name=<urlencoded>` 伪 URL → 返回 resolver 期望的 payload:
            {"items": [{"id":..., "name":..., "fields":{"业务线":...}}, ...], "id": <first_id>}

        同时把每个命中的 story 按真实 URL 写入缓存,供后续 get_cached 命中。
        失败抛 RuntimeError,由调用方包装 warning。
        """
        parsed = urllib.parse.urlparse(probe)
        project_key = parsed.netloc
        if not project_key:
            raise RuntimeError(f"mql probe missing project_key: {probe!r}")
        qs = urllib.parse.parse_qs(parsed.query)
        name = qs.get("name", [""])[0]
        if not name:
            raise RuntimeError(f"mql probe missing name: {probe!r}")

        core = _strip_card_prefix(name)
        safe_core = core.replace("'", "\\'").replace("%", "")
        mql = (
            f"select `work_item_id`,`name`,`{_BUSINESS_FIELD_KEY}` "
            f"from `{project_key}`.`story` "
            f"where `name` like '%{safe_core}%'"
        )
        cmd = [
            "bytedcli", "--json", "meego", "workitem", "list",
            "--project-key", project_key,
            "--mql", mql,
        ]
        r = subprocess.run(
            cmd, capture_output=True, text=True, env=_env_with_npm_bin(), timeout=60
        )
        if r.returncode != 0:
            try:
                err = json.loads(r.stdout or "{}")
                msg = (err.get("error") or {}).get("message") or r.stderr or "unknown"
            except Exception:
                msg = r.stderr or r.stdout or "unknown"
            raise RuntimeError(f"meego MQL CLI failed: {msg.strip()[:200]}")
        try:
            payload = json.loads(r.stdout)
        except json.JSONDecodeError as e:
            raise RuntimeError(f"meego MQL returned non-JSON: {e}; raw={r.stdout[:200]}")
        if payload.get("status") != "success":
            msg = (payload.get("error") or {}).get("message") or "unknown"
            raise RuntimeError(f"meego MQL status=error: {msg[:200]}")

        data = payload.get("data") or {}
        contents = (data.get("result") or {}).get("content") or []
        if not contents:
            return {}
        try:
            inner = json.loads(contents[0]["text"])
        except (json.JSONDecodeError, KeyError) as e:
            raise RuntimeError(f"meego MQL inner payload parse failed: {e}")

        items = []
        for _gid, rows in (inner.get("data") or {}).items():
            for row in rows:
                parsed_row = {
                    f["key"]: _extract_field_value(f)
                    for f in row.get("moql_field_list") or []
                }
                wid = parsed_row.get("work_item_id")
                if wid is None:
                    continue
                fields_dict = {}
                biz = parsed_row.get(_BUSINESS_FIELD_KEY)
                if biz:
                    fields_dict[_BUSINESS_FIELD_ALIAS] = biz
                story = {
                    "id": wid,
                    "name": parsed_row.get("name"),
                    "fields": fields_dict,
                }
                items.append(story)
                # 写缓存:供后续 _pick_meego_by_business_line 调 getter(real_url) 时命中
                self._cache[_make_url(project_key, wid)] = story

        if not items:
            return {}
        return {"items": items, "id": items[0]["id"]}


if __name__ == "__main__":
    # 命令行自测:python3 meego_mql_helper.py "mql://zhifu?name=...uri-encoded-name..."
    import sys
    if len(sys.argv) < 2:
        print("Usage: meego_mql_helper.py 'mql://<project_key>?name=<urlencoded>'", file=sys.stderr)
        sys.exit(1)
    g = MqlGetter()
    result = g.lookup_by_name(sys.argv[1])
    print(json.dumps(result, ensure_ascii=False, indent=2))
