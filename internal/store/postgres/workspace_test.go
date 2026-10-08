package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/mcp-runtime/cully/internal/memory"
	"github.com/mcp-runtime/cully/internal/store/remote"
	"github.com/mcp-runtime/cully/internal/transport/datahttp"
	mcpt "github.com/mcp-runtime/cully/internal/transport/mcp"
	"github.com/mcp-runtime/cully/internal/workspace"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestWorkspaceMCPAndAtomicClaims(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	data := httptest.NewServer(datahttp.Handler(memory.Service{Store: s}, "test-service-key", nil))
	defer data.Close()
	r, _ := remote.New(data.URL, "test-service-key")
	verifier := func(_ context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		scope := []string{"tools:read", "tools:write"}
		if token == "reader" {
			scope = scope[:1]
		}
		return &auth.TokenInfo{UserID: token, Scopes: scope, Expiration: time.Now().Add(time.Hour)}, nil
	}
	server := httptest.NewServer(mcpt.Handler(memory.Service{Store: r}, mcpt.AuthConfig{Mode: "oauth", Issuer: "https://issuer.test", Resource: "https://public.test/mcp", Verifier: verifier}, "/mcp", "test"))
	defer server.Close()
	connect := func(token string) *sdk.ClientSession {
		t.Helper()
		client := sdk.NewClient(&sdk.Implementation{Name: "workspace-test", Version: "1"}, nil)
		session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: server.URL + "/mcp", HTTPClient: &http.Client{Transport: signedTransport{token}}, DisableStandaloneSSE: true}, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { session.Close() })
		return session
	}
	lead, dev, next := connect("lead"), connect("dev"), connect("next")
	call := func(client *sdk.ClientSession, v workspace.Input) *workspace.Result {
		t.Helper()
		name := "cully_workspace_read"
		if v.Write() {
			name = "cully_workspace_write"
		}
		out, err := client.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: v})
		if err != nil || out.IsError {
			t.Fatalf("%s: %v %+v", v.Action, err, out)
		}
		b, _ := json.Marshal(out.StructuredContent)
		var result memory.Result
		if err = json.Unmarshal(b, &result); err != nil {
			t.Fatal(err)
		}
		return result.Workspace
	}
	team := call(lead, workspace.Input{Action: "team_create", Name: "Engineering"}).TeamID
	devID := call(dev, workspace.Input{Action: "whoami"}).Principal
	nextID := call(next, workspace.Input{Action: "whoami"}).Principal
	if devID != workspace.Principal("https://issuer.test", "dev") {
		t.Fatal("issuer-scoped identity lost across private API")
	}
	for _, id := range []string{devID, nextID} {
		call(lead, workspace.Input{Action: "team_member", TeamID: team, Principal: id, Role: "member"})
	}
	project := call(lead, workspace.Input{Action: "project_create", TeamID: team, Name: "Auth", Repository: "https://github.com/mcp-runtime/cully"}).Project.ID
	for _, id := range []string{devID, nextID} {
		call(lead, workspace.Input{Action: "project_member", TeamID: team, ProjectID: project, Principal: id, Role: "member"})
	}
	task := call(lead, workspace.Input{Action: "task_create", TeamID: team, ProjectID: project, Name: "Fix auth", Criteria: []string{"Negative tests pass"}}).Task
	// Public read tool and read-only OAuth tokens cannot dispatch mutations.
	for _, c := range []*sdk.ClientSession{lead, connect("reader")} {
		name := "cully_workspace_write"
		if c == lead {
			name = "cully_workspace_read"
		}
		out, err := c.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: workspace.Input{Action: "task_claim", TeamID: team, ProjectID: project, TaskID: task.ID, Version: 1, Agent: "codex"}})
		if err != nil || !out.IsError {
			t.Fatal("scope/action escalation succeeded")
		}
	}
	// Two transactions racing for one task have exactly one winner.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, id := range []string{devID, nextID} {
		wg.Add(1)
		go func(actor string) {
			defer wg.Done()
			v := workspace.Input{Action: "task_claim", TeamID: team, ProjectID: project, TaskID: task.ID, Version: 1, Agent: "codex"}
			_, err := (memory.Service{Store: s}).Execute(ctx, actor, memory.Request{Operation: "workspace", Workspace: &v})
			results <- err
		}(id)
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for err := range results {
		if err == nil {
			wins++
		} else if errors.Is(err, memory.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("wins=%d conflicts=%d", wins, conflicts)
	}
	board := call(lead, workspace.Input{Action: "board", TeamID: team, ProjectID: project})
	task = &board.Tasks[0]
	ownerClient := dev
	if task.Owner == nextID {
		ownerClient = next
	}
	call(ownerClient, workspace.Input{Action: "task_checkpoint", TeamID: team, ProjectID: project, TaskID: task.ID, Version: 2, Checkpoint: "Patch ready on feature branch", NextStep: "Run acceptance test", Branch: "fix/auth"})
	if got := call(lead, workspace.Input{Action: "task_get", TeamID: team, ProjectID: project, TaskID: task.ID}); got.Task.Checkpoint == "" {
		t.Fatal("portable checkpoint missing")
	}
	call(ownerClient, workspace.Input{Action: "task_submit", TeamID: team, ProjectID: project, TaskID: task.ID, Version: 3, Revision: "commit-1", Artifact: "https://github.com/mcp-runtime/cully/pull/19"})
	incomplete, err := lead.CallTool(ctx, &sdk.CallToolParams{Name: "cully_workspace_write", Arguments: workspace.Input{Action: "task_approve", TeamID: team, ProjectID: project, TaskID: task.ID, Version: 4, Revision: "commit-1"}})
	if err != nil || !incomplete.IsError {
		t.Fatal("missing evidence accepted through private API")
	}
	packet := call(lead, workspace.Input{Action: "task_get", TeamID: team, ProjectID: project, TaskID: task.ID})
	if packet.Task.State != "review" || packet.Task.Version != 4 || len(packet.Task.Evidence) != 0 {
		t.Fatal("rejected approval changed packet or hid its gaps")
	}
	call(ownerClient, workspace.Input{Action: "task_submit", TeamID: team, ProjectID: project, TaskID: task.ID, Version: 4, Revision: "commit-1", Artifact: "https://github.com/mcp-runtime/cully/pull/19", Evidence: []workspace.Evidence{{Criterion: 0, Check: "auth regression", Status: "pass", Revision: "commit-1", ObservedAt: time.Now().Add(-time.Second)}}})
	call(lead, workspace.Input{Action: "task_approve", TeamID: team, ProjectID: project, TaskID: task.ID, Version: 5, Revision: "commit-1"})
	lesson := call(ownerClient, workspace.Input{Action: "learning_draft", TeamID: team, ProjectID: project, TaskID: task.ID, Lesson: "Validate issuer before accepting identity", AppliesWhen: "OAuth task", Limitations: "One provider tested"}).Learning
	if len(call(lead, workspace.Input{Action: "lessons", TeamID: team, ProjectID: project}).Learnings) != 0 {
		t.Fatal("private database draft leaked through MCP")
	}
	call(ownerClient, workspace.Input{Action: "learning_publish", TeamID: team, ProjectID: project, LearningID: lesson.ID, Version: 1})
	call(lead, workspace.Input{Action: "playbook_adopt", TeamID: team, ProjectID: project, LearningID: lesson.ID, Version: 2, Steps: []string{"Run a wrong-issuer test"}})
	if len(call(lead, workspace.Input{Action: "lessons", TeamID: team, ProjectID: project}).Playbooks) != 1 {
		t.Fatal("published guidance missing through private API")
	}
	call(ownerClient, workspace.Input{Action: "learning_unshare", TeamID: team, ProjectID: project, LearningID: lesson.ID, Version: 2})
	withdrawn := call(lead, workspace.Input{Action: "lessons", TeamID: team, ProjectID: project})
	if len(withdrawn.Learnings) != 0 || len(withdrawn.Playbooks) != 0 {
		t.Fatal("withdrawn database source leaked through MCP")
	}
	// Direct IDs, project IDs and cross-team IDs are all scoped on the server.
	otherTeam := call(lead, workspace.Input{Action: "team_create", Name: "Other"}).TeamID
	for _, v := range []workspace.Input{{Action: "task_get", TeamID: otherTeam, ProjectID: project, TaskID: task.ID}, {Action: "task_get", TeamID: team, ProjectID: task.ID, TaskID: task.ID}} {
		out, err := lead.CallTool(ctx, &sdk.CallToolParams{Name: "cully_workspace_read", Arguments: v})
		if err != nil || !out.IsError {
			t.Fatal("cross-boundary direct ID read")
		}
	}
	call(lead, workspace.Input{Action: "team_member", TeamID: team, Principal: devID, Role: "remove"})
	denied, err := dev.CallTool(ctx, &sdk.CallToolParams{Name: "cully_workspace_read", Arguments: workspace.Input{Action: "board", TeamID: team, ProjectID: project}})
	if err != nil || !denied.IsError {
		t.Fatal("removed member read project")
	}
	var count int
	if err = s.Pool.QueryRow(ctx, "SELECT count(*) FROM cully_workspace_events WHERE team_id=$1::uuid", team).Scan(&count); err != nil || count < 10 {
		t.Fatalf("audit events missing: %d %v", count, err)
	}
	if err = s.Pool.QueryRow(ctx, "SELECT count(*) FROM cully_entries").Scan(&count); err != nil || count != 0 {
		t.Fatal("workspace wrote into private memory")
	}
}
