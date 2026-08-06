from __future__ import annotations

from typing import Any, Dict
import os
import re
import json

from typing import Any, Dict, Generic, List, Optional, Type, TypeVar, Union, Literal
from pydantic import BaseModel, Field, ConfigDict, TypeAdapter, model_validator


T = TypeVar("T")


def parse_as(tp: Type[T], obj: Any) -> T:
    """Pydantic v2 typed parsing via TypeAdapter."""
    return TypeAdapter(tp).validate_python(obj)

class BuildLogsResponse(BaseModel):
    """Build logs response containing logs and metadata."""
    model_config = ConfigDict(extra="allow")

    name: str
    logs: List[Dict[str, Any]]
    raw_logs: Optional[Any] = None


ANSI_ESCAPE_RE = re.compile(r"\x1B\[[0-?]*[ -/]*[@-~]")



class ResponseEnvelope(BaseModel, Generic[T]):
    """Generic response envelope structure {code, data, error?, message?}."""

    model_config = ConfigDict(extra="allow")

    code: int
    data: Optional[T] = None
    error: Optional[str] = None
    message: Optional[str] = None


def try_parse_envelope(obj: Any, tp: Type[T]) -> Union[ResponseEnvelope[T], T]:
    """If data looks like {code, data}, parse as envelope; otherwise parse as target type."""
    if isinstance(obj, dict) and "code" in obj and "data" in obj:
        env = TypeAdapter(ResponseEnvelope[tp]).validate_python(obj)  # type: ignore[index]
        return env
    return parse_as(tp, obj)


class CicdCreateRequest(BaseModel):
    """/api/v2/versions/cicd_create/ request body model."""

    model_config = ConfigDict(extra="allow")

    create_user: Optional[str] = None
    repo_id: Optional[int] = None
    repo_name: Optional[str] = None
    version: Optional[str] = None
    desc: Optional[str] = None
    type: Optional[Literal["online", "offline", "test", "private"]] = None
    pub_base: Optional[Literal["branch_base", "commit_base", "tag_base"]] = None
    branch_name: Optional[str] = None
    commit_hash: Optional[str] = None
    git_tag: Optional[str] = None
    user_envs: Optional[Dict[str, str]] = Field(default=None, description="CUSTOM_* envs")
    build_image: Optional[str] = None
    arch: Optional[List[str]] = Field(default_factory=lambda: ["x86_64"])
    has_preonline: Optional[bool] = None
    sync_aws: Optional[bool] = None
    sync_bvc: Optional[bool] = None
    sync_oss: Optional[bool] = None
    skip_version_check: Optional[bool] = None
    mergebuild: Optional[List[Any]] = None
    use_cache: Optional[bool] = None

    def to_api_payload(self) -> Dict[str, Any]:
        """Render as API payload dict; serialize user_envs as JSON string."""
        data: Dict[str, Any] = {}
        if self.create_user is not None:
            data["create_user"] = self.create_user
        if self.pub_base is not None:
            data["pub_base"] = self.pub_base
        if self.repo_id is not None:
            data["repo_id"] = self.repo_id
        if self.repo_name is not None:
            data["repo_name"] = self.repo_name
        if self.version is not None:
            data["version"] = self.version
        if self.desc is not None:
            data["desc"] = self.desc
        if self.type is not None:
            data["type"] = self.type
        if self.branch_name is not None:
            data["branch_name"] = self.branch_name
        if self.commit_hash is not None:
            data["commit_hash"] = self.commit_hash
        if self.git_tag is not None:
            data["git_tag"] = self.git_tag
        if self.build_image is not None:
            data["build_image"] = self.build_image
        if self.arch is not None:
            data["arch"] = list(self.arch)
        if self.has_preonline is not None:
            data["has_preonline"] = self.has_preonline
        if self.sync_aws is not None:
            data["sync_aws"] = self.sync_aws
        if self.sync_bvc is not None:
            data["sync_bvc"] = self.sync_bvc
        if self.sync_oss is not None:
            data["sync_oss"] = self.sync_oss
        if self.skip_version_check is not None:
            data["skip_version_check"] = self.skip_version_check
        if self.mergebuild is not None:
            data["mergebuild"] = self.mergebuild
        if self.use_cache is not None:
            data["use_cache"] = self.use_cache
        if self.user_envs is not None:
            data["user_envs"] = json.dumps(
                self.user_envs,
                ensure_ascii=False,
                separators=(",", ":"),
            )
        return data


