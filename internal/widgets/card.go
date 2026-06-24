package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type CardConfig struct {
	Borders term.BorderSet `json:"-"`
	Style   term.Style     `json:"-"`
}

type CardWidget struct {
	Config CardConfig
	Child  *TreeWidget
	rect   Rect
}

func NewCardWidget(config CardConfig) *CardWidget {
	return &CardWidget{Config: config}
}

func (c *CardWidget) SetRect(r Rect) {
	c.rect = r
}

func (c *CardWidget) GetRect() Rect {
	return c.rect
}

func (c *CardWidget) Render(surface Surface) {
	w, h := surface.Size()
	if w < 2 || h < 2 {
		return
	}

	borderStyle := c.Config.Style
	if borderStyle == 0 {
		borderStyle = term.StyleBorder
	}

	surface.DrawBorder(0, 0, w, h, c.Config.Borders, borderStyle)

	if c.Child != nil && w > 2 && h > 2 {
		inner := surface.Sub(Rect{X: 1, Y: 1, W: w - 2, H: h - 2})
		c.Child.SetRect(Rect{X: c.rect.X + 1, Y: c.rect.Y + 1, W: w - 2, H: h - 2})
		c.Child.Render(inner)
	}
}

func (c *CardWidget) HandleEvent(ev tcell.Event) bool {
	if c.Child != nil {
		return c.Child.HandleEvent(ev)
	}
	return false
}
