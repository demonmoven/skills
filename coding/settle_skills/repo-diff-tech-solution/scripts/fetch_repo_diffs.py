#!/usr/bin/env python3
"""拉取多个仓库「指定分支 vs base(默认 master)」的 diff，输出结构化摘要。

用法:
    python3 fetch_repo_diffs.py --config repos.json
    python3 fetch_repo_diffs.py --repo-url <url> --branch <branch> [--base master]

config.json 结构:
{
  "repos": [
    {"repo_url": "https://code.xxx/group/repo", "branch": "feature/x"},
    {"repo_url": "https://code.xxx/group/repo2", "branch": "dev", "base": "main"}
  ],
  "output_dir": "output/repo_diffs",   // 可选
  "base": "master"                       // 可选，全局默认 base 分支
}

输出到 output_dir:
- <repo>__<branch>.diff        每个仓库的完整 diff（超限会截断并标记）
- diff_summary.json            结构化摘要（供程序/分析读取）
- diff_summary.md              人类可读的汇总（供撰写技术方案时参考）
"""
import argparse
import json
import os
import re
import subprocess
import sys
import tempfile

MAX_DIFF_CHARS = 200_000  # 单仓库完整 diff 字符上限，超出则截断


def run(cmd, cwd=None, timeout=600):
    p = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True,
                       timeout=timeout, errors="replace")
    return p.returncode, p.stdout, p.stderr


def repo_slug(repo_url):
    slug = re.sub(r"\.git$", "", repo_url.rstrip("/"))
    slug = re.sub(r"^https?://[^/]+/", "", slug)
    return slug.replace("/", "_") or "repo"


def ref_exists(cwd, ref):
    code, _, _ = run(["git", "rev-parse", "--verify", "--quiet", ref], cwd=cwd)
    return code == 0


def resolve_base(cwd, wanted):
    """返回实际存在的 base 远程分支引用，找不到时回退 master/main。"""
    candidates = [wanted, "master", "main", "develop"]
    seen = set()
    for c in candidates:
        if not c or c in seen:
            continue
        seen.add(c)
        if ref_exists(cwd, f"origin/{c}"):
            return c
    return None


def numstat(cwd, base_ref, branch_ref):
    code, out, err = run(["git", "diff", "--numstat", f"{base_ref}...{branch_ref}"], cwd=cwd)
    files = []
    total_add = total_del = 0
    if code == 0:
        for line in out.splitlines():
            parts = line.split("\t")
            if len(parts) < 3:
                continue
            add, dele, path = parts[0], parts[1], "\t".join(parts[2:])
            a = 0 if add == "-" else int(add)
            d = 0 if dele == "-" else int(dele)
            total_add += a
            total_del += d
            files.append({"path": path, "additions": a, "deletions": d, "binary": add == "-"})
    return files, total_add, total_del, (err if code != 0 else "")


def name_status(cwd, base_ref, branch_ref):
    code, out, _ = run(["git", "diff", "--name-status", f"{base_ref}...{branch_ref}"], cwd=cwd)
    status = {}
    if code == 0:
        for line in out.splitlines():
            parts = line.split("\t")
            if len(parts) >= 2:
                status[parts[-1]] = parts[0]
    return status


def branch_authors(cwd, base_ref, branch_ref):
    """统计分支相对 base 引入的提交作者（按提交数降序），用于排期章节的负责人推断。"""
    code, out, _ = run(["git", "log", "--format=%an", f"{base_ref}..{branch_ref}"], cwd=cwd)
    counts = {}
    if code == 0:
        for name in out.splitlines():
            name = name.strip()
            if name:
                counts[name] = counts.get(name, 0) + 1
    return [{"name": n, "commits": c}
            for n, c in sorted(counts.items(), key=lambda kv: kv[1], reverse=True)]


def process_repo(entry, default_base, output_dir):
    repo_url = entry["repo_url"]
    branch = entry["branch"]
    wanted_base = entry.get("base") or default_base or "master"
    slug = repo_slug(repo_url)
    result = {
        "repo_url": repo_url,
        "repo_name": slug,
        "branch": branch,
        "base": wanted_base,
        "ok": False,
        "error": "",
        "files_changed": 0,
        "insertions": 0,
        "deletions": 0,
        "changed_files": [],
        "authors": [],
        "diff_file": "",
        "truncated": False,
    }

    with tempfile.TemporaryDirectory() as tmp:
        clone_dir = os.path.join(tmp, "repo")
        # blobless 部分克隆：快，且 diff 时按需拉取 blob
        code, _, err = run(
            ["git", "clone", "--filter=blob:none", "--no-checkout", "--quiet", repo_url, clone_dir],
            timeout=1200,
        )
        if code != 0:
            # 回退到普通克隆
            code, _, err = run(["git", "clone", "--quiet", repo_url, clone_dir], timeout=1800)
            if code != 0:
                result["error"] = f"clone 失败: {err.strip()[:500]}"
                return result

        # 确保 branch ref 存在（部分克隆可能未取到该远程分支）
        if not ref_exists(clone_dir, f"origin/{branch}"):
            run(["git", "fetch", "--quiet", "origin", branch], cwd=clone_dir, timeout=600)
        if not ref_exists(clone_dir, f"origin/{branch}"):
            result["error"] = f"未找到分支 origin/{branch}"
            return result

        base = resolve_base(clone_dir, wanted_base)
        if not base:
            result["error"] = f"未找到 base 分支（尝试过 {wanted_base}/master/main/develop）"
            return result
        result["base"] = base

        base_ref = f"origin/{base}"
        branch_ref = f"origin/{branch}"

        files, add, dele, nerr = numstat(clone_dir, base_ref, branch_ref)
        if nerr:
            result["error"] = f"diff 计算失败: {nerr.strip()[:500]}"
            return result
        status_map = name_status(clone_dir, base_ref, branch_ref)
        for f in files:
            f["status"] = status_map.get(f["path"], "M")

        result["files_changed"] = len(files)
        result["insertions"] = add
        result["deletions"] = dele
        result["changed_files"] = files
        result["authors"] = branch_authors(clone_dir, base_ref, branch_ref)

        # 完整 diff
        code, diff_out, _ = run(["git", "diff", f"{base_ref}...{branch_ref}"], cwd=clone_dir)
        if len(diff_out) > MAX_DIFF_CHARS:
            diff_out = diff_out[:MAX_DIFF_CHARS] + "\n\n... [diff 已截断，超过上限，完整变更请结合 changed_files 列表分析] ...\n"
            result["truncated"] = True

        os.makedirs(output_dir, exist_ok=True)
        diff_path = os.path.join(output_dir, f"{slug}__{branch.replace('/', '_')}.diff")
        with open(diff_path, "w", encoding="utf-8") as fh:
            fh.write(diff_out)
        result["diff_file"] = os.path.relpath(diff_path)
        result["ok"] = True
    return result


