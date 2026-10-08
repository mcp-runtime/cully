package workspace

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

func createTask(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	out := Result{TeamID: t.ID}

	if err := required(v.Name); err != nil {
		return out, "", err
	}
	if len(v.Criteria) == 0 || len(t.Tasks) >= 500 {
		return out, "", ErrInvalid
	}
	for _, criterion := range v.Criteria {
		if required(criterion) != nil {
			return out, "", ErrInvalid
		}
	}
	// Dependencies only reference existing non-cancelled tasks. They cannot
	// be edited, so a newly created task cannot introduce a dependency cycle.
	for _, id := range v.Dependencies {
		dep := t.Tasks[id]
		if dep == nil || dep.ProjectID != p.ID {
			return out, "", ErrForbidden
		}
		if dep.State == "cancelled" {
			return out, "", ErrInvalid
		}
	}
	task := &Task{ID: uuid.NewString(), ProjectID: p.ID, Name: v.Name, Criteria: v.Criteria, Dependencies: v.Dependencies, State: "ready", Version: 1, UpdatedAt: now, Attempts: []Attempt{}, Evidence: []Evidence{}}
	t.Tasks[task.ID] = task
	out.Task = task

	return out, task.ID, nil
}

// mutateTask enforces the common version and terminal-state rules once.
func mutateTask(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	task := t.Tasks[v.TaskID]
	out := Result{TeamID: t.ID, Task: task}
	if task == nil || task.ProjectID != p.ID {
		return out, "", ErrForbidden
	}
	if err := conflict(task.Version, v.Version); err != nil {
		return out, "", err
	}
	if task.State == "done" || task.State == "cancelled" {
		return out, "", ErrConflict
	}
	var err error
	switch v.Action {
	case "task_claim":
		err = claimTask(t, actor, task, v, now)
	case "task_approve":
		err = approveTask(actor, p, task, v, now)
	case "task_state":
		if v.State == "cancelled" && task.State == "ready" {
			err = cancelTask(t, task)
		} else {
			err = updateOwnedTask(t, actor, p, task, v, now)
		}
	default:
		err = updateOwnedTask(t, actor, p, task, v, now)
	}
	if err != nil {
		return out, "", err
	}
	task.Version++
	task.UpdatedAt = now
	return out, task.ID, nil
}
func claimTask(t *Team, actor string, task *Task, v Input, now time.Time) error {
	if required(v.Agent) != nil {
		return ErrInvalid
	}
	if len(task.Attempts) >= 50 {
		return ErrInvalid
	}
	if task.State != "ready" { // Explicit reclaim is allowed after expiry, never automatic.
		if (task.State == "review" && task.Owner != actor) || len(task.Attempts) == 0 || now.Before(task.Attempts[len(task.Attempts)-1].LeaseUntil) {
			return ErrConflict
		}
	}
	for _, id := range task.Dependencies {
		dep := t.Tasks[id]
		if dep == nil || dep.State != "done" {
			return ErrConflict
		}
	}
	task.Owner = actor
	task.State = "active"
	task.Evidence = nil
	task.Attempts = append(task.Attempts, Attempt{ID: uuid.NewString(), Initiator: actor, Agent: v.Agent, SessionRef: v.SessionRef, StartedAt: now, LeaseUntil: now.Add(LeaseDuration)})

	return nil
}
func cancelTask(t *Team, task *Task) error {
	for _, other := range t.Tasks {
		if other == nil || other.ID == task.ID || other.State == "cancelled" || other.State == "done" {
			continue
		}
		if slices.Contains(other.Dependencies, task.ID) {
			return ErrConflict
		}
	}
	task.Owner = ""
	task.State = "cancelled"
	task.NextStep = ""
	task.Evidence = nil
	return nil
}

func updateOwnedTask(t *Team, actor string, p *Project, task *Task, v Input, now time.Time) error {
	if task.Owner != actor {
		return ErrForbidden
	}
	if len(task.Attempts) == 0 || !now.Before(task.Attempts[len(task.Attempts)-1].LeaseUntil) {
		return ErrConflict
	}
	switch v.Action {
	case "task_checkpoint":
		if required(v.Checkpoint, v.NextStep) != nil {
			return ErrInvalid
		}
		task.Checkpoint = v.Checkpoint
		task.NextStep = v.NextStep
		task.Branch = v.Branch
		task.Attempts[len(task.Attempts)-1].SessionRef = v.SessionRef
		task.Attempts[len(task.Attempts)-1].LeaseUntil = now.Add(LeaseDuration)
		if task.State == "review" {
			task.State = "active"
			task.Evidence = nil
		}
	case "task_release":
		if required(task.Checkpoint, task.NextStep) != nil {
			return ErrInvalid
		}
		task.Owner = ""
		task.State = "ready"
		task.Evidence = nil
	case "task_state":
		if !slices.Contains([]string{"active", "blocked", "cancelled"}, v.State) || (v.State == "blocked" && required(v.NextStep) != nil) {
			return ErrInvalid
		}
		if v.State == "cancelled" {
			return cancelTask(t, task)
		}
		task.State = v.State
		task.NextStep = v.NextStep
		task.Evidence = nil
	case "task_submit":
		candidate := *task
		candidate.Revision = v.Revision
		candidate.Artifact = v.Artifact
		candidate.Evidence = append([]Evidence(nil), v.Evidence...)
		for i := range candidate.Evidence {
			candidate.Evidence[i].Source = "reported"
		}
		if err := validateEvidence(&candidate, now); err != nil {
			return err
		}
		task.Revision = candidate.Revision
		task.Artifact = candidate.Artifact
		task.Evidence = candidate.Evidence
		task.State = "review"
		task.ReviewGuidanceVersion = p.GuidanceVersion
	default:
		return ErrInvalid
	}
	return nil
}
