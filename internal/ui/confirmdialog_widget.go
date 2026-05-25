package ui

import (
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ConfirmDialogWidget struct {
	BaseWidget
	Message   string
	Selected  int
	Borders   *term.BorderSet
	OnConfirm func()
	OnDismiss func()
}

func NewConfirmDialogWidget(message string) *ConfirmDialogWidget {
	return &ConfirmDialogWidget{
		Message:  message,
		Selected: 1,
	}
}

func (d *ConfirmDialogWidget) Focusable() bool { return true }

func (d *ConfirmDialogWidget) Render(surface *RenderSurface) {
	sw, _ := surface.Size()

	msgW := len([]rune(d.Message)) + 4
	boxW := 30
	if msgW > boxW {
		boxW = msgW
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

	yesLabel := " Yes "
	noLabel := " No "
	totalW := len(yesLabel) + 2 + len(noLabel)
	startX := boxX + (boxW-totalW)/2

	yesStyle := term.StylePaletteItem
	noStyle := term.StylePaletteItem
	if d.Selected == 0 {
		yesStyle = term.StylePaletteSelected
	} else {
		noStyle = term.StylePaletteSelected
	}

	bx := startX
	for _, ch := range yesLabel {
		surface.SetCell(bx, btnY, term.Cell{Ch: ch, Style: yesStyle})
		bx++
	}
	bx += 2
	for _, ch := range noLabel {
		surface.SetCell(bx, btnY, term.Cell{Ch: ch, Style: noStyle})
		bx++
	}
}

func (d *ConfirmDialogWidget) HandleEvent(ev tcell.Event) EventResult {
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventConsumed
	}

	switch kev.Key() {
	case tcell.KeyEscape:
		if d.OnDismiss != nil {
			d.OnDismiss()
		}
		return EventConsumed
	case tcell.KeyLeft, tcell.KeyRight, tcell.KeyTab:
		d.Selected = 1 - d.Selected
		return EventConsumed
	case tcell.KeyEnter:
		if d.Selected == 0 {
			if d.OnConfirm != nil {
				d.OnConfirm()
			}
		} else {
			if d.OnDismiss != nil {
				d.OnDismiss()
			}
		}
		return EventConsumed
	case tcell.KeyRune:
		if kev.Rune() == 'y' || kev.Rune() == 'Y' {
			if d.OnConfirm != nil {
				d.OnConfirm()
			}
			return EventConsumed
		}
		if kev.Rune() == 'n' || kev.Rune() == 'N' {
			if d.OnDismiss != nil {
				d.OnDismiss()
			}
			return EventConsumed
		}
	}

	return EventConsumed
}
