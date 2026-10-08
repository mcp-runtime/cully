package cully

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

// The interactive replay is a step player beside file activity. Only
// runReplayTUI touches the terminal, so everything else is testable with strings.
//
// Keys: space pause; left/right step; up/down speed; f files; / filter; q quit.

const (
	viewSteps = "steps"
	viewFiles = "files"

	splitPanelCols  = 100
	replayMinSpeed  = 0.25
	replayMaxSpeed  = 64.0
	replayStartWait = 400 * time.Millisecond
)

type replayKey struct {
	Kind string // up down left right pgup pgdn enter space esc tab backspace rune click wheelup wheeldown ctrlc
	R    rune
	X, Y int // 1-based terminal cell for clicks
}

// parseReplayKeys decodes raw terminal input, including SGR 1006 mouse reports.
func parseReplayKeys(b []byte) []replayKey {
	var keys []replayKey
	for i := 0; i < len(b); {
		c := b[i]
		switch {
		case c == 0x1b && i+2 < len(b) && b[i+1] == '[' && b[i+2] == '<':
			j := i + 3
			for j < len(b) && b[j] != 'M' && b[j] != 'm' {
				j++
			}
			if j >= len(b) {
				return keys
			}
			parts := strings.Split(string(b[i+3:j]), ";")
			if len(parts) == 3 {
				btn, _ := strconv.Atoi(parts[0])
				x, _ := strconv.Atoi(parts[1])
				y, _ := strconv.Atoi(parts[2])
				switch {
				case btn == 64:
					keys = append(keys, replayKey{Kind: "wheelup", X: x, Y: y})
				case btn == 65:
					keys = append(keys, replayKey{Kind: "wheeldown", X: x, Y: y})
				case btn == 0 && b[j] == 'M':
					keys = append(keys, replayKey{Kind: "click", X: x, Y: y})
				}
			}
			i = j + 1
		case c == 0x1b && i+2 < len(b) && b[i+1] == '[':
			code, n := b[i+2], 3
			kind := ""
			switch code {
			case 'A':
				kind = "up"
			case 'B':
				kind = "down"
			case 'C':
				kind = "right"
			case 'D':
				kind = "left"
			case '5', '6':
				if i+3 < len(b) && b[i+3] == '~' {
					n = 4
					kind = map[byte]string{'5': "pgup", '6': "pgdn"}[code]
				}
			}
			if kind != "" {
				keys = append(keys, replayKey{Kind: kind})
			}
			i += n
		case c == 0x1b:
			keys = append(keys, replayKey{Kind: "esc"})
			i++
		case c == 3:
			keys = append(keys, replayKey{Kind: "ctrlc"})
			i++
		case c == '\r' || c == '\n':
			keys = append(keys, replayKey{Kind: "enter"})
			i++
		case c == '\t':
			keys = append(keys, replayKey{Kind: "tab"})
			i++
		case c == 0x7f || c == 0x08:
			keys = append(keys, replayKey{Kind: "backspace"})
			i++
		case c == ' ':
			keys = append(keys, replayKey{Kind: "space", R: ' '})
			i++
		case c >= 0x20 && c < 0x7f:
			keys = append(keys, replayKey{Kind: "rune", R: rune(c)})
			i++
		default:
			i++
		}
	}
	return keys
}

type replayModel struct {
	Doc           replayDoc
	View          string
	PrevView      string
	Width, Height int
	Color         bool
	Selected      int
	Expanded      map[int]bool
	Scroll        int
	Filter        string
	Filtering     bool
	filterBuf     string
	Pos           int // steps revealed in the player
	Paused        bool
	Speed         float64
	Quit          bool
}

func newReplayModel(doc replayDoc, o replayOptions, width, height int, color bool) *replayModel {
	m := &replayModel{Doc: doc, View: viewSteps, Width: width, Height: height, Color: color, Speed: o.Speed, Paused: true}
	if m.Speed <= 0 {
		m.Speed = 1
	}
	return m
}

