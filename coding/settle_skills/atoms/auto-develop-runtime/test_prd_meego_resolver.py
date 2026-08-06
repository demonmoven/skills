# -*- coding: utf-8 -*-
"""
单元测试:prd_meego_resolver 修复 readonly-block 卡片识别

测试用真实 PRD markdown 文本(来自 lark-doc +fetch 的本地缓存),
覆盖以下场景:
  1. 回归:明文 URL 仍能被识别(_extract_meego_urls)
  2. 新增:readonly-block type="meego" 卡片能被识别(_extract_meego_card_names)
  3. 新增:卡片 name → MQL 反查 → 拼装标准 URL(_resolve_meego_cards_to_urls)
  4. 集成:resolve() 飞书分支 + 真实 PRD,卡片反查成功路径
  5. 集成:resolve() 飞书分支 + 真实 PRD,卡片反查失败路径(warn 不阻塞)
  6. 兼容:卡片属性 type 在 name 前 / name 在 type 前两种写法
"""
import json
import sys
import unittest
from typing import Any, Dict
from urllib.parse import parse_qs, urlparse

sys.path.insert(0, "/home/mira/.session/141960289299/patch")
from prd_meego_resolver import (
    _extract_meego_card_names,
    _extract_meego_urls,
    _resolve_meego_cards_to_urls,
    resolve,
)

# 用 step 4 抓下来的真实 PRD markdown
with open("/home/mira/.session/141960289299/prd.md", encoding="utf-8") as f:
    PRD_RAW = json.loads(f.read())["data"]["document"]["content"]

PRD_URL = "https://bytedance.larkoffice.com/wiki/Fkp1w3Cz7if2MTk99TQcPvu3nJ4"

PROFILE: Dict[str, Any] = {
    "meego": {
        "url_pattern": r"https?://meego\.larkoffice\.com/[\w-]+/story/detail/\d+",
        "project_keys": ["zhifu"],
        "business_lines": ["支付", "渠道对接层", "合众-清结算"],
        "prd_field_names": ["需求文档", "PRD 文档", "PRD"],
    }
}


class TestExtractors(unittest.TestCase):
    def test_url_extractor_returns_empty_on_real_prd(self):
        # 真实 PRD 内确实 0 个明文 URL —— 这就是 bug 的根因
        self.assertEqual(_extract_meego_urls(PRD_RAW, PROFILE), [])

    def test_url_extractor_still_works_on_plain_url(self):
        # 回归:明文 URL 不能丢
        md = "见 https://meego.larkoffice.com/zhifu/story/detail/12345 跟进。"
        self.assertEqual(
            _extract_meego_urls(md, PROFILE),
            ["https://meego.larkoffice.com/zhifu/story/detail/12345"],
        )

    def test_card_name_extractor_on_real_prd(self):
        names = _extract_meego_card_names(PRD_RAW)
        self.assertEqual(len(names), 2)
        self.assertEqual(
            names,
            [
                "[需求]【收单分账产品】完结分账相关API接口去除商家分账的产品码校验",
                "[需求]【结算分账产品】完结分账相关API接口去除商家分账的产品码校验",
            ],
        )

    def test_card_name_extractor_attribute_order(self):
        # 兼容两种属性顺序
        md_a = '<readonly-block name="A" type="meego"></readonly-block>'
        md_b = '<readonly-block type="meego" name="B"></readonly-block>'
        md_c = '<readonly-block  type="meego"  name="C"  foo="x"></readonly-block>'
        self.assertEqual(_extract_meego_card_names(md_a), ["A"])
        self.assertEqual(_extract_meego_card_names(md_b), ["B"])
        self.assertEqual(_extract_meego_card_names(md_c), ["C"])

    def test_card_name_extractor_ignores_non_meego_type(self):
        md = '<readonly-block name="X" type="bits"></readonly-block>'
        self.assertEqual(_extract_meego_card_names(md), [])

    def test_card_name_extractor_deduplicates(self):
        md = (
            '<readonly-block name="A" type="meego"></readonly-block>'
            '<readonly-block name="A" type="meego"></readonly-block>'
        )
        self.assertEqual(_extract_meego_card_names(md), ["A"])


