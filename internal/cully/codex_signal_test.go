package cully

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCodexSignalHookStoresCountersWithoutPayloads(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	t.Setenv("CULLY_PANE_SESSION", "pane-a")
	RunSignalHook(strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"go test ./... secret-value"},"tool_response":{"exit_code":1,"output":"private-output"}}`))
	data, err := os.ReadFile(signalFile("pane-a"))
	if err != nil || string(data) != "T!\n" {
		t.Fatalf("unexpected signal data: %q, %v", data, err)
	}
	stats := readToolStats("pane-a")
	if stats.Tools != 1 || stats.Errors != 1 || stats.Checks != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	for i := 0; i < 3; i++ {
		RunSignalHook(strings.NewReader(`{"tool_name":"apply_patch","tool_response":{}}`))
	}
	stats = readToolStats("pane-a")
	if stats.Edits != 3 || stats.Checks != 1 || stats.EditsSinceCheck != 3 {
		t.Fatalf("unexpected stats after edits: %+v", stats)
	}
	if got := statsAdvice(stats); len(got) != 1 || !strings.Contains(got[0], "focused check") {
		t.Fatalf("unexpected advice: %v", got)
	}
	RunSignalHook(strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"go test ./..."},"tool_response":{"exit_code":0}}`))
	if got := statsAdvice(readToolStats("pane-a")); len(got) != 1 || !strings.HasPrefix(got[0], "MEMO|") {
		t.Fatalf("successful check did not clear edit advice: %v", got)
	}
}

