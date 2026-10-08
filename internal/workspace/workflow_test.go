package workspace

import (
	"errors"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Team, *Project, *Task, time.Time) {
	t.Helper()
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	team, _ := NewTeam("lead", "Example")
	team.Members["dev"] = "member"
	team.Members["next"] = "member"
	team.Members["viewer"] = "member"
	team.Members["unassigned"] = "member"
	out, _, err := team.Apply("lead", Input{Action: "project_create", TeamID: team.ID, Name: "Cully", Repository: "https://github.com/mcp-runtime/cully"}, now)
	if err != nil {
		t.Fatal(err)
	}
	p := out.Project
	p.Members["dev"] = "member"
	p.Members["next"] = "member"
	p.Members["viewer"] = "viewer"
	out, _, err = team.Apply("lead", Input{Action: "task_create", TeamID: team.ID, ProjectID: p.ID, Name: "Fix sign-in", Criteria: []string{"Valid users can sign in", "Wrong issuer is denied"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	return team, p, out.Task, now
}
func apply(t *testing.T, team *Team, p *Project, actor string, v Input, now time.Time) Result {
	t.Helper()
	v.TeamID = team.ID
	v.ProjectID = p.ID
	out, event, err := team.Apply(actor, v, now)
	if err != nil {
		t.Fatal(err)
	}
	if v.Write() && event == nil {
		t.Fatal("mutation missing audit event")
	}
	return out
}
func TestHandoffReviewAndLearning(t *testing.T) {
	team, p, task, now := fixture(t)
	apply(t, team, p, "dev", Input{Action: "task_claim", TaskID: task.ID, Version: 1, Agent: "claude"}, now)
	apply(t, team, p, "dev", Input{Action: "task_checkpoint", TaskID: task.ID, Version: 2, Checkpoint: "Issuer mismatch found; patch on branch", NextStep: "Add regression checks", Branch: "fix/auth"}, now)
	apply(t, team, p, "dev", Input{Action: "task_release", TaskID: task.ID, Version: 3}, now)
	packet := apply(t, team, p, "next", Input{Action: "task_get", TaskID: task.ID}, now)
	if packet.Task.Checkpoint == "" || packet.Task.NextStep == "" || packet.Task.Version != 4 {
		t.Fatal("handoff lost")
	}
	apply(t, team, p, "next", Input{Action: "task_claim", TaskID: task.ID, Version: 4, Agent: "codex"}, now)
	evidence := []Evidence{{Criterion: 0, Check: "Sign-in E2E", Status: "pass", Revision: "commit-a", ObservedAt: now, Source: "independently-verified"}, {Criterion: 1, Check: "Wrong issuer test", Status: "pass", Revision: "commit-a", ObservedAt: now}}
	apply(t, team, p, "next", Input{Action: "task_submit", TaskID: task.ID, Version: 5, Revision: "commit-a", Artifact: "https://github.com/mcp-runtime/cully/pull/19", Evidence: evidence}, now)
	if task.State != "review" || task.Evidence[0].Source != "reported" {
		t.Fatal("submission must be reported evidence awaiting review")
	}
	apply(t, team, p, "lead", Input{Action: "task_approve", TaskID: task.ID, Version: 6, Revision: "commit-a"}, now)
	if task.State != "done" || task.ApprovedBy != "lead" || task.ApprovedVersion != 7 || len(task.Attempts) != 2 {
		t.Fatal("delivery state lost")
	}
	l := apply(t, team, p, "next", Input{Action: "learning_draft", TaskID: task.ID, Lesson: "Pin issuer to configured provider", AppliesWhen: "OAuth validation", Limitations: "Tested with one issuer"}, now).Learning
	if len(apply(t, team, p, "dev", Input{Action: "lessons"}, now).Learnings) != 0 {
		t.Fatal("private draft leaked")
	}
	apply(t, team, p, "next", Input{Action: "learning_publish", LearningID: l.ID, Version: 1}, now)
	apply(t, team, p, "lead", Input{Action: "playbook_adopt", LearningID: l.ID, Version: 2, Steps: []string{"Verify configured issuer", "Run negative issuer test"}}, now)
	if len(apply(t, team, p, "dev", Input{Action: "lessons"}, now).Playbooks) != 1 {
		t.Fatal("adopted guidance missing")
	}
	apply(t, team, p, "next", Input{Action: "learning_unshare", LearningID: l.ID, Version: 2}, now)
	out := apply(t, team, p, "dev", Input{Action: "lessons"}, now)
	if len(out.Learnings) != 0 || len(out.Playbooks) != 0 {
		t.Fatal("withdrawn source or guidance leaked")
	}
}
func TestAuthorizationAndReviewFailures(t *testing.T) {
	team, p, task, now := fixture(t)
	for _, actor := range []string{"outsider", "unassigned", "viewer"} {
		_, _, err := team.Apply(actor, Input{Action: "task_claim", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Version: 1, Agent: "codex"}, now)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("%s gained write access: %v", actor, err)
		}
	}
	apply(t, team, p, "dev", Input{Action: "task_claim", TaskID: task.ID, Version: 1, Agent: "codex"}, now)
	base := Input{Action: "task_submit", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Version: 2, Revision: "r1", Artifact: "https://example.test/pr/1", Evidence: []Evidence{{Criterion: 0, Check: "test", Status: "pass", Revision: "r1", ObservedAt: now}}}
	apply(t, team, p, "dev", base, now)
	if task.State != "review" {
		t.Fatal("incomplete packet must be visible to reviewer")
	}
	if _, _, err := team.Apply("lead", Input{Action: "task_approve", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Version: 3, Revision: "r1"}, now); !errors.Is(err, ErrInvalid) {
		t.Fatal("missing evidence allowed at acceptance")
	}
	base.Version = 3
	base.Evidence = append(base.Evidence, Evidence{Criterion: 1, Check: "test", Status: "pass", Revision: "r1", ObservedAt: now})
	base.Evidence[1].Status = "fail"
	apply(t, team, p, "dev", base, now)
	if _, _, err := team.Apply("lead", Input{Action: "task_approve", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Version: 4, Revision: "r1"}, now); !errors.Is(err, ErrInvalid) {
		t.Fatal("failed check allowed at acceptance")
	}
	base.Version = 4
	base.Evidence[1].Status = "pass"
	apply(t, team, p, "dev", base, now)
	p.Members["dev"] = "maintainer"
	if _, _, err := team.Apply("dev", Input{Action: "task_approve", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Version: 5, Revision: "r1"}, now); !errors.Is(err, ErrForbidden) {
		t.Fatal("self approval allowed")
	}
	for _, v := range []Input{{Version: 2, Revision: "r1"}, {Version: 5, Revision: "r2"}} {
		v.Action = "task_approve"
		v.TeamID = team.ID
		v.ProjectID = p.ID
		v.TaskID = task.ID
		if _, _, err := team.Apply("lead", v, now); !errors.Is(err, ErrConflict) {
			t.Fatal("stale approval allowed")
		}
	}
	p.GuidanceVersion++
	if _, _, err := team.Apply("lead", Input{Action: "task_approve", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Version: 5, Revision: "r1"}, now); !errors.Is(err, ErrConflict) {
		t.Fatal("changed guidance approval allowed")
	}
	apply(t, team, p, "lead", Input{Action: "team_member", Principal: "dev", Role: "remove"}, now)
	if _, _, err := team.Apply("dev", Input{Action: "task_get", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID}, now); !errors.Is(err, ErrForbidden) {
		t.Fatal("removed member could read")
	}
}
func TestStaleClaimAndDependencies(t *testing.T) {
	team, p, task, now := fixture(t)
	dep := apply(t, team, p, "dev", Input{Action: "task_create", Name: "Release", Criteria: []string{"Merged"}, Dependencies: []string{task.ID}}, now).Task
	if _, _, err := team.Apply("dev", Input{Action: "task_claim", TeamID: team.ID, ProjectID: p.ID, TaskID: dep.ID, Version: 1, Agent: "claude"}, now); !errors.Is(err, ErrConflict) {
		t.Fatal("dependency ignored")
	}
	apply(t, team, p, "dev", Input{Action: "task_claim", TaskID: task.ID, Version: 1, Agent: "claude"}, now)
	if _, _, err := team.Apply("next", Input{Action: "task_claim", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Version: 2, Agent: "codex"}, now); !errors.Is(err, ErrConflict) {
		t.Fatal("duplicate live claim")
	}
	later := now.Add(LeaseDuration + time.Second)
	if !apply(t, team, p, "next", Input{Action: "task_get", TaskID: task.ID}, later).Stale {
		t.Fatal("disconnection not visible")
	}
	apply(t, team, p, "next", Input{Action: "task_claim", TaskID: task.ID, Version: 2, Agent: "codex"}, later)
	if task.Owner != "next" || len(task.Attempts) != 2 {
		t.Fatal("explicit reclaim missing")
	}
	if _, _, err := team.Apply("dev", Input{Action: "task_checkpoint", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Version: 3, Checkpoint: "old", NextStep: "old"}, later); !errors.Is(err, ErrForbidden) {
		t.Fatal("old attempt overwrote new one")
	}
}
func TestIssuerAndLinks(t *testing.T) {
	if Principal("issuer-a", "alice") == Principal("issuer-b", "alice") {
		t.Fatal("issuer collision")
	}
	for _, link := range []string{"file:///home/a", "http://remote.test", "https://user:pass@remote.test", "https://remote.test?token=secret"} {
		if (Input{Action: "team_create", Repository: link}).Validate() == nil {
			t.Fatalf("accepted %s", link)
		}
	}
}

func TestDispatchPreservesAccessAndAuditBoundaries(t *testing.T) {
	team, p, task, now := fixture(t)
	read := Input{Action: "task_get", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID}
	out, event, err := team.Apply("viewer", read, now)
	if err != nil || event != nil || out.Task.ID != task.ID {
		t.Fatalf("authorized read must not create an audit mutation: %v %+v", err, event)
	}
	unknown := read
	unknown.Action = "task_typo"
	if _, event, err := team.Apply("lead", unknown, now); !errors.Is(err, ErrInvalid) || event != nil || task.Version != 1 {
		t.Fatal("unknown action dispatched or changed task state")
	}
	// Team admins can manage grants but cannot read project content without a grant.
	delete(p.Members, "lead")
	p.Members["next"] = "maintainer"
	if _, _, err := team.Apply("lead", read, now); !errors.Is(err, ErrForbidden) {
		t.Fatal("team admin bypassed project content membership")
	}
	member := Input{Action: "project_member", TeamID: team.ID, ProjectID: p.ID, Principal: "dev", Role: "viewer"}
	_, event, err = team.Apply("lead", member, now)
	if err != nil || event == nil || event.ProjectID != p.ID || event.TargetID != "dev" || p.Members["dev"] != "viewer" {
		t.Fatal("membership management lost its authorized audit target")
	}
	if _, _, err := team.Apply("dev", Input{Action: "task_claim", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Version: 1, Agent: "codex"}, now); !errors.Is(err, ErrForbidden) {
		t.Fatal("viewer gained mutation access")
	}
	for _, input := range []Input{
		{Action: "team_member", TeamID: team.ID, Principal: "lead", Role: "remove"},
		{Action: "team_member", TeamID: team.ID, Principal: "next", Role: "remove"},
		{Action: "project_member", TeamID: team.ID, ProjectID: p.ID, Principal: "next", Role: "remove"},
	} {
		if _, event, err := team.Apply("lead", input, now); !errors.Is(err, ErrInvalid) || event != nil {
			t.Fatalf("removed the last responsible role: %s %v", input.Action, err)
		}
	}
	_, event, err = team.Apply("lead", Input{Action: "team_member", TeamID: team.ID, Principal: "viewer", Role: "remove"}, now)
	if err != nil || event == nil || event.ProjectID != "" || event.TargetID != "viewer" || team.Members["viewer"] != "" {
		t.Fatal("team membership mutation polluted project audit scope")
	}
}

func TestCancelledDependencyBlocksAndRejectsDeadlock(t *testing.T) {
	team, p, task, now := fixture(t)
	apply(t, team, p, "dev", Input{Action: "task_claim", TaskID: task.ID, Version: 1, Agent: "claude"}, now)
	dep := apply(t, team, p, "dev", Input{Action: "task_create", Name: "Follow-up", Criteria: []string{"Done"}, Dependencies: []string{task.ID}}, now).Task
	if _, _, err := team.Apply("dev", Input{Action: "task_state", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Version: 2, State: "cancelled"}, now); !errors.Is(err, ErrConflict) {
		t.Fatal("cancelled a dependency that still has dependents")
	}
	apply(t, team, p, "dev", Input{Action: "task_state", TaskID: dep.ID, Version: 1, State: "cancelled"}, now)
	if _, _, err := team.Apply("dev", Input{Action: "task_create", TeamID: team.ID, ProjectID: p.ID, Name: "Blocked", Criteria: []string{"Done"}, Dependencies: []string{dep.ID}}, now); !errors.Is(err, ErrInvalid) {
		t.Fatal("accepted a cancelled dependency")
	}
	if _, _, err := team.Apply("dev", Input{Action: "learning_draft", TeamID: team.ID, ProjectID: p.ID, TaskID: task.ID, Lesson: "too early", AppliesWhen: "now", Limitations: "none"}, now); !errors.Is(err, ErrConflict) {
		t.Fatal("drafted a lesson before the task was done")
	}
}
