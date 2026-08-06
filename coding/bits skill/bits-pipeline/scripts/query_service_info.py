#!/usr/bin/env python3
"""
query_service_info.py - Two-step service info query for pipeline-orchestration skill.

Step 1: Get service type via /api/v2/appcenter/get_component_meta_by_subject
Step 2: Get SCM/Git details via /api/v2/appcenter/batch_component_info

Auth: reuses same JWT chain as get_service_type.py / bits_pipeline_cli.

Usage:
  python3 query_service_info.py --psm bits.nexus_flow.api
  python3 query_service_info.py --psm bits.nexus_flow.api --env cn --control-planes cn
  python3 query_service_info.py --psm bits.nexus_flow.api --jwt-token "$CLOUD_JWT"

Output: JSON with service_type, project_type, scm_name, git_repo_name, git_repo_url, etc.
"""

import argparse
import json
import os
import shlex
import ssl
import sys
import time
from pathlib import Path
from urllib import error, parse, request

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

MAX_JWT_AGE_SECONDS = 3600

ENV_BASE_URLS = {
    "cn":       "https://bits.bytedance.net",
    "i18n-tt":  "https://bits.bytedance.net",
    "i18n-bd":  "https://bits.bytedance.net",
    "eu-ttp":   "https://v656wnw3.eu-ttp-fn.tiktoke.org",
    "us-ttp":   "https://0t9jybpb.us-ttp-fn.tiktokd.net",
}

JWT_HOSTS = {
    "cn":       "https://cloud.bytedance.net",
    "i18n-tt":  "https://cloud-i18n.bytedance.net",
    "eu-ttp":   "https://cloud-i18n.tiktoke.org",
    "us-ttp":   "https://cloud-tx.tiktokd.net",
    "i18n-bd":  "https://cloud-i18nbd.byted.org",
}

ENV_TO_JWT_REGION = {
    "cn":       "cn",
    "i18n-tt":  "i18n-tt",
    "i18n-bd":  "i18n-bd",
    "eu-ttp":   "eu-ttp",
    "us-ttp":   "us-ttp",
}

# componentType (API 1) -> human-readable service type
COMPONENT_TYPE_MAP = {
    "service":               "tce",
    "tcc":                   "tcc",
    "function":              "faas",
    "web":                   "web",
    "web_ttp":               "web_ttp",
    "client_component":      "client",
    "hybrid_component":      "hybrid",
    "customized_component":  "customized",
    "cronjob":               "cronjob",
    "application":           "application",
}

# componentType (API 1) -> ProjectType enum string (API 2 request body)
COMPONENT_TO_PROJECT_TYPE_STR = {
    "service":               "PROJECT_TYPE_TCE",
    "function":              "PROJECT_TYPE_FAAS",
    "tcc":                   "PROJECT_TYPE_TCC",
    "web":                   "PROJECT_TYPE_WEB",
    "cronjob":               "PROJECT_TYPE_CRONJOB",
    "hybrid_component":      "PROJECT_TYPE_HYBRID",
    "client_component":      "PROJECT_TYPE_LIBRARY",
    "customized_component":  "PROJECT_TYPE_CUSTOMIZED_COMPONENT",
}

# componentType -> ProjectType enum int
COMPONENT_TO_PROJECT_TYPE_INT = {
    "service":               1,
    "function":              2,
    "cronjob":               3,
    "web":                   4,
    "hybrid_component":      5,
    "tcc":                   12,
    "customized_component":  100,
}

# ControlPlane name -> int
CONTROL_PLANE_MAP = {
    "cn":       1,
    "i18n":     2,
    "eu-ttp":   4,
    "us-ttp":   5,
    "i18n-bd":  6,
}

# ControlPlane name -> enum string
CONTROL_PLANE_STR_MAP = {
    "cn":       "CONTROL_PLANE_CN",
    "i18n":     "CONTROL_PLANE_I18N",
    "eu-ttp":   "CONTROL_PLANE_EU_TTP",
    "us-ttp":   "CONTROL_PLANE_US_TTP",
    "i18n-bd":  "CONTROL_PLANE_I18N_BD",
}


# ---------------------------------------------------------------------------
# JWT helpers (shared logic with get_service_type.py)
# ---------------------------------------------------------------------------

def _get_env_file(env_file_path=None):
    if env_file_path:
        return Path(env_file_path)
    return Path.home() / ".bits_pipeline_cli.env"


