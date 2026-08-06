#!/usr/bin/env python3
"""
AI 友好度仓库扫描器
扫描团队仓库的 AI 友好度，输出评分到 CSV 和飞书多维表格。

用法:
  # 仅输出 CSV
  python3 scripts/scan_repos.py --input repos.txt --output results.csv

  # 输出到飞书多维表格
  python3 scripts/scan_repos.py --input repos.txt --bitable-app appXXX --bitable-table tblXXX

repos.txt 格式（每行一个仓库地址）:
  git@code.byted.org:douyin/sawyer.git
  git@code.byted.org:douyin/creator_ai_sawyer.git
"""

import argparse
import atexit
import csv
import glob
import hashlib
import json
import math
import os
import re
import shutil
import subprocess
import sys
import tempfile
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import asdict, dataclass, field
from datetime import datetime
from pathlib import Path

# ── 权重 ──
WEIGHT_CODE = 0.40
WEIGHT_TEST = 0.35
WEIGHT_CONTEXT = 0.25

# ── 排除目录/文件 ────────────────────────────────────
GO_EXCLUDE_DIRS = {"vendor", "kitex_gen", "third_party", ".git"}
GO_EXCLUDE_PATTERNS = {".gen.go", ".pb.go", "_mock.go", "wire_gen.go"}
TS_EXCLUDE_DIRS = {"node_modules", "dist", "build", ".next", ".git"}
TS_EXCLUDE_PATTERNS = {".d.ts"}
PY_EXCLUDE_DIRS = {"venv", ".venv", "__pycache__", ".tox", ".git"}
COMMON_EXCLUDE_DIRS = {".git", ".svn", ".hg"}


# ── 数据结构 ──────────────────────────────────────────
@dataclass
class RepoReport:
    repo_url: str = ""
    repo_name: str = ""
    languages: str = ""
    # 文件存在性（独立维度）
    claude_md_exists: bool = False
    claude_md_lines: int = 0
    architecture_doc_exists: bool = False
    architecture_doc_source: str = ""  # 架构文档来源（文件名或"README.md 中的章节"）
    architecture_doc_lines: int = 0
    sub_claude_md_count: int = 0
    # 内容质量（仅当文件存在时评估，由大模型打分）
    claude_md_quality: str = ""        # 待大模型评估
    architecture_doc_quality: str = "" # 待大模型评估
    context_score: float = 0.0        # 上下文文件得分（满分 5）
    # 代码架构
    total_source_files: int = 0
    files_over_500: int = 0
    files_over_500_ratio: float = 0.0
    avg_file_lines: float = 0.0
    type_safety_violations: int = 0
    code_score: float = 0.0
    # 测试架构
    test_files_count: int = 0
    test_ratio: float = 0.0
    has_ci_config: bool = False
    has_makefile_targets: bool = False
    test_score: float = 0.0
    # 代码质量（SlopCodeBench 论文指标）
    high_cc_functions: int = 0        # CC>10 的函数数
    max_cc: int = 0                   # 最大圈复杂度
    erosion: float = 0.0              # 结构侵蚀度 [0,1]
    verbosity: float = 0.0            # 冗余度 [0,1]
    anti_pattern_count: int = 0       # 反模式总数
    clone_ratio: float = 0.0          # 代码克隆率 [0,1]
    # 汇总
    total_score: float = 0.0
    grade: str = ""
    overall_summary: str = ""        # 整体说明（优势/短板/建议）
    scan_error: str = ""
    scan_time: str = ""


# ── 工具函数 ──────────────────────────────────────────
def parse_repo_name(url: str) -> str:
    """从 git URL 提取 org/repo 名称"""
    m = re.search(r"[:/]([^/]+/[^/]+?)(?:\.git)?$", url.strip())
    return m.group(1) if m else url.strip()


def is_binary(filepath: Path) -> bool:
    """快速判断文件是否为二进制"""
    try:
        with open(filepath, "rb") as f:
            chunk = f.read(8192)
            return b"\x00" in chunk
    except (OSError, IOError):
        return True


def count_lines(filepath: Path) -> int:
    """计算文件行数"""
    try:
        if is_binary(filepath):
            return 0
        with open(filepath, encoding="utf-8", errors="replace") as f:
            return sum(1 for _ in f)
    except (OSError, IOError):
        return 0


def read_text(filepath: Path) -> str:
    """安全读取文本文件"""
    try:
        if is_binary(filepath):
            return ""
        return filepath.read_text(encoding="utf-8", errors="replace")
    except (OSError, IOError):
        return ""


def should_exclude_dir(path: Path, exclude_dirs: set) -> bool:
    """检查路径是否在排除目录中"""
    return any(part in exclude_dirs for part in path.parts)


# ── 克隆管理 ──────────────────────────────────────────
def clone_repo(url: str, workdir: Path) -> Path | None:
    """浅克隆仓库，返回本地路径"""
    repo_name = parse_repo_name(url).replace("/", "_")
    dest = workdir / repo_name
    try:
        subprocess.run(
            ["git", "clone", "--depth", "1", "--single-branch", url.strip(), str(dest)],
            capture_output=True,
            timeout=120,
            check=True,
        )
        return dest
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as e:
        print(f"  ✗ 克隆失败: {url} ({e})", file=sys.stderr)
        return None


def clone_all(urls: list[str], workdir: Path, max_workers: int) -> dict[str, Path | None]:
    """并发克隆所有仓库"""
    results = {}
    total = len(urls)
    print(f"开始克隆 {total} 个仓库 (并发={max_workers})...")

    with ThreadPoolExecutor(max_workers=max_workers) as pool:
        futures = {pool.submit(clone_repo, url, workdir): url for url in urls}
        for i, future in enumerate(as_completed(futures), 1):
            url = futures[future]
            name = parse_repo_name(url)
            try:
                path = future.result()
                status = "✓" if path else "✗"
                print(f"  [{i}/{total}] {status} {name}")
                results[url] = path
            except Exception as e:
                print(f"  [{i}/{total}] ✗ {name}: {e}", file=sys.stderr)
                results[url] = None
    return results


# ── 语言检测 ──────────────────────────────────────────
def detect_languages(repo_path: Path) -> list[str]:
    """检测仓库使用的编程语言"""
    langs = []
    if (repo_path / "go.mod").exists():
        langs.append("Go")
    if (repo_path / "package.json").exists():
        langs.append("TypeScript")
    if any(
        (repo_path / f).exists()
        for f in ("requirements.txt", "pyproject.toml", "setup.py", "Pipfile")
    ):
        langs.append("Python")
    if (repo_path / "Cargo.toml").exists():
        langs.append("Rust")
    if any((repo_path / f).exists() for f in ("pom.xml", "build.gradle", "build.gradle.kts")):
        langs.append("Java")
    return langs if langs else ["Unknown"]


