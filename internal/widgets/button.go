package widgets

import (
	"strings"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type ButtonConfig struct {
	Label   string `json:"label"`
	Command string `json:"command"`
	Style   term.Style `json:"-"`

	OnCommand func(command string)
}

type ButtonWidget struct {
	BaseWidget
	Config      ButtonConfig
	label       string
	accelIndex  int
	accelRune   rune
	focused     bool
}

func NewButtonWidget(config ButtonConfig) *ButtonWidget {
	b := &ButtonWidget{Config: config, accelIndex: -1}
	idx := strings.IndexByte(config.Label, '&')
	if idx >= 0 && idx < len(config.Label)-1 {
		runes := []rune(config.Label)
		runeIdx := len([]rune(config.Label[:idx]))
		b.label = string(runes[:runeIdx]) + string(runes[runeIdx+1:])
		b.accelIndex = runeIdx
		b.accelRune = runes[runeIdx+1]
	} else {
		b.label = config.Label
	}
	return b
}

func (b *ButtonWidget) Height() int { return 1 + b.BoxOverheadH() }
func (b *ButtonWidget) Width() int {
	w := len([]rune(b.label)) + b.BoxOverheadW()
	return w
}

func (b *ButtonWidget) Focusable() bool    { return true }
func (b *ButtonWidget) SetFocused(f bool)  { b.focused = f }
func (b *ButtonWidget) IsFocused() bool    { return b.focused }

func (b *ButtonWidget) Render(surface Surface) {
	style := b.Config.Style
	if style == 0 {
		style = term.StyleButton
	}
	if b.focused {
		style = term.StyleButtonFocused
	}

	padded := b.BorderedInterior(surface)
	pw, _ := padded.Size()
	fillH := 1 + b.Box.PaddingTop + b.Box.PaddingBottom
	for y := 0; y < fillH; y++ {
		for x := 0; x < pw; x++ {
			padded.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}
	}

	inner := b.RenderBox(surface)
	w, _ := inner.Size()
	if w <= 0 {
		return
	}

	x := 0
	for i, ch := range b.label {
		if x >= w {
			break
		}
		cell := term.Cell{Ch: ch, Style: style}
		if i == b.accelIndex {
			cell.Underline = true
		}
		inner.SetCell(x, 0, cell)
		x++
	}
}

func (b *ButtonWidget) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		if e.Buttons()&tcell.Button1 != 0 {
			mx, my := e.Position()
			r := b.GetRect()
			if mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H {
				b.trigger()
				return true
			}
		}
	case *tcell.EventKey:
		if b.focused {
			if e.Key() == tcell.KeyEnter || (e.Key() == tcell.KeyRune && e.Rune() == ' ') {
				b.trigger()
				return true
			}
		}
		if b.accelRune != 0 && e.Key() == tcell.KeyRune {
			r := e.Rune()
			if r == b.accelRune || r == b.accelRune+32 || r == b.accelRune-32 {
				b.trigger()
				return true
			}
		}
	}
	return false
}

func (b *ButtonWidget) trigger() {
	if b.Config.OnCommand != nil && b.Config.Command != "" {
		b.Config.OnCommand(b.Config.Command)
	}
}
