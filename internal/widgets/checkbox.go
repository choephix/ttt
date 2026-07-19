package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

// Bordered checkbox geometry: left border, padding, mark, padding, right border.
const (
	checkboxPad  = 1
	checkboxBoxW = 1 + checkboxPad + 1 + checkboxPad + 1
)

type CheckboxConfig struct {
	Label string
	// Bordered draws the box as a 3x3 bordered square with the label to its
	// right, so it lines up with bordered inputs and selects in a form.
	Bordered bool
	Checked  bool
	Style    term.Style `json:"-"`
	OnChange func(checked bool)
}

type CheckboxWidget struct {
	BaseWidget
	Config  CheckboxConfig
	focused bool
}

func NewCheckboxWidget(config CheckboxConfig) *CheckboxWidget {
	return &CheckboxWidget{Config: config}
}

func (c *CheckboxWidget) Height() int {
	if c.Config.Bordered {
		return 3 + c.BoxOverheadH()
	}
	return 1 + c.BoxOverheadH()
}

// Only the border reflects focus, matching InputWidget; the mark and label keep
// default styling so the row does not flash a highlighted background.
func (c *CheckboxWidget) renderBordered(inner Surface) {
	w, h := inner.Size()
	if w < checkboxBoxW+2 || h < 3 {
		return
	}

	borderStyle := term.StyleBorder
	if c.focused {
		borderStyle = term.StyleBorderActive
	}
	textStyle := c.Config.Style
	if textStyle == 0 {
		textStyle = term.StyleDefault
	}

	mark := ' '
	if c.Config.Checked {
		mark = 'x'
	}
	inner.DrawBorder(0, 0, checkboxBoxW, 3, widgetBorders(c.Box), borderStyle)
	inner.SetCell(1+checkboxPad, 1, term.Cell{Ch: mark, Style: textStyle})

	x := checkboxBoxW + 1
	for _, ch := range c.Config.Label {
		if x >= w {
			break
		}
		inner.SetCell(x, 1, term.Cell{Ch: ch, Style: textStyle})
		x++
	}
}
func (c *CheckboxWidget) Width() int { return 0 }

func (c *CheckboxWidget) Focusable() bool   { return true }
func (c *CheckboxWidget) SetFocused(f bool) { c.focused = f }
func (c *CheckboxWidget) IsFocused() bool   { return c.focused }

func (c *CheckboxWidget) Render(surface Surface) {
	inner := c.RenderBox(surface)
	w, _ := inner.Size()
	if w < 4 {
		return
	}

	if c.Config.Bordered {
		c.renderBordered(inner)
		return
	}

	style := c.Config.Style
	if style == 0 {
		style = term.StyleDefault
	}
	if c.focused {
		style = term.StyleButtonFocused
	}

	for x := range w {
		inner.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
	}

	mark := ' '
	if c.Config.Checked {
		mark = 'x'
	}
	inner.SetCell(0, 0, term.Cell{Ch: ' ', Style: style})
	inner.SetCell(1, 0, term.Cell{Ch: '[', Style: style})
	inner.SetCell(2, 0, term.Cell{Ch: mark, Style: style})
	inner.SetCell(3, 0, term.Cell{Ch: ']', Style: style})
	inner.SetCell(4, 0, term.Cell{Ch: ' ', Style: style})

	x := 5
	for _, ch := range c.Config.Label {
		if x >= w {
			break
		}
		inner.SetCell(x, 0, term.Cell{Ch: ch, Style: style})
		x++
	}
}

func (c *CheckboxWidget) toggle() {
	c.Config.Checked = !c.Config.Checked
	if c.Config.OnChange != nil {
		c.Config.OnChange(c.Config.Checked)
	}
}

func (c *CheckboxWidget) HandleEvent(ev tcell.Event) EventResult {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		if !c.focused {
			return EventIgnored
		}
		if tev.Key() == tcell.KeyEnter || (tev.Key() == tcell.KeyRune && tev.Rune() == ' ') {
			c.toggle()
			return EventConsumed
		}
	case *tcell.EventMouse:
		if tev.Buttons()&tcell.Button1 != 0 {
			mx, my := tev.Position()
			r := c.GetRect()
			if mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H {
				c.toggle()
				return EventConsumed
			}
		}
	}
	return EventIgnored
}
