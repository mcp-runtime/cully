package cully

import (
	"strings"
	"testing"
	"time"
)

var loopBase = time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

// mkEvents builds events one minute apart from a compact spec: E edit,
// F failed check (sig "a"), G failed check (sig "b"), P passing check (sig "a"), S search.
func mkEvents(spec string) []journalEvent {
	var out []journalEvent
	for i, c := range spec {
		ev := journalEvent{Time: loopBase.Add(time.Duration(i) * time.Minute)}
		switch c {
		case 'E':
			ev.Class = journalEdit
		case 'F':
			ev.Class, ev.Failed, ev.Sig = journalCheck, true, "a"
		case 'G':
			ev.Class, ev.Failed, ev.Sig = journalCheck, true, "b"
		case 'P':
			ev.Class, ev.Sig = journalCheck, "a"
		case 'S':
			ev.Class = journalSearch
		}
		out = append(out, ev)
	}
	return out
}

func findLoop(r loopReport, kind string) (loopFinding, bool) {
	for _, l := range r.Loops {
		if l.Kind == kind {
			return l, true
		}
	}
	return loopFinding{}, false
}

func TestRepeatFailureDetected(t *testing.T) {
	r := detectLoops(mkEvents("FEFEF"))
	f, ok := findLoop(r, loopRepeatFailure)
	if !ok || f.Attempts != 3 || !f.EditBetween {
		t.Fatalf("report = %+v", r)
	}
	if !f.First.Equal(loopBase) || !f.Last.Equal(loopBase.Add(4*time.Minute)) {
		t.Fatalf("times = %v %v", f.First, f.Last)
	}
	if !strings.Contains(r.Advice()[0], "failed 3 times after edits") {
		t.Fatalf("advice = %v", r.Advice())
	}
}

func TestRepeatFailureWithoutEdits(t *testing.T) {
	f, ok := findLoop(detectLoops(mkEvents("FFF")), loopRepeatFailure)
	if !ok || f.EditBetween {
		t.Fatalf("finding = %+v %v", f, ok)
	}
}

func TestBelowThresholdAndDifferentSigs(t *testing.T) {
	for _, spec := range []string{"", "FF", "FGF", "EFEF"} {
		if r := detectLoops(mkEvents(spec)); r.Active() {
			t.Fatalf("%q reported %+v", spec, r)
		}
	}
}

func TestPassingCheckResetsRepeatFailure(t *testing.T) {
	if r := detectLoops(mkEvents("FFPF")); r.Active() {
		t.Fatalf("reset ignored: %+v", r)
	}
	// A passing check of a different command does not reset this one.
	events := mkEvents("FFF")
	events = append(events, journalEvent{Time: loopBase.Add(10 * time.Minute), Class: journalCheck, Sig: "other"})
	if _, ok := findLoop(detectLoops(events), loopRepeatFailure); !ok {
		t.Fatal("unrelated passing check reset the loop")
	}
}

func TestRepeatFailureWindowExpiry(t *testing.T) {
	// Three failures 20 minutes apart and more than 20 events apart: no loop.
	var events []journalEvent
	for i := 0; i < 3; i++ {
		events = append(events, journalEvent{Time: loopBase.Add(time.Duration(i) * time.Hour), Class: journalCheck, Failed: true, Sig: "a"})
		for j := 0; j < loopWindowEvents; j++ {
			events = append(events, journalEvent{Time: loopBase.Add(time.Duration(i)*time.Hour + time.Second), Class: journalSearch})
		}
	}
	if r := detectLoops(events); r.Active() {
		t.Fatalf("expired window reported %+v", r)
	}
	// Within the event window but beyond 15 minutes still counts.
	slow := mkEvents("FFF")
	for i := range slow {
		slow[i].Time = loopBase.Add(time.Duration(i) * 40 * time.Minute)
	}
	if _, ok := findLoop(detectLoops(slow), loopRepeatFailure); !ok {
		t.Fatal("failures within 20 events must count")
	}
	// Within 15 minutes but beyond 20 events still counts.
	fast := []journalEvent{{Time: loopBase, Class: journalCheck, Failed: true, Sig: "a"}}
	for j := 0; j < 25; j++ {
		fast = append(fast, journalEvent{Time: loopBase.Add(time.Minute), Class: journalSearch})
	}
	for i := 0; i < 2; i++ {
		fast = append(fast, journalEvent{Time: loopBase.Add(2 * time.Minute), Class: journalCheck, Failed: true, Sig: "a"})
	}
	if _, ok := findLoop(detectLoops(fast), loopRepeatFailure); !ok {
		t.Fatal("failures within 15 minutes must count")
	}
}

