# Changelog

## Unreleased

## 0.9.1

- Run a single end-to-end CI check that follows the documented path: install, `cully setup`, MCP use and uninstall, replacing the separate compose-only job.
- Describe Cully in the docs introduction as a companion for work and everyday life, not only session resume and guidance.

## 0.9.0

- `cully setup` now runs the local stack without OAuth on port 3393 for Cully MCP and fixed uncommon loopback ports for the data API, Mem0 and both databases, picks the next free port when one is busy, saves the choice in `.env`, and keeps ports of an already running stack.

## 0.8.3

- Preserve richer Cully MCP counts when an older running wrapper rewrites its earlier state schema during an upgrade, using a private thread-scoped counter snapshot.
- Add an opt-in live check confirming an advisor's actual semantic Cully call reaches its linked session counter without increasing foreground activity.

## 0.8.2

- Count every observed Cully MCP tool by name across foreground, advisor and startup calls, with saved combined and per-source totals restored on resume.
- Show all Cully tool counts in expanded session instruments and distinguish semantic context recall from direct recall calls.
- Preserve earlier category totals during telemetry upgrades, without inventing names for previously grouped Other calls.

## 0.8.1

- Reduce panel row spacing: adjacent metric rows, one blank row after section titles and between major sections; keep the three-column gap.
- Restore saved same-thread instruments before the first panel frame when resuming an explicit Codex thread ID, while the resume picker loads them on SessionStart.
- Check Cully MCP through the coding agent's configured identity after opening a thread; only actual response hooks update health/authentication, without adding probe calls to foreground counters.

## 0.8.0

- Organize the Codex pane into Model & activity, Project & usage, and Cully MCP groups, with aligned columns on wide screens and consistent spacing when stacked.
- Add a small uniform gap between wide-panel metric rows and a larger break before the advisor, shedding optional spacing first on short screens.
- Count directly observed Cully MCP calls per session, including log/context/recall/search/get, and show last-observed health/authentication with explicit unknown and failure states.
- Show verified MCP health as a green Healthy or red Unhealthy heading badge, with a clear awaiting-response startup state and no duplicate Health row.
- Save private thread-scoped MCP and tool totals, verification state and native model/context/token/quota instruments locally; restore them when resuming the same Codex thread and keep different threads isolated.
- Keep the last observed native token totals when a narrow footer omits them, and replace cumulative totals without double counting.
- Omit Codex cost/cache placeholders when reliable billable cost is unavailable; retain model, context and token usage.

## 0.7.0

- Make the Codex pane's row spacing consistent within each section, with a single separator row between identity, instruments, advice and controls.
- Require a changelog update in every PR through repository instructions, a PR checklist and a dedicated CI check.
- Add a compact full-width Codex pane with two prioritized comments, suggestion/tip counts and an interactive advisor opened by mouse or Ctrl+] / F6.
- Preview and accept advice with mouse/arrow selection and Enter; apply supported shared instruction/skill changes locally or hand task actions to the current coding input without discarding its draft.
- Share owner-scoped memory recall and targeted research across Claude, Codex and Cursor advisor workers, with isolated native CLI adapters and clear startup/source status.
- Add an evidence-based red Messy phase, terminal-aware command guidance, native-background styling and adaptive group spacing while retaining full session instruments.
- Release Codex sessions and process descendants when the wrapper exits or loses its terminal.

## 0.6.1

- Bind Codex pane signals to their SessionStart session when hooks run through a persistent app server, without saving prompts or tool output.
- Document the distinct Claude, Codex and Cursor surfaces, live signal checks, and local CLI setup with a remote MCP/data/Mem0 stack.

## 0.6.0

- Replace `cully agent setup` with one `cully setup` command for local services, detected agent integrations and the advisor; use `--mcp-url` for existing servers.
- Log startup stages and return errors when the advisor or another component fails; keep `--prepare` configuration-only.
- Install project hooks in the caller's directory, and have the installer direct local users to the complete setup command.

## Earlier changes

- Accept GitHub source archives with global PAX headers so the published CLI can download and start its matching self-hosted stack.
- Clarify that `cully setup` prepares configuration and starts the full local stack; `--prepare` stops before starting services for team configuration.
- Add `cully uninstall` for local Docker and agent cleanup, with explicit `--purge-data` for memory deletion.
- Combine local agent guidance and shared memory as Cully.
- Rename the local command, agent integrations, MCP tools and configuration to Cully.
- Port the MCP and private memory data services from Python to Go.
- Add self-hosted Mem0 semantic recall with transactional indexing retries, source hydration and deletion propagation.
- Store authoritative records in plain PostgreSQL with owner isolation, full-text search and IST timestamps; use Mem0 for semantic recall.
- Add an explicit fresh-database schema, VM cutover documentation, separate CI checks, database E2E and server container builds.
- Add separate website/docs delivery and a gated personal-service release workflow.
