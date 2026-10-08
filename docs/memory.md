---
title: Use Cully memory
description: Save useful notes and find them later from connected coding agents.
---

# Use Cully memory

These tools remain owner-scoped. Team tasks and published project lessons use the separate [workspace workflow](/team-workflows); a `company` section or repository URL does not grant shared access. Workspace lessons currently use live project lookup, not Mem0 recall.

After [starting Cully on your laptop](/hosting) or [connecting to a team server](/agents), your agent uses Cully MCP tools to save and find useful work across sessions.

The installed Cully skill and session hooks ask Claude Code, Codex and Cursor to find relevant notes when substantive work starts and save a concise note when that work ends. A record can capture the project and task, approach, outcome, checks, blocker or missed step, and next step. When a client supplies a hook session ID, Cully gives the agent an opaque `session_ref` to save with the note. This distinguishes sessions without storing a raw client ID or transcript. The connected agent decides what is useful and calls `cully_log` with its own MCP identity. Codex asks you to trust newly installed hooks in `/hooks` before they run. The local advisor analyzes session signals for suggestions; it does not hold an OAuth token or upload session transcripts. You can still ask the agent to save, find or correct a note explicitly.

## What you can save

A record has a summary, the assistant that wrote it and a `personal` or `company` section. Work records can include a GitHub project URL, what changed, the result, a lesson and next steps. Personal records can cover goals or routines without a project URL. Avoid putting credentials or full transcripts in notes.

Both sections belong to the same owner. `company` is a label for organizing your notes; it does not share them with coworkers. On a default private server, the fixed owner configured by setup owns every record. With OAuth, each signed-in user's verified identity is the owner.

## Find and change notes

| Tool | Use it to |
| --- | --- |
| `cully_log` | Save a structured note. |
| `cully_context` | Get a few short, task-oriented previews before opening full records. |
| `cully_recent` | See recent notes. |
| `cully_search` | Find text in saved notes. |
| `cully_recall` | Find semantically related notes through Mem0. |
| `cully_get` | Read one note by ID. |
| `cully_update` | Correct a note. |
| `cully_delete` | Remove a note. |
| `cully_projects` | List projects with recent activity. |
| `cully_session` | Start or update the current agent session with its project, branch and an optional task name. A new task name saves one `task` record linked to the session; the same name again keeps or relinks that record without adding a duplicate. Omitting the task keeps the current link; `clear_task` unlinks it while keeping the task record. |
| `cully_session_get` | Get one session by its `session_ref`, including its task. |

Tool results contain `entry`, `entries`, `projects` or `deleted`, depending on the operation. Read tools require `tools:read` and mutations require `tools:write` when OAuth is enabled. Tasks are saved through `cully_session`; `cully_log` rejects `entry_type: task` and points at it instead.

PostgreSQL holds the source notes. `cully_search` and `cully_recent` read those records directly. Mem0 is part of the standard Cully stack and provides semantic candidates for `cully_recall`; Cully checks them against live, owned records before returning them. A new or edited note may take a little time to appear in recall, while text search remains available.

For a normal task, the agent starts with `cully_context`. It uses PostgreSQL text search when you provide a query, recent records when you do not, and Mem0 when `mode: semantic` is requested. The result is a bounded preview, with `cully_get` available for the full note. See [how Cully keeps sessions focused](/session-optimization).

You do not have to ask for every handoff: the installed hooks and skill prompt the connected agent during substantive work. To check a particular decision, ask the agent, for example, “What did we decide about OAuth for this repository, and which note supports it?” It should search the owned records and cite a date or record ID. If it cannot reach MCP, it should say that the note was not saved or found, rather than claiming continuity succeeded.

## Record details

`cully_log` requires a summary, assistant and section. Entry types are `work`, `issue`, `learning` and `decision`; `work` is the default. Optional fields include project URL, opaque session reference, category, approach, outcome, issue, learning, next steps and tags. GitHub project URLs normalize to `https://github.com/owner/repo`.

Text fields allow up to 8,000 characters and records allow up to 20 tags of 64 characters. Search and recent lookup can filter by project or `session_ref`; search also supports section, category, entry type and time. The reference groups Cully notes from one client session; it is not a link to reopen that client's conversation. Changes through `cully_update` affect only supplied fields, except that a task entry linked to a session keeps its session's section. See [Mem0 recall](/mem0) for its indexing behavior.

GitHub HTTPS and SSH remotes are accepted for `project_url` and normalize to the same URL. Personal notes need no project URL. Available categories are `career`, `fitness`, `relationship`, `finance`, `food`, `water`, `reading`, `mood`, `check-in` and `other`. `occurred_at` accepts an RFC3339 timestamp with an offset; Cully uses the current time when it is omitted and returns times in Asia/Kolkata (`+05:30`). The source database stores instants as `timestamptz`.

`cully_search` and `cully_recall` require query text. Search and recent limits are capped at 50 records; `cully_projects` is capped at 100. `cully_context` returns three short previews by default and at most five. An empty optional text value in `cully_update` clears that field; an empty summary is rejected. Reading or updating a missing or differently owned record returns no entry, and deletion reports whether an owned record was removed.

Shared notes are written by `cully_log`. The advisor's local counters and diagnostic snapshots support session guidance; they are not additional shared memory records. Cully rejects recognized secret patterns in note text, but you should still keep credentials out of notes. Embeddings are managed by Mem0 rather than supplied as tool inputs.
