"""Timestamp parsing and serialization for Buddy's IST-facing contract."""

from __future__ import annotations

from datetime import datetime

from buddy_mcp.settings import IST


def parse_timestamp(value: str, *, field: str = "timestamp") -> datetime:
    try:
        parsed = datetime.fromisoformat(value)
    except ValueError as exc:
        raise ValueError(f"{field} must be an ISO 8601 timestamp") from exc
    if parsed.tzinfo is None:
        raise ValueError(f"{field} must include a timezone")
    return parsed.astimezone(IST)


def now_ist() -> datetime:
    return datetime.now(IST)


def format_timestamp(value: datetime | None) -> str | None:
    if value is None:
        return None
    if value.tzinfo is None:
        raise ValueError("database timestamp must include a timezone")
    return value.astimezone(IST).isoformat()
