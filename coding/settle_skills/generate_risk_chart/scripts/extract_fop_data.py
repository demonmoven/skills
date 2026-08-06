#!/usr/bin/env python3
"""
FOP 平台数据采集脚本

从 FOP 平台采集追光清结算的 DDA（实时核对）和 HSQL（离线核对）任务数据。

使用方式：
    # 在 Aime 平台中执行（使用内置浏览器工具）
    python3 scripts/extract_fop_data.py --org_id ORG240313190058294165643265 --output_dir ./data

    # 独立运行（需配合 Selenium/Playwright）
    python3 scripts/extract_fop_data.py --org_id ORG240313190058294165643265 --output_dir ./data --driver selenium

依赖：
    - Aime 平台: 内置 browser 工具（无需额外安装）
    - 独立运行: pip install selenium 或 pip install playwright

说明：
    本脚本通过浏览器自动化从 FOP 前端表格中提取数据。
    页面使用 Arco Design 组件库渲染表格，通过 JS 注入方式提取 DOM 中的数据。
"""

import json
import os
import sys
import re
import time
import argparse
import subprocess
from typing import Any, Dict, List, Optional


# ===== 浏览器工具适配层 =====

class AimeBrowserDriver:
    """Aime 平台内置浏览器驱动"""

    def __init__(self, toolset_dir: str):
        self.toolset_dir = toolset_dir
        self.session_id = None
        self.tab_id = None

    def _run_tool(self, tool: str, payload: dict) -> str:
        cmd = [
            "python3",
            os.path.join(self.toolset_dir, f"{tool}.py"),
            json.dumps(payload, ensure_ascii=False, separators=(",", ":")),
        ]
        res = subprocess.run(cmd, capture_output=True, text=True)
        if res.returncode != 0:
            raise RuntimeError(f"{tool} failed: {res.stderr or res.stdout}")
        return res.stdout

    def acquire_session(self) -> str:
        text = self._run_tool("browser_list_sessions", {})
        matches = re.findall(r"session_id:([^\s]+)\s+status:([^\s\]]+)", text)
        free = [sid for sid, status in matches if status == "free"]
        if not free:
            raise RuntimeError("no free browser session")
        session_id = free[0]
        self._run_tool("browser_acquire_session", {"session_id": session_id})
        self.session_id = session_id
        return session_id

    def release_session(self):
        if self.session_id:
            try:
                self._run_tool("browser_release_session", {"session_id": self.session_id})
            except Exception:
                pass

    def navigate(self, url: str) -> int:
        text = self._run_tool(
            "browser_navigate",
            {"session_id": self.session_id, "url": url, "new_tab": True},
        )
        ids = re.findall(r"tab_id:(\d+)", text)
        if not ids:
            raise RuntimeError(f"cannot find tab_id: {text}")
        self.tab_id = int(ids[-1])
        return self.tab_id

    def wait(self, wait_ms: int = 8000):
        self._run_tool("browser_wait", {
            "session_id": self.session_id,
            "tab_id": self.tab_id,
            "wait_ms": wait_ms,
        })

    def eval_js(self, expression: str, timeout_ms: int = 20000) -> Any:
        out = self._run_tool("browser_eval", {
            "session_id": self.session_id,
            "tab_id": self.tab_id,
            "expression": expression,
            "timeout_ms": timeout_ms,
        })
        return self._parse_eval_json(out)

    def _parse_eval_json(self, out: str) -> Any:
        if "truncated:true" in out:
            raise RuntimeError("browser_eval output truncated")
        idx = out.find("AIME_JSON:")
        if idx == -1:
            raise RuntimeError(f"AIME_JSON marker not found: {out[:200]}")
        raw = out[idx + len("AIME_JSON:"):].strip()
        if not raw:
            return None
        first = raw[0]
        if first in "[{":
            open_ch, close_ch = (first, "]" if first == "[" else "}")
            depth = 0
            for i, ch in enumerate(raw):
                if ch == open_ch:
                    depth += 1
                elif ch == close_ch:
                    depth -= 1
                    if depth == 0:
                        return json.loads(raw[: i + 1])
            raise RuntimeError(f"cannot find JSON end: {raw[:200]}")
        line = raw.splitlines()[0]
        try:
            return json.loads(line)
        except Exception:
            return line


