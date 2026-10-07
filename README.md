<p align="center">
  <img src="assets/brand/cully-logo.png" alt="Cully" width="420">
</p>

# Cully

**Your companion for better work and everyday life.**

[Website](https://cully.net) · [Documentation](https://docs.cully.net)

Cully remembers what you do and how you work. It helps you guide coding agents, manage projects, and spot ways to improve your workflow. When you ask it to remember personal things, it can help with your goals, routines and everyday life too.

| Component | What it does |
| --- | --- |
| `cully` | Local advisor, agent setup, session status and suggestions |
| `cully-mcp` | Shared memory tools; private single-owner mode by default, optional OAuth |
| `cully-data` | Private PostgreSQL API and Mem0 projection worker |

Works with Claude Code, Codex and Cursor. Installed hooks ask the connected agent to find a few short work notes when a task starts and save a concise summary when substantive work finishes, so another session or agent can continue from it. The standard self-hosted stack runs PostgreSQL for authoritative records and Mem0 for semantic indexing and recall. See [how Cully keeps sessions focused](https://docs.cully.net/session-optimization). Companies can also [deploy Cully with MCP Runtime](https://docs.cully.net/team-deployment#example-run-cully-with-mcp-runtime).

## Get started

```sh
curl -fsSL https://cully.net/install.sh | sh -s -- --agent codex
```

Open a new terminal, then run `cully setup` to start the local memory stack and advisor and connect detected coding agents. Use `--agent codex` to select Codex. Setup logs each stage and reports completion only after all components are ready. See the [quickstart](https://docs.cully.net/quickstart) for prerequisites and other agents.

Licensed under [Apache 2.0](LICENSE).
