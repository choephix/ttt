package ui

import (
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ConfirmDialogWidget struct {
	BaseWidget
	Message    string
	Buttons    []string
	Selected   int
	Borders    *term.BorderSet
	OnButton   []func()
	OnDismiss  func()
	btnSpans   [][2]int
	btnY       int
}

func NewConfirmDialogWidget(message string) *ConfirmDialogWidget {
	return &ConfirmDialogWidget{
		Message:  message,
		Buttons:  []string{"Yes", "No"},
		OnButton: make([]func(), 2),
		Selected: 1,
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
	boxW := 30
	if msgW > boxW {
		boxW = msgW
	}
	if btnW > boxW {
		boxW = btnW
	}
	if boxW > sw-4 {
		boxW = sw - 4
	}
	boxX := (sw - boxW) / 2
	boxY := 2
	boxH := 5

	b := term.DoubleBorderSet()
	if d.Borders != nil {
		b = *d.Borders
	}
	surface.DrawBorder(boxX, boxY, boxW, boxH, b, term.StyleBorder)

	// Message
	msgY := boxY + 1
	for x := boxX + 1; x < boxX+boxW-1; x++ {
		surface.SetCell(x, msgY, term.Cell{Ch: ' ', Style: term.StylePaletteItem})
	}
	mx := boxX + 2
	for _, ch := range d.Message {
		if mx < boxX+boxW-2 {
			surface.SetCell(mx, msgY, term.Cell{Ch: ch, Style: term.StylePaletteItem})
			mx++
		}
	}

	// Blank row
	for x := boxX + 1; x < boxX+boxW-1; x++ {
		surface.SetCell(x, boxY+2, term.Cell{Ch: ' ', Style: term.StylePaletteItem})
	}

	// Buttons row
	btnY := boxY + 3
	for x := boxX + 1; x < boxX+boxW-1; x++ {
		surface.SetCell(x, btnY, term.Cell{Ch: ' ', Style: term.StylePaletteItem})
	}

	labels := make([]string, len(d.Buttons))
	totalW := 0
	for i, btn := range d.Buttons {
		labels[i] = " " + btn + " "
		totalW += len([]rune(labels[i]))
	}
	totalW += (len(labels) - 1) * 2
	startX := boxX + (boxW-totalW)/2

	d.btnY = btnY
	d.btnSpans = make([][2]int, len(labels))
	bx := startX
	for i, label := range labels {
		style := term.StylePaletteItem
		if d.Selected == i {
			style = term.StylePaletteSelected
		}
		d.btnSpans[i][0] = bx
		for _, ch := range label {
			surface.SetCell(bx, btnY, term.Cell{Ch: ch, Style: style})
			bx++
		}
		d.btnSpans[i][1] = bx
		if i < len(labels)-1 {
			bx += 2
		}
	}
}

func (d *ConfirmDialogWidget) HandleEvent(ev tcell.Event) EventResult {
	if mev, ok := ev.(*tcell.EventMouse); ok {
		if mev.Buttons()&tcell.Button1 != 0 {
			mx, my := mev.Position()
			if my == d.btnY {
				for i, span := range d.btnSpans {
					if mx >= span[0] && mx < span[1] {
						d.Selected = i
						if d.OnButton[i] != nil {
							d.OnButton[i]()
						}
						return EventConsumed
					}
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
			if len(btn) > 0 && (kev.Rune() == rune(btn[0]) || kev.Rune() == rune(btn[0]+32)) {
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