class SeleniumBrowserDriver:
    """Selenium 浏览器驱动（独立部署时使用）"""

    def __init__(self):
        try:
            from selenium import webdriver
            from selenium.webdriver.chrome.options import Options
        except ImportError:
            raise RuntimeError("请安装 selenium: pip install selenium")

        options = Options()
        options.add_argument("--headless")
        options.add_argument("--no-sandbox")
        options.add_argument("--disable-dev-shm-usage")
        self.driver = webdriver.Chrome(options=options)

    def acquire_session(self) -> str:
        return "selenium-session"

    def release_session(self):
        if self.driver:
            self.driver.quit()

    def navigate(self, url: str) -> int:
        self.driver.get(url)
        return 0

    def wait(self, wait_ms: int = 8000):
        time.sleep(wait_ms / 1000.0)

    def eval_js(self, expression: str, timeout_ms: int = 20000) -> Any:
        # 移除 AIME_JSON 标记，直接执行并返回
        js = expression.replace(
            'return "AIME_JSON:" + JSON.stringify(',
            'return JSON.stringify('
        ).replace(
            'return "AIME_JSON:" +JSON.stringify(',
            'return JSON.stringify('
        )
        result = self.driver.execute_script(js)
        if result and isinstance(result, str):
            return json.loads(result)
        return result


# ===== JS 表达式模板 =====

DDA_EXPR_TEMPLATE = """(function(){
  var rows = document.querySelectorAll("table tbody tr, .arco-table-body tr");
  var data = [];
  rows.forEach(function(row, idx) {
    if (idx < __START__ || idx >= __END__) return;
    var cells = row.querySelectorAll("td");
    if (cells.length >= 14) {
      data.push({
        code: cells[0].textContent.trim(),
        name: cells[1].textContent.trim(),
        status: cells[2].textContent.trim(),
        mode: cells[3].textContent.trim(),
        createTime: cells[5].textContent.trim(),
        modifyTime: cells[6].textContent.trim(),
        priority: cells[9].textContent.trim(),
        modifier: cells[13].textContent.trim()
      });
    }
  });
  return "AIME_JSON:" + JSON.stringify(data);
})()"""

HSQL_EXPR_TEMPLATE = """(function(){
  var rows = document.querySelectorAll("table tbody tr, .arco-table-body tr");
  var data = [];
  rows.forEach(function(row, idx) {
    if (idx < __START__ || idx >= __END__) return;
    var cells = row.querySelectorAll("td");
    if (cells.length >= 8) {
      data.push({
        code: cells[0].textContent.trim(),
        name: cells[1].textContent.trim(),
        priority: cells[4].textContent.trim(),
        status: cells[6].textContent.trim(),
        creator: cells[7].textContent.trim()
      });
    }
  });
  return "AIME_JSON:" + JSON.stringify(data);
})()"""

HAS_NEXT_EXPR = """(function(){
  var next = document.querySelector(".arco-pagination-item-next");
  var hasNext = false;
  if (next && !next.classList.contains("arco-pagination-item-disabled")) {
    hasNext = true;
  }
  var active = document.querySelector(".arco-pagination .arco-pagination-item-active");
  var page = active ? active.textContent.trim() : null;
  return "AIME_JSON:" + JSON.stringify({hasNext: hasNext, currentPage: page});
})()"""

CLICK_NEXT_EXPR = """(function(){
  var next = document.querySelector(".arco-pagination-item-next");
  if (next && !next.classList.contains("arco-pagination-item-disabled")) {
    next.click();
    return "AIME_JSON:" + JSON.stringify("clicked");
  }
  return "AIME_JSON:" + JSON.stringify("no-next");
})()"""

CLICK_SEARCH_EXPR = """(function(){
  var buttons = Array.from(document.querySelectorAll("button"));
  var btn = buttons.find(function(b){ return b.textContent && b.textContent.trim() === "查询"; });
  if (btn) {
    btn.click();
    return "AIME_JSON:" + JSON.stringify("clicked");
  }
  return "AIME_JSON:" + JSON.stringify("not-found");
})()"""


# ===== 核心提取逻辑 =====

