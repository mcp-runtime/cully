package cully

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The HTML export is ONE self-contained file: inline CSS and JS, no network,
// no external fonts or scripts. It embeds only the journal-derived document
// that `--json` prints (project-relative paths, operations, command labels,
// timestamps, check results, loop notes). Recorded text reaches the page in two
// safe ways only: as JSON in a data block (json.Marshal escapes < > & and the
// line separators) rendered with textContent, and HTML-escaped in the
// <noscript> transcript. A strict CSP allows only the two inline blocks, by
// hash.

const replayHTMLNote = "Generated locally by Cully. Contains file paths and command names, no file contents."

const replayHTMLPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="%%CSP%%">
<meta name="color-scheme" content="light dark">
<title>%%TITLE%%</title>
<style>%%STYLE%%</style>
</head>
<body>
<header><h1 id="title">%%TITLE%%</h1><div class="sub" id="sub"></div></header>
<div class="bar">
<div class="controls">
<button id="back" type="button" aria-label="Previous step">&larr;</button>
<button id="play" type="button">Play</button>
<button id="fwd" type="button" aria-label="Next step">&rarr;</button>
<label>Speed <select id="speed"><option value="0.5">0.5x</option><option value="1" selected>1x</option><option value="2">2x</option><option value="4">4x</option><option value="8">8x</option></select></label>
<label>Operation <select id="op"><option value="">All</option><option value="read">Read</option><option value="edit">Edit</option><option value="create">Create</option><option value="delete">Delete</option><option value="move">Move</option><option value="check">Checks</option><option value="run">Commands</option></select></label>
<label>File <input id="file" type="text" placeholder="filter by path" autocomplete="off"></label>
<label>Agent <select id="agent"></select></label>
</div>
<div class="scrub"><div class="markers" id="markers"></div><input id="pos" type="range" min="0" max="0" value="0" aria-label="Replay position"></div>
<div class="times"><span id="t0"></span><span id="counts"></span><span id="t1"></span></div>
</div>
<main>
<section><h2>Step by step</h2><div id="feed"></div></section>
<section class="wide"><h2>File activity</h2><div class="tree" id="files"></div></section>
<section><h2>Summary</h2><div id="summary"></div></section>
<section><h2>Reconcile</h2><div id="reconcile"></div></section>
</main>
<noscript><section><h2>Transcript (JavaScript is off)</h2><pre>%%NOSCRIPT%%</pre></section></noscript>
<footer>%%NOTE%%</footer>
<script type="application/json" id="data">%%DATA%%</script>
<script>%%SCRIPT%%</script>
</body>
</html>
`

func cspHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// renderReplayHTML builds the whole page as a string.
func renderReplayHTML(doc replayDoc) (string, error) {
	export := replayExport(doc)
	for i := range export.Steps {
		export.Steps[i].Time = export.Steps[i].Time.Truncate(time.Millisecond)
	}
	data, err := json.Marshal(export)
	if err != nil {
		return "", err
	}
	csp := strings.Join([]string{
		"default-src 'none'",
		"style-src " + cspHash(replayHTMLStyle),
		"script-src " + cspHash(replayHTMLScript),
		"base-uri 'none'", "form-action 'none'",
	}, "; ")
	title := "Replay · " + doc.Meta.Agent
	if doc.Meta.Project != "" {
		title += " · " + doc.Meta.Project
	}
	text := renderReplayText(doc, replayOptions{}, replayPlainWidth, false)
	return strings.NewReplacer(
		"%%CSP%%", csp, // built only from constants and base64 hashes
		"%%TITLE%%", html.EscapeString(title),
		"%%STYLE%%", replayHTMLStyle,
		"%%NOSCRIPT%%", html.EscapeString(text),
		"%%NOTE%%", html.EscapeString(replayHTMLNote),
		"%%DATA%%", string(data),
		"%%SCRIPT%%", replayHTMLScript,
	).Replace(replayHTMLPage), nil
}

// writeReplayHTML writes the page with owner-only permissions.
func writeReplayHTML(path string, doc replayDoc) error {
	page, err := renderReplayHTML(doc)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write replay: %w", err)
	}
	if _, err := f.WriteString(page); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
