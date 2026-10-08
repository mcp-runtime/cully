package workspace

import (
	"time"

	"github.com/google/uuid"
)

func adoptPlaybook(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	out := Result{TeamID: t.ID}

	l := t.Learnings[v.LearningID]
	if l == nil || l.ProjectID != p.ID {
		return out, "", ErrForbidden
	}
	if !l.Published && l.Author != actor {
		return out, "", ErrForbidden
	}
	if err := conflict(l.Version, v.Version); err != nil {
		return out, "", err
	}
	if p.Members[actor] != "maintainer" || !l.Published {
		return out, "", ErrForbidden
	}
	task := t.Tasks[l.TaskID]
	if task == nil || task.State != "done" || task.Version != l.TaskVersion {
		return out, "", ErrConflict
	}
	if len(v.Steps) == 0 || len(t.Playbooks) >= 200 {
		return out, "", ErrInvalid
	}
	for _, s := range v.Steps {
		if required(s) != nil {
			return out, "", ErrInvalid
		}
	}
	for _, b := range t.Playbooks {
		if b.LearningID == l.ID && b.Status == "active" {
			return out, "", ErrConflict
		}
	}
	book := &Playbook{ID: uuid.NewString(), ProjectID: p.ID, LearningID: l.ID, SourceVersion: l.Version, Maintainer: actor, Steps: v.Steps, Version: 1, Status: "active"}
	t.Playbooks[book.ID] = book
	p.GuidanceVersion++
	out.Playbooks = []Playbook{*book}
	return out, book.ID, nil
}
