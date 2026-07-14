package ui

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type tabSpan struct {
	start, end int
	label      string
	active     bool
}

type Tab struct {
	Name     string
	Dirty    bool
	Active   bool
	Closable bool
	Preview  bool
}

type TabBarWidget struct {
	BaseWidget
	Tabs              []Tab
	Borders           *term.BorderSet
	ScrollOffset      int
	MoreButton        *MoreButtonWidget
	OnTabClick        func(index int)
	OnTabClose        func(index int)
	OnTabDoubleClick  func(index int)
	OnTabRightClick   func(index, screenX, screenY int)
	OnPrevTab         func()
	OnNextTab         func()
	OnEmptySpaceClick func()
	tabSpans          []tabSpan
	renderArrowW      int // arrow-gutter width from the last Render, reused by HandleEvent
	renderInnerRight  int // right edge of the tab zone from the last Render, reused by HandleEvent
	hasOverflowLeft   bool
	hasOverflowRight  bool
	totalTabWidth     int
	closeDownX        int // screen X where mouse-down hit a close button, -1 if none
	closeDownY        int
	wasPressed        bool
	lastTabClickTime  int64
	lastTabClick      int
}

func NewTabBarWidget() *TabBarWidget {
	return &TabBarWidget{closeDownX: -1, lastTabClick: -1}
}

func (t *TabBarWidget) SetTabs(tabs []Tab) {
	t.Tabs = tabs
}

func (t *TabBarWidget) tabLabel(tab Tab) string {
	name := filepath.Base(tab.Name)
	label := " "
	if tab.Dirty {
		label += "● "
	}
	label += name
	if tab.Active && tab.Closable {
		label += " x"
	}
	label += " "
	return label
}

