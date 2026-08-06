from __future__ import annotations

from datetime import datetime, timezone
from pathlib import Path
import re

SUPPORTED_TYPES = [
    "long_method",
    "large_class",
    "long_parameter_list",
    "divergent_change",
    "shotgun_surgery",
    "feature_envy",
    "data_clumps",
    "primitive_obsession",
    "switch_statements",
    "parallel_inheritance_hierarchies",
    "lazy_class",
    "speculative_generality",
    "temporary_field",
    "message_chains",
    "middle_man",
    "inappropriate_intimacy",
    "alternative_classes_with_different_interfaces",
    "incomplete_library_class",
    "data_class",
    "refused_bequest",
    "comments",
    "duplicated_code",
    "staticcheck",
    "gosimple",
    "stylecheck",
    "gofmt",
    "goimports",
    "gocyclo",
    "errcheck",
    "ineffassign",
    "unused",
    "revive",
    "cyclop",
    "govet",
    "typecheck",
]

CODE_SMELL_TYPES = {
    "long_method",
    "large_class",
    "long_parameter_list",
    "divergent_change",
    "shotgun_surgery",
    "feature_envy",
    "data_clumps",
    "primitive_obsession",
    "switch_statements",
    "parallel_inheritance_hierarchies",
    "lazy_class",
    "speculative_generality",
    "temporary_field",
    "message_chains",
    "middle_man",
    "inappropriate_intimacy",
    "alternative_classes_with_different_interfaces",
    "incomplete_library_class",
    "data_class",
    "refused_bequest",
    "comments",
}

FORMATTER_TYPES = {"gofmt", "goimports"}
DEFAULT_LINTERS = [
    "staticcheck",
    "gosimple",
    "stylecheck",
    "gocyclo",
    "errcheck",
    "ineffassign",
    "unused",
    "revive",
    "cyclop",
    "govet",
]
BOOTSTRAP_ALLOWED_LINTERS = set(DEFAULT_LINTERS) | {"typecheck"}

LOW_RISK_TYPES = {
    "long_method",
    "large_class",
    "long_parameter_list",
    "data_clumps",
    "primitive_obsession",
    "lazy_class",
    "temporary_field",
    "middle_man",
    "data_class",
    "comments",
    "gosimple",
    "stylecheck",
}

MEDIUM_RISK_TYPES = {"ineffassign", "unused", "revive", "duplicated_code"}
HIGH_RISK_TYPES = {
    "divergent_change",
    "shotgun_surgery",
    "feature_envy",
    "switch_statements",
    "parallel_inheritance_hierarchies",
    "speculative_generality",
    "message_chains",
    "inappropriate_intimacy",
    "alternative_classes_with_different_interfaces",
    "incomplete_library_class",
    "refused_bequest",
    "staticcheck",
    "gocyclo",
    "errcheck",
    "cyclop",
    "govet",
    "typecheck",
}

TYPE_HANDLERS = {
    "long_method": "llm",
    "large_class": "llm",
    "long_parameter_list": "llm",
    "divergent_change": "llm",
    "shotgun_surgery": "llm",
    "feature_envy": "llm",
    "data_clumps": "llm",
    "primitive_obsession": "llm",
    "switch_statements": "llm",
    "parallel_inheritance_hierarchies": "llm",
    "lazy_class": "llm",
    "speculative_generality": "llm",
    "temporary_field": "llm",
    "message_chains": "llm",
    "middle_man": "llm",
    "inappropriate_intimacy": "llm",
    "alternative_classes_with_different_interfaces": "llm",
    "incomplete_library_class": "llm",
    "data_class": "llm",
    "refused_bequest": "llm",
    "comments": "llm",
    "duplicated_code": "tool",
    "staticcheck": "tool",
    "gosimple": "tool",
    "stylecheck": "tool",
    "gofmt": "tool",
    "goimports": "tool",
    "gocyclo": "tool",
    "errcheck": "tool",
    "ineffassign": "tool",
    "unused": "tool",
    "revive": "tool",
    "cyclop": "tool",
    "govet": "tool",
    "typecheck": "tool",
}

