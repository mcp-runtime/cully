---
title: Installer options
description: Choose an agent, set up PATH, pin a release or build Cully from source.
---

# Installer options

Follow the [quickstart](/quickstart) for the normal laptop setup. The installer accepts `--agent claude`, `--agent codex`, `--agent cursor` or `--agent all`; with no agent selected, it installs the CLI in `~/.local/bin`. After installation, `cully setup` detects installed agents, registers their integrations and starts the full local stack and advisor daemon.

## macOS

Install Docker Desktop, start it, then run the installer from the [quickstart](/quickstart). The installer clears the download quarantine flag on the `cully` binary. Open a new terminal afterwards so your zsh startup file is reloaded.

## Linux

Install Docker Engine with the Compose plugin and make sure your user can run `docker` without `sudo`. Run the installer from the [quickstart](/quickstart), then open a new terminal so your Bash startup file is reloaded.

Windows is not supported by the installer.

## Binary location and PATH

| Install target | CLI path |
| --- | --- |
| Claude Code | `~/.claude/bin/cully` |
| Codex | `~/.codex/bin/cully` |
| Cursor | `~/.cursor/bin/cully` |
| All agents or automatic detection | `~/.local/bin/cully` |

The installer adds the selected directory to your zsh or Bash startup file when needed. Open a new terminal before typing `cully`, or use the full path printed by the installer in your current terminal. A piped installer cannot update the shell that launched it. With a custom `CLAUDE_CONFIG_DIR`, `CODEX_HOME` or `CURSOR_CONFIG_DIR`, or another shell, add the printed directory to PATH yourself. Existing shell settings are preserved.

## Connect to a server already running

If a company runs Cully for you, use its MCP URL when installing:

<InstallCommand template="curl -fsSL https://cully.net/install.sh | sh -s -- --agent {agent} --mcp-url https://mcp.example.com/mcp --oauth" />

Use the URL and sign-in instructions your company provides. Leave off `--oauth` for a private single-user endpoint. You can add the server later with `cully setup --agent AGENT --mcp-url URL`. See [connect an agent](/agents).

## Build the CLI from source

If a prebuilt release cannot be downloaded, check GitHub release access or install from source with Go 1.26 or newer:

<InstallCommand template="curl -fsSL https://cully.net/install.sh | sh -s -- --from-source --agent {agent}" />

A source build can take several minutes while Go downloads its toolchain and dependencies. You can [inspect the installer](https://github.com/mcp-runtime/cully/blob/main/install.sh) before running it.

## If the download stalls

The default installer downloads a prebuilt CLI from GitHub Releases and shows progress. It stops a binary download after two minutes or when the connection stalls, then prints a retry message; it does not silently switch to a source build. Check access to GitHub Releases and retry, or choose `--from-source` explicitly. Set `CULLY_VERSION` to a release tag when you need a particular version; a source build otherwise uses the current main branch.

## Check and adjust the installation

Run `cully status` to see the advisor, agent integration, MCP connection and continuity hooks. Run `cully setup` for the full laptop setup, or `cully setup --agent AGENT --mcp-url URL` to install integrations and start the advisor with an existing server. If only the server connection is missing, use `cully mcp add --agent AGENT --url URL`; add `--oauth` when that server requires sign-in. Cully keeps unrelated agent settings. For client-specific sign-in and commands, see [connect an agent](/agents).
