package ui

import (
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type InfoEntry struct {
	Key  string
	Desc string
}

type InfoDialogWidget struct {
	BaseWidget
	Title        string
	Entries      []InfoEntry
	Width        int
	Height       int
	Borders      *term.BorderSet
	InvertStyles bool

	OnDismiss func()

	scrollTop int
	btnHit    HitRegion
}

func NewInfoDialogWidget(title string, entries []InfoEntry) *InfoDialogWidget {
	return &InfoDialogWidget{
		Title:   title,
		Entries: entries,
	}
}

func (d *InfoDialogWidget) Focusable() bool { return true }

func (d *InfoDialogWidget) Render(surface *RenderSurface) {
	sw, sh := surface.Size()

	keyColW := 0
	for _, e := range d.Entries {
		if len([]rune(e.Key)) > keyColW {
			keyColW = len([]rune(e.Key))
		}
	}
	keyColW += 2

	descColW := 0
	for _, e := range d.Entries {
		if len([]rune(e.Desc)) > descColW {
			descColW = len([]rune(e.Desc))
		}
	}

	contentW := keyColW + descColW + 2
	titleW := len([]rune(d.Title)) + 4
	if titleW > contentW {
		contentW = titleW
	}

	boxW := contentW + 9 // 2 border + 2 padding + 5 extra width
	if d.Width > 0 {
		boxW = d.Width
	}
	if boxW > sw-4 {
		boxW = sw - 4
	}
	if boxW < 20 {
		boxW = 20
	}

	visibleEntries := len(d.Entries)
	if d.Height > 0 {
		if visibleEntries > d.Height {
			visibleEntries = d.Height
		}
	}
	maxVisibleH := sh - 8
	if visibleEntries > maxVisibleH {
		visibleEntries = maxVisibleH
	}
	if visibleEntries < 1 {
		visibleEntries = 1
	}

	// top border + title + separator + entries + gutter + button row + bottom border
	boxH := visibleEntries + 6
	boxX := (sw - boxW) / 2
	boxY := (sh - boxH) / 2
	if boxY < 1 {
		boxY = 1
	}

	b := term.DoubleBorderSet()
	if d.Borders != nil {
		b = *d.Borders
	}

	surface.ClearRect(boxX, boxY, boxW, boxH, term.StylePaletteItem)
	surface.DrawBorder(boxX, boxY, boxW, boxH, b, term.StyleBorder)

	surface.ClearRect(boxX+1, boxY+1, boxW-2, 1, term.StylePaletteItem)
	titleX := boxX + (boxW-len([]rune(d.Title)))/2
	surface.DrawText(titleX, boxY+1, d.Title, boxX+boxW-2, term.StylePaletteItem)

	sepY := boxY + 2
	for x := boxX + 1; x < boxX+boxW-1; x++ {
		surface.SetCell(x, sepY, term.Cell{Ch: b.Horizontal, Style: term.StyleBorder})
	}

	if d.scrollTop > len(d.Entries)-visibleEntries {
		d.scrollTop = len(d.Entries) - visibleEntries
	}
	if d.scrollTop < 0 {
		d.scrollTop = 0
	}

	innerW := boxW - 4
	for i := 0; i < visibleEntries && d.scrollTop+i < len(d.Entries); i++ {
		entry := d.Entries[d.scrollTop+i]
		y := boxY + 3 + i

		keyRunes := []rune(entry.Key)
		kw := keyColW
		if kw > innerW/2 {
			kw = innerW / 2
		}
		kx := boxX + 2 + kw - len(keyRunes)
		if kx < boxX+2 {
			kx = boxX + 2
		}
		keyStyle, descStyle := term.StylePaletteItem, term.StyleMuted
		if d.InvertStyles {
			keyStyle, descStyle = descStyle, keyStyle
		}
		surface.DrawText(kx, y, entry.Key, boxX+2+kw, keyStyle)

		descX := boxX + 2 + kw + 2
		surface.DrawText(descX, y, entry.Desc, boxX+boxW-2, descStyle)
	}

	if d.scrollTop > 0 {
		surface.SetCell(boxX+boxW-2, boxY+3, term.Cell{Ch: '^', Style: term.StyleMuted})
	}
	if d.scrollTop+visibleEntries < len(d.Entries) {
		surface.SetCell(boxX+boxW-2, boxY+3+visibleEntries-1, term.Cell{Ch: 'v', Style: term.StyleMuted})
	}

	// Close button with 1 row gutter above
	btnY := boxY + boxH - 2
	btnLabel := " Close "
	btnRunes := []rune(btnLabel)
	btnX := boxX + (boxW-len(btnRunes))/2
	surface.ClearRect(boxX+1, btnY-1, boxW-2, 1, term.StylePaletteItem)
	surface.ClearRect(boxX+1, btnY, boxW-2, 1, term.StylePaletteItem)
	for j, ch := range btnRunes {
		surface.SetCell(btnX+j, btnY, term.Cell{Ch: ch, Style: term.StylePaletteSelected})
	}
	d.btnHit = HitRegion{X: btnX, Y: btnY, W: len(btnRunes)}

	if len(d.Entries) > visibleEntries && visibleEntries > 1 {
		sbX := boxX + boxW - 2
		sbTop := boxY + 3
		ratio := float64(d.scrollTop) / float64(len(d.Entries)-visibleEntries)
		thumbY := sbTop + int(ratio*float64(visibleEntries-1))
		for y := sbTop; y < sbTop+visibleEntries; y++ {
			ch := ' '
			style := term.StyleScrollbar
			if y == thumbY {
				style = term.StyleScrollbarThumb
			}
			surface.SetCell(sbX, y, term.Cell{Ch: rune(ch), Style: style})
		}
	}
}

func (d *InfoDialogWidget) HandleEvent(ev tcell.Event) EventResult {
	if mev, ok := ev.(*tcell.EventMouse); ok {
		btn := mev.Buttons()
		mx, my := mev.Position()
		if btn&tcell.Button1 != 0 {
			if d.btnHit.Contains(mx, my) {
				if d.OnDismiss != nil {
					d.OnDismiss()
				}
				return EventConsumed
			}
			return EventConsumed
		}
		if btn&tcell.WheelUp != 0 {
			d.scrollTop -= 3
			if d.scrollTop < 0 {
				d.scrollTop = 0
			}
			return EventConsumed
		}
		if btn&tcell.WheelDown != 0 {
			max := len(d.Entries) - 5
			if max < 0 {
				max = 0
			}
			d.scrollTop += 3
			if d.scrollTop > max {
				d.scrollTop = max
			}
			return EventConsumed
		}
		return EventConsumed
	}

	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventConsumed
	}

	switch kev.Key() {
	case tcell.KeyEscape, tcell.KeyEnter:
		if d.OnDismiss != nil {
			d.OnDismiss()
		}
	case tcell.KeyUp:
		d.scrollTop--
		if d.scrollTop < 0 {
			d.scrollTop = 0
		}
	case tcell.KeyDown:
		d.scrollTop++
		max := len(d.Entries) - 5
		if max < 0 {
			max = 0
		}
		if d.scrollTop > max {
			d.scrollTop = max
		}
	}

	return EventConsumed
}
