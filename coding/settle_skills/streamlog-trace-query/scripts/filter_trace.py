#!/usr/bin/env python3
import argparse
import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable


def _expand_path(path: str) -> Path:
    return Path(os.path.expanduser(path)).resolve()


def _default_result_dir() -> Path:
    env_path = os.environ.get("STREAMLOG_RESULT_CACHE")
    if env_path:
        return _expand_path(env_path)
    return _expand_path("~/.cache/streamlog-results")


def _load_json(path: Path) -> dict:
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


def _parse_time_to_us(value: str) -> int | None:
    if value is None:
        return None
    s = str(value).strip()
    if not s:
        return None
    if s.isdigit():
        ts = int(s)
        if ts > 10**14:
            return ts
        if ts > 10**11:
            return ts * 1000
        return ts * 1000 * 1000
    try:
        dt = datetime.fromisoformat(s)
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return int(dt.timestamp() * 1000000)
    except Exception:
        return None


def _ts_to_iso(ts_raw: str | int | None) -> str:
    if ts_raw is None:
        return ""
    try:
        ts = int(ts_raw)
    except Exception:
        return str(ts_raw)
    if ts > 10**14:
        sec = ts / 1e6
    elif ts > 10**11:
        sec = ts / 1e3
    else:
        sec = ts
    return datetime.fromtimestamp(sec, tz=timezone.utc).isoformat()


def _ts_to_local_text(ts_raw: str | int | None) -> str:
    """Format timestamp as local time: YYYY-MM-DD HH:MM:SS.mmm"""
    if ts_raw is None:
        return ""
    try:
        ts = int(ts_raw)
    except Exception:
        return str(ts_raw)

    # Determine unit and millisecond part
    if ts > 10**14:  # microseconds
        sec = ts / 1e6
        ms = int((ts % 1_000_000) / 1_000)
    elif ts > 10**11:  # milliseconds
        sec = ts / 1e3
        ms = int(ts % 1_000)
    else:  # seconds
        sec = ts
        ms = 0

    dt_local = datetime.fromtimestamp(sec, tz=timezone.utc).astimezone()
    return dt_local.strftime("%Y-%m-%d %H:%M:%S") + f".{ms:03d}"


def _format_kv_as_text(kv: dict) -> str:
    """Render a kv dict as a single human-readable line."""
    level = kv.get("_level") or kv.get("level") or ""
    ts = _ts_to_local_text(kv.get("__timestamp"))
    location = kv.get("_location") or ""
    ipv4 = kv.get("_ipv4") or ""
    psm = kv.get("_psm") or ""
    logid = kv.get("__logid") or ""
    cluster = kv.get("_cluster") or ""
    stage = kv.get("_stage") or ""
    idc = kv.get("_idc") or ""
    spanid = kv.get("_spanid") or ""

    head = [
        str(level),
        str(ts),
        str(location),
        str(ipv4),
        str(psm),
        str(logid),
        str(cluster),
        str(stage),
        str(idc),
        str(spanid),
    ]

    skip_keys = {
        "id",
        "_level",
        "level",
        "__timestamp",
        "_location",
        "_ipv4",
        "_psm",
        "__logid",
        "_cluster",
        "_stage",
        "_idc",
        "_spanid",
    }

    tail = []
    for k, v in (kv or {}).items():
        if k in skip_keys:
            continue
        if v is None:
            continue
        tail.append(f"{k}={v}")
    return " ".join([p for p in head if p]) + (" " + " ".join(tail) if tail else "")


def _flatten_records(data: dict) -> Iterable[dict]:
    items = (data.get("data") or {}).get("items") or []
    for item in items:
        group = item.get("group") or {}
        group_psm = group.get("psm")
        for entry in item.get("value") or []:
            kv_list = entry.get("kv_list") or []
            kv = {it.get("key"): it.get("value") for it in kv_list if isinstance(it, dict)}
            # entry 上的 id 不在 kv_list 里；为了方便定位与去重，补到 kv 输出中。
            entry_id = entry.get("id")
            if entry_id and "id" not in kv:
                kv["id"] = entry_id
            ts_raw = kv.get("__timestamp")
            yield {
                "id": entry.get("id") or kv.get("__id") or "",
                "timestamp_raw": ts_raw,
                "timestamp": _ts_to_iso(ts_raw),
                "psm": kv.get("_psm") or group_psm or "",
                "level": entry.get("level") or kv.get("_level") or "",
                "spanid": kv.get("_spanid") or "",
                "logid": kv.get("__logid") or "",
                "msg": kv.get("_msg") or "",
                "group": group,
                "kv": kv,
            }


