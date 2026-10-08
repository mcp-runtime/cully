---
title: Solo mode
description: A copilot for your coding agent and work environment, with session health, advice, replay, handoff and private memory.
---

# Solo mode

**A copilot for your coding agent and work environment.**

Run your preferred agent inside Cully. It helps you notice repeated failures, context pressure and unchecked edits, recover useful context, understand what happened and continue in another agent. Your journal stays local and your saved memory stays owner-scoped.

```sh
cully setup --all
cully run codex       # or claude, cursor, another executable
```

Use `cully status` for observed health, `cully replay` for session activity, `cully rescue` when stuck, and `cully handoff --print` for continuation. Hooks provide supported signals; missing values remain unavailable. Local rule warnings work offline. Durable memory and model-backed advice need their configured services.

For an existing memory server, use `cully setup --mcp-url URL` and add `--oauth` if it requires sign-in. A shared server does not make your private notes visible to other people. `personal` and `company` are private organizational labels.

Start with [quickstart](/quickstart), [terminal](/terminal), [advisor](/advisor), [session intelligence](/session-intelligence) and [memory](/memory). For shared ownership, handoffs and review, use [Team mode](/team-workflows). Team mode adds authorized project records without importing your journal or notes. See [product vision](/product-direction).
