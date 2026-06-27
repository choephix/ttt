package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type ScrollViewWidget struct {
	BaseWidget
	Child ScrollableWidget

	scrollX      int
	scrollY      int
	vbar         scrollbar
	focused      bool
	lastContentH int
	lastViewH    int
}

func NewScrollViewWidget(child ScrollableWidget) *ScrollViewWidget {
	return &ScrollViewWidget{Child: child}
}

func (sv *ScrollViewWidget) Height() int { return 0 }
func (sv *ScrollViewWidget) Width() int  { return 0 }

func (sv *ScrollViewWidget) EnsureVisible(x, y int) {
	r := sv.rect
	contentW, contentH := sv.Child.ScrollSize()
	viewW, viewH := sv.viewportSize(r.W, r.H, contentW, contentH)

	if y < sv.scrollY {
		sv.scrollY = y
	}
	if y >= sv.scrollY+viewH {
		sv.scrollY = y - viewH + 1
	}
	if x < sv.scrollX {
		sv.scrollX = x
	}
	if x >= sv.scrollX+viewW {
		sv.scrollX = x - viewW + 1
	}
}

func (sv *ScrollViewWidget) viewportSize(w, h, contentW, contentH int) (int, int) {
	vw, vh := w, h
	if contentH > h {
		vw--
	}
	if contentW > vw {
		vh--
	}
	if contentH > vh && vw == w {
		vw--
	}
	if vw < 0 {
		vw = 0
	}
	if vh < 0 {
		vh = 0
	}
	return vw, vh
}

func (sv *ScrollViewWidget) Render(surface Surface) {
	w, h := surface.Size()
	if w <= 0 || h <= 0 || sv.Child == nil {
		return
	}

	contentW, contentH := sv.Child.ScrollSize()
	viewW, viewH := sv.viewportSize(w, h, contentW, contentH)
	hasVBar := contentH > viewH
	hasHBar := contentW > viewW

	sv.lastContentH = contentH
	sv.lastViewH = viewH
	sv.clamp(contentW, contentH, viewW, viewH)

	virt := newVirtualSurface(contentW, contentH)
	sv.Child.SetRect(Rect{X: 0, Y: 0, W: contentW, H: contentH})
	sv.Child.Render(virt)

	for y := range viewH {
		for x := range viewW {
			sx, sy := x+sv.scrollX, y+sv.scrollY
			if sx < contentW && sy < contentH {
				surface.SetCell(x, y, virt.cells[sy][sx])
			}
		}
	}

	if hasVBar {
		sv.vbar.X = sv.rect.X + w - 1
		sv.vbar.Y = sv.rect.Y
		sv.vbar.Height = viewH
		sv.vbar.TotalItems = contentH
		sv.vbar.TopItem = sv.scrollY
		sv.vbar.Render(surface, w-1, 0)
	}

	if hasHBar {
		sv.renderHBar(surface, viewW, h-1, contentW)
	}
}

func (sv *ScrollViewWidget) renderHBar(surface Surface, barW, y, totalW int) {
	if totalW <= barW || barW <= 0 {
		return
	}
	thumbH := barW * barW / totalW
	if thumbH < 1 {
		thumbH = 1
	}
	scrollable := totalW - barW
	thumbPos := sv.scrollX * (barW - thumbH) / scrollable
	if thumbPos+thumbH > barW {
		thumbPos = barW - thumbH
	}

	for x := range barW {
		style := term.StyleScrollbar
		if x >= thumbPos && x < thumbPos+thumbH {
			style = term.StyleScrollbarThumb
		}
		surface.SetCell(x, y, term.Cell{Ch: '█', Style: style})
	}
}

func (sv *ScrollViewWidget) HandleEvent(ev tcell.Event) EventResult {
	if newTop, consumed := sv.vbar.HandleEvent(ev); consumed {
		sv.scrollY = newTop
		return EventConsumed
	}

	switch e := ev.(type) {
	case *tcell.EventMouse:
		btn := e.Buttons()
		if btn&tcell.WheelUp != 0 {
			sv.scrollY -= 3
			if sv.scrollY < 0 {
				sv.scrollY = 0
			}
			return EventConsumed
		}
		if btn&tcell.WheelDown != 0 {
			sv.scrollY += 3
			if sv.lastContentH > 0 {
				maxY := sv.lastContentH - sv.lastViewH
				if maxY < 0 {
					maxY = 0
				}
				if sv.scrollY > maxY {
					sv.scrollY = maxY
				}
			}
			return EventConsumed
		}
	}

	if sv.Child != nil {
		return sv.Child.HandleEvent(ev)
	}
	return EventIgnored
}

func (sv *ScrollViewWidget) Focusable() bool           { return true }
func (sv *ScrollViewWidget) SetFocused(focused bool)    { sv.focused = focused }
func (sv *ScrollViewWidget) IsFocused() bool            { return sv.focused }

func (sv *ScrollViewWidget) clamp(contentW, contentH, viewW, viewH int) {
	maxY := contentH - viewH
	if maxY < 0 {
		maxY = 0
	}
	if sv.scrollY > maxY {
		sv.scrollY = maxY
	}
	if sv.scrollY < 0 {
		sv.scrollY = 0
	}

	maxX := contentW - viewW
	if maxX < 0 {
		maxX = 0
	}
	if sv.scrollX > maxX {
		sv.scrollX = maxX
	}
	if sv.scrollX < 0 {
		sv.scrollX = 0
	}
}
