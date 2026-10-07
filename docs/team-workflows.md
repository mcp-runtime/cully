---
title: Run projects with coding agents
description: How a team uses coding-agent sessions, Cully memory and the advisor to manage project work, including production issues.
---

# Run projects with coding agents

Cully treats the coding agent your team already uses as the way work gets done and recorded. A person starts a session in Claude Code, Codex or Cursor. The Cully skill and hooks make that session start from earlier notes, and save a short record of what happened before it ends. There is no separate workflow engine to build: the agent does the work, and the skill tells it how to keep the project's trail.

This page explains how that works today, how it fits a team, and which parts are still a direction rather than a feature. For the mechanics of one session, see [keep sessions focused](/session-optimization) and [memory](/memory).

## What an agent does in a session

| Moment | What the agent does | Cully tool |
| --- | --- | --- |
| Task starts | Looks up a few short notes about this project and task, and opens a full note only when it matters. | `cully_context`, `cully_get` |
| A blocker or choice appears | Records the reusable part as it happens. | `cully_log` with `issue`, `learning` or `decision` |
| Work finishes | Saves one concise note: what changed, why, what worked, what failed, and what should happen next. | `cully_log` with `work` |
| Someone asks about past work | Searches the owner's notes and cites a date or record ID. | `cully_search`, `cully_recall`, `cully_projects` |

Because every agent calls the same MCP tools, a note written from Codex can start a session in Claude Code. The `next steps` field is the handoff: the next session reads it instead of rereading a conversation.

## Example: fixing a production issue

1. A developer starts a session and describes the incident. The agent calls `cully_context` for the service and finds a note from an earlier outage.
2. While debugging, the agent logs the cause as an `issue` and the check that confirmed it as a `learning`.
3. The fix ships. The agent saves a `work` note with the change, the verification and a follow-up, such as an alert that should exist.
4. The next day, the same person or another of their agents asks what happened with the service. The agent searches the notes and answers with the dates and records behind it.

The same shape works for planning, review, release and support tasks. The role changes, but the loop stays the same: look up context, do the work, record the result, hand off.

## Where optimization fits

A session that records its work can also be improved. The local advisor reads the signals your agent exposes, such as context pressure, repeated tool faults or searches, and a missing verifier. It suggests a project instruction, skill or MCP connection that would help. You preview a suggestion with `cully apply <n> --dry-run` before anything changes. See the [local advisor](/advisor) for the commands and [session optimization](/session-optimization) for how the pieces work together.

Claude Code supplies the richest live signals. Codex and Cursor get continuity prompts and Cully commands, but advice depends on the data each client exposes.

## Solo, self-hosted and team

| Mode | What it gives you | Start with |
| --- | --- | --- |
| Try it on your laptop | The full stack in Docker with one stable owner and no sign-in. A good way to get a feel for the loop above. | [Quickstart](/quickstart), [self-hosting](/hosting) |
| Team server | One Cully server for many people, each signed in through the company's identity provider. Everyone's agents write to the same server, and each person's records stay private to them. | [Team deployment](/team-deployment), [OAuth](/oauth) |

A team server is what lets one person move between laptops and agents without losing their trail, and gives ops one place to run and secure the service.

## Learning across a team: direction, not a feature yet

The goal is for what one person's agent learns to help others. For example, a debugging approach that worked during a production issue could surface as a suggestion for someone in ops or product. Cully does not do this today:

- Records are owner-scoped. The `company` section is a label for one person's notes. Coworkers cannot read each other's records, and the advisor does not send suggestions between people.
- Cully does not schedule agents, assign work or trigger sessions. An agent acts when a person starts a session.

Sharing what a team learns needs an explicit contract for who can see a note, how it is attributed, and how its author edits or withdraws it. The [shared learning design](/shared-learning) describes how it would work, and the [roadmap](/roadmap) lists it as planned work. Until it ships, a team can still share a lesson the usual way: the person copies a note's text into the project's instructions or a skill that everyone's agents load.

## Next steps

- Try the loop on your laptop with the [quickstart](/quickstart).
- Read [memory](/memory) for what a record contains and how to find it.
- Ops teams can follow [team deployment](/team-deployment) to give each person their own signed-in records.
