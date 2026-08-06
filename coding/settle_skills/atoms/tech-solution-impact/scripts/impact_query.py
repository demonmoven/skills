#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
影响面查询引擎：给定改动的 DB 字段 / DB 表 / 配置项，输出受影响的功能入口（Kitex/MQ）及调用链。

数据来源：支付域代码依赖图谱
  - graph.json         全量节点 + 边（用于回溯调用链）
  - impact_index.json  预计算的「DB字段/配置项 -> 受影响入口」倒排索引（权威入口清单）

节点类型：Entry / Method / DBField / DBTable / Config / RPCService / ProtoField / Repo
边类型：  calls / db_reads / db_writes / reads / contains / exposes / rpc_calls / db_shared

用法示例：
  python3 impact_query.py bytepay_charge_contract.scene
  python3 impact_query.py bytepay_charge_contract SEND_LARK_MSG_ALERT_LIST
  python3 impact_query.py --graph-dir /path/to/output --json bytepay_settle_rule.merchant_id
"""
import argparse
import json
import os
import sys
from collections import defaultdict, deque


def resolve_graph_dir(explicit):
    candidates = []
    if explicit:
        candidates.append(explicit)
    if os.environ.get("CODE_GRAPH_OUTPUT_DIR"):
        candidates.append(os.environ["CODE_GRAPH_OUTPUT_DIR"])
    ws = os.environ.get("AIME_WORKSPACE_PATH")
    if ws:
        candidates.append(os.path.join(ws, "code_graph_system", "output"))
    cur = os.getcwd()
    for _ in range(8):
        candidates.append(os.path.join(cur, "code_graph_system", "output"))
        parent = os.path.dirname(cur)
        if parent == cur:
            break
        cur = parent
    for c in candidates:
        if c and os.path.isfile(os.path.join(c, "impact_index.json")):
            return c
    return None


class ImpactGraph:
    def __init__(self, graph_dir):
        self.dir = graph_dir
        self.index = self._load("impact_index.json")
        self.graph = None
        self.nodes = {}
        self.calls_rev = defaultdict(list)
        self.entry_of = defaultdict(list)
        self.access = defaultdict(list)
        self._graph_loaded = False

    def _load(self, name):
        path = os.path.join(self.dir, name)
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)

    def load_graph(self):
        if self._graph_loaded:
            return
        self.graph = self._load("graph.json")
        self.nodes = {n["id"]: n for n in self.graph["nodes"]}
        for e in self.graph["edges"]:
            t = e["type"]
            if t == "calls":
                self.calls_rev[e["dst"]].append(e["src"])
            elif t == "contains" and e["src"].startswith("entry:"):
                self.entry_of[e["dst"]].append(e["src"])
            elif t in ("db_reads", "db_writes", "reads"):
                self.access[e["dst"]].append((e["src"], t))
        self._graph_loaded = True

    def classify(self, raw):
        raw = raw.strip()
        cfg_key = "cfg:" + raw
        if cfg_key in self.index:
            return ("config", [cfg_key], None)
        low = raw.lower()
        for k in self.index:
            if k.startswith("cfg:") and k[4:].lower() == low:
                return ("config", [k], None)
        if "." in raw:
            fkey = "db:" + raw
            if fkey in self.index:
                return ("field", [fkey], None)
            for k in self.index:
                if k.startswith("db:") and k[3:].lower() == low:
                    return ("field", [k], None)
            return ("unknown", [], self._suggest(raw))
        prefix = "db:" + raw + "."
        table_keys = [k for k in self.index if k.startswith(prefix)]
        if not table_keys:
            plow = ("db:" + raw + ".").lower()
            table_keys = [k for k in self.index if k.lower().startswith(plow)]
        if table_keys:
            return ("table", sorted(table_keys), None)
        return ("unknown", [], self._suggest(raw))

    def _suggest(self, raw):
        low = raw.lower()
        hits = [k for k in self.index if low in k.lower()]
        return sorted(hits)[:15]

    def entries_for(self, matched_keys):
        s = set()
        for k in matched_keys:
            for e in self.index[k].get("entries", []):
                s.add(e)
        return sorted(s)

    def _target_node_ids(self, kind, matched_keys):
        ids = set()
        for k in matched_keys:
            if k.startswith("cfg:"):
                ids.add(k)
            elif k.startswith("db:"):
                name = k[3:]
                table = name.split(".")[0]
                ids.add("db:" + table)
        return ids

    def chains(self, kind, matched_keys, want_entries, max_chains_per_entry=1, max_depth=18):
        self.load_graph()
        targets = self._target_node_ids(kind, matched_keys)
        want = set(want_entries)
        result = defaultdict(list)
        seen_entry_seed = defaultdict(set)
        for tgt in targets:
            for method, acc in self.access.get(tgt, []):
                q = deque([(method, [method])])
                visited = {method}
                while q:
                    cur, path = q.popleft()
                    for ent in self.entry_of.get(cur, []):
                        if ent in want and method not in seen_entry_seed[ent]:
                            if len(result[ent]) < max_chains_per_entry:
                                chain = [ent] + path[::-1]
                                result[ent].append({"access": acc, "target": tgt, "chain": chain})
                                seen_entry_seed[ent].add(method)
                    if len(path) >= max_depth:
                        continue
                    for pred in self.calls_rev.get(cur, []):
                        if pred not in visited:
                            visited.add(pred)
                            q.append((pred, path + [pred]))
        return result

    def entry_meta(self, entry_id):
        n = self.nodes.get(entry_id) if self._graph_loaded else None
        if not n:
            name = entry_id.split("::")[-1]
            return {"name": name, "kind": "", "psm": "", "service": "", "repo": ""}
        m = n.get("meta", {})
        return {
            "name": n.get("name", ""),
            "kind": m.get("kind", ""),
            "psm": m.get("psm", ""),
            "service": m.get("service", ""),
            "repo": n.get("repo", ""),
        }

    def method_short(self, node_id):
        if node_id.startswith("entry:"):
            return "【入口】" + node_id.split("::")[-1]
        if node_id.startswith("go:"):
            body = node_id[3:]
            return body.split("/")[-1]
        if node_id.startswith("db:"):
            return "【表】" + node_id[3:]
        if node_id.startswith("cfg:"):
            return "【配置】" + node_id[4:]
        return node_id

    def shared_note(self, kind, matched_keys):
        if not self._graph_loaded:
            return []
        notes = []
        for k in matched_keys:
            if not k.startswith("db:"):
                continue
            table = k[3:].split(".")[0]
            tn = self.nodes.get("db:" + table)
            if tn and tn.get("meta", {}).get("shared_by"):
                notes.append((table, tn["meta"]["shared_by"]))
        seen = set()
        uniq = []
        for t, s in notes:
            if t not in seen:
                seen.add(t)
                uniq.append((t, s))
        return uniq


def run(targets, graph_dir, max_chains, as_json, no_chains):
    gd = resolve_graph_dir(graph_dir)
    if not gd:
        msg = "未找到图谱数据（impact_index.json）。请用 --graph-dir 指定 code_graph_system/output 目录，或设置环境变量 CODE_GRAPH_OUTPUT_DIR。"
        if as_json:
            print(json.dumps({"error": msg}, ensure_ascii=False))
        else:
            print("❌ " + msg)
        return 2

    g = ImpactGraph(gd)
    g.load_graph()
    report = {"graph_dir": gd, "targets": []}

    for raw in targets:
        kind, keys, suggest = g.classify(raw)
        item = {"input": raw, "kind": kind, "matched_keys": keys}
        if kind == "unknown":
            item["suggestions"] = suggest
            report["targets"].append(item)
            continue
        entries = g.entries_for(keys)
        item["entry_count"] = len(entries)
        chains = {}
        if not no_chains:
            chains = g.chains(kind, keys, entries, max_chains_per_entry=max_chains)
        entry_list = []
        repos = set()
        for e in entries:
            meta = g.entry_meta(e)
            repos.add(meta["repo"])
            ce = chains.get(e, [])
            entry_list.append({
                "id": e,
                "name": meta["name"],
                "kind": meta["kind"],
                "psm": meta["psm"],
                "service": meta["service"],
                "repo": meta["repo"],
                "chains": [{"access": c["access"], "target": c["target"], "chain": c["chain"]} for c in ce],
            })
        item["repos"] = sorted(r for r in repos if r)
        item["entries"] = entry_list
        item["shared"] = g.shared_note(kind, keys)
        report["targets"].append(item)

    if as_json:
        print(json.dumps(report, ensure_ascii=False, indent=2))
    else:
        print(render_markdown(report, g))
    return 0


def render_markdown(report, g):
    out = []
    out.append("# 影响面查询结果\n")
    out.append(f"> 数据目录：`{report['graph_dir']}`\n")
    for item in report["targets"]:
        out.append(f"\n## 🎯 改动项：`{item['input']}`\n")
        if item["kind"] == "unknown":
            out.append("**未在图谱中命中该改动项。**\n")
            if item.get("suggestions"):
                out.append("可能想找的是：")
                for s in item["suggestions"]:
                    out.append(f"- `{s}`")
            else:
                out.append("图谱中无相近条目，请确认表名/字段名/配置 key 是否正确。")
            continue
        kind_cn = {"config": "配置项", "field": "DB 字段", "table": "DB 表"}[item["kind"]]
        out.append(f"- 类型：**{kind_cn}**")
        if item["kind"] == "table":
            out.append(f"- 命中该表下 {len(item['matched_keys'])} 个被使用字段")
        out.append(f"- 受影响功能入口：**{item['entry_count']}** 个")
        if item.get("repos"):
            out.append(f"- 涉及仓库：{', '.join('`' + r + '`' for r in item['repos'])}")
        if item.get("shared"):
            out.append("- ⚠️ **跨仓库共享**：")
            for t, s in item["shared"]:
                out.append(f"  - `{t}` 被多个仓库共享：{s}")
        if item["entry_count"] == 0:
            out.append("\n无功能入口读写该改动项（可能仅内部使用或未被入口链路覆盖）。")
            continue
        out.append("\n### 受影响功能入口\n")
        out.append("| 入口 | 类型 | 服务/PSM | 仓库 |")
        out.append("|------|------|----------|------|")
        for e in item["entries"]:
            kind_tag = {"kitex": "Kitex RPC", "mq": "MQ 消费"}.get(e["kind"], e["kind"] or "-")
            svc = e["service"] or ""
            psm = e["psm"] or ""
            svc_psm = (svc + "<br>" + psm).strip("<br>") if (svc or psm) else "-"
            out.append(f"| `{e['name']}` | {kind_tag} | {svc_psm} | {e['repo']} |")
        chains_exist = any(e["chains"] for e in item["entries"])
        if chains_exist:
            out.append("\n### 调用链（入口 → 数据/配置访问点）\n")
            for e in item["entries"]:
                if not e["chains"]:
                    continue
                for c in e["chains"]:
                    acc_cn = {"db_reads": "读", "db_writes": "写", "reads": "读配置"}.get(c["access"], c["access"])
                    tgt = g.method_short(c["target"])
                    out.append(f"- **{e['name']}** （{acc_cn} {tgt}）")
                    steps = " → ".join(g.method_short(x) for x in c["chain"])
                    out.append(f"  - `{steps}`")
    return "\n".join(out)


def main():
    ap = argparse.ArgumentParser(description="支付域改动影响面查询")
    ap.add_argument("targets", nargs="+", help="改动项：DB字段(table.field) / DB表(table) / 配置项(KEY)，可多个")
    ap.add_argument("--graph-dir", help="包含 graph.json / impact_index.json 的目录")
    ap.add_argument("--max-chains", type=int, default=1, help="每个入口展示的调用链数量（默认 1）")
    ap.add_argument("--json", action="store_true", help="输出 JSON")
    ap.add_argument("--no-chains", action="store_true", help="只列入口，不计算调用链（更快）")
    args = ap.parse_args()
    sys.exit(run(args.targets, args.graph_dir, args.max_chains, args.json, args.no_chains))


if __name__ == "__main__":
    main()
