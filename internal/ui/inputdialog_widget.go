package ui

import (
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type InputDialogWidget struct {
	BaseWidget
	Title        string
	ConfirmLabel string
	Input        InputWidget
	Borders      *term.BorderSet
	OnSubmit     func(value string)
	OnDismiss    func()
	focusedBtn   int // 0 = input, 1 = cancel, 2 = save
	boxX         int
	boxY         int
	boxW         int
}

func NewInputDialogWidget(title, placeholder, initial string) *InputDialogWidget {
	d := &InputDialogWidget{
		Title: title,
		Input: InputWidget{Prefix: " ❯ ", Style: term.StylePaletteItem, Placeholder: placeholder},
	}
	d.Input.SetText(initial)
	return d
}

func (d *InputDialogWidget) Focusable() bool { return true }

func (d *InputDialogWidget) CursorPosition() (int, int, bool) {
	if d.focusedBtn == 0 {
		return d.Input.CursorX(d.boxX + 1), d.boxY + 3, true
	}
	return 0, 0, false
}

func (d *InputDialogWidget) Render(surface *RenderSurface) {
	sw, _ := surface.Size()

	d.boxW = 50
	if d.boxW > sw-4 {
		d.boxW = sw - 4
	}
	boxH := 7
	d.boxX = (sw - d.boxW) / 2
	d.boxY = 2

	b := term.DoubleBorderSet()
	if d.Borders != nil {
		b = *d.Borders
	}
	bs := term.StyleBorder

	surface.ClearRect(d.boxX, d.boxY, d.boxW, boxH, term.StylePaletteItem)
	surface.DrawBorder(d.boxX, d.boxY, d.boxW, boxH, b, bs)

	// Title inside box
	surface.DrawText(d.boxX+2, d.boxY+1, d.Title, d.boxX+d.boxW-2, term.StylePaletteItem)

	// Input row
	inputW := d.boxW - 2
	d.Input.Render(surface, d.boxX+1, d.boxY+3, inputW)

	// Buttons row
	btnY := d.boxY + 5
	cancelLabel := " Cancel "
	confirmText := "Save"
	if d.ConfirmLabel != "" {
		confirmText = d.ConfirmLabel
	}
	saveLabel := " " + confirmText + " "

	cancelStyle := term.StyleMuted
	saveStyle := term.StyleMuted
	if d.focusedBtn == 1 {
		cancelStyle = term.StylePaletteSelected
	}
	if d.focusedBtn == 2 {
		saveStyle = term.StylePaletteSelected
	}

	saveX := d.boxX + d.boxW - 2 - len([]rune(saveLabel))
	surface.DrawText(saveX, btnY, saveLabel, 0, saveStyle)
	cancelX := saveX - 1 - len([]rune(cancelLabel))
	surface.DrawText(cancelX, btnY, cancelLabel, 0, cancelStyle)
}

func (d *InputDialogWidget) HandleEvent(ev tcell.Event) EventResult {
	if mev, ok := ev.(*tcell.EventMouse); ok {
		if mev.Buttons()&tcell.Button1 != 0 {
			mx, my := mev.Position()
			btnY := d.boxY + 5
			if my == btnY {
				confirmText := "Save"
				if d.ConfirmLabel != "" {
					confirmText = d.ConfirmLabel
				}
				saveLabel := " " + confirmText + " "
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
			if my == d.boxY+3 {
				d.focusedBtn = 0
				d.Input.HandleTextClick(mx)
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
