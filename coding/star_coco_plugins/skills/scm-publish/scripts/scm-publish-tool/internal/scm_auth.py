"""Authentication helpers for SCM OpenAPI client.

Manages a personal JWT token for x-jwt-token header.
Always include header: domain: "scm;v1".
"""
import base64
import json
import os
from typing import Optional

from internal.exceptions import ApiError

_global_token: Optional[str] = os.getenv('SCM_JWT_TOKEN')


def set_token(token: str) -> None:
    """Set global JWT token used by SCMClient.

    Args:
        token: The JWT token string.
    """
    global _global_token
    _global_token = token.strip()


def get_token() -> str:
    """Get current global JWT token.

    Raises:
        ApiError: If no token is configured.
    """
    if not _global_token:
        raise ApiError(
            "Missing JWT Token. "
            "Please set the SCM_JWT_TOKEN environment variable or pass --jwt flag. "
            "Use 'byte-cli login get-jwt' to obtain a personal JWT token."
        )
    return _global_token


def decode_jwt_username() -> Optional[str]:
    """Decode username from the current JWT token payload.

    Parses the JWT payload (without signature verification) to extract
    the user identity. Tries common claim fields in order.

    Returns:
        Username string if found, None otherwise.
    """
    token = get_token()
    try:
        parts = token.split('.')
        if len(parts) != 3:
            return None
        payload_b64 = parts[1]
        # Fix base64url padding
        padding = 4 - len(payload_b64) % 4
        if padding != 4:
            payload_b64 += '=' * padding
        payload = json.loads(base64.urlsafe_b64decode(payload_b64))
        # Try common username fields in priority order
        for field in ("user_name", "preferred_username", "name", "sub"):
            val = payload.get(field)
            if isinstance(val, str) and val:
                return val
        # Try email → username
        email = payload.get("email", "")
        if isinstance(email, str) and "@" in email:
            return email.split("@")[0]
        return None
    except Exception:
        return None
