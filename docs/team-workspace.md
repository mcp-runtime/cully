---
title: Team workspace design
description: Proposed project coordination, agent handoffs, verified delivery and shared learning for Cully teams.
---

# Team workspace design

::: warning Proposed, not shipped
This is a product and implementation proposal. Cully already wraps coding agents and stores owner-scoped sessions, task links and memory. It does not yet provide shared project permissions, a team board, delegated execution or team learning. See [current capabilities](/capabilities) for available features and [team deployment](/team-deployment) for today's multi-user setup.
:::

## The product promise

**Keep your team's agent work moving, whoever picks it up next.**

A team should be able to see what is underway, continue work with a different person or agent, verify the outcome and reuse what it learned. Cully stays around the agents people already use. It coordinates project work and its evidence without replacing the coding agent or intercepting model traffic.

The first buyer hypothesis is an engineering team using more than one coding agent across repositories. Their problem is fragmented work: sessions end, decisions disappear, teammates repeat investigations and reviewers cannot tell what was actually checked. These are hypotheses to validate with teams, not claims of proven demand or productivity gains.

## What makes the team product worth adopting

| Proposed capability | User experience | Value to validate |
| --- | --- | --- |
| Team work board and decision inbox | See active tasks, blockers, waiting reviews and decisions needed, with a human owner and freshness timestamp. | Less manual status chasing. |
| Portable task handoff | Continue a teammate's task in another agent with decisions, attempted approaches, open questions and artifact links. | Less repeated investigation and setup. |
| Evidence-backed review | Open a concise delivery packet showing acceptance criteria, commit/PR, checks and unresolved risks. | Faster review with fewer unsupported completion claims. |
| Shared, versioned playbooks | Relevant lessons become reusable guidance with author, sources, applicability and explicit adoption. | Fewer repeated mistakes and easier onboarding. |
| Conflict and stale-work warnings | Notice duplicate task claims, overlapping declared work areas and disconnected sessions before wasting effort. | Less conflicting agent work. |
| Team access and accountability | Project-scoped permissions and an event trail show who delegated, shared, approved or changed work. | A deployment the team can govern. |

The first sellable slice combines the board, handoff and review packet. Shared learning strengthens that workflow; unrestricted autonomous scheduling is a later capability.

## The work model

| Entity | Responsibility |
| --- | --- |
| Team | Members, roles, access policy and audit scope. |
| Project | Goal, stable ID, repository links, project members and versioned guidance. A repository URL is an attribute, not the project's identity. |
| Milestone | Optional delivery target and linked tasks; no invented completion percentages. |
| Task | Stable ID, outcome, acceptance criteria, human owner, dependencies, lifecycle and linked artifacts. |
| Execution attempt | One effort toward a task, with the initiating user, agent, execution host, capability set and result. |
| Session | A concrete client session attached to an attempt, with a checkpoint and supported observed signals. |
| Learning | An attributed lesson derived from work, with explicit visibility and source links. |

A task can have multiple attempts and sessions. A crashed session does not delete its task. Existing owner-scoped task entries and session links remain the current implementation; this proposal does not pretend they already supply team assignments or shared lifecycle state. During implementation, map those entries to stable task IDs and preserve authored notes separately from authoritative task state. Existing private records remain private unless their owner explicitly shares selected material.

## One task from start to finish

1. A lead creates a task such as “Fix registry authentication” with a repository and testable acceptance criteria.
2. A developer claims it and starts their preferred agent. Cully prepares bounded context from authorized project decisions, approved playbooks and prior attempts.
3. The task becomes active. Its card shows the assigned human, agent, branch, last update and observed warnings. Unsupported signals display as unavailable.
4. A blocker moves it to blocked and names the decision or dependency needed. A teammate can receive an explicit handoff rather than a copied transcript.
5. The agent proposes a result. Cully assembles an evidence packet and moves the task to review, not automatically to done.
6. The designated reviewer accepts the result when the criteria are met. A reusable lesson can be drafted for sharing and adopted into project guidance.

Proposed task states are `ready`, `active`, `blocked`, `review`, `done` and `cancelled`. Dependency cycles are rejected, and dependent tasks become ready only when their required predecessors satisfy the configured completion policy. Task state differs from execution state: an attempt may be disconnected, failed or waiting for input while the task still needs work.

## A board that helps decisions