def _match_filters(record: dict, args) -> bool:
    if args.psm_set and record.get("psm") not in args.psm_set:
        return False
    if args.level_set and record.get("level") not in args.level_set:
        return False
    if args.spanid and record.get("spanid") != args.spanid:
        return False
    if args.logid and record.get("logid") != args.logid:
        return False
    if args.keyword:
        msg = record.get("msg") or ""
        if args.ignore_case:
            if args.keyword.lower() not in msg.lower():
                return False
        else:
            if args.keyword not in msg:
                return False
    if args.time_from_us or args.time_to_us:
        ts_raw = record.get("timestamp_raw")
        ts_us = _parse_time_to_us(ts_raw)
        if ts_us is None:
            return False
        if args.time_from_us and ts_us < args.time_from_us:
            return False
        if args.time_to_us and ts_us > args.time_to_us:
            return False
    return True


def _pick_latest_file(cache_dir: Path, logid: str | None) -> Path | None:
    if not cache_dir.exists():
        return None
    files = list(cache_dir.glob("*.json"))
    if logid:
        files = [f for f in files if logid in f.name]
    if not files:
        return None
    files.sort(key=lambda p: p.stat().st_mtime, reverse=True)
    return files[0]


def main():
    parser = argparse.ArgumentParser(description="Filter cached streamlog trace results.")
    parser.add_argument("--input", default="", help="input json file (cache result)")
    parser.add_argument("--cache-dir", default=str(_default_result_dir()), help="cache dir")
    parser.add_argument("--logid", default="", help="logid to pick latest cache file")
    parser.add_argument("--psm-list", default="", help="comma-separated psm list")
    parser.add_argument("--level-list", default="", help="comma-separated levels (Info,Warn,Error,...) ")
    parser.add_argument("--spanid", default="", help="span id to match")
    parser.add_argument("--keyword", default="", help="substring to search in _msg")
    parser.add_argument("--ignore-case", action="store_true", help="case-insensitive keyword match")
    parser.add_argument("--time-from", default="", help="start time (epoch or ISO) inclusive")
    parser.add_argument("--time-to", default="", help="end time (epoch or ISO) inclusive")
    parser.add_argument("--limit", type=int, default=50, help="max records to print")
    parser.add_argument("--summary", action="store_true", help="print summary only")
    parser.add_argument(
        "--format",
        choices=["jsonl", "text"],
        default="jsonl",
        help="output format: jsonl (default) or text (human-readable line)",
    )
    parser.add_argument("--out", default="", help="write output to file")
    args = parser.parse_args()

    input_path = None
    if args.input:
        input_path = _expand_path(args.input)
    else:
        input_path = _pick_latest_file(_expand_path(args.cache_dir), args.logid or None)

    if not input_path or not input_path.exists():
        print("No cache file found. Provide --input or ensure cache exists.", file=sys.stderr)
        return 2

    data = _load_json(input_path)

    args.psm_set = {s for s in args.psm_list.split(",") if s}
    args.level_set = {s for s in args.level_list.split(",") if s}
    args.spanid = args.spanid or ""
    args.logid = args.logid or ""
    args.keyword = args.keyword or ""
    args.time_from_us = _parse_time_to_us(args.time_from)
    args.time_to_us = _parse_time_to_us(args.time_to)

    records = []
    for record in _flatten_records(data):
        if _match_filters(record, args):
            records.append(record)

    if args.summary:
        from collections import Counter

        psm_counter = Counter(r.get("psm") for r in records)
        level_counter = Counter(r.get("level") for r in records)
        print("total", len(records))
        print("psm_counts", dict(psm_counter))
        print("level_counts", dict(level_counter))
        return 0

    output_items = [(r.get("kv") or {}) for r in records]

    if args.out:
        out_path = _expand_path(args.out)
        out_path.parent.mkdir(parents=True, exist_ok=True)
        with out_path.open("w", encoding="utf-8") as f:
            for item in output_items:
                if args.format == "text":
                    f.write(_format_kv_as_text(item) + "\n")
                else:
                    f.write(json.dumps(item, ensure_ascii=False) + "\n")
        print(f"Saved filtered records to {out_path}")
        return 0

    for item in output_items[: args.limit]:
        if args.format == "text":
            print(_format_kv_as_text(item))
        else:
            print(json.dumps(item, ensure_ascii=False))

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
