//go:build !windows

package cully

// paneDivider remembers the requested height across terminal resizes. Automatic
// growth stops after a drag, so refreshing advice cannot undo the users choice.
type paneDivider struct {
	Height           int
	Manual, Dragging bool
}

func (d *paneDivider) top(rows, contentRows int) int {
	if !d.Manual {
		return paneTop(rows, contentRows)
	}
	if rows < 12 {
		return rows
	}
	minimum := 8
	if rows >= 24 {
		minimum = 12
	}
	return rows - min(max(1, d.Height), rows-minimum)
}

// handle consumes only a left-button gesture beginning on the divider. Drag
// motion and release remain owned by the divider even outside the panel.
func (d *paneDivider) handle(mouse *panelMouse, rows, top int) bool {
	if d.Dragging {
		if mouse.Release || mouse.Button&32 != 0 && mouse.Button&3 == 0 {
			d.Height = rows - mouse.Y + 1
			d.Manual = true
			if mouse.Release {
				d.Dragging = false
			}
		}
		return true
	}
	if rows >= 12 && top < rows && !mouse.Release && mouse.Button == 0 && mouse.Y == top+1 {
		d.Dragging = true
		return true
	}
	return false
}
