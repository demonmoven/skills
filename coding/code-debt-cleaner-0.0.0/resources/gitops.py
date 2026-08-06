#!/usr/bin/env python3

from __future__ import annotations

import argparse
from datetime import datetime
from pathlib import Path
import subprocess
import sys

from common import log


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="gitops.py",
        description="Code-Debt-Cleaner GitOps - Git 操作脚本",
    )
    parser.add_argument("command", choices=["create-branch", "commit", "push", "all"])
    parser.add_argument("-b", "--branch", default="", help="指定分支名")
    parser.add_argument("-m", "--message", default="feat: clean_code_debt", help="指定提交信息")
    parser.add_argument(
        "--paths",
        nargs="*",
        default=[],
        help="commit 时要 stage 的文件路径列表；如果为空，需要显式传 --all",
    )
    parser.add_argument("--all", action="store_true", help="commit 时显式 stage 全部改动")
    parser.add_argument("-p", "--push", action="store_true", help="create-branch 或 commit 后自动推送")
    return parser


def run(command: list[str]) -> None:
    result = subprocess.run(command, check=False, text=True)
    if result.returncode != 0:
        raise SystemExit(result.returncode)


def create_branch(branch_name: str) -> str:
    if not branch_name:
        branch_name = f"code-debt-cleaner-{datetime.now().strftime('%Y%m%d-%H%M%S')}"
    log("gitops", "创建新分支...")
    log("gitops", f"  分支名: {branch_name}")
    run(["git", "checkout", "-b", branch_name])
    log("gitops", "✓ 分支创建成功")
    print(branch_name)
    return branch_name


def commit(message: str, paths: list[str], stage_all: bool) -> None:
    log("gitops", "提交修改...")
    log("gitops", f'  提交信息: "{message}"')
    if stage_all:
        run(["git", "add", "--all"])
    elif paths:
        run(["git", "add", "--", *paths])
    else:
        raise SystemExit("commit requires --paths ... or explicit --all")
    run(["git", "commit", "-m", message])
    log("gitops", "✓ 提交成功")


def push() -> None:
    log("gitops", "推送到远程...")
    run(["git", "push", "-u", "origin", "HEAD"])
    log("gitops", "✓ 推送成功")


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()

    log("gitops", "==========================================")
    log("gitops", "Code-Debt-Cleaner GitOps 开始运行")
    log("gitops", "==========================================")

    if args.command == "create-branch":
        create_branch(args.branch)
        if args.push:
            push()
        return

    if args.command == "commit":
        commit(args.message, args.paths, args.all)
        if args.push:
            push()
        return

    if args.command == "push":
        push()
        return

    create_branch(args.branch)
    commit(args.message, args.paths, args.all)
    push()


if __name__ == "__main__":
    main()
