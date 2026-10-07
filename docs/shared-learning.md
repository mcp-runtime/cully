---
title: Shared learning design
description: Design for letting a team's coding agents learn from each other's sessions, planned and not yet available.
---

# Shared learning design

::: warning Planned, not available yet
Cully's records are private to their owner today. This page describes the design for team shared learning so the work can be reviewed before it is built. See [run projects with agents](/team-workflows) for what works now.
:::

## Goal

In team mode, what one person's coding agent learns should help the rest of the team. A teammate's agent finds something that works, Cully keeps it, and a different teammate's agent gets it at the start of a relevant session. The learning can be about anything: a debugging approach, a release step, a product decision, an ops check, a review habit or a way of prompting. A coding agent plus skills is the whole tool; Cully adds no workflow engine.

## A new component beside the advisor

Shared learning is a new part of Cully, the learning system. It sits next to the local advisor and the memory service and does a different job.

| Component | Scope | Job |
| --- | --- | --- |
| Local advisor | One person, on their machine | Reads session signals and suggests improvements to instructions, skills and MCP setup. Works offline. |
| Memory | One owner's records | Stores and finds notes through MCP. |
| Learning system (new) | A team, on the server | Collects lessons people choose to share, groups repeated strategies, and works out which ones help whom. |

The learning system builds on memory for storage and Mem0 for similarity, and it reuses the advisor's suggestion flow for delivery. It runs as a worker in the data service, like the Mem0 projection worker, and exposes its results through MCP tools.

The advisor keeps its current boundary: it holds no OAuth token and works without a server. So team tips reach the advisor through the agent. The agent fetches them with its own MCP identity, and an installed hook saves them to the local suggestion store. The advisor then lists them in `cully suggestions` like any other suggestion, and they stay available offline once fetched.

## Principles

1. **Private by default.** A note becomes visible to the team only when it is marked `team`. The author can edit or withdraw it at any time.
2. **Generic, not tied to one kind of work.** Incidents are one example. The model does not contain incident-specific fields.
3. **One authorization path.** Team reads and writes go through the shared validation in `internal/memory`, used by both the MCP and data API transports.
4. **No transcripts.** Cully stores distilled notes only, and keeps rejecting recognized secret patterns.
5. **Agent-side distillation first.** The connected agent writes the lesson, so the server needs no model or key.

## The record

A shared learning is a small record built on the existing `learning` entry type:

| Field | Meaning |
| --- | --- |
| Lesson | What to do or avoid. |
| Applies when | Free text describing the situation the lesson fits. Matching uses this field. |
| Evidence | What happened that taught it. |
| Author, project, tags | Attribution and context. An optional free-form `kind` tag, such as `debugging`, `release` or `review`, helps people browse; matching does not depend on it. |
| Visibility | `private` (default) or `team`. |

## Data model changes

- New `teams` and `memberships` tables. Membership comes from an admin-managed table in the first version; mapping an identity-provider group claim can follow.
- `cully_entries` gains `team_id` and `visibility`, with an index for team reads.
- The author's `owner_subject` stays on every record. Withdrawing a record removes its Mem0 projection through the existing projection job.

## Delivery in four slices

**1. Share a learning.** `cully_log` accepts `visibility: team`. The Cully skill tells the agent to share a reusable lesson, with when it applies, and to keep ordinary work notes private.

**2. Find shared learnings.** `cully_context`, `cully_recall`, `cully_search` and `cully_recent` accept `scope: team`. A team read checks membership in one shared function. At session start, `cully_context` returns up to three short team tips for the project and task. Each shows its author, date and record ID, and `cully_get` opens the full note.

**3. Learn the strategy.** At the end of a session the agent writes a short strategy note: what it tried first, what worked and what it would change. The server groups similar notes by Mem0 similarity. When several people converge on an approach, it ranks that tip higher and keeps its sources.

**4. Suggest it to a teammate.** The learning system compares a member's own notes with the team's strategies, using that member's own MCP identity. When a teammate uses one that this member does not, it returns a tip. A hook saves the tip locally, and the advisor lists it in `cully suggestions`. Applying it goes through `cully apply --dry-run` and writes to a shared instruction or skill. Accept and dismiss feedback is stored and used for ranking. Each tip links to its source notes and the matching [optimization guide](/session-optimization).

Role skills such as dev, ops and product, incident grouping and webhook-started agent runs are optional layers that can follow once this loop works.

## Safety and quality

| Risk | Plan |
| --- | --- |
| A private note leaks | Visibility defaults to `private`, and nothing is auto-shared. Tests cover non-members, withdrawn notes and private notes in team results. |
| An unsafe note is shared | Existing secret rejection stays. The skill adds a short "safe to share" rule. |
| Tips become noise | Three tips per session at most, ranked by relevance and acceptance feedback. |
| Trust in a tip | Every tip shows its author and source, and the author can withdraw it. |

## Measuring it

Track the share of sessions that retrieve a team tip, tip acceptance rate, repeat work on the same problem and time to resolve. Change ranking only from those results.

## Open decisions

- Team identity: admin-managed table first, or a group claim from the start.
- Visibility default: private unless shared (recommended), or team-visible for `learning` notes.
- Distillation: agent-side only (recommended), or add an optional server-side summarizer later.