func (m *replayModel) bodyHeight() int { return max(1, m.Height-5) }

func (m *replayModel) restartPlayer() {
	m.View = viewSteps
	m.Pos = 0
	m.Paused = false
}

func stepMatches(s replayStep, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(s.Path, filter) || strings.Contains(s.To, filter) || strings.Contains(s.Detail, filter) || strings.Contains(s.Cmd, filter)
}

// Update applies one key or mouse event.
func (m *replayModel) Update(k replayKey) {
	if k.Kind == "ctrlc" {
		m.Quit = true
		return
	}
	if m.Filtering {
		switch k.Kind {
		case "enter":
			m.Filter, m.Filtering = m.filterBuf, false
			m.afterFilter()
		case "esc":
			m.Filtering, m.filterBuf, m.Filter = false, "", ""
		case "backspace":
			if r := []rune(m.filterBuf); len(r) > 0 {
				m.filterBuf = string(r[:len(r)-1])
			}
		case "rune", "space":
			if len(m.filterBuf) < 80 {
				m.filterBuf += string(k.R)
			}
		}
		return
	}
	switch k.Kind {
	case "rune", "space", "enter", "esc", "tab", "up", "down", "left", "right", "pgup", "pgdn", "click", "wheelup", "wheeldown":
	default:
		return
	}
	if k.Kind == "rune" {
		switch k.R {
		case 'q':
			m.Quit = true
			return
		case '/':
			m.Filtering, m.filterBuf = true, m.Filter
			return
		case 'f':
			if m.View == viewFiles {
				m.View = viewSteps
			} else {
				m.PrevView, m.View = m.View, viewFiles
				m.Pos = len(m.Doc.Steps)
			}
			return
		case 's', 'p':
			m.restartPlayer()
			return
		}
	}
	switch m.View {
	case viewSteps:
		m.updatePlayer(k)
	case viewFiles:
		if k.Kind == "esc" {
			m.View = viewSteps
		}
	}
}

func (m *replayModel) afterFilter() {}

func (m *replayModel) updatePlayer(k replayKey) {
	switch k.Kind {
	case "space":
		m.Paused = !m.Paused
	case "left":
		m.Pos, m.Paused = max(0, m.Pos-1), true
	case "right":
		m.Pos, m.Paused = min(len(m.Doc.Steps), m.Pos+1), true
	case "up":
		m.Speed = min(replayMaxSpeed, m.Speed*2)
	case "down":
		m.Speed = max(replayMinSpeed, m.Speed/2)
	case "esc":
		m.Filter = ""
	}
}

// Tick advances playback by one step and reports whether anything changed.
func (m *replayModel) Tick() bool {
	if m.View != viewSteps || m.Paused || m.Pos >= len(m.Doc.Steps) {
		return false
	}
	m.Pos++
	return true
}

// NextDelay is the wait before the next Tick, from the recorded timestamps.
func (m *replayModel) NextDelay() time.Duration {
	if m.Pos == 0 {
		return replayStartWait
	}
	if m.Pos >= len(m.Doc.Steps) {
		return time.Hour
	}
	return replayDelay(m.Doc.Steps[m.Pos].Time.Sub(m.Doc.Steps[m.Pos-1].Time), m.Speed)
}

// ---------------------------------------------------------------------------
// Pure layout

func fitTo(s string, w int) string {
	if w < 1 {
		return ""
	}
	s = ansi.Truncate(s, w, "…")
	if d := w - ansi.StringWidth(s); d > 0 {
		s += strings.Repeat(" ", d)
	}
	return s
}

func joinColumns(left, right []string, lw, rw, height int) []string {
	out := make([]string, 0, height)
	for i := 0; i < height; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		out = append(out, fitTo(l, lw)+" │ "+fitTo(r, rw))
	}
	return out
}

func clipLines(lines []string, height, width int) []string {
	out := make([]string, 0, height)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			out = append(out, ansi.Truncate(lines[i], width, "…"))
		} else {
			out = append(out, "")
		}
	}
	return out
}

