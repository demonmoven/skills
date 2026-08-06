# -*- coding: utf-8 -*-
"""
auto-develop team_profile.yaml 加载器。

加载顺序(优先级降序):
1. 环境变量 AUTO_DEVELOP_PROFILE_PATH 指向的文件
2. 当前 sandbox 中 settle_skill 同步路径下的 config/team_profile.yaml
   (即 ~/charge-cache/<skill_repo_short>/config/team_profile.yaml)

加载失败 / 字段缺失 → 抛 RuntimeError(fail-fast),由路由层负责把错误透传给用户。
不内置默认 profile,所有团队对等接入。
"""

import os
import re
from pathlib import Path
from typing import Any, Dict, List, Optional

try:
    import yaml  # PyYAML
except ImportError as e:
    raise RuntimeError(
        "auto-develop team profile loader 需要 PyYAML;"
        "在 sandbox 内 pip install --user pyyaml 后重试。"
    ) from e


REQUIRED_TOP_FIELDS = ["display_name", "skill_repo", "skill_branch", "meego", "bits"]
REQUIRED_MEEGO_FIELDS = ["business_lines", "project_keys", "url_pattern", "prd_field_names"]
REQUIRED_BITS_FIELDS = ["meego_param_mode"]
ALLOWED_MEEGO_PARAM_MODES = {"url", "id"}


def _candidate_paths() -> List[Path]:
    paths: List[Path] = []
    env_path = os.environ.get("AUTO_DEVELOP_PROFILE_PATH")
    if env_path:
        paths.append(Path(env_path).expanduser())

    # sandbox 默认探测
    home = Path(os.environ.get("HOME", "/home/mira"))
    paths.append(home / "charge-cache" / "settle_skill" / "config" / "team_profile.yaml")
    return paths


def _validate(profile: Dict[str, Any], src: Path) -> None:
    missing_top = [k for k in REQUIRED_TOP_FIELDS if k not in profile]
    if missing_top:
        raise RuntimeError(
            f"team_profile.yaml at {src} missing required top-level fields: {missing_top}.\n"
            f"请按 auto-develop 接入规范补全(参考 settle_skill 仓库 config/team_profile.yaml)。"
        )

    meego = profile.get("meego") or {}
    missing_m = [k for k in REQUIRED_MEEGO_FIELDS if k not in meego]
    if missing_m:
        raise RuntimeError(
            f"team_profile.yaml at {src} 缺少 meego.{missing_m} 字段。"
        )

    bits = profile.get("bits") or {}
    missing_b = [k for k in REQUIRED_BITS_FIELDS if k not in bits]
    if missing_b:
        raise RuntimeError(
            f"team_profile.yaml at {src} 缺少 bits.{missing_b} 字段。"
        )

    if bits["meego_param_mode"] not in ALLOWED_MEEGO_PARAM_MODES:
        raise RuntimeError(
            f"team_profile.yaml at {src} bits.meego_param_mode "
            f"必须是 {ALLOWED_MEEGO_PARAM_MODES} 之一,当前 = {bits['meego_param_mode']!r}"
        )

    # 类型 / 非空校验
    if not isinstance(meego.get("business_lines"), list) or not meego["business_lines"]:
        raise RuntimeError(f"team_profile.yaml at {src} meego.business_lines 必须是非空 list")
    if not isinstance(meego.get("project_keys"), list) or not meego["project_keys"]:
        raise RuntimeError(f"team_profile.yaml at {src} meego.project_keys 必须是非空 list")
    if not isinstance(meego.get("prd_field_names"), list) or not meego["prd_field_names"]:
        raise RuntimeError(f"team_profile.yaml at {src} meego.prd_field_names 必须是非空 list")

    try:
        re.compile(meego["url_pattern"])
    except re.error as e:
        raise RuntimeError(
            f"team_profile.yaml at {src} meego.url_pattern 不是合法正则: {e}"
        )

    # 可选字段 preferred_business_line:若提供必须是非空 str(2026-06-03 起)
    pbl = meego.get("preferred_business_line")
    if pbl is not None and (not isinstance(pbl, str) or not pbl.strip()):
        raise RuntimeError(
            f"team_profile.yaml at {src} meego.preferred_business_line "
            f"若提供则必须是非空字符串,当前 = {pbl!r}"
        )


def load_profile(explicit_path: Optional[str] = None) -> Dict[str, Any]:
    """
    加载并校验 team_profile,返回扁平 dict。
    explicit_path 优先级最高,其次 env 变量,最后 sandbox 默认。
    """
    paths: List[Path] = []
    if explicit_path:
        paths.append(Path(explicit_path).expanduser())
    paths.extend(_candidate_paths())

    tried: List[str] = []
    for p in paths:
        tried.append(str(p))
        if p.exists() and p.is_file():
            with p.open("r", encoding="utf-8") as f:
                profile = yaml.safe_load(f)
            if not isinstance(profile, dict):
                raise RuntimeError(
                    f"team_profile.yaml at {p} 顶层必须是 mapping,实际类型 = {type(profile).__name__}"
                )
            _validate(profile, p)
            profile["__source_path__"] = str(p)
            return profile

    raise RuntimeError(
        "auto-develop team_profile.yaml 未找到。\n"
        f"已尝试路径(优先级降序):\n  - " + "\n  - ".join(tried) + "\n\n"
        "接入规范:在你的 skill 仓库根目录创建 config/team_profile.yaml,字段见\n"
        "bytepay/settle_skill@feature/ai_native:config/team_profile.yaml,然后将\n"
        "AUTO_DEVELOP_PROFILE_PATH 环境变量指向该文件,或确保 settle_skill 同步路径下存在。"
    )


if __name__ == "__main__":
    import json
    import sys
    try:
        p = load_profile(sys.argv[1] if len(sys.argv) > 1 else None)
        print(json.dumps(p, ensure_ascii=False, indent=2))
    except RuntimeError as e:
        print(f"[FAIL] {e}", file=sys.stderr)
        sys.exit(1)

