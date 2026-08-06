#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
技术方案变更点解析器。

输入：一份已下载到本地的技术方案 Markdown（飞书文档导出的 .lark.md，或任意 .md），
输出：结构化「变更点候选清单」（JSON），每个变更点已解析成可直接喂给影响面查询的目标
（DB 字段 table.field / DB 表 table / 配置项 KEY）。

核心思想：图谱（impact_index.json）就是「支付/结算域实际存在的表、字段、配置」的权威字典。
方案文档里出现的、且能在图谱字典中命中的标识符，绝大多数就是真实的变更/波及对象。
用「图谱字典命中」做主过滤，再用「变更关键词邻近度」打分，可在不依赖大模型的情况下
得到高精度候选，交由上层（Agent 语义理解）复核补全。

用法：
  python3 extract_change_points.py <doc.lark.md> [--graph-dir DIR] [--json]

输出（--json 默认开启，人读时可去掉看摘要）：
  {
    "doc": "...",
    "change_points": [
       {"raw": "channel_code", "type": "field",
        "resolved_targets": ["bytepay_settle_factor_match.channel_code", ...],
        "signal": true, "occurrences": 6, "evidence": ["...新增 channel_code 字段..."]},
       ...
    ],
    "review_candidates": [ ... ],
    "unresolved_hints": [ ... ]
  }
