package widgets

import (
	"github.com/gdamore/tcell/v2"
)

type VStackWidget struct {
	Children []Widget
	rect     Rect
}

func NewVStackWidget(children ...Widget) *VStackWidget {
	return &VStackWidget{Children: children}
}

func (v *VStackWidget) Height() int { return 0 }
func (v *VStackWidget) Width() int  { return 0 }

func (v *VStackWidget) SetRect(r Rect) {
	v.rect = r
}

func (v *VStackWidget) GetRect() Rect {
	return v.rect
}

func (v *VStackWidget) Render(surface Surface) {
	w, h := surface.Size()
	if len(v.Children) == 0 || w <= 0 || h <= 0 {
		return
	}

	fixedTotal := 0
	growCount := 0
	for _, child := range v.Children {
		ch := child.Height()
		if ch > 0 {
			fixedTotal += ch
		} else {
			growCount++
		}
	}

	growH := 0
	growRemainder := 0
	if growCount > 0 {
		remaining := h - fixedTotal
		if remaining > 0 {
			growH = remaining / growCount
			growRemainder = remaining % growCount
		}
	}

	y := 0
	growIndex := 0
	for _, child := range v.Children {
		ch := child.Height()
		if ch == 0 {
			ch = growH
			if growIndex < growRemainder {
				ch++
			}
			growIndex++
		}
		if ch <= 0 || y >= h {
			continue
		}
		if y+ch > h {
			ch = h - y
		}
		sub := surface.Sub(Rect{X: 0, Y: y, W: w, H: ch})
		child.SetRect(Rect{X: v.rect.X, Y: v.rect.Y + y, W: w, H: ch})
		child.Render(sub)
		y += ch
	}
}

func (v *VStackWidget) HandleEvent(ev tcell.Event) bool {
	for _, child := range v.Children {
		if child.HandleEvent(ev) {
			return true
		}
	}
	return false
}
