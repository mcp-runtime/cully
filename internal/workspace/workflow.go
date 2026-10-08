package workspace

import (
	"fmt"
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

// Apply runs under the repository's team-row transaction lock. Callers
// must discard state on error. Handlers return only authorized projections.
func (t *Team) Apply(actor string, v Input, now time.Time) (Result, *Event, error) {
	out := Result{TeamID: t.ID}
	if t.Members[actor] == "" || v.TeamID != t.ID {
		return out, nil, ErrForbidden
	}
	op, ok := operations[v.Action]
	if !ok || op.handle == nil {
		return out, nil, ErrInvalid
	}
	var p *Project
	if op.scope == projectScope || op.scope == membershipScope {
		var err error
		p, err = t.project(actor, v.ProjectID, op.write)
		// Team admins may manage project grants without gaining content access.
		if op.scope == membershipScope && t.Members[actor] == "admin" {
			p = t.Projects[v.ProjectID]
			err = nil
			if p == nil {
				err = ErrForbidden
			}
		}
		if err != nil {
			return out, nil, err
		}
	}
	out, target, err := op.handle(t, actor, p, v, now)
	if err != nil || !op.write {
		return out, nil, err
	}
	projectID := ""
	if op.scope == projectScope || op.scope == membershipScope {
		projectID = v.ProjectID
	}
	if out.Project != nil {
		projectID = out.Project.ID
	}
	return out, &Event{ID: uuid.NewString(), TeamID: t.ID, ProjectID: projectID, Actor: actor, Action: v.Action, TargetID: target, At: now}, nil
}
