---
title: Shared learning design
description: Proposed private drafting, explicit team sharing and maintained project playbooks.
---

# Shared learning design

::: warning Proposed, not shipped
Current records are owner-scoped. Shared visibility, team retrieval, playbooks and a learning worker are not available yet. This is one part of the [team workspace design](/team-workspace), not a separate replacement for task management.
:::

## Goal

Turn a useful lesson from one task into guidance for the next authorized teammate, with its evidence, author and limits intact. For example, a developer shares why an OAuth resource mismatch broke sign-in; another agent sees that lesson when preparing an authentication task.

Memory stores authored records. The advisor interprets session signals. The proposed learning system prepares relevant lesson candidates and playbook suggestions. It does not treat similarity or repeated advice as proof that an approach works.

## Draft privately, publish explicitly

1. The foreground agent can draft a lesson from a completed task: what worked, when it applies, evidence and limitations.
2. The draft belongs to the author and is private. A background job may propose a draft or sharing suggestion; it cannot publish it.
3. The author reviews the exact content and selects an authorized project or team. The server checks publication permission before committing visibility.
4. Another member's task preparation retrieves a bounded set of currently authorized lessons, with author, revision and source links.
5. A project maintainer can promote a reviewed lesson into a versioned playbook. Adoption into a skill or instruction file is explicit and previewable.

A user's opt-in to background analysis is separate from permission to share a note or change project guidance. Existing recognition of secret patterns remains useful but does not guarantee a note is safe to publish.

## Records and playbooks

| Record | Required information |
| --- | --- |
| Learning | Stable ID, author, lesson, applies-when text, limitations, visibility, boundary ID, revision, source task/artifact links and publication state. |
| Playbook | Maintainer, applicability, steps, supporting learning IDs/revisions, status and version. |
| Suggestion | Recipient/project scope, source revisions, reason offered, freshness, acceptance/dismissal feedback and expiry. |

Private, project and team visibility are enforced server-side. Publishing into one project does not publish to every project in the team. Source links must be authorized too; a shared summary must not reveal the existence or title of an inaccessible private task.

Keep PostgreSQL authoritative. Mem0 supplies candidates which are hydrated from live authorized records before use. Grouped strategies retain source IDs/revisions; withdrawn sources invalidate affected derived suggestions rather than leaving a detached copy circulating as current advice.

## Background processing without new credential exposure

MVP drafting and durable writes stay with the foreground agent through its configured MCP identity. Start with explicit retrieval in task preparation and session hooks; do not require another headless model call on every session start or stop.

An optional local learning queue may later reuse agent adapters for read-only preparation. It must be opt-in, coalesce duplicate jobs and define job IDs, deadlines, concurrency, bounded retries and cancellation. It must honor agent capability restrictions and never publish notes or write shared instructions. The daemon does not collect OAuth tokens or raw transcripts; adapters use the host client's configured authorized connection.

A server worker can group already-published records within their authorized boundary. It cannot inspect private member notes to compare teammates. Personalized comparison happens under the requesting user's authorization and returns suggestions to that user. It does not disclose private usage patterns to coworkers.

## Withdrawal and stale advice

An author can edit, unshare or delete a lesson. A maintainer can archive an adopted playbook; removing a source marks dependent guidance for review. Use a transaction to update source state and queue projection/derived-record invalidation. Retrieval enforces current visibility immediately, even if asynchronous cleanup is pending.

Managed suggestions carry source revision and a short expiry. Revalidate membership and source validity before opening full records or applying a tip. Expired or unverifiable team tips may not be applied offline; immediate local advisor warnings remain available. Membership removal denies subsequent server reads and invalidates managed caches at their next synchronization/expiry. Cully cannot erase information already read or manually exported, and must say so.

## Trust and relevance

- Shared lessons are reference data, not executable commands or privileged instructions. Tool restrictions and project policy do not come from a retrieved note.
- Present author, source revision, evidence and limitations. Show whether evidence was reported by an agent or independently verified.
- Return at most three short tips by default, matched to project/task context and current access. A member can dismiss or mute them.
- Deduplicate similar notes without overwriting authorship. Rank using relevance and explicit usefulness feedback; repeated publication alone is not quality evidence.
- Give maintainers a review queue for outdated or disputed playbooks. Learning authors cannot silently overwrite project-wide guidance.

## Delivery and verification

Ship after the team/project authorization foundation. First implement private drafts and explicit publish/unshare, then authorized lookup, then maintained playbooks. Add ranking and background grouping only when retrieval and invalidation are reliable.

Required cases include non-member direct reads, cross-team/project IDs on publication, private sources in derived tips, stale Mem0 candidates, withdrawal before application, role removal, deleted evidence and queued jobs executing after access changes. Prove that publishing or adopting requires the configured permission, and that retrying a job does not duplicate publication.

Measure lesson usefulness, handoff preparation time and repeated investigation with explicit cohort counts. Acceptance feedback improves suggestions; it does not become an individual employee score.
