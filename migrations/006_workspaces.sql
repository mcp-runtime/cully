-- A bounded small-team aggregate serializes membership checks and mutations
-- in one row transaction. Private memory tables and Mem0 remain untouched.
CREATE TABLE cully_workspaces (
    id uuid PRIMARY KEY,
    data jsonb NOT NULL CHECK (octet_length(data::text) <= 2097152),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE cully_workspace_events (
    id uuid PRIMARY KEY,
    team_id uuid NOT NULL REFERENCES cully_workspaces(id),
    project_id uuid,
    actor text NOT NULL,
    action text NOT NULL,
    target_id text NOT NULL,
    occurred_at timestamptz NOT NULL
);
CREATE INDEX cully_workspace_events_team_time ON cully_workspace_events(team_id, occurred_at);
