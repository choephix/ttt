package ui

import (
	"fmt"
	"ttt/internal/term"
	"ttt/internal/terminal"

	"github.com/gdamore/tcell/v2"
	"github.com/hinshun/vt10x"
)

type TerminalColorPalette struct {
	Fg      term.DirectColor
	Bg      term.DirectColor
	ANSI    [16]term.DirectColor
	Color256 [256]term.DirectColor
}

type TerminalWidget struct {
	BaseWidget
	Term    *terminal.Terminal
	Palette TerminalColorPalette
	focused bool
}

func NewTerminalWidget(t *terminal.Terminal, palette TerminalColorPalette) *TerminalWidget {
	return &TerminalWidget{
		Term:    t,
		Palette: palette,
	}
}

func (tw *TerminalWidget) Focusable() bool { return true }

func (tw *TerminalWidget) SetFocused(f bool) { tw.focused = f }

func (tw *TerminalWidget) WantsRawKeys() bool { return tw.focused }

func (tw *TerminalWidget) CursorPosition() (x, y int, visible bool) {
	if tw.Term == nil {
		return 0, 0, false
	}
	r := tw.GetRect()
	cx, cy := tw.Term.CursorPos()
	return r.X + cx, r.Y + cy, tw.focused
}

func (tw *TerminalWidget) Render(surface *RenderSurface) {
	if tw.Term == nil {
		return
	}
	w, h := surface.Size()

	tw.Term.Snapshot(func(view vt10x.View) {
		cols, rows := view.Size()
		for y := 0; y < h && y < rows; y++ {
			for x := 0; x < w && x < cols; x++ {
				g := view.Cell(x, y)
				cell := tw.glyphToCell(g)
				surface.SetCell(x, y, cell)
			}
		}
	})
}

func (tw *TerminalWidget) glyphToCell(g vt10x.Glyph) term.Cell {
	ch := g.Char
	if ch == 0 {
		ch = ' '
	}

	cell := term.Cell{
		Ch:     ch,
		Direct: true,
		Fg:     tw.resolveColor(g.FG, true),
		Bg:     tw.resolveColor(g.BG, false),
	}

	if g.Mode&terminal.AttrBold != 0 {
		cell.Attrs |= term.CellAttrBold
	}
	if g.Mode&terminal.AttrUnderline != 0 {
		cell.Attrs |= term.CellAttrUnderline
	}
	if g.Mode&terminal.AttrItalic != 0 {
		cell.Attrs |= term.CellAttrItalic
	}
	if g.Mode&terminal.AttrReverse != 0 {
		cell.Attrs |= term.CellAttrReverse
	}
	if g.Mode&terminal.AttrBlink != 0 {
		cell.Attrs |= term.CellAttrBlink
	}

	return cell
}

func (tw *TerminalWidget) resolveColor(c vt10x.Color, isFg bool) term.DirectColor {
	if c == vt10x.DefaultFG {
		return tw.Palette.Fg
	}
	if c == vt10x.DefaultBG {
		return tw.Palette.Bg
	}
	idx := int(c)
	if idx >= 0 && idx < 16 {
		return tw.Palette.ANSI[idx]
	}
	if idx >= 16 && idx < 256 {
		return tw.Palette.Color256[idx]
	}
	if isFg {
		return tw.Palette.Fg
	}
	return tw.Palette.Bg
}

func (tw *TerminalWidget) HandleEvent(ev tcell.Event) EventResult {
	if tw.Term == nil {
		return EventIgnored
	}

	switch tev := ev.(type) {
	case *tcell.EventKey:
		data := keyToVT(tev)
		if data != "" {
			tw.Term.WriteString(data)
			return EventConsumed
		}
	case *tcell.EventMouse:
		return EventIgnored
	}

	return EventIgnored
}

func keyToVT(ev *tcell.EventKey) string {
	if ev.Key() == tcell.KeyRune {
		mod := ev.Modifiers()
		if mod&tcell.ModAlt != 0 {
			return fmt.Sprintf("\x1b%c", ev.Rune())
		}
		return string(ev.Rune())
	}

	switch ev.Key() {
	case tcell.KeyEnter:
		return "\r"
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return "\x7f"
	case tcell.KeyTab:
		return "\t"
	case tcell.KeyEscape:
		return "\x1b"
	case tcell.KeyUp:
		return "\x1b[A"
	case tcell.KeyDown:
		return "\x1b[B"
	case tcell.KeyRight:
		return "\x1b[C"
	case tcell.KeyLeft:
		return "\x1b[D"
	case tcell.KeyHome:
		return "\x1b[H"
	case tcell.KeyEnd:
		return "\x1b[F"
	case tcell.KeyInsert:
		return "\x1b[2~"
	case tcell.KeyDelete:
		return "\x1b[3~"
	case tcell.KeyPgUp:
		return "\x1b[5~"
	case tcell.KeyPgDn:
		return "\x1b[6~"
	case tcell.KeyF1:
		return "\x1bOP"
	case tcell.KeyF2:
		return "\x1bOQ"
	case tcell.KeyF3:
		return "\x1bOR"
	case tcell.KeyF4:
		return "\x1bOS"
	case tcell.KeyF5:
		return "\x1b[15~"
	case tcell.KeyF6:
		return "\x1b[17~"
	case tcell.KeyF7:
		return "\x1b[18~"
	case tcell.KeyF8:
		return "\x1b[19~"
	case tcell.KeyF9:
		return "\x1b[20~"
	case tcell.KeyF10:
		return "\x1b[21~"
	case tcell.KeyF11:
		return "\x1b[23~"
	case tcell.KeyF12:
		return "\x1b[24~"
	}

	if ev.Key() >= tcell.KeyCtrlA && ev.Key() <= tcell.KeyCtrlZ {
		return string(rune(ev.Key() - tcell.KeyCtrlA + 1))
	}

	return ""
}

func ParseHexColor(hex string) term.DirectColor {
	if len(hex) == 0 {
		return term.DirectColor{}
	}
	if hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return term.DirectColor{}
	}
	r := hexByte(hex[0:2])
	g := hexByte(hex[2:4])
	b := hexByte(hex[4:6])
	return term.DirectColor{R: r, G: g, B: b, Set: true}
}

func hexByte(s string) byte {
	var v byte
	for _, c := range s {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v |= byte(c - '0')
		case c >= 'a' && c <= 'f':
			v |= byte(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			v |= byte(c - 'A' + 10)
		}
	}
	return v
}

func Build256Palette() [256]term.DirectColor {
	var p [256]term.DirectColor
	// 16-231: 6x6x6 color cube
	for i := 16; i < 232; i++ {
		idx := i - 16
		r := byte((idx / 36) * 51)
		g := byte(((idx / 6) % 6) * 51)
		b := byte((idx % 6) * 51)
		p[i] = term.DirectColor{R: r, G: g, B: b, Set: true}
	}
	// 232-255: grayscale
	for i := 232; i < 256; i++ {
		v := byte((i-232)*10 + 8)
		p[i] = term.DirectColor{R: v, G: v, B: v, Set: true}
	}
	return p
}