def write_markdown(results, base_default, md_path):
    lines = ["# 多仓库分支变更汇总", ""]
    ok = [r for r in results if r["ok"]]
    lines.append(f"- 仓库总数: {len(results)}，成功: {len(ok)}，失败: {len(results) - len(ok)}")
    lines.append("")
    for r in results:
        lines.append(f"## {r['repo_name']}")
        lines.append("")
        lines.append(f"- 仓库: {r['repo_url']}")
        lines.append(f"- 对比: `{r['branch']}` vs `{r['base']}`")
        if not r["ok"]:
            lines.append(f"- 状态: ❌ 失败 - {r['error']}")
            lines.append("")
            continue
        lines.append(f"- 变更文件数: {r['files_changed']}，+{r['insertions']} / -{r['deletions']}"
                     + ("（diff 已截断）" if r["truncated"] else ""))
        if r.get("authors"):
            authors_str = "、".join(f"{a['name']}({a['commits']})" for a in r["authors"])
            lines.append(f"- 提交作者（按提交数）: {authors_str}")
        lines.append(f"- diff 文件: `{r['diff_file']}`")
        lines.append("")
        lines.append("| 文件 | 状态 | +/- |")
        lines.append("| --- | --- | --- |")
        for f in r["changed_files"][:200]:
            add = "bin" if f.get("binary") else f"+{f['additions']}/-{f['deletions']}"
            lines.append(f"| {f['path']} | {f.get('status', 'M')} | {add} |")
        if len(r["changed_files"]) > 200:
            lines.append(f"| ... 其余 {len(r['changed_files']) - 200} 个文件省略 ... | | |")
        lines.append("")
    with open(md_path, "w", encoding="utf-8") as fh:
        fh.write("\n".join(lines))


def main():
    ap = argparse.ArgumentParser(description="拉取多仓库分支 vs base 的 diff")
    ap.add_argument("--config", help="repos 配置 JSON 文件路径")
    ap.add_argument("--repo-url", help="单仓库模式：仓库地址")
    ap.add_argument("--branch", help="单仓库模式：对比分支")
    ap.add_argument("--base", default="master", help="base 分支，默认 master")
    ap.add_argument("--output-dir", default="output/repo_diffs", help="输出目录")
    args = ap.parse_args()

    default_base = args.base
    output_dir = args.output_dir
    repos = []
    if args.config:
        with open(args.config, encoding="utf-8") as fh:
            cfg = json.load(fh)
        repos = cfg.get("repos", [])
        default_base = cfg.get("base", default_base)
        output_dir = cfg.get("output_dir", output_dir)
    elif args.repo_url and args.branch:
        repos = [{"repo_url": args.repo_url, "branch": args.branch}]
    else:
        ap.error("需要 --config，或同时提供 --repo-url 与 --branch")

    if not repos:
        ap.error("repos 列表为空")

    os.makedirs(output_dir, exist_ok=True)
    results = []
    for entry in repos:
        print(f"[*] 处理 {entry.get('repo_url')} @ {entry.get('branch')} ...", file=sys.stderr)
        r = process_repo(entry, default_base, output_dir)
        state = "OK" if r["ok"] else f"FAIL: {r['error']}"
        print(f"    -> {state}", file=sys.stderr)
        results.append(r)

    summary = {"base_default": default_base, "output_dir": output_dir, "repos": results}
    json_path = os.path.join(output_dir, "diff_summary.json")
    with open(json_path, "w", encoding="utf-8") as fh:
        json.dump(summary, fh, ensure_ascii=False, indent=2)
    md_path = os.path.join(output_dir, "diff_summary.md")
    write_markdown(results, default_base, md_path)

    ok = sum(1 for r in results if r["ok"])
    print(f"\n✅ 完成: {ok}/{len(results)} 个仓库成功")
    print(f"   结构化摘要: {json_path}")
    print(f"   可读摘要:   {md_path}")
    if ok == 0:
        sys.exit(2)


if __name__ == "__main__":
    main()
