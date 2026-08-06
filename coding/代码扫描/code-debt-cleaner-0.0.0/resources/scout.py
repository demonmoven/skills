#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

from common import (
    CODE_SMELL_TYPES,
    DEFAULT_LINTERS,
    DESCRIPTIONS,
    FORMATTER_TYPES,
    GOLANGCI_FAILURE_PATTERNS,
    SUPPORTED_TYPES,
    get_risk,
    log,
    split_csv,
    utc_timestamp,
    version_ge,
)


class ScoutError(Exception):
    def __init__(self, status: str, message: str, exit_code: int = 1) -> None:
        super().__init__(message)
        self.status = status
        self.message = message
        self.exit_code = exit_code


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="scout.py",
        description="Code-Debt-Cleaner Scout - 扫描 Go 项目中的代码债务",
    )
    parser.add_argument("-d", "--dir", default="./", help="指定扫描目录 (默认: ./)")
    parser.add_argument("-t", "--types", default="", help="指定要扫描的问题类型，多个类型用逗号分隔")
    parser.add_argument("-l", "--limit", type=int, default=0, help="限制扫描结果数量")
    parser.add_argument("-f", "--files", default="", help="只扫描指定文件，多个文件用逗号分隔")
    parser.add_argument("--config", default=".golangci.yml", help="指定仓库内 golangci 配置文件")
    parser.add_argument(
        "--report-json",
        default="/tmp/code-debt-cleaner-report.json",
        help="指定 JSON 结果输出路径",
    )
    parser.add_argument("--report-text", default="", help="指定 golangci 文本结果输出路径")
    parser.add_argument("--enable-typecheck", action="store_true", help="显式开启 typecheck")
    return parser.parse_args()


def read_go_mod_version(go_mod_path: Path) -> str:
    for line in go_mod_path.read_text().splitlines():
        if line.startswith("go "):
            return line.split()[1].strip()
    return ""


def read_go_runtime() -> str:
    if shutil.which("go") is None:
        return ""
    output = subprocess.run(["go", "version"], check=False, capture_output=True, text=True)
    parts = output.stdout.strip().split()
    if len(parts) >= 3:
        return parts[2].removeprefix("go")
    return ""


def read_golangci_version() -> str:
    if shutil.which("golangci-lint") is None:
        return ""
    output = subprocess.run(["golangci-lint", "--version"], check=False, capture_output=True, text=True)
    for token in output.stdout.split():
        if token.count(".") == 2 and token[0].isdigit():
            return token
    return ""


def emit_result(
    *,
    status: str,
    report_json: Path,
    toolchain: dict[str, str],
    command: str,
    reasons: list[str],
    issues: list[dict[str, str]],
) -> None:
    payload = {
        "scanner": "code-debt-cleaner-scout",
        "status": status,
        "timestamp": utc_timestamp(),
        "toolchain": toolchain,
        "command": command,
        "blocked_reasons": reasons,
        "issue_count": len(issues),
        "issues": issues,
    }
    report_json.parent.mkdir(parents=True, exist_ok=True)
    content = json.dumps(payload, ensure_ascii=False, indent=2)
    report_json.write_text(content + "\n")
    print(content)


def check_go_project(cwd: Path) -> None:
    log("scout", "检查是否在 Go 项目根目录...")
    if not (cwd / "go.mod").exists():
        raise ScoutError("error", "go.mod not found in current directory")
    log("scout", "✓ 找到 go.mod 文件，确认是 Go 项目")


def validate_requested_types(requested_types: list[str], enable_typecheck: bool) -> None:
    for issue_type in requested_types:
        if issue_type not in SUPPORTED_TYPES:
            raise ScoutError("error", f"unsupported scan type: {issue_type}")
        if issue_type in FORMATTER_TYPES:
            raise ScoutError(
                "error",
                f"formatter '{issue_type}' is not supported in the main scout pipeline; run formatter checks separately",
            )
        if issue_type == "typecheck" and not enable_typecheck:
            raise ScoutError(
                "error",
                "typecheck is disabled by default; rerun with --enable-typecheck to opt in",
            )