"""
import argparse
import json
import os
import re
import sys
from collections import defaultdict, OrderedDict

CHANGE_SIGNALS = [
    "新增", "增加", "新加", "添加", "扩展", "增补", "补齐", "变更", "修改", "调整",
    "删除", "废弃", "下线", "重构", "支持", "改造", "新口径", "维度", "字段", "列",
    "表结构", "加字段", "加列", "新增维度", "DDL", "alter", "新增字段",
]

STOP_TOKENS = {
    "id", "key", "type", "code", "name", "status", "value", "data", "time",
    "date", "ext", "url", "case", "scene", "mode", "list", "count", "index",
    "true", "false", "null", "int", "string", "bool", "text", "json",
}

FILE_EXTS = {
    "puml", "svg", "png", "jpg", "jpeg", "gif", "md", "json", "html", "htm",
    "csv", "xlsx", "xls", "pdf", "txt", "go", "py", "java", "sql", "yaml", "yml",
}


def resolve_graph_dir(explicit):
    cands = []
    if explicit:
        cands.append(explicit)
    if os.environ.get("CODE_GRAPH_OUTPUT_DIR"):
        cands.append(os.environ["CODE_GRAPH_OUTPUT_DIR"])
    ws = os.environ.get("AIME_WORKSPACE_PATH")
    if ws:
        cands.append(os.path.join(ws, "code_graph_system", "output"))
    cur = os.getcwd()
    for _ in range(8):
        cands.append(os.path.join(cur, "code_graph_system", "output"))
        parent = os.path.dirname(cur)
        if parent == cur:
            break
        cur = parent
    for c in cands:
        if c and os.path.isfile(os.path.join(c, "impact_index.json")):
            return c
    return None


def load_dictionary(graph_dir):
    with open(os.path.join(graph_dir, "impact_index.json"), "r", encoding="utf-8") as f:
        idx = json.load(f)
    col2keys = defaultdict(list)
    tables = {}
    field_keys = {}
    configs = {}
    entry_count = {}
    for k, v in idx.items():
        n = len(v.get("entries", []))
        if k.startswith("db:"):
            body = k[3:]
            if "." in body:
                table, col = body.split(".", 1)
                col2keys[col.lower()].append(body)
                tables.setdefault(table.lower(), table)
                field_keys[body.lower()] = body
                entry_count[body] = n
        elif k.startswith("cfg:"):
            key = k[4:]
            configs[key.lower()] = key
            entry_count[key] = n
    return {
        "col2keys": col2keys,
        "tables": tables,
        "field_keys": field_keys,
        "configs": configs,
        "entry_count": entry_count,
    }


def clean_text(raw):
    raw = re.sub(r"<!--.*?-->", " ", raw, flags=re.S)
    raw = re.sub(r"<[^>]+>", " ", raw)
    lines = [ln.strip() for ln in raw.splitlines()]
    return [ln for ln in lines if ln]


TOKEN_RE = re.compile(r"[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)?")


def has_signal(line):
    return any(s.lower() in line.lower() for s in CHANGE_SIGNALS)


def snippet(line, maxlen=90):
    line = line.strip()
    return line if len(line) <= maxlen else line[:maxlen] + "…"


def extract(doc_path, graph_dir):
    with open(doc_path, "r", encoding="utf-8") as f:
        raw = f.read()
    lines = clean_text(raw)
    d = load_dictionary(graph_dir)

    hits = OrderedDict()

    def note(token, line, signal):
        rec = hits.setdefault(token, {"occ": 0, "signal": False, "evidence": []})
        rec["occ"] += 1
        if signal:
            rec["signal"] = True
            if len(rec["evidence"]) < 3:
                rec["evidence"].append(snippet(line))
        elif not rec["evidence"] and len(rec["evidence"]) < 3:
            rec["evidence"].append(snippet(line))

    for line in lines:
        sig = has_signal(line)
        for m in TOKEN_RE.finditer(line):
            tok = m.group(0)
            low = tok.lower()
            if "." in low:
                if low.rsplit(".", 1)[1] in FILE_EXTS:
                    continue
                if low in d["field_keys"] or low.split(".", 1)[0] in d["tables"] or low.split(".", 1)[1] in d["col2keys"]:
                    note(tok, line, sig)
                elif re.search(r"[a-z]+_[a-z]", low):
                    note(tok, line, sig)
            else:
                if low in STOP_TOKENS:
                    continue
                if low in d["tables"] or low in d["col2keys"] or low in d["configs"]:
                    note(tok, line, sig)

    change_points = []
    review = []
    unresolved = []

    for tok, rec in hits.items():
        low = tok.lower()
        cp = {"raw": tok, "occurrences": rec["occ"], "signal": rec["signal"], "evidence": rec["evidence"]}
        if "." in low:
            if low in d["field_keys"]:
                cp["type"] = "field"
                cp["resolved_targets"] = [d["field_keys"][low]]
            elif low.split(".", 1)[0] in d["tables"]:
                cp["type"] = "table"
                cp["resolved_targets"] = [d["tables"][low.split(".", 1)[0]]]
            else:
                cp["type"] = "dataset"
                cp["resolved_targets"] = []
                unresolved.append(cp)
                continue
        elif low in d["configs"]:
            cp["type"] = "config"
            cp["resolved_targets"] = [d["configs"][low]]
        elif low in d["tables"]:
            cp["type"] = "table"
            cp["resolved_targets"] = [d["tables"][low]]
        elif low in d["col2keys"]:
            cp["type"] = "field"
            cp["resolved_targets"] = sorted(d["col2keys"][low])
        else:
            continue

        cp["entry_hint"] = sum(d["entry_count"].get(t, 0) for t in cp["resolved_targets"])

        if rec["signal"]:
            change_points.append(cp)
        else:
            review.append(cp)

    change_points.sort(key=lambda x: (-x["entry_hint"], x["raw"]))
    review.sort(key=lambda x: (-x["entry_hint"], x["raw"]))

    return {
        "doc": doc_path,
        "graph_dir": graph_dir,
        "change_points": change_points,
        "review_candidates": review,
        "unresolved_hints": unresolved,
    }


def render(result):
    out = []
    out.append("# 技术方案变更点解析\n")
    out.append(f"> 文档：`{result['doc']}`\n")
    cps = result["change_points"]
    out.append(f"\n## ✅ 变更点候选（含变更信号，共 {len(cps)} 项）\n")
    if not cps:
        out.append("_未在方案中命中带变更信号的图谱对象。_")
    for cp in cps:
        tgts = ", ".join(f"`{t}`" for t in cp["resolved_targets"]) or "-"
        out.append(f"- **{cp['raw']}**（{cp['type']}，预估入口 {cp['entry_hint']}）→ {tgts}")
        for ev in cp["evidence"]:
            out.append(f"    - 依据：{ev}")
    rc = result["review_candidates"]
    out.append(f"\n## 🔎 待复核（命中图谱但无明确变更信号，共 {len(rc)} 项）\n")
    for cp in rc[:30]:
        tgts = ", ".join(f"`{t}`" for t in cp["resolved_targets"]) or "-"
        out.append(f"- {cp['raw']}（{cp['type']}）→ {tgts}")
    uh = result["unresolved_hints"]
    if uh:
        out.append(f"\n## ⚠️ 疑似数据集/未建模对象（需人工排查，共 {len(uh)} 项）\n")
        for cp in uh[:30]:
            out.append(f"- `{cp['raw']}`：{cp['evidence'][0] if cp['evidence'] else ''}")
    return "\n".join(out)


def main():
    ap = argparse.ArgumentParser(description="技术方案变更点解析")
    ap.add_argument("doc", help="本地技术方案 markdown 文件路径（飞书导出的 .lark.md）")
    ap.add_argument("--graph-dir", help="包含 impact_index.json 的目录")
    ap.add_argument("--out", help="将结构化 JSON 写入该文件")
    ap.add_argument("--md", action="store_true", help="额外打印人读 Markdown 摘要")
    args = ap.parse_args()

    gd = resolve_graph_dir(args.graph_dir)
    if not gd:
        print("❌ 未找到图谱数据 impact_index.json，请用 --graph-dir 指定 code_graph_system/output", file=sys.stderr)
        sys.exit(2)
    if not os.path.isfile(args.doc):
        print(f"❌ 文档不存在：{args.doc}", file=sys.stderr)
        sys.exit(2)

    result = extract(args.doc, gd)
    if args.out:
        with open(args.out, "w", encoding="utf-8") as f:
            json.dump(result, f, ensure_ascii=False, indent=2)
        print(f"✅ 变更点已写入：{args.out}（变更点 {len(result['change_points'])}，待复核 {len(result['review_candidates'])}）")
    if args.md or not args.out:
        print(render(result))


if __name__ == "__main__":
    main()
