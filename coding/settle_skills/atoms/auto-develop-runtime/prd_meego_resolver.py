# -*- coding: utf-8 -*-
"""
PRD ↔ Meego 双向解析原子。

入口:resolve(input_url, profile, lark_doc_fetcher, meego_workitem_getter)
返回:dict 包含 prd_url / meego_url / prd_md / meego_story / warnings / needs_user_input

设计要点:
1. 飞书 URL → 抽 Meego(失败留空,看板 panel 标"待补",warn 不阻塞)
2. Meego URL → 取 PRD 文档字段(profile.meego.prd_field_names 任一命中即用);文档字段空 → needs_user_input=True 阻塞
3. 飞书 PRD 含多个 Meego 链接 → 按 profile.meego.business_lines 匹配业务线,选最匹配的;全无匹配兜底取第一个 + 警告
4. 不操作 IO;由调用方注入 fetcher/getter callable,便于在 sandbox / 本地 / 测试环境复用

profile schema 见 config/team_profile.yaml(本仓库根)。
"""

import re
from typing import Any, Callable, Dict, List, Optional, Tuple

LARK_URL_RE = re.compile(
    r'https?://[a-zA-Z0-9.-]*larkoffice\.com/(?:wiki|docx|docs)/[A-Za-z0-9_-]+'
)

# 飞书 PRD 模板中"嵌入式 Meego 卡片"的形态,markdown 导出形如:
#   <readonly-block name="[需求]xxx" type="meego"></readonly-block>
# 卡片只携带 name,不携带 story_id / URL,需要走 Meego MQL 反查
# 兼容 name / type 两种属性顺序
READONLY_MEEGO_RE = re.compile(
    r'<readonly-block\b[^>]*?'
    r'(?:'
    r'name="(?P<name1>[^"]+)"[^>]*?type="meego"'
    r'|'
    r'type="meego"[^>]*?name="(?P<name2>[^"]+)"'
    r')'
    r'[^>]*>',
    re.IGNORECASE,
)

# Meego URL 拼装模板,用 project_key + story_id 组装回标准 URL
MEEGO_URL_TEMPLATE = "https://meego.larkoffice.com/{project_key}/story/detail/{story_id}"


def _is_meego_url(url: str, profile: Dict[str, Any]) -> bool:
    pattern = profile.get("meego", {}).get("url_pattern")
    if not pattern:
        raise ValueError("profile.meego.url_pattern missing")
    return bool(re.match(pattern, url))


def _is_lark_url(url: str) -> bool:
    return bool(LARK_URL_RE.match(url))


def _extract_meego_urls(text: str, profile: Dict[str, Any]) -> List[str]:
    pattern = profile.get("meego", {}).get("url_pattern")
    if not pattern:
        return []
    return list(dict.fromkeys(re.findall(pattern, text)))  # 去重保序


def _extract_meego_card_names(text: str) -> List[str]:
    """
    抽取飞书 PRD markdown 中所有 `<readonly-block type="meego" name="...">` 卡片的 name。
    卡片是飞书新版 PRD 模板挂 Meego 关联的默认形式,无 URL 无 story_id,
    需要后续走 Meego MQL 反查。返回去重保序的 name 列表。
    """
    if not text:
        return []
    names: List[str] = []
    for m in READONLY_MEEGO_RE.finditer(text):
        name = m.group("name1") or m.group("name2")
        if name:
            names.append(name)
    return list(dict.fromkeys(names))


