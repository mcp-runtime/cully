# Cully development

- One Go module, with CLI, MCP and private data API entry points.
- Keep memory validation and authenticated owner scoping shared across transports.
- Preserve user-owned agent configuration; apply managed changes explicitly.
- `cully setup` owns local services, agent integrations and advisor startup; use `--mcp-url` for an existing server. Keep setup scripts, installer and docs on this single command.
- Run Go formatting, vet and race tests before publishing changes.
- Every PR must update `CHANGELOG.md` with a concise entry under `Unreleased` describing the final user-visible change, fix, docs or maintenance work. At release time, move those entries into the versioned section and keep `Unreleased` ready for the next PR. Follow `.github/PULL_REQUEST_TEMPLATE.md`; the changelog CI check enforces the file update.
- Keep credentials, transcripts and personal memory out of Git.
- See docs/architecture.md for the current system design.

<!-- cully:codex:start -->
## Cully

- Use the shared project skill at `.cully/skills/cully/SKILL.md` for Cully session controls.
- In Codex, use /prompts:cully (or type /cully and select the saved cully prompt) for the in-session cully command.
- For each substantive task, find relevant prior work with Cully MCP and save one concise work summary before finishing. Follow the Cully skill for project identity, owner section, and private-data rules.
- Use the configured Cully MCP tools for shared personal and project memory.
<!-- cully:codex:end -->
