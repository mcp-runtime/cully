"""Validation and normalization for Buddy tool inputs."""

from __future__ import annotations

import math
import re
from urllib.parse import urlparse


MAX_TEXT = 8000
MAX_TAGS = 20
EMBEDDING_DIM = 1536
SECRET_PATTERNS = [
    re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    re.compile(r"\b(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})\b"),
    re.compile(r"(?i)\b(?:password|secret|api[_-]?key|access[_-]?token)\s*[:=]\s*[^\s,;]{8,}"),
]


def normalize_project_url(value: str) -> str:
    value = value.strip()
    match = re.fullmatch(r"git@github\.com:([^/]+)/([^/]+?)(?:\.git)?", value, re.I)
    if match:
        owner, repo = match.groups()
    else:
        parsed = urlparse(value)
        if parsed.scheme not in {"https", "http"} or parsed.hostname != "github.com":
            raise ValueError("project_url must be a GitHub repository URL")
        pieces = parsed.path.strip("/").split("/")
        if len(pieces) != 2:
            raise ValueError("project_url must point to OWNER/REPO")
        owner, repo = pieces
        repo = repo.removesuffix(".git")
    if not re.fullmatch(r"[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?", owner):
        raise ValueError("invalid GitHub owner in project_url")
    if not re.fullmatch(r"[A-Za-z0-9_.-]{1,100}", repo):
        raise ValueError("invalid GitHub repository in project_url")
    return f"https://github.com/{owner.lower()}/{repo.lower()}"


def checked_text(field: str, value: str | None, *, required: bool = False) -> str | None:
    if value is None:
        if required:
            raise ValueError(f"{field} is required")
        return None
    value = value.strip()
    if required and not value:
        raise ValueError(f"{field} cannot be empty")
    if len(value) > MAX_TEXT:
        raise ValueError(f"{field} is longer than {MAX_TEXT} characters")
    if any(pattern.search(value) for pattern in SECRET_PATTERNS):
        raise ValueError(f"{field} appears to contain a credential; remove it before saving")
    return value or None


def checked_embedding(values: list[float] | None) -> str | None:
    if values is None:
        return None
    if len(values) != EMBEDDING_DIM:
        raise ValueError(f"embedding must contain exactly {EMBEDDING_DIM} values")
    vector = [float(value) for value in values]
    if not all(math.isfinite(value) for value in vector):
        raise ValueError("embedding values must be finite numbers")
    return "[" + ",".join(format(value, ".9g") for value in vector) + "]"
