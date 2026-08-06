#!/usr/bin/env python3
"""从 Bits 开发任务(develop flow)链接中提取：需求(Meego/PRD) 与 关联代码仓库+分支。

依赖已安装的 `bytedcli`（bits 子命令走 SSO 鉴权；meego 子命令需 `bytedcli meego login`）。

用法:
    python3 extract_from_bits.py --bits-url "<url>" [--output-dir output]
    python3 extract_from_bits.py --dev-id 2540374 [--output-dir output]

输出到 output_dir:
- bits_context.json  结构化上下文（title / 需求 / 仓库分支绑定 / PRD 提取情况）
- repos.json         直接可喂给 fetch_repo_diffs.py 的仓库配置

说明:
- 仓库与分支来自 `bytedcli bits develop inspect-changes`（实际绑定 MR 的 code-change 卡片，最可靠）。
- diff 的 base 分支取每个绑定的 target_branch（通常 master）。
- PRD: 先从 develop get 拿到 Meego 需求 URL，再尝试 `bytedcli meego workitem get` 抽取需求里的 PRD/文档飞书链接；
  若 Meego 未登录，会在 bits_context.json 标记 need_meego_login=true，由上层决定后续处理。
"""
import argparse
import json
import os
import re
import subprocess
import sys

CODE_HOST = "https://code.byted.org"

# Meego 需求里可能承载 PRD/设计文档的字段名关键词
PRD_FIELD_HINTS = ["prd", "trd", "需求文档", "产品文档", "设计文档", "方案", "文档链接", "技术方案", "prd链接"]
DOC_URL_RE = re.compile(r"https?://[a-z0-9.\-]*(?:larkoffice|feishu|larksuite)\.[a-z]+/[^\s\"'<>)\]]+", re.I)


def run_json(cmd, timeout=180):
    """执行 bytedcli 命令并解析 stdout 里的 JSON。返回 (data_or_none, raw_stdout, err)。"""
    try:
        p = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout, errors="replace")
    except Exception as e:  # noqa
        return None, "", f"执行失败: {e}"
    out = p.stdout.strip()
    # bytedcli -j 正常只输出 JSON；容错：截取第一个 { 到最后一个 }
    if out and not out.startswith("{"):
        s, e = out.find("{"), out.rfind("}")
        if s != -1 and e != -1:
            out = out[s:e + 1]
    try:
        return json.loads(out), out, p.stderr
    except Exception:
        return None, out, p.stderr


def parse_dev_id(bits_url):
    m = re.search(r"/detail/(\d+)", bits_url)
    return m.group(1) if m else None


def extract_prd_from_meego(work_item_id, project_key):
    """尝试用 bytedcli meego workitem get 抽取 PRD 文档链接。"""
    result = {"tried": True, "need_login": False, "prd_urls": [], "raw_available": False, "error": ""}
    data, raw, err = run_json([
        "bytedcli", "meego", "workitem", "get",
        "--work-item-id", str(work_item_id),
        "--project-key", str(project_key), "-j",
    ])
    if data is None:
        result["error"] = (err or "无法解析 meego 输出")[:300]
        return result
    if data.get("status") == "error":
        emsg = (data.get("error") or {}).get("message", "")
        code = (data.get("error") or {}).get("code", "")
        if code == "MEEGO_AUTH_REQUIRED" or "未登录" in emsg or "login" in emsg.lower():
            result["need_login"] = True
        result["error"] = emsg[:300]
        return result
    result["raw_available"] = True
    # 优先在疑似 PRD 字段里找飞书文档链接，找不到再全量兜底
    hinted, fallback = [], []
    for u in DOC_URL_RE.findall(raw):
        (hinted if any(h in raw.lower() for h in PRD_FIELD_HINTS) else fallback).append(u)
    urls = list(dict.fromkeys(hinted + fallback))
    # 过滤掉 meego 自身的 story 链接（那是需求本身不是 PRD 文档）
    result["prd_urls"] = [u for u in urls if "meego." not in u]
    return result


