"""aiohttp-based async client for ByteDance SCM OpenAPI.

- Always sends headers: domain: "scm;v1" and x-jwt-token: <personal JWT>
- Robust error handling: non-2xx or envelope.code != 0 -> ApiError
- Typed models and deserialization via Pydantic
"""
from __future__ import annotations

import json
from typing import Any, Dict, List, Optional, Tuple, TypeVar, Union

import aiohttp

from internal import models
from internal.exceptions import ApiError
from internal.scm_auth import get_token

T = TypeVar("T")


class SCMClient:
    """Async SCM OpenAPI client using aiohttp.

    Args:
        base_url: Base URL, defaults to https://scm.byted.org
        timeout: Total timeout in seconds for requests
        session: Optional external aiohttp.ClientSession; if not provided, client manages its own
    """

    def __init__(
        self,
        base_url: str = "https://scm.byted.org",
        *,
        timeout: float = 30.0,
        session: Optional[aiohttp.ClientSession] = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self._timeout = aiohttp.ClientTimeout(total=timeout)
        self._session_external = session is not None
        self._session = session

    async def __aenter__(self) -> "SCMClient":
        await self._ensure_session()
        return self

    async def __aexit__(self, exc_type, exc, tb) -> None:  # pragma: no cover
        await self.close()

    async def close(self) -> None:
        if self._session and not self._session_external:
            await self._session.close()
            self._session = None

    async def _ensure_session(self) -> None:
        if self._session is None:
            self._session = aiohttp.ClientSession(timeout=self._timeout)

    def _headers(self) -> Dict[str, str]:
        return {
            "x-jwt-token": get_token(),
            'Content-Type': 'application/json',
            'Accept': '*/*',
            'domain': "scm;v1",
        }

    async def _request(
        self,
        method: str,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        json_body: Optional[Dict[str, Any]] = None,
    ) -> Tuple[int, Any]:
        await self._ensure_session()
        assert self._session is not None
        url = f"{self.base_url}{path}"
        headers = self._headers()
        async with self._session.request(method, url, headers=headers, params=params, json=json_body) as resp:
            status = resp.status
            text = await resp.text()
            try:
                data = json.loads(text)
            except Exception:
                data = text
            if status < 200 or status >= 300:
                snippet = text[:300] if isinstance(text, str) else str(text)[:300]
                raise ApiError(
                    f"HTTP request failed: {method} {path}", status=status, body_snippet=snippet
                )
            return status, data

    # --- Public methods ---
    async def cicd_create(self, req: models.CicdCreateRequest) -> models.CicdCreateData:
        """POST /api/v2/versions/cicd_create/"""
        payload = req.to_api_payload()
        status, data = await self._request(
            "POST",
            "/api/v2/versions/cicd_create/",
            json_body=payload,
        )
        parsed = models.try_parse_envelope(data, models.CicdCreateData)
        if isinstance(parsed, models.ResponseEnvelope):
            if parsed.code != 0:
                snippet = str(data)[:300]
                raise ApiError(
                    "Business response failed (envelope.code != 0)",
                    status=status,
                    envelope_code=parsed.code,
                    body_snippet=snippet,
                )
            return parsed.data or models.CicdCreateData()
        return parsed

    async def repos_by_names(
        self, repo_names: List[str], *, universe: Optional[bool] = None
    ) -> models.ReposByNamesResponse:
        """GET /api/v2/repos/by_names

        Query parameter encoding uses repeated keys: repo_names=a&repo_names=b.
        """
        params: Dict[str, Any] = {"repo_names": list(repo_names)}
        if universe is not None:
            params["universe"] = str(universe).lower()
        status, data = await self._request("GET", "/api/v2/repos/by_names", params=params)
        parsed = models.try_parse_envelope(data, List[models.RepoInfo])
        if isinstance(parsed, models.ResponseEnvelope):
            if parsed.code != 0:
                raise ApiError(
                    "Business response failed (envelope.code != 0)",
                    status=status,
                    envelope_code=parsed.code,
                    body_snippet=str(data)[:300],
                )
            repos = parsed.data or []
            return models.ReposByNamesResponse(repos=repos)
        repos_list = parsed
        return models.ReposByNamesResponse(repos=repos_list)

    async def get_version(
        self, repo_id: Union[int, str], version_name: str
    ) -> models.VersionDetail:
        """GET /api/repos/{repo_id}/versions/{version_name}"""
        path = f"/api/repos/{repo_id}/versions/{version_name}"
        status, data = await self._request("GET", path)
        parsed = models.try_parse_envelope(data, models.VersionDetail)
        if isinstance(parsed, models.ResponseEnvelope):
            if parsed.code != 0:
                raise ApiError(
                    "Business response failed (envelope.code != 0)",
                    status=status,
                    envelope_code=parsed.code,
                    body_snippet=str(data)[:300],
                )
            return parsed.data or models.VersionDetail()
        return parsed
