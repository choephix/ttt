package ui

import (
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type Scrollbar struct {
	X          int // absolute screen X for hit testing
	Y          int // absolute screen Y for hit testing
	Height     int
	TotalItems int
	TopItem    int
	dragging   bool
	dragOffset int
}

func (s *Scrollbar) Visible() bool {
	return s.TotalItems > s.Height && s.Height > 0
}

func (s *Scrollbar) ThumbPos() (top, height int) {
	if s.TotalItems <= s.Height {
		return 0, s.Height
	}
	height = s.Height * s.Height / s.TotalItems
	if height < 1 {
		height = 1
	}
	scrollable := s.TotalItems - s.Height
	top = s.TopItem * (s.Height - height) / scrollable
	if top+height > s.Height {
		top = s.Height - height
	}
	return
}

func (s *Scrollbar) Render(surface *RenderSurface, rx, ry int) {
	if !s.Visible() {
		return
	}
	thumbTop, thumbH := s.ThumbPos()
	for y := 0; y < s.Height; y++ {
		if y >= thumbTop && y < thumbTop+thumbH {
			surface.SetCell(rx, ry+y, term.Cell{Ch: '█', Style: term.StyleScrollbarThumb})
		} else {
			surface.SetCell(rx, ry+y, term.Cell{Ch: ' ', Style: term.StyleScrollbar})
		}
	}
}

func (s *Scrollbar) HandleEvent(ev tcell.Event) (newTopItem int, consumed bool) {
	mev, ok := ev.(*tcell.EventMouse)
	if !ok {
		return s.TopItem, false
	}

	mx, my := mev.Position()
	btn := mev.Buttons()

	if s.dragging {
		if btn == tcell.ButtonNone {
			s.dragging = false
			return s.TopItem, false
		}
		if btn&tcell.Button1 != 0 {
			relY := my - s.Y
			return s.posToTopItem(relY - s.dragOffset), true
		}
	}

	if btn&tcell.Button1 != 0 && mx == s.X && my >= s.Y && my < s.Y+s.Height {
		relY := my - s.Y
		thumbTop, thumbH := s.ThumbPos()

		s.dragging = true
		if relY >= thumbTop && relY < thumbTop+thumbH {
			s.dragOffset = relY - thumbTop
		} else {
			s.dragOffset = thumbH / 2
			return s.posToTopItem(relY - s.dragOffset), true
		}
		return s.TopItem, true
	}

	return s.TopItem, false
}

func (s *Scrollbar) IsDragging() bool { return s.dragging }

func (s *Scrollbar) posToTopItem(thumbTop int) int {
	_, thumbH := s.ThumbPos()
	maxThumbTop := s.Height - thumbH
	if maxThumbTop <= 0 {
		return 0
	}
	if thumbTop < 0 {
		thumbTop = 0
	}
	if thumbTop > maxThumbTop {
		thumbTop = maxThumbTop
	}
	scrollable := s.TotalItems - s.Height
	top := thumbTop * scrollable / maxThumbTop
	if top < 0 {
		top = 0
	}
	if top > scrollable {
		top = scrollable
	}
	return top
}
