---
title: The Cully terminal
description: Run Claude Code, Codex, Cursor or any coding agent in one terminal with a live health bar and advisor.
---

# The Cully terminal

`cully run` starts a coding agent inside a terminal that Cully owns. The agent gets a slightly smaller screen. Cully uses the rest for a health bar and the advisor, so the agent's own clear-screen and cursor codes can never erase them.

It is an observation surface, not another terminal emulator you have to learn. You use your agent exactly as before.

## Start an agent

```sh
cully run claude [ARGS...]
cully run codex  [ARGS...]
cully run cursor [ARGS...]     # starts cursor-agent
cully run AGENT  [ARGS...]     # any executable on your PATH
```

`cully claude`, `cully codex` and `cully cursor` are shortcuts. Arguments after the agent name go to the agent unchanged, for example `cully run claude --continue` or `cully run codex resume`.

An agent Cully does not know yet still runs in the terminal with the common instruments. Add a dedicated adapter later only if the agent offers more signals.

Requirements: an interactive terminal at least 20 columns by 8 rows, macOS or Linux. Exiting the wrapper stops the agent's process group. It does not create a detached tmux session.

## The health bar

The first row under the agent is a one-line health bar. Every value is measured. A value with no data is left out.

```text
── Codex │ oauth-service:main │ 27m │ Context 63% │ ⚠ Loop 3x │ ● 2 unchecked ── Ctrl+] advisor ──
```

| Segment | Meaning |
| --- | --- |
| Agent | Which agent runs here |
| `project:branch` | The project folder and Git branch |
| Time | How long the session has run |
| Context | Share of the context window used, when the agent reports a window size. Yellow from 75%, red from 90%. |
| `⚠ Loop 3x` | The journal shows the same command failing 3 times, or repeated edit and failed-check cycles. See [loop detection](/session-intelligence#loop-detection). |
| `● 2 unchecked` | Edits since the last check |

## The advisor panel

Below the health bar, the panel shows session instruments and two prioritized advisor comments. Click anywhere in the panel, or press `Ctrl+]` or `F6`, to open the full advisor. Select with the mouse or Up and Down, preview with Enter, and press Tab for every instrument. See the [advisor](/advisor) for what it analyzes and how you apply a suggestion. In Warp, open the advisor with `Ctrl+]` or `F6` first: mouse tracking is deferred so Warp scroll mode is not consumed while you work.

The panel grows only as far as its content needs and always leaves the agent at least 12 rows on a normal screen (8 on a short one). Terminals shorter than 12 rows give the agent the whole screen.

## What each agent supplies

All agents get the health bar, tool counts, check and edit state, loops, Git changes, daemon health and advice. What differs is what the agent itself reports.

| Agent | Extra signals in the terminal | How Cully gets them |
| --- | --- | --- |
| Codex | Model, reasoning effort, context, tokens, quota, fast mode | Reads the native footer rows in memory, never a transcript |
| Claude Code | Context pressure | A hook-fed snapshot. The visible Claude status line is turned off inside the terminal, so there is one display. |
| Cursor CLI | Tool events from shell, file edits and MCP calls | Cursor hooks |
| Other agents | Process and Git state only, until it has an adapter | None needed |

Cost and cache values are shown only when an agent reports them. Cully never invents them.

## What the terminal records

Hooks send each tool call to a local [session journal](/session-intelligence#the-session-journal). The journal powers loop detection, `cully timeline`, `cully replay`, `cully handoff`, `cully rescue` and `cully status`. It is stored in your Cully directory with owner-only permissions and pruned after 30 days. See [privacy](/privacy) for exactly what it holds.

The hooks are inactive outside the Cully terminal, so an ordinary agent session leaves no advisor files.
