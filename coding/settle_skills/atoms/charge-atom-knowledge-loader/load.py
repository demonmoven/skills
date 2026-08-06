#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""charge-atom-knowledge-loader / load.py"""

import argparse
import json
import os
import shlex
import subprocess
import sys

LARK_DOCS_DIR = "/data/plugins/market/lark-docs-skill/skills/lark-docs-skill"


def _default_config_path():
    # <repo>/config/knowledge_sources.yaml
    return os.path.abspath(
        os.path.join(os.path.dirname(__file__), "..", "..", "config", "knowledge_sources.yaml")
    )


def _load_yaml(path):
    try:
        import yaml

        with open(path, "r", encoding="utf-8") as f:
            return yaml.safe_load(f)
    except ImportError:
        # 极简兜底:手写解析(只支持本仓库这个具体格式)
        return _parse_minimal(path)


def _parse_minimal(path):
    # 仅解析 version + sources 列表,字段值不含 ":" 和 "[" 的简单情形
    data = {"version": None, "sources": []}
    cur = None
    with open(path, "r", encoding="utf-8") as f:
        for raw in f:
            line = raw.rstrip("\n")
            if not line.strip() or line.lstrip().startswith("#"):
                continue
            if line.startswith("version:"):
                data["version"] = line.split(":", 1)[1].strip()
            elif line.startswith("sources:"):
                continue
            elif line.lstrip().startswith("- "):
                cur = {}
                data["sources"].append(cur)
                kv = line.lstrip()[2:]
                if ":" in kv:
                    k, v = kv.split(":", 1)
                    cur[k.strip()] = _parse_value(v.strip())
            elif cur is not None and ":" in line:
                k, v = line.split(":", 1)
                cur[k.strip()] = _parse_value(v.strip())
    return data


def _parse_value(v):
    v = v.split("#", 1)[0].strip()  # 剥行内注释
    if v.startswith("[") and v.endswith("]"):
        return [x.strip() for x in v[1:-1].split(",") if x.strip()]
    if v in ("true", "True"):
        return True
    if v in ("false", "False"):
        return False
    return v


def _load_lark_docs(url, dest):
    os.makedirs(os.path.dirname(dest), exist_ok=True)

    # 用 `tail -n +2` 去掉首行 banner
    cmd = (
        f"python3.11 -m lark_docs read-doc {shlex.quote(url)} --extract-field content "
        f"| tail -n +2"
    )
    p = subprocess.run(
        ["bash", "-lc", cmd],
        cwd=LARK_DOCS_DIR,
        capture_output=True,
        text=True,
        timeout=120,
    )
    if p.returncode != 0:
        raise RuntimeError(f"lark-docs-skill failed: {p.stderr.strip()[:500]}")

    with open(dest, "w", encoding="utf-8") as f:
        f.write(p.stdout)
    return os.path.getsize(dest)


LOADERS = {"lark-docs-skill": _load_lark_docs}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default=_default_config_path())
    ap.add_argument("--scope", default="charge")
    ap.add_argument("--refresh", choices=["auto", "force", "cache-only"], default="auto")
    args = ap.parse_args()

    cfg = _load_yaml(args.config) or {}
    scopes_filter = set(s.strip() for s in (args.scope or "").split(",") if s.strip())

    loaded, failed, warnings = [], [], []
    for src in cfg.get("sources", []) or []:
        key = src.get("key")
        scopes = src.get("scope") or []
        if scopes_filter and not (set(scopes) & scopes_filter):
            continue

        cache_path = os.path.expanduser(src.get("cache_path", ""))
        url = src.get("url")
        loader = src.get("loader")
        required = bool(src.get("required", False))
        policy = src.get("refresh_policy", "cross_session")

        cache_ok = bool(cache_path) and os.path.exists(cache_path) and os.path.getsize(cache_path) > 0

        need_fetch = False
        if args.refresh == "force":
            need_fetch = True
        elif args.refresh == "cache-only":
            need_fetch = False
        else:  # auto
            if policy == "per_run":
                need_fetch = True
            elif policy == "manual":
                # 仅手动:永不自动拉取;若无缓存则失败
                need_fetch = False
            else:  # cross_session
                need_fetch = not cache_ok

        try:
            if need_fetch:
                if loader not in LOADERS:
                    raise NotImplementedError(f"loader '{loader}' not implemented")
                size = LOADERS[loader](url, cache_path)
                loaded.append(
                    {
                        "key": key,
                        "cache_path": cache_path,
                        "bytes": size,
                        "hit_cache": False,
                        "loader": loader,
                    }
                )
            else:
                if not cache_ok:
                    if args.refresh == "cache-only":
                        raise FileNotFoundError(f"cache miss and refresh=cache-only: {cache_path}")
                    if policy == "manual":
                        raise FileNotFoundError(f"cache miss and refresh_policy=manual: {cache_path}")
                    raise FileNotFoundError(f"cache miss: {cache_path}")
                loaded.append(
                    {
                        "key": key,
                        "cache_path": cache_path,
                        "bytes": os.path.getsize(cache_path),
                        "hit_cache": True,
                        "loader": loader,
                    }
                )
        except Exception as e:
            entry = {"key": key, "error": str(e), "required": required}
            if required:
                failed.append(entry)
            else:
                warnings.append(entry)

    out = {"loaded": loaded, "failed": failed, "warnings": warnings}
    print(json.dumps(out, ensure_ascii=False, indent=2))
    if failed:
        for f in failed:
            print(f"[ERROR] {f.get('key')}: {f.get('error')}", file=sys.stderr)
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()

