#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""charge-atom-progress-render 入口

子命令:
  init    — 首次创建快照(写空骨架 + task meta + 立刻渲染上传)
  patch   — 增量合并 stage / step / hero,深度 merge 后再渲染上传
  upload  — 不修改快照,仅重新渲染上传 progress.json + progress.html(快照已经被外部直接改写时使用)
  render  — 用 --data @file.json 全量覆盖,然后渲染上传(调试用)
  publish — 内置一站式上传:upload progress.json → 注入 json_url 到 HTML → upload progress.html → 写 latest_urls.json

输出 stdout 一段 JSON,包含 html_url / json_url / session_id / updated_at / local_json / local_html。

约定:
  - 工作目录: ~/files/charge-progress/<session_id>/
  - 模板:    与 render.py 同目录的 template.html
  - 上传:    本脚本通过 mira-runtime 上传管道直接发布两个文件,
            上传成功后**强制**把 progress.json 的公网 URL 回注到 progress.html 的 __JSON_URL__,
            再覆盖上传 HTML,最终把 {html_url, json_url, updated_at} 写入 latest_urls.json。
            mira-runtime 不可用时(local 调试),退化为只输出本地路径,并由 --strict 控制是否报错。
"""

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import time
from pathlib import Path

HERE = Path(__file__).resolve().parent
TEMPLATE_PATH = HERE / "template.html"

# 把骨架放在同目录,允许独立 import
sys.path.insert(0, str(HERE))

# 上传通道抽象层(替代原有 _try_web_hosting_publish / _try_mira_runtime_upload /
# _try_cli_upload 三个内联函数,详见 uploader.py 顶部注释)
from uploader import (  # noqa: E402
    UploadError,
    publish_file as _channel_publish_file,
    publish_dir as _channel_publish_dir,
    registered_channels,
)
from default_payload import build_initial_snapshot  # noqa: E402


# ---------- 工具函数 ----------

def session_dir(session_id: str) -> Path:
    base = Path(os.path.expanduser("~/files/charge-progress")) / session_id
    base.mkdir(parents=True, exist_ok=True)
    return base


def load_snapshot(session_id: str) -> dict:
    f = session_dir(session_id) / "progress.json"
    if not f.exists():
        return {}
    with f.open("r", encoding="utf-8") as fh:
        return json.load(fh)


def save_snapshot(session_id: str, data: dict) -> Path:
    f = session_dir(session_id) / "progress.json"
    data.setdefault("task", {})["updated_at"] = int(time.time())
    with f.open("w", encoding="utf-8") as fh:
        json.dump(data, fh, ensure_ascii=False, indent=2)
    return f


def load_latest_urls(session_id: str) -> dict:
    f = session_dir(session_id) / "latest_urls.json"
    if not f.exists():
        return {}
    try:
        with f.open("r", encoding="utf-8") as fh:
            return json.load(fh)
    except Exception:
        return {}


def save_latest_urls(session_id: str, html_url: str, json_url: str) -> Path:
    f = session_dir(session_id) / "latest_urls.json"
    payload = {
        "html_url": html_url,
        "json_url": json_url,
        "updated_at": int(time.time()),
    }
    with f.open("w", encoding="utf-8") as fh:
        json.dump(payload, fh, ensure_ascii=False, indent=2)
    return f


def deep_merge(dst: dict, patch: dict) -> dict:
    """字典深合并:patch 字段覆盖 dst,list 整体替换(语义清晰)。"""
    for k, v in patch.items():
        if isinstance(v, dict) and isinstance(dst.get(k), dict):
            deep_merge(dst[k], v)
        else:
            dst[k] = v
    return dst


def update_step(steps: list, step_id: str, **kw) -> None:
    for s in steps:
        if s.get("id") == step_id:
            s.update({k: v for k, v in kw.items() if v is not None})
            return


def update_stage(stages: list, stage_id: str, status=None, panels=None, **kw) -> None:
    for s in stages:
        if s.get("id") == stage_id:
            if status is not None:
                s["status"] = status
            if panels is not None:
                s["panels"] = panels
            for k, v in kw.items():
                if v is not None:
                    s[k] = v
            return


def parse_hero_update(spec: str) -> list:
    """语法: '变更文件=4|单元测试=PASS:ok|流水线状态=RUNNING:run:0.57'
    每段 key=value[:tone[:fill]]。返回要应用到 hero 数组上的 patch list。
    """
    if not spec:
        return []
    out = []
    for chunk in spec.split("|"):
        chunk = chunk.strip()
        if not chunk or "=" not in chunk:
            continue
        k, rest = chunk.split("=", 1)
        parts = rest.split(":")
        v = parts[0]
        tone = parts[1] if len(parts) > 1 and parts[1] else None
        fill = float(parts[2]) if len(parts) > 2 and parts[2] else None
        item = {"k": k.strip(), "v": v.strip()}
        if tone:
            item["tone"] = tone
        if fill is not None:
            item["fill"] = fill
        out.append(item)
    return out


def apply_hero_patch(hero: list, patch_items: list) -> None:
    for p in patch_items:
        for h in hero:
            if h.get("k") == p["k"]:
                h.update(p)
                break
        else:
            hero.append(p)


def load_at_arg(value: str):
    """支持 '@path/to/file.json' 读取本地 JSON,否则把字符串当 JSON 解析。"""
    if not value:
        return None
    if value.startswith("@"):
        with open(value[1:], "r", encoding="utf-8") as fh:
            return json.load(fh)
    return json.loads(value)


# ---------- 渲染 ----------

def render_html(session_id: str, data: dict, json_url: str = "progress.json") -> Path:
    """渲染 HTML。json_url 为前端轮询的 URL:
       - 默认 'progress.json'(相对路径,适合同目录托管)
       - 跨域托管时传入完整 https URL(由 init/patch/publish 调用方在拿到上传 URL 后回填)

    幂等:如果之前已渲染过、且 latest_urls.json 中已有 json_url,
    本次又传入了默认相对路径,会自动用 latest_urls.json 中保存的真实 URL,
    避免误把已经有效的 HTML 退回到 404 状态。
    """
    if not TEMPLATE_PATH.exists():
        raise SystemExit(f"模板文件不存在: {TEMPLATE_PATH}")
    tpl = TEMPLATE_PATH.read_text(encoding="utf-8")
    title = data.get("task", {}).get("title") or "计费域研发流程实时进展"

    # 关键修复①:只要 json_url 是默认值,就尝试从 latest_urls.json 找回真实公网 URL
    if json_url in ("progress.json", "", None):
        latest = load_latest_urls(session_id)
        candidate = latest.get("json_url", "")
        if isinstance(candidate, str) and candidate.startswith(("http://", "https://")):
            json_url = candidate

    initial = json.dumps(data, ensure_ascii=False)
    html = (tpl
            .replace("__TITLE__", title)
            .replace("__JSON_URL__", json_url)
            .replace("__INITIAL_DATA__", initial))

    out = session_dir(session_id) / "progress.html"
    # 关键修复②:如果上一版 HTML 已经包含了真实 URL,而这一次仍是默认相对路径,
    # 则保留上一版的真实 URL(防止 patch 时被默认值悄悄覆盖)
    if json_url in ("progress.json", "", None) and out.exists():
        prev = out.read_text(encoding="utf-8")
        m = re.search(r'const JSON_URL = "([^"]+)"', prev)
        if m and m.group(1) not in ("progress.json", "", None):
            html = html.replace('const JSON_URL = "progress.json"',
                                f'const JSON_URL = "{m.group(1)}"')
    out.write_text(html, encoding="utf-8")
    return out


# ---------- 上传 ----------
# 注意: 所有具体上传通道(mira_runtime / mira-cli / web-hosting)都迁移到 uploader.py,
# 见 uploader.py 顶部 docstring。本文件只保留薄薄一层 web-hosting 的"目录适配"逻辑
# (把 progress.html 复制为 dist/index.html),其余统一委托给 uploader。

def _try_web_hosting_publish(session_id: str, sd: Path) -> dict:
    """走 uploader.publish_dir(web-hosting 通道)发布整目录。

    web-hosting CLI 要求 dist/index.html 存在,而 atom 产物叫 progress.html。
    在临时目录里同时放 index.html + progress.json,不污染原 sd。

    失败(env 缺失 / CLI 不可用 / 部署报错)统一返回 {},由调用方回退到 publish_file。
    """
    import tempfile
    with tempfile.TemporaryDirectory(prefix="charge-progress-dist-") as tmp:
        tmp_dir = Path(tmp)
        try:
            shutil.copy2(sd / "progress.html", tmp_dir / "index.html")
            shutil.copy2(sd / "progress.json", tmp_dir / "progress.json")
        except Exception:
            return {}
        app_name = os.environ.get("HOSTING_DEPLOY_NAME") or f"charge-progress-{session_id}"
        return _channel_publish_dir(tmp_dir, app_name=app_name)


def upload_file(local: Path, strict: bool = False) -> str:
    """上传一个本地文件,返回公网 URL。委托给 uploader.publish_file。

    保留本函数签名是为了不破坏 render.py 内其它 publish() / cmd_upload() 调用方。
    新代码请直接 import uploader.publish_file。
    """
    return _channel_publish_file(local, strict=strict)


# ---------- publish:一站式发布(P0 关键修复) ----------

def publish(session_id: str, *, strict: bool = False) -> dict:
    """一站式发布流程(原 SKILL.md 第 142-152 行约定的 4 步胶水):
       1) 读取 progress.json
       2) 上传 progress.json 取得 json_url
       3) 用 json_url 重渲 progress.html(注入到 __JSON_URL__)
       4) 上传 progress.html 取得 html_url
       5) 写 latest_urls.json {html_url, json_url, updated_at}

    任一步失败:
       - strict=True  → 抛异常
       - strict=False → 返回部分字段 + warning
    """
    sd = session_dir(session_id)
    json_path = sd / "progress.json"
    html_path = sd / "progress.html"
    if not json_path.exists():
        raise UploadError(f"progress.json 不存在,请先 init: {json_path}")

    result = {
        "session_id": session_id,
        "local_json": str(json_path),
        "local_html": str(html_path),
    }

    # ---- 优先尝试 web-hosting 永久态(需要 DATA_AGENT_TITAN_PASSPORT_ID)----
    # 命中即直接返回 app_url + 永久 json_url,跳过 mira_runtime / CLI 通道。
    # 任何失败(env 缺失 / CLI 找不到 / RPC 报错)都静默回退,保证 backward-compat。
    wh = _try_web_hosting_publish(session_id, sd)
    if wh.get("app_url"):
        save_latest_urls(session_id, wh["app_url"], wh.get("json_url", ""))
        result["html_url"] = wh["app_url"]
        result["json_url"] = wh.get("json_url", "")
        result["channel"] = "web-hosting"
        result["updated_at"] = int(time.time())
        return result

    # 步骤 1+2:上传 JSON
    try:
        json_url = upload_file(json_path, strict=strict)
    except UploadError as e:
        if strict:
            raise
        result["warning"] = f"上传 progress.json 失败: {e}"
        json_url = ""

    if json_url:
        # 步骤 3:重渲 HTML 注入真实 json_url
        snapshot = load_snapshot(session_id)
        html_path = render_html(session_id, snapshot, json_url=json_url)
        result["local_html"] = str(html_path)

    # 步骤 4:上传 HTML
    html_url = ""
    try:
        html_url = upload_file(html_path, strict=strict)
    except UploadError as e:
        if strict:
            raise
        result.setdefault("warning", f"上传 progress.html 失败: {e}")

    # 步骤 5:写 latest_urls.json
    if html_url or json_url:
        save_latest_urls(session_id, html_url, json_url)

    result["html_url"] = html_url
    result["json_url"] = json_url
    result["updated_at"] = int(time.time())
    return result


# ---------- 子命令 ----------

def _emit(payload: dict) -> None:
    print(json.dumps(payload, ensure_ascii=False))


def cmd_init(args):
    snapshot = load_snapshot(args.session_id)
    now = int(time.time())
    task_meta = {
        "title":       args.task_title or snapshot.get("task", {}).get("title", ""),
        "prd_url":     args.prd_url    or snapshot.get("task", {}).get("prd_url", ""),
        "repo":        args.repo       or snapshot.get("task", {}).get("repo", ""),
        "branch":      args.branch     or snapshot.get("task", {}).get("branch", ""),
        "owner":       args.owner      or snapshot.get("task", {}).get("owner", ""),
        "session_id":  args.session_id,
        "started_at":  snapshot.get("task", {}).get("started_at") or now,
        "updated_at":  now,
    }
    if not snapshot:
        snapshot = build_initial_snapshot(task_meta)
    else:
        snapshot["task"] = task_meta
    json_path = save_snapshot(args.session_id, snapshot)
    html_path = render_html(args.session_id, snapshot, json_url=args.json_url or "progress.json")

    out = {
        "session_id":  args.session_id,
        "local_json":  str(json_path),
        "local_html":  str(html_path),
        "updated_at":  snapshot["task"]["updated_at"],
        "action":      "init",
    }
    if not args.no_publish:
        try:
            pub = publish(args.session_id, strict=args.strict)
            out.update({k: v for k, v in pub.items() if k in ("html_url", "json_url", "warning")})
            out["local_html"] = pub.get("local_html", out["local_html"])
        except UploadError as e:
            if args.strict:
                raise
            out["warning"] = str(e)
    _emit(out)


def cmd_patch(args):
    snapshot = load_snapshot(args.session_id)
    if not snapshot:
        raise SystemExit(f"快照不存在,请先 init: session_id={args.session_id}")

    if args.stage:
        update_stage(
            snapshot.setdefault("stages", []),
            args.stage,
            status=args.status,
            title=args.stage_title,
            subtitle=args.stage_subtitle,
        )

    if args.panel_json:
        panels_payload = load_at_arg(args.panel_json)
        if not isinstance(panels_payload, list):
            raise SystemExit("--panel-json 必须是 panel 数组")
        if not args.stage:
            raise SystemExit("使用 --panel-json 需要同时指定 --stage")
        update_stage(snapshot["stages"], args.stage, panels=panels_payload)

    if args.stage and args.status and args.sync_step:
        update_step(
            snapshot.setdefault("steps", []),
            args.stage,
            status=args.status,
            sub=args.step_sub,
        )

    if args.hero_update:
        items = parse_hero_update(args.hero_update)
        apply_hero_patch(snapshot.setdefault("hero", []), items)

    if args.merge_json:
        merge_payload = load_at_arg(args.merge_json)
        if not isinstance(merge_payload, dict):
            raise SystemExit("--merge-json 必须是对象")
        deep_merge(snapshot, merge_payload)

    json_path = save_snapshot(args.session_id, snapshot)
    html_path = render_html(args.session_id, snapshot, json_url=args.json_url or "progress.json")

    out = {
        "session_id":  args.session_id,
        "local_json":  str(json_path),
        "local_html":  str(html_path),
        "updated_at":  snapshot["task"]["updated_at"],
        "action":      "patch",
        "stage":       args.stage,
        "status":      args.status,
    }
    if not args.no_publish:
        try:
            pub = publish(args.session_id, strict=args.strict)
            out.update({k: v for k, v in pub.items() if k in ("html_url", "json_url", "warning")})
            out["local_html"] = pub.get("local_html", out["local_html"])
        except UploadError as e:
            if args.strict:
                raise
            out["warning"] = str(e)
    _emit(out)


def cmd_upload(args):
    """重新渲染并(可选)发布。
       - 不传 --json-url:延用 latest_urls.json 中已有的 json_url(由 render_html 内部回看)
       - 传 --json-url <URL>:把外部 agent 上传 progress.json 后拿到的公网 URL 注入 HTML
    """
    snapshot = load_snapshot(args.session_id)
    if not snapshot:
        raise SystemExit(f"快照不存在: session_id={args.session_id}")
    json_path = save_snapshot(args.session_id, snapshot)
    html_path = render_html(args.session_id, snapshot, json_url=args.json_url or "progress.json")

    out = {
        "session_id":  args.session_id,
        "local_json":  str(json_path),
        "local_html":  str(html_path),
        "updated_at":  snapshot["task"]["updated_at"],
        "action":      "upload",
    }

    if args.json_url and args.json_url.startswith(("http://", "https://")):
        # 外部已上传 JSON,只需要发布 HTML
        try:
            html_url = upload_file(html_path, strict=args.strict)
        except UploadError as e:
            if args.strict:
                raise
            out["warning"] = str(e)
            html_url = ""
        if html_url:
            save_latest_urls(args.session_id, html_url, args.json_url)
            out["html_url"] = html_url
            out["json_url"] = args.json_url
    elif not args.no_publish:
        try:
            pub = publish(args.session_id, strict=args.strict)
            out.update({k: v for k, v in pub.items() if k in ("html_url", "json_url", "warning")})
        except UploadError as e:
            if args.strict:
                raise
            out["warning"] = str(e)
    _emit(out)


def cmd_render(args):
    if not args.data:
        raise SystemExit("render 子命令必须传 --data @file.json 或内联 JSON")
    payload = load_at_arg(args.data)
    if not isinstance(payload, dict):
        raise SystemExit("--data 必须是对象")
    payload.setdefault("task", {})["session_id"] = args.session_id
    json_path = save_snapshot(args.session_id, payload)
    html_path = render_html(args.session_id, payload, json_url=args.json_url or "progress.json")
    out = {
        "session_id":  args.session_id,
        "local_json":  str(json_path),
        "local_html":  str(html_path),
        "updated_at":  payload["task"]["updated_at"],
        "action":      "render",
    }
    if not args.no_publish:
        try:
            pub = publish(args.session_id, strict=args.strict)
            out.update({k: v for k, v in pub.items() if k in ("html_url", "json_url", "warning")})
        except UploadError as e:
            if args.strict:
                raise
            out["warning"] = str(e)
    _emit(out)


def cmd_publish(args):
    """单独发布。
       - 同时传 --html-url + --json-url:外部已上传完毕,直接落 latest_urls.json + 注入 HTML 模板
       - 否则走内置上传通道
    """
    if args.html_url and args.json_url:
        snapshot = load_snapshot(args.session_id)
        if not snapshot:
            raise SystemExit(f"快照不存在: session_id={args.session_id}")
        render_html(args.session_id, snapshot, json_url=args.json_url)
        save_latest_urls(args.session_id, args.html_url, args.json_url)
        _emit({
            "session_id": args.session_id,
            "html_url":   args.html_url,
            "json_url":   args.json_url,
            "updated_at": int(time.time()),
            "action":     "publish:external",
        })
        return

    pub = publish(args.session_id, strict=args.strict)
    pub["action"] = "publish:builtin"
    _emit(pub)


# ---------- argparse ----------

def _add_publish_flags(p):
    p.add_argument("--strict", action="store_true",
                   help="任一上传步骤失败立即报错退出(默认仅 warn,不阻塞主链路)")
    p.add_argument("--no-publish", action="store_true",
                   help="只写本地不发布(纯本地调试用)")


def build_parser():
    p = argparse.ArgumentParser(prog="charge-atom-progress-render")
    sub = p.add_subparsers(dest="cmd", required=True)

    # init
    p_init = sub.add_parser("init", help="首次创建快照 + 自动发布")
    p_init.add_argument("--session-id", required=True)
    p_init.add_argument("--task-title")
    p_init.add_argument("--prd-url")
    p_init.add_argument("--repo")
    p_init.add_argument("--branch")
    p_init.add_argument("--owner")
    p_init.add_argument("--json-url",
                        help="HTML 中嵌入的 progress.json 公网 URL(可选,默认尝试自动 publish 拿到)")
    _add_publish_flags(p_init)
    p_init.set_defaults(func=cmd_init)

    # patch
    p_patch = sub.add_parser("patch", help="增量合并 + 自动发布")
    p_patch.add_argument("--session-id", required=True)
    p_patch.add_argument("--stage", help="如 s4,与 steps[].id 共用")
    p_patch.add_argument("--status", choices=["done", "run", "warn", "err", "pend"])
    p_patch.add_argument("--stage-title")
    p_patch.add_argument("--stage-subtitle")
    p_patch.add_argument("--panel-json", help="@file.json 或内联 JSON 数组,整体替换该 stage 的 panels")
    p_patch.add_argument("--sync-step", action="store_true",
                         help="同时把 steps[id=stage] 的状态同步为 status")
    p_patch.add_argument("--step-sub", help="同步 step 时的 sub 副标题")
    p_patch.add_argument("--hero-update",
                         help="语法: 'k=v[:tone[:fill]]|k=v...',例如 '变更文件=4|单元测试=PASS:ok'")
    p_patch.add_argument("--merge-json", help="@file.json 任意深合并补丁(高级用法)")
    p_patch.add_argument("--json-url", help="HTML 中嵌入的 progress.json 公网 URL(可选)")
    _add_publish_flags(p_patch)
    p_patch.set_defaults(func=cmd_patch)

    # upload
    p_up = sub.add_parser("upload", help="重渲染 +(可选)发布")
    p_up.add_argument("--session-id", required=True)
    p_up.add_argument("--json-url", help="外部 agent 上传 progress.json 后拿到的公网 URL,注入 HTML")
    _add_publish_flags(p_up)
    p_up.set_defaults(func=cmd_upload)

    # render
    p_render = sub.add_parser("render", help="全量覆盖 + 自动发布")
    p_render.add_argument("--session-id", required=True)
    p_render.add_argument("--data", required=True, help="@file.json 或内联 JSON")
    p_render.add_argument("--json-url", help="HTML 中嵌入的 progress.json 公网 URL(可选)")
    _add_publish_flags(p_render)
    p_render.set_defaults(func=cmd_render)

    # publish — 一站式发布
    p_pub = sub.add_parser("publish", help="一站式上传 progress.json + progress.html,写 latest_urls.json")
    p_pub.add_argument("--session-id", required=True)
    p_pub.add_argument("--html-url",
                       help="外部已上传 HTML 拿到的 URL,与 --json-url 配合使用,内置上传通道将被跳过")
    p_pub.add_argument("--json-url",
                       help="外部已上传 JSON 拿到的 URL,与 --html-url 配合使用,内置上传通道将被跳过")
    _add_publish_flags(p_pub)
    p_pub.set_defaults(func=cmd_publish)

    return p


def main():
    args = build_parser().parse_args()
    args.func(args)


if __name__ == "__main__":
    main()

