package cully

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// paneHUD is the one-line Agent Health bar that replaces the panel's top rule:
//
//	── Codex │ cully:main │ 27m │ Context 63% │ ⚠ Loop 3x │ ● 2 unchecked ── Ctrl+] advisor ──
//
// It shows only measured values, so a segment with no data is left out. The
// whole panel is clickable, and the bar ends with the advisor hint.
func paneHUD(cols int, stats toolStats, view sessionView, now time.Time) string {
	if cols < 24 {
		return ""
	}
	var parts []string
	if view.Agent != "" {
		parts = append(parts, bold+journalAgentName(view.Agent)+rst)
	}
	if project := filepath.Base(view.Project); project != "" && project != "." && project != string(filepath.Separator) {
		if view.Branch != "" {
			project += ":" + view.Branch
		}
		parts = append(parts, project)
	}
	if !view.Started.IsZero() {
		parts = append(parts, healthDuration(now.Sub(view.Started)))
	}
	if view.ContextKnown {
		used := min(100, max(0, 100-view.ContextLeft))
		color := ""
		switch {
		case used >= 90:
			color = red
		case used >= 75:
			color = yellow
		}
		parts = append(parts, color+"Context "+strconv.Itoa(used)+"%"+rst)
	}
	if view.Loops.Active() {
		parts = append(parts, yellow+"⚠ Loop "+strconv.Itoa(view.Loops.Loops[0].Attempts)+"x"+rst)
	}
	if stats.EditsSinceCheck > 0 {
		parts = append(parts, "● "+strconv.Itoa(stats.EditsSinceCheck)+" unchecked")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, dim+" │ "+rst)
}

// hudRule draws the HUD inside a horizontal rule of the given width.
func hudRule(width int, hud string) string {
	if hud == "" || width < 24 {
		return dim + strings.Repeat("─", max(0, width)) + rst
	}
	const hint = " Ctrl+] advisor "
	body := ansi.Truncate(hud, max(1, width-len(hint)-6), "…")
	left := dim + "── " + rst + body + dim + " "
	fill := width - ansi.StringWidth(ansi.Strip(left)) - len(hint) - 2
	if fill < 2 {
		return left + dim + strings.Repeat("─", max(0, width-ansi.StringWidth(ansi.Strip(left)))) + rst
	}
	return left + strings.Repeat("─", fill) + hint + "──" + rst
}
