// Package workspace owns the authorized manual-launch team workflow.
// Private memory and local journals never enter this model implicitly.
package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalid = errors.New("invalid workspace input")
var ErrForbidden = errors.New("workspace access denied")
var ErrConflict = errors.New("workspace version or claim conflict")

// Principal binds shared access to a verified issuer and subject. Existing
// private memory owner IDs are intentionally unaffected.
func Principal(issuer, subject string) string {
	sum := sha256.Sum256([]byte(issuer + "\x00" + subject))
	return "oauth-" + hex.EncodeToString(sum[:])
}

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

type Evidence struct {
	Criterion  int       `json:"criterion"`
	Check      string    `json:"check"`
	Status     string    `json:"status"`
	Revision   string    `json:"revision"`
	ObservedAt time.Time `json:"observed_at"`
	// MVP evidence is always reported; a client cannot self-attest verification.
	Source string `json:"source,omitempty"`
}
type Team struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Members   map[string]string    `json:"members"`
	Projects  map[string]*Project  `json:"projects"`
	Tasks     map[string]*Task     `json:"tasks"`
	Learnings map[string]*Learning `json:"learnings"`
	Playbooks map[string]*Playbook `json:"playbooks"`
}
type Project struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Repository      string            `json:"repository"`
	Members         map[string]string `json:"members"`
	GuidanceVersion int               `json:"guidance_version"`
}
type Attempt struct {
	ID         string    `json:"id"`
	Initiator  string    `json:"initiator"`
	Agent      string    `json:"agent"`
	SessionRef string    `json:"session_ref,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	LeaseUntil time.Time `json:"lease_until"`
}
type Task struct {
	ID                    string     `json:"id"`
	ProjectID             string     `json:"project_id"`
	Name                  string     `json:"name"`
	Criteria              []string   `json:"criteria"`
	Dependencies          []string   `json:"dependencies"`
	State                 string     `json:"state"`
	Owner                 string     `json:"owner,omitempty"`
	Version               int        `json:"version"`
	Attempts              []Attempt  `json:"attempts"`
	Branch                string     `json:"branch,omitempty"`
	Checkpoint            string     `json:"checkpoint,omitempty"`
	NextStep              string     `json:"next_step,omitempty"`
	Revision              string     `json:"revision,omitempty"`
	Artifact              string     `json:"artifact,omitempty"`
	Evidence              []Evidence `json:"evidence"`
	UpdatedAt             time.Time  `json:"updated_at"`
	ReviewGuidanceVersion int        `json:"review_guidance_version,omitempty"`
	ApprovedBy            string     `json:"approved_by,omitempty"`
	ApprovedVersion       int        `json:"approved_version,omitempty"`
}
type Learning struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	TaskID      string `json:"task_id"`
	TaskVersion int    `json:"task_version"`
	Artifact    string `json:"artifact,omitempty"`
	Revision    string `json:"revision,omitempty"`
	Author      string `json:"author"`
	Lesson      string `json:"lesson"`
	AppliesWhen string `json:"applies_when"`
	Limitations string `json:"limitations"`
	Version     int    `json:"version"`
	Published   bool   `json:"published"`
}
type Playbook struct {
	ID            string   `json:"id"`
	ProjectID     string   `json:"project_id"`
	LearningID    string   `json:"learning_id"`
	SourceVersion int      `json:"source_version"`
	Maintainer    string   `json:"maintainer"`
	Steps         []string `json:"steps"`
	Version       int      `json:"version"`
	Status        string   `json:"status"`
}
type Event struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	ProjectID string    `json:"project_id,omitempty"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	TargetID  string    `json:"target_id"`
	At        time.Time `json:"at"`
}
type Result struct {
	Principal string     `json:"principal,omitempty"`
	TeamID    string     `json:"team_id,omitempty"`
	Project   *Project   `json:"project,omitempty"`
	Projects  []Project  `json:"projects,omitempty"`
	Task      *Task      `json:"task,omitempty"`
	Tasks     []Task     `json:"tasks,omitempty"`
	Learning  *Learning  `json:"learning,omitempty"`
	Learnings []Learning `json:"learnings,omitempty"`
	Playbooks []Playbook `json:"playbooks,omitempty"`
	Stale     bool       `json:"stale,omitempty"`
}

func (v Input) Write() bool {
	switch v.Action {
	case "whoami", "projects", "board", "inbox", "task_get", "lessons":
		return false
	default:
		return true
	}
}

func (v Input) Validate() error {
	switch v.Action {
	case "whoami":
		return nil
	case "team_create", "team_member", "project_create", "project_member", "projects", "board", "inbox", "task_get", "task_create", "task_claim", "task_checkpoint", "task_release", "task_state", "task_submit", "task_approve", "learning_draft", "learning_publish", "learning_unshare", "learning_edit", "learning_delete", "playbook_adopt", "lessons":
	default:
		return fmt.Errorf("%w: unknown action", ErrInvalid)
	}
	for _, s := range []string{v.TeamID, v.ProjectID, v.TaskID, v.LearningID} {
		if s != "" {
			if _, err := uuid.Parse(s); err != nil {
				return fmt.Errorf("%w: IDs must be UUIDs", ErrInvalid)
			}
		}
	}
	if v.Action != "team_create" && v.TeamID == "" {
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
