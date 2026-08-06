#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
技术方案影响面聚合器。

串联「变更点解析」与「影响面查询」，把多个变更点的影响面结果聚合成一份全量报告：
  - 变更点 → 受影响入口的映射（正向）
  - 受影响入口 → 命中它的变更点（反向，用于定位高危入口）
  - 跨仓库风险汇总（共享表 / 入口跨 PSM）
  - 回归建议优先级（P0/P1/P2）

两种运行方式（结果一致，二选一）：
  1) 内置引擎（默认，自包含）：直接读取 code_graph_system/output 下的
     impact_index.json（权威入口倒排索引）与 graph.json（入口元信息 / 共享标记），
     算法与同目录 impact_query.py 一致（同源实现，2026-07 同步）。
  2) 复用外部结果：先用同目录 impact_query.py 对
     `--list-targets` 打印出的目标批量查询并 `--json` 落盘，再用 `--impact-json` 传入本脚本聚合。

用法：
  # 打印去重后的查询目标（可交给 impact_query.py 批量查询）
  python3 analyze_impact.py --change-points cp.json --list-targets

  # 一步到位（内置引擎）：解析结果 -> 聚合报告
  python3 analyze_impact.py --change-points cp.json --out-md report_body.md --out-json agg.json

  # 复用外部 impact_query.py 的 --json 输出做聚合
  python3 analyze_impact.py --change-points cp.json --impact-json raw.json --out-md report_body.md