# ── 上下文架构扫描 ────────────────────────────────────
def scan_context(repo_path: Path) -> dict:
    """扫描上下文架构层"""
    # 检查 CLAUDE.md / AGENTS.md
    claude_md = None
    for name in ("CLAUDE.md", "AGENTS.md"):
        p = repo_path / name
        if p.exists():
            claude_md = p
            break

    claude_md_exists = claude_md is not None
    claude_md_lines = count_lines(claude_md) if claude_md else 0

    # 架构文档检测：扫描根目录和 docs/ 下所有 .md 文件，看内容是否包含架构描述
    arch_doc = None
    arch_doc_source = ""
    arch_heading_pattern = re.compile(
        r"^#{1,3}\s+.*(架构|[Aa]rchitecture|[Ss]ystem\s*[Dd]esign|技术方案|模块.{0,4}(设计|说明)|(项目|工程).{0,4}(结构|概览|目录)|[Pp]roject\s*[Ss]tructure|[Cc]ode\s*[Mm]ap|[Dd]irectory\s*[Ss]tructure|系统.{0,4}(概览|设计)).*$",
        re.MULTILINE,
    )

    # 1. 优先检查文件名本身就暗示架构的文件
    arch_name_pattern = re.compile(
        r"(architect|design|structure|架构|设计|概览)", re.IGNORECASE
    )
    for p in sorted(repo_path.glob("*.md")):
        if arch_name_pattern.search(p.stem):
            arch_doc = p
            arch_doc_source = p.name
            break

    # 2. 检查 docs/ doc/ 目录
    if not arch_doc:
        for docs_dir in ("docs", "doc"):
            d = repo_path / docs_dir
            if d.is_dir():
                for p in sorted(d.glob("*.md")):
                    if arch_name_pattern.search(p.stem):
                        arch_doc = p
                        arch_doc_source = f"{docs_dir}/{p.name}"
                        break
            if arch_doc:
                break

    # 3. 如果文件名没命中，扫描根目录所有 .md 文件内容，看是否有架构相关章节
    if not arch_doc:
        for p in sorted(repo_path.glob("*.md")):
            if p.name.upper().startswith("CLAUDE") or p.name.upper().startswith("AGENT"):
                continue  # 跳过 AI 上下文文件
            content = read_text(p)
            headings = arch_heading_pattern.findall(content)
            if headings:
                arch_doc = p
                arch_doc_source = f"{p.name}（含架构章节）"
                break

    architecture_doc_exists = arch_doc is not None
    architecture_doc_lines = count_lines(arch_doc) if arch_doc else 0

    # 子目录 CLAUDE.md 数量
    sub_claude_count = 0
    for p in repo_path.rglob("CLAUDE.md"):
        if p.parent != repo_path and ".git" not in p.parts:
            sub_claude_count += 1
    for p in repo_path.rglob("AGENTS.md"):
        if p.parent != repo_path and ".git" not in p.parts:
            if not (p.is_symlink() and p.resolve().name == "CLAUDE.md"):
                sub_claude_count += 1

    # 上下文文件得分（满分 5）
    # 无 CLAUDE.md/AGENTS.md → 0 分（且总分直接归零）
    ctx_score = 0.0
    if claude_md_exists:
        ctx_score += 2.0
        # CLAUDE.md 质量加分（质量由大模型单独评估，扫描时暂不可用）
        # A 级 +1.0，B 级 +0.5，后续通过质量评估回填
        if architecture_doc_exists:
            ctx_score += 1.5
        if sub_claude_count > 0:
            ctx_score += 0.5
    ctx_score = min(ctx_score, 5.0)

    return {
        "claude_md_exists": claude_md_exists,
        "claude_md_lines": claude_md_lines,
        "architecture_doc_exists": architecture_doc_exists,
        "architecture_doc_source": arch_doc_source,
        "architecture_doc_lines": architecture_doc_lines,
        "sub_claude_md_count": sub_claude_count,
        "context_score": round(ctx_score, 1),
    }


# ── 代码架构扫描 ──────────────────────────────────────
def _collect_source_files(repo_path: Path, languages: list[str]) -> list[Path]:
    """收集源代码文件，排除生成代码和依赖"""
    files = []
    seen = set()

    def add_files(pattern: str, exclude_dirs: set, exclude_patterns: set):
        for p in repo_path.rglob(pattern):
            if p in seen or not p.is_file():
                continue
            if should_exclude_dir(p.relative_to(repo_path), exclude_dirs):
                continue
            if any(str(p).endswith(pat) for pat in exclude_patterns):
                continue
            seen.add(p)
            files.append(p)

    if "Go" in languages:
        add_files("*.go", GO_EXCLUDE_DIRS, GO_EXCLUDE_PATTERNS)
    if "TypeScript" in languages:
        for ext in ("*.ts", "*.tsx"):
            add_files(ext, TS_EXCLUDE_DIRS, TS_EXCLUDE_PATTERNS)
    if "Python" in languages:
        add_files("*.py", PY_EXCLUDE_DIRS, set())
    if "Java" in languages:
        add_files("*.java", COMMON_EXCLUDE_DIRS, set())
    if "Rust" in languages:
        add_files("*.rs", COMMON_EXCLUDE_DIRS | {"target"}, set())
    if "Unknown" in languages:
        # 尝试常见扩展名
        for ext in ("*.go", "*.ts", "*.tsx", "*.py", "*.java", "*.rs", "*.js", "*.jsx"):
            add_files(ext, COMMON_EXCLUDE_DIRS | {"vendor", "node_modules"}, set())

    return files


def _count_type_violations(repo_path: Path, languages: list[str]) -> int:
    """统计业务代码中的类型安全违规"""
    violations = 0

    if "Go" in languages:
        # 只检查 service/ 和 handler/ 目录
        go_pattern = re.compile(r"\binterface\{\}|\bany\b|map\[string\]interface\{\}")
        for dirn in ("service", "handler"):
            dir_path = repo_path / dirn
            if not dir_path.is_dir():
                continue
            for p in dir_path.rglob("*.go"):
                if should_exclude_dir(p.relative_to(repo_path), GO_EXCLUDE_DIRS):
                    continue
                if any(str(p).endswith(pat) for pat in GO_EXCLUDE_PATTERNS):
                    continue
                content = read_text(p)
                violations += len(go_pattern.findall(content))

    if "TypeScript" in languages:
        # 检查 src/ 目录
        ts_pattern = re.compile(r":\s*any\b|as\s+any\b|<any>")
        src_path = repo_path / "src"
        if not src_path.is_dir():
            # 尝试常见的前端项目结构
            for candidate in repo_path.rglob("src"):
                if candidate.is_dir() and not should_exclude_dir(
                    candidate.relative_to(repo_path), TS_EXCLUDE_DIRS
                ):
                    src_path = candidate
                    break
        if src_path.is_dir():
            for ext in ("*.ts", "*.tsx"):
                for p in src_path.rglob(ext):
                    if should_exclude_dir(p.relative_to(repo_path), TS_EXCLUDE_DIRS):
                        continue
                    if any(str(p).endswith(pat) for pat in TS_EXCLUDE_PATTERNS):
                        continue
                    content = read_text(p)
                    violations += len(ts_pattern.findall(content))

    return violations


