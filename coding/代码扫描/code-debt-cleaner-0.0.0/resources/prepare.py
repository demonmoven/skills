#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import random
from pathlib import Path

from common import (
    FORMATTER_TYPES,
    HIGH_RISK_TYPES,
    LOW_RISK_TYPES,
    MEDIUM_RISK_TYPES,
    TYPE_HANDLERS,
    find_go_directories,
    log,
    repo_relative_path,
    utc_timestamp,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="prepare.py",
        description="Code-Debt-Cleaner Prepare - 随机选择问题类型和目录",
    )
    parser.add_argument("-d", "--dir", default="./", help="指定扫描目录 (默认: ./)")
    return parser.parse_args()


def check_go_project(cwd: Path) -> None:
    log("prepare", "检查是否在 Go 项目根目录...")
    if not (cwd / "go.mod").exists():
        print(json.dumps({"error": "Not in a Go project root (go.mod not found)"}))
        raise SystemExit(1)
    log("prepare", "✓ 找到 go.mod 文件，确认是 Go 项目")


def randomize_types(rng: random.Random) -> list[str]:
    excluded = set(FORMATTER_TYPES) | {"typecheck"}
    pool = list((LOW_RISK_TYPES | MEDIUM_RISK_TYPES | HIGH_RISK_TYPES) - excluded)
    pool.sort()
    sample_size = min(len(pool), rng.randint(2, 4))
    return rng.sample(pool, sample_size)


def randomize_directories(scan_dir: Path, cwd: Path, rng: random.Random) -> list[str]:
    directories = find_go_directories(scan_dir)
    if not directories:
        return ["./"]

    sample_size = min(len(directories), rng.randint(1, 3))
    chosen = rng.sample(directories, sample_size)
    return [repo_relative_path(path, cwd) for path in chosen]


def main() -> None:
    args = parse_args()
    cwd = Path.cwd()
    scan_dir = (cwd / args.dir).resolve()
    rng = random.Random()

    log("prepare", "==========================================")
    log("prepare", "Code-Debt-Cleaner Prepare 开始运行")
    log("prepare", f"扫描目录: {args.dir}")
    log("prepare", "==========================================")

    check_go_project(cwd)

    log("prepare", "随机选择问题类型...")
    selected_types = randomize_types(rng)
    all_selected = ",".join(selected_types)
    log("prepare", f"  随机选择的问题类型: {all_selected}")

    tool_types: list[str] = []
    llm_types: list[str] = []
    for issue_type in selected_types:
        handler = TYPE_HANDLERS.get(issue_type, "llm")
        if handler == "tool":
            tool_types.append(issue_type)
        else:
            llm_types.append(issue_type)

    log("prepare", "随机选择目录...")
    directories = randomize_directories(scan_dir, cwd, rng)
    log("prepare", f"  随机选择的目录: {' '.join(directories)}")

    log("prepare", "==========================================")
    log("prepare", "准备完成")
    log("prepare", "==========================================")

    payload = {
        "preparer": "code-debt-cleaner-prepare",
        "timestamp": utc_timestamp(),
        "all_selected_types": all_selected,
        "tool_types": ",".join(tool_types),
        "llm_types": ",".join(llm_types),
        "directories": directories,
    }
    print(json.dumps(payload, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