func TestCodexSignalHookOnlyRunsInsidePane(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	t.Setenv("CULLY_PANE_SESSION", "")
	RunSignalHook(strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"rg secret"}}`))
	if _, err := os.Stat(signalFile("pane-a")); !os.IsNotExist(err) {
		t.Fatalf("signal file created outside pane: %v", err)
	}
}

func TestCodexSignalHookBindsOnlyOneLivePaneForCwd(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	t.Setenv("CULLY_PANE_SESSION", "")
	t.Setenv("MODEL_HINT_GUARD", "")
	cwd := t.TempDir()
	if err := registerPane("pane-a", cwd); err != nil {
		t.Fatal(err)
	}
	start := fmt.Sprintf(`{"cwd":%q,"session_id":"thread-a"}`, cwd)
	RunContinuityHook("codex", "start", strings.NewReader(start), io.Discard)
	binding, err := os.ReadFile(paneBindingFile("pane-a"))
	if err != nil || strings.Contains(string(binding), "thread-a") {
		t.Fatalf("missing or raw session binding: %q, %v", binding, err)
	}
	event := fmt.Sprintf(`{"cwd":%q,"session_id":"thread-a","tool_name":"Bash","tool_input":{"command":"rg file"},"tool_response":{}}`, cwd)
	RunSignalHook(strings.NewReader(event))
	data, err := os.ReadFile(signalFile("pane-a"))
	if err != nil || string(data) != "S.\n" {
		t.Fatalf("hook did not route to active pane: %q, %v", data, err)
	}
	otherSession := fmt.Sprintf(`{"cwd":%q,"session_id":"thread-b","tool_name":"Bash","tool_input":{"command":"rg file"},"tool_response":{}}`, cwd)
	RunSignalHook(strings.NewReader(otherSession))
	data, _ = os.ReadFile(signalFile("pane-a"))
	if string(data) != "S.\n" {
		t.Fatalf("another Codex session reached the pane: %q", data)
	}

	if err := registerPane("pane-b", cwd); err != nil {
		t.Fatal(err)
	}
	RunSignalHook(strings.NewReader(event))
	data, _ = os.ReadFile(signalFile("pane-a"))
	if string(data) != "S.\n" {
		t.Fatalf("ambiguous pane received signal: %q", data)
	}
	if _, err := os.Stat(signalFile("pane-b")); !os.IsNotExist(err) {
		t.Fatalf("second pane received signal: %v", err)
	}

	if err := os.Remove(paneRegistrationFile("pane-b")); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-time.Minute)
	if err := os.Chtimes(paneRegistrationFile("pane-a"), stale, stale); err != nil {
		t.Fatal(err)
	}
	RunSignalHook(strings.NewReader(event))
	data, _ = os.ReadFile(signalFile("pane-a"))
	if string(data) != "S.\n" {
		t.Fatalf("stale pane received signal: %q", data)
	}
}

func TestCodexAdviceUsesBoundedCounters(t *testing.T) {
	got := statsAdvice(toolStats{Tools: 20, Errors: 3, Searches: 11, Edits: 2, EditsSinceCheck: 2})
	if len(got) != 3 || !strings.Contains(got[0], "failed") || !strings.Contains(got[1], "searches") || !strings.Contains(got[2], "check") {
		t.Fatalf("unexpected advice: %v", got)
	}
}

func TestCodexSignalClassifiesShellToolsWithoutGuessingNestedCalls(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		class       byte
	}{
		{"functions.exec_command", `{"cmd":"go test ./internal/cully"}`, 'T'},
		{"exec_command", `{"cmd":"rg -n Context internal/cully"}`, 'S'},
		{"functions.apply_patch", `{}`, 'E'},
		{"functions.create_file", `{"file_path":"new.go"}`, 'E'},
		{"NotebookEdit", `{"notebook_path":"analysis.ipynb"}`, 'E'},
		{"Bash", `{"command":"go vet ./..."}`, 'T'},
		{"shell", `{"command":["go","test","./..."]}`, 'T'},
		{"Read", `{"file_path":"a.go"}`, 'S'},
		{"Task", `{"description":"explore"}`, 'A'},
		{"Agent", `{"description":"explore"}`, 'A'},
		{"WebFetch", `{"url":"https://example.com"}`, 'W'},
		{"WebSearch", `{"query":"go"}`, 'W'},
		{"Bash", `{"command":"git commit -m x"}`, 'G'},
		{"Bash", `{"command":"git commit-tree HEAD"}`, 'O'},
		{"exec_command", `{"cmd":"gofmt -w internal/cully/codex_status.go"}`, 'O'},
		{"Bash", `{"command":"go fmt ./..."}`, 'O'},
		{"functions.exec", `{"code":"await tools.exec_command({cmd: 'go test ./...'})"}`, 'O'},
	} {
		if got := toolClass(toolEvent{ToolName: tc.name, ToolInput: json.RawMessage(tc.input)}); got != tc.class {
			t.Fatalf("%s class=%c want=%c", tc.name, got, tc.class)
		}
	}
}

func TestCodexFormattingDoesNotClearPendingVerification(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	t.Setenv("CULLY_PANE_SESSION", "pane-a")
	for _, event := range []string{
		`{"tool_name":"functions.apply_patch","tool_response":{}}`,
		`{"tool_name":"functions.exec_command","tool_input":{"cmd":"gofmt -w app.go"},"tool_response":{"exit_code":0}}`,
		`{"tool_name":"functions.exec_command","tool_input":{"cmd":"go test ./..."},"tool_response":{"exit_code":1}}`,
	} {
		RunSignalHook(strings.NewReader(event))
	}
	stats := readToolStats("pane-a")
	if stats.EditsSinceCheck != 1 || stats.Checks != 1 || stats.Errors != 1 {
		t.Fatalf("formatting or failed test incorrectly cleared verification: %+v", stats)
	}
	RunSignalHook(strings.NewReader(`{"tool_name":"functions.exec_command","tool_input":{"cmd":"go test ./..."},"tool_response":{"exit_code":0}}`))
	if stats = readToolStats("pane-a"); stats.EditsSinceCheck != 0 || stats.Checks != 2 {
		t.Fatalf("successful direct check didn't clear verification: %+v", stats)
	}
}

func TestCodexFailuresRequireExplicitStructuredEvidence(t *testing.T) {
	for _, raw := range []string{`{"output":"failed exit_code 1"}`, `{"content":[{"type":"text","text":"exit_code: 1"}]}`, `"command failed"`} {
		if toolFailed(json.RawMessage(raw)) {
			t.Fatalf("arbitrary nested output must not invent failure: %s", raw)
		}
	}
}
