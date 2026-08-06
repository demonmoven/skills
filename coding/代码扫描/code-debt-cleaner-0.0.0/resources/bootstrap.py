#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
from pathlib import Path

from common import BOOTSTRAP_ALLOWED_LINTERS, DEFAULT_LINTERS, log, split_csv, utc_timestamp


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="bootstrap.py",
        description="Code-Debt-Cleaner Bootstrap - 生成仓库初始 .golangci.yml",
    )
    parser.add_argument(
        "--repo-root",
        default=".",
        help="目标仓库根目录，默认当前目录",
    )
    parser.add_argument(
        "--output",
        default=".golangci.yml",
        help="输出路径。相对路径默认写到 repo-root 下",
    )
    parser.add_argument(
        "--enable",
        action="append",
        default=[],
        help="额外启用的 linter，支持重复传参或逗号分隔",
    )
    parser.add_argument(
        "--disable",
        action="append",
        default=[],
        help="从默认集合中移除的 linter，支持重复传参或逗号分隔",
    )
    parser.add_argument(
        "--enable-typecheck",
        action="store_true",
        help="显式把 typecheck 加进初始配置",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="覆盖已存在的输出文件",
    )
    return parser.parse_args()


def flatten_csv_args(values: list[str]) -> list[str]:
    flattened: list[str] = []
    for value in values:
        flattened.extend(split_csv(value))
    return flattened


def validate_linters(values: list[str], flag_name: str) -> None:
    invalid = sorted(set(values) - BOOTSTRAP_ALLOWED_LINTERS)
    if invalid:
        joined = ",".join(invalid)
        allowed = ",".join(sorted(BOOTSTRAP_ALLOWED_LINTERS))
        raise SystemExit(f"{flag_name} contains unsupported linters: {joined}. allowed: {allowed}")


def render_yaml(enabled_linters: list[str]) -> str:
    enable_lines = "\n".join(f"    - {name}" for name in enabled_linters)
    return f"""version: "2"

linters:
  default: none
  enable:
{enable_lines}

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
  uniq-by-line: true

output:
  path-mode: abs
  sort-order:
    - linter
    - file
  show-stats: false

run:
  relative-path-mode: gomod
  modules-download-mode: readonly
  tests: true

linters-settings:
  cyclop:
    max-complexity: 20
  gocyclo:
    min-complexity: 20

formatters:
  enable: []
"""


def main() -> None:
    args = parse_args()
    repo_root = Path(args.repo_root).expanduser().resolve()
    if not (repo_root / "go.mod").exists():
        raise SystemExit(f"repo root does not contain go.mod: {repo_root}")

    output_path = Path(args.output).expanduser()
    if not output_path.is_absolute():
        output_path = (repo_root / output_path).resolve()

    extra_enabled = flatten_csv_args(args.enable)
    disabled = flatten_csv_args(args.disable)
    if args.enable_typecheck:
        extra_enabled.append("typecheck")

    validate_linters(extra_enabled, "--enable")
    validate_linters(disabled, "--disable")

    enabled = list(DEFAULT_LINTERS)
    for item in extra_enabled:
        if item not in enabled:
            enabled.append(item)
    enabled = [item for item in enabled if item not in set(disabled)]

    if not enabled:
        raise SystemExit("enabled linter set is empty after applying --disable")

    if output_path.exists() and not args.force:
        raise SystemExit(f"output already exists: {output_path}. rerun with --force to overwrite")

    output_path.parent.mkdir(parents=True, exist_ok=True)

    log("bootstrap", "==========================================")
    log("bootstrap", "Code-Debt-Cleaner Bootstrap 开始运行")
    log("bootstrap", f"repo root: {repo_root}")
    log("bootstrap", f"output: {output_path}")
    log("bootstrap", f"enabled linters: {','.join(enabled)}")
    log("bootstrap", "==========================================")

    output_path.write_text(render_yaml(enabled))

    payload = {
        "bootstrapper": "code-debt-cleaner-bootstrap",
        "timestamp": utc_timestamp(),
        "repo_root": str(repo_root),
        "output": str(output_path),
        "enabled_linters": enabled,
    }
    print(json.dumps(payload, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