def scan_code(repo_path: Path, languages: list[str]) -> dict:
    """扫描代码架构层"""
    source_files = _collect_source_files(repo_path, languages)
    total = len(source_files)

    # 统计行数
    file_lines = []
    over_500 = 0
    for f in source_files:
        lines = count_lines(f)
        file_lines.append(lines)
        if lines > 500:
            over_500 += 1

    avg_lines = sum(file_lines) / len(file_lines) if file_lines else 0
    ratio = over_500 / total if total > 0 else 0

    # 类型安全违规
    violations = _count_type_violations(repo_path, languages)

    # 评分（从 5 分扣减）
    score = 5.0
    if ratio > 0.15:
        score -= 1.0
    elif ratio > 0.05:
        score -= 0.5
    if violations > 20:
        score -= 1.5
    elif violations > 10:
        score -= 1.0
    elif violations > 5:
        score -= 0.5
    if avg_lines > 300:
        score -= 0.5
    if total == 0:
        score = 0.0
    score = max(score, 0.0)

    return {
        "total_source_files": total,
        "files_over_500": over_500,
        "files_over_500_ratio": round(ratio, 3),
        "avg_file_lines": round(avg_lines, 1),
        "type_safety_violations": violations,
        "code_score": round(score, 1),
    }


# ── 测试架构扫描 ──────────────────────────────────────
def scan_test(repo_path: Path, languages: list[str], total_source: int) -> dict:
    """扫描测试架构层"""
    test_files = set()

    if "Go" in languages:
        for p in repo_path.rglob("*_test.go"):
            if not should_exclude_dir(p.relative_to(repo_path), GO_EXCLUDE_DIRS):
                test_files.add(p)

    if "TypeScript" in languages:
        for pattern in ("*.test.ts", "*.test.tsx", "*.spec.ts", "*.spec.tsx"):
            for p in repo_path.rglob(pattern):
                if not should_exclude_dir(p.relative_to(repo_path), TS_EXCLUDE_DIRS):
                    test_files.add(p)

    if "Python" in languages:
        for pattern in ("test_*.py", "*_test.py"):
            for p in repo_path.rglob(pattern):
                if not should_exclude_dir(p.relative_to(repo_path), PY_EXCLUDE_DIRS):
                    test_files.add(p)
        # tests/ 目录下的 .py 文件
        tests_dir = repo_path / "tests"
        if tests_dir.is_dir():
            for p in tests_dir.rglob("*.py"):
                test_files.add(p)

    test_count = len(test_files)
    test_ratio = test_count / total_source if total_source > 0 else 0

    # Makefile targets
    has_makefile_targets = False
    makefile = repo_path / "Makefile"
    if makefile.exists():
        content = read_text(makefile)
        targets = {"test", "lint", "build"}
        found = sum(1 for t in targets if re.search(rf"^{t}\s*:", content, re.MULTILINE))
        has_makefile_targets = found >= 2

    # CI 配置（含字节内部 .codebase/pipelines）
    has_ci = (
        (repo_path / ".codebase" / "pipelines").is_dir()
        or (repo_path / ".gitlab-ci.yml").exists()
        or (repo_path / ".github" / "workflows").is_dir()
        or (repo_path / "Jenkinsfile").exists()
        or (repo_path / ".circleci").is_dir()
    )

    # build.sh (字节跳动项目常用)
    has_build_script = (repo_path / "build.sh").exists()

    # 评分
    score = 0.0
    if test_count > 0:
        score += 2.0
    if test_ratio > 0.10:
        score += 1.5
    elif test_ratio > 0.05:
        score += 0.5
    if has_ci:
        score += 1.0
    if has_makefile_targets or has_build_script:
        score += 0.5
    score = min(score, 5.0)

    return {
        "test_files_count": test_count,
        "test_ratio": round(test_ratio, 3),
        "has_ci_config": has_ci,
        "has_makefile_targets": has_makefile_targets or has_build_script,
        "test_score": round(score, 1),
    }


# ── 代码质量指标（SlopCodeBench 论文启发）──────────────

CC_THRESHOLD = 10   # 高复杂度函数阈值
CLONE_WINDOW = 6    # 克隆检测滑动窗口行数


def _extract_brace_block(lines: list[str], start: int, max_lines: int = 500) -> list[str]:
    """从 start 行开始，提取到匹配闭合大括号为止的代码块"""
    brace_depth = 0
    started = False
    block = []
    for j in range(start, min(start + max_lines, len(lines))):
        line = lines[j]
        block.append(line)
        for ch in line:
            if ch == '{':
                brace_depth += 1
                started = True
            elif ch == '}':
                brace_depth -= 1
        if started and brace_depth <= 0:
            break
    return block


def _clean_for_cc(content: str) -> str:
    """去除字符串字面量和注释，避免正则误匹配"""
    s = re.sub(r'"(?:[^"\\]|\\.)*"', '""', content)
    s = re.sub(r'`[^`]*`', '``', s)
    s = re.sub(r"'(?:[^'\\]|\\.)*'", "''", s)
    s = re.sub(r'//.*$', '', s, flags=re.MULTILINE)
    s = re.sub(r'/\*.*?\*/', '', s, flags=re.DOTALL)
    return s


def _calc_cc_go(func_body: str) -> int:
    """计算 Go 函数体的圈复杂度"""
    clean = _clean_for_cc(func_body)
    cc = 1
    cc += len(re.findall(r'\bif\b', clean))
    cc += len(re.findall(r'\bfor\b', clean))
    cc += len(re.findall(r'\bcase\b', clean))
    cc += len(re.findall(r'&&', clean))
    cc += len(re.findall(r'\|\|', clean))
    return cc


def _calc_cc_ts(func_body: str) -> int:
    """计算 TypeScript 函数体的圈复杂度"""
    clean = _clean_for_cc(func_body)
    cc = 1
    cc += len(re.findall(r'\bif\b', clean))
    cc += len(re.findall(r'\bfor\b', clean))
    cc += len(re.findall(r'\bwhile\b', clean))
    cc += len(re.findall(r'\bcase\b', clean))
    cc += len(re.findall(r'\bcatch\b', clean))
    cc += len(re.findall(r'&&', clean))
    cc += len(re.findall(r'\|\|', clean))
    cc += len(re.findall(r'\?\?', clean))
    return cc


