package cully

import (
	"strings"
	"testing"
	"time"
)

func writeSession(t *testing.T, cwd, session, spec string) []journalEvent {
	t.Helper()
	start := time.Now().UTC().Add(-time.Duration(len(session)) * time.Hour)
	var events []journalEvent
	for i, c := range spec {
		ev := journalEvent{Time: start.Add(time.Duration(i) * time.Second)}
		switch c {
		case 'E':
			ev.Class = journalEdit
		case 'T':
			ev.Class = journalCheck
		case 'S':
			ev.Class = journalSearch
		}
		events = append(events, ev)
		appendJournal(cwd, session, ev)
	}
	return events
}

func TestWorkflowHintFires(t *testing.T) {
	cwd := journalTestEnv(t)
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		writeSession(t, cwd, s, "SET T")
	}
	writeSession(t, cwd, "f", "ES") // one unchecked session: 5/6 = 83%
	current := writeSession(t, cwd, "current-session", "SE")
	got := workflowHints(cwd, current)
	if len(got) != 1 || !strings.HasPrefix(got[0], "ADV|You usually run checks after editing") {
		t.Fatalf("hints = %v", got)
	}
}

func TestWorkflowHintSilentCases(t *testing.T) {
	t.Run("no data", func(t *testing.T) {
		cwd := journalTestEnv(t)
		if got := workflowHints(cwd, mkEvents("E")); got != nil {
			t.Fatalf("hints = %v", got)
		}
	})
	t.Run("too few sessions", func(t *testing.T) {
		cwd := journalTestEnv(t)
		writeSession(t, cwd, "a", "ET")
		writeSession(t, cwd, "b", "ET")
		if got := workflowHints(cwd, mkEvents("E")); got != nil {
			t.Fatalf("hints = %v", got)
		}
	})
	t.Run("sessions without edits are ignored", func(t *testing.T) {
		cwd := journalTestEnv(t)
		writeSession(t, cwd, "a", "ET")
		writeSession(t, cwd, "b", "ET")
		writeSession(t, cwd, "c", "SSS")
		writeSession(t, cwd, "d", "SSS")
		if got := workflowHints(cwd, mkEvents("E")); got != nil {
			t.Fatalf("hints = %v", got)
		}
	})
	t.Run("share below 80 percent", func(t *testing.T) {
		cwd := journalTestEnv(t)
		for _, s := range []string{"a", "b", "c"} {
			writeSession(t, cwd, s, "ET")
		}
		writeSession(t, cwd, "d", "E") // 3/4 = 75%
		if got := workflowHints(cwd, mkEvents("E")); got != nil {
			t.Fatalf("hints = %v", got)
		}
	})
	t.Run("current session already checked or has no edits", func(t *testing.T) {
		cwd := journalTestEnv(t)
		for _, s := range []string{"a", "b", "c"} {
			writeSession(t, cwd, s, "ET")
		}
		for _, spec := range []string{"EP", "SSS", "EPS"} {
			if got := workflowHints(cwd, mkEvents(spec)); got != nil {
				t.Fatalf("%q hints = %v", spec, got)
			}
		}
	})
}

func TestWorkflowHintInCombinedAdvice(t *testing.T) {
	cwd := journalTestEnv(t)
	t.Chdir(cwd)
	for _, s := range []string{"a", "b", "c"} {
		writeSession(t, currentDir(), s, "ET")
	}
	writeSession(t, currentDir(), "now", "E")
	got := strings.Join(combinedAdvice("now", toolStats{Tools: 1}, sessionView{}), "\n")
	if !strings.Contains(got, "You usually run checks after editing") {
		t.Fatalf("advice = %s", got)
	}
}
