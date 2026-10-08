package cully

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type codingAgent struct {
	ID          string
	Name        string
	ConfigDir   string
	ProjectDirs []string
	UserDirs    []string
	MCPFiles    []string
}

func codingAgents(cwd string) []codingAgent {
	out := make([]codingAgent, 0, len(agentCatalog()))
	for _, spec := range agentCatalog() {
		paths := spec.ConfigPaths(cwd)
		out = append(out, codingAgent{
			ID:          spec.ID,
			Name:        spec.DisplayName,
			ConfigDir:   spec.ConfigDir(),
			ProjectDirs: paths.ProjectDirs,
			UserDirs:    paths.UserDirs,
			MCPFiles:    paths.MCPFiles,
		})
	}
	return out
}

func sharedSkillRoot(cwd string) string {
	return filepath.Join(cwd, ".cully", "skills")
}

func sharedSkillPath(cwd, name string) string {
	return filepath.Join(sharedSkillRoot(cwd), name, "SKILL.md")
}

// CodexConfigDir returns the Codex config directory, honoring CODEX_HOME.
func CodexConfigDir() string {
	if d := os.Getenv("CODEX_HOME"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".codex"
	}
	return filepath.Join(home, ".codex")
}

// CursorConfigDir returns the Cursor config directory, honoring CURSOR_CONFIG_DIR.
func CursorConfigDir() string {
	if d := os.Getenv("CURSOR_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".cursor"
	}
	return filepath.Join(home, ".cursor")
}

func listCodingAgents(cwd string) string {
	parts := make([]string, 0, 3)
	for _, agent := range codingAgents(cwd) {
		project := existingDirs(agent.ProjectDirs)
		user := existingDirs(agent.UserDirs)
		state := "ready"
		if project == 0 && user == 0 {
			state = "not configured"
		}
		parts = append(parts, agent.ID+":"+state)
	}
	return strings.Join(parts, " ")
}

func allProjectDirs(cwd string, pick func(codingAgent) []string) []string {
	var dirs []string
	for _, agent := range codingAgents(cwd) {
		dirs = append(dirs, pick(agent)...)
	}
	return dirs
}

func existingDirs(dirs []string) int {
	n := 0
	for _, d := range dirs {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			n++
		}
	}
	return n
}

func sortedNames(set map[string]bool) string {
	names := make([]string, 0, len(set))
	for k := range set {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, " ")
}