"""
import argparse
import json
import os
import sys
from collections import defaultdict, OrderedDict


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
        p = os.path.dirname(cur)
        if p == cur:
            break
        cur = p
    for c in cands:
        if c and os.path.isfile(os.path.join(c, "impact_index.json")):
            return c
    return None


class Engine:
    def __init__(self, graph_dir):
        with open(os.path.join(graph_dir, "impact_index.json"), "r", encoding="utf-8") as f:
            self.index = json.load(f)
        self.nodes = {}
        self._graph_loaded = False
        self.graph_dir = graph_dir

    def load_graph(self):
        if self._graph_loaded:
            return
        with open(os.path.join(self.graph_dir, "graph.json"), "r", encoding="utf-8") as f:
            g = json.load(f)
        self.nodes = {n["id"]: n for n in g["nodes"]}
        self._graph_loaded = True

    def classify(self, raw):
        raw = raw.strip()
        low = raw.lower()
        cfg_key = "cfg:" + raw
        if cfg_key in self.index:
            return "config", [cfg_key]
        for k in self.index:
            if k.startswith("cfg:") and k[4:].lower() == low:
                return "config", [k]
        if "." in raw:
            fkey = "db:" + raw
            if fkey in self.index:
                return "field", [fkey]
            for k in self.index:
                if k.startswith("db:") and k[3:].lower() == low:
                    return "field", [k]
            return "unknown", []
        prefix = ("db:" + raw + ".").lower()
        table_keys = [k for k in self.index if k.lower().startswith(prefix)]
        if table_keys:
            return "table", sorted(table_keys)
        return "unknown", []

    def entries_for(self, keys):
        s = set()
        for k in keys:
            for e in self.index[k].get("entries", []):
                s.add(e)
        return sorted(s)

    def entry_meta(self, eid):
        self.load_graph()
        n = self.nodes.get(eid)
        if not n:
            return {"name": eid.split("::")[-1], "kind": "", "psm": "", "service": "", "repo": ""}
        m = n.get("meta", {})
        return {
            "name": n.get("name", ""),
            "kind": m.get("kind", ""),
            "psm": m.get("psm", ""),
            "service": m.get("service", ""),
            "repo": n.get("repo", ""),
        }

    def shared_note(self, keys):
        self.load_graph()
        seen, uniq = set(), []
        for k in keys:
            if not k.startswith("db:"):
                continue
            table = k[3:].split(".")[0]
            tn = self.nodes.get("db:" + table)
            if tn and tn.get("meta", {}).get("shared_by") and table not in seen:
                seen.add(table)
                uniq.append([table, tn["meta"]["shared_by"]])
        return uniq

    def query(self, target):
        kind, keys = self.classify(target)
        if kind == "unknown":
            return {"input": target, "kind": "unknown", "matched_keys": [], "entry_count": 0, "entries": [], "repos": [], "shared": []}
        entries = self.entries_for(keys)
        repos, elist = set(), []
        for e in entries:
            m = self.entry_meta(e)
            repos.add(m["repo"])
            elist.append({"id": e, **m})
        return {
            "input": target,
            "kind": kind,
            "matched_keys": keys,
            "entry_count": len(entries),
            "entries": elist,
            "repos": sorted(r for r in repos if r),
            "shared": self.shared_note(keys),
        }


def collect_targets(cp_data, include_review):
    target2cp = OrderedDict()
    cp_meta = {}
    groups = list(cp_data.get("change_points", []))
    if include_review:
        groups += list(cp_data.get("review_candidates", []))
    for cp in groups:
        raw = cp["raw"]
        cp_meta[raw] = {"type": cp.get("type"), "evidence": cp.get("evidence", []), "signal": cp.get("signal", False)}
        for t in cp.get("resolved_targets", []):
            target2cp.setdefault(t, set()).add(raw)
    return target2cp, cp_meta


def build_impact_map(targets, impact_json, graph_dir):
    result = {}
    if impact_json:
        with open(impact_json, "r", encoding="utf-8") as f:
            raw = json.load(f)
        for item in raw.get("targets", []):
            result[item["input"]] = item
    missing = [t for t in targets if t not in result]
    if missing:
        eng = Engine(graph_dir)
        for t in missing:
            result[t] = eng.query(t)
    return result


def aggregate(cp_data, impact_json, graph_dir, include_review):
    target2cp, cp_meta = collect_targets(cp_data, include_review)
    targets = list(target2cp.keys())
    impact = build_impact_map(targets, impact_json, graph_dir)

    entry_cps = defaultdict(set)
    entry_info = {}
    cp_agg = defaultdict(lambda: {"entries": set(), "repos": set(), "shared": []})
    shared_tables = {}

    for t, item in impact.items():
        cps = target2cp.get(t, set())
        for e in item.get("entries", []):
            eid = e["id"]
            entry_cps[eid].update(cps)
            entry_info[eid] = e
        for r in item.get("repos", []):
            for cp in cps:
                cp_agg[cp]["repos"].add(r)
        for eid in [e["id"] for e in item.get("entries", [])]:
            for cp in cps:
                cp_agg[cp]["entries"].add(eid)
        for tbl, by in item.get("shared", []):
            shared_tables[tbl] = by
            for cp in cps:
                if [tbl, by] not in cp_agg[cp]["shared"]:
                    cp_agg[cp]["shared"].append([tbl, by])

    all_entries = sorted(entry_cps.keys())
    all_repos = sorted({entry_info[e]["repo"] for e in all_entries if entry_info[e].get("repo")})

    shared_repo_names = set()
    for by in shared_tables.values():
        for r in (by if isinstance(by, list) else [by]):
            shared_repo_names.add(r)

    entry_rows = []
    for eid in all_entries:
        info = entry_info[eid]
        hit = sorted(entry_cps[eid])
        in_shared = info.get("repo") in shared_repo_names and bool(shared_tables)
        score = len(hit) * 2 + (2 if in_shared else 0) + (1 if info.get("kind") == "kitex" else 0)
        if len(hit) >= 3 or (in_shared and len(hit) >= 2):
            prio = "P0"
        elif len(hit) >= 2 or in_shared:
            prio = "P1"
        else:
            prio = "P2"
        entry_rows.append({
            "id": eid,
            "name": info.get("name"),
            "kind": info.get("kind"),
            "psm": info.get("psm"),
            "service": info.get("service"),
            "repo": info.get("repo"),
            "hit_by": hit,
            "hit_count": len(hit),
            "priority": prio,
            "score": score,
        })
    prio_rank = {"P0": 0, "P1": 1, "P2": 2}
    entry_rows.sort(key=lambda x: (prio_rank[x["priority"]], -x["score"], -x["hit_count"], x["name"] or ""))

    cp_rows = []
    unresolved = []
    for cp_raw, meta in cp_meta.items():
        agg = cp_agg.get(cp_raw, {"entries": set(), "repos": set(), "shared": []})
        resolved = []
        for cp in cp_data.get("change_points", []) + cp_data.get("review_candidates", []):
            if cp["raw"] == cp_raw:
                resolved = cp.get("resolved_targets", [])
                break
        cp_rows.append({
            "raw": cp_raw,
            "type": meta["type"],
            "signal": meta["signal"],
            "resolved_targets": resolved,
            "entry_count": len(agg["entries"]),
            "repos": sorted(agg["repos"]),
            "shared": agg["shared"],
            "evidence": meta["evidence"],
        })
    cp_rows.sort(key=lambda x: (0 if x["signal"] else 1, -x["entry_count"], x["raw"]))

    for cp in cp_data.get("unresolved_hints", []):
        unresolved.append({"raw": cp["raw"], "type": cp.get("type"), "evidence": cp.get("evidence", [])})

    return {
        "doc": cp_data.get("doc"),
        "summary": {
            "change_point_count": len([c for c in cp_rows if c["signal"]]),
            "review_count": len([c for c in cp_rows if not c["signal"]]),
            "total_entries": len(all_entries),
            "total_repos": len(all_repos),
            "repos": all_repos,
            "p0": len([e for e in entry_rows if e["priority"] == "P0"]),
            "p1": len([e for e in entry_rows if e["priority"] == "P1"]),
            "p2": len([e for e in entry_rows if e["priority"] == "P2"]),
            "shared_tables": [{"table": t, "shared_by": b} for t, b in shared_tables.items()],
        },
        "change_points": cp_rows,
        "entries": entry_rows,
        "unresolved": unresolved,
    }


def kind_cn(k):
    return {"kitex": "Kitex RPC", "mq": "MQ 消费", "field": "DB 字段", "table": "DB 表", "config": "配置项"}.get(k, k or "-")


def render_md(agg):
    s = agg["summary"]
    o = []
    o.append(f"> 变更点 {s['change_point_count']} 个 · 待复核 {s['review_count']} 个 · 受影响入口 {s['total_entries']} 个 · 涉及仓库 {s['total_repos']} 个 · 回归优先级 P0/P1/P2 = {s['p0']}/{s['p1']}/{s['p2']}\n")

    if s["shared_tables"]:
        o.append("\n## 跨仓库风险汇总\n")
        o.append("以下表被多个仓库共享读写，改动需同步评估所有相关仓库，最易漏改：\n")
        for st in s["shared_tables"]:
            by = st["shared_by"]
            by = "、".join(by) if isinstance(by, list) else str(by)
            o.append(f"- `{st['table']}` ← 共享方：{by}")

    o.append("\n## 变更点 → 受影响入口\n")
    o.append("| 变更点 | 类型 | 受影响入口 | 涉及仓库 | 跨仓共享 |")
    o.append("|--------|------|-----------|----------|----------|")
    for c in agg["change_points"]:
        if not c["signal"]:
            continue
        repos = "、".join(c["repos"]) or "-"
        shared = "⚠️ " + "、".join(t for t, _ in c["shared"]) if c["shared"] else "-"
        o.append(f"| `{c['raw']}` | {kind_cn(c['type'])} | {c['entry_count']} | {repos} | {shared} |")

    review = [c for c in agg["change_points"] if not c["signal"]]
    if review:
        o.append("\n**待复核对象**（命中图谱但方案未明确标注变更，请人工确认是否纳入）：")
        for c in review:
            o.append(f"- `{c['raw']}`（{kind_cn(c['type'])}，入口 {c['entry_count']}）")

    o.append("\n## 回归建议优先级\n")
    o.append("按「被多少变更点命中 + 是否跨仓共享」排序，P0 最需优先回归：\n")
    o.append("| 优先级 | 入口 | 类型 | 服务/PSM | 仓库 | 命中变更点 |")
    o.append("|--------|------|------|----------|------|-----------|")
    for e in agg["entries"]:
        svc = e.get("service") or e.get("psm") or "-"
        hit = "、".join(e["hit_by"])
        o.append(f"| {e['priority']} | `{e['name']}` | {kind_cn(e['kind'])} | {svc} | {e.get('repo') or '-'} | {hit} |")

    if agg["unresolved"]:
        o.append("\n## 图谱未覆盖 / 需人工排查\n")
        o.append("以下对象疑似离线数据集或图谱未建模，影响面无法自动查询，请人工评估：\n")
        for u in agg["unresolved"]:
            ev = u["evidence"][0] if u["evidence"] else ""
            o.append(f"- `{u['raw']}`：{ev}")
    return "\n".join(o)


def main():
    ap = argparse.ArgumentParser(description="技术方案影响面聚合")
    ap.add_argument("--change-points", required=True, help="extract_change_points.py 产出的 JSON")
    ap.add_argument("--impact-json", help="可选：impact_query.py --json 的输出，用于复用外部查询结果")
    ap.add_argument("--graph-dir", help="包含 graph.json / impact_index.json 的目录（内置引擎用）")
    ap.add_argument("--include-review", action="store_true", help="把「待复核」候选也纳入查询与聚合（默认只算带变更信号的）")
    ap.add_argument("--list-targets", action="store_true", help="仅打印去重后的查询目标（每行一个），供 impact_query.py 批量查询")
    ap.add_argument("--out-md", help="报告正文（Markdown）输出路径")
    ap.add_argument("--out-json", help="聚合结果 JSON 输出路径")
    args = ap.parse_args()

    with open(args.change_points, "r", encoding="utf-8") as f:
        cp_data = json.load(f)

    if args.list_targets:
        target2cp, _ = collect_targets(cp_data, args.include_review)
        print("\n".join(target2cp.keys()))
        return

    graph_dir = None
    if not args.impact_json:
        graph_dir = resolve_graph_dir(args.graph_dir)
        if not graph_dir:
            print("❌ 未找到图谱数据，请用 --graph-dir 指定，或提供 --impact-json", file=sys.stderr)
            sys.exit(2)

    agg = aggregate(cp_data, args.impact_json, graph_dir, args.include_review)
    md = render_md(agg)

    if args.out_json:
        with open(args.out_json, "w", encoding="utf-8") as f:
            json.dump(agg, f, ensure_ascii=False, indent=2)
    if args.out_md:
        with open(args.out_md, "w", encoding="utf-8") as f:
            f.write(md)
    if not args.out_md and not args.out_json:
        print(md)
    else:
        s = agg["summary"]
        print(f"✅ 聚合完成：变更点 {s['change_point_count']}、受影响入口 {s['total_entries']}、仓库 {s['total_repos']}、P0/P1/P2 = {s['p0']}/{s['p1']}/{s['p2']}")
        if args.out_md:
            print(f"   报告正文 → {args.out_md}")
        if args.out_json:
            print(f"   聚合 JSON → {args.out_json}")


if __name__ == "__main__":
    main()
