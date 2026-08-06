#!/usr/bin/env python3

import argparse
import json
import os
import re
import sys
import urllib.request
import urllib.error

_RAG_API_URL_MAP = {
    "boe": "https://bytest.bytedance.net/ut_proxy/assert_api/knowledge-process-agent/spec_search",
    "boei18n": "https://bytest-i18n-bd.byteintl.net/ut_proxy/assert_api/knowledge-process-agent/spec_search",
}
RAG_API_URL = _RAG_API_URL_MAP.get(
    os.environ.get("RUNS_ON_ENV", ""),
    "http://assert.byted.org/knowledge-process-agent/spec_search",
)
DEFAULT_TOP_K = 3
DEFAULT_CALLER = "ut"
TIMEOUT_SECONDS = 15

DESC_PREFIX_RE = re.compile(r'^description:\s*"[^"]*"\s*\n?', re.MULTILINE)


def build_request_body(query, language, top_k, caller):
    return {
        "query": query,
        "language": language,
        "top_k": top_k,
        "caller": caller,
        #"debug": True, # 只有在线下 debug 或回归时开启 debug 模式
        "custom": True,
        "extra_pack_paths": [f"Official/Quality/UnitTest/{language}", "Official/Quality/UnitTest/general"]
    }


def extract_title(content):
    m = re.match(r'^description:\s*"([^"]*)"', content)
    if m:
        return m.group(1)
    first_line = content.split("\n", 1)[0].strip()
    return first_line[:120] if first_line else "Untitled"


def clean_content(content):
    return DESC_PREFIX_RE.sub("", content, count=1).strip()


def format_markdown(recalls):
    if not recalls:
        return "RAG 查询无结果。"

    parts = []
    for i, item in enumerate(recalls, 1):
        raw_content = item.get("content", "")
        score = item.get("final_score", 0.0)
        title = extract_title(raw_content)
        body = clean_content(raw_content)

        parts.append(f"### [{i}] (score: {score:.2f}) {title}\n\n{body}")

    return "\n\n---\n\n".join(parts) + "\n"


def do_query(query, language, top_k, caller):
    body = build_request_body(query, language, top_k, caller)
    data = json.dumps(body).encode("utf-8")

    req = urllib.request.Request(
        RAG_API_URL,
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT_SECONDS) as resp:
            raw = resp.read().decode("utf-8")
            raw_response = json.loads(raw)
    except urllib.error.HTTPError as e:
        error_body = ""
        try:
            error_body = e.read().decode("utf-8")
        except Exception:
            pass
        print(f"RAG 查询失败: HTTP {e.code}: {error_body}", file=sys.stderr)
        return None
    except urllib.error.URLError as e:
        print(f"RAG 查询失败: URL error: {e.reason}", file=sys.stderr)
        return None
    except Exception as e:
        print(f"RAG 查询失败: {e}", file=sys.stderr)
        return None

    if not isinstance(raw_response, dict):
        print("RAG 查询失败: unexpected response format", file=sys.stderr)
        return None

    code = raw_response.get("code", -1)
    if code != 200:
        print(f"RAG 查询失败: API returned code {code}: {raw_response.get('message', '')}", file=sys.stderr)
        return None

    recalls = raw_response.get("data", {}).get("recalls", [])
    return format_markdown(recalls)


def main():
    parser = argparse.ArgumentParser(description="RAG knowledge retrieval for unit test generation")
    parser.add_argument("--query", required=True, help="Natural language query describing the knowledge needed")
    parser.add_argument("--language", required=True, help="Programming language identifier (go, python, java, javascript, kotlin, cpp, swift, rust)")
    parser.add_argument("--top-k", type=int, default=DEFAULT_TOP_K, help=f"Number of results to return (default: {DEFAULT_TOP_K})")
    parser.add_argument("--caller", default=DEFAULT_CALLER, help=f"Caller identifier (default: {DEFAULT_CALLER})")
    args = parser.parse_args()

    result = do_query(args.query, args.language, args.top_k, args.caller)
    if result is not None:
        print(result)
    else:
        sys.exit(1)


if __name__ == "__main__":
    main()