DESCRIPTIONS = {
    "duplicated_code": "Duplicate code: duplicate code fragments detected",
    "long_method": "Long method: function does too many things",
    "large_class": "Large class: class has too many responsibilities",
    "long_parameter_list": "Long parameter list: function has too many parameters",
    "divergent_change": "Divergent change: class is changed for many different reasons",
    "shotgun_surgery": "Shotgun surgery: one change requires modifying many places",
    "feature_envy": "Feature envy: method seems more interested in another class",
    "data_clumps": "Data clumps: data items that always go together",
    "primitive_obsession": "Primitive obsession: using primitives instead of objects",
    "switch_statements": "Switch statements: overuse of switch or if-else chains",
    "parallel_inheritance_hierarchies": "Parallel inheritance hierarchies: adding a subclass forces adding another",
    "lazy_class": "Lazy class: class does too little",
    "speculative_generality": "Speculative generality: building for a future that never comes",
    "temporary_field": "Temporary field: field only used in certain circumstances",
    "message_chains": "Message chains: long chains of method calls",
    "middle_man": "Middle man: class delegates everything",
    "inappropriate_intimacy": "Inappropriate intimacy: classes know too much about each other",
    "alternative_classes_with_different_interfaces": "Alternative classes: similar functionality but different interfaces",
    "incomplete_library_class": "Incomplete library class: library class needs patching",
    "data_class": "Data class: class has only data, no behavior",
    "refused_bequest": "Refused bequest: subclass doesn't use parent's methods",
    "comments": "Comments: too many comments indicate code is hard to understand",
    "staticcheck": "Static check: finds bugs and performance issues",
    "gosimple": "Simplifies code: suggests more idiomatic Go constructs",
    "stylecheck": "Style check: enforces Go style guidelines",
    "gofmt": "Code formatting: file needs gofmt",
    "goimports": "Import formatting: file needs goimports",
    "gocyclo": "Cyclomatic complexity: function has high complexity",
    "errcheck": "Error check: unchecked errors detected",
    "ineffassign": "Ineffective assignment: unused assignment",
    "unused": "Unused code: unused constants, variables, functions or types",
    "revive": "Revive: fast, configurable, extensible Go linter",
    "cyclop": "Cyclop: checks function cyclomatic complexity",
    "govet": "Go vet: suspicious constructs detected",
    "typecheck": "Type check: type checking issues",
}

GOLANGCI_FAILURE_PATTERNS = [
    r"context loading failed",
    r"failed to load packages",
    r"go\.mod requires go >=",
    r"missing go\.sum entry",
    r"could not load export data",
    r"build constraints exclude all Go files",
]


def log(prefix: str, message: str) -> None:
    print(f"[LOG] [{prefix}] {message}")


def utc_timestamp() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def get_risk(issue_type: str) -> str:
    if issue_type in LOW_RISK_TYPES:
        return "low"
    if issue_type in MEDIUM_RISK_TYPES:
        return "medium"
    return "high"


def split_csv(raw: str) -> list[str]:
    if not raw:
        return []
    return [part.strip() for part in raw.split(",") if part.strip()]


def parse_go_version(version: str) -> tuple[int, int, int]:
    cleaned = version.removeprefix("go")
    cleaned = re.split(r"[^0-9.]", cleaned, maxsplit=1)[0]
    parts = [int(part) for part in cleaned.split(".") if part]
    while len(parts) < 3:
        parts.append(0)
    return tuple(parts[:3])


def version_ge(lhs: str, rhs: str) -> bool:
    return parse_go_version(lhs) >= parse_go_version(rhs)


def find_go_directories(scan_dir: Path, max_depth: int = 4) -> list[Path]:
    results: list[tuple[int, Path]] = []
    root = scan_dir.resolve()
    for path in root.rglob("*"):
        if not path.is_dir():
            continue
        if any(part in {".git", "vendor", "testdata", "mocks", "opp"} for part in path.parts):
            continue
        rel = path.relative_to(root)
        depth = len(rel.parts)
        if depth < 1 or depth > max_depth:
            continue
        if path.name.startswith("."):
            continue
        if list(path.glob("*.go")):
            results.append((depth, path))
    results.sort(key=lambda item: item[0], reverse=True)
    return [path for _, path in results]


def repo_relative_path(path: Path, cwd: Path) -> str:
    try:
        rel = path.resolve().relative_to(cwd.resolve())
        return f"./{rel.as_posix()}"
    except ValueError:
        return path.as_posix()
