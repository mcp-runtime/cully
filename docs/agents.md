---
title: Connect an agent
description: Connect Claude Code, Codex or Cursor to Cully and its memory tools.
---

# Connect an agent

The [quickstart](/quickstart) starts Cully and connects your first agent. To connect another agent or use an existing Cully server, give that agent the MCP URL printed by setup.

<InstallCommand template="cully setup --agent {agent} --mcp-url http://127.0.0.1:8080/mcp" />

Pick your agent and use the URL printed by your server. Restart your agent after setup. In Codex, review and trust the Cully hooks in `/hooks` when prompted. For an OAuth-enabled HTTPS server, append `--oauth` and sign in using the steps below.

| Agent | Local Cully controls | OAuth sign-in, when enabled |
| --- | --- | --- |
| Claude Code | Status line and `/cully suggestions` | Use `/mcp`. |
| Codex | Run `cully codex` for a live advisor pane, or use `/prompts:cully suggestions`. | Run `codex mcp login cully`. |
| Cursor | Project `/cully suggestions` or ask Cursor to check suggestions | Use Cursor's MCP settings. |

The [local advisor guide](/advisor) shows what each integration can display. The [memory guide](/memory) explains what the MCP tools save and retrieve.

### Claude Code

Run `cully setup --agent claude --mcp-url URL`, using the URL printed by your private stack or supplied by your team. Restart Claude Code. Cully installs a status line, session and stop hooks, a `/cully` command and a skill. If the server uses OAuth, add `--oauth` to setup and use `/mcp` to sign in.

### Codex

Run `cully setup --agent codex --mcp-url URL`, then restart Codex. Cully installs a session-start hook, an asynchronous tool-count hook, a `/prompts:cully` prompt, managed `AGENTS.md` guidance and the skill. Codex's native footer settings remain yours. Review Cully's hooks in `/hooks` when Codex asks you to trust them. With an OAuth server, add `--oauth` and run `codex mcp login cully`.

For the single live Cully display in Codex, start `cully codex` (pass any normal Codex options after `codex`). Cully places Codex in the upper half and suggestions in the lower half. The launcher needs an interactive terminal and uses a full-screen view; terminal windows shorter than 16 rows give Codex the whole screen. Setup also installs an asynchronous `PostToolUse` hook that counts searches, edits, checks and failures while this pane is open. It saves only counters, not prompts, commands, tool output or transcripts. Codex's own footer settings remain under your control.

### Cursor

Run `cully setup --agent cursor --mcp-url URL`, then restart Cursor. Cully installs local continuity hooks, a project `/cully` command and the skill. With an OAuth server, add `--oauth` and sign in from Cursor's MCP settings. User-level hooks installed on your laptop do not run in Cursor cloud agents; connected cloud agents can still use the Cully MCP tools.

## If you already installed the advisor

Use `cully mcp add --agent codex --url URL` to add only the MCP connection. Add `--oauth` when the server requires sign-in. If you installed Cully before continuity hooks were available, rerun `cully setup --agent codex` (with your agent name and `--mcp-url URL` if using an existing server) to refresh the skill and hooks. Setup preserves unrelated agent configuration and does not replace a Cully connection that points at another URL. Run `cully status` to inspect the local integration.

Running `cully setup` starts the standard local memory stack and advisor, and installs integrations for detected agents. Use `--agent AGENT` to select one or `--agent all` for all supported agents. With `--mcp-url URL`, setup connects to that existing server and starts the local advisor without starting Docker services. Setup does not sign you into an OAuth server automatically. For server requirements, see [self-hosting](/hosting) and [team deployment](/team-deployment).

For detailed client configuration, see the [client reference](https://github.com/mcp-runtime/cully/blob/main/clients/README.md).
