// Package workspace owns the authorized manual-launch team workflow.
// Private memory and local journals never enter this model implicitly.
package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
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
