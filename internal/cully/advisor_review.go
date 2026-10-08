package cully

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// A review holds the exact proposed bytes and baseline. Accept never asks a
// model to regenerate an approved change or runs a suggested shell command.
type advisorReview struct {
	Summary, Detail, Handoff, Path string
	Before, After                  []byte
	Existed                        bool
	Mode                           os.FileMode
}

func advisorHandoff(line string) advisorReview {
	text := stripSeverityPrefix(line)
	return advisorReview{Summary: "Ask the current coding agent", Detail: "Add this request to the current input. Review it there and press Enter to send.\n\n" + text, Handoff: "Please assess this Cully suggestion for the current task and apply it if appropriate: " + text}
}

func prepareAdvisorReview(agent, cwd, line string) (advisorReview, error) {
	return prepareAdvisorReviewContext(context.Background(), agent, cwd, line)
}

func prepareAdvisorReviewContext(ctx context.Context, agent, cwd, line string) (advisorReview, error) {
	fallback := advisorHandoff(line)
	prompt := applyInstr + "\n\nThis is a PREVIEW only. Do not edit files, run commands, install anything, or write memory. Return one JSON plan for this suggestion. Prefer one small shared project skill or instruction change. Actions requiring task knowledge should be notes only. Active agent: " + agent + "\nSUGGESTION:\n" + stripSeverityPrefix(line)
	out, err := runAdvisorAgentContext(ctx, agent, cwd, false, prompt)
	if err != nil {
		return fallback, fmt.Errorf("preview worker: %s", advisorFailureReason(err))
	}
	plan, err := parseApplyPlan(out)
	if err != nil {
		return fallback, fmt.Errorf("preview worker returned no usable plan")
	}
	review, err := localAdvisorReview(cwd, line, plan)
	if err != nil {
		fallback.Detail = "This action needs the coding agent: " + err.Error() + "\n\n" + fallback.Detail
		return fallback, nil
	}
	return review, nil
}

var advisorSkillName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func localAdvisorReview(cwd, line string, plan applyPlan) (advisorReview, error) {
	if len(plan.MCPServers) > 0 || len(plan.ShellCommands) > 0 {
		return advisorReview{}, fmt.Errorf("integration or command changes require an agent review")
	}
	if plan.ClaudeMDSection != "" && (plan.SkillName != "" || plan.SkillContent != "") {
		return advisorReview{}, fmt.Errorf("multiple changes require an agent review")
	}
	review := advisorReview{Summary: plan.Summary, Mode: 0o644}
	if plan.ClaudeMDSection != "" {
		review.Path = sharedSkillPath(cwd, "cully")
	} else if plan.SkillContent != "" && advisorSkillName.MatchString(plan.SkillName) && plan.SkillName != "cully" {
		review.Path = sharedSkillPath(cwd, plan.SkillName)
	} else {
		return advisorReview{}, fmt.Errorf("no supported local file change")
	}
	if err := advisorLocalPath(cwd, review.Path); err != nil {
		return advisorReview{}, err
	}
	before, err := os.ReadFile(review.Path)
	if err != nil && !os.IsNotExist(err) {
		return advisorReview{}, err
	}
	review.Before = before
	review.Existed = err == nil
	if review.Existed {
		info, err := os.Stat(review.Path)
		if err != nil {
			return advisorReview{}, err
		}
		review.Mode = info.Mode().Perm()
	}
	if plan.ClaudeMDSection != "" {
		marker := "<!-- " + suggestionMarker(line) + " -->"
		if strings.Contains(string(before), marker) {
			return advisorReview{}, fmt.Errorf("this rule is already present")
		}
		addition := marker + "\n" + strings.TrimSpace(plan.ClaudeMDSection) + "\n"
		review.After = append(bytes.Clone(before), []byte("\n\n"+addition)...)
		review.Detail = "Append to " + review.Path + "\n\n" + addition
	} else {
		if review.Existed {
			return advisorReview{}, fmt.Errorf("existing user-owned skill will be preserved")
		}
		review.After = []byte(strings.TrimSpace(plan.SkillContent) + "\n")
		review.Detail = "Create " + review.Path + "\n\n" + string(review.After)
	}
	if len(review.After) > 256*1024 {
		return advisorReview{}, fmt.Errorf("proposed change is too large")
	}
	if plan.Notes != "" {
		review.Detail += "\nFollow-up: " + plan.Notes
	}
	return review, nil
}

func advisorLocalPath(cwd, path string) error {
	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("change leaves the project")
	}
	cursor := root
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		cursor = filepath.Join(cursor, part)
		info, err := os.Lstat(cursor)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic-link target requires agent review")
		}
		if cursor == target && !info.Mode().IsRegular() {
			return fmt.Errorf("target is not a regular file")
		}
	}
	return nil
}

func acceptAdvisorReview(cwd string, review advisorReview) error {
	if review.Path == "" || len(review.After) == 0 {
		return fmt.Errorf("no local change to apply")
	}
	if err := advisorLocalPath(cwd, review.Path); err != nil {
		return err
	}
	current, err := os.ReadFile(review.Path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if (err == nil) != review.Existed || !bytes.Equal(current, review.Before) {
		return fmt.Errorf("file changed since preview; open a fresh preview")
	}
	if err := os.MkdirAll(filepath.Dir(review.Path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(review.Path), ".cully-review-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck
	if err := tmp.Chmod(review.Mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(review.After); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), review.Path)
}
