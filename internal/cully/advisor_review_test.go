package cully

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdvisorReviewPreservesInstructionsAndRejectsStalePreview(t *testing.T) {
	cwd := t.TempDir()
	path := sharedSkillPath(cwd, "cully")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte("User-owned rules\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	review, err := localAdvisorReview(cwd, "ADV|Add verification skill", applyPlan{Summary: "Add verification guidance", ClaudeMDSection: "Run a focused check after edits."})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if string(before) != string(original) {
		t.Fatal("preview changed files")
	}
	if !strings.Contains(review.Detail, "Run a focused check") {
		t.Fatal(review)
	}
	changed := []byte("User modified this after preview\n")
	os.WriteFile(path, changed, 0o600)
	if err := acceptAdvisorReview(cwd, review); err == nil {
		t.Fatal("stale preview accepted")
	}
	os.WriteFile(path, original, 0o600)
	if err := acceptAdvisorReview(cwd, review); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(after), string(original)) || !strings.Contains(string(after), "Run a focused check") {
		t.Fatal(string(after))
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatal("permissions changed")
	}
}

func TestAdvisorReviewDefersCommandsAndPreservesUserSkills(t *testing.T) {
	cwd := t.TempDir()
	plans := []applyPlan{
		{ShellCommands: []string{"touch should-not-run"}},
		{MCPServers: map[string]any{"example": map[string]any{"command": "example"}}},
		{SkillName: "../../escape", SkillContent: "unsafe"},
		{ClaudeMDSection: "rule", SkillName: "new-skill", SkillContent: "skill"},
	}
	for _, plan := range plans {
		if _, err := localAdvisorReview(cwd, "ADV|test", plan); err == nil {
			t.Fatal("unsafe/unsupported plan accepted", plan)
		}
	}
	path := sharedSkillPath(cwd, "existing")
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("mine"), 0o644)
	if _, err := localAdvisorReview(cwd, "ADV|skill", applyPlan{SkillName: "existing", SkillContent: "replacement"}); err == nil {
		t.Fatal("user skill overwritten")
	}
	outside := t.TempDir()
	os.Symlink(outside, filepath.Join(cwd, ".cully", "skills", "linked"))
	if _, err := localAdvisorReview(cwd, "ADV|skill", applyPlan{SkillName: "linked", SkillContent: "replacement"}); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := os.Stat(filepath.Join(cwd, "should-not-run")); !os.IsNotExist(err) {
		t.Fatal("command ran")
	}
}

func TestAdvisorReviewCreatesOnlyPreviewedSkill(t *testing.T) {
	cwd := t.TempDir()
	review, err := localAdvisorReview(cwd, "ADV|Create a project skill", applyPlan{Summary: "Create check skill", SkillName: "focused-check", SkillContent: "# Focused check\nRun relevant tests."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(review.Path); !os.IsNotExist(err) {
		t.Fatal("preview wrote file")
	}
	if err := acceptAdvisorReview(cwd, review); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(review.Path)
	if string(got) != "# Focused check\nRun relevant tests.\n" {
		t.Fatal(string(got))
	}
	if err := acceptAdvisorReview(cwd, review); err == nil {
		t.Fatal("repeated creation accepted")
	}
}
