ALTER TABLE cully_entries DROP CONSTRAINT IF EXISTS cully_entries_entry_type_check;
-- Add without scanning existing rows while this migration holds an exclusive
-- lock. Migration 005 validates after this transaction commits.
ALTER TABLE cully_entries ADD CONSTRAINT cully_entries_entry_type_check CHECK (entry_type IN ('work','issue','learning','decision','task')) NOT VALID;
CREATE TABLE cully_sessions (
  owner_subject text NOT NULL,
  session_ref text NOT NULL,
  section text NOT NULL CHECK (section IN ('personal','company')),
  project_url text,
  assistant text NOT NULL,
  branch text,
  task_id uuid REFERENCES cully_entries(id) ON DELETE SET NULL,
  started_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (owner_subject, session_ref)
);
CREATE INDEX cully_sessions_owner_project_seen_idx ON cully_sessions(owner_subject, project_url, last_seen_at DESC);
