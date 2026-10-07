---
title: Use the local advisor
description: Check your session and review Cully suggestions inside your coding agent.
---

# Use the local advisor

Cully checks the session signals your coding agent makes available and suggests practical improvements. It can point out a missing project instruction, a useful skill or an MCP connection. Suggestions are advisory: you review a change before applying it.

`cully setup` installs the agent integrations and starts the advisor daemon along with the local memory services. After [setting up Cully](/quickstart), restart your agent and check suggestions:

| Agent | In the agent |
| --- | --- |
| Claude Code | Watch the Cully status line or run `/cully suggestions`. |
| Codex | Start `cully codex` for a live lower advisor pane, or run `/prompts:cully suggestions`. |
| Cursor | Run the project `/cully suggestions` command or ask Cursor to check Cully suggestions. |

You can always run these in a terminal:

```sh
cully status
cully suggestions
```

`cully status` shows whether the daemon is running. If startup failed, resolve the reported error and rerun `cully setup` with the same options. For an existing server, keep the `--mcp-url URL` option. Claude Code provides live hook signals and a Cully status line. Codex shows Cully advice in the optional `cully codex` pane; Cursor uses its supported command and status surfaces. The available advice may differ between agents.

| Agent | Installed local integration |
| --- | --- |
| Claude Code | Live Cully status line, session and stop hooks, `/cully` command and skill. |
| Codex | Optional `cully codex` advisor pane, session-start and asynchronous tool-count hooks, `/prompts:cully`, managed `AGENTS.md` pointer and skill. Setup leaves Codex's native footer settings to the user. |
| Cursor | Session and response hooks, project `/cully` command and skill. |

Claude Code supplies the richest live signals to the background advisor. It can identify high context pressure, repeated tool faults or searches, cost and rate pressure, and a missing verifier, then suggest a native control or a useful MCP integration. The optional Codex pane uses bounded tool counters to flag repeated failures, broad searching and edits without a check. It does not inspect prompt or tool content. Cursor still gets continuity prompts and Cully commands. The [session optimization guide](/session-optimization) explains the difference.

## Preview a suggestion

If Cully lists a numbered improvement, preview it with the number shown:

```sh
cully apply 1 --dry-run
```

Review the preview. To apply it, run `cully apply 1` and confirm the proposed change. Cully preserves unrelated agent settings and does not silently replace user-owned configuration.

## Commands

| Command | What it does |
| --- | --- |
| `cully status [directory]` | Shows session warnings and integration state for the current or specified directory. |
| `cully suggestions` | Lists improvements and informational notes. |
| `cully apply <n> --dry-run` | Previews a numbered improvement. |
| `cully apply <n>` | Applies one after confirmation. |
| `cully setup [--agent AGENT]` | Starts the local stack, installs agent integrations and starts the advisor; detects agents by default. |
| `cully setup --mcp-url URL [--agent AGENT]` | Connects an existing server, installs agent integrations and starts the advisor. |
| `cully setup --prepare` | Prepares editable stack configuration without starting services or installing integrations. |
| `cully uninstall AGENT` | Removes one agent's Cully-managed integration settings. |
| `cully uninstall` | Stops the local stack and removes managed agent integrations; keeps memory data. |
| `cully uninstall --purge-data` | Also deletes local memory volumes and self-hosted configuration. |
| `cully mcp add --agent AGENT --url URL` | Adds Cully MCP to an existing agent setup; include `--oauth` when the server uses sign-in. |
| `cully version` | Prints the CLI version. |

Cully's [memory tools](/memory) keep useful context across sessions. The installed Cully hooks ask the connected agent to retrieve and save concise work notes. The local advisor can still show session guidance if the memory server is temporarily unavailable.

Applying a suggestion can update a project instruction or skill, register an MCP integration, or run an accepted install command. Review the dry run first; Cully keeps unrelated MCP connections and user-owned settings. Hook, worker and daemon entry points are internal commands installed by Cully, not steps you need to run yourself.

The [session optimization guide](/session-optimization) shows how bounded memory previews, native agent controls and advisor suggestions work together without loading an old transcript into a new task.

## Controls

Set these environment variables only when you want to change the defaults:

| Variable | Effect |
| --- | --- |
| `CULLY_ANALYZE_DISABLE=1` | Disable advisor analysis while keeping the status line. |
| `CULLY_ANALYZE_PROMPTS=0` | Exclude recent prompt text from analyzer signals. |
| `CULLY_DISPLAY` | Use `minimal`, `full` or `debug` display. |
| `CULLY_COST_INDEX` | Use `eco`, `normal` or `perf` cost estimate. |
| `CULLY_ALERT_CHIME=1` | Ring a terminal bell at critical context pressure. |
| `CULLY_DEBUG=1` | Enable local debug logs. |

Cully honors `CLAUDE_CONFIG_DIR`, `CODEX_HOME` and `CURSOR_CONFIG_DIR` for custom agent directories. Local snapshots and diagnostics support the advisor; they are not automatically saved as memory records.
