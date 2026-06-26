package ui

import (
	"github.com/eugenioenko/ttt/internal/markdown"
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

const hoverMaxVisibleLines = 12

type HoverWidget struct {
	BaseWidget
	Lines      []markdown.Line
	AnchorX    int
	AnchorY    int
	OffsetX    int
	OffsetY    int
	Borders    *term.BorderSet
	scrollTop  int
	scrollLeft int
	maxLineW   int
	vscrollbar Scrollbar
	hscrollbar HScrollbar
}

func NewHoverWidget(text string, x, y int) *HoverWidget {
	lines := markdown.Render(text)
	maxW := 0
	for _, line := range lines {
		if w := len([]rune(line.Text())); w > maxW {
			maxW = w
		}
	}
	return &HoverWidget{
		Lines:    lines,
		AnchorX:  x,
		AnchorY:  y,
		maxLineW: maxW,
	}
}

func (h *HoverWidget) Focusable() bool { return false }

func (h *HoverWidget) visibleLines() int {
	v := len(h.Lines)
	if v > hoverMaxVisibleLines {
		v = hoverMaxVisibleLines
	}
	return v
}

func (h *HoverWidget) Render(surface Surface) {
	if len(h.Lines) == 0 {
		return
	}
	sw, sh := surface.Size()

	visLines := h.visibleLines()
	hasVScroll := len(h.Lines) > visLines

	contentW := h.maxLineW + 2
	maxContentW := sw - 6
	if maxContentW < 20 {
		maxContentW = 20
	}
	if contentW > maxContentW {
		contentW = maxContentW
	}
	hasHScroll := h.maxLineW > contentW

	menuW := contentW + 2
	if hasVScroll {
		menuW++
	}
	menuH := visLines + 2
	if hasHScroll {
		menuH++
	}

	localX := h.AnchorX - h.OffsetX
	localY := h.AnchorY - h.OffsetY

	x := localX
	if x+menuW > sw {
		x = sw - menuW
	}
	if x < 0 {
		x = 0
	}

	spaceAbove := localY
	y := localY - menuH
	if y < 0 {
		if spaceAbove < menuH {
			y = localY + 1
			if y+menuH > sh {
				menuH = sh - y
				visLines = menuH - 2
				if hasHScroll {
					visLines--
				}
			}
		} else {
			y = 0
		}
	}

	b := term.SingleBorderSet()
	if h.Borders != nil {
		b = *h.Borders
	}
	surface.DrawBorder(x, y, menuW, menuH, b, term.StyleBorder)
	st := term.StylePaletteItem

	for i := 0; i < visLines; i++ {
		lineIdx := h.scrollTop + i
		if lineIdx >= len(h.Lines) {
			break
		}
		row := y + 1 + i
		innerW := menuW - 2
		if hasVScroll {
			innerW--
		}
		lineText := h.Lines[lineIdx].Text()
		if lineText == "---" && len(h.Lines[lineIdx].Spans) == 1 && h.Lines[lineIdx].Spans[0].Style == term.StyleBorder {
			surface.SetCell(x, row, term.Cell{Ch: b.LeftTee, Style: term.StyleBorder})
			for bx := x + 1; bx < x+1+innerW; bx++ {
				surface.SetCell(bx, row, term.Cell{Ch: b.Horizontal, Style: term.StyleBorder})
			}
			surface.SetCell(x+menuW-1, row, term.Cell{Ch: b.RightTee, Style: term.StyleBorder})
			continue
		}
		for bx := x + 1; bx < x+1+innerW; bx++ {
			surface.SetCell(bx, row, term.Cell{Ch: ' ', Style: st})
		}
		styles := buildStyleRuns(h.Lines[lineIdx])
		runes := []rune(lineText)
		for j := 0; j < contentW && h.scrollLeft+j < len(runes); j++ {
			idx := h.scrollLeft + j
			cellStyle := st
			if idx < len(styles) {
				cellStyle = styles[idx]
			}
			surface.SetCell(x+1+j, row, term.Cell{Ch: runes[idx], Style: cellStyle})
		}
	}

	if hasVScroll {
		h.vscrollbar.X = h.OffsetX + x + menuW - 2
		h.vscrollbar.Y = h.OffsetY + y + 1
		h.vscrollbar.Height = visLines
		h.vscrollbar.TotalItems = len(h.Lines)
		h.vscrollbar.TopItem = h.scrollTop
		h.vscrollbar.Render(surface, x+menuW-2, y+1)
	}

	if hasHScroll {
		trackW := menuW - 2
		if hasVScroll {
			trackW--
		}
		h.hscrollbar.X = h.OffsetX + x + 1
		h.hscrollbar.Y = h.OffsetY + y + menuH - 2
		h.hscrollbar.Width = trackW
		h.hscrollbar.TotalCols = h.maxLineW
		h.hscrollbar.LeftCol = h.scrollLeft
		h.hscrollbar.Render(surface, x+1, y+menuH-2)
	}

	h.SetRect(Rect{X: h.OffsetX + x, Y: h.OffsetY + y, W: menuW, H: menuH})
}

func (h *HoverWidget) HandleEvent(ev tcell.Event) EventResult {
	if newTop, consumed := h.vscrollbar.HandleEvent(ev); consumed {
		h.scrollTop = newTop
		return EventConsumed
	}
	if newLeft, consumed := h.hscrollbar.HandleEvent(ev); consumed {
		h.scrollLeft = newLeft
		return EventConsumed
	}

	switch tev := ev.(type) {
	case *tcell.EventKey:
		return EventDismissed
	case *tcell.EventMouse:
		mx, my := tev.Position()
		r := h.GetRect()
		inside := mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H
		btn := tev.Buttons()
		if inside {
			if btn&tcell.WheelUp != 0 {
				if h.scrollTop > 0 {
					h.scrollTop--
				}
				return EventConsumed
			}
			if btn&tcell.WheelDown != 0 {
				max := len(h.Lines) - h.visibleLines()
				if max < 0 {
					max = 0
				}
				if h.scrollTop < max {
					h.scrollTop++
				}
				return EventConsumed
			}
			if btn&tcell.WheelLeft != 0 {
				if h.scrollLeft > 0 {
					h.scrollLeft -= 4
					if h.scrollLeft < 0 {
						h.scrollLeft = 0
					}
				}
				return EventConsumed
			}
			if btn&tcell.WheelRight != 0 {
				max := h.maxLineW - h.contentWidth()
				if max < 0 {
					max = 0
				}
				if h.scrollLeft < max {
					h.scrollLeft += 4
					if h.scrollLeft > max {
						h.scrollLeft = max
					}
				}
				return EventConsumed
			}
			return EventConsumed
		}
		if h.vscrollbar.IsDragging() || h.hscrollbar.IsDragging() {
			return EventConsumed
		}
		if btn != tcell.ButtonNone {
			return EventDismissed
		}
	}
	return EventIgnored
}

func (h *HoverWidget) IsDragging() bool {
	return h.vscrollbar.IsDragging() || h.hscrollbar.IsDragging()
}

func (h *HoverWidget) contentWidth() int {
	r := h.GetRect()
	w := r.W - 2
	if len(h.Lines) > h.visibleLines() {
		w--
	}
	return w
}

func buildStyleRuns(line markdown.Line) []term.Style {
	var styles []term.Style
	for _, span := range line.Spans {
		for range []rune(span.Text) {
			styles = append(styles, span.Style)
		}
	}
	return styles
}
