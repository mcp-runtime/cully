package mcp

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// repoRoot resolves the workspace root from this test file so the contract
// below reads the same manifests and skill sources that ship to clients.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

var manifestToolRE = regexp.MustCompile(`(?m)^\s*-\s*name:\s*(cully_[a-z_]+)\s*$`)

// Only backticked names count as tool references; bare cully_ identifiers
// such as table names are not tools.
var docToolRE = regexp.MustCompile("`(cully_[a-z_]+)`")

// manifestTools returns the cully_* tool names a deployment manifest gates.
func manifestTools(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, m := range manifestToolRE.FindAllSubmatch(raw, -1) {
		name := string(m[1])
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		t.Fatalf("%s lists no tools", path)
	}
	return names
}

// TestToolDiscoveryMatchesManifests keeps registered tools, deployment
// manifests and the canonical skill in agreement. A tool missing from a
// manifest may be blocked or unmetered at the gateway; a name used in the
// skill but never registered sends agents down a dead end.
func TestToolDiscoveryMatchesManifests(t *testing.T) {
	var registered []string
	for _, tool := range toolRegistrations() {
		registered = append(registered, tool.name)
	}
	slices.Sort(registered)

	root := repoRoot(t)
	for _, manifest := range []string{
		filepath.Join(root, ".mcp", "servers.yaml"),
		filepath.Join(root, ".mcp", "examples", "no-oauth", "servers.yaml"),
	} {
		gated := manifestTools(t, manifest)
		slices.Sort(gated)
		if !slices.Equal(registered, gated) {
			t.Fatalf("%s gates %v, server registers %v", manifest, gated, registered)
		}
	}

	raw, err := os.ReadFile(filepath.Join(root, "skills", "cully", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range docToolRE.FindAllSubmatch(raw, -1) {
		if name := string(m[1]); !slices.Contains(registered, name) {
			t.Fatalf("skill references unregistered tool %s", name)
		}
	}
	if !strings.Contains(string(raw), "`clear_task`") {
		t.Fatal("skill does not document task unlinking")
	}
}
