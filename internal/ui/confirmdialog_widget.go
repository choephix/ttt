package ui

import (
	"unicode"

	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ConfirmDialogWidget struct {
	BaseWidget
	Message   string
	Buttons   []string
	Selected  int
	Borders   *term.BorderSet
	OnButton  []func()
	OnDismiss func()
	btnHits   []HitRegion
}

func NewConfirmDialogWidget(message string) *ConfirmDialogWidget {
	return &ConfirmDialogWidget{
		Message:  message,
		Buttons:  []string{"Yes", "No"},
		OnButton: make([]func(), 2),
		Selected: 1,
	}
}

func NewConfirmDialogWidget2(message, btn0, btn1 string) *ConfirmDialogWidget {
	return &ConfirmDialogWidget{
		Message:  message,
		Buttons:  []string{btn0, btn1},
		OnButton: make([]func(), 2),
		Selected: 0,
	}
}

func NewConfirmDialogWidget3(message, btn0, btn1, btn2 string) *ConfirmDialogWidget {
	return &ConfirmDialogWidget{
		Message:  message,
		Buttons:  []string{btn0, btn1, btn2},
		OnButton: make([]func(), 3),
		Selected: 2,
	}
}

func (d *ConfirmDialogWidget) Focusable() bool { return true }

func (d *ConfirmDialogWidget) Render(surface *RenderSurface) {
	sw, _ := surface.Size()

	msgW := len([]rune(d.Message)) + 4
	btnW := 4
	for _, btn := range d.Buttons {
		btnW += len([]rune(btn)) + 4
	}
	boxW := msgW
	if btnW > boxW {
		boxW = btnW
	}
	maxW := 60
	if sw-4 < maxW {
		maxW = sw - 4
	}
	if boxW > maxW {
		boxW = maxW
	}
	boxX := (sw - boxW) / 2
	boxY := 2
	boxH := 5

	b := term.DoubleBorderSet()
	if d.Borders != nil {
		b = *d.Borders
	}
	surface.DrawBorder(boxX, boxY, boxW, boxH, b, term.StyleBorder)

	surface.ClearRect(boxX+1, boxY+1, boxW-2, boxH-2, term.StylePaletteItem)

	// Message
	surface.DrawText(boxX+2, boxY+1, d.Message, boxX+boxW-2, term.StylePaletteItem)

	// Buttons row
	btnY := boxY + 3

	labels := make([]string, len(d.Buttons))
	totalW := 0
	for i, btn := range d.Buttons {
		labels[i] = " " + btn + " "
		totalW += len([]rune(labels[i]))
	}
	totalW += (len(labels) - 1) * 2
	startX := boxX + (boxW-totalW)/2

	d.btnHits = make([]HitRegion, len(labels))
	bx := startX
	for i, label := range labels {
		style := term.StylePaletteItem
		if d.Selected == i {
			style = term.StylePaletteSelected
		}
		labelRunes := []rune(label)
		d.btnHits[i] = HitRegion{X: bx, Y: btnY, W: len(labelRunes)}
		for j, ch := range labelRunes {
			cell := term.Cell{Ch: ch, Style: style}
			if j == 1 {
				cell.Underline = true
			}
			surface.SetCell(bx+j, btnY, cell)
		}
		bx += len(labelRunes)
		if i < len(labels)-1 {
			bx += 2
		}
	}
}

func (d *ConfirmDialogWidget) HandleEvent(ev tcell.Event) EventResult {
	if mev, ok := ev.(*tcell.EventMouse); ok {
		if mev.Buttons()&tcell.Button1 != 0 {
			mx, my := mev.Position()
			for i, hit := range d.btnHits {
				if hit.Contains(mx, my) {
					d.Selected = i
					if d.OnButton[i] != nil {
						d.OnButton[i]()
					}
					return EventConsumed
				}
			}
		}
		return EventConsumed
	}

	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventConsumed
	}

	n := len(d.Buttons)

	switch kev.Key() {
	case tcell.KeyEscape:
		if d.OnDismiss != nil {
			d.OnDismiss()
		}
		return EventConsumed
	case tcell.KeyLeft:
		d.Selected = (d.Selected - 1 + n) % n
		return EventConsumed
	case tcell.KeyRight, tcell.KeyTab:
		d.Selected = (d.Selected + 1) % n
		return EventConsumed
	case tcell.KeyEnter:
		if d.Selected >= 0 && d.Selected < len(d.OnButton) && d.OnButton[d.Selected] != nil {
			d.OnButton[d.Selected]()
		}
		return EventConsumed
	case tcell.KeyRune:
		for i, btn := range d.Buttons {
			first := []rune(btn)
			if len(first) > 0 && unicode.ToLower(kev.Rune()) == unicode.ToLower(first[0]) {
				d.Selected = i
				if d.OnButton[i] != nil {
					d.OnButton[i]()
				}
				return EventConsumed
			}
		}
	}

	return EventConsumed
}