func timeCol(t, start, end time.Time, w int) int {
	total := end.Sub(start)
	if total <= 0 || w <= 1 {
		return 0
	}
	c := int(float64(t.Sub(start)) / float64(total) * float64(w-1))
	return min(max(c, 0), w-1)
}

// renderScrubber draws a timeline bar with failed checks (✕) and loops (⚠)
// marked along it. mark is the current position; the span [spanA, spanB] is
// drawn heavier. It returns two lines: the markers and the track.
func renderScrubber(doc replayDoc, mark, spanA, spanB time.Time, width int, color bool) []string {
	start, end := doc.Meta.Start, doc.Meta.End
	left, right := stampOf(start), stampOf(end)
	w := width - len(left) - len(right) - 2
	if w < 8 {
		return []string{"", ""}
	}
	marks := []rune(strings.Repeat(" ", w))
	track := []rune(strings.Repeat("─", w))
	if !spanA.IsZero() {
		for c := timeCol(spanA, start, end, w); c <= timeCol(spanB, start, end, w); c++ {
			track[c] = '▬'
		}
	}
	mc := timeCol(mark, start, end, w)
	for c := 0; c < mc; c++ {
		if track[c] == '─' {
			track[c] = '━'
		}
	}
	track[mc] = '●'
	for _, s := range doc.Steps {
		c := timeCol(s.Time, start, end, w)
		switch {
		case s.Kind == "loop":
			marks[c] = '⚠'
		case s.Kind == "check" && s.Failed && marks[c] != '⚠':
			marks[c] = '✕'
		}
	}
	mk := string(marks)
	if color {
		mk = strings.NewReplacer("⚠", yellow+"⚠"+rst, "✕", red+"✕"+rst).Replace(mk)
	}
	return []string{strings.Repeat(" ", len(left)+1) + mk, paint(color, dim, left) + " " + string(track) + " " + paint(color, dim, right)}
}

type fileState struct {
	Path                           string
	Edits, Reads                   int
	Created, Deleted, Moved, Wrote bool
}

func (f fileState) badge() string {
	switch {
	case f.Deleted:
		return "−"
	case f.Created:
		return "+"
	case f.Edits > 0 || f.Wrote:
		return "~"
	case f.Moved:
		return "→"
	}
	return "·"
}

