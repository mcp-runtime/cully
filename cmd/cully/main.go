// Command cully helps users guide coding agents and improve their workflow.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mcp-runtime/cully/internal/cully"
)

var version = "dev"

const help = `Cully — your companion for better work and everyday life.

Usage:
  cully setup [--agent claude|codex|cursor|all] [--oauth] [--prepare]
                                         Start the full local stack, integrations and advisor
  cully setup [--agent AGENT] --mcp-url URL [--oauth]
                                         Use an existing server; set up integrations and advisor
  cully uninstall [--purge-data]           Stop local stack and remove managed agent setup
  cully uninstall [claude|codex|cursor|all] Remove one or all agent integrations
  cully status [directory]                Show session and agent status
  cully suggestions                       Review suggested improvements
  cully codex [ARGS...]                   Run Codex with a live Cully advisor pane
  cully apply <n> [--dry-run] [--yes] [--cwd DIR]
  cully mcp add --url URL [--agent claude|codex|cursor] [--oauth]
  cully version

Shared personal and project memory is available through Cully's MCP tools.
Docs: https://docs.cully.net
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(help)
		return nil
	}
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Print(help)
	case "setup":
		return runSetup(args[1:])
	case "uninstall":
		return runUninstall(args[1:])
	case "status":
		if len(args) > 2 {
			return fmt.Errorf("usage: cully status [directory]")
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if len(args) == 2 {
			cwd = args[1]
		}
		cully.RunStatus(os.Stdout, cwd)
	case "suggestions":
		cully.RunList(os.Stdout)
	case "codex":
		return cully.RunCodexPane(args[1:], os.Stdin, os.Stdout)
	case "apply":
		n, yes, dryRun, cwd, err := parseApply(args[1:])
		if err != nil {
			return err
		}
		return cully.RunApply(n, cwd, yes, dryRun)
	case "version", "--version", "-v":
		fmt.Println("cully", version)
	case "mcp":
		if len(args) < 2 || args[1] != "add" {
			return fmt.Errorf("usage: cully mcp add --url URL [--agent claude|codex|cursor] [--oauth]")
		}
		fs := flag.NewFlagSet("mcp add", flag.ContinueOnError)
		agent := fs.String("agent", "", "coding agent; default: detect configured agents")
		endpoint := fs.String("url", "", "MCP URL of your deployment (required)")
		oauth := fs.Bool("oauth", false, "print sign-in instructions for an OAuth-protected server")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if len(fs.Args()) != 0 {
			return fmt.Errorf("mcp add: unexpected arguments")
		}
		if *endpoint == "" {
			return fmt.Errorf("mcp add: --url is required")
		}
		return cully.AddMCP(os.Stdout, *agent, *endpoint, *oauth)
	case "_internal":
		return runInternal(args[1:])
	default:
		return fmt.Errorf("unknown command %q; run cully help", args[0])
	}
	return nil
}

func runSetup(args []string) error {
	options, err := parseSetup(args)
	if err != nil {
		return err
	}
	if options.endpoint != "" {
		fmt.Println("Setting up Cully with an existing MCP server")
		var targets []string
		if options.target != "" {
			targets = append(targets, options.target)
		}
		if err := cully.InstallWithMCP(options.endpoint, options.oauth, targets...); err != nil {
			return err
		}
		fmt.Println("Setup complete. Restart your coding agent to load the Cully skill, hooks and MCP tools.")
		return nil
	}
	return runSelfHost(options.target, options.oauth, options.prepare)
}

type setupOptions struct {
	target, endpoint string
	oauth, prepare   bool
}

func parseSetup(args []string) (setupOptions, error) {
	var options setupOptions
	legacyTarget := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		legacyTarget, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	selected := fs.String("agent", "", "coding agent to connect: claude, codex, cursor, or all; default: detect installed agents")
	fs.StringVar(&options.endpoint, "mcp-url", "", "connect to an existing MCP server instead of starting local memory services")
	fs.BoolVar(&options.oauth, "oauth", false, "server uses OAuth; print sign-in instructions")
	fs.BoolVar(&options.prepare, "prepare", false, "download the self-hosted stack and create editable configuration without starting services")
	if err := fs.Parse(args); err != nil {
		return setupOptions{}, err
	}
	usage := fmt.Errorf("usage: cully setup [--agent claude|codex|cursor|all] [--mcp-url URL] [--oauth] [--prepare]")
	if len(fs.Args()) > 1 || (legacyTarget != "" && len(fs.Args()) != 0) {
		return setupOptions{}, usage
	}
	if legacyTarget != "" {
		options.target = legacyTarget
	}
	if len(fs.Args()) == 1 {
		options.target = fs.Args()[0]
	}
	agentFlagSet := false
	endpointFlagSet := false
	fs.Visit(func(option *flag.Flag) {
		if option.Name == "agent" {
			agentFlagSet = true
		}
		if option.Name == "mcp-url" {
			endpointFlagSet = true
		}
	})
	if agentFlagSet {
		if *selected == "" || options.target != "" {
			return setupOptions{}, usage
		}
		options.target = *selected
	}
	if endpointFlagSet && (options.endpoint == "" || options.prepare) {
		return setupOptions{}, fmt.Errorf("--mcp-url requires a URL and cannot be combined with --prepare")
	}
	if options.target != "" && options.target != "claude" && options.target != "codex" && options.target != "cursor" && options.target != "all" {
		return setupOptions{}, fmt.Errorf("choose --agent claude, codex, cursor, or all")
	}
	return options, nil
}

func parseApply(args []string) (n int, yes, dryRun bool, cwd string, err error) {
	if len(args) == 0 {
		err = fmt.Errorf("usage: cully apply <n> [--dry-run] [--yes] [--cwd DIR]")
		return
	}
	n, err = strconv.Atoi(args[0])
	if err != nil || n < 1 {
		err = fmt.Errorf("apply: suggestion number must be a positive integer")
		return
	}
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	fs.BoolVar(&yes, "yes", false, "apply without a prompt")
	fs.BoolVar(&dryRun, "dry-run", false, "show plan only")
	fs.StringVar(&cwd, "cwd", "", "project directory")
	err = fs.Parse(args[1:])
	if err == nil && len(fs.Args()) != 0 {
		err = fmt.Errorf("apply: unexpected arguments")
	}
	return
}

// Internal entry points are installed by Cully; they are not user commands.
func runInternal(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing internal action")
	}
	switch args[0] {
	case "statusline":
		cully.RunStatusline(os.Stdin, os.Stdout)
	case "analyze":
		cully.RunAnalyze(os.Stdin)
	case "cleanup":
		cully.RunCleanup(os.Stdin)
	case "continuity":
		if len(args) != 3 {
			return fmt.Errorf("internal continuity requires agent and event")
		}
		cully.RunContinuityHook(args[1], args[2], os.Stdin, os.Stdout)
	case "codex-signal":
		cully.RunCodexSignalHook(os.Stdin)
	case "daemon":
		cully.RunDaemon()
	case "stop-daemon":
		return cully.StopDaemon()
	case "worker":
		if len(args) != 4 {
			return fmt.Errorf("internal worker requires signals, session and directory")
		}
		cully.RunWorker(args[1], args[2], args[3])
	case "self-hosted-credentials":
		return runSelfHostCredentials(args[1:])
	default:
		return fmt.Errorf("unknown internal action %q", args[0])
	}
	return nil
}