func (t *TabBarWidget) Render(surface Surface) {
	w, _ := surface.Size()

	b := term.SingleBorderSet()
	if t.Borders != nil {
		b = *t.Borders
	}
	bs := term.StyleBorder

	// Compute spans: active tab needs 2 extra cols for │ │ side borders
	spans := make([]tabSpan, len(t.Tabs))
	pos := 0
	activeIdx := -1
	for i, tab := range t.Tabs {
		label := t.tabLabel(tab)
		labelW := len([]rune(label))
		spanW := labelW
		if tab.Active {
			spanW += 2
		}
		spans[i] = tabSpan{start: pos, end: pos + spanW, label: label, active: tab.Active}
		if tab.Active {
			activeIdx = i
		}
		pos += spanW
	}
	t.tabSpans = spans

	t.totalTabWidth = pos

	// Overflow is measured against the space left after the ⋮ MoreButton, so the
	// arrow gutter is reserved before tabs can overlap the chevron. (issue #354)
	moreW := 0
	if t.MoreButton != nil && w >= 5 {
		moreW = 4
	}
	hasOverflow := pos > w-moreW
	arrowW := 0
	if hasOverflow {
		arrowW = 3 // " ◀ " or " ▶ "
	}
	innerLeft := arrowW
	t.renderArrowW = arrowW
	innerRight := w - moreW - arrowW
	t.renderInnerRight = innerRight
	innerW := innerRight - innerLeft
	if innerW < 1 {
		innerW = 1
	}

	// Scroll to keep active tab visible within the inner zone
	if activeIdx >= 0 {
		s := spans[activeIdx]
		if s.end-t.ScrollOffset > innerW {
			t.ScrollOffset = s.end - innerW
		}
		if s.start < t.ScrollOffset {
			t.ScrollOffset = s.start
		}
	}
	// Never scroll past the last tab, so closing tabs can't strand the view on
	// the final tab with empty space to its right.
	if maxScroll := pos - innerW; t.ScrollOffset > maxScroll {
		t.ScrollOffset = maxScroll
	}
	if t.ScrollOffset < 0 {
		t.ScrollOffset = 0
	}

	t.hasOverflowLeft = t.ScrollOffset > 0
	t.hasOverflowRight = pos-t.ScrollOffset > innerW

	// Row 0: top of active tab ┌───┐, spaces elsewhere
	for x := 0; x < w; x++ {
		surface.SetCell(x, 0, term.Cell{Ch: ' '})
	}
	if activeIdx >= 0 {
		s := spans[activeIdx]
		sx := s.start - t.ScrollOffset + innerLeft
		ex := s.end - t.ScrollOffset + innerLeft
		if sx >= innerLeft && sx < innerRight {
			surface.SetCell(sx, 0, term.Cell{Ch: b.TopLeft, Style: bs})
		}
		for x := sx + 1; x < ex-1; x++ {
			if x >= innerLeft && x < innerRight {
				surface.SetCell(x, 0, term.Cell{Ch: b.Horizontal, Style: bs})
			}
		}
		if ex-1 > sx && ex-1 >= innerLeft && ex-1 < innerRight {
			surface.SetCell(ex-1, 0, term.Cell{Ch: b.TopRight, Style: bs})
		}
	}

	// Row 1: tab labels, active tab has │ sides
	for x := 0; x < w; x++ {
		surface.SetCell(x, 1, term.Cell{Ch: ' '})
	}
	for i, s := range spans {
		sx := s.start - t.ScrollOffset + innerLeft
		ex := s.end - t.ScrollOffset + innerLeft
		dirty := t.Tabs[i].Dirty
		if s.active {
			if sx >= innerLeft && sx < innerRight {
				surface.SetCell(sx, 1, term.Cell{Ch: b.Vertical, Style: bs})
			}
			for ci, ch := range []rune(s.label) {
				style := term.StyleActiveTab
				if dirty && ch == '●' {
					style = term.StyleWarning
				}
				x := sx + 1 + ci
				if x >= innerLeft && x < innerRight {
					surface.SetCell(x, 1, term.Cell{Ch: ch, Style: style, Italic: t.Tabs[i].Preview})
				}
			}
			if ex-1 >= innerLeft && ex-1 < innerRight {
				surface.SetCell(ex-1, 1, term.Cell{Ch: b.Vertical, Style: bs})
			}
		} else {
			for ci, ch := range []rune(s.label) {
				style := term.StyleInactiveTab
				if dirty && ch == '●' {
					style = term.StyleWarning
				}
				x := sx + ci
				if x >= innerLeft && x < innerRight {
					surface.SetCell(x, 1, term.Cell{Ch: ch, Style: style, Italic: t.Tabs[i].Preview})
				}
			}
		}
	}

	// Row 2: baseline ─┘  └─, horizontal line with gap for active tab
	for x := 0; x < w; x++ {
		surface.SetCell(x, 2, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	if activeIdx >= 0 {
		s := spans[activeIdx]
		sx := s.start - t.ScrollOffset + innerLeft
		ex := s.end - t.ScrollOffset + innerLeft
		if sx >= innerLeft && sx < innerRight {
			surface.SetCell(sx, 2, term.Cell{Ch: b.BottomRight, Style: bs})
		}
		for x := sx + 1; x < ex-1; x++ {
			if x >= innerLeft && x < innerRight {
				surface.SetCell(x, 2, term.Cell{Ch: ' '})
			}
		}
		if ex-1 > sx && ex-1 >= innerLeft && ex-1 < innerRight {
			surface.SetCell(ex-1, 2, term.Cell{Ch: b.BottomLeft, Style: bs})
		}
	}

	// Arrow zones: " ◀ " on left, " ▶ " on right
	if t.hasOverflowLeft {
		surface.SetCell(1, 1, term.Cell{Ch: '◀', Style: term.StyleMuted})
	}
	if t.hasOverflowRight {
		surface.SetCell(innerRight+1, 1, term.Cell{Ch: '▶', Style: term.StyleMuted})
	}

	if t.MoreButton != nil && w >= 5 {
		r := t.GetRect()
		t.MoreButton.SetRect(Rect{X: r.X + w - 4, Y: r.Y + 1, W: 3, H: 1})
		moreSurface := surface.Sub(Rect{X: w - 4, Y: 1, W: 3, H: 1})
		t.MoreButton.Render(moreSurface)
	}
}

func (t *TabBarWidget) HandleEvent(ev tcell.Event) EventResult {
	mev, ok := ev.(*tcell.EventMouse)
	if !ok {
		return EventIgnored
	}
	r := t.GetRect()
	mx, my := mev.Position()
	btn := mev.Buttons()

	slog.Debug("tabBar", "mx", mx, "my", my, "btn", btn, "rect", r, "hasMore", t.MoreButton != nil)

	if t.MoreButton != nil {
		if t.MoreButton.HandleEvent(ev) == EventConsumed {
			return EventConsumed
		}
	}

	if my < r.Y || my >= r.Y+r.H || mx < r.X || mx >= r.X+r.W {
		slog.Debug("tabBar", "action", "outOfBounds")
		return EventIgnored
	}

	// Mouse wheel on tab bar switches tabs
	if btn&tcell.WheelUp != 0 && t.OnPrevTab != nil {
		t.OnPrevTab()
		return EventConsumed
	}
	if btn&tcell.WheelDown != 0 && t.OnNextTab != nil {
		t.OnNextTab()
		return EventConsumed
	}

	// Reuse Render's gutter width so click hit-tests line up with the screen.
	arrowW := t.renderArrowW

	if btn&tcell.Button2 != 0 && t.OnTabRightClick != nil {
		localX := mx - r.X - arrowW + t.ScrollOffset
		for i, s := range t.tabSpans {
			if localX >= s.start && localX < s.end {
				t.OnTabRightClick(i, mx, my)
				return EventConsumed
			}
		}
	}

	if btn&tcell.ButtonMiddle != 0 && t.OnTabClose != nil {
		localX := mx - r.X - arrowW + t.ScrollOffset
		for i, s := range t.tabSpans {
			if localX >= s.start && localX < s.end {
				t.OnTabClose(i)
				return EventConsumed
			}
		}
	}

	// Mouse release: only close if released at the exact same screen position as mouse-down
	if btn == tcell.ButtonNone {
		t.wasPressed = false
		if t.closeDownX >= 0 && mx == t.closeDownX && my == t.closeDownY {
			t.closeDownX = -1
			localX := mx - r.X - arrowW + t.ScrollOffset
			for i, s := range t.tabSpans {
				if s.active && localX == s.end-3 && t.OnTabClose != nil {
					t.OnTabClose(i)
					return EventConsumed
				}
			}
		}
		t.closeDownX = -1
		return EventIgnored
	}

	if btn&tcell.Button1 == 0 {
		return EventIgnored
	}

	freshClick := !t.wasPressed
	t.wasPressed = true
	if !freshClick {
		return EventConsumed
	}

	// Clicks in the reserved ◀/▶ arrow columns are consumed here so they never
	// fall through to the empty-space click handler and spawn a tab (the
	// "jumping to the other side" bug when clicking a hidden chevron).
	// Scroll only when there is something hidden in that direction; the overflow
	// flag is set only when the active tab isn't already at that end, so it can't
	// wrap.
	if arrowW > 0 && mx >= r.X && mx < r.X+arrowW {
		if t.hasOverflowLeft && t.OnPrevTab != nil {
			t.OnPrevTab()
		}
		return EventConsumed
	}
	rightZoneStart := r.X + t.renderInnerRight
	if arrowW > 0 && mx >= rightZoneStart && mx < rightZoneStart+arrowW {
		if t.hasOverflowRight && t.OnNextTab != nil {
			t.OnNextTab()
		}
		return EventConsumed
	}

	localX := mx - r.X - arrowW + t.ScrollOffset
	for i, s := range t.tabSpans {
		if localX >= s.start && localX < s.end {
			if s.active && t.OnTabClose != nil {
				closeX := s.end - 3
				if localX == closeX {
					t.closeDownX = mx
					t.closeDownY = my
					return EventConsumed
				}
			}
			now := time.Now().UnixMilli()
			if i == t.lastTabClick && now-t.lastTabClickTime < DoubleClickMs {
				t.lastTabClick = -1
				t.lastTabClickTime = 0
				if t.OnTabDoubleClick != nil {
					t.OnTabDoubleClick(i)
				}
				return EventConsumed
			}
			t.lastTabClick = i
			t.lastTabClickTime = now
			if t.OnTabClick != nil {
				t.OnTabClick(i)
			}
			return EventConsumed
		}
	}

	if t.OnEmptySpaceClick != nil {
		t.OnEmptySpaceClick()
	}
	return EventConsumed
}
