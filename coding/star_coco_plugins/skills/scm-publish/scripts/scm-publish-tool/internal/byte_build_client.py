from __future__ import annotations

import asyncio
import random
from typing import Iterable, Optional, Set

import aiohttp
from aiohttp import ClientResponse

from internal.exceptions import ApiError, HttpError, snippet
from internal.models import BuildLogsResponse


class BytebuildNightlyClient:
    """Async client for ByteBuild Nightly unauthenticated API.
    API Doc: https://cloud.bytedance.net/bam/rd/scm.bytebuild.nightly/api_doc/show_doc

    Base URL: https://bytebuild-nightly.byted.org
    Endpoint: GET /api/v1/record/logs/{record_id}/building

    Supports HTTP status-based retry with exponential backoff (e.g., for 404
    caused by log propagation delays).
    """

    __slots__ = (
        "_session",
        "base_url",
        "timeout",
        "retries",
        "retry_statuses",
        "retry_backoff_base",
        "retry_backoff_max",
        "retry_jitter",
    )

    def __init__(
        self,
        *,
        base_url: str = "https://bytebuild-nightly.byted.org",
        timeout: float = 30.0,
        retries: int = 3,
        retry_statuses: Optional[Iterable[int]] = None,
        retry_backoff_base: float = 1.0,
        retry_backoff_max: float = 10.0,
        retry_jitter: float = 0.25,
        session: Optional[aiohttp.ClientSession] = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.retries = max(0, int(retries))
        self.retry_statuses: Set[int] = set(retry_statuses or {404})
        self.retry_backoff_base = max(0.0, float(retry_backoff_base))
        self.retry_backoff_max = max(self.retry_backoff_base, float(retry_backoff_max))
        self.retry_jitter = max(0.0, float(retry_jitter))
        self._session = session

    async def __aenter__(self) -> "BytebuildNightlyClient":
        if self._session is None:
            timeout = aiohttp.ClientTimeout(total=self.timeout)
            self._session = aiohttp.ClientSession(timeout=timeout)
        return self

    async def __aexit__(self, exc_type, exc, tb) -> None:
        await self.close()

    async def close(self) -> None:
        if self._session and not self._session.closed:
            await self._session.close()

    async def _ensure_session(self) -> aiohttp.ClientSession:
        if self._session is None or self._session.closed:
            timeout = aiohttp.ClientTimeout(total=self.timeout)
            self._session = aiohttp.ClientSession(timeout=timeout)
        return self._session

    async def get_building_logs(self, record_id: int, step_name: str) -> BuildLogsResponse:
        """Fetch building logs for a ByteBuild record.

        GET /api/v1/record/logs/{record_id}/building

        No authentication required. Do not attach Authorization or domain headers.
        """
        path = f"/api/v1/record/logs/{int(record_id)}/{step_name}"
        url = f"{self.base_url}{path}"

        attempt = 0
        last_exc: Optional[Exception] = None
        while attempt <= self.retries:
            try:
                session = await self._ensure_session()
                async with session.get(url, headers={"Accept": "application/json"}, params={"limit": "-1"}) as resp:
                    try:
                        return await self._parse_build_logs(resp)
                    except HttpError as e:
                        last_exc = e
                        if (
                            e.status in self.retry_statuses
                            and attempt < self.retries
                        ):
                            await self._sleep_with_backoff(attempt)
                            attempt += 1
                            continue
                        raise
            except (aiohttp.ClientError, asyncio.TimeoutError) as e:
                last_exc = e
                if attempt == self.retries:
                    raise ApiError(
                        f"Network/timeout error while calling ByteBuild Nightly: {e.__class__.__name__}",
                        status=None,
                        body_snippet=None,
                    ) from e
                attempt += 1
                await self._sleep_with_backoff(attempt - 1)
        if last_exc:
            raise last_exc
        raise ApiError("Unknown error in get_building_logs")

    async def _sleep_with_backoff(self, attempt: int) -> None:
        """Sleep using exponential backoff with optional jitter."""
        delay = min(
            self.retry_backoff_base * (2 ** attempt),
            self.retry_backoff_max,
        )
        if self.retry_jitter > 0:
            delay += random.random() * self.retry_jitter * delay
        await asyncio.sleep(delay)

    async def _parse_build_logs(self, resp: ClientResponse) -> BuildLogsResponse:
        status = resp.status
        if status < 200 or status >= 300:
            body = await resp.text()
            raise HttpError(
                "ByteBuild Nightly API returned non-2xx",
                status=status,
                body_snippet=snippet(body),
            )

        try:
            data = await resp.json(content_type=None)
        except Exception as e:
            body = None
            try:
                body = await resp.text()
            except Exception:
                pass
            raise ApiError(
                "Failed to parse JSON for BuildLogsResponse",
                status=status,
                body_snippet=snippet(body),
            ) from e

        try:
            return BuildLogsResponse.model_validate(data)
        except Exception as e:
            raise ApiError(
                "Response did not match BuildLogsResponse schema",
                status=status,
                body_snippet=None,
            ) from e