def _resolve_meego_cards_to_urls(
    names: List[str],
    profile: Dict[str, Any],
    meego_workitem_getter: Callable[[str], Dict[str, Any]],
) -> Tuple[List[str], List[str]]:
    """
    将 readonly-block 卡片 name 反查为标准 Meego URL。

    实现策略:
    1. 通过注入的 `meego_workitem_getter` 的 `search`/`mql` 能力按 name 精确匹配 story
       - 路由层注入的 getter 在第一个参数为 `mql://<project_key>?name=<encoded>` 的特殊伪 URL 时
         应改走 `bytedcli meego workitem list --project-key <pk> --mql "select id,name from story where name='<name>'"`
       - 老版本路由层未实现此协议时,本函数会捕获异常并 warn,不抛
    2. 命中后用 profile.meego.project_keys[0] + story_id 拼装 URL_TEMPLATE
    3. 完全未命中或 getter 不支持反查 → 返回 ([], warnings) 让上层走"⚠️ 待补充"分支

    return: (resolved_urls, warnings)
    """
    warnings: List[str] = []
    project_keys = profile.get("meego", {}).get("project_keys") or []
    if not project_keys:
        warnings.append("meego_card_lookup_skipped: profile.meego.project_keys is empty")
        return [], warnings
    project_key = project_keys[0]

    resolved: List[str] = []
    for name in names:
        try:
            # 约定:伪 URL 协议 `mql://<pk>?name=<urlencoded>`,由路由层 getter 翻译为 MQL 查询
            from urllib.parse import quote
            probe = f"mql://{project_key}?name={quote(name, safe='')}"
            story = meego_workitem_getter(probe)
        except Exception as e:
            warnings.append(f"meego_card_lookup_failed: name={name!r} -> {e}")
            continue

        if not isinstance(story, dict):
            warnings.append(f"meego_card_lookup_bad_payload: name={name!r} type={type(story).__name__}")
            continue

        # 兼容多种 payload 形态:{id|story_id|workitem_id} / {data:{id}} / {items:[{id}]}
        sid = (
            story.get("id")
            or story.get("story_id")
            or story.get("workitem_id")
            or (story.get("data") or {}).get("id")
            or (((story.get("items") or [{}])[0]) or {}).get("id")
        )
        if not sid:
            warnings.append(f"meego_card_lookup_no_id: name={name!r}")
            continue

        resolved.append(MEEGO_URL_TEMPLATE.format(project_key=project_key, story_id=sid))

    return list(dict.fromkeys(resolved)), warnings


def _extract_lark_url(text: str) -> Optional[str]:
    m = LARK_URL_RE.search(text or "")
    return m.group(0) if m else None


def _pick_meego_by_business_line(
    candidates: List[str],
    profile: Dict[str, Any],
    meego_workitem_getter: Callable[[str], Dict[str, Any]],
) -> Tuple[Optional[str], Optional[Dict[str, Any]], List[str]]:
    """
    多 Meego 候选时,按业务线匹配挑最优。

    匹配优先级(2026-06-03 起):
    1. `meego.preferred_business_line`(本域硬约束)— 命中即选,**不再看其它**
    2. `meego.business_lines`(通配候选池)— preferred 未命中时退回
    3. 全无命中 → 兜底取第一个 fetched + warn

    return: (picked_url, story_payload, warnings)
    """
    meego_cfg = profile.get("meego", {}) or {}
    preferred = meego_cfg.get("preferred_business_line") or ""
    business_lines = meego_cfg.get("business_lines") or []
    warnings: List[str] = []

    if len(candidates) == 1:
        try:
            story = meego_workitem_getter(candidates[0])
            return candidates[0], story, warnings
        except Exception as e:
            warnings.append(f"meego_get_failed: {candidates[0]} -> {e}")
            return candidates[0], None, warnings

    # 多个候选:逐个 get
    fetched: List[Tuple[str, Dict[str, Any]]] = []
    for url in candidates:
        try:
            story = meego_workitem_getter(url)
            fetched.append((url, story))
        except Exception as e:
            warnings.append(f"meego_get_failed: {url} -> {e}")

    # 优先级 1:preferred_business_line 严格匹配(子串包含即命中,大小写敏感)
    if preferred:
        for url, story in fetched:
            bl = _read_business_line(story)
            if bl and preferred in bl:
                warnings.append(
                    f"meego_pick_by_preferred: matched '{preferred}' in '{bl}' -> {url}"
                )
                return url, story, warnings

    # 优先级 2:business_lines 通配
    for url, story in fetched:
        bl = _read_business_line(story)
        if bl and any(line in bl for line in business_lines):
            return url, story, warnings

    # 全无业务线匹配:兜底取第一个 fetched(若都失败,取 candidates[0])
    if fetched:
        warnings.append(
            f"multi_meego_no_business_match: preferred='{preferred}' "
            f"business_lines={business_lines} -> fallback to first {fetched[0][0]}"
        )
        return fetched[0][0], fetched[0][1], warnings

    warnings.append(f"multi_meego_all_get_failed: fallback to first {candidates[0]}")
    return candidates[0], None, warnings


