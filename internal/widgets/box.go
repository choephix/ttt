package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type BoxConfig struct {
	BorderTop    bool `json:"borderTop,omitempty"`
	BorderBottom bool `json:"borderBottom,omitempty"`
	BorderLeft   bool `json:"borderLeft,omitempty"`
	BorderRight  bool `json:"borderRight,omitempty"`

	PaddingTop    int `json:"paddingTop,omitempty"`
	PaddingBottom int `json:"paddingBottom,omitempty"`
	PaddingLeft   int `json:"paddingLeft,omitempty"`
	PaddingRight  int `json:"paddingRight,omitempty"`

	MarginTop    int `json:"marginTop,omitempty"`
	MarginBottom int `json:"marginBottom,omitempty"`
	MarginLeft   int `json:"marginLeft,omitempty"`
	MarginRight  int `json:"marginRight,omitempty"`

	Borders term.BorderSet `json:"-"`
	Style   term.Style     `json:"-"`
}

type BoxWidget struct {
	Config BoxConfig
	Child  Widget
	rect   Rect
}

func NewBoxWidget(config BoxConfig) *BoxWidget {
	return &BoxWidget{Config: config}
}

func NewBoxWithBorder(borders term.BorderSet) *BoxWidget {
	return &BoxWidget{Config: BoxConfig{
		BorderTop: true, BorderBottom: true, BorderLeft: true, BorderRight: true,
		Borders: borders,
	}}
}

func NewBoxWithPadding(padding int) *BoxWidget {
	return &BoxWidget{Config: BoxConfig{
		PaddingTop: padding, PaddingBottom: padding, PaddingLeft: padding, PaddingRight: padding,
	}}
}

func NewBoxWithBorderAndPadding(borders term.BorderSet, padding int) *BoxWidget {
	return &BoxWidget{Config: BoxConfig{
		BorderTop: true, BorderBottom: true, BorderLeft: true, BorderRight: true,
		PaddingTop: padding, PaddingBottom: padding, PaddingLeft: padding, PaddingRight: padding,
		Borders: borders,
	}}
}

func (b *BoxWidget) Height() int { return 0 }
func (b *BoxWidget) Width() int  { return 0 }

func (b *BoxWidget) SetRect(r Rect) {
	b.rect = r
}

func (b *BoxWidget) GetRect() Rect {
	return b.rect
}

func (b *BoxWidget) Render(surface Surface) {
	w, h := surface.Size()

	borderStyle := b.Config.Style
	if borderStyle == 0 {
		borderStyle = term.StyleBorder
	}
	bs := b.Config.Borders

	mx := b.Config.MarginLeft
	my := b.Config.MarginTop
	mw := w - b.Config.MarginLeft - b.Config.MarginRight
	mh := h - b.Config.MarginTop - b.Config.MarginBottom
	if mw <= 0 || mh <= 0 {
		return
	}

	bTop := 0
	bBottom := 0
	bLeft := 0
	bRight := 0
	if b.Config.BorderTop {
		bTop = 1
	}
	if b.Config.BorderBottom {
		bBottom = 1
	}
	if b.Config.BorderLeft {
		bLeft = 1
	}
	if b.Config.BorderRight {
		bRight = 1
	}

	if b.Config.BorderTop {
		for x := mx + bLeft; x < mx+mw-bRight; x++ {
			surface.SetCell(x, my, term.Cell{Ch: bs.Horizontal, Style: borderStyle})
		}
	}
	if b.Config.BorderBottom {
		for x := mx + bLeft; x < mx+mw-bRight; x++ {
			surface.SetCell(x, my+mh-1, term.Cell{Ch: bs.Horizontal, Style: borderStyle})
		}
	}
	if b.Config.BorderLeft {
		for y := my + bTop; y < my+mh-bBottom; y++ {
			surface.SetCell(mx, y, term.Cell{Ch: bs.Vertical, Style: borderStyle})
		}
	}
	if b.Config.BorderRight {
		for y := my + bTop; y < my+mh-bBottom; y++ {
			surface.SetCell(mx+mw-1, y, term.Cell{Ch: bs.Vertical, Style: borderStyle})
		}
	}

	if b.Config.BorderTop && b.Config.BorderLeft {
		surface.SetCell(mx, my, term.Cell{Ch: bs.TopLeft, Style: borderStyle})
	}
	if b.Config.BorderTop && b.Config.BorderRight {
		surface.SetCell(mx+mw-1, my, term.Cell{Ch: bs.TopRight, Style: borderStyle})
	}
	if b.Config.BorderBottom && b.Config.BorderLeft {
		surface.SetCell(mx, my+mh-1, term.Cell{Ch: bs.BottomLeft, Style: borderStyle})
	}
	if b.Config.BorderBottom && b.Config.BorderRight {
		surface.SetCell(mx+mw-1, my+mh-1, term.Cell{Ch: bs.BottomRight, Style: borderStyle})
	}

	innerX := mx + bLeft + b.Config.PaddingLeft
	innerY := my + bTop + b.Config.PaddingTop
	innerW := mw - bLeft - bRight - b.Config.PaddingLeft - b.Config.PaddingRight
	innerH := mh - bTop - bBottom - b.Config.PaddingTop - b.Config.PaddingBottom

	if b.Child != nil && innerW > 0 && innerH > 0 {
		inner := surface.Sub(Rect{X: innerX, Y: innerY, W: innerW, H: innerH})
		b.Child.SetRect(Rect{X: b.rect.X + innerX, Y: b.rect.Y + innerY, W: innerW, H: innerH})
		b.Child.Render(inner)
	}
}

func (b *BoxWidget) HandleEvent(ev tcell.Event) bool {
	if b.Child != nil {
		return b.Child.HandleEvent(ev)
	}
	return false
}
