package workspace

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

type Input struct {
	Action       string     `json:"action"`
	TeamID       string     `json:"team_id,omitempty"`
	ProjectID    string     `json:"project_id,omitempty"`
	TaskID       string     `json:"task_id,omitempty"`
	LearningID   string     `json:"learning_id,omitempty"`
	Version      int        `json:"version,omitempty"`
	Name         string     `json:"name,omitempty"`
	Principal    string     `json:"principal,omitempty"`
	Role         string     `json:"role,omitempty"`
	Repository   string     `json:"repository,omitempty"`
	Criteria     []string   `json:"criteria,omitempty"`
	Dependencies []string   `json:"dependencies,omitempty"`
	State        string     `json:"state,omitempty"`
	Agent        string     `json:"agent,omitempty"`
	SessionRef   string     `json:"session_ref,omitempty"`
	Branch       string     `json:"branch,omitempty"`
	Checkpoint   string     `json:"checkpoint,omitempty"`
	NextStep     string     `json:"next_step,omitempty"`
	Revision     string     `json:"revision,omitempty"`
	Artifact     string     `json:"artifact,omitempty"`
	Evidence     []Evidence `json:"evidence,omitempty"`
	Lesson       string     `json:"lesson,omitempty"`
	AppliesWhen  string     `json:"applies_when,omitempty"`
	Limitations  string     `json:"limitations,omitempty"`
	Steps        []string   `json:"steps,omitempty"`
	Query        string     `json:"query,omitempty"`
}

func (v Input) Write() bool {
	op, ok := operations[v.Action]
	return !ok || op.write
}
func (v Input) Validate() error {
	op, ok := operations[v.Action]
	if !ok {
		return fmt.Errorf("%w: unknown action", ErrInvalid)
	}
	if v.Action == "whoami" {
		return nil
	}

	for _, s := range []string{v.TeamID, v.ProjectID, v.TaskID, v.LearningID} {
		if s != "" {
			if _, err := uuid.Parse(s); err != nil {
				return fmt.Errorf("%w: IDs must be UUIDs", ErrInvalid)
			}
		}
	}
	if op.scope != globalScope && v.TeamID == "" {
		return fmt.Errorf("%w: team_id required", ErrInvalid)
	}
	if v.Version < 0 {
		return ErrInvalid
	}
	if len(v.Criteria) > 20 || len(v.Dependencies) > 20 || len(v.Evidence) > 40 || len(v.Steps) > 20 {
		return ErrInvalid
	}
	texts := []string{v.Name, v.Principal, v.Role, v.Repository, v.State, v.Agent, v.SessionRef, v.Branch, v.Checkpoint, v.NextStep, v.Revision, v.Artifact, v.Lesson, v.AppliesWhen, v.Limitations, v.Query}
	texts = append(texts, v.Criteria...)
	texts = append(texts, v.Steps...)
	for _, e := range v.Evidence {
		texts = append(texts, e.Check, e.Status, e.Revision, e.Source)
	}
	for _, s := range texts {
		if len(s) > 8000 || strings.ContainsRune(s, '\x00') {
			return ErrInvalid
		}
	}
	if len(v.Principal) > 512 {
		return ErrInvalid
	}
	for _, link := range []string{v.Repository, v.Artifact} {
		if link == "" {
			continue
		}
		u, err := url.Parse(link)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("%w: links require HTTPS without credentials, query or fragment", ErrInvalid)
		}
	}
	return nil
}
