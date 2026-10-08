---
title: Product direction
description: Where Cully is going. Shipped capabilities are separated from proposals so nothing planned reads as available.
---

# Product direction

Cully is the intelligent workspace around your coding agents. The long-term idea is that Cully understands **how coding agents are working**, not only what they remember.

This page separates **shipped** work from **proposals**. Proposals are not promises and carry no dates. For the exact list of what exists today, see [current capabilities](/capabilities).

```text
Terminal and hooks ──► Session journal ──► Session intelligence ──► Advisor and review
 (observe)              (understand)         (decide)                 (advise)
                                                  │
                                                  └──► Project memory (remember) ──► Improve
```

## Shipped

| Capability | What it gives you |
| --- | --- |
| [One terminal for every agent](/terminal) | Claude Code, Codex, Cursor CLI and any executable share one wrapper |
| Agent health bar | Agent, project, time, context, loops and unchecked edits at a glance |
| [Session journal](/session-intelligence#the-session-journal) | A private, coarse record every other feature reads |
| [Loop detection](/session-intelligence#loop-detection) | "The same command failed 3 times after edits" |
| `cully replay` and `cully timeline` | What the agent did, step by step, and which files it touched |
| `cully handoff` | A structured handoff for the next agent |
| `cully rescue` | Evidence and recovery steps when a session is stuck |
| `cully status` | Measured session health and a rule-based risk label |
| [Session forensics](/use-cases) | Flight-recorder postmortems, audit-grade review and replayable playbooks from the same record |
| Project workflow hint | "You usually run checks after editing here" |

## Next

Work that builds on the journal and needs moderate additions.

### Guardrails

| | |
| --- | --- |
| User problem | An agent can run a destructive command or touch a protected path before you notice. |
| Experience | A short rule file in the project. When a tool call matches, Cully warns in the health bar, and for agents with pre-tool hooks it can ask before the call runs. |
| Reuses | The hook path, the journal, the advisor panel. |
| Missing | Pre-tool hooks for each agent, a rule format, an approval prompt in the terminal. |
| Complexity | Medium. |
| Differentiation | High. Observation becomes control. |

### Cully Bench

| | |
| --- | --- |
| User problem | Cully cannot yet say whether it saves time or tokens. |
| Experience | A published benchmark of multi-session tasks with and without Cully. |
| Reuses | The journal for time to first useful action, repeated searches and repeated failures. |
| Missing | A task suite, a harness, a way to score continuation correctness. |
| Complexity | Medium to high. |
| Differentiation | Credibility. Replaces "ship faster" with numbers. |

### A lighter local install

| | |
| --- | --- |
| User problem | Trying Cully starts PostgreSQL, Mem0, a data API and MCP in Docker. |
| Experience | A single-binary mode that needs no Docker for the terminal, journal, replay, handoff and rescue, with memory added when you want it. |
| Reuses | Everything in the terminal and journal already runs without the memory stack. |
| Missing | A lightweight memory store, or a clear "terminal only" install path. |
| Complexity | Medium. |
| Differentiation | Lowers the first-run cost. |

### Session report

| | |
| --- | --- |
| User problem | After a long session you want a summary for a pull request. |
| Experience | `cully report` renders the journal as a short review note: what changed, what was verified, what is open. |
| Reuses | Replay, handoff, health. |
| Missing | A format and a place to attach it. |
| Complexity | Low to medium. |
| Differentiation | Medium. |

## Team workspace proposal

The [team workspace design](/team-workspace) connects shared project/task state, portable cross-agent handoffs, evidence-backed review and maintained team playbooks. It includes research references, access boundaries, staged acceptance gates and a commercial validation plan. These are proposals; current sessions and task links remain owner-scoped. This direction adds coordination around existing agents while preserving the principle that Cully does not intercept model traffic.

## Later

Ideas that need new data, new instrumentation or a lot of history first.

### Learned team workflow

| | |
| --- | --- |
| User problem | "We always run a security review after authorization changes" lives only in people's heads. |
| Experience | When a session type usually ends with certain steps and some are missing, Cully says so. |
| Reuses | The journal, project memory, the existing workflow hint. |
| Missing | Classifying what kind of work a session was, without reading prompts. Team-level storage. |
| Complexity | High. |
| Differentiation | High, if it is accurate. |

### Agent performance intelligence

| | |
| --- | --- |
| User problem | Which agent should take this task? |
| Experience | A comparison across work types from your own history. |
| Reuses | The journal records which agent did what. |
| Missing | Measured outcomes: task completion, rework, human intervention and test results, over hundreds of sessions. |
| Complexity | High. |
| Differentiation | High. |
| Principle | No arbitrary scores. Any comparison must come from signals Cully measures, with the sample size shown. |

### An agent cockpit

| | |
| --- | --- |
| User problem | Several agents run at once and you lose track. |
| Experience | One view of every running session, its health and its open risks. |
| Reuses | The terminal, the health bar, the journal. |
| Missing | A multi-session view, in the terminal or the browser. |
| Complexity | High. |
| Differentiation | Others are building cockpits. Cully's edge would be understanding the work, not the panes. |

## Principles

- **Do not make a commodity feature the identity.** Memory alone, a terminal alone, an MCP server alone or an advisor alone is not Cully. The combination is.
- **Measure before you claim.** Anything shown as a number comes from a signal Cully has.
- **Stay around the agent.** Cully is not another coding agent and does not intercept your agent's model traffic.
- **Keep it private by default.** New data must justify itself in [privacy](/privacy).