def _extract_functions_go(content: str) -> list[dict]:
    """从 Go 源码提取函数列表 [{name, sloc, cc}]"""
    functions = []
    lines = content.split('\n')
    func_re = re.compile(r'func\s+(?:\([^)]*\)\s+)?(\w+)\s*\(')
    i = 0
    while i < len(lines):
        m = func_re.match(lines[i].lstrip())
        if not m:
            i += 1
            continue
        name = m.group(1)
        block = _extract_brace_block(lines, i)
        body = '\n'.join(block)
        sloc = sum(1 for l in block if l.strip() and not l.strip().startswith('//'))
        cc = _calc_cc_go(body)
        functions.append({"name": name, "sloc": max(sloc, 1), "cc": cc})
        i += len(block)
    return functions


def _extract_functions_ts(content: str) -> list[dict]:
    """从 TypeScript/JavaScript 源码提取函数列表（简化版）"""
    functions = []
    lines = content.split('\n')
    patterns = [
        re.compile(r'(?:export\s+)?(?:async\s+)?function\s+(\w+)'),
        re.compile(r'(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\('),
        re.compile(r'^\s+(?:async\s+)?(\w+)\s*\([^)]*\)\s*(?::\s*[^{]+)?\s*\{'),
    ]
    skip = {'if', 'for', 'while', 'switch', 'catch', 'return', 'new', 'throw',
            'import', 'from', 'require', 'else', 'class', 'super', 'this'}
    i = 0
    while i < len(lines):
        name = None
        for pat in patterns:
            m = pat.search(lines[i])
            if m:
                name = m.group(1)
                break
        if not name or name in skip:
            i += 1
            continue
        if '{' not in lines[i] and (i + 1 >= len(lines) or '{' not in lines[i + 1]):
            i += 1
            continue
        block = _extract_brace_block(lines, i)
        body = '\n'.join(block)
        sloc = sum(1 for l in block if l.strip() and not l.strip().startswith('//'))
        cc = _calc_cc_ts(body)
        functions.append({"name": name, "sloc": max(sloc, 1), "cc": cc})
        i += len(block)
    return functions


def _count_anti_patterns(filepath: Path, lang: str) -> int:
    """统计文件中的反模式行数"""
    content = read_text(filepath)
    if not content:
        return 0
    flagged = set()
    lines = content.split('\n')

    if lang == "Go":
        for i, line in enumerate(lines):
            s = line.strip()
            # 原始错误返回（未 wrap）
            if re.match(r'return\s+(nil,\s*)?err\s*$', s):
                flagged.add(i)
            # 空错误处理块
            if re.match(r'if\s+err\s*!=\s*nil\s*\{\s*\}', s):
                flagged.add(i)
            # TODO/FIXME/HACK
            if re.search(r'//\s*(TODO|FIXME|HACK|XXX)\b', s, re.IGNORECASE):
                flagged.add(i)
            # 深层嵌套（≥5 层 tab 缩进）
            indent = len(line) - len(line.lstrip('\t'))
            if indent >= 5 and s and not s.startswith('//'):
                flagged.add(i)
            # 被注释掉的代码
            if re.match(r'//\s*(if|for|func|return|var|const|type)\b', s):
                flagged.add(i)

    elif lang == "TypeScript":
        is_test = 'test' in str(filepath).lower() or 'spec' in str(filepath).lower()
        for i, line in enumerate(lines):
            s = line.strip()
            # console.log（非测试文件）
            if not is_test and re.search(r'\bconsole\.(log|debug|info)\b', s):
                flagged.add(i)
            # 空 catch 块
            if re.match(r'catch\s*\([^)]*\)\s*\{\s*\}', s):
                flagged.add(i)
            # @ts-ignore / @ts-nocheck
            if '// @ts-ignore' in s or '// @ts-nocheck' in s:
                flagged.add(i)
            # TODO/FIXME/HACK
            if re.search(r'//\s*(TODO|FIXME|HACK|XXX)\b', s, re.IGNORECASE):
                flagged.add(i)
            # 非空断言
            if re.search(r'\w+!\.\w', s) or re.search(r'\w+!\[', s):
                flagged.add(i)
            # 深层嵌套（≥5 层 4-space 缩进）
            indent = (len(line) - len(line.lstrip(' '))) // 4
            if indent >= 5 and s and not s.startswith('//'):
                flagged.add(i)

    return len(flagged)


def _detect_clones(source_files: list[Path], window_size: int = CLONE_WINDOW) -> tuple[int, int]:
    """检测跨文件代码克隆，返回 (克隆行数, 有效总行数)"""
    window_hashes: dict[str, list[tuple[int, int]]] = {}  # hash -> [(file_idx, line_no)]
    all_normalized: list[list[str]] = []
    total_lines = 0

    for fi, filepath in enumerate(source_files[:300]):
        content = read_text(filepath)
        if not content:
            all_normalized.append([])
            continue
        normalized = []
        for line in content.split('\n'):
            s = line.strip()
            if not s or s.startswith('//') or s.startswith('#') or s.startswith('/*'):
                continue
            if s.startswith('import ') or s.startswith('package ') or s.startswith('from '):
                continue
            if len(s) < 10:
                continue
            normalized.append(re.sub(r'\s+', ' ', s))
        all_normalized.append(normalized)
        total_lines += len(normalized)

        for i in range(len(normalized) - window_size + 1):
            window = '\n'.join(normalized[i : i + window_size])
            h = hashlib.md5(window.encode()).hexdigest()
            if h not in window_hashes:
                window_hashes[h] = []
            window_hashes[h].append((fi, i))

    clone_positions: set[tuple[int, int]] = set()
    for positions in window_hashes.values():
        if len(positions) > 1:
            for fi, start in positions:
                for offset in range(window_size):
                    clone_positions.add((fi, start + offset))

    return len(clone_positions), max(total_lines, 1)


