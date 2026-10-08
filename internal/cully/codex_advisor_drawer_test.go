//go:build !windows

package cully

import (
	"strings"
	"testing"
)

func TestAdvisorDrawerSelectionMouseAndReview(t *testing.T) {
	var d advisorDrawer
	d.open([]string{"ADV|Run focused checks.", "ADV|Plan the next task.", "MEMO|More details."})
	d.move(1)
	if d.Selected != 1 || d.Review != nil {
		t.Fatal(d)
	}
	screen := d.render(80, 24, toolStats{}, sessionView{Project: "project"})
	if !strings.Contains(screen, "▶ Next · Plan the next task.") || !strings.Contains(screen, "Preview selected action") {
		t.Fatal(screen)
	}
	found := false
	for row, index := range d.HitRows {
		if index == 1 && row > 3 {
			found = true
		}
	}
	if !found || d.AcceptRow <= 3 {
		t.Fatal("missing visible mouse targets", d.HitRows)
	}
	review := advisorHandoff(d.Items[d.Selected])
	d.Review = &review
	d.Scroll = 0
	screen = d.render(80, 24, toolStats{}, sessionView{})
	if !strings.Contains(screen, "Add request to coding session") || !strings.Contains(screen, "press Enter to") || !strings.Contains(screen, "send.") {
		t.Fatal(screen)
	}
	d.escape()
	if !d.Open || d.Review != nil {
		t.Fatal("Esc must return from preview to list")
	}
	d.escape()
	if d.Open {
		t.Fatal("Esc must close list")
	}
}

func TestAdvisorDrawerSelectionVisibleAfterPagingAndResize(t *testing.T) {
	var d advisorDrawer
	d.open([]string{"ADV|one", "ADV|two", "ADV|three", "ADV|four", "ADV|five", "ADV|six", "ADV|seven", "ADV|eight"})
	d.move(10000)
	screen := d.render(40, 12, toolStats{}, sessionView{})
	if !strings.Contains(screen, "▶ Next · eight") {
		t.Fatal("selected item hidden", screen)
	}
	d.Details = true
	d.Scroll = 10000
	_ = d.render(40, 12, toolStats{}, sessionView{})
	d.escape()
	if d.Details || !d.Open {
		t.Fatal(d)
	}
}

func TestAdvisorHandoffCannotInjectTerminalControls(t *testing.T) {
	got := terminalSafeText(advisorHandoff("ADV|safe\x1b[201~\x1b[31mtext\x00").Handoff)
	if strings.ContainsAny(got, "\x1b\x00") {
		t.Fatal(got)
	}
}
