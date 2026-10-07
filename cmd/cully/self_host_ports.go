package main

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
)

// selfHostPorts lists the loopback ports published by the local stack, with preferred values.
// Every service has a fixed preferred port; a busy one is replaced by the next free port.
var selfHostPorts = []struct {
	key       string
	preferred int
}{
	{"CULLY_MCP_PORT", 3393},
	{"CULLY_DATA_API_PORT", 43393},
	{"CULLY_MEM0_PORT", 53393},
	{"CULLY_DB_PORT", 55393},
	{"CULLY_MEM0_DB_PORT", 56393},
}

func portFree(port int) bool {
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	return listener.Close() == nil
}

// choosePort returns the first usable port, starting with the saved value, then the preferred one.
func choosePort(saved string, preferred int, taken map[int]bool, free func(int) bool) (int, error) {
	candidates := []int{}
	if value, err := strconv.Atoi(strings.TrimSpace(saved)); err == nil && value > 0 && value < 65536 {
		candidates = append(candidates, value)
	}
	candidates = append(candidates, preferred)
	for _, port := range candidates {
		if !taken[port] && free(port) {
			return port, nil
		}
	}
	for port := preferred + 1; port < 65536 && port <= preferred+2000; port++ {
		if !taken[port] && free(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free local port found near %d", preferred)
}

// ensureSelfHostPorts saves the chosen ports in .env. When the stack is already running its
// published ports are kept, because the running containers hold them.
func ensureSelfHostPorts(envPath string, running bool) (map[string]int, error) {
	values, err := readSelfHostEnv(envPath)
	if err != nil {
		return nil, err
	}
	chosen := make(map[string]int)
	taken := make(map[int]bool)
	for _, entry := range selfHostPorts {
		saved := values[entry.key]
		if running && saved != "" {
			if port, err := strconv.Atoi(saved); err == nil {
				chosen[entry.key] = port
				taken[port] = true
				continue
			}
		}
		if running && entry.key == "CULLY_MCP_PORT" {
			// A stack from before port selection publishes the old default.
			chosen[entry.key] = 8080
			taken[8080] = true
			continue
		}
		port, err := choosePort(saved, entry.preferred, taken, portFree)
		if err != nil {
			return nil, err
		}
		chosen[entry.key] = port
		taken[port] = true
	}
	if err := writeSelfHostEnvValues(envPath, chosen); err != nil {
		return nil, err
	}
	return chosen, nil
}

func writeSelfHostEnvValues(path string, values map[string]int) error {
	contents, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(contents), "\n"), "\n")
	if len(contents) == 0 {
		lines = nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	done := make(map[string]bool)
	for i, line := range lines {
		key, _, ok := strings.Cut(strings.TrimSpace(line), "=")
		key = strings.TrimSpace(key)
		if value, found := values[key]; ok && found && !strings.HasPrefix(key, "#") {
			lines[i] = fmt.Sprintf("%s=%d", key, value)
			done[key] = true
		}
	}
	for _, key := range keys {
		if !done[key] {
			lines = append(lines, fmt.Sprintf("%s=%d", key, values[key]))
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

func runSelfHostPorts(args []string) error {
	if len(args) == 0 || len(args) > 2 || args[0] != "ensure" || (len(args) == 2 && args[1] != "--running") {
		return fmt.Errorf("usage: cully _internal self-hosted-ports ensure [--running]")
	}
	chosen, err := ensureSelfHostPorts(".env", len(args) == 2)
	if err != nil {
		return err
	}
	for _, entry := range selfHostPorts {
		fmt.Printf("%s=%d\n", entry.key, chosen[entry.key])
	}
	return nil
}