def handle_code_smells(requested_types: list[str]) -> None:
    smells = [issue_type for issue_type in requested_types if issue_type in CODE_SMELL_TYPES]
    if smells:
        log("scout", "⚠️  注意：以下代码坏味道类型需要人工识别，无法自动扫描：")
        log("scout", f"    {','.join(smells)}")
        log("scout", "    这些类型需要通过代码审查来识别，请手动检查代码或使用 IDE 进行分析")


def preflight_checks(
    *,
    config_path: Path,
    toolchain: dict[str, str],
) -> None:
    if not config_path.exists():
        raise ScoutError("error", f"golangci config file not found: {config_path}")
    if shutil.which("go") is None:
        raise ScoutError("blocked", "go command is not available in PATH")
    if shutil.which("golangci-lint") is None:
        raise ScoutError("blocked", "golangci-lint is not available in PATH; install the repo-pinned version first")

    go_mod = toolchain["go_mod"]
    go_runtime = toolchain["go_runtime"]
    if go_mod and go_runtime and not version_ge(go_runtime, go_mod):
        raise ScoutError("blocked", f"go.mod requires go >= {go_mod} (running go{go_runtime})")

    golangci_version = toolchain["golangci_lint"]
    if golangci_version and not golangci_version.startswith("2."):
        raise ScoutError("blocked", f"golangci-lint v2 is required, current version is {golangci_version}")


def build_linter_list(requested_types: list[str]) -> list[str]:
    if not requested_types:
        return list(DEFAULT_LINTERS)
    linters: list[str] = []
    for issue_type in requested_types:
        if issue_type == "duplicated_code":
            continue
        if issue_type in CODE_SMELL_TYPES:
            continue
        linters.append(issue_type)
    return linters


def normalize_scan_dirs(raw: str) -> list[str]:
    return [part for part in raw.split() if part]


def parse_golangci_issues(json_path: Path, cwd: Path) -> list[dict[str, str]]:
    priority_order = [
        "stylecheck",
        "gosimple",
        "ineffassign",
        "unused",
        "revive",
        "typecheck",
        "gocyclo",
        "cyclop",
        "errcheck",
        "govet",
        "staticcheck",
    ]

    def priority(linter: str) -> int:
        try:
            return priority_order.index(linter)
        except ValueError:
            return len(priority_order)

    if not json_path.exists() or not json_path.read_text().strip():
        return []

    data = json.loads(json_path.read_text())
    issues = data.get("Issues", [])
    seen: dict[tuple[str, str], int] = {}
    parsed: list[dict[str, str]] = []
    for issue in sorted(issues, key=lambda item: priority(item.get("FromLinter", ""))):
        issue_type = issue.get("FromLinter", "")
        pos = issue.get("Pos", {}) or {}
        filename = pos.get("Filename", "")
        if not issue_type or not filename:
            continue
        key = (filename, issue_type)
        if seen.get(key, 0) >= 3:
            continue
        seen[key] = seen.get(key, 0) + 1
        path = Path(filename)
        if not path.is_absolute():
            path = (cwd / path).resolve()
        parsed.append(
            {
                "type": issue_type,
                "file": str(path),
                "line": str(pos.get("Line", 0) or 0),
            }
        )
    return parsed


def classify_golangci_failure(stderr: str) -> str:
    for pattern in GOLANGCI_FAILURE_PATTERNS:
        if __import__("re").search(pattern, stderr):
            return "blocked"
    return "error"