def _load_env_file(env_file):
    values = {}
    if not env_file.exists():
        return values
    for raw in env_file.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line[len("export "):]
        if "=" not in line:
            continue
        k, v = line.split("=", 1)
        v = v.strip()
        if len(v) >= 2 and v[0] == v[-1] and v[0] in ("'", '"'):
            v = v[1:-1]
        values[k.strip()] = v
    return values


def _write_env_file(env_file, items):
    existing = _load_env_file(env_file)
    existing.update(items)
    lines = [f"export {k}={shlex.quote(v)}" for k, v in sorted(existing.items())]
    env_file.parent.mkdir(parents=True, exist_ok=True)
    env_file.write_text("\n".join(lines) + "\n", encoding="utf-8")


def _parse_created_at(raw):
    if not raw:
        return None
    try:
        return int(raw)
    except (TypeError, ValueError):
        return None


def _ensure_jwt_not_expired(created_at):
    if created_at is None:
        return
    age = int(time.time()) - created_at
    if age > MAX_JWT_AGE_SECONDS:
        print(
            f"Error: JWT is older than 1 hour ({age}s). "
            "Please re-run with --sa-secret to refresh, or provide a new --jwt-token.",
            file=sys.stderr,
        )
        sys.exit(1)


def _fetch_service_jwt(sa_secret, jwt_host, timeout=30, insecure=False):
    url = f"{jwt_host}/auth/api/v1/jwt"
    req = request.Request(url=url, method="GET")
    req.add_header("Authorization", f"Bearer {sa_secret}")
    context = ssl._create_unverified_context() if insecure else None
    try:
        with request.urlopen(req, timeout=timeout, context=context) as resp:
            jwt_token = resp.headers.get("x-jwt-token", "")
            if not jwt_token:
                logid = resp.headers.get("x-tt-logid", "")
                print(f"Error: JWT exchange failed, status={resp.getcode()}, logID={logid}",
                      file=sys.stderr)
                sys.exit(1)
            return jwt_token
    except error.HTTPError as e:
        logid = e.headers.get("x-tt-logid", "") if e.headers else ""
        print(f"Error: JWT exchange failed, status={e.code}, logID={logid}", file=sys.stderr)
        sys.exit(1)
    except error.URLError as e:
        print(f"Error: JWT exchange failed: {e.reason}", file=sys.stderr)
        sys.exit(1)


def resolve_jwt_token(args):
    """Resolve JWT: --jwt-token > $CLOUD_JWT > env file cache > --sa-secret auto exchange."""
    env_file = _get_env_file(getattr(args, "env_file", None))

    jwt_token = getattr(args, "jwt_token", None)
    if jwt_token:
        created_at = _parse_created_at(os.getenv("CLOUD_JWT_CREATED_AT"))
        _ensure_jwt_not_expired(created_at)
        return jwt_token

    jwt_token = os.getenv("CLOUD_JWT")
    if jwt_token:
        created_at = _parse_created_at(os.getenv("CLOUD_JWT_CREATED_AT"))
        _ensure_jwt_not_expired(created_at)
        return jwt_token

    file_values = _load_env_file(env_file)
    jwt_token = file_values.get("CLOUD_JWT", "")
    if jwt_token:
        created_at = _parse_created_at(file_values.get("CLOUD_JWT_CREATED_AT"))
        _ensure_jwt_not_expired(created_at)
        return jwt_token

    sa_secret = (
        getattr(args, "sa_secret", None)
        or os.getenv("SERVICE_SECRET")
        or os.getenv("SA_SECRET")
    )
    if not sa_secret:
        print(
            "Error: No JWT available. Provide --jwt-token, set $CLOUD_JWT, "
            "or provide --sa-secret / $SERVICE_SECRET to auto-exchange.",
            file=sys.stderr,
        )
        sys.exit(1)

    env = getattr(args, "env", "cn").lower()
    region = ENV_TO_JWT_REGION.get(env, "cn")
    jwt_host = JWT_HOSTS[region]
    insecure = getattr(args, "insecure", False)
    jwt_token = _fetch_service_jwt(sa_secret, jwt_host, insecure=insecure)
    created_at_str = str(int(time.time()))
    os.environ["CLOUD_JWT"] = jwt_token
    os.environ["CLOUD_JWT_CREATED_AT"] = created_at_str
    _write_env_file(env_file, {
        "CLOUD_JWT": jwt_token,
        "CLOUD_JWT_CREATED_AT": created_at_str,
    })
    return jwt_token


# ---------------------------------------------------------------------------
# HTTP helper
# ---------------------------------------------------------------------------

