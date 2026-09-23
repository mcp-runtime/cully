"""Compatibility entry point for Buddy's remote MCP service.

The implementation lives in :mod:`buddy_mcp`; this module preserves the old
``python buddy_service.py`` command and validation imports.
"""

from buddy_mcp.server import main
from buddy_mcp.validation import EMBEDDING_DIM, checked_embedding, checked_text, normalize_project_url

__all__ = ["EMBEDDING_DIM", "checked_embedding", "checked_text", "normalize_project_url", "main"]


if __name__ == "__main__":
    main()