def run_golangci(
    *,
    config_path: Path,
    linters: list[str],
    targets: list[str],
    report_text: str,
) -> tuple[int, str, Path, str]:
    stdout_file = Path(tempfile.mkstemp(prefix="cdc-golangci-", suffix=".json")[1])
    text_output = report_text or "stderr"
    command = [
        "golangci-lint",
        "run",
        "-c",
        str(config_path),
        "--default=none",
        f"--enable-only={','.join(linters)}",
        "--allow-serial-runners",
        f"--output.json.path={stdout_file}",
        f"--output.text.path={text_output}",
        *targets,
    ]
    proc = subprocess.run(command, check=False, capture_output=True, text=True)
    stderr = proc.stderr.strip()
    if stderr:
        log("scout", "扫描日志:")
        for line in stderr.splitlines():
            if line.strip():
                log("scout", f"  {line}")
    return proc.returncode, stderr, stdout_file, " ".join(command)


def scan_golangci(
    *,
    cwd: Path,
    scan_dirs: list[str],
    scan_files: list[str],
    config_path: Path,
    linters: list[str],
    report_text: str,
) -> tuple[list[dict[str, str]], str]:
    if not linters:
        log("scout", "没有需要 golangci-lint 扫描的 analyzer 类型")
        return [], ""

    log("scout", f"启用的 linters: {','.join(linters)}")

    aggregated: dict[tuple[str, str], dict[str, str]] = {}
    last_command = ""
    targets_by_dir = [scan_files] if scan_files else [[f"{scan_dir}/..." for scan_dir in scan_dirs]]

    for targets in targets_by_dir:
        for target in targets:
            if scan_files:
                display_target = target
                actual_targets = [target]
            else:
                display_target = target
                actual_targets = [target]

            log("scout", f"扫描目录: {display_target}")
            log("scout", "正在运行 golangci-lint 扫描...")
            exit_code, stderr, stdout_file, last_command = run_golangci(
                config_path=config_path,
                linters=linters,
                targets=actual_targets,
                report_text=report_text,
            )
            parsed = parse_golangci_issues(stdout_file, cwd)
            stdout_file.unlink(missing_ok=True)

            for issue in parsed:
                key = (issue["type"], issue["file"])
                if key not in aggregated:
                    aggregated[key] = {
                        "type": issue["type"],
                        "file": issue["file"],
                        "risk": get_risk(issue["type"]),
                        "description": DESCRIPTIONS.get(issue["type"], "Unknown issue type"),
                    }
                    line = issue["line"]
                    if line and line != "0":
                        log("scout", f"发现 {issue['type']}: {issue['file']}:{line}")
                    else:
                        log("scout", f"发现 {issue['type']}: {issue['file']}")

            if exit_code != 0 and not parsed:
                status = classify_golangci_failure(stderr)
                raise ScoutError(status, stderr or "golangci-lint failed without JSON issues")

    log("scout", f"golangci-lint 扫描完成，找到 {len(aggregated)} 个文件")
    return list(aggregated.values()), last_command


def scan_duplicate_code(
    *,
    cwd: Path,
    requested_types: list[str],
    scan_dirs: list[str],
    existing: list[dict[str, str]],
) -> tuple[list[dict[str, str]], str]:
    if "duplicated_code" not in requested_types:
        return existing, ""
    if shutil.which("dupl") is None:
        raise ScoutError("blocked", "dupl is not available in PATH; install the repo-pinned version first")

    log("scout", "扫描重复代码...")
    files: list[str] = []
    for scan_dir in scan_dirs:
        root = (cwd / scan_dir).resolve()
        for path in root.rglob("*.go"):
            if any(part in {".git", "vendor", "testdata", "mocks", "opp"} for part in path.parts):
                continue
            files.append(str(path))

    if not files:
        return existing, ""

    command = ["dupl", "-t", "100", *files]
    proc = subprocess.run(command, check=False, capture_output=True, text=True)
    seen = {(issue["type"], issue["file"]) for issue in existing}
    for line in proc.stdout.splitlines():
        if not line.strip():
            continue
        file_path = line.split(":", 1)[0]
        path = Path(file_path)
        if not path.is_absolute():
            path = (cwd / path).resolve()
        key = ("duplicated_code", str(path))
        if key in seen or not path.exists():
            continue
        seen.add(key)
        existing.append(
            {
                "type": "duplicated_code",
                "file": str(path),
                "risk": get_risk("duplicated_code"),
                "description": DESCRIPTIONS["duplicated_code"],
            }
        )
        log("scout", f"发现 duplicated_code: {path}")

    log("scout", f"重复代码扫描完成，找到 {sum(1 for issue in existing if issue['type'] == 'duplicated_code')} 个文件")
    return existing, " ".join(command)


