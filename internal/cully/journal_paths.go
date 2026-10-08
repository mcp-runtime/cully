package cully

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// Replay extraction turns one tool call into project-relative file operations
// and a coarse command label.
//
// File paths come only from structured file tools (read, write, edit, delete,
// apply_patch). A shell command contributes a label — the program and, for a
// known tool, its subcommand — and never a path. Arguments, URLs, output and
// file contents are not stored.
//
// CULLY_JOURNAL_PATHS=0 disables Path, To, Op and Cmd.

const (
	opRead   = "read"
	opWrite  = "write"
	opEdit   = "edit"
	opCreate = "create"
	opDelete = "delete"
	opMove   = "move"

	journalPathMax   = 200
	journalOpsPerHit = 50
)

var journalCmdPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+( [A-Za-z0-9._-]+)?$`)

func journalPathsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("CULLY_JOURNAL_PATHS"))) {
	case "0", "false", "off", "no":
		return false
	}
	return true
}

type fileOp struct {
	Path, Op, To string
}

// pathHasScheme reports whether p is a URL or other scheme, including the
// collapsed form filepath.Join produces from "https://...".
func pathHasScheme(p string) bool {
	if strings.Contains(p, "://") {
		return true
	}
	i := strings.IndexByte(p, ':')
	if i <= 1 {
		return false
	}
	for _, r := range p[:i] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '+' && r != '.' && r != '-' {
			return false
		}
	}
	return true
}

// relProjectPath converts a path to a clean project-relative path, or "" when
// it must not be recorded.
func relProjectPath(cwd, p string) string {
	if p == "" || cwd == "" || len(p) > 4*journalPathMax || pathHasScheme(p) {
		return ""
	}
	for _, r := range p {
		if unicode.IsControl(r) {
			return ""
		}
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	p = filepath.Clean(p)
	roots := []string{filepath.Clean(cwd)}
	if real, err := filepath.EvalSymlinks(roots[0]); err == nil && real != roots[0] {
		roots = append(roots, real)
	}
	for _, root := range roots {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) || pathHasScheme(rel) {
			continue
		}
		if len(rel) > journalPathMax {
			return ""
		}
		return rel
	}
	return ""
}

func toolBaseName(name string) string {
	name = strings.ToLower(name)
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	if i := strings.LastIndex(name, "__"); i >= 0 {
		name = name[i+2:]
	}
	return name
}

// extractToolActivity returns file operations from structured file tools and a
// command label from shell tools. Shell text is never scanned for paths.
func extractToolActivity(event toolEvent) (ops []fileOp, cmd string) {
	name := toolBaseName(event.ToolName)
	add := func(op, path, to string) {
		if len(ops) >= journalOpsPerHit {
			return
		}
		rel := relProjectPath(event.Cwd, path)
		if rel == "" {
			return
		}
		o := fileOp{Path: rel, Op: op}
		if to != "" {
			if o.To = relProjectPath(event.Cwd, to); o.To == "" {
				return
			}
		}
		ops = append(ops, o)
	}
	switch name {
	case "read", "read_file", "readfile", "view":
		add(opRead, toolInputPath(event.ToolInput, "file_path", "path", "target_file"), "")
	case "write", "create_file":
		add(opWrite, toolInputPath(event.ToolInput, "file_path", "path", "target_file"), "")
	case "edit", "multiedit", "str_replace", "strreplace":
		add(opEdit, toolInputPath(event.ToolInput, "file_path", "path", "target_file"), "")
	case "notebookedit":
		add(opEdit, toolInputPath(event.ToolInput, "notebook_path", "file_path", "path"), "")
	case "delete":
		add(opDelete, toolInputPath(event.ToolInput, "file_path", "path", "target_file"), "")
	case "apply_patch":
		for _, o := range patchFileOps(patchText(event.ToolInput)) {
			add(o.Op, o.Path, o.To)
		}
		cmd = "apply_patch"
	case "bash", "exec_command", "shell":
		cmd = commandLabelOf(shellCommandOf(event.ToolInput))
	}
	return ops, cmd
}

func toolInputPath(raw json.RawMessage, keys ...string) string {
	var in map[string]json.RawMessage
	if json.Unmarshal(raw, &in) != nil {
		return ""
	}
	for _, k := range keys {
		var s string
		if json.Unmarshal(in[k], &s) == nil && s != "" {
			return s
		}
	}
	return ""
}

func shellCommandOf(raw json.RawMessage) string {
	var in struct {
		Command json.RawMessage `json:"command"`
		Cmd     string          `json:"cmd"`
	}
	if json.Unmarshal(raw, &in) != nil {
		return ""
	}
	var s string
	if json.Unmarshal(in.Command, &s) == nil && s != "" {
		return s
	}
	var list []string
	if json.Unmarshal(in.Command, &list) == nil && len(list) > 0 {
		return strings.Join(list, " ")
	}
	return in.Cmd
}

// patchText returns the apply_patch body from a known field. Other strings in
// the tool input, including file contents, are ignored.
func patchText(raw json.RawMessage) string {
	var in map[string]json.RawMessage
	if json.Unmarshal(raw, &in) != nil {
		return ""
	}
	for _, k := range []string{"input", "patch", "diff"} {
		var s string
		if json.Unmarshal(in[k], &s) == nil && strings.Contains(s, "*** ") {
			return s
		}
	}
	return ""
}

func patchFileOps(text string) []fileOp {
	var ops []fileOp
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "*** Add File: "):
			ops = append(ops, fileOp{Path: strings.TrimSpace(line[len("*** Add File: "):]), Op: opCreate})
		case strings.HasPrefix(line, "*** Update File: "):
			ops = append(ops, fileOp{Path: strings.TrimSpace(line[len("*** Update File: "):]), Op: opEdit})
		case strings.HasPrefix(line, "*** Delete File: "):
			ops = append(ops, fileOp{Path: strings.TrimSpace(line[len("*** Delete File: "):]), Op: opDelete})
		case strings.HasPrefix(line, "*** Move to: "):
			if n := len(ops); n > 0 && ops[n-1].Op == opEdit && ops[n-1].To == "" {
				ops[n-1].Op, ops[n-1].To = opMove, strings.TrimSpace(line[len("*** Move to: "):])
			}
		}
		if len(ops) >= journalOpsPerHit {
			break
		}
	}
	return ops
}

var (
	envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	subcmdPattern  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

var subcommandPrograms = map[string]bool{
	"git": true, "go": true, "npm": true, "pnpm": true, "yarn": true, "cargo": true,
	"docker": true, "kubectl": true, "make": true, "pip": true, "pip3": true, "uv": true,
	"bun": true, "deno": true, "helm": true, "gh": true, "terraform": true, "dotnet": true,
	"mvn": true, "gradle": true, "poetry": true, "brew": true,
}

// Only fixed CLI verbs may enter a command label. A first positional argument
// can be a make target, script name, resource, or secret, even when it looks
// like a subcommand. Programs without a bounded verb vocabulary keep only
// their program name.
var journalSubcommands = map[string]map[string]bool{
	"git":       {"add": true, "branch": true, "checkout": true, "clean": true, "clone": true, "commit": true, "diff": true, "fetch": true, "init": true, "log": true, "merge": true, "mv": true, "pull": true, "push": true, "rebase": true, "remote": true, "reset": true, "restore": true, "rm": true, "show": true, "stash": true, "status": true, "switch": true, "tag": true, "worktree": true},
	"go":        {"build": true, "clean": true, "doc": true, "env": true, "fmt": true, "generate": true, "get": true, "install": true, "list": true, "mod": true, "run": true, "test": true, "tool": true, "vet": true, "version": true, "work": true},
	"npm":       {"audit": true, "ci": true, "exec": true, "install": true, "publish": true, "run": true, "test": true, "uninstall": true, "update": true},
	"pnpm":      {"add": true, "audit": true, "build": true, "exec": true, "install": true, "publish": true, "remove": true, "run": true, "test": true, "update": true},
	"yarn":      {"add": true, "build": true, "install": true, "remove": true, "run": true, "test": true, "up": true},
	"cargo":     {"add": true, "build": true, "check": true, "clean": true, "clippy": true, "doc": true, "fmt": true, "install": true, "publish": true, "run": true, "test": true, "update": true},
	"docker":    {"build": true, "compose": true, "exec": true, "images": true, "inspect": true, "logs": true, "pull": true, "push": true, "run": true, "stop": true},
	"kubectl":   {"apply": true, "create": true, "delete": true, "describe": true, "exec": true, "get": true, "logs": true, "rollout": true, "scale": true},
	"pip":       {"check": true, "download": true, "freeze": true, "install": true, "list": true, "show": true, "uninstall": true},
	"pip3":      {"check": true, "download": true, "freeze": true, "install": true, "list": true, "show": true, "uninstall": true},
	"uv":        {"add": true, "build": true, "lock": true, "pip": true, "publish": true, "remove": true, "run": true, "sync": true, "tool": true},
	"bun":       {"add": true, "build": true, "install": true, "remove": true, "run": true, "test": true},
	"deno":      {"add": true, "check": true, "compile": true, "fmt": true, "lint": true, "run": true, "task": true, "test": true},
	"helm":      {"dependency": true, "install": true, "lint": true, "list": true, "package": true, "rollback": true, "template": true, "test": true, "uninstall": true, "upgrade": true},
	"gh":        {"api": true, "auth": true, "issue": true, "pr": true, "release": true, "repo": true, "run": true, "workflow": true},
	"terraform": {"apply": true, "destroy": true, "fmt": true, "init": true, "output": true, "plan": true, "show": true, "validate": true},
	"dotnet":    {"add": true, "build": true, "clean": true, "new": true, "publish": true, "restore": true, "run": true, "test": true},
	"poetry":    {"add": true, "build": true, "check": true, "install": true, "lock": true, "publish": true, "remove": true, "run": true, "update": true},
	"brew":      {"doctor": true, "install": true, "list": true, "outdated": true, "uninstall": true, "update": true, "upgrade": true},
}

var skippedPrefixes = map[string]bool{"sudo": true, "time": true, "nohup": true, "command": true, "env": true, "exec": true}

// noiseCommands are shell setup, not the work the label should name.
var noiseCommands = map[string]bool{
	"cd": true, "pushd": true, "popd": true, "true": true, "false": true,
	"export": true, "set": true, "source": true, ":": true,
}

func commandLabel(prog string, args []string) string {
	if !journalCmdPattern.MatchString(prog) || strings.Contains(prog, "..") {
		return ""
	}
	label := prog
	if len(args) > 0 {
		sub := args[0]
		if journalSubcommands[prog][sub] && subcmdPattern.MatchString(sub) {
			label += " " + sub
		}
	}
	if !journalCmdPattern.MatchString(label) {
		return ""
	}
	return label
}

func labelOfSegment(part string) string {
	fields := strings.Fields(part)
	for len(fields) > 0 {
		w := fields[0]
		if i := strings.IndexByte(w, '='); i > 0 && envNamePattern.MatchString(w[:i]) || skippedPrefixes[w] {
			fields = fields[1:]
			continue
		}
		break
	}
	if len(fields) == 0 {
		return ""
	}
	return commandLabel(filepath.Base(fields[0]), fields[1:])
}

// commandLabelOf returns the program and optional subcommand of a shell
// command. A later segment that is a known tool (go test, git commit) wins
// over a leading wrapper such as echo in a pipeline.
func commandLabelOf(command string) string {
	command = strings.TrimSpace(command)
	if command == "" || len(command) > 8192 {
		return ""
	}
	for _, r := range command {
		if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
			return ""
		}
	}
	var first, preferred string
	for _, part := range splitCommandSegments(command) {
		label := labelOfSegment(part)
		if label == "" {
			continue
		}
		prog := label
		if i := strings.IndexByte(label, ' '); i >= 0 {
			prog = label[:i]
		}
		if first == "" && !noiseCommands[prog] {
			first = label
		}
		if subcommandPrograms[prog] {
			preferred = label
			break
		}
	}
	if preferred != "" {
		return preferred
	}
	return first
}

func splitCommandSegments(command string) []string {
	var segs []string
	var b strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(command); i++ {
		c := command[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
			}
			b.WriteByte(c)
		case c == '\'' && !inDouble:
			inSingle = true
			b.WriteByte(c)
		case inDouble:
			if c == '\\' && i+1 < len(command) {
				b.WriteByte(c)
				i++
				b.WriteByte(command[i])
				continue
			}
			if c == '"' {
				inDouble = false
			}
			b.WriteByte(c)
		case c == '"':
			inDouble = true
			b.WriteByte(c)
		case c == ';' || c == '\n' || c == '|' || c == '&':
			segs = append(segs, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if b.Len() > 0 {
		segs = append(segs, b.String())
	}
	return segs
}