def _read_business_line(story: Optional[Dict[str, Any]]) -> Optional[str]:
    """从 meego rich payload 中尽可能取出 '业务线' / '产品线' 字段值。"""
    if not story:
        return None
    # bytedcli meego workitem get --rich 的 fields 区,字段名键值
    candidates = ("业务线", "产品线", "业务方向", "业务领域")
    fields = story.get("fields") or story.get("data") or {}
    if isinstance(fields, dict):
        for key in candidates:
            v = fields.get(key)
            if v:
                return v if isinstance(v, str) else (v.get("name") or str(v))
    return None


def _read_prd_url_from_meego(story: Dict[str, Any], profile: Dict[str, Any]) -> Optional[str]:
    """从 meego rich payload 中按 profile.meego.prd_field_names 顺序匹配 PRD 文档字段。"""
    prd_field_names = profile.get("meego", {}).get("prd_field_names") or []
    fields = story.get("fields") or story.get("data") or {}
    if not isinstance(fields, dict):
        return None
    for key in prd_field_names:
        v = fields.get(key)
        if not v:
            continue
        # 字段值可能是 str / dict(含 url)/list
        if isinstance(v, str):
            url = _extract_lark_url(v) or (v if _is_lark_url(v) else None)
            if url:
                return url
        elif isinstance(v, dict):
            for k in ("url", "link", "href"):
                if v.get(k) and _is_lark_url(v[k]):
                    return v[k]
            # 兜底从 dict 文本表达里抽
            url = _extract_lark_url(str(v))
            if url:
                return url
        elif isinstance(v, list):
            for item in v:
                if isinstance(item, str):
                    url = _extract_lark_url(item) or (item if _is_lark_url(item) else None)
                    if url:
                        return url
                elif isinstance(item, dict):
                    for k in ("url", "link", "href"):
                        if item.get(k) and _is_lark_url(item[k]):
                            return item[k]
    return None


