//go:build windows

package cully

import "os/exec"

func runAgentCommand(cmd *exec.Cmd) error { return cmd.Run() }

func agentExitCode(exit *exec.ExitError) int {
	code := exit.ExitCode()
	if code < 0 {
		return 1
	}
	return code
}