Start with one project view and a team inbox, available through CLI and MCP; a web board can follow. Keep these views backed by the same domain operations.

- **Needs your attention:** review requests, blockers with an owner, permission requests and stale executions.
- **In progress:** task owner, delegated agent, current attempt, last acknowledged update and artifact links.
- **Ready next:** dependency-ready tasks and their acceptance criteria; assignment is explicit in the first release.
- **Delivery readiness:** remaining criteria and missing evidence for a milestone. Do not infer readiness from agent activity or a green test alone.

A disconnected client is marked stale after a configurable interval; it is not declared failed from silence. Use atomic task claims, an expiring attempt lease and version checks so two clients cannot silently overwrite the same assignment. An expired lease requires explicit reclaim, and must not silently launch another agent while the original might still be working.

## Handoff and collision prevention

A portable handoff contains the task ID, intended outcome, latest accepted checkpoint, approach tried, decisions, next action, repository/branch/worktree references, exact artifact revisions and verification gaps. Private notes can be attached only after an explicit sharing choice. The recipient receives only records they are authorized to read.

Cully does not transfer model conversations, agent credentials or a live process between laptops. Local file paths are not portable artifacts; use repository references and instructions to reconstruct the work. Record whether uncommitted work exists without uploading file contents automatically.

Task claims prevent duplicate ownership of the same attempt. Project-authorized declared file areas or branches can produce advisory overlap warnings across tasks, with a freshness timestamp. They cannot prove two agents will conflict; shell edits and unavailable hooks leave observation gaps. Isolate parallel execution in separate worktrees when supported.

## Completion with evidence

The review packet links each acceptance criterion to supporting evidence: artifact revision, check name, status, timestamp and source. Distinguish agent-reported checks from checks fetched through a configured integration or independently reproduced. A local journal's pass/fail event is useful session context; it is not a complete CI attestation.

Pin approval to the task version, relevant guidance version and artifact revision. New changes invalidate affected approval or verification. Server authorization controls state transitions; an agent cannot approve its own result just by claiming success. Projects configure whether completion requires human acceptance, merge evidence, an external tracker transition or a combination. External failures remain visible and never become successful delivery by inference.

## Delegation without losing control

Keep a human accountable for the task and record the agent separately. MVP delegation prepares a context packet and links a manually started supported session. Managed execution comes later, after adapters can prove launch, status, cancellation and checkpoint behavior.

Each managed attempt must define permissions, executable/host, wall-time limit, retry limit and concurrency allowance. Spend limits apply only where a provider exposes reliable usage and an adapter can enforce them; otherwise show spend as unavailable and offer time/concurrency limits. Cully cannot enforce arbitrary tools or token budgets merely by wrapping a terminal. Publish an adapter capability matrix and reject unsupported policy requirements.

Approvals for publishing, deploying or destructive actions must be enforced through supported execution/tool boundaries. Advisory warnings remain labeled advisory for agents without those controls. Agent identity is an execution attribution, never a substitute for server-verified user/project authorization.

## Shared learning feeds the next task

[Shared learning design](/shared-learning) describes private drafting, explicit publication, authorized retrieval, withdrawal and versioned playbooks. Lessons surface during task preparation and review, rather than becoming another feed people must read.

A playbook has applicability, steps, supporting sources, maintainer, revision and status. A project maintainer explicitly promotes an accepted lesson into guidance. Repeated similar notes are candidates for review, not proof that a strategy is correct. Agents receive source notes as reference material, not as higher-priority instructions. Personal suggestions do not silently edit project-wide instructions.

## Access and data boundaries

- Use verified OAuth identity for shared multi-user deployments, with principals bound to issuer and subject rather than a display name or email. Private no-OAuth mode remains a configured single-owner installation; it is not a team identity mechanism.
- Start with admin-managed team membership and project membership. Team admins manage membership; project maintainers manage project guidance and review policy; members access assigned projects; viewers cannot mutate tasks. Roles alone do not grant every project to every member.
- Enforce membership and capabilities in shared domain operations, including direct record reads, full-text lookup, Mem0 candidate hydration, artifacts, exports and background jobs. Reject cross-team IDs and unauthorized links on writes as well as reads.
- Represent record visibility as private, project or team with an explicit owning boundary. Organization-wide sharing is out of scope for the first slice.
- Removing membership immediately blocks new server reads/writes. Previously viewed information cannot be unlearned or recalled from arbitrary exports; managed caches expire and revalidate as described in the learning design.
- The local journal stays local by default. Team progress uses opt-in bounded checkpoints and metadata, not raw transcripts, command arguments, source files or prompts.
- Record server-side changes to membership, task assignment/state, publication, approval and execution policy. Do not claim tamper-proof audit evidence from the local journal.