class CicdContext(BaseModel):
    model_config = ConfigDict(extra="allow")

    url: Optional[str] = None
    repo_id: Optional[int] = None
    version_version: Optional[str] = None
    version_id: Optional[int] = None


class CicdCreateData(BaseModel):
    model_config = ConfigDict(extra="allow")

    state: Optional[str] = None
    context: Optional[CicdContext] = None


class CicdCreateResponse(BaseModel):
    """Convenience wrapper when endpoint returns data directly without envelope."""

    model_config = ConfigDict(extra="allow")

    state: Optional[str] = None
    context: Optional[CicdContext] = None


class RepoInfo(BaseModel):
    model_config = ConfigDict(extra="allow")

    id: Optional[int] = None
    repo_name: Optional[str] = None
    disabled: Optional[bool] = None
    git_name: Optional[str] = None


class ReposByNamesResponse(BaseModel):
    """Lightweight wrapper for repo list when API doesn't use envelope."""

    model_config = ConfigDict(extra="allow")

    repos: Optional[List[RepoInfo]] = Field(default_factory=list)


class VersionDetail(BaseModel):
    model_config = ConfigDict(extra="allow")

    repos: Optional[int] = None
    type: Optional[str] = None
    commit_hash: Optional[str] = None
    branch_name: Optional[str] = None
    arch: Optional[List[str]] = None
    builds: Optional[List[Dict[str, Any]]] = None
    status_display: Optional[str] = None
    status: Optional[str] = None
    repo_name: Optional[str] = None
    version: Optional[str] = None
    build_url: Optional[str] = None
    failed_step: Optional[str] = None
    build_info: Optional[Dict[str, Dict[str, Any]]] = None

    def is_build_ok(self) -> bool:
        return self.status == "build_ok"

    def is_build_failed(self) -> bool:
        return self.status == "build_failed"

    def is_building(self) -> bool:
        return self.status == "building"

    def is_terminal(self) -> bool:
        if self.is_build_ok() or self.is_build_failed():
            return True
        return False

    def get_build_num(self) -> Optional[str]:
        if not self.build_info:
            return None
        for arch_info in self.build_info.values():
            if not arch_info.get("build_num", None):
                continue
            return arch_info["build_num"]
        return None

    def get_build_ok_string(self, version_id: str):
        return f"""
✅ SCM Build Success!

Repository: {self.repo_name}
Branch: {self.branch_name}
Version: {self.version}
Version ID: {version_id}
Build Status: {self.status_display}

Build Log: {self.build_url}
"""

    def get_build_failed_string_with_logs(self, version_id, build_logs):
        build_logs_text = self._format_build_logs(build_logs)
        build_logs_unfiltered_text = self._format_build_logs_unfiltered(build_logs)

        try:
            # Use SCM_LOG_OUTPUT_DIR env var, fallback to ./build-logs relative to tool directory
            log_dir = os.environ.get("SCM_LOG_OUTPUT_DIR")
            if not log_dir:
                # Default to build-logs directory relative to tool location
                tool_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
                log_dir = os.path.join(tool_dir, "build-logs")

            if not os.path.exists(log_dir):
                os.makedirs(log_dir, exist_ok=True)

            build_num = self.get_build_num() or "unknown"
            filename = f"{self.version}-{build_num}.log"
            file_path = os.path.join(log_dir, filename)

            with open(file_path, "w", encoding="utf-8") as f:
                f.write(build_logs_text)

            filename_unfiltered = f"{self.version}-{build_num}-unfiltered.log"
            file_path_unfiltered = os.path.join(log_dir, filename_unfiltered)

            with open(file_path_unfiltered, "w", encoding="utf-8") as f:
                f.write(build_logs_unfiltered_text)

            file_size = os.path.getsize(file_path)
            line_count = len(build_logs_text.splitlines())

            log_details = f"Log Path: {file_path}\nLine Count: {line_count}\nLog Size: {file_size} bytes"
        except Exception as e:
            log_details = f"Failed to save log: {e}\n\nRaw log:\n{build_logs_text}"

        return f"""
❌ SCM Build Failed!

Repository: {self.repo_name}
Branch: {self.branch_name}
Version: {self.version}
Version ID: {version_id}
Build Status: {self.status_display}
Failed Step: {self.failed_step}
Build Number: {self.get_build_num()}

Build Log Info:
{log_details}

Tips: Key error messages are usually located at the end of the log
"""

    def get_build_failed_string_without_logs(self, version_id: str):
        return f"""
❌ SCM Build Failed!

Repository: {self.repo_name}
Branch: {self.branch_name}
Version: {self.version}
Version ID: {version_id}
Build Status: {self.status_display}
Failed Step: {self.failed_step}

Note: Could not retrieve build number. Please check SCM dashboard for detailed logs.
"""

    @staticmethod
    def _format_build_logs(build_logs: Any) -> str:
        if not isinstance(build_logs, list):
            return ANSI_ESCAPE_RE.sub("", str(build_logs))
        lines = []
        skip_tokens = (" Annotate ", "go: downloading")
        skip_patterns = [
            re.compile(r"Blade\(info\):"),
            re.compile(r"CUDA deps:"),
            re.compile(r"^\+"),
            re.compile(r" \++ "),
            re.compile(r" \++\["),
            re.compile(r" \+\["),
            re.compile(r"\[\d+/\d+\]"),
        ]

        for entry in build_logs:
            if not isinstance(entry, dict):
                message = ANSI_ESCAPE_RE.sub("", str(entry))
                if any(token in message for token in skip_tokens):
                    continue
                if any(pattern.search(message) for pattern in skip_patterns):
                    continue
                lines.append(message)
                continue
            line_no = entry.get("l", "")
            message_str = ANSI_ESCAPE_RE.sub("", str(entry.get("m", "")))
            if any(token in message_str for token in skip_tokens):
                continue
            if any(pattern.search(message_str) for pattern in skip_patterns):
                continue
            lines.append(f"{line_no}:{message_str}")
        return "\n".join(lines)

    @staticmethod
    def _format_build_logs_unfiltered(build_logs: Any) -> str:
        if not isinstance(build_logs, list):
            return ANSI_ESCAPE_RE.sub("", str(build_logs))
        lines = []
        for entry in build_logs:
            if not isinstance(entry, dict):
                lines.append(ANSI_ESCAPE_RE.sub("", str(entry)))
                continue
            line_no = entry.get("l", "")
            message_str = ANSI_ESCAPE_RE.sub("", str(entry.get("m", "")))
            lines.append(f"{line_no}:{message_str}")
        return "\n".join(lines)

    def get_build_timeout_string(self, version_id: str, max_poll_attempts: int, poll_interval: int):
        return f"""
⏰ SCM Build Timeout!

Repository: {self.repo_name}
Branch: {self.branch_name}
Version: {self.version}
Version ID: {version_id}
Max Poll Attempts: {max_poll_attempts}
Poll Interval: {poll_interval}s

Build may still be in progress. Please check SCM dashboard:
https://cloud.bytedance.net/scm/detail/{self.repos}/versions/?simpleLayout=1
"""


class VersionDetailResponse(BaseModel):
    model_config = ConfigDict(extra="allow")

    detail: Optional[VersionDetail] = None