def scan_quality(repo_path: Path, languages: list[str], source_files: list[Path]) -> dict:
    """扫描代码质量指标（SlopCodeBench 论文启发）

    返回:
        high_cc_functions: CC>10 的函数数
        max_cc:            最大圈复杂度
        erosion:           结构侵蚀度 — 高复杂度函数 mass 占总 mass 的比例
        verbosity:         冗余度 — (反模式行 + 克隆行) / 总 LOC
        anti_pattern_count: 反模式总数
        clone_ratio:       代码克隆率
    """
    all_functions: list[dict] = []
    anti_pattern_total = 0
    total_loc = 0

    for filepath in source_files:
        total_loc += count_lines(filepath)
        content = read_text(filepath)
        if not content:
            continue

        fname = str(filepath)
        if fname.endswith('.go') and "Go" in languages:
            all_functions.extend(_extract_functions_go(content))
            anti_pattern_total += _count_anti_patterns(filepath, "Go")
        elif any(fname.endswith(ext) for ext in ('.ts', '.tsx', '.js', '.jsx')):
            all_functions.extend(_extract_functions_ts(content))
            anti_pattern_total += _count_anti_patterns(filepath, "TypeScript")

    # 圈复杂度
    high_cc = [f for f in all_functions if f["cc"] > CC_THRESHOLD]
    max_cc_val = max((f["cc"] for f in all_functions), default=0)

    # 结构侵蚀度: mass(f) = CC(f) × √SLOC(f)
    total_mass = sum(f["cc"] * math.sqrt(f["sloc"]) for f in all_functions) if all_functions else 0
    high_mass = sum(f["cc"] * math.sqrt(f["sloc"]) for f in high_cc)
    erosion = high_mass / total_mass if total_mass > 0 else 0.0

    # 代码克隆
    clone_lines, clone_total = _detect_clones(source_files)
    clone_ratio = clone_lines / clone_total if clone_total > 0 else 0.0

    # 冗余度 = (反模式行 ∪ 克隆行) / LOC
    verbosity = min((anti_pattern_total + clone_lines) / total_loc if total_loc > 0 else 0.0, 1.0)

    return {
        "high_cc_functions": len(high_cc),
        "max_cc": max_cc_val,
        "erosion": round(erosion, 3),
        "verbosity": round(verbosity, 3),
        "anti_pattern_count": anti_pattern_total,
        "clone_ratio": round(clone_ratio, 3),
    }


# ── 单仓库扫描 ───────────────────────────────────────
def scan_repo(url: str, repo_path: Path) -> RepoReport:
    """扫描单个仓库，返回完整报告"""
    report = RepoReport(
        repo_url=url.strip(),
        repo_name=parse_repo_name(url),
        scan_time=datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
    )

    try:
        languages = detect_languages(repo_path)
        report.languages = ", ".join(languages)

        # 扫描
        ctx = scan_context(repo_path)
        code = scan_code(repo_path, languages)
        test = scan_test(repo_path, languages, code["total_source_files"])
        quality = scan_quality(
            repo_path, languages,
            _collect_source_files(repo_path, languages),
        )

        # 填充报告
        for k, v in {**ctx, **code, **test, **quality}.items():
            setattr(report, k, v)

        # 总分：无 CLAUDE.md/AGENTS.md → 直接 0 分
        if not ctx["claude_md_exists"]:
            report.total_score = 0.0
        else:
            report.total_score = round(
                code["code_score"] * WEIGHT_CODE
                + test["test_score"] * WEIGHT_TEST
                + ctx["context_score"] * WEIGHT_CONTEXT,
                1,
            )

        # 等级
        if report.total_score >= 4.0:
            report.grade = "A"
        elif report.total_score >= 3.0:
            report.grade = "B"
        elif report.total_score >= 2.0:
            report.grade = "C"
        else:
            report.grade = "D"

    except Exception as e:
        report.scan_error = str(e)
        report.grade = "D"

    return report


# ── CSV 输出 ──────────────────────────────────────────
def generate_html_report(reports: list[RepoReport], scan_id: str, history_dir: str, output_path: str):
    """生成 HTML 可视化报告"""
    # ── SNAPSHOT 数据 ──
    snapshot = []
    for r in reports:
        snapshot.append({
            "repo_url": r.repo_url,
            "repo_name": r.repo_name,
            "languages": r.languages,
            "claude_md_exists": r.claude_md_exists,
            "claude_md_lines": r.claude_md_lines,
            "claude_md_quality": r.claude_md_quality,
            "architecture_doc_exists": r.architecture_doc_exists,
            "architecture_doc_source": r.architecture_doc_source,
            "architecture_doc_lines": r.architecture_doc_lines,
            "architecture_doc_quality": r.architecture_doc_quality,
            "sub_claude_md_count": r.sub_claude_md_count,
            "total_source_files": r.total_source_files,
            "files_over_500": r.files_over_500,
            "files_over_500_ratio": r.files_over_500_ratio,
            "avg_file_lines": r.avg_file_lines,
            "type_safety_violations": r.type_safety_violations,
            "code_score": r.code_score,
            "test_files_count": r.test_files_count,
            "test_ratio": r.test_ratio,
            "has_ci_config": r.has_ci_config,
            "has_makefile_targets": r.has_makefile_targets,
            "test_score": r.test_score,
            "context_score": r.context_score,
            "high_cc_functions": r.high_cc_functions,
            "max_cc": r.max_cc,
            "erosion": r.erosion,
            "verbosity": r.verbosity,
            "anti_pattern_count": r.anti_pattern_count,
            "clone_ratio": r.clone_ratio,
            "total_score": r.total_score,
            "grade": r.grade,
            "overall_summary": r.overall_summary,
            "scan_error": r.scan_error,
            "scan_time": r.scan_time,
        })

    # ── HISTORY 数据 ──
    history = []
    scan_ids = set()
    csv_pattern = os.path.join(history_dir, "scan_*.csv")
    for csv_path in glob.glob(csv_pattern):
        filename = os.path.basename(csv_path)
        # 从文件名提取 scan_id，例如 scan_2026-04-06.csv → 2026-04-06
        match = re.match(r"scan_(.+)\.csv$", filename)
        if not match:
            continue
        file_scan_id = match.group(1)
        scan_ids.add(file_scan_id)
        try:
            with open(csv_path, encoding="utf-8-sig") as f:
                reader = csv.DictReader(f)
                for row in reader:
                    try:
                        history.append({
                            "scan_id": file_scan_id,
                            "repo_name": row.get("repo_name", ""),
                            "grade": row.get("grade", ""),
                            "total_score": float(row.get("total_score", 0)),
                            "code_score": float(row.get("code_score", 0)),
                            "test_score": float(row.get("test_score", 0)),
                            "context_score": float(row.get("context_score", 0)),
                            "claude_md_exists": row.get("claude_md_exists", "").lower() == "true",
                            "architecture_doc_exists": row.get("architecture_doc_exists", "").lower() == "true",
                            "erosion": float(row.get("erosion", 0) or 0),
                            "verbosity": float(row.get("verbosity", 0) or 0),
                            "clone_ratio": float(row.get("clone_ratio", 0) or 0),
                            "high_cc_functions": int(row.get("high_cc_functions", 0) or 0),
                            "anti_pattern_count": int(row.get("anti_pattern_count", 0) or 0),
                        })
                    except (ValueError, KeyError):
                        continue
        except (OSError, csv.Error):
            continue

    # 确保当前 scan_id 也在列表中
    scan_ids.add(scan_id)

    # ── META 数据 ──
    meta = {
        "scan_id": scan_id,
        "scan_time": scan_id,
        "total_repos": len(reports),
        "scan_ids": sorted(scan_ids),
    }

    # ── 读取模板并替换 ──
    script_dir = os.path.dirname(os.path.abspath(__file__))
    template_path = os.path.join(script_dir, "report_template.html")
    with open(template_path, encoding="utf-8") as f:
        html = f.read()

    html = html.replace("__SNAPSHOT_DATA__", json.dumps(snapshot, ensure_ascii=False))
    html = html.replace("__HISTORY_DATA__", json.dumps(history, ensure_ascii=False))
    html = html.replace("__META_DATA__", json.dumps(meta, ensure_ascii=False))

    with open(output_path, "w", encoding="utf-8") as f:
        f.write(html)


