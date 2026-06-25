package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type CheckboxConfig struct {
	Label    string
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

func (c *CheckboxWidget) Height() int { return 1 + c.BoxOverheadH() }
func (c *CheckboxWidget) Width() int  { return 0 }

func (c *CheckboxWidget) Focusable() bool   { return true }
func (c *CheckboxWidget) SetFocused(f bool) { c.focused = f }
func (c *CheckboxWidget) IsFocused() bool   { return c.focused }

func (c *CheckboxWidget) Render(surface Surface) {
	inner := c.RenderBox(surface)
	w, _ := inner.Size()
	if w < 4 {
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

func (c *CheckboxWidget) HandleEvent(ev tcell.Event) bool {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		if tev.Key() == tcell.KeyEnter || (tev.Key() == tcell.KeyRune && tev.Rune() == ' ') {
			c.toggle()
			return true
		}
	case *tcell.EventMouse:
		if tev.Buttons()&tcell.Button1 != 0 {
			mx, my := tev.Position()
			r := c.GetRect()
			if mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H {
				c.toggle()
				return true
			}
		}
	}
	return false
}
