# Cully development

- One Go module, with CLI, MCP and private data API entry points.
- Keep memory validation and authenticated owner scoping shared across transports.
- Preserve user-owned agent configuration; apply managed changes explicitly.
- `cully setup` owns local services, agent integrations and advisor startup; use `--mcp-url` for an existing server. Keep setup scripts, installer and docs on this single command.
- Run Go formatting, vet and race tests before publishing changes.
- Keep credentials, transcripts and personal memory out of Git.
- See docs/architecture.md for the current system design.