def limit_issues(issues: list[dict[str, str]], max_issues: int, user_limit: int) -> list[dict[str, str]]:
    deduped: list[dict[str, str]] = []
    seen: set[tuple[str, str]] = set()
    for issue in issues:
        key = (issue["type"], issue["file"])
        if key in seen:
            continue
        seen.add(key)
        deduped.append(issue)
    if len(deduped) > max_issues:
        deduped = deduped[:max_issues]
    if user_limit and len(deduped) > user_limit:
        deduped = deduped[:user_limit]
    return deduped


def main() -> None:
    args = parse_args()
    cwd = Path.cwd()
    report_json = Path(args.report_json)
    config_path = Path(args.config)
    if not config_path.is_absolute():
        config_path = (cwd / config_path).resolve()

    requested_types = split_csv(args.types)
    scan_files = split_csv(args.files)
    scan_dirs = normalize_scan_dirs(args.dir)
    toolchain = {
        "go_mod": "",
        "go_runtime": "",
        "golangci_lint": "",
    }
    reasons: list[str] = []
    last_command = ""
    issues: list[dict[str, str]] = []

    log("scout", "==========================================")
    log("scout", "Code-Debt-Cleaner Scout 开始运行")
    log("scout", "⚠️  此脚本只扫描，不修改任何文件")
    log("scout", f"扫描目录: {args.dir}")
    log("scout", f"配置文件: {config_path}")
    log("scout", f"JSON 结果: {report_json}")
    if args.report_text:
        log("scout", f"文本结果: {args.report_text}")
    if requested_types:
        log("scout", f"扫描类型: {','.join(requested_types)}")
    if scan_files:
        log("scout", f"扫描文件: {','.join(scan_files)}")
    if args.limit:
        log("scout", f"结果限制: {args.limit}")
    if args.enable_typecheck:
        log("scout", "typecheck: enabled")
    log("scout", "==========================================")

    try:
        check_go_project(cwd)
        toolchain["go_mod"] = read_go_mod_version(cwd / "go.mod")
        toolchain["go_runtime"] = read_go_runtime()
        toolchain["golangci_lint"] = read_golangci_version()

        validate_requested_types(requested_types, args.enable_typecheck)
        handle_code_smells(requested_types)
        preflight_checks(config_path=config_path, toolchain=toolchain)

        linters = build_linter_list(requested_types)
        issues, last_command = scan_golangci(
            cwd=cwd,
            scan_dirs=scan_dirs,
            scan_files=scan_files,
            config_path=config_path,
            linters=linters,
            report_text=args.report_text,
        )
        issues, duplicate_command = scan_duplicate_code(
            cwd=cwd,
            requested_types=requested_types,
            scan_dirs=scan_dirs,
            existing=issues,
        )
        if duplicate_command:
            last_command = duplicate_command
        issues = limit_issues(issues, 20, args.limit)
    except ScoutError as exc:
        reasons.append(exc.message)
        emit_result(
            status=exc.status,
            report_json=report_json,
            toolchain=toolchain,
            command=last_command,
            reasons=reasons,
            issues=[],
        )
        raise SystemExit(exc.exit_code)

    log("scout", "==========================================")
    log("scout", f"扫描完成，共找到 {len(issues)} 个问题")
    log("scout", f"结果已写入: {report_json}")
    log("scout", "==========================================")

    emit_result(
        status="ok",
        report_json=report_json,
        toolchain=toolchain,
        command=last_command,
        reasons=reasons,
        issues=issues,
    )


if __name__ == "__main__":
    main()