def main():
    ap = argparse.ArgumentParser(description="从 Bits 开发任务链接提取需求与仓库分支")
    ap.add_argument("--bits-url", help="Bits develop flow 链接")
    ap.add_argument("--dev-id", help="开发任务 devBasicId（与 --bits-url 二选一）")
    ap.add_argument("--extra-repos", help="额外补充仓库的 JSON 文件路径，内容为 [{\"repo_url\":..,\"branch\":..,\"base\":..(可选)}]，用于兜底 Bits 未挂载的仓库")
    ap.add_argument("--output-dir", default="output", help="输出目录")
    args = ap.parse_args()

    bits_url = args.bits_url
    dev_id = args.dev_id or (parse_dev_id(bits_url) if bits_url else None)
    if not dev_id:
        ap.error("需要 --dev-id，或能从 --bits-url 中解析出 /detail/<id>")

    os.makedirs(args.output_dir, exist_ok=True)
    ctx = {"dev_id": dev_id, "bits_url": bits_url, "title": "", "creator": "",
           "requirement": None, "repos": [], "prd": {}, "warnings": []}

    # 1) develop get —— 拿标题、需求(workItems)
    get_cmd = ["bytedcli", "bits", "develop", "get"]
    if bits_url:
        get_cmd += ["--url", bits_url]
    else:
        get_cmd += ["--dev-id", dev_id]
    get_cmd += ["-j"]
    gdata, _, gerr = run_json(get_cmd)
    if gdata and gdata.get("status") == "success":
        d = gdata["data"]
        ctx["title"] = d.get("title", "")
        ctx["creator"] = d.get("creator", "")
        work_items = (((d.get("related") or {}).get("data") or {}).get("workItems")) or []
        if work_items:
            wi = work_items[0]
            ctx["requirement"] = {
                "id": wi.get("id"), "platform": wi.get("platform"),
                "project_key": wi.get("spaceKey") or wi.get("spaceId"),
                "url": wi.get("url"), "name": wi.get("name"),
            }
    else:
        ctx["warnings"].append(f"develop get 未成功: {(gerr or '')[:200]}")

    # 2) inspect-changes —— 拿仓库 + 分支（最可靠）
    idata, _, ierr = run_json(["bytedcli", "bits", "develop", "inspect-changes", "--dev-id", str(dev_id), "-j"])
    if idata and idata.get("status") == "success":
        for b in idata["data"].get("bindings", []):
            repo_path = b.get("repo_path", "")
            ctx["repos"].append({
                "repo_url": f"{CODE_HOST}/{repo_path}" if repo_path else "",
                "repo_path": repo_path,
                "branch": b.get("source_branch", ""),
                "base": b.get("target_branch") or "master",
                "mr_url": b.get("url", ""),
                "binding_state": b.get("binding_state", ""),
            })
    else:
        ctx["warnings"].append(f"inspect-changes 未成功: {(ierr or '')[:200]}")

    # 3) PRD —— 通过 Meego 需求抽取飞书文档链接
    req = ctx["requirement"]
    if req and req.get("platform") == "meego" and req.get("id") and req.get("project_key"):
        prd = extract_prd_from_meego(req["id"], req["project_key"])
        ctx["prd"] = prd
        if prd.get("need_login"):
            ctx["warnings"].append("Meego 未登录，无法自动抽取 PRD，请先执行 `bytedcli meego login`，或改用手动传入 prd_content。")
    else:
        ctx["prd"] = {"tried": False, "prd_urls": [], "need_login": False,
                      "error": "未从开发任务解析到 Meego 需求，无法自动定位 PRD"}

    # 合并 Bits 提取仓库 + 手动补充仓库，按 (repo_url, branch) 去重
    merged = []
    seen = set()
    sources = {}  # (url,branch) -> 来源标记

    def add_repo(repo_url, branch, base, origin):
        if not repo_url or not branch:
            return
        key = (repo_url.rstrip("/"), branch)
        if key in seen:
            return
        seen.add(key)
        sources[key] = origin
        merged.append({"repo_url": repo_url, "branch": branch, "base": base or "master"})

    for r in ctx["repos"]:
        add_repo(r["repo_url"], r["branch"], r["base"], "bits")

    extra_loaded = 0
    if args.extra_repos:
        try:
            with open(args.extra_repos, encoding="utf-8") as f:
                extra = json.load(f)
            if isinstance(extra, dict):
                extra = extra.get("repos", [])
            for r in extra:
                before = len(seen)
                add_repo(r.get("repo_url", ""), r.get("branch", ""), r.get("base"), "extra")
                if len(seen) > before:
                    extra_loaded += 1
        except Exception as e:  # noqa
            ctx["warnings"].append(f"读取 extra_repos 失败: {str(e)[:200]}")
    ctx["extra_repos_added"] = extra_loaded
    ctx["merged_repos"] = [{**m, "origin": sources[(m["repo_url"].rstrip("/"), m["branch"])]} for m in merged]

    # 写出上下文与 repos.json
    ctx_path = os.path.join(args.output_dir, "bits_context.json")
    with open(ctx_path, "w", encoding="utf-8") as f:
        json.dump(ctx, f, ensure_ascii=False, indent=2)

    repos_cfg = {"repos": merged, "output_dir": os.path.join(args.output_dir, "repo_diffs")}
    repos_path = os.path.join(args.output_dir, "repos.json")
    with open(repos_path, "w", encoding="utf-8") as f:
        json.dump(repos_cfg, f, ensure_ascii=False, indent=2)

    # 控制台摘要
    print(f"标题: {ctx['title']}")
    if req:
        print(f"需求: {req.get('name')} -> {req.get('url')}")
    print(f"Bits 关联仓库 ({len(ctx['repos'])}):")
    for r in ctx["repos"]:
        print(f"  - {r['repo_path']} @ {r['branch']} (base {r['base']}) MR:{r['mr_url']}")
    if args.extra_repos:
        print(f"手动补充仓库: 新增 {ctx['extra_repos_added']} 个（已按 repo_url+branch 去重）")
    print(f"合并去重后待处理仓库 ({len(merged)}):")
    for m in ctx["merged_repos"]:
        print(f"  - {m['repo_url']} @ {m['branch']} (base {m['base']}) [{m['origin']}]")
    prd = ctx.get("prd", {})
    if prd.get("prd_urls"):
        print("PRD 候选文档:")
        for u in prd["prd_urls"]:
            print(f"  - {u}")
    elif prd.get("need_login"):
        print("PRD: 需要先 `bytedcli meego login` 才能自动抽取")
    else:
        print(f"PRD: 未自动获取到（{prd.get('error','')}）")
    for w in ctx["warnings"]:
        print(f"[warn] {w}", file=sys.stderr)
    print(f"\n上下文: {ctx_path}")
    print(f"仓库配置(可喂给 fetch_repo_diffs.py): {repos_path}")

    if not repos_cfg["repos"]:
        sys.exit(2)


if __name__ == "__main__":
    main()