// fileStatesAt folds the first n steps into per-file state.
func fileStatesAt(steps []replayStep, n int, filter string) []fileState {
	states := map[string]*fileState{}
	get := func(p string) *fileState {
		if states[p] == nil {
			states[p] = &fileState{Path: p}
		}
		return states[p]
	}
	for _, s := range steps[:min(n, len(steps))] {
		if s.Kind != "file" || s.Failed {
			continue
		}
		if filter != "" && !strings.Contains(s.Path, filter) && !strings.Contains(s.To, filter) {
			continue
		}
		f := get(s.Path)
		switch s.Op {
		case opRead:
			f.Reads++
		case opCreate:
			f.Created, f.Deleted = true, false
		case opEdit, opWrite:
			f.Edits++
			f.Deleted = false
		case opDelete:
			f.Deleted = true
		case opMove:
			f.Deleted = true
			if s.To != "" {
				t := get(s.To)
				t.Moved, t.Deleted = true, false
			}
		}
	}
	out := make([]fileState, 0, len(states))
	for _, f := range states {
		out = append(out, *f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func splitDir(p string) (dir, name string) {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i+1], p[i+1:]
	}
	return "", p
}

// renderFilePanel is the live FILE ACTIVITY tree: files touched so far, each
// with an operation badge and an edit-count heat bar. Hotspots stand out.
func renderFilePanel(steps []replayStep, n int, filter string, width, height int, color bool) []string {
	states := fileStatesAt(steps, n, filter)
	title := fmt.Sprintf("FILE ACTIVITY (%d)", len(states))
	if filter != "" {
		title += " /" + safeText(filter)
	}
	lines := []string{paint(color, bold, title)}
	if len(states) == 0 {
		lines = append(lines, paint(color, dim, "No files yet"))
		return clipLines(lines, height, width)
	}
	lastDir := "\x00"
	for _, f := range states {
		dir, name := splitDir(f.Path)
		if dir != lastDir {
			lastDir = dir
			if dir != "" {
				lines = append(lines, paint(color, dim, safeText(dir)))
			}
		}
		indent := ""
		if dir != "" {
			indent = "  "
		}
		heat := ""
		hot := f.Edits >= replayHotspotEdits
		if f.Edits > 0 {
			heat = " " + strings.Repeat("█", min(f.Edits, 8)) + fmt.Sprintf(" ×%d", f.Edits)
			if hot {
				heat += " ▲"
			}
		}
		line := indent + f.badge() + " " + safeText(name) + heat
		if hot {
			line = paint(color, bold+yellow, line)
		} else if f.Deleted {
			line = paint(color, red, line)
		} else if f.Created {
			line = paint(color, green, line)
		} else if f.Reads > 0 && f.Edits == 0 {
			line = paint(color, dim, line)
		}
		lines = append(lines, line)
	}
	if len(lines) > height {
		more := len(lines) - (height - 1)
		lines = append(lines[:height-1], paint(color, dim, fmt.Sprintf("… %d more lines", more)))
	}
	return clipLines(lines, height, width)
}

// renderStepFeed shows the steps revealed so far: past steps dim, the current
// one highlighted, future steps hidden.
func renderStepFeed(steps []replayStep, n int, filter string, width, height int, color bool) []string {
	var lines []string
	cur := n - 1
	for i := 0; i < n && i < len(steps); i++ {
		s := steps[i]
		if !stepMatches(s, filter) {
			continue
		}
		if i == cur {
			lines = append(lines, paint(color, bold, "▸ ")+renderStepLine(s, color))
		} else {
			lines = append(lines, "  "+paint(color, dim, ansi.Strip(renderStepLine(s, false))))
		}
	}
	if len(lines) == 0 {
		lines = []string{paint(color, dim, "Press → to reveal the first step")}
	}
	if len(lines) > height {
		lines = lines[len(lines)-height:]
	}
	return clipLines(lines, height, width)
}

func (m *replayModel) footer() []string {
	var f replayFooter
	var cur time.Time
	if len(m.Doc.Steps) > 0 {
		idx := len(m.Doc.Steps) - 1
		if m.View == viewSteps {
			idx = min(m.Pos, len(m.Doc.Steps)) - 1
		}
		if idx >= 0 {
			f, cur = m.Doc.Steps[idx].Footer, m.Doc.Steps[idx].Time
		}
	}
	elapsed := time.Duration(0)
	if !cur.IsZero() {
		elapsed = cur.Sub(m.Doc.Meta.Start)
	}
	status := renderFooterText(f, m.Color) + " · elapsed " + formatDuration(elapsed) + " / " + formatDuration(m.Doc.Meta.Duration())
	var keys string
	switch {
	case m.Filtering:
		keys = "filter by file: " + safeText(m.filterBuf) + "▏  (enter apply · esc clear)"
	case m.View == viewSteps:
		state := "playing"
		if m.Paused {
			state = "paused"
		} else if m.Pos >= len(m.Doc.Steps) {
			state = "end"
		}
		status += fmt.Sprintf(" · %s %sx", state, strconv.FormatFloat(m.Speed, 'f', -1, 64))
		keys = "space pause · ←/→ step · ↑/↓ speed · f files · / filter · q quit"
	case m.View == viewFiles:
		keys = "f/esc back · / filter · q quit"
	default:
		keys = "space pause · ←/→ step · f files · / filter · q quit"
	}
	if m.Filter != "" && !m.Filtering {
		status += " · filter /" + safeText(m.Filter)
	}
	return []string{status, paint(m.Color, dim, keys)}
}

// Render lays out one full frame: a title strip and scrubber on top, the body,
// and a footer. Every line is at most Width cells.
func (m *replayModel) Render() []string {
	w, h := m.Width, m.Height
	var mark, spanA, spanB time.Time
	switch {
	case len(m.Doc.Steps) > 0:
		idx := len(m.Doc.Steps) - 1
		if m.View == viewSteps {
			idx = min(m.Pos, len(m.Doc.Steps)) - 1
		}
		mark = m.Doc.Meta.Start
		if idx >= 0 {
			mark = m.Doc.Steps[idx].Time
		}
	default:
		mark = m.Doc.Meta.Start
	}
	lines := []string{renderHeader(m.Doc, m.Color)}
	lines = append(lines, renderScrubber(m.Doc, mark, spanA, spanB, w, m.Color)...)
	body := m.bodyHeight()
	switch m.View {
	case viewSteps:
		if w >= splitPanelCols {
			lw := w * 6 / 10
			lines = append(lines, joinColumns(
				renderStepFeed(m.Doc.Steps, m.Pos, m.Filter, lw, body, m.Color),
				renderFilePanel(m.Doc.Steps, m.Pos, m.Filter, w-lw-3, body, m.Color), lw, w-lw-3, body)...)
		} else {
			fh := body / 3
			lines = append(lines, renderStepFeed(m.Doc.Steps, m.Pos, m.Filter, w, body-fh-1, m.Color)...)
			lines = append(lines, paint(m.Color, dim, strings.Repeat("─", w)))
			lines = append(lines, renderFilePanel(m.Doc.Steps, m.Pos, m.Filter, w, fh, m.Color)...)
		}
	case viewFiles:
		lines = append(lines, renderFilePanel(m.Doc.Steps, m.Pos, m.Filter, w, body, m.Color)...)
	}
	lines = append(lines, m.footer()...)
	return clipLines(lines, h, w)
}

// ---------------------------------------------------------------------------
// Terminal loop

// runReplayTUI runs the interactive replay on a real terminal.
func runReplayTUI(out, in *os.File, doc replayDoc, o replayOptions) error {
	cols, rows, err := term.GetSize(int(out.Fd()))
	if err != nil || cols < 40 || rows < 10 {
		_, err := io.WriteString(out, renderReplayText(doc, o, max(cols, replayPlainWidth), os.Getenv("NO_COLOR") == ""))
		return err
	}
	state, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(in.Fd()), state) //nolint:errcheck
	if _, err := io.WriteString(out, "\x1b[?1049h\x1b[?25l\x1b[?1000h\x1b[?1006h"); err != nil {
		return err
	}
	defer io.WriteString(out, "\x1b[?1000l\x1b[?1006l\x1b[0m\x1b[?25h\x1b[?1049l") //nolint:errcheck

	m := newReplayModel(doc, o, cols, rows, os.Getenv("NO_COLOR") == "")
	input := make(chan []replayKey, 8)
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := in.Read(buf)
			if n > 0 {
				input <- parseReplayKeys(append([]byte(nil), buf[:n]...))
			}
			if err != nil {
				close(input)
				return
			}
		}
	}()
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)

	draw := func() {
		var b strings.Builder
		b.WriteString("\x1b[H")
		for i, l := range m.Render() {
			if i > 0 {
				b.WriteString("\r\n")
			}
			b.WriteString(l + "\x1b[0m\x1b[K")
		}
		_, _ = io.WriteString(out, b.String())
	}
	timer := time.NewTimer(m.NextDelay())
	defer timer.Stop()
	for !m.Quit {
		draw()
		wait := time.Hour
		if m.View == viewSteps && !m.Paused && m.Pos < len(doc.Steps) {
			wait = m.NextDelay()
		}
		timer.Reset(wait)
		select {
		case keys, ok := <-input:
			if !ok {
				return nil
			}
			for _, k := range keys {
				m.Update(k)
			}
		case <-timer.C:
			m.Tick()
		case <-winch:
			if c, r, err := term.GetSize(int(out.Fd())); err == nil {
				m.Width, m.Height = c, r
				_, _ = io.WriteString(out, "\x1b[2J")
			}
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}
	return nil
}
