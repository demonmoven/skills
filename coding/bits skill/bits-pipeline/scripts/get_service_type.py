#!/usr/bin/env python3
"""
get_service_type.py - Query service type by PSM

Auth mechanism aligned with bits_pipeline_cli:
  Priority: --jwt-token > $CLOUD_JWT > ~/.bits_pipeline_cli.env > --sa-secret > $SERVICE_SECRET/$SA_SECRET

Usage:
  python3 get_service_type.py --psm <psm> --env cn --sa-secret "$SA_SECRET"
  python3 get_service_type.py --psm <psm> --jwt-token "$CLOUD_JWT"

Note: The API /api/v2/appcenter/get_component_meta_by_subject is NOT in CLI's
42 RPC endpoints, so this script remains standalone but shares CLI's auth logic and cache.
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
# Constants (aligned with bits_pipeline_cli.py)
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


# ---------------------------------------------------------------------------
# JWT cache (shared with CLI: ~/.bits_pipeline_cli.env)
# ---------------------------------------------------------------------------

def _get_env_file(env_file_path=None):
    """Return the env file Path, defaulting to ~/.bits_pipeline_cli.env."""
    if env_file_path:
        return Path(env_file_path)
    return Path.home() / ".bits_pipeline_cli.env"


def _load_env_file(env_file):
    """Read key=value pairs from the env file (compatible with CLI format)."""
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
        # Strip surrounding quotes (single or double)
        if len(v) >= 2 and v[0] == v[-1] and v[0] in ("'", '"'):
            v = v[1:-1]
        values[k.strip()] = v
    return values


def _write_env_file(env_file, items):
    """Merge items into the env file (compatible with CLI format)."""
    existing = _load_env_file(env_file)
    existing.update(items)
    lines = [f"export {k}={shlex.quote(v)}" for k, v in sorted(existing.items())]
    env_file.parent.mkdir(parents=True, exist_ok=True)
    env_file.write_text("\n".join(lines) + "\n", encoding="utf-8")


def _parse_created_at(raw):
    """Parse a timestamp string to int, returning None on failure."""
    if not raw:
        return None
    try:
        return int(raw)
    except (TypeError, ValueError):
        return None


def _ensure_jwt_not_expired(created_at):
    """Raise if the JWT is older than 1 hour."""
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


# ---------------------------------------------------------------------------
# JWT fetch (aligned with CLI's fetch_service_jwt)
# ---------------------------------------------------------------------------

def _fetch_service_jwt(sa_secret, jwt_host, timeout=30, insecure=False):
    """Exchange sa_secret for a JWT via the cloud auth API."""
    url = f"{jwt_host}/auth/api/v1/jwt"
    req = request.Request(url=url, method="GET")
    req.add_header("Authorization", f"Bearer {sa_secret}")

    context = None
    if insecure:
        context = ssl._create_unverified_context()

    try:
        with request.urlopen(req, timeout=timeout, context=context) as resp:
            jwt_token = resp.headers.get("x-jwt-token", "")
            if not jwt_token:
                logid = resp.headers.get("x-tt-logid", "")
                print(
                    f"Error: JWT exchange failed, status={resp.getcode()}, logID={logid}",
                    file=sys.stderr,
                )
                sys.exit(1)
            return jwt_token
    except error.HTTPError as e:
        logid = e.headers.get("x-tt-logid", "") if e.headers else ""
        print(f"Error: JWT exchange failed, status={e.code}, logID={logid}", file=sys.stderr)
        sys.exit(1)
    except error.URLError as e:
        print(f"Error: JWT exchange failed: {e.reason}", file=sys.stderr)
        sys.exit(1)


# ---------------------------------------------------------------------------
# JWT resolve (aligned with CLI's resolve_jwt_token priority chain)
# ---------------------------------------------------------------------------

def resolve_jwt_token(args):
    """
    Resolve JWT token using the same priority chain as bits_pipeline_cli:
      1. --jwt-token argument
      2. $CLOUD_JWT environment variable
      3. ~/.bits_pipeline_cli.env file cache
      4. --sa-secret / $SERVICE_SECRET / $SA_SECRET -> auto exchange
    """
    env_file = _get_env_file(getattr(args, "env_file", None))

    # 1) Explicit --jwt-token
    jwt_token = getattr(args, "jwt_token", None)
    if jwt_token:
        created_at = _parse_created_at(os.getenv("CLOUD_JWT_CREATED_AT"))
        _ensure_jwt_not_expired(created_at)
        return jwt_token

    # 2) $CLOUD_JWT env var
    jwt_token = os.getenv("CLOUD_JWT")
    if jwt_token:
        created_at = _parse_created_at(os.getenv("CLOUD_JWT_CREATED_AT"))
        _ensure_jwt_not_expired(created_at)
        return jwt_token

    # 3) Cached in env file
    file_values = _load_env_file(env_file)
    jwt_token = file_values.get("CLOUD_JWT", "")
    if jwt_token:
        created_at = _parse_created_at(file_values.get("CLOUD_JWT_CREATED_AT"))
        _ensure_jwt_not_expired(created_at)
        return jwt_token

    # 4) Auto exchange via sa_secret
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

    # Persist to env and file (shared with CLI)
    os.environ["CLOUD_JWT"] = jwt_token
    os.environ["CLOUD_JWT_CREATED_AT"] = created_at_str
    _write_env_file(env_file, {
        "CLOUD_JWT": jwt_token,
        "CLOUD_JWT_CREATED_AT": created_at_str,
    })

    return jwt_token


# ---------------------------------------------------------------------------
# Core: query service type
# ---------------------------------------------------------------------------

def get_service_type(jwt_token, psm, env="cn", timeout=30, insecure=False):
    """
    Query service type by PSM via /api/v2/appcenter/get_component_meta_by_subject.

    Args:
        jwt_token: JWT for authentication
        psm: Service PSM identifier
        env: Environment key (cn/i18n-tt/i18n-bd/eu-ttp/us-ttp)
        timeout: HTTP timeout in seconds
        insecure: Skip TLS certificate verification

    Returns:
        str: Mapped service type (tce/faas/tcc/...) or empty string
    """
    host = ENV_BASE_URLS.get(env, ENV_BASE_URLS["cn"])
    encoded_psm = parse.quote(psm, safe="")
    url = f"{host}/api/v2/appcenter/get_component_meta_by_subject?subject={encoded_psm}"

    req = request.Request(url=url, method="GET")
    req.add_header("x-jwt-token", jwt_token)

    context = None
    if insecure:
        context = ssl._create_unverified_context()

    try:
        with request.urlopen(req, timeout=timeout, context=context) as resp:
            status = resp.getcode()
            text = resp.read().decode("utf-8")
    except error.HTTPError as e:
        body = ""
        try:
            body = e.read().decode("utf-8")
        except Exception:
            pass
        error_info = {
            "error": "Failed to get service type",
            "http_status": e.code,
            "response": body,
        }
        print(json.dumps(error_info, indent=2, ensure_ascii=False), file=sys.stderr)
        sys.exit(1)
    except error.URLError as e:
        print(f"Error: Request failed: {e.reason}", file=sys.stderr)
        sys.exit(1)

    if status < 200 or status >= 300:
        error_info = {
            "error": "Failed to get service type",
            "http_status": status,
            "response": text,
        }
        print(json.dumps(error_info, indent=2, ensure_ascii=False), file=sys.stderr)
        sys.exit(1)

    try:
        response = json.loads(text)
        data = response.get("data") or []
        for item in data:
            if item.get("psm") == psm:
                component_type = item.get("componentType")
                return COMPONENT_TYPE_MAP.get(component_type, component_type)
        return ""
    except json.JSONDecodeError:
        return ""


# ---------------------------------------------------------------------------
# CLI entry point
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Query service type by PSM (auth aligned with bits_pipeline_cli)",
        epilog=(
            "Auth priority: --jwt-token > $CLOUD_JWT > ~/.bits_pipeline_cli.env "
            "> --sa-secret > $SERVICE_SECRET/$SA_SECRET\n\n"
            "Examples:\n"
            '  python3 get_service_type.py --psm my.psm.name --env cn --sa-secret "$SA_SECRET"\n'
            '  python3 get_service_type.py --psm my.psm.name --jwt-token "$CLOUD_JWT"\n'
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--psm", required=True, help="Service PSM to query")
    parser.add_argument("--env", default="cn",
                        choices=list(ENV_BASE_URLS.keys()),
                        help="Environment (default: cn)")
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
    result = get_service_type(
        jwt_token=jwt_token,
        psm=args.psm,
        env=args.env,
        timeout=args.timeout,
        insecure=args.insecure,
    )
    print(result)


if __name__ == "__main__":
    main()
