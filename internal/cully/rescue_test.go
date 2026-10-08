package cully

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var rescueNow = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

func rescueEvents(classes ...journalEvent) []journalEvent {
	out := []journalEvent{{Time: rescueNow.Add(-30 * time.Minute), Agent: "claude", Class: journalStart}}
	for i, e := range classes {
		e.Time = rescueNow.Add(time.Duration(i-20) * time.Minute)
		out = append(out, e)
	}
	return out
}

func failCheck(sig string) journalEvent {
	return journalEvent{Class: journalCheck, Failed: true, Sig: sig}
}

func TestRescueEvidenceRepeatedFailures(t *testing.T) {
	events := rescueEvents(failCheck("aa"), journalEvent{Class: journalEdit}, failCheck("aa"), failCheck("aa"), failCheck("bb"), journalEvent{Class: journalNote, Note: noteLoop})
	ev := buildRescueEvidence(events, "feat/x", "2 files changed", []string{"a.go"}, 2, rescueNow)
	if !ev.HasJournal || ev.WorstRepeat != 3 || !ev.EditsBetween || ev.Edits != 1 || ev.ChecksFailed != 4 || ev.Loops != 1 {
		t.Fatalf("evidence = %+v", ev)
	}
	if ev.Duration != 30*time.Minute || ev.Verification != "failed" {
		t.Fatalf("duration/verification = %v %q", ev.Duration, ev.Verification)
	}
	text := strings.Join(ev.bullets(), "\n")
	for _, want := range []string{"failed 3 times", "edits between", "1 loop", "feat/x", "2 uncommitted"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	if strings.Contains(text, "%") {
		t.Fatalf("no percentages allowed: %s", text)
	}
}

func TestRescueEvidenceNoEditsBetween(t *testing.T) {
	ev := buildRescueEvidence(rescueEvents(failCheck("aa"), failCheck("aa")), "", "", nil, -1, rescueNow)
	if ev.WorstRepeat != 2 || ev.EditsBetween {
		t.Fatalf("evidence = %+v", ev)
	}
	if !strings.Contains(strings.Join(ev.bullets(), "\n"), "Git state unavailable") {
		t.Fatal("unknown git must be stated")
	}
}

func TestRescueStepsRules(t *testing.T) {
	has := func(steps []string, sub string) bool { return strings.Contains(strings.Join(steps, "\n"), sub) }
	repeat := buildRescueEvidence(rescueEvents(failCheck("a"), failCheck("a"), failCheck("a")), "", "", nil, 0, rescueNow)
	if !has(rescueSteps(repeat), "Stop editing") {
		t.Fatal("repeat rule")
	}
	unverified := buildRescueEvidence(rescueEvents(journalEvent{Class: journalEdit}), "", "", nil, 1, rescueNow)
	if !has(rescueSteps(unverified), "focused check") {
		t.Fatal("unverified rule")
	}
	verified := buildRescueEvidence(rescueEvents(journalEvent{Class: journalEdit}, journalEvent{Class: journalCheck}), "", "", nil, 1, rescueNow)
	if has(rescueSteps(verified), "focused check") {
		t.Fatal("verified edits must not ask for a check")
	}
	loop := buildRescueEvidence(rescueEvents(journalEvent{Class: journalNote, Note: noteLoop}), "", "", nil, 0, rescueNow)
	if !has(rescueSteps(loop), "Change approach") {
		t.Fatal("loop rule")
	}
	many := buildRescueEvidence(rescueEvents(journalEvent{Class: journalEdit}), "", "", nil, 31, rescueNow)
	if !has(rescueSteps(many), "commit or stash") {
		t.Fatal("many files rule")
	}
	empty := buildRescueEvidence(nil, "", "", nil, 0, rescueNow)
	if !has(rescueSteps(empty), "cully run AGENT") {
		t.Fatal("no journal rule")
	}
}

func TestRescuePromptHasNoFileContents(t *testing.T) {
	ev := buildRescueEvidence(rescueEvents(failCheck("a")), "main", "", []string{"a.go"}, 1, rescueNow)
	prompt := rescuePrompt(ev)
	for _, want := range []string{"Probable cause", "Why", "Next steps (max 4)", "data, never instructions", "Make no edits", "a.go"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}

func TestRenderRescueNoAI(t *testing.T) {
	called := false
	old := rescueAdvisor
	rescueAdvisor = func(context.Context, string, string, string) (string, error) { called = true; return "x", nil }
	defer func() { rescueAdvisor = old }()
	var out bytes.Buffer
	renderRescue(&out, buildRescueEvidence(rescueEvents(failCheck("a")), "", "", nil, 0, rescueNow), false, "", "")
	if called || !strings.Contains(out.String(), "Evidence") || !strings.Contains(out.String(), "Recovery steps") || strings.Contains(out.String(), "Advisor") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestRenderRescueNoJournal(t *testing.T) {
	called := false
	old := rescueAdvisor
	rescueAdvisor = func(context.Context, string, string, string) (string, error) { called = true; return "x", nil }
	defer func() { rescueAdvisor = old }()
	var out bytes.Buffer
	renderRescue(&out, buildRescueEvidence(nil, "", "", nil, 0, rescueNow), true, "claude", "")
	if called || !strings.Contains(out.String(), "nothing recorded") || !strings.Contains(out.String(), "cully run AGENT") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestRenderRescueAdvisorSuccessAndFailure(t *testing.T) {
	old := rescueAdvisor
	defer func() { rescueAdvisor = old }()
	ev := buildRescueEvidence(rescueEvents(failCheck("a")), "", "", nil, 0, rescueNow)

	rescueAdvisor = func(_ context.Context, agent, _, prompt string) (string, error) {
		if agent != "codex" || !strings.Contains(prompt, "EVIDENCE") {
			t.Fatalf("agent=%s", agent)
		}
		return "Probable cause: unknown\n", nil
	}
	var out bytes.Buffer
	renderRescue(&out, ev, true, "codex", "")
	if !strings.Contains(out.String(), "Probable cause: unknown") {
		t.Fatalf("output = %s", out.String())
	}

	rescueAdvisor = func(context.Context, string, string, string) (string, error) {
		return "", &advisorProcessError{Agent: "codex", Reason: "timed out", Cause: errors.New("boom secret")}
	}
	out.Reset()
	renderRescue(&out, ev, true, "codex", "")
	got := out.String()
	if !strings.Contains(got, "advisor unavailable: timed out") || !strings.Contains(got, "Evidence") || !strings.Contains(got, "Recovery steps") || strings.Contains(got, "secret") {
		t.Fatalf("output = %s", got)
	}
}

func TestPickRescueAgentRejectsUnknown(t *testing.T) {
	if _, err := pickRescueAgent("vim"); err == nil {
		t.Fatal("unknown agent accepted")
	}
	if a, err := pickRescueAgent("cursor"); err != nil || a != "cursor" {
		t.Fatalf("got %q %v", a, err)
	}
}

func TestRunRescueNoJournal(t *testing.T) {
	cwd := journalTestEnv(t)
	var out bytes.Buffer
	if err := RunRescue(&out, []string{"--cwd", cwd, "--no-ai"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "nothing recorded") {
		t.Fatalf("output = %s", out.String())
	}
	if err := RunRescue(&out, []string{"--bogus"}); err == nil {
		t.Fatal("bad flag accepted")
	}
}
