package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type BoxWidget struct {
	BaseWidget
	Child Widget
}

func NewBoxWidget(bm BoxModel) *BoxWidget {
	return &BoxWidget{BaseWidget: BaseWidget{Box: bm}}
}

func NewBoxWithBorder(borders term.BorderSet) *BoxWidget {
	return NewBoxWidget(BoxModel{
		BorderTop: true, BorderBottom: true, BorderLeft: true, BorderRight: true,
		Borders: borders,
	})
}

func NewBoxWithPadding(padding int) *BoxWidget {
	return NewBoxWidget(BoxModel{
		PaddingTop: padding, PaddingBottom: padding, PaddingLeft: padding, PaddingRight: padding,
	})
}

func NewBoxWithBorderAndPadding(borders term.BorderSet, padding int) *BoxWidget {
	return NewBoxWidget(BoxModel{
		BorderTop: true, BorderBottom: true, BorderLeft: true, BorderRight: true,
		PaddingTop: padding, PaddingBottom: padding, PaddingLeft: padding, PaddingRight: padding,
		Borders: borders,
	})
}

func (b *BoxWidget) Height() int {
	if b.Child != nil {
		ch := b.Child.Height()
		if ch > 0 {
			return ch + b.BoxOverheadH()
		}
	}
	return 0
}

func (b *BoxWidget) Width() int {
	if b.Child != nil {
		cw := b.Child.Width()
		if cw > 0 {
			return cw + b.BoxOverheadW()
		}
	}
	return 0
}

func (b *BoxWidget) Render(surface Surface) {
	inner := b.RenderBox(surface)
	iw, ih := inner.Size()
	if b.Child != nil && iw > 0 && ih > 0 {
		r := b.GetRect()
		ox := b.Box.MarginLeft + b.Box.PaddingLeft
		oy := b.Box.MarginTop + b.Box.PaddingTop
		if b.Box.BorderLeft {
			ox++
		}
		if b.Box.BorderTop {
			oy++
		}
		b.Child.SetRect(Rect{X: r.X + ox, Y: r.Y + oy, W: iw, H: ih})
		b.Child.Render(inner)
	}
}

func (b *BoxWidget) HandleEvent(ev tcell.Event) bool {
	if b.Child != nil {
		return b.Child.HandleEvent(ev)
	}
	return false
}
