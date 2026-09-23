"""PostgreSQL persistence and owner-scoped Buddy queries."""

from __future__ import annotations

from datetime import datetime
from typing import Any
from uuid import UUID

from psycopg.rows import dict_row
from psycopg_pool import ConnectionPool

from buddy_mcp.time_utils import format_timestamp


SCHEMA = """
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE IF NOT EXISTS buddy_entries (
  id uuid PRIMARY KEY,
  owner_subject text NOT NULL,
  project_url text NOT NULL,
  theme text NOT NULL DEFAULT 'mcp',
  entry_type text NOT NULL CHECK (entry_type IN ('work','issue','learning','decision')),
  summary text NOT NULL,
  approach text,
  outcome text,
  issue text,
  learning text,
  next_steps text,
  assistant text NOT NULL,
  tags text[] NOT NULL DEFAULT '{}',
  occurred_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  search_vector tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(summary,'')), 'A') ||
    setweight(to_tsvector('english', coalesce(approach,'')), 'B') ||
    setweight(to_tsvector('english', coalesce(outcome,'')), 'B') ||
    setweight(to_tsvector('english', coalesce(issue,'')), 'A') ||
    setweight(to_tsvector('english', coalesce(learning,'')), 'A') ||
    setweight(to_tsvector('english', coalesce(next_steps,'')), 'C')
  ) STORED,
  embedding vector(1536)
);
ALTER TABLE buddy_entries ADD COLUMN IF NOT EXISTS embedding vector(1536);
ALTER TABLE buddy_entries ADD COLUMN IF NOT EXISTS owner_subject text;
CREATE INDEX IF NOT EXISTS buddy_entries_search_idx ON buddy_entries USING gin(search_vector);
CREATE INDEX IF NOT EXISTS buddy_entries_project_time_idx ON buddy_entries(project_url, occurred_at DESC);
CREATE INDEX IF NOT EXISTS buddy_entries_type_idx ON buddy_entries(entry_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS buddy_entries_owner_project_time_idx ON buddy_entries(owner_subject, project_url, occurred_at DESC);
CREATE INDEX IF NOT EXISTS buddy_entries_embedding_idx ON buddy_entries USING hnsw (embedding vector_cosine_ops) WHERE embedding IS NOT NULL;
"""

UPDATABLE_FIELDS = {
    "summary",
    "approach",
    "outcome",
    "issue",
    "learning",
    "next_steps",
    "tags",
    "embedding",
    "updated_at",
}


def as_record(row: dict[str, Any]) -> dict[str, Any]:
    result = dict(row)
    result.pop("search_rank", None)
    result.pop("embedding", None)
    result.pop("owner_subject", None)
    if result.get("id") is not None:
        result["id"] = str(result["id"])
    if "tags" in result:
        result["tags"] = list(result.get("tags") or [])
    for field in ("occurred_at", "created_at", "updated_at", "last_activity"):
        if field in result:
            result[field] = format_timestamp(result[field])
    return result


