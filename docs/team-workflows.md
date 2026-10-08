---
title: Work across people and agents
description: What Cully supports today and the proposed workflow for shared project work.
---

# Work across people and agents

Cully already helps one person continue work across coding agents. A team deployment gives each signed-in person private memory on a shared service. The next step is shared project coordination: a task can pass between people and agents while its outcome, evidence and lessons stay connected.

## What works today

| Capability | Current behavior |
| --- | --- |
| Run an agent | `cully run AGENT` provides the terminal and supported session signals. |
| Track a session/task link | `cully_session` stores an owner-scoped session and optional task; `cully_session_get` retrieves it. |
| Save useful work | `cully_log` records decisions, attempts, outcomes and next steps. |
| Continue with another agent | Your agents can retrieve your authorized notes through `cully_context` and `cully_get`. |
| Inspect an interrupted session | `cully replay`, `cully handoff` and `cully rescue` use the local session journal. |
| Host for multiple users | OAuth identifies each user's private records. The `company` label does not share records with coworkers. |

Task links are not yet a shared task board, dependency graph or assignment system. A local handoff does not automatically publish journal data to teammates. Consult [memory](/memory), [session intelligence](/session-intelligence) and [team deployment](/team-deployment) for the shipped behavior.

## The proposed team workflow

::: warning Design, not available yet
Shared tasks, cross-member handoffs, review packets and team playbooks below are proposals. The [team workspace design](/team-workspace) defines implementation stages and access rules.
:::

1. A lead creates a project task with a clear outcome and acceptance criteria.
2. A member claims it, prepares authorized project context and starts their preferred agent.
3. Cully links the attempt to the task and records a bounded checkpoint when work stops or needs a decision.
4. Another authorized member resumes with that checkpoint in a different agent. The original human owner remains accountable until ownership is explicitly changed.
5. A reviewer sees the result, artifact revision, check evidence and missing criteria before accepting delivery.
6. The author chooses whether to share a reusable lesson. A maintainer can adopt it into a project playbook for later tasks.

For example, one teammate investigates a production authentication failure, another implements the fix, and a reviewer accepts the verified PR. A later task retrieves the approved lesson without exposing anyone's private notes or transcripts.

## Where to start

Use [quickstart](/quickstart) for a private installation, or [team deployment](/team-deployment) for shared hosting with separate user identities. Review [team workspace design](/team-workspace) for project coordination and [shared learning design](/shared-learning) for publication and playbooks. The [roadmap](/roadmap) separates current work from proposed stages.
