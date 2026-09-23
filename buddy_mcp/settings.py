"""Runtime configuration and shared time-zone policy."""

from __future__ import annotations

import os
from dataclasses import dataclass
from datetime import timedelta, timezone


IST = timezone(timedelta(hours=5, minutes=30), name="IST")
DEFAULT_ISSUER = "https://auth.mcpruntime.org/mcp-auth"
DEFAULT_RESOURCE = "https://workspace.mcpruntime.org/buddy/mcp"


@dataclass(frozen=True)
class BuddySettings:
    database_url: str
    issuer: str = DEFAULT_ISSUER
    resource: str = DEFAULT_RESOURCE
    host: str = "0.0.0.0"
    port: int = 8080

    @classmethod
    def from_env(cls) -> "BuddySettings":
        database_url = os.environ.get("BUDDY_DATABASE_URL", "").strip()
        if not database_url:
            raise RuntimeError("BUDDY_DATABASE_URL is required")
        return cls(
            database_url=database_url,
            issuer=os.getenv("MCP_AUTH_ISSUER", DEFAULT_ISSUER).rstrip("/"),
            resource=os.getenv("BUDDY_MCP_URL", DEFAULT_RESOURCE).rstrip("/"),
            host=os.getenv("BUDDY_HOST", "0.0.0.0"),
            port=int(os.getenv("BUDDY_PORT", "8080")),
        )
