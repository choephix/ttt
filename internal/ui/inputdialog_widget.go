package ui

import (
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type InputDialogWidget struct {
	BaseWidget
	Title     string
	Input     string
	Borders   *term.BorderSet
	OnSubmit  func(value string)
	OnDismiss func()
}

func NewInputDialogWidget(title, initial string) *InputDialogWidget {
	return &InputDialogWidget{
		Title: title,
		Input: initial,
	}
}

func (d *InputDialogWidget) Focusable() bool { return true }

func (d *InputDialogWidget) Render(surface *RenderSurface) {
	sw, _ := surface.Size()

	titleW := len([]rune(d.Title)) + 4
	boxW := 40
	if titleW > boxW {
		boxW = titleW
	}
	if boxW > sw-4 {
		boxW = sw - 4
	}
	boxX := (sw - boxW) / 2
	boxY := 2
	boxH := 3

	b := term.DoubleBorderSet()
	if d.Borders != nil {
		b = *d.Borders
	}
	bs := term.StylePaletteBorder

	for x := boxX; x < boxX+boxW; x++ {
		surface.SetCell(x, boxY, term.Cell{Ch: b.Horizontal, Style: bs})
		surface.SetCell(x, boxY+boxH-1, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	for y := boxY; y < boxY+boxH; y++ {
		surface.SetCell(boxX, y, term.Cell{Ch: b.Vertical, Style: bs})
		surface.SetCell(boxX+boxW-1, y, term.Cell{Ch: b.Vertical, Style: bs})
	}
	surface.SetCell(boxX, boxY, term.Cell{Ch: b.TopLeft, Style: bs})
	surface.SetCell(boxX+boxW-1, boxY, term.Cell{Ch: b.TopRight, Style: bs})
	surface.SetCell(boxX, boxY+boxH-1, term.Cell{Ch: b.BottomLeft, Style: bs})
	surface.SetCell(boxX+boxW-1, boxY+boxH-1, term.Cell{Ch: b.BottomRight, Style: bs})

	tx := boxX + 2
	for _, ch := range d.Title {
		if tx < boxX+boxW-1 {
			surface.SetCell(tx, boxY, term.Cell{Ch: ch, Style: bs})
			tx++
		}
	}

	inputY := boxY + 1
	for x := boxX + 1; x < boxX+boxW-1; x++ {
		surface.SetCell(x, inputY, term.Cell{Ch: ' ', Style: term.StylePaletteInput})
	}
	x := boxX + 2
	for _, ch := range d.Input {
		if x < boxX+boxW-2 {
			surface.SetCell(x, inputY, term.Cell{Ch: ch, Style: term.StylePaletteInput})
			x++
		}
	}
}

func (d *InputDialogWidget) HandleEvent(ev tcell.Event) EventResult {
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
	case tcell.KeyEnter:
		if d.OnSubmit != nil && d.Input != "" {
			d.OnSubmit(d.Input)
		}
		return EventConsumed
	case tcell.KeyRune:
		d.Input += string(kev.Rune())
		return EventConsumed
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(d.Input) > 0 {
			runes := []rune(d.Input)
			d.Input = string(runes[:len(runes)-1])
		}
		return EventConsumed
	}

	return EventConsumed
}
