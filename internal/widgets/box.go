package widgets

import (
	"github.com/gdamore/tcell/v2"
)

type BoxConfig struct {
	PaddingTop    int `json:"paddingTop,omitempty"`
	PaddingBottom int `json:"paddingBottom,omitempty"`
	PaddingLeft   int `json:"paddingLeft,omitempty"`
	PaddingRight  int `json:"paddingRight,omitempty"`
}

type BoxWidget struct {
	Config BoxConfig
	Child  *TreeWidget
	rect   Rect
}

func NewBoxWidget(config BoxConfig) *BoxWidget {
	return &BoxWidget{Config: config}
}

func (b *BoxWidget) SetRect(r Rect) {
	b.rect = r
}

func (b *BoxWidget) GetRect() Rect {
	return b.rect
}

func (b *BoxWidget) Render(surface Surface) {
	w, h := surface.Size()

	innerX := b.Config.PaddingLeft
	innerY := b.Config.PaddingTop
	innerW := w - b.Config.PaddingLeft - b.Config.PaddingRight
	innerH := h - b.Config.PaddingTop - b.Config.PaddingBottom

	if b.Child == nil || innerW <= 0 || innerH <= 0 {
		return
	}

	inner := surface.Sub(Rect{X: innerX, Y: innerY, W: innerW, H: innerH})
	b.Child.SetRect(Rect{X: b.rect.X + innerX, Y: b.rect.Y + innerY, W: innerW, H: innerH})
	b.Child.Render(inner)
}

func (b *BoxWidget) HandleEvent(ev tcell.Event) bool {
	if b.Child != nil {
		return b.Child.HandleEvent(ev)
	}
	return false
}