def _http_request(url, jwt_token, method="GET", body=None, timeout=30, insecure=False):
    """Generic HTTP request returning parsed JSON."""
    req = request.Request(url=url, method=method)
    req.add_header("X-Jwt-Token", jwt_token)
    if body is not None:
        req.add_header("Content-Type", "application/json")
        req.data = json.dumps(body).encode("utf-8")

    context = ssl._create_unverified_context() if insecure else None
    try:
        with request.urlopen(req, timeout=timeout, context=context) as resp:
            text = resp.read().decode("utf-8")
            return json.loads(text)
    except error.HTTPError as e:
        err_body = ""
        try:
            err_body = e.read().decode("utf-8")
        except Exception:
            pass
        print(json.dumps({
            "error": f"HTTP {method} failed",
            "url": url,
            "http_status": e.code,
            "response": err_body,
        }, indent=2, ensure_ascii=False), file=sys.stderr)
        sys.exit(1)
    except error.URLError as e:
        print(f"Error: Request to {url} failed: {e.reason}", file=sys.stderr)
        sys.exit(1)


# ---------------------------------------------------------------------------
# Step 1: Query component meta (service type)
# ---------------------------------------------------------------------------

def query_component_meta(jwt_token, psm, env="cn", timeout=30, insecure=False):
    """
    GET /api/v2/appcenter/get_component_meta_by_subject?subject=<psm>
    Returns list of ComponentMeta dicts.
    """
    host = ENV_BASE_URLS.get(env, ENV_BASE_URLS["cn"])
    encoded_psm = parse.quote(psm, safe="")
    url = f"{host}/api/v2/appcenter/get_component_meta_by_subject?subject={encoded_psm}"
    resp = _http_request(url, jwt_token, method="GET", timeout=timeout, insecure=insecure)

    data = resp.get("data") or []
    if not data:
        print(f"Warning: No component found for PSM '{psm}'", file=sys.stderr)
    return data


def pick_primary_component(components, psm):
    """
    Pick the primary (deployable) component.
    Priority: service(TCE) > function(FaaS) > web > cronjob > hybrid > tcc > others.
    """
    priority = ["service", "function", "web", "cronjob", "hybrid_component", "tcc"]
    by_type = {c.get("componentType"): c for c in components}

    for ct in priority:
        if ct in by_type:
            return by_type[ct]

    return components[0] if components else None


# ---------------------------------------------------------------------------
# Step 2: Query batch component info (SCM/Git details)
# ---------------------------------------------------------------------------

def query_component_detail(jwt_token, psm, component_type, env="cn",
                           control_planes=None, timeout=30, insecure=False):
    """
    POST /api/v2/appcenter/batch_component_info
    Returns ProjectInfo with controlPlaneDetail containing SCM/Git info.
    """
    if control_planes is None:
        control_planes = ["cn"]

    project_type_str = COMPONENT_TO_PROJECT_TYPE_STR.get(component_type)
    if not project_type_str:
        print(f"Warning: Unknown componentType '{component_type}', cannot query detail",
              file=sys.stderr)
        return None

    cp_ints = [CONTROL_PLANE_MAP.get(cp, 1) for cp in control_planes]
    cp_strs = [CONTROL_PLANE_STR_MAP.get(cp, "CONTROL_PLANE_CN") for cp in control_planes]

    host = ENV_BASE_URLS.get(env, ENV_BASE_URLS["cn"])
    url = f"{host}/api/v2/appcenter/batch_component_info"

    body = {
        "queryProjects": [{
            "projectUniqueId": psm,
            "projectType": project_type_str,
            "controlPlanes": cp_ints,
        }],
        "controlPlanes": cp_strs,
        "cacheExpiration": 3034404415,
    }

    resp = _http_request(url, jwt_token, method="POST", body=body,
                         timeout=timeout, insecure=insecure)
    projects = resp.get("projects") or []
    return projects[0] if projects else None


# ---------------------------------------------------------------------------
# Combine results
# ---------------------------------------------------------------------------

