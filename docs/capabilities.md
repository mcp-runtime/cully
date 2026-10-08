---
title: Current capabilities
description: What Cully does today, what works only partly, and what is planned, with the agent each capability applies to.
---

# Current capabilities

This page separates what ships today from what is planned. Cully's other pages describe only the first two groups.

**Shipped** means it works as described and has tests. **Partial** means it works with limits listed here. **Planned** means it does not exist yet; see [product direction](/product-direction).

## Observe

| Capability | Status | Notes |
| --- | --- | --- |
| [Cully terminal](/terminal) for Claude Code, Codex, Cursor CLI and any executable | Shipped | macOS and Linux. Not available on Windows. |
| Agent health bar | Shipped | Agent, project and branch, time, context, loops, unchecked edits. Segments with no data are omitted. |
| Tool events from hooks | Shipped | Claude Code and Codex hooks, and Cursor shell, file-edit and MCP hooks. Hooks do nothing outside the Cully terminal. |
| Cursor shell exit codes | Partial | Cursor's shell hook reports no exit code, so a failed Cursor command is not marked failed. Loop detection is weaker for Cursor. |
| Native footer (model, tokens, quota) | Partial | Codex only. |
| Context pressure | Partial | Codex and Claude Code. Other agents show none. |
| Cost and cache values | Planned | Shown only if an agent reports them. |

## Understand

| Capability | Status | Notes |
| --- | --- | --- |
| [Session journal](/session-intelligence#the-session-journal) | Shipped | Local, bounded, no contents. See [privacy](/privacy). |
| [Loop detection](/session-intelligence#loop-detection) | Shipped | Rule based. Reports what repeated, never a cause. |
| Verification state | Shipped | Whether a check passed since the last edit. |
| [Session health](/session-intelligence#cully-status) and a rule-based risk label | Shipped | The label names its reason. It is not a score. |
| Session task and progress | Partial | The task comes from `cully task` and the linked session record; a progress percentage is still planned because the journal has no measured source for it. |

## Remember

| Capability | Status | Notes |
| --- | --- | --- |
| [Project memory](/memory) through MCP | Shipped | PostgreSQL source records, Mem0 semantic recall, owner-scoped. |
| `cully_context` bounded recall | Shipped | Up to five short previews. |
| Continuity hooks | Shipped | Claude Code and Codex start hooks; Claude Code and Cursor turn-end reminders. |
| Shared learning across a team | Planned | Not available yet. |

## Advise

| Capability | Status | Notes |
| --- | --- | --- |
| [Advisor](/advisor) with context, search, failure and verification warnings | Shipped | Rule warnings work offline. |
| Model-backed advisor analysis | Partial | Uses the agent's own CLI in a headless run. Needs that CLI and its Cully MCP connection. |
| Clickable advisor in the terminal | Shipped | Click the panel, or `Ctrl+]` / `F6`. |
| Project workflow hint | Shipped | After three earlier sessions with edits where checks usually followed, it warns when the current session has unchecked edits. |
| Guardrails that block or approve actions | Planned | |

## Continue and review

| Capability | Status | Notes |
| --- | --- | --- |
| `cully timeline` | Shipped | Grouped history of a session. |
| `cully replay` | Shipped | Step player, file activity, git reconcile, JSON and offline HTML export. File paths come from structured file tools. The interactive view and the HTML page have been exercised in a pseudo-terminal and a stubbed DOM, not yet in a range of terminals and browsers. |
| `cully handoff` | Shipped | A structured handoff you can print or start another agent with. |
| `cully rescue` | Shipped | Evidence and recovery steps; an optional headless advisor adds a probable cause. |
| `cully status` session health | Shipped | |
| [Session forensics](/use-cases) | Shipped | Flight-recorder postmortems, audit-grade session review and replayable playbooks, composed of timeline, replay, reconcile and handoff. |

## Improve

| Capability | Status | Notes |
| --- | --- | --- |
| Workflow hint from past sessions | Shipped | Narrow on purpose. |
| Learned team workflow | Planned | |
| Agent performance comparison | Planned | Needs measured outcomes first. See [product direction](/product-direction#later). |

## What Cully does not claim

- It has not measured token or time savings. A benchmark is on the [roadmap](/product-direction#next).
- It does not read transcripts or send them anywhere.
- It does not see inside an agent's reasoning. It sees tool calls and results.
- A missing signal is shown as missing, never guessed.