def write_csv(reports: list[RepoReport], output_path: str):
    """输出扫描结果到 CSV"""
    if not reports:
        return
    fieldnames = list(asdict(reports[0]).keys())
    with open(output_path, "w", newline="", encoding="utf-8-sig") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        for r in reports:
            writer.writerow(asdict(r))
    print(f"\nCSV 已保存: {output_path}")


# ── 飞书多维表格输出 ──────────────────────────────────
def push_to_bitable(reports: list[RepoReport], app_token: str, table_id: str):
    """将结果写入飞书多维表格"""
    print(f"\n正在写入飞书多维表格 (app={app_token}, table={table_id})...")

    # 构建记录（仅包含表中已有的字段，类型需匹配表定义）
    records = []
    for r in reports:
        # 仓库地址是 URL 类型 text 字段，需要 {text, link} 对象格式
        repo_url_obj = {"text": r.repo_name, "link": r.repo_url.replace("git@code.byted.org:", "https://code.byted.org/").replace(".git", "")} if r.repo_url else {"text": "", "link": ""}
        fields = {
            "仓库地址": repo_url_obj,
            "仓库名称": r.repo_name,
            "语言": r.languages,
            "CLAUDE.md": r.claude_md_exists,
            "CLAUDE.md 行数": r.claude_md_lines,
            "CLAUDE.md 质量": r.claude_md_quality or "",
            "架构文档": r.architecture_doc_exists,
            "架构文档来源": r.architecture_doc_source or "",
            "架构文档行数": r.architecture_doc_lines,
            "架构文档质量": r.architecture_doc_quality or "",
            "子目录 CLAUDE.md": r.sub_claude_md_count,
            "源文件数": r.total_source_files,
            "超500行文件数": r.files_over_500,
            "超500行占比": round(r.files_over_500_ratio, 3),
            "平均行数": r.avg_file_lines,
            "类型违规数": r.type_safety_violations,
            "代码得分": r.code_score,
            "测试文件数": r.test_files_count,
            "测试文件占比": round(r.test_ratio, 3),
            "CI 配置": r.has_ci_config,
            "构建入口": r.has_makefile_targets,
            "测试得分": r.test_score,
            "上下文得分": r.context_score,
            "高复杂度函数数": r.high_cc_functions,
            "最大圈复杂度": r.max_cc,
            "结构侵蚀度": r.erosion or "",
            "冗余度": r.verbosity or "",
            "反模式数": r.anti_pattern_count,
            "代码克隆率": r.clone_ratio or "",
            "总分": r.total_score,
            "等级": r.grade,
            "整体说明": r.overall_summary or "",
            "错误信息": r.scan_error or "",
        }
        # 扫描时间是 datetime 类型，需要传毫秒时间戳
        if r.scan_time:
            try:
                dt = datetime.strptime(r.scan_time, "%Y-%m-%d %H:%M:%S")
                fields["扫描时间"] = int(dt.timestamp() * 1000)
            except (ValueError, TypeError):
                pass
        records.append({"fields": fields})

    # 分批写入（每批最多 50 条，避免 payload 过大导致 lark-cli 静默失败）
    batch_size = 50
    for i in range(0, len(records), batch_size):
        batch = records[i : i + batch_size]
        payload = json.dumps({"records": batch}, ensure_ascii=False)

        try:
            result = subprocess.run(
                [
                    "lark-cli", "api", "POST",
                    f"/open-apis/bitable/v1/apps/{app_token}/tables/{table_id}/records/batch_create",
                    "--data", payload,
                ],
                capture_output=True,
                text=True,
                timeout=60,
            )
            output = result.stdout or result.stderr or ""
            resp = json.loads(output) if output.strip() else {}
            if resp.get("code") == 0:
                print(f"  ✓ 写入 {len(batch)} 条记录成功")
            else:
                print(f"  ✗ 写入失败: {resp.get('msg', result.stderr)}", file=sys.stderr)
        except Exception as e:
            print(f"  ✗ 写入异常: {e}", file=sys.stderr)


# ── 多维表格清空 ──────────────────────────────────────
def clear_bitable_table(app_token: str, table_id: str):
    """清空快照表的所有记录，为重新写入做准备"""
    print(f"  正在清空快照表...")
    all_ids = []
    offset = 0
    limit = 500

    while True:
        try:
            result = subprocess.run(
                [
                    "lark-cli", "base", "+record-list",
                    "--base-token", app_token,
                    "--table-id", table_id,
                    "--limit", str(limit),
                    "--offset", str(offset),
                ],
                capture_output=True, text=True, timeout=30,
            )
            resp = json.loads(result.stdout) if result.stdout else {}
            data = resp.get("data", {})
            ids = data.get("record_id_list", [])
            all_ids.extend(ids)
            if not data.get("has_more") or not ids:
                break
            offset += len(ids)
        except Exception as e:
            print(f"  ✗ 列举记录失败: {e}", file=sys.stderr)
            break

    if not all_ids:
        print(f"  表中无记录，跳过清空")
        return

    for i in range(0, len(all_ids), 500):
        batch = all_ids[i : i + 500]
        try:
            result = subprocess.run(
                [
                    "lark-cli", "api", "POST",
                    f"/open-apis/bitable/v1/apps/{app_token}/tables/{table_id}/records/batch_delete",
                    "--data", json.dumps({"records": batch}),
                ],
                capture_output=True, text=True, timeout=30,
            )
            resp = json.loads(result.stdout) if result.stdout else {}
            if resp.get("code") == 0:
                print(f"  ✓ 删除 {len(batch)} 条记录")
            else:
                print(f"  ✗ 删除失败: {resp.get('msg', result.stderr)}", file=sys.stderr)
        except Exception as e:
            print(f"  ✗ 删除异常: {e}", file=sys.stderr)

    print(f"  ✓ 已清空 {len(all_ids)} 条记录")


