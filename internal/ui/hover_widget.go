package ui

import (
	"strings"

	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

const hoverMaxVisibleLines = 12
const hoverMaxWidth = 60

type HoverWidget struct {
	BaseWidget
	Lines     []string
	AnchorX   int
	AnchorY   int
	OffsetX   int
	OffsetY   int
	Borders   *term.BorderSet
	scrollTop  int
	scrollLeft int
	maxLineW   int
}

func NewHoverWidget(text string, x, y int) *HoverWidget {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	maxW := 0
	for _, line := range lines {
		if w := len([]rune(line)); w > maxW {
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

func (h *HoverWidget) Render(surface *RenderSurface) {
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
			y = h.AnchorY + 1
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
		if h.Lines[lineIdx] == "---" {
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
		runes := []rune(h.Lines[lineIdx])
		for j := 0; j < contentW && h.scrollLeft+j < len(runes); j++ {
			surface.SetCell(x+1+j, row, term.Cell{Ch: runes[h.scrollLeft+j], Style: st})
		}
	}

	if hasVScroll {
		sb := Scrollbar{
			X:          x + menuW - 2,
			Y:          y + 1,
			Height:     visLines,
			TotalItems: len(h.Lines),
			TopItem:    h.scrollTop,
		}
		sb.Render(surface, x+menuW-2, y+1)
	}

	if hasHScroll {
		hScrollRow := y + menuH - 2
		trackW := menuW - 2
		if hasVScroll {
			trackW--
		}
		maxScroll := h.maxLineW - contentW
		if maxScroll < 1 {
			maxScroll = 1
		}
		thumbW := trackW * contentW / h.maxLineW
		if thumbW < 1 {
			thumbW = 1
		}
		thumbX := 0
		if maxScroll > 0 {
			thumbX = h.scrollLeft * (trackW - thumbW) / maxScroll
		}
		for i := 0; i < trackW; i++ {
			ch := ' '
			style := term.StyleScrollbar
			if i >= thumbX && i < thumbX+thumbW {
				ch = '█'
				style = term.StyleScrollbarThumb
			}
			surface.SetCell(x+1+i, hScrollRow, term.Cell{Ch: rune(ch), Style: style})
		}
	}

	h.SetRect(Rect{X: x, Y: y, W: menuW, H: menuH})
}

func (h *HoverWidget) HandleEvent(ev tcell.Event) EventResult {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		return EventDismissed
	case *tcell.EventMouse:
		btn := tev.Buttons()
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
			visW := h.visibleContentWidth()
			max := h.maxLineW - visW
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
		if btn != tcell.ButtonNone {
			return EventDismissed
		}
	}
	return EventIgnored
}

func (h *HoverWidget) visibleContentWidth() int {
	sw, _ := h.GetRect().W, h.GetRect().H
	w := sw - 2
	if len(h.Lines) > h.visibleLines() {
		w--
	}
	return w
}