def query_service_info(jwt_token, psm, env="cn", control_planes=None,
                       timeout=30, insecure=False):
    """Two-step query: meta -> detail. Returns a unified JSON dict."""
    # Step 1: get component meta
    components = query_component_meta(jwt_token, psm, env, timeout, insecure)
    primary = pick_primary_component(components, psm)

    if not primary:
        return {
            "psm": psm,
            "found": False,
            "service_type": "",
            "error": f"No component found for PSM '{psm}'",
        }

    component_type = primary.get("componentType", "")
    service_type = COMPONENT_TYPE_MAP.get(component_type, component_type)
    project_type_int = COMPONENT_TO_PROJECT_TYPE_INT.get(component_type, 0)

    result = {
        "psm": psm,
        "found": True,
        "service_type": service_type,                # tce / faas / tcc / ...
        "component_type": component_type,            # raw API value
        "project_type": project_type_int,            # enum int
        "project_type_str": COMPONENT_TO_PROJECT_TYPE_STR.get(component_type, ""),
        "name": primary.get("name", ""),
        "component_id": primary.get("componentId", ""),
        "creator": primary.get("creator", ""),
        "git_repo_name": primary.get("gitRepoName", ""),
        "git_framework": primary.get("gitFramework", ""),
        "is_monorepo": primary.get("isMonorepo", False),
        "mono_sub_path": primary.get("monoSubPath", ""),
    }

    # Step 2: get SCM/Git detail
    detail = query_component_detail(
        jwt_token, psm, component_type, env,
        control_planes=control_planes, timeout=timeout, insecure=insecure,
    )

    if detail:
        scm_list = []
        main_scm_name = ""
        main_git_repo_name = ""
        main_git_repo_url = ""
        first_scm_name = ""
        first_git_repo_name = ""
        first_git_repo_url = ""

        for cpd in detail.get("controlPlaneDetail") or []:
            for scm in cpd.get("scmInfo") or []:
                scm_entry = {
                    "scm_id": scm.get("id", ""),
                    "scm_name": scm.get("name", ""),
                    "is_main": scm.get("isMain", False),
                    "git_repo_name": scm.get("gitRepoName", ""),
                    "git_repo_url": scm.get("gitRepoUrl", ""),
                    "git_repo_id": scm.get("gitRepoId", ""),
                }
                scm_list.append(scm_entry)
                # Track first SCM as fallback
                if not first_scm_name:
                    first_scm_name = scm.get("name", "")
                    first_git_repo_name = scm.get("gitRepoName", "")
                    first_git_repo_url = scm.get("gitRepoUrl", "")
                # Prefer main SCM
                if scm.get("isMain", False):
                    main_scm_name = scm.get("name", "")
                    main_git_repo_name = scm.get("gitRepoName", "")
                    main_git_repo_url = scm.get("gitRepoUrl", "")

        result["scm_list"] = scm_list
        result["scm_name"] = main_scm_name or first_scm_name
        result["git_repo_name"] = main_git_repo_name or first_git_repo_name or result["git_repo_name"]
        result["git_repo_url"] = main_git_repo_url or first_git_repo_url
        result["deploy_link"] = ""
        for cpd in detail.get("controlPlaneDetail") or []:
            if cpd.get("link"):
                result["deploy_link"] = cpd["link"]
                break
    else:
        result["scm_list"] = []
        result["scm_name"] = ""
        result["git_repo_url"] = ""
        result["deploy_link"] = ""

    return result


# ---------------------------------------------------------------------------
# CLI entry
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Query service type and SCM/Git info by PSM (two-step)",
        epilog=(
            "Examples:\n"
            "  python3 query_service_info.py --psm bits.nexus_flow.api\n"
            "  python3 query_service_info.py --psm bits.nexus_flow.api --control-planes cn i18n\n"
            '  python3 query_service_info.py --psm bits.appcenter.api --jwt-token "$JWT"\n'
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--psm", required=True,
                        help="Service PSM (三段式, e.g. bits.nexus_flow.api)")
    parser.add_argument("--env", default="cn",
                        choices=list(ENV_BASE_URLS.keys()),
                        help="Environment (default: cn)")
    parser.add_argument("--control-planes", dest="control_planes", nargs="+",
                        default=["cn"],
                        choices=list(CONTROL_PLANE_MAP.keys()),
                        help="Control planes to query (default: cn)")
    parser.add_argument("--jwt-token", dest="jwt_token",
                        help="Provide JWT token directly")
    parser.add_argument("--sa-secret", dest="sa_secret",
                        help="Service account secret for auto JWT exchange")
    parser.add_argument("--env-file", dest="env_file",
                        help="JWT cache file path (default: ~/.bits_pipeline_cli.env)")
    parser.add_argument("--insecure", action="store_true",
                        help="Skip TLS certificate verification")
    parser.add_argument("--timeout", type=int, default=30,
                        help="HTTP timeout seconds (default: 30)")

    args = parser.parse_args()
    jwt_token = resolve_jwt_token(args)
    result = query_service_info(
        jwt_token=jwt_token,
        psm=args.psm,
        env=args.env,
        control_planes=args.control_planes,
        timeout=args.timeout,
        insecure=args.insecure,
    )
    print(json.dumps(result, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
