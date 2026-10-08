package main

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChoosePortFallsBackWhenBusy(t *testing.T) {
	busy := map[int]bool{3393: true, 3394: true}
	free := func(port int) bool { return !busy[port] }
	port, err := choosePort("", 3393, map[int]bool{}, free)
	if err != nil || port != 3395 {
		t.Fatalf("got %d, %v", port, err)
	}
	port, err = choosePort("4000", 3393, map[int]bool{}, free)
	if err != nil || port != 4000 {
		t.Fatalf("saved port not kept: %d, %v", port, err)
	}
	port, err = choosePort("4000", 3393, map[int]bool{4000: true}, func(int) bool { return true })
	if err != nil || port != 3393 {
		t.Fatalf("taken saved port reused: %d, %v", port, err)
	}
}

func TestEnsureSelfHostPortsPersistsDistinctPorts(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("CULLY_MCP_HOST=mcp.example.com\nCULLY_MCP_PORT=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:3393")
	if err == nil {
		defer listener.Close()
	}
	chosen, err := ensureSelfHostPorts(path, false)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	for _, entry := range selfHostPorts {
		if chosen[entry.key] == 0 || seen[chosen[entry.key]] {
			t.Fatalf("bad ports: %v", chosen)
		}
		seen[chosen[entry.key]] = true
	}
	if chosen["CULLY_MCP_PORT"] == 1 {
		t.Fatal("kept unusable port")
	}
	contents, _ := os.ReadFile(path)
	if !strings.Contains(string(contents), "CULLY_MCP_HOST=mcp.example.com") || strings.Count(string(contents), "CULLY_MCP_PORT=") != 1 {
		t.Fatalf("env not updated cleanly: %s", contents)
	}
}

func TestEnsureSelfHostPortsKeepsRunningStack(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	chosen, err := ensureSelfHostPorts(path, true)
	if err != nil || chosen["CULLY_MCP_PORT"] != 8080 {
		t.Fatalf("legacy running stack changed: %v, %v", chosen, err)
	}
}
