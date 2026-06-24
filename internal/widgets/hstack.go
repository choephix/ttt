package widgets

import (
	"github.com/gdamore/tcell/v2"
)

type HStackWidget struct {
	Children []Widget
	rect     Rect
}

func NewHStackWidget(children ...Widget) *HStackWidget {
	return &HStackWidget{Children: children}
}

func (s *HStackWidget) Height() int { return 0 }
func (s *HStackWidget) Width() int  { return 0 }

func (s *HStackWidget) SetRect(r Rect) {
	s.rect = r
}

func (s *HStackWidget) GetRect() Rect {
	return s.rect
}

func (s *HStackWidget) Render(surface Surface) {
	w, h := surface.Size()
	if len(s.Children) == 0 || w <= 0 || h <= 0 {
		return
	}

	fixedTotal := 0
	growCount := 0
	for _, child := range s.Children {
		cw := child.Width()
		if cw > 0 {
			fixedTotal += cw
		} else {
			growCount++
		}
	}

	growW := 0
	growRemainder := 0
	if growCount > 0 {
		remaining := w - fixedTotal
		if remaining > 0 {
			growW = remaining / growCount
			growRemainder = remaining % growCount
		}
	}

	x := 0
	growIndex := 0
	for _, child := range s.Children {
		cw := child.Width()
		if cw == 0 {
			cw = growW
			if growIndex < growRemainder {
				cw++
			}
			growIndex++
		}
		if cw <= 0 || x >= w {
			continue
		}
		if x+cw > w {
			cw = w - x
		}
		sub := surface.Sub(Rect{X: x, Y: 0, W: cw, H: h})
		child.SetRect(Rect{X: s.rect.X + x, Y: s.rect.Y, W: cw, H: h})
		child.Render(sub)
		x += cw
	}
}

func (s *HStackWidget) HandleEvent(ev tcell.Event) bool {
	for _, child := range s.Children {
		if child.HandleEvent(ev) {
			return true
		}
	}
	return false
}
