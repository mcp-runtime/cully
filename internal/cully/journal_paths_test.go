package cully

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func makeToolEvent(cwd, name string, input any) toolEvent {
	raw, _ := json.Marshal(input)
	return toolEvent{Cwd: cwd, ToolName: name, ToolInput: raw}
}

func TestRelProjectPath(t *testing.T) {
	cwd := t.TempDir()
	cases := []struct {
		name, in, want string
	}{
		{"relative", "internal/a.go", "internal/a.go"},
		{"dot prefix", "./a/../b.go", "b.go"},
		{"absolute inside", filepath.Join(cwd, "x", "y.go"), "x/y.go"},
		{"absolute outside", "/etc/passwd", ""},
		{"escape", "../secret", ""},
		{"deep escape", "a/../../secret", ""},
		{"root itself", ".", ""},
		{"empty", "", ""},
		{"double dot file name", "..foo", "..foo"},
		{"control char", "a\x00b", ""},
		{"newline", "a\nb", ""},
		{"too long", strings.Repeat("a", 201), ""},
		{"huge", strings.Repeat("a/", 5000), ""},
		{"url", "https://hooks.example.com/services/T00/secret", ""},
		{"collapsed url", "https:/hooks.example.com/secret", ""},
		{"file url", "file:///etc/passwd", ""},
		{"spaces", "my dir/my file.txt", "my dir/my file.txt"},
		{"metachar name", "a;rm -rf $(x).go", "a;rm -rf $(x).go"},
	}
	for _, c := range cases {
		if got := relProjectPath(cwd, c.in); got != c.want {
			t.Errorf("%s: relProjectPath(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
	if got := relProjectPath("", "a"); got != "" {
		t.Errorf("empty cwd = %q", got)
	}
}

func TestExtractToolActivity(t *testing.T) {
	cwd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cwd, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(cwd, "abs.go")
	patch := "*** Begin Patch\n*** Add File: new.go\n+package x\n+*** Delete File: fake.go\n*** Update File: old.go\n*** Move to: moved.go\n@@\n-a\n+b\n*** Update File: keep.go\n*** Delete File: gone.go\n*** End Patch"
	cases := []struct {
		name  string
		event toolEvent
		ops   []fileOp
		cmd   string
	}{
		{"Read", makeToolEvent(cwd, "Read", map[string]string{"file_path": "a.go"}), []fileOp{{Path: "a.go", Op: "read"}}, ""},
		{"Read absolute", makeToolEvent(cwd, "Read", map[string]string{"file_path": abs}), []fileOp{{Path: "abs.go", Op: "read"}}, ""},
		{"Read outside", makeToolEvent(cwd, "Read", map[string]string{"file_path": "/etc/hosts"}), nil, ""},
		{"Read escape", makeToolEvent(cwd, "Read", map[string]string{"file_path": "../x"}), nil, ""},
		{"Write", makeToolEvent(cwd, "Write", map[string]string{"file_path": "w.go", "content": "SECRET"}), []fileOp{{Path: "w.go", Op: "write"}}, ""},
		{"CreateFile", makeToolEvent(cwd, "Create_File", map[string]string{"file_path": "new.go"}), []fileOp{{Path: "new.go", Op: "write"}}, ""},
		{"Edit", makeToolEvent(cwd, "Edit", map[string]string{"file_path": "e.go", "old_string": "x", "new_string": "SECRET"}), []fileOp{{Path: "e.go", Op: "edit"}}, ""},
		{"MultiEdit", makeToolEvent(cwd, "MultiEdit", map[string]string{"file_path": "m.go"}), []fileOp{{Path: "m.go", Op: "edit"}}, ""},
		{"NotebookEdit", makeToolEvent(cwd, "NotebookEdit", map[string]string{"notebook_path": "n.ipynb"}), []fileOp{{Path: "n.ipynb", Op: "edit"}}, ""},
		{"Glob", makeToolEvent(cwd, "Glob", map[string]string{"pattern": "**/*.go", "path": "src"}), nil, ""},
		{"Grep", makeToolEvent(cwd, "Grep", map[string]string{"pattern": "x"}), nil, ""},
		{"mcp namespaced", makeToolEvent(cwd, "mcp__fs__Read", map[string]string{"file_path": "a.go"}), []fileOp{{Path: "a.go", Op: "read"}}, ""},
		{"cursor delete", makeToolEvent(cwd, "Delete", map[string]string{"path": "d.go"}), []fileOp{{Path: "d.go", Op: "delete"}}, ""},
		{"apply_patch", makeToolEvent(cwd, "apply_patch", map[string]string{"input": patch}), []fileOp{
			{Path: "new.go", Op: "create"}, {Path: "old.go", Op: "move", To: "moved.go"},
			{Path: "keep.go", Op: "edit"}, {Path: "gone.go", Op: "delete"}}, "apply_patch"},
		{"patch via shell", makeToolEvent(cwd, "exec_command", map[string]string{"cmd": "apply_patch <<'EOF'\n" + patch + "\nEOF"}), nil, "apply_patch"},
		{"patch escape", makeToolEvent(cwd, "apply_patch", map[string]string{"input": "*** Add File: /etc/x\n*** Update File: ../y"}), nil, "apply_patch"},
		{"rm", makeToolEvent(cwd, "Bash", map[string]string{"command": "rm -rf a.go b.go"}), nil, "rm"},
		{"rm glob", makeToolEvent(cwd, "Bash", map[string]string{"command": "rm *.tmp 'x y.txt'"}), nil, "rm"},
		{"rm variable", makeToolEvent(cwd, "Bash", map[string]string{"command": `rm "$HOME/a" ~/b $X`}), nil, "rm"},
		{"rm absolute outside", makeToolEvent(cwd, "Bash", map[string]string{"command": "rm /etc/passwd"}), nil, "rm"},
		{"rm quoted spaces", makeToolEvent(cwd, "Bash", map[string]string{"command": `rm "my dir/my file.txt"`}), nil, "rm"},
		{"rm double dash", makeToolEvent(cwd, "Bash", map[string]string{"command": "rm -- -weird"}), nil, "rm"},
		{"unlink", makeToolEvent(cwd, "Bash", map[string]string{"command": "unlink a.go"}), nil, "unlink"},
		{"git rm", makeToolEvent(cwd, "Bash", map[string]string{"command": "git rm a.go"}), nil, "git rm"},
		{"git rm cached", makeToolEvent(cwd, "Bash", map[string]string{"command": "git rm --cached a.go"}), nil, "git rm"},
		{"mv", makeToolEvent(cwd, "Bash", map[string]string{"command": "mv a.go b.go"}), nil, "mv"},
		{"git mv", makeToolEvent(cwd, "Bash", map[string]string{"command": "git mv a.go b.go"}), nil, "git mv"},
		{"mv many", makeToolEvent(cwd, "Bash", map[string]string{"command": "mv a b c"}), nil, "mv"},
		{"mv outside", makeToolEvent(cwd, "Bash", map[string]string{"command": "mv a.go /tmp/b.go"}), nil, "mv"},
		{"cat", makeToolEvent(cwd, "Bash", map[string]string{"command": "cat a.go b.go"}), nil, "cat"},
		{"cat url", makeToolEvent(cwd, "Bash", map[string]string{"command": "cat https://example.com/x?token=1"}), nil, "cat"},
		{"head -n", makeToolEvent(cwd, "Bash", map[string]string{"command": "head -n 5 a.go"}), nil, "head"},
		{"sed -n", makeToolEvent(cwd, "Bash", map[string]string{"command": "sed -n '1,10p' a.go"}), nil, "sed"},
		{"sed -i unsure", makeToolEvent(cwd, "Bash", map[string]string{"command": "sed -i s/a/b/ a.go"}), nil, "sed"},
		{"rg pattern path", makeToolEvent(cwd, "Bash", map[string]string{"command": "rg foo a.go"}), nil, "rg"},
		{"rg dir", makeToolEvent(cwd, "Bash", map[string]string{"command": "rg foo src"}), nil, "rg"},
		{"rg value flag", makeToolEvent(cwd, "Bash", map[string]string{"command": "rg -g '*.go' foo a.go"}), nil, "rg"},
		{"grep -rn", makeToolEvent(cwd, "Bash", map[string]string{"command": "grep -rn foo a.go"}), nil, "grep"},
		{"touch", makeToolEvent(cwd, "Bash", map[string]string{"command": "touch a.go"}), nil, "touch"},
		{"tee", makeToolEvent(cwd, "Bash", map[string]string{"command": "echo hi | tee out.txt"}), nil, "echo"},
		{"redirect", makeToolEvent(cwd, "Bash", map[string]string{"command": "echo hi > out.txt"}), nil, "echo"},
		{"pipeline prefers go", makeToolEvent(cwd, "Bash", map[string]string{"command": "echo hi | go test ./..."}), nil, "go test"},
		{"append and stderr", makeToolEvent(cwd, "Bash", map[string]string{"command": "go test ./... >> log.txt 2>&1"}), nil, "go test"},
		{"fd redirect file", makeToolEvent(cwd, "Bash", map[string]string{"command": "go build 2>err.txt"}), nil, "go build"},
		{"heredoc unsure", makeToolEvent(cwd, "Bash", map[string]string{"command": "cat > a.go <<EOF\nrm b.go\nEOF"}), nil, "cat"},
		{"cd unsure", makeToolEvent(cwd, "Bash", map[string]string{"command": "cd sub && rm a.go"}), nil, "rm"},
		{"subshell unsure", makeToolEvent(cwd, "Bash", map[string]string{"command": "rm $(echo a.go)"}), nil, "rm"},
		{"quoted rm text", makeToolEvent(cwd, "Bash", map[string]string{"command": `echo "rm a.go"`}), nil, "echo"},
		{"chain", makeToolEvent(cwd, "Bash", map[string]string{"command": "rm a.go; mv b.go c.go && cat d.go"}), nil, "rm"},
		{"env prefix", makeToolEvent(cwd, "Bash", map[string]string{"command": "TOKEN=abc123 go test ./..."}), nil, "go test"},
		{"sudo", makeToolEvent(cwd, "Bash", map[string]string{"command": "sudo rm a.go"}), nil, "rm"},
		{"git commit args dropped", makeToolEvent(cwd, "Bash", map[string]string{"command": `git commit -m "secret message"`}), nil, "git commit"},
		{"git flag first", makeToolEvent(cwd, "Bash", map[string]string{"command": "git -C /x status"}), nil, "git"},
		{"npm install", makeToolEvent(cwd, "Bash", map[string]string{"command": "npm install left-pad"}), nil, "npm install"},
		{"make target omitted", makeToolEvent(cwd, "Bash", map[string]string{"command": "make customer_secret"}), nil, "make"},
		{"unknown git verb omitted", makeToolEvent(cwd, "Bash", map[string]string{"command": "git customer_secret"}), nil, "git"},
		{"npm script omitted", makeToolEvent(cwd, "Bash", map[string]string{"command": "npm customer_secret"}), nil, "npm"},
		{"url never", makeToolEvent(cwd, "Bash", map[string]string{"command": "curl https://example.com/x?token=1"}), nil, "curl"},
		{"program with weird chars", makeToolEvent(cwd, "Bash", map[string]string{"command": "$(evil) x"}), nil, ""},
		{"script path", makeToolEvent(cwd, "Bash", map[string]string{"command": "./scripts/run.sh --flag"}), nil, "run.sh"},
		{"huge command", makeToolEvent(cwd, "Bash", map[string]string{"command": "rm " + strings.Repeat("a", 20000)}), nil, ""},
		{"codex argv", makeToolEvent(cwd, "shell", map[string]any{"command": []string{"go", "test", "./..."}}), nil, "go test"},
		{"exec_command cmd", makeToolEvent(cwd, "exec_command", map[string]string{"cmd": "rm a.go"}), nil, "rm"},
		{"unknown tool", makeToolEvent(cwd, "WebFetch", map[string]string{"url": "https://x"}), nil, ""},
		{"bad input", toolEvent{Cwd: cwd, ToolName: "Read", ToolInput: json.RawMessage(`"oops"`)}, nil, ""},
	}
	for _, c := range cases {
		ops, cmd := extractToolActivity(c.event)
		if len(ops) == 0 {
			ops = nil
		}
		if !reflect.DeepEqual(ops, c.ops) || cmd != c.cmd {
			t.Errorf("%s: got ops=%+v cmd=%q, want ops=%+v cmd=%q", c.name, ops, cmd, c.ops, c.cmd)
		}
	}
}

func TestExtractedCmdAlwaysMatchesPattern(t *testing.T) {
	cwd := t.TempDir()
	hostile := []string{"git 'a b'", "go -x", "rm; cat", "ls $(pwd)", "`id`", "a\x00b c", "git status\nrm x", strings.Repeat("x", 300) + " y"}
	for _, h := range hostile {
		_, cmd := extractToolActivity(makeToolEvent(cwd, "Bash", map[string]string{"command": h}))
		if cmd != "" && !journalCmdPattern.MatchString(cmd) {
			t.Errorf("%q produced invalid label %q", h, cmd)
		}
	}
}

func TestRecordJournalToolStoresPathsAndHonorsSwitch(t *testing.T) {
	cwd := journalTestEnv(t)
	patch := "*** Add File: a.go\n+SECRET BODY\n*** Update File: b.go\n"
	recordJournalTool("codex", "s1", makeToolEvent(cwd, "apply_patch", map[string]string{"input": patch}), 'E', false)
	recordJournalTool("claude", "s1", makeToolEvent(cwd, "Bash", map[string]string{"command": "go test ./secret/..."}), 'T', true)
	file, events, _ := latestJournal(cwd)
	if len(events) != 3 || events[0].Path != "a.go" || events[0].Op != "create" || events[1].Class != journalFileOp || events[1].Path != "b.go" {
		t.Fatalf("events = %+v", events)
	}
	if events[2].Cmd != "go test" || !events[2].Failed || events[2].Sig == "" {
		t.Fatalf("check event = %+v", events[2])
	}
	raw, _ := os.ReadFile(file.Path)
	for _, leak := range []string{"SECRET", "secret", cwd} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("journal leaked %q: %s", leak, raw)
		}
	}

	t.Setenv("CULLY_JOURNAL_PATHS", "0")
	recordJournalTool("claude", "s2", makeToolEvent(cwd, "Edit", map[string]string{"file_path": "z.go"}), 'E', false)
	recordJournalTool("claude", "s2", makeToolEvent(cwd, "Bash", map[string]string{"command": "go test"}), 'T', false)
	for _, f := range projectJournals(cwd) {
		if f.Session != "s2" {
			continue
		}
		evs := readJournal(f.Path)
		if len(evs) != 2 || evs[0].Class != journalEdit || evs[1].Class != journalCheck {
			t.Fatalf("disabled events = %+v", evs)
		}
		for _, e := range evs {
			if e.Path != "" || e.Op != "" || e.Cmd != "" || e.To != "" {
				t.Fatalf("paths recorded while disabled: %+v", e)
			}
		}
	}
}

func TestOldJournalLinesStillParse(t *testing.T) {
	cwd := journalTestEnv(t)
	if err := os.MkdirAll(journalDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	line := `{"t":"2026-10-01T10:00:00Z","a":"claude","c":"E"}` + "\n"
	if err := os.WriteFile(journalPath(cwd, "old"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	if ev := readJournal(journalPath(cwd, "old")); len(ev) != 1 || ev[0].Path != "" {
		t.Fatalf("events = %+v", ev)
	}
}

func TestCursorEventsCarryPaths(t *testing.T) {
	cwd := t.TempDir()
	ev, ok := cursorToolEvent([]byte(`{"hook_event_name":"afterFileEdit","cwd":` + jsonQuote(cwd) + `,"file_path":` + jsonQuote(filepath.Join(cwd, "a/b.go")) + `}`))
	if !ok {
		t.Fatal("not decoded")
	}
	if ops, _ := extractToolActivity(ev); len(ops) != 1 || ops[0].Op != "edit" || ops[0].Path != "a/b.go" {
		t.Fatalf("ops = %+v", ops)
	}
	ev, _ = cursorToolEvent([]byte(`{"hook_event_name":"postToolUse","cwd":` + jsonQuote(cwd) + `,"tool_name":"Delete","tool_input":{"path":"gone.go"}}`))
	if ops, _ := extractToolActivity(ev); len(ops) != 1 || ops[0].Op != "delete" || toolClass(ev) != 'E' {
		t.Fatalf("delete ops = %+v class %c", ops, toolClass(ev))
	}
	ev, _ = cursorToolEvent([]byte(`{"hook_event_name":"afterShellExecution","cwd":` + jsonQuote(cwd) + `,"command":"rm old.go"}`))
	if ops, cmd := extractToolActivity(ev); len(ops) != 0 || cmd != "rm" {
		t.Fatalf("shell ops = %+v %q", ops, cmd)
	}
	ev, _ = cursorToolEvent([]byte(`{"hook_event_name":"afterShellExecution","cwd":` + jsonQuote(cwd) + `,"command":"cat https://example.com/hook?token=secret"}`))
	if ops, cmd := extractToolActivity(ev); len(ops) != 0 || cmd != "cat" || strings.Contains(cmd, "token") {
		t.Fatalf("url shell = %+v %q", ops, cmd)
	}
}

func jsonQuote(s string) string { b, _ := json.Marshal(s); return string(b) }
