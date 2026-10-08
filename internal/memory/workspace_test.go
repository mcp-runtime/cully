package memory

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mcp-runtime/cully/internal/workspace"
)

func TestWorkspaceSharesMemoryValidation(t *testing.T) {
	for _, v := range []workspace.Input{
		{Action: "task_checkpoint", TeamID: uuid.NewString(), Checkpoint: "api_key=1234567890abcdef"},
		{Action: "learning_draft", TeamID: uuid.NewString(), Lesson: "-----BEGIN PRIVATE KEY-----"},
		{Action: "task_claim", TeamID: uuid.NewString(), SessionRef: "raw-native-session"},
	} {
		r := Request{Operation: "workspace", Workspace: &v}
		if err := r.Validate(); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid workspace content accepted: %v", err)
		}
	}
	v := workspace.Input{Action: "whoami"}
	for _, r := range []Request{{Operation: "workspace", Workspace: &v, Log: &LogInput{}}, {Operation: "log", Workspace: &v}} {
		if err := r.Validate(); !errors.Is(err, ErrInvalid) {
			t.Fatal("mixed operation accepted")
		}
	}
}
