package ui

import (
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type InputDialogWidget struct {
	BaseWidget
	Title      string
	Input      InputWidget
	Borders    *term.BorderSet
	OnSubmit   func(value string)
	OnDismiss  func()
	focusedBtn int // 0 = input, 1 = cancel, 2 = save
	boxX       int
	boxY       int
	boxW       int
}

func NewInputDialogWidget(title, initial string) *InputDialogWidget {
	d := &InputDialogWidget{
		Title: title,
		Input: InputWidget{Prefix: " > ", Style: term.StylePaletteItem},
	}
	d.Input.SetText(initial)
	return d
}

func (d *InputDialogWidget) Focusable() bool { return true }

func (d *InputDialogWidget) CursorPosition() (int, int, bool) {
	if d.focusedBtn == 0 {
		return d.Input.CursorX(d.boxX + 1), d.boxY + 1, true
	}
	return 0, 0, false
}

func (d *InputDialogWidget) Render(surface *RenderSurface) {
	sw, sh := surface.Size()

	d.boxW = 50
	if d.boxW > sw-4 {
		d.boxW = sw - 4
	}
	boxH := 4
	d.boxX = (sw - d.boxW) / 2
	d.boxY = (sh - boxH) / 2
	if d.boxY < 1 {
		d.boxY = 1
	}

	b := term.DoubleBorderSet()
	if d.Borders != nil {
		b = *d.Borders
	}
	bs := term.StyleBorder

	// Clear interior
	for y := d.boxY; y < d.boxY+boxH; y++ {
		for x := d.boxX; x < d.boxX+d.boxW; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: term.StylePaletteItem})
		}
	}

	// Top/bottom borders
	for x := d.boxX + 1; x < d.boxX+d.boxW-1; x++ {
		surface.SetCell(x, d.boxY, term.Cell{Ch: b.Horizontal, Style: bs})
		surface.SetCell(x, d.boxY+boxH-1, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	// Side borders
	for y := d.boxY + 1; y < d.boxY+boxH-1; y++ {
		surface.SetCell(d.boxX, y, term.Cell{Ch: b.Vertical, Style: bs})
		surface.SetCell(d.boxX+d.boxW-1, y, term.Cell{Ch: b.Vertical, Style: bs})
	}
	// Corners
	surface.SetCell(d.boxX, d.boxY, term.Cell{Ch: b.TopLeft, Style: bs})
	surface.SetCell(d.boxX+d.boxW-1, d.boxY, term.Cell{Ch: b.TopRight, Style: bs})
	surface.SetCell(d.boxX, d.boxY+boxH-1, term.Cell{Ch: b.BottomLeft, Style: bs})
	surface.SetCell(d.boxX+d.boxW-1, d.boxY+boxH-1, term.Cell{Ch: b.BottomRight, Style: bs})

	// Title on top border
	tx := d.boxX + 2
	for _, ch := range d.Title {
		if tx < d.boxX+d.boxW-2 {
			surface.SetCell(tx, d.boxY, term.Cell{Ch: ch, Style: bs})
			tx++
		}
	}

	// Input row
	inputW := d.boxW - 2
	d.Input.Render(surface, d.boxX+1, d.boxY+1, inputW)

	// Buttons row
	btnY := d.boxY + 2
	cancelLabel := " Cancel "
	saveLabel := " Save "

	cancelStyle := term.StyleMuted
	saveStyle := term.StyleMuted
	if d.focusedBtn == 1 {
		cancelStyle = term.StylePaletteSelected
	}
	if d.focusedBtn == 2 {
		saveStyle = term.StylePaletteSelected
	}

	bx := d.boxX + d.boxW - 2 - len([]rune(saveLabel))
	for _, ch := range saveLabel {
		surface.SetCell(bx, btnY, term.Cell{Ch: ch, Style: saveStyle})
		bx++
	}
	bx = d.boxX + d.boxW - 2 - len([]rune(saveLabel)) - 1 - len([]rune(cancelLabel))
	for _, ch := range cancelLabel {
		surface.SetCell(bx, btnY, term.Cell{Ch: ch, Style: cancelStyle})
		bx++
	}
}

func (d *InputDialogWidget) HandleEvent(ev tcell.Event) EventResult {
	if mev, ok := ev.(*tcell.EventMouse); ok {
		if mev.Buttons()&tcell.Button1 != 0 {
			mx, my := mev.Position()
			btnY := d.boxY + 2
			if my == btnY {
				saveLabel := " Save "
				cancelLabel := " Cancel "
				saveEnd := d.boxX + d.boxW - 2
				saveStart := saveEnd - len([]rune(saveLabel))
				cancelEnd := saveStart - 1
				cancelStart := cancelEnd - len([]rune(cancelLabel))
				if mx >= saveStart && mx < saveEnd {
					d.focusedBtn = 2
					if d.OnSubmit != nil && d.Input.Text != "" {
						d.OnSubmit(d.Input.Text)
					}
					return EventConsumed
				}
				if mx >= cancelStart && mx < cancelEnd {
					d.focusedBtn = 1
					if d.OnDismiss != nil {
						d.OnDismiss()
					}
					return EventConsumed
				}
			}
			if my == d.boxY+1 {
				d.focusedBtn = 0
			}
		}
		return EventConsumed
	}

	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventConsumed
	}

	switch kev.Key() {
	case tcell.KeyEscape:
		if d.OnDismiss != nil {
			d.OnDismiss()
		}
	case tcell.KeyTab:
		d.focusedBtn = (d.focusedBtn + 1) % 3
	case tcell.KeyBacktab:
		d.focusedBtn = (d.focusedBtn + 2) % 3
	case tcell.KeyEnter:
		if d.focusedBtn == 1 {
			if d.OnDismiss != nil {
				d.OnDismiss()
			}
		} else {
			if d.OnSubmit != nil && d.Input.Text != "" {
				d.OnSubmit(d.Input.Text)
			}
		}
	default:
		if d.focusedBtn != 0 {
			d.focusedBtn = 0
		}
		d.Input.HandleEvent(ev)
	}

	return EventConsumed
}