## Delivery stages and acceptance gates

| Stage | Deliverable | Gate before the next stage |
| --- | --- | --- |
| 1. Team foundation | Admin-managed membership, project IDs/access, stable tasks and atomic claims, CLI/MCP task operations and server audit events. | Two teams cannot read or modify each other's records, direct IDs, artifacts or semantic results; removed members lose access; concurrent claims have one winner. |
| 2. First team workflow | Project board/inbox, session checkpoints, cross-agent handoff and review packets with manual launch. | Two people using different agents continue one task; stale state is visible; only an authorized reviewer can complete it with the configured evidence. |
| 3. Shared learning | Explicit publication, project/team lookup, source-linked tips, withdrawal and maintained playbooks. | Private notes never enter shared retrieval; edits/withdrawals invalidate derived tips and managed caches; no automatic instruction changes. |
| 4. Managed execution | Capability-aware adapters, isolated worktrees, bounded runs, cancellation, recovery and optional tracker integration. | Duplicate events do not duplicate runs; restart/cancellation and policy enforcement pass per-adapter checks; unsupported controls are explicit. |
| 5. Measured improvement | Team workflow suggestions and agent comparisons using verified outcomes. | Show cohort/sample size, missing data and confounders; validate recommendations before enabling any automatic routing. |

GitHub issue/PR links are the first artifact integration because they fit coding work. Choose whether Cully or the tracker owns each synchronized field. Signed events, delivery deduplication, reconciliation and external permission checks are required before bidirectional writes. Linear/Jira and chat notifications are later integration choices, not prerequisites for the first workflow. Project-aware onboarding is a low-cost follow-on: prepare a bounded starter packet from authorized guidance and completed tasks.

## Demo and commercial validation

Demonstrate one task: one developer starts it, another continues in a different agent, the reviewer sees missing evidence, the checks are supplied, and the accepted lesson improves a later task. Include a private note that never appears to the teammate and a membership removal that stops new reads.

Measure time to first useful action after handoff, repeated investigation, time waiting for review, reopened tasks and playbook usefulness. Compare like-for-like work with and without Cully. Show counts and denominators; no arbitrary productivity score or individual surveillance leaderboard. Record cost only when measured and consented.

The commercial hypothesis is that teams pay for coordination, review confidence and governed knowledge reuse across their existing agents. Self-hosted deployment, project controls and integration support could support a team offering. Validate willingness to adopt and pay before fixing packaging, pricing or claiming savings.

## Research and how it informs this proposal

These are primary-source product/engineering references checked on 2026-10-08. They demonstrate patterns, not market validation for Cully. The design choices above are our interpretation.

- [Linear: AI agents](https://linear.app/docs/agents-in-linear) describes delegated agents with a human assignee retaining responsibility. Cully should retain human accountability alongside agent execution.
- [Linear: agent interaction](https://linear.app/developers/agent-interaction) makes agent lifecycle and activity visible. Cully should distinguish task state from attempt/session state and surface requests for input.
- [Linear: coding sessions](https://linear.app/docs/coding-sessions) connects delegated work to a development environment and reviewable changes. Cully's opportunity is portable continuation and evidence across existing agent tools; a hosted sandbox is not an MVP prerequisite.
- [Atlassian: Rovo agent accounts](https://support.atlassian.com/studio/docs/understand-rovo-agent-accounts/) separates user-context actions from separately authorized agent accounts. Cully should make effective identity and project access explicit before enabling automation.
- [Anthropic: decoupling the brain from the hands](https://www.anthropic.com/engineering/managed-agents) separates recoverable sessions, harness logic and execution environment. Cully should keep durable work state independent of any one adapter or running process.

## Decisions for implementation

Adopt private-by-default sharing, admin-managed membership, project-scoped access, human-owned tasks and manual launch for the first team workflow. Leave automated ranking, autonomous dispatch, group-claim membership mapping and hosted sandboxes for measured follow-on work. These choices keep the first slice demonstrable without promising controls the current adapters cannot enforce.
