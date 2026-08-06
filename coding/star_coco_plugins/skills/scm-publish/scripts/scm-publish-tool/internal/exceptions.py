"""Exceptions for SCM OpenAPI client.

Contains ApiError and HttpError, plus helpers.
"""
from __future__ import annotations

from typing import Optional


class ApiError(Exception):
    """API logical error or general request failure.

    Used for envelope code != 0, missing auth, or other client-side conditions.
    """

    def __init__(
        self,
        message: str,
        *,
        status: Optional[int] = None,
        body_snippet: Optional[str] = None,
        envelope_code: Optional[int] = None,
    ) -> None:
        self.message = message
        self.status = status
        self.body_snippet = body_snippet
        self.envelope_code = envelope_code
        super().__init__(self.__str__())

    def __str__(self) -> str:  # pragma: no cover
        parts = [self.message]
        if self.status is not None:
            parts.append(f"status={self.status}")
        if self.envelope_code is not None:
            parts.append(f"code={self.envelope_code}")
        if self.body_snippet:
            parts.append(f"body={self.body_snippet}")
        return "; ".join(parts)


class HttpError(ApiError):
    """HTTP non-2xx error."""
    pass


def snippet(text: Optional[str], max_len: int = 300) -> Optional[str]:
    if not text:
        return None
    return text[:max_len]