def resolve(
    input_url: str,
    profile: Dict[str, Any],
    lark_doc_fetcher: Callable[[str], str],
    meego_workitem_getter: Callable[[str], Dict[str, Any]],
) -> Dict[str, Any]:
    """
    主入口。

    :param input_url: 用户给的链接(飞书 URL 或 Meego URL)
    :param profile: team_profile.yaml 加载后的 dict
    :param lark_doc_fetcher: callable(url) -> str 飞书文档 markdown 内容
    :param meego_workitem_getter: callable(url) -> dict bytedcli meego workitem get --rich --json 的 .data
    :return: dict {prd_url, meego_url, prd_md, meego_story, warnings, needs_user_input, user_prompt}
    """
    result: Dict[str, Any] = {
        "prd_url": None,
        "meego_url": None,
        "prd_md": None,
        "meego_story": None,
        "warnings": [],
        "needs_user_input": False,
        "user_prompt": None,
    }

    if _is_meego_url(input_url, profile):
        # ===== Meego 分支 =====
        result["meego_url"] = input_url
        try:
            story = meego_workitem_getter(input_url)
            result["meego_story"] = story
        except Exception as e:
            result["warnings"].append(f"meego_get_failed: {e}")
            result["needs_user_input"] = True
            result["user_prompt"] = (
                f"无法读取 Meego 工单 {input_url}(错误:{e})。\n"
                f"请确认 1) bytedcli meego login 已完成 2) 当前账号对该工单有读权限,\n"
                f"或直接粘贴飞书 PRD 文档 URL 让流程继续。"
            )
            return result

        prd_url = _read_prd_url_from_meego(story, profile)
        if not prd_url:
            result["needs_user_input"] = True
            result["user_prompt"] = (
                f"该 Meego 工单 {input_url} 中未找到 PRD 文档字段"
                f"(已尝试字段名:{profile.get('meego', {}).get('prd_field_names')})。\n"
                f"请补充飞书 PRD 文档 URL 后回复,流程会继续。"
            )
            return result

        result["prd_url"] = prd_url
        try:
            result["prd_md"] = lark_doc_fetcher(prd_url)
        except Exception as e:
            result["warnings"].append(f"lark_fetch_failed: {prd_url} -> {e}")
            result["needs_user_input"] = True
            result["user_prompt"] = (
                f"读取飞书 PRD 文档 {prd_url} 失败(错误:{e})。\n"
                f"请确认 lark-doc 工具链可用、当前账号对文档有读权限,然后回复"
                f"「重试」或粘贴新的 PRD URL。"
            )
            return result

        return result

    elif _is_lark_url(input_url):
        # ===== 飞书 PRD 分支 =====
        result["prd_url"] = input_url
        try:
            md = lark_doc_fetcher(input_url)
            result["prd_md"] = md
        except Exception as e:
            result["warnings"].append(f"lark_fetch_failed: {e}")
            result["needs_user_input"] = True
            result["user_prompt"] = (
                f"读取飞书 PRD 文档 {input_url} 失败(错误:{e})。\n"
                f"请确认 lark-doc 工具链可用、当前账号对文档有读权限。"
            )
            return result

        meego_candidates = _extract_meego_urls(md, profile)

        # 兜底:无明文 URL 时,尝试识别 `<readonly-block type="meego">` 嵌入卡片并反查 story_id
        # 飞书新版 PRD 模板已默认用卡片代替明文 URL 挂 Meego,不做此兜底会导致 100% 漏识别
        card_lookup_warns: List[str] = []
        detected_card_names: List[str] = []
        if not meego_candidates:
            detected_card_names = _extract_meego_card_names(md)
            if detected_card_names:
                resolved_urls, card_lookup_warns = _resolve_meego_cards_to_urls(
                    detected_card_names, profile, meego_workitem_getter
                )
                meego_candidates = resolved_urls

        if not meego_candidates:
            # 没找到 Meego URL —— warn 不阻塞,留空
            # 若检测到 readonly-block 卡片但反查失败,把卡片标题列出来便于人工补
            if detected_card_names:
                result["warnings"].append(
                    "no_meego_url_in_prd_but_cards_detected: 飞书 PRD 内检测到 "
                    f"{len(detected_card_names)} 个 Meego 嵌入卡片但 MQL 反查未拿到 story_id;"
                    f"卡片标题=[{', '.join(repr(n) for n in detected_card_names)}];"
                    "看板 s1 panel 会标记为「⚠️ 待补充」。可在对话中直接补明文 Meego URL,"
                    "或确认 1) bytedcli meego login 已完成 2) 路由层 getter 已实现 mql:// 协议。"
                )
                result["warnings"].extend(card_lookup_warns)
            else:
                result["warnings"].append(
                    "no_meego_url_in_prd: 飞书 PRD 内未发现 Meego 链接或嵌入卡片,meego_url 留空,"
                    "看板 s1 panel 会标记为「⚠️ 待补充」。可在后续步骤通过对话补 Meego URL。"
                )
            return result

        if len(meego_candidates) == 1:
            url = meego_candidates[0]
            try:
                story = meego_workitem_getter(url)
                result["meego_url"] = url
                result["meego_story"] = story
            except Exception as e:
                result["warnings"].append(f"meego_get_failed: {url} -> {e};仍登记 meego_url 但 story 为空")
                result["meego_url"] = url
            return result

        # 多个候选:按业务线挑
        picked, story, warns = _pick_meego_by_business_line(
            meego_candidates, profile, meego_workitem_getter
        )
        result["meego_url"] = picked
        result["meego_story"] = story
        result["warnings"].extend(warns)
        return result

    else:
        # 既不是 lark 也不是 meego
        result["needs_user_input"] = True
        result["user_prompt"] = (
            f"输入 URL 既不是飞书文档(*.larkoffice.com/wiki|docx|docs/...)"
            f"也不是 Meego 工单(匹配 profile.meego.url_pattern)。\n"
            f"请提供合法链接。当前输入:{input_url}"
        )
        return result