class TestCardLookup(unittest.TestCase):
    def test_lookup_success_uses_mql_protocol_and_pads_url(self):
        seen_probes = []

        def fake_getter(probe: str) -> Dict[str, Any]:
            seen_probes.append(probe)
            # 验证 getter 收到的伪 URL 协议
            u = urlparse(probe)
            assert u.scheme == "mql"
            assert u.netloc == "zhifu"
            qs = parse_qs(u.query)
            assert "name" in qs
            # 模拟 MQL 命中
            if "收单" in qs["name"][0]:
                return {"id": "1001"}
            if "结算" in qs["name"][0]:
                return {"items": [{"id": "1002"}]}
            return {}

        names = _extract_meego_card_names(PRD_RAW)
        urls, warns = _resolve_meego_cards_to_urls(names, PROFILE, fake_getter)
        self.assertEqual(
            urls,
            [
                "https://meego.larkoffice.com/zhifu/story/detail/1001",
                "https://meego.larkoffice.com/zhifu/story/detail/1002",
            ],
        )
        self.assertEqual(warns, [])
        self.assertEqual(len(seen_probes), 2)

    def test_lookup_skip_when_no_project_keys(self):
        profile_no_pk = {"meego": {"url_pattern": PROFILE["meego"]["url_pattern"]}}
        urls, warns = _resolve_meego_cards_to_urls(
            ["X"], profile_no_pk, lambda _u: {"id": "9"}
        )
        self.assertEqual(urls, [])
        self.assertTrue(any("project_keys is empty" in w for w in warns))

    def test_lookup_warn_when_getter_raises(self):
        def bad(_p):
            raise RuntimeError("MEEGO_AUTH_REQUIRED")

        urls, warns = _resolve_meego_cards_to_urls(["A", "B"], PROFILE, bad)
        self.assertEqual(urls, [])
        self.assertEqual(len(warns), 2)
        self.assertTrue(all("meego_card_lookup_failed" in w for w in warns))

    def test_lookup_warn_when_no_id_in_payload(self):
        urls, warns = _resolve_meego_cards_to_urls(
            ["A"], PROFILE, lambda _p: {"some_other_field": 1}
        )
        self.assertEqual(urls, [])
        self.assertTrue(any("meego_card_lookup_no_id" in w for w in warns))


class TestResolveIntegration(unittest.TestCase):
    """端到端跑 resolve():飞书分支 + 真实 PRD,覆盖卡片反查成功 / 失败两条路径。"""

    def _lark_fetcher(self, _url: str) -> str:
        return PRD_RAW

    def test_resolve_card_lookup_success(self):
        def meego_getter(probe: str) -> Dict[str, Any]:
            if probe.startswith("mql://"):
                # 命中第一个就够,resolver 会按 _pick_meego_by_business_line 兜底
                return {"id": "777"}
            # 二阶段:resolver 拿到 URL 后再 get_workitem 拿 story
            return {"fields": {"业务线": "支付"}}

        out = resolve(PRD_URL, PROFILE, self._lark_fetcher, meego_getter)
        self.assertIsNone(out["user_prompt"])
        self.assertFalse(out["needs_user_input"])
        self.assertEqual(out["prd_url"], PRD_URL)
        # 反查成功 → meego_url 不再是空
        self.assertTrue(
            out["meego_url"],
            f"expected meego_url to be filled, got {out['meego_url']!r}; warnings={out['warnings']}",
        )
        self.assertIn("zhifu/story/detail/777", out["meego_url"])

    def test_resolve_card_lookup_failure_does_not_block(self):
        # 模拟 sandbox 内 Meego 未登录
        def meego_getter(_probe: str) -> Dict[str, Any]:
            raise RuntimeError("MEEGO_AUTH_REQUIRED")

        out = resolve(PRD_URL, PROFILE, self._lark_fetcher, meego_getter)
        # 不阻塞:沿用现有"Meego 缺失永不阻塞飞书分支"原则
        self.assertFalse(out["needs_user_input"])
        self.assertEqual(out["prd_url"], PRD_URL)
        self.assertFalse(out["meego_url"])
        # 但 warning 必须把卡片标题列出来,避免"沉默失败"
        joined = " || ".join(out["warnings"])
        self.assertIn("no_meego_url_in_prd_but_cards_detected", joined)
        self.assertIn("收单分账", joined)
        self.assertIn("结算分账", joined)

    def test_resolve_plain_url_still_works(self):
        # 回归:PRD 内若有明文 URL,完全不走 readonly-block 兜底
        md_with_url = (
            "<title>x</title>\n"
            "见 https://meego.larkoffice.com/zhifu/story/detail/55555 跟进。"
        )

        def meego_getter(_u: str) -> Dict[str, Any]:
            return {"fields": {"业务线": "支付"}}

        out = resolve(
            PRD_URL, PROFILE, lambda _u: md_with_url, meego_getter
        )
        self.assertEqual(
            out["meego_url"],
            "https://meego.larkoffice.com/zhifu/story/detail/55555",
        )


if __name__ == "__main__":
    unittest.main(verbosity=2)