class BuddyRepository:
    """Owns SQL access and enforces per-subject data isolation."""

    def __init__(self, database_url: str) -> None:
        self._pool = ConnectionPool(
            database_url,
            min_size=1,
            max_size=8,
            kwargs={"row_factory": dict_row},
            open=False,
        )

    def initialize(self) -> None:
        self._pool.open(wait=True)
        with self._pool.connection() as conn:
            conn.execute(SCHEMA)

    def close(self) -> None:
        self._pool.close()

    def create_entry(self, owner_subject: str, values: dict[str, Any]) -> dict[str, Any]:
        row = self._fetch_one(
            """INSERT INTO buddy_entries
               (id,owner_subject,project_url,entry_type,summary,approach,outcome,issue,learning,next_steps,assistant,tags,embedding,occurred_at)
               VALUES (%(id)s,%(owner_subject)s,%(project_url)s,%(entry_type)s,%(summary)s,%(approach)s,%(outcome)s,%(issue)s,%(learning)s,%(next_steps)s,%(assistant)s,%(tags)s,%(embedding)s::vector,%(occurred_at)s)
               RETURNING id,project_url,theme,entry_type,summary,approach,outcome,issue,learning,next_steps,assistant,tags,occurred_at,created_at,updated_at""",
            {**values, "owner_subject": owner_subject},
        )
        return as_record(row)

    def search_entries(
        self,
        owner_subject: str,
        *,
        query: str,
        embedding: str | None,
        project: str | None,
        entry_type: str | None,
        since: datetime | None,
        limit: int,
    ) -> list[dict[str, Any]]:
        rows = self._fetch_all(
            """SELECT id,project_url,theme,entry_type,summary,approach,outcome,issue,learning,next_steps,assistant,tags,occurred_at,created_at,updated_at,
                      (CASE WHEN %(embedding)s::vector IS NOT NULL AND embedding IS NOT NULL
                            THEN 1 - (embedding <=> %(embedding)s::vector) ELSE 0 END
                       + CASE WHEN %(query)s <> ''
                              THEN ts_rank_cd(search_vector, websearch_to_tsquery('english', %(query)s)) ELSE 0 END) AS search_rank
               FROM buddy_entries
               WHERE owner_subject=%(owner_subject)s
                 AND ((%(query)s <> '' AND search_vector @@ websearch_to_tsquery('english', %(query)s))
                   OR (%(embedding)s::vector IS NOT NULL AND embedding IS NOT NULL))
                 AND (%(project)s::text IS NULL OR project_url=%(project)s::text)
                 AND (%(entry_type)s::text IS NULL OR entry_type=%(entry_type)s::text)
                 AND (%(since)s::timestamptz IS NULL OR occurred_at >= %(since)s::timestamptz)
               ORDER BY search_rank DESC, occurred_at DESC LIMIT %(limit)s""",
            {
                "owner_subject": owner_subject,
                "query": query,
                "embedding": embedding,
                "project": project,
                "entry_type": entry_type,
                "since": since,
                "limit": limit,
            },
        )
        return [as_record(row) for row in rows]

    def recent_entries(
        self, owner_subject: str, *, project: str | None, entry_type: str | None, limit: int
    ) -> list[dict[str, Any]]:
        rows = self._fetch_all(
            """SELECT * FROM buddy_entries
               WHERE owner_subject=%(owner_subject)s
                 AND (%(project)s::text IS NULL OR project_url=%(project)s::text)
                 AND (%(entry_type)s::text IS NULL OR entry_type=%(entry_type)s::text)
               ORDER BY occurred_at DESC LIMIT %(limit)s""",
            {"owner_subject": owner_subject, "project": project, "entry_type": entry_type, "limit": limit},
        )
        return [as_record(row) for row in rows]

    def get_entry(self, owner_subject: str, identifier: UUID) -> dict[str, Any] | None:
        row = self._fetch_optional(
            "SELECT * FROM buddy_entries WHERE owner_subject=%s AND id=%s",
            (owner_subject, identifier),
        )
        return as_record(row) if row else None

    def update_entry(
        self, owner_subject: str, identifier: UUID, fields: dict[str, Any]
    ) -> dict[str, Any] | None:
        if not fields or not fields.keys() <= UPDATABLE_FIELDS:
            raise ValueError("invalid update fields")
        assignments = ",".join(
            f"{field}=%({field})s::vector" if field == "embedding" else f"{field}=%({field})s"
            for field in fields
        )
        row = self._fetch_optional(
            f"UPDATE buddy_entries SET {assignments} WHERE owner_subject=%(owner_subject)s AND id=%(id)s RETURNING *",
            {**fields, "owner_subject": owner_subject, "id": identifier},
        )
        return as_record(row) if row else None

    def delete_entry(self, owner_subject: str, identifier: UUID) -> bool:
        row = self._fetch_optional(
            "DELETE FROM buddy_entries WHERE owner_subject=%s AND id=%s RETURNING id",
            (owner_subject, identifier),
        )
        return row is not None

    def project_summaries(self, owner_subject: str, limit: int) -> list[dict[str, Any]]:
        rows = self._fetch_all(
            """SELECT project_url,count(*) AS entry_count,max(occurred_at) AS last_activity
               FROM buddy_entries WHERE owner_subject=%s
               GROUP BY project_url ORDER BY last_activity DESC LIMIT %s""",
            (owner_subject, limit),
        )
        return [as_record(row) for row in rows]

    def _fetch_one(self, query: str, params: Any) -> dict[str, Any]:
        with self._pool.connection() as conn:
            return conn.execute(query, params).fetchone()

    def _fetch_optional(self, query: str, params: Any) -> dict[str, Any] | None:
        with self._pool.connection() as conn:
            return conn.execute(query, params).fetchone()

    def _fetch_all(self, query: str, params: Any) -> list[dict[str, Any]]:
        with self._pool.connection() as conn:
            return conn.execute(query, params).fetchall()