# ── 历史记录写入 ──────────────────────────────────────
def push_to_history(reports: list[RepoReport], app_token: str, table_id: str, scan_id: str):
    """将扫描结果追加到历史记录表（用于趋势追踪）"""
    print(f"\n正在写入历史记录表 (scan_id={scan_id})...")

    records = []
    for r in reports:
        repo_url_obj = {"text": r.repo_name, "link": r.repo_url.replace("git@code.byted.org:", "https://code.byted.org/").replace(".git", "")} if r.repo_url else {"text": "", "link": ""}
        fields = {
            "扫描批次": scan_id,
            "仓库地址": repo_url_obj,
            "仓库名称": r.repo_name,
            "语言": r.languages,
            "CLAUDE.md": r.claude_md_exists,
            "CLAUDE.md 行数": r.claude_md_lines,
            "CLAUDE.md 质量": r.claude_md_quality or "",
            "架构文档": r.architecture_doc_exists,
            "架构文档来源": r.architecture_doc_source or "",
            "架构文档行数": r.architecture_doc_lines,
            "架构文档质量": r.architecture_doc_quality or "",
            "子目录 CLAUDE.md": r.sub_claude_md_count,
            "源文件数": r.total_source_files,
            "超500行文件数": r.files_over_500,
            "超500行占比": round(r.files_over_500_ratio, 3),
            "平均行数": r.avg_file_lines,
            "类型违规数": r.type_safety_violations,
            "代码得分": r.code_score,
            "测试文件数": r.test_files_count,
            "测试文件占比": round(r.test_ratio, 3),
            "CI 配置": r.has_ci_config,
            "构建入口": r.has_makefile_targets,
            "测试得分": r.test_score,
            "上下文得分": r.context_score,
            "高复杂度函数数": r.high_cc_functions,
            "最大圈复杂度": r.max_cc,
            "结构侵蚀度": r.erosion or "",
            "冗余度": r.verbosity or "",
            "反模式数": r.anti_pattern_count,
            "代码克隆率": r.clone_ratio or "",
            "总分": r.total_score,
            "等级": r.grade,
            "整体说明": r.overall_summary or "",
            "错误信息": r.scan_error or "",
        }
        # 扫描时间是 datetime 类型，需要传毫秒时间戳
        if r.scan_time:
            try:
                dt = datetime.strptime(r.scan_time, "%Y-%m-%d %H:%M:%S")
                fields["扫描时间"] = int(dt.timestamp() * 1000)
            except (ValueError, TypeError):
                pass
        records.append({"fields": fields})

    batch_size = 50
    for i in range(0, len(records), batch_size):
        batch = records[i : i + batch_size]
        payload = json.dumps({"records": batch}, ensure_ascii=False)
        try:
            result = subprocess.run(
                [
                    "lark-cli", "api", "POST",
                    f"/open-apis/bitable/v1/apps/{app_token}/tables/{table_id}/records/batch_create",
                    "--data", payload,
                ],
                capture_output=True, text=True, timeout=60,
            )
            output = result.stdout or result.stderr or ""
            resp = json.loads(output) if output.strip() else {}
            if resp.get("code") == 0:
                print(f"  ✓ 历史记录写入 {len(batch)} 条")
            else:
                print(f"  ✗ 写入失败: {resp.get('msg', result.stderr)}", file=sys.stderr)
        except Exception as e:
            print(f"  ✗ 写入异常: {e}", file=sys.stderr)


# ── 对比上次扫描 ──────────────────────────────────────
def compare_with_previous(reports: list[RepoReport], prev_csv: str) -> dict:
    """对比当前扫描和上次扫描，返回变化摘要"""
    prev = {}
    try:
        with open(prev_csv, encoding="utf-8-sig") as f:
            reader = csv.DictReader(f)
            for row in reader:
                name = row.get("repo_name", "")
                try:
                    score = float(row.get("total_score", 0))
                except (ValueError, TypeError):
                    score = 0.0
                prev[name] = {
                    "total_score": score,
                    "grade": row.get("grade", ""),
                    "claude_md_exists": row.get("claude_md_exists", "").lower() == "true",
                }
    except Exception as e:
        print(f"  ⚠ 读取上次 CSV 失败: {e}", file=sys.stderr)
        return {}

    improved, degraded, new_claude, lost_claude = [], [], [], []

    for r in reports:
        if r.repo_name not in prev:
            continue
        p = prev[r.repo_name]
        diff = r.total_score - p["total_score"]
        if diff >= 0.5:
            improved.append((r.repo_name, p["total_score"], r.total_score))
        elif diff <= -0.5:
            degraded.append((r.repo_name, p["total_score"], r.total_score))
        if r.claude_md_exists and not p["claude_md_exists"]:
            new_claude.append(r.repo_name)
        elif not r.claude_md_exists and p["claude_md_exists"]:
            lost_claude.append(r.repo_name)

    return {
        "improved": improved,
        "degraded": degraded,
        "new_claude": new_claude,
        "lost_claude": lost_claude,
    }


