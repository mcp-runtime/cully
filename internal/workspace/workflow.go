package workspace

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const LeaseDuration = 30 * time.Minute

func required(values ...string) error {
	for _, s := range values {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%w: required field missing", ErrInvalid)
		}
	}
	return nil
}
func conflict(current, supplied int) error {
	if current != supplied {
		return fmt.Errorf("%w: refresh the record before retrying", ErrConflict)
	}
	return nil
}
func (t *Team) project(actor, id string, write bool) (*Project, error) {
	p := t.Projects[id]
	if t.Members[actor] == "" || p == nil || p.Members[actor] == "" || (write && p.Members[actor] == "viewer") {
		return nil, ErrForbidden
	}
	return p, nil
}
func NewTeam(actor, name string) (*Team, error) {
	if err := required(actor, name); err != nil {
		return nil, err
	}
	return &Team{ID: uuid.NewString(), Name: name, Members: map[string]string{actor: "admin"}, Projects: map[string]*Project{}, Tasks: map[string]*Task{}, Learnings: map[string]*Learning{}, Playbooks: map[string]*Playbook{}}, nil
}

// Apply is executed under the repository's team-row transaction lock. Callers
// must discard the state on error. All output is a projection authorized for
// actor; the raw aggregate is never returned by a transport.
func (t *Team) Apply(actor string, v Input, now time.Time) (Result, *Event, error) {
	out := Result{TeamID: t.ID}
	if t.Members[actor] == "" || v.TeamID != t.ID {
		return out, nil, ErrForbidden
	}
	target := t.ID
	if v.Action == "projects" {
		for _, p := range t.Projects {
			if p.Members[actor] != "" {
				out.Projects = append(out.Projects, *p)
			}
		}
		sort.Slice(out.Projects, func(i, j int) bool { return out.Projects[i].ID < out.Projects[j].ID })
		return out, nil, nil
	}
	if v.Action == "team_member" {
		if t.Members[actor] != "admin" {
			return out, nil, ErrForbidden
		}
		if err := required(v.Principal); err != nil {
			return out, nil, err
		}
		if v.Role != "admin" && v.Role != "member" && v.Role != "remove" {
			return out, nil, ErrInvalid
		}
		if t.Members[v.Principal] == "admin" && v.Role != "admin" {
			n := 0
			for _, role := range t.Members {
				if role == "admin" {
					n++
				}
			}
			if n <= 1 {
				return out, nil, fmt.Errorf("%w: keep at least one admin", ErrInvalid)
			}
		}
		if len(t.Members) >= 100 && t.Members[v.Principal] == "" {
			return out, nil, ErrInvalid
		}
		if v.Role == "remove" {
			delete(t.Members, v.Principal)
			for _, p := range t.Projects {
				delete(p.Members, v.Principal)
			}
		} else {
			t.Members[v.Principal] = v.Role
		}
		target = v.Principal
	} else if v.Action == "project_create" {
		if t.Members[actor] != "admin" {
			return out, nil, ErrForbidden
		}
		if err := required(v.Name, v.Repository); err != nil {
			return out, nil, err
		}
		if len(t.Projects) >= 50 {
			return out, nil, ErrInvalid
		}
		p := &Project{ID: uuid.NewString(), Name: v.Name, Repository: v.Repository, Members: map[string]string{actor: "maintainer"}, GuidanceVersion: 1}
		t.Projects[p.ID] = p
		out.Project = p
		target = p.ID
		v.ProjectID = p.ID
	} else {
		p, err := t.project(actor, v.ProjectID, v.Write())
		if v.Action == "project_member" && t.Members[actor] == "admin" {
			p = t.Projects[v.ProjectID]
			err = nil
			if p == nil {
				err = ErrForbidden
			}
		}
		if err != nil {
			return out, nil, err
		}
		switch v.Action {
		case "project_member":
			if p.Members[actor] != "maintainer" && t.Members[actor] != "admin" {
				return out, nil, ErrForbidden
			}
			if t.Members[v.Principal] == "" {
				return out, nil, ErrForbidden
			}
			if !slices.Contains([]string{"maintainer", "member", "viewer", "remove"}, v.Role) {
				return out, nil, ErrInvalid
			}
			if p.Members[v.Principal] == "maintainer" && v.Role != "maintainer" {
				n := 0
				for _, role := range p.Members {
					if role == "maintainer" {
						n++
					}
				}
				if n <= 1 {
					return out, nil, ErrInvalid
				}
			}
			if v.Role == "remove" {
				delete(p.Members, v.Principal)
			} else {
				p.Members[v.Principal] = v.Role
			}
			target = v.Principal
		case "board", "inbox":
			for _, task := range t.Tasks {
				if task.ProjectID == p.ID {
					if v.Action == "inbox" && task.State != "blocked" && task.State != "review" && !(task.State == "active" && len(task.Attempts) > 0 && !now.Before(task.Attempts[len(task.Attempts)-1].LeaseUntil)) {
						continue
					}
					out.Tasks = append(out.Tasks, *task)
				}
			}
			sort.Slice(out.Tasks, func(i, j int) bool { return out.Tasks[i].ID < out.Tasks[j].ID })
		case "lessons":
			for _, l := range t.Learnings {
				task := t.Tasks[l.TaskID]
				current := task != nil && task.ProjectID == p.ID && task.Version == l.TaskVersion
				if l.ProjectID == p.ID && ((l.Published && current) || l.Author == actor) && (v.Query == "" || strings.Contains(strings.ToLower(l.Lesson+" "+l.AppliesWhen), strings.ToLower(v.Query))) {
					out.Learnings = append(out.Learnings, *l)
				}
			}
			sort.Slice(out.Learnings, func(i, j int) bool { return out.Learnings[i].ID < out.Learnings[j].ID })
			if len(out.Learnings) > 3 {
				out.Learnings = out.Learnings[:3]
			}
			for _, book := range t.Playbooks {
				l := t.Learnings[book.LearningID]
				if book.ProjectID == p.ID && book.Status == "active" && l != nil && l.Published && l.Version == book.SourceVersion && t.Tasks[l.TaskID] != nil && t.Tasks[l.TaskID].Version == l.TaskVersion {
					out.Playbooks = append(out.Playbooks, *book)
				}
			}
			sort.Slice(out.Playbooks, func(i, j int) bool { return out.Playbooks[i].ID < out.Playbooks[j].ID })
			if len(out.Playbooks) > 3 {
				out.Playbooks = out.Playbooks[:3]
			}
		case "task_create":
			if err := required(v.Name); err != nil {
				return out, nil, err
			}
			if len(v.Criteria) == 0 || len(t.Tasks) >= 500 {
				return out, nil, ErrInvalid
			}
			for _, criterion := range v.Criteria {
				if required(criterion) != nil {
					return out, nil, ErrInvalid
				}
			}
			// Dependencies only reference existing tasks. They cannot be edited,
			// so a newly created task cannot introduce a dependency cycle.
			for _, id := range v.Dependencies {
				dep := t.Tasks[id]
				if dep == nil || dep.ProjectID != p.ID {
					return out, nil, ErrForbidden
				}
			}
			task := &Task{ID: uuid.NewString(), ProjectID: p.ID, Name: v.Name, Criteria: v.Criteria, Dependencies: v.Dependencies, State: "ready", Version: 1, UpdatedAt: now, Attempts: []Attempt{}, Evidence: []Evidence{}}
			t.Tasks[task.ID] = task
			out.Task = task
			target = task.ID
		case "learning_draft":
			task := t.Tasks[v.TaskID]
			if task == nil || task.ProjectID != p.ID {
				return out, nil, ErrForbidden
			}
			if err := required(v.Lesson, v.AppliesWhen, v.Limitations); err != nil {
				return out, nil, err
			}
			if len(t.Learnings) >= 500 {
				return out, nil, ErrInvalid
			}
			l := &Learning{ID: uuid.NewString(), ProjectID: p.ID, TaskID: task.ID, TaskVersion: task.Version, Artifact: task.Artifact, Revision: task.Revision, Author: actor, Lesson: v.Lesson, AppliesWhen: v.AppliesWhen, Limitations: v.Limitations, Version: 1}
			t.Learnings[l.ID] = l
			out.Learning = l
			target = l.ID
		case "learning_publish", "learning_unshare", "learning_edit", "learning_delete", "playbook_adopt":
			l := t.Learnings[v.LearningID]
			if l == nil || l.ProjectID != p.ID {
				return out, nil, ErrForbidden
			}
			if !l.Published && l.Author != actor {
				return out, nil, ErrForbidden
			}
			if err := conflict(l.Version, v.Version); err != nil {
				return out, nil, err
			}
			if v.Action == "playbook_adopt" {
				if p.Members[actor] != "maintainer" || !l.Published {
					return out, nil, ErrForbidden
				}
				task := t.Tasks[l.TaskID]
				if task == nil || task.State != "done" || task.Version != l.TaskVersion {
					return out, nil, ErrConflict
				}
				if len(v.Steps) == 0 || len(t.Playbooks) >= 200 {
					return out, nil, ErrInvalid
				}
				for _, s := range v.Steps {
					if required(s) != nil {
						return out, nil, ErrInvalid
					}
				}
				for _, b := range t.Playbooks {
					if b.LearningID == l.ID && b.Status == "active" {
						return out, nil, ErrConflict
					}
				}
				book := &Playbook{ID: uuid.NewString(), ProjectID: p.ID, LearningID: l.ID, SourceVersion: l.Version, Maintainer: actor, Steps: v.Steps, Version: 1, Status: "active"}
				t.Playbooks[book.ID] = book
				p.GuidanceVersion++
				out.Playbooks = []Playbook{*book}
				target = book.ID
			} else {
				if l.Author != actor {
					return out, nil, ErrForbidden
				}
				if v.Action == "learning_edit" {
					if required(v.Lesson, v.AppliesWhen, v.Limitations) != nil {
						return out, nil, ErrInvalid
					}
					l.Lesson = v.Lesson
					l.AppliesWhen = v.AppliesWhen
					l.Limitations = v.Limitations
					task := t.Tasks[l.TaskID]
					if task == nil {
						return out, nil, ErrForbidden
					}
					l.TaskVersion = task.Version
					l.Artifact = task.Artifact
					l.Revision = task.Revision
				}
				if v.Action == "learning_publish" && (t.Tasks[l.TaskID] == nil || t.Tasks[l.TaskID].Version != l.TaskVersion) {
					return out, nil, ErrConflict
				}
				l.Published = v.Action == "learning_publish"
				l.Version++
				out.Learning = l
				target = l.ID
				for _, b := range t.Playbooks {
					if b.LearningID == l.ID && b.Status == "active" {
						b.Status = "needs_review"
						b.Version++
						p.GuidanceVersion++
					}
				}
				if v.Action == "learning_delete" {
					delete(t.Learnings, l.ID)
					out.Learning = nil
				}
			}
		default:
			task := t.Tasks[v.TaskID]
			if task == nil || task.ProjectID != p.ID {
				return out, nil, ErrForbidden
			}
			out.Task = task
			target = task.ID
			if v.Action == "task_get" {
				if len(task.Attempts) > 0 {
					out.Stale = now.After(task.Attempts[len(task.Attempts)-1].LeaseUntil) && slices.Contains([]string{"active", "blocked"}, task.State)
				}
				return out, nil, nil
			}
			if err := conflict(task.Version, v.Version); err != nil {
				return out, nil, err
			}
			if task.State == "done" || task.State == "cancelled" {
				return out, nil, ErrConflict
			}
			if v.Action == "task_claim" {
				if required(v.Agent) != nil {
					return out, nil, ErrInvalid
				}
				if len(task.Attempts) >= 50 {
					return out, nil, ErrInvalid
				}
				if task.State != "ready" { // Explicit reclaim is allowed after expiry, never automatic.
					if (task.State == "review" && task.Owner != actor) || len(task.Attempts) == 0 || now.Before(task.Attempts[len(task.Attempts)-1].LeaseUntil) {
						return out, nil, ErrConflict
					}
				}
				for _, id := range task.Dependencies {
					if t.Tasks[id].State != "done" {
						return out, nil, ErrConflict
					}
				}
				task.Owner = actor
				task.State = "active"
				task.Evidence = nil
				task.Attempts = append(task.Attempts, Attempt{ID: uuid.NewString(), Initiator: actor, Agent: v.Agent, SessionRef: v.SessionRef, StartedAt: now, LeaseUntil: now.Add(LeaseDuration)})
			} else if v.Action == "task_approve" {
				if p.Members[actor] != "maintainer" || task.Owner == actor {
					return out, nil, ErrForbidden
				}
				// A reviewer cannot approve an attempt they contributed to.
				for _, a := range task.Attempts {
					if a.Initiator == actor {
						return out, nil, ErrForbidden
					}
				}
				if task.State != "review" || v.Revision != task.Revision || task.ReviewGuidanceVersion != p.GuidanceVersion {
					return out, nil, ErrConflict
				}
				if err := readyForReview(task, now); err != nil {
					return out, nil, err
				}
				task.State = "done"
				task.ApprovedBy = actor
				task.ApprovedVersion = task.Version + 1
			} else {
				if task.Owner != actor {
					return out, nil, ErrForbidden
				}
				if len(task.Attempts) == 0 || !now.Before(task.Attempts[len(task.Attempts)-1].LeaseUntil) {
					return out, nil, ErrConflict
				}
				switch v.Action {
				case "task_checkpoint":
					if required(v.Checkpoint, v.NextStep) != nil {
						return out, nil, ErrInvalid
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
						return out, nil, ErrInvalid
					}
					task.Owner = ""
					task.State = "ready"
					task.Evidence = nil
				case "task_state":
					if !slices.Contains([]string{"active", "blocked", "cancelled"}, v.State) || (v.State == "blocked" && required(v.NextStep) != nil) {
						return out, nil, ErrInvalid
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
						return out, nil, err
					}
					task.Revision = candidate.Revision
					task.Artifact = candidate.Artifact
					task.Evidence = candidate.Evidence
					task.State = "review"
					task.ReviewGuidanceVersion = p.GuidanceVersion
				default:
					return out, nil, ErrInvalid
				}
			}
			task.Version++
			task.UpdatedAt = now
		}
	}
	if !v.Write() {
		return out, nil, nil
	}
	return out, &Event{ID: uuid.NewString(), TeamID: t.ID, ProjectID: v.ProjectID, Actor: actor, Action: v.Action, TargetID: target, At: now}, nil
}

func readyForReview(t *Task, now time.Time) error {
	if err := validateEvidence(t, now); err != nil {
		return err
	}
	for i := range t.Criteria {
		found := false
		for _, e := range t.Evidence {
			if e.Criterion == i {
				if e.Status != "pass" {
					return fmt.Errorf("%w: failed criterion", ErrInvalid)
				}
				found = true
			}
		}
		if !found {
			return fmt.Errorf("%w: missing criterion evidence", ErrInvalid)
		}
	}
	return nil
}

// Submission preserves gaps and failures for review; acceptance requires
// complete passing evidence. A report still has to be structurally valid.
func validateEvidence(t *Task, now time.Time) error {
	if required(t.Revision, t.Artifact) != nil {
		return fmt.Errorf("%w: artifact and revision required", ErrInvalid)
	}
	for _, e := range t.Evidence {
		if e.Criterion < 0 || e.Criterion >= len(t.Criteria) || required(e.Check) != nil || !slices.Contains([]string{"pass", "fail"}, e.Status) || e.Revision != t.Revision || e.ObservedAt.IsZero() || e.ObservedAt.After(now) {
			return fmt.Errorf("%w: evidence must name a criterion, check, revision and past timestamp", ErrInvalid)
		}
	}
	return nil
}