def extract_table(driver, is_dda: bool = True, page_size: int = 50) -> List[Dict[str, Any]]:
    """从当前页面提取所有表格数据（自动翻页）"""
    rows_all = []
    expr_tpl = DDA_EXPR_TEMPLATE if is_dda else HSQL_EXPR_TEMPLATE
    page_index = 1

    while True:
        print(f"  提取第 {page_index} 页...", flush=True)
        for start in range(0, page_size, 10):
            end = start + 10
            expr = expr_tpl.replace("__START__", str(start)).replace("__END__", str(end))
            arr = driver.eval_js(expr)
            if arr and isinstance(arr, list):
                rows_all.extend(arr)

        info = driver.eval_js(HAS_NEXT_EXPR)
        if not info or not isinstance(info, dict) or not info.get("hasNext"):
            break

        driver.eval_js(CLICK_NEXT_EXPR)
        time.sleep(3)
        page_index += 1

    return rows_all


def main():
    parser = argparse.ArgumentParser(description="FOP 平台数据采集 - 追光清结算任务")
    parser.add_argument("--org_id", default="ORG240313190058294165643265", help="FOP 组织 ID")
    parser.add_argument("--output_dir", default="./data", help="输出目录")
    parser.add_argument("--driver", choices=["aime", "selenium"], default="aime",
                        help="浏览器驱动类型 (aime=平台内置, selenium=独立运行)")
    parser.add_argument("--browser_toolset_dir", default=None,
                        help="Aime browser toolset 目录路径（仅 aime 驱动时需要）")
    parser.add_argument("--domain", default="fop.bytedance.net",
                        help="FOP 平台域名")
    args = parser.parse_args()

    os.makedirs(args.output_dir, exist_ok=True)

    # 构造 URL
    dda_url = (
        f"https://{args.domain}/boss/check-core/dda/config-new"
        f"?OrganizationId={args.org_id}&PageNo=1&PageSize=50"
    )
    hsql_url = (
        f"https://{args.domain}/check-core/check/hsql/list"
        f"?SystemIds={args.org_id}&TaskStatus=ALL&ScheduleType=Hsql&PageNo=1&PageSize=50"
    )

    # 初始化驱动
    if args.driver == "aime":
        toolset_dir = args.browser_toolset_dir or os.path.join(
            os.path.dirname(os.path.abspath(__file__)),
            "..", "inner_skills", "browser", "scripts", "browser_toolset"
        )
        driver = AimeBrowserDriver(toolset_dir)
    else:
        driver = SeleniumBrowserDriver()

    driver.acquire_session()
    print(f"浏览器会话已建立")

    try:
        # ===== DDA 采集 =====
        print(f"\n[DDA] 导航到: {dda_url}")
        driver.navigate(dda_url)
        driver.wait(8000)
        time.sleep(3)

        try:
            driver.eval_js(CLICK_SEARCH_EXPR)
            time.sleep(3)
        except Exception:
            pass

        print("[DDA] 开始提取...")
        dda_rows = extract_table(driver, is_dda=True)
        print(f"[DDA] 提取完成，共 {len(dda_rows)} 条记录")

        dda_output = os.path.join(args.output_dir, "dda_data.json")
        with open(dda_output, "w", encoding="utf-8") as f:
            json.dump(dda_rows, f, ensure_ascii=False, indent=2)
        print(f"[DDA] 已保存到 {dda_output}")

        # ===== HSQL 采集 =====
        print(f"\n[HSQL] 导航到: {hsql_url}")
        driver.navigate(hsql_url)
        driver.wait(8000)
        time.sleep(3)

        try:
            driver.eval_js(CLICK_SEARCH_EXPR)
            time.sleep(3)
        except Exception:
            pass

        print("[HSQL] 开始提取...")
        hsql_rows = extract_table(driver, is_dda=False)
        print(f"[HSQL] 提取完成，共 {len(hsql_rows)} 条记录")

        hsql_output = os.path.join(args.output_dir, "hsql_data.json")
        with open(hsql_output, "w", encoding="utf-8") as f:
            json.dump(hsql_rows, f, ensure_ascii=False, indent=2)
        print(f"[HSQL] 已保存到 {hsql_output}")

    finally:
        driver.release_session()

    print(f"\n采集完成！DDA: {len(dda_rows)} 条, HSQL: {len(hsql_rows)} 条")


if __name__ == "__main__":
    main()