# ── 飞书 webhook 通知 ────────────────────────────────
def send_webhook(
    reports: list[RepoReport],
    webhook_url: str,
    scan_id: str,
    changes: dict | None = None,
    bitable_url: str = "",
):
    """发送扫描摘要到飞书群 webhook"""
    total = len(reports)
    grades = {"A": 0, "B": 0, "C": 0, "D": 0}
    claude_count = sum(1 for r in reports if r.claude_md_exists)
    arch_count = sum(1 for r in reports if r.architecture_doc_exists)
    avg_score = sum(r.total_score for r in reports) / total if total else 0

    for r in reports:
        grades[r.grade] = grades.get(r.grade, 0) + 1

    elements = [
        {
            "tag": "div",
            "text": {
                "tag": "lark_md",
                "content": (
                    f"**扫描批次**: {scan_id}\n"
                    f"**仓库数量**: {total}\n"
                    f"**平均分**: {avg_score:.1f} / 5.0\n"
                    f"**等级分布**: A={grades['A']}  B={grades['B']}  "
                    f"C={grades['C']}  D={grades['D']}\n"
                    f"**CLAUDE.md 覆盖率**: {claude_count}/{total} "
                    f"({claude_count * 100 // total if total else 0}%)\n"
                    f"**架构文档覆盖率**: {arch_count}/{total} "
                    f"({arch_count * 100 // total if total else 0}%)"
                ),
            },
        },
    ]

    if changes:
        change_lines = []
        for name, old, new in changes.get("improved", []):
            change_lines.append(f"📈 {name}: {old:.1f} → {new:.1f}")
        for name, old, new in changes.get("degraded", []):
            change_lines.append(f"📉 {name}: {old:.1f} → {new:.1f}")
        for name in changes.get("new_claude", []):
            change_lines.append(f"✅ {name} 新增 CLAUDE.md")
        for name in changes.get("lost_claude", []):
            change_lines.append(f"⚠️ {name} 丢失 CLAUDE.md")
        if change_lines:
            elements.append({"tag": "hr"})
            elements.append({
                "tag": "div",
                "text": {
                    "tag": "lark_md",
                    "content": "**与上次对比**:\n" + "\n".join(change_lines),
                },
            })

    if bitable_url:
        elements.append({"tag": "hr"})
        elements.append({
            "tag": "action",
            "actions": [{
                "tag": "button",
                "text": {"tag": "plain_text", "content": "查看详情"},
                "url": bitable_url,
                "type": "primary",
            }],
        })

    card = {
        "msg_type": "interactive",
        "card": {
            "header": {
                "title": {"tag": "plain_text", "content": f"🔍 AI 友好度巡检报告 - {scan_id}"},
                "template": "blue",
            },
            "elements": elements,
        },
    }

    try:
        req = urllib.request.Request(
            webhook_url,
            data=json.dumps(card, ensure_ascii=False).encode("utf-8"),
            headers={"Content-Type": "application/json"},
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            result = json.loads(resp.read().decode("utf-8"))
            if result.get("code") == 0 or result.get("StatusCode") == 0:
                print(f"  ✓ 飞书通知已发送")
            else:
                print(f"  ✗ 通知发送失败: {result}", file=sys.stderr)
    except Exception as e:
        print(f"  ✗ 通知发送异常: {e}", file=sys.stderr)


# ── 汇总统计 ──────────────────────────────────────────
def print_summary(reports: list[RepoReport]):
    """打印扫描汇总"""
    total = len(reports)
    grades = {"A": 0, "B": 0, "C": 0, "D": 0}
    claude_count = 0
    arch_count = 0
    errors = 0

    for r in reports:
        grades[r.grade] = grades.get(r.grade, 0) + 1
        if r.claude_md_exists:
            claude_count += 1
        if r.architecture_doc_exists:
            arch_count += 1
        if r.scan_error:
            errors += 1

    avg_score = sum(r.total_score for r in reports) / total if total else 0

    print(f"\n{'='*60}")
    print(f"扫描完成: {total} 个仓库")
    print(f"{'='*60}")
    print(f"代码+测试平均分: {avg_score:.1f} / 5.0")
    print(f"等级分布: A={grades['A']}  B={grades['B']}  C={grades['C']}  D={grades['D']}")
    print(f"CLAUDE.md 覆盖率: {claude_count}/{total} ({claude_count/total*100:.0f}%)")
    print(f"架构文档覆盖率: {arch_count}/{total} ({arch_count/total*100:.0f}%)")
    if errors:
        print(f"扫描失败: {errors} 个仓库")
    print(f"{'='*60}")
    if claude_count > 0 or arch_count > 0:
        print(f"\n下一步: 使用 --keep-clones 保留克隆目录，让大模型直接读取")
        print(f"CLAUDE.md / ARCHITECTURE.md 文件并评估内容质量")


# ── 主流程 ────────────────────────────────────────────
def main():
    parser = argparse.ArgumentParser(description="AI 友好度仓库扫描器")
    parser.add_argument("--input", required=True, help="仓库地址列表文件（每行一个）")
    parser.add_argument("--output", default="scan_results.csv", help="CSV 输出路径")
    parser.add_argument("--bitable-app", help="飞书多维表格 app_token")
    parser.add_argument("--bitable-table", help="飞书多维表格 table_id")
    parser.add_argument("--workdir", help="克隆工作目录（默认临时目录）")
    parser.add_argument("--concurrency", type=int, default=5, help="并发克隆数（默认 5）")
    parser.add_argument("--keep-clones", action="store_true", help="保留克隆的仓库")
    parser.add_argument("--skip-bitable", action="store_true", help="跳过写入飞书多维表格")
    parser.add_argument("--clear-snapshot", action="store_true", help="写入前清空快照表")
    parser.add_argument("--history-table", help="历史记录表 table_id（追加写入）")
    parser.add_argument("--webhook", help="飞书 webhook URL（发送巡检摘要）")
    parser.add_argument("--prev-csv", help="上次扫描的 CSV 路径（用于对比变化）")
    parser.add_argument("--scan-id", help="扫描批次标识（默认当前日期）")
    parser.add_argument("--bitable-url", default="", help="多维表格访问链接（通知中附带）")
    parser.add_argument("--skip-report", action="store_true", help="跳过生成 HTML 报告")
    args = parser.parse_args()

    # 读取仓库列表
    with open(args.input, encoding="utf-8") as f:
        urls = [line.strip() for line in f if line.strip() and not line.startswith("#")]

    if not urls:
        print("仓库列表为空", file=sys.stderr)
        sys.exit(1)

    print(f"共 {len(urls)} 个仓库待扫描")

    # 准备工作目录
    if args.workdir:
        workdir = Path(args.workdir)
        workdir.mkdir(parents=True, exist_ok=True)
    else:
        workdir = Path(tempfile.mkdtemp(prefix="repo-scan-"))

    if not args.keep_clones:
        atexit.register(lambda: shutil.rmtree(workdir, ignore_errors=True))

    print(f"工作目录: {workdir}")

    # 克隆
    clones = clone_all(urls, workdir, args.concurrency)

    # 扫描
    reports = []
    success_count = sum(1 for v in clones.values() if v is not None)
    print(f"\n开始扫描 {success_count} 个仓库...")

    for i, (url, path) in enumerate(clones.items(), 1):
        name = parse_repo_name(url)
        if path is None:
            report = RepoReport(
                repo_url=url.strip(),
                repo_name=name,
                scan_error="克隆失败",
                grade="D",
                scan_time=datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            )
        else:
            print(f"  [{i}/{len(clones)}] 扫描 {name}...")
            report = scan_repo(url, path)
        reports.append(report)

    # 按总分降序排列
    reports.sort(key=lambda r: r.total_score, reverse=True)

    # 输出
    output_path = args.output
    write_csv(reports, output_path)
    print_summary(reports)

    scan_id = args.scan_id or datetime.now().strftime("%Y-%m-%d")

    # 生成 HTML 报告
    if not args.skip_report:
        output_dir = os.path.dirname(output_path) or "."
        report_path = os.path.join(output_dir, f"report_{scan_id}.html")
        generate_html_report(reports, scan_id, output_dir, report_path)
        print(f"HTML 报告已生成: {report_path}")

    # 写入飞书多维表格（快照）
    if not args.skip_bitable and args.bitable_app and args.bitable_table:
        if args.clear_snapshot:
            clear_bitable_table(args.bitable_app, args.bitable_table)
        push_to_bitable(reports, args.bitable_app, args.bitable_table)
    elif not args.skip_bitable and (args.bitable_app or args.bitable_table):
        print("\n提示: 需同时提供 --bitable-app 和 --bitable-table 才能写入飞书多维表格")

    # 写入历史记录表
    if args.bitable_app and args.history_table:
        push_to_history(reports, args.bitable_app, args.history_table, scan_id)

    # 对比上次扫描 + 发送通知
    if args.webhook:
        changes = None
        if args.prev_csv and os.path.exists(args.prev_csv):
            changes = compare_with_previous(reports, args.prev_csv)
        send_webhook(reports, args.webhook, scan_id, changes, args.bitable_url)


if __name__ == "__main__":
    main()
