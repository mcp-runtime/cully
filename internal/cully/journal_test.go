package cully

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func journalTestEnv(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	dir := t.TempDir()
	return dir
}

func TestJournalRecordsOnlyCoarseFacts(t *testing.T) {
	cwd := journalTestEnv(t)
	appendJournal(cwd, "s1", journalEvent{Agent: "claude", Class: journalCheck, Failed: true, Sig: commandSig("go  test ./...")})
	file, events, ok := latestJournal(cwd)
	if !ok || len(events) != 1 || file.Session != "s1" {
		t.Fatalf("journal = %+v %+v %v", file, events, ok)
	}
	raw, err := os.ReadFile(file.Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "go test") || strings.Contains(string(raw), cwd) {
		t.Fatalf("journal leaked a command or path: %s", raw)
	}
	if info, _ := os.Stat(file.Path); info.Mode().Perm() != 0o600 {
		t.Fatalf("journal mode = %v", info.Mode().Perm())
	}
}

func TestCommandSigNormalizesCommands(t *testing.T) {
	if commandSig("go test   ./...") != commandSig(" GO test ./... ") {
		t.Fatal("equivalent commands must share a signature")
	}
	if commandSig("go test ./a") == commandSig("go test ./b") {
		t.Fatal("different commands must differ")
	}
	if commandSig("  ") != "" {
		t.Fatal("empty command has no signature")
	}
}

func TestLatestJournalIsPerProject(t *testing.T) {
	a, b := journalTestEnv(t), t.TempDir()
	appendJournal(a, "one", journalEvent{Class: journalEdit})
	appendJournal(b, "two", journalEvent{Class: journalSearch})
	if file, _, _ := latestJournal(a); file.Session != "one" {
		t.Fatalf("project a session = %q", file.Session)
	}
	if file, _, _ := latestJournal(b); file.Session != "two" {
		t.Fatalf("project b session = %q", file.Session)
	}
}

func TestJournalStaysBounded(t *testing.T) {
	cwd := journalTestEnv(t)
	path := journalPath(cwd, "big")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	line := `{"t":"2026-10-08T10:00:00Z","c":"O"}` + "\n"
	if err := os.WriteFile(path, []byte(strings.Repeat(line, journalMaxBytes/len(line)+10)), 0o600); err != nil {
		t.Fatal(err)
	}
	before := len(readJournal(path))
	appendJournal(cwd, "big", journalEvent{Class: journalEdit})
	after := len(readJournal(path))
	if after >= before {
		t.Fatalf("journal did not shrink: %d -> %d", before, after)
	}
}

func TestPruneJournalsDropsOldFiles(t *testing.T) {
	cwd := journalTestEnv(t)
	appendJournal(cwd, "old", journalEvent{Class: journalEdit})
	appendJournal(cwd, "new", journalEvent{Class: journalEdit})
	old := time.Now().Add(-journalMaxAge - time.Hour)
	if err := os.Chtimes(journalPath(cwd, "old"), old, old); err != nil {
		t.Fatal(err)
	}
	pruneJournals()
	files := projectJournals(cwd)
	if len(files) != 1 || files[0].Session != "new" {
		t.Fatalf("files = %+v", files)
	}
}

func TestPaneSignalHookWritesJournal(t *testing.T) {
	cwd := journalTestEnv(t)
	t.Setenv("CULLY_PANE_SESSION", "pane-1")
	RunPaneSignalHook("claude", strings.NewReader(`{"cwd":"`+cwd+`","session_id":"x","tool_name":"Bash","tool_input":{"command":"go test ./..."},"tool_response":{"exit_code":1}}`))
	_, events, ok := latestJournal(cwd)
	if !ok || len(events) != 1 {
		t.Fatalf("events = %+v", events)
	}
	event := events[0]
	if event.Agent != "claude" || event.Class != journalCheck || !event.Failed || event.Sig != commandSig("go test ./...") {
		t.Fatalf("event = %+v", event)
	}
}