func TestEditCycleDetected(t *testing.T) {
	r := detectLoops(mkEvents("EGEFEG"))
	f, ok := findLoop(r, loopEditCycle)
	if !ok || f.Attempts != 3 || !f.EditBetween {
		t.Fatalf("report = %+v", r)
	}
	if !strings.Contains(r.Advice()[0], "3 edits were each followed by a failed check") {
		t.Fatalf("advice = %v", r.Advice())
	}
}

func TestEditCycleNeedsEditsAndResets(t *testing.T) {
	for _, spec := range []string{"GGG", "EGEG", "EGEGPEG", "EGEGEPEG"} {
		if _, ok := findLoop(detectLoops(mkEvents(spec)), loopEditCycle); ok {
			t.Fatalf("%q reported a cycle", spec)
		}
	}
	if _, ok := findLoop(detectLoops(mkEvents("EGPEGEGEG")), loopEditCycle); !ok {
		t.Fatal("cycles after a reset must count")
	}
}

func TestLoopNoteWrittenOnce(t *testing.T) {
	cwd := journalTestEnv(t)
	for _, c := range "FEFEF" {
		ev := mkEvents(string(c))[0]
		ev.Time = time.Now().UTC()
		appendJournal(cwd, "s", ev)
	}
	for i := 0; i < 3; i++ {
		if r, _ := sessionLoops(cwd, "s"); !r.Active() {
			t.Fatal("loop not detected")
		}
	}
	notes := 0
	for _, ev := range readJournal(journalPath(cwd, "s")) {
		if ev.Class == journalNote && ev.Note == noteLoop {
			notes++
		}
	}
	if notes != 1 {
		t.Fatalf("loop notes = %d, want 1", notes)
	}
	// A passing check resets, and a later loop is noted again.
	appendJournal(cwd, "s", journalEvent{Class: journalCheck, Sig: "a"})
	time.Sleep(2 * time.Millisecond)
	for i := 0; i < 3; i++ {
		appendJournal(cwd, "s", journalEvent{Class: journalCheck, Failed: true, Sig: "a"})
	}
	sessionLoops(cwd, "s")
	notes = 0
	for _, ev := range readJournal(journalPath(cwd, "s")) {
		if ev.Class == journalNote && ev.Note == noteLoop {
			notes++
		}
	}
	if notes != 2 {
		t.Fatalf("loop notes after reset = %d, want 2", notes)
	}
}

func TestCombinedAdviceContainsLoopLine(t *testing.T) {
	cwd := journalTestEnv(t)
	t.Chdir(cwd)
	session := "loopy"
	for _, c := range "FEFEF" {
		ev := mkEvents(string(c))[0]
		ev.Time = time.Now().UTC()
		appendJournal(currentDir(), session, ev)
	}
	got := strings.Join(combinedAdvice(session, toolStats{Tools: 5}, sessionView{}), "\n")
	if !strings.Contains(got, "CAUT|Cully noticed something: the same command failed 3 times after edits.") {
		t.Fatalf("advice = %s", got)
	}
	view := sessionView{Loops: detectLoops(mkEvents("FEFEF"))}
	rows := strings.Join(sessionStatusRows(120, nil, toolStats{}, view), "\n")
	if !strings.Contains(rows, "Loops") {
		t.Fatal("Loops row missing")
	}
	if strings.Contains(strings.Join(sessionStatusRows(120, nil, toolStats{}, sessionView{}), "\n"), "Loops") {
		t.Fatal("Loops row shown without data")
	}
}

func TestLoopCountMatchesDetectionOutsideTheTerminal(t *testing.T) {
	now := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	var events []journalEvent
	for i := 0; i < loopMinAttempts; i++ {
		events = append(events, journalEvent{Time: now.Add(time.Duration(i) * time.Minute), Class: journalCheck, Failed: true, Sig: "abc"})
	}
	if got := loopCount(events); got != 1 {
		t.Fatalf("loopCount without notes = %d, want 1", got)
	}
	noted := append(append([]journalEvent{}, events...),
		journalEvent{Time: now, Class: journalNote, Note: noteLoop},
		journalEvent{Time: now, Class: journalNote, Note: noteLoop})
	if got := loopCount(noted); got != 2 {
		t.Fatalf("loopCount with two notes = %d, want 2", got)
	}
	if got := loopCount(nil); got != 0 {
		t.Fatalf("loopCount(nil) = %d", got)
	}
}
