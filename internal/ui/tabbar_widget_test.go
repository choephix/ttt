package ui

import (
	"fmt"
	"testing"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v3"
)

func TestTabBarRender(t *testing.T) {
	tb := NewTabBarWidget()
	tb.SetTabs([]Tab{
		{Name: "main.go", Active: true},
		{Name: "buf.go", Dirty: true},
	})
	tb.SetRect(Rect{X: 0, Y: 0, W: 30, H: 3})

	grid := makeGrid(30, 3)
	surface := NewRenderSurface(grid, Rect{X: 0, Y: 0, W: 30, H: 3})
	tb.Render(surface)

	// Row 0: top border of active tab
	if grid[0][0].Ch != '┌' {
		t.Fatalf("expected TopLeft at row 0 col 0, got '%c'", grid[0][0].Ch)
	}
	// Row 1: active tab label with │ sides
	if grid[1][0].Ch != '│' {
		t.Fatalf("expected Vertical at row 1 col 0, got '%c'", grid[1][0].Ch)
	}
	if grid[1][2].Ch != 'm' {
		t.Fatalf("expected 'm' at row 1 col 2, got '%c'", grid[1][2].Ch)
	}
	if grid[1][2].Style != term.StyleActiveTab {
		t.Fatal("active tab should have StyleActiveTab")
	}
	// Row 2: baseline with gap
	if grid[2][0].Ch != '┘' {
		t.Fatalf("expected BottomRight at row 2 col 0, got '%c'", grid[2][0].Ch)
	}
}

func TestTabBarPreviewLabelIsItalic(t *testing.T) {
	tb := NewTabBarWidget()
	tb.SetTabs([]Tab{{Name: "preview.go", Active: true, Preview: true}})
	tb.SetRect(Rect{X: 0, Y: 0, W: 30, H: 3})
	grid := makeGrid(30, 3)
	tb.Render(NewRenderSurface(grid, Rect{X: 0, Y: 0, W: 30, H: 3}))

	if !grid[1][2].Italic {
		t.Fatal("preview tab label should be italic")
	}
}

func TestTabBarDoubleClickTargetsTab(t *testing.T) {
	tb := NewTabBarWidget()
	tb.SetTabs([]Tab{{Name: "preview.go", Active: true, Preview: true}})
	tb.SetRect(Rect{X: 0, Y: 0, W: 30, H: 3})
	tb.Render(NewRenderSurface(makeGrid(30, 3), Rect{X: 0, Y: 0, W: 30, H: 3}))

	clicked, doubleClicked := -1, -1
	tb.OnTabClick = func(index int) { clicked = index }
	tb.OnTabDoubleClick = func(index int) { doubleClicked = index }
	span := tb.tabSpans[0]
	x := span.start + (span.end-span.start)/2
	for range 2 {
		tb.HandleEvent(tcell.NewEventMouse(x, 1, tcell.Button1, 0))
		tb.HandleEvent(tcell.NewEventMouse(x, 1, tcell.ButtonNone, 0))
	}

	if clicked != 0 {
		t.Fatalf("first click targeted tab %d, want 0", clicked)
	}
	if doubleClicked != 0 {
		t.Fatalf("double-click targeted tab %d, want 0", doubleClicked)
	}
}

func TestTabBarEmptySpaceClick(t *testing.T) {
	tb := NewTabBarWidget()
	tb.SetTabs([]Tab{{Name: "main.go", Active: true}})
	tb.SetRect(Rect{X: 0, Y: 0, W: 30, H: 3})
	tb.Render(NewRenderSurface(makeGrid(30, 3), Rect{X: 0, Y: 0, W: 30, H: 3}))

	clicks := 0
	tb.OnEmptySpaceClick = func() { clicks++ }
	x := tb.tabSpans[0].end + 1
	tb.HandleEvent(tcell.NewEventMouse(x, 1, tcell.Button1, 0))
	tb.HandleEvent(tcell.NewEventMouse(x, 1, tcell.ButtonNone, 0))

	if clicks != 1 {
		t.Fatalf("empty-space click fired %d times, want 1", clicks)
	}
}

func TestTabBarMiddleClickClosesTargetWithoutActivating(t *testing.T) {
	tb := NewTabBarWidget()
	tb.SetTabs([]Tab{
		{Name: "first.go", Active: true},
		{Name: "second.go"},
	})
	tb.SetRect(Rect{X: 0, Y: 0, W: 40, H: 3})
	tb.Render(NewRenderSurface(makeGrid(40, 3), Rect{X: 0, Y: 0, W: 40, H: 3}))

	closed, activated := -1, -1
	tb.OnTabClose = func(index int) { closed = index }
	tb.OnTabClick = func(index int) { activated = index }
	second := tb.tabSpans[1]
	x := second.start + (second.end-second.start)/2
	tb.HandleEvent(tcell.NewEventMouse(x, 1, tcell.ButtonMiddle, 0))

	if closed != 1 {
		t.Fatalf("middle click closed tab %d, want 1", closed)
	}
	if activated != -1 {
		t.Fatalf("middle click activated tab %d before closing", activated)
	}
}

func TestTabBarOverflowArrows(t *testing.T) {
	tb := NewTabBarWidget()
	tb.SetTabs([]Tab{
		{Name: "main.go", Active: true, Closable: true},
		{Name: "buffer.go"},
		{Name: "cursor.go"},
		{Name: "highlight.go"},
		{Name: "undo.go"},
		{Name: "selection.go"},
	})
	tb.SetRect(Rect{X: 0, Y: 0, W: 30, H: 3})

	grid := makeGrid(30, 3)
	surface := NewRenderSurface(grid, Rect{X: 0, Y: 0, W: 30, H: 3})
	tb.Render(surface)

	if tb.hasOverflowLeft {
		t.Fatal("should not have left overflow when active tab is first")
	}
	if !tb.hasOverflowRight {
		t.Fatal("should have right overflow with narrow width")
	}
	// Right arrow is at innerRight+1 = (30-3)+1 = 28
	if grid[1][28].Ch != '▶' {
		t.Fatalf("expected right arrow at row 1 col 28, got '%c'", grid[1][28].Ch)
	}
}

func TestTabBarOverflowScrollLeft(t *testing.T) {
	tb := NewTabBarWidget()
	tb.SetTabs([]Tab{
		{Name: "main.go"},
		{Name: "buffer.go"},
		{Name: "cursor.go"},
		{Name: "highlight.go"},
		{Name: "undo.go", Active: true, Closable: true},
	})
	tb.SetRect(Rect{X: 0, Y: 0, W: 30, H: 3})

	grid := makeGrid(30, 3)
	surface := NewRenderSurface(grid, Rect{X: 0, Y: 0, W: 30, H: 3})
	tb.Render(surface)

	if !tb.hasOverflowLeft {
		t.Fatal("should have left overflow when active tab is scrolled right")
	}
	// Left arrow " ◀ " — chevron is at col 1
	if grid[1][1].Ch != '◀' {
		t.Fatalf("expected left arrow at row 1 col 1, got '%c'", grid[1][1].Ch)
	}
}

// renderInMoreButtonWindow sizes the bar to exactly the total tab width with a
// MoreButton present, so tabs overflow the inner zone but not the full width —
// the #354 window. Returns the rendered grid and the bar width.
func renderInMoreButtonWindow(t *testing.T) (*TabBarWidget, [][]term.Cell, int) {
	t.Helper()
	tb := NewTabBarWidget()
	tb.MoreButton = NewMoreButtonWidget()
	tb.SetTabs([]Tab{
		{Name: "main.go", Active: true, Closable: true},
		{Name: "buffer.go"},
		{Name: "cursor.go"},
	})

	tb.SetRect(Rect{X: 0, Y: 0, W: 200, H: 3})
	probe := makeGrid(200, 3)
	tb.Render(NewRenderSurface(probe, Rect{X: 0, Y: 0, W: 200, H: 3}))
	w := tb.totalTabWidth

	tb.SetRect(Rect{X: 0, Y: 0, W: w, H: 3})
	grid := makeGrid(w, 3)
	tb.Render(NewRenderSurface(grid, Rect{X: 0, Y: 0, W: w, H: 3}))
	return tb, grid, w
}

// TestTabBarCloseHitInMoreButtonWindow guards issue #354: clicking the active
// tab's rendered × in the MoreButton window must close it (was a dead click).
func TestTabBarCloseHitInMoreButtonWindow(t *testing.T) {
	tb, grid, w := renderInMoreButtonWindow(t)

	if tb.renderArrowW == 0 {
		t.Fatal("expected the arrow gutter to be reserved once the strip overflows the inner zone")
	}

	// Find the rendered close × of the active tab (row 1, StyleActiveTab).
	closeX := -1
	for x := 0; x < w; x++ {
		if grid[1][x].Ch == 'x' && grid[1][x].Style == term.StyleActiveTab {
			closeX = x
		}
	}
	if closeX < 0 {
		t.Fatal("test setup: could not find rendered close × for active tab")
	}

	closed := -1
	tb.OnTabClose = func(i int) { closed = i }

	// A real click is mouse-down then mouse-up at the same cell.
	tb.HandleEvent(tcell.NewEventMouse(closeX, 1, tcell.Button1, 0))
	tb.HandleEvent(tcell.NewEventMouse(closeX, 1, tcell.ButtonNone, 0))

	if closed != 0 {
		t.Fatalf("clicking the visible close × should close tab 0, got closed=%d", closed)
	}
}

// TestTabBarChevronNotDrawnOverTab: when a chevron shows, its gutter must be
// reserved so tabs never render on top of it.
func TestTabBarChevronNotDrawnOverTab(t *testing.T) {
	tb, grid, _ := renderInMoreButtonWindow(t)

	if (tb.hasOverflowLeft || tb.hasOverflowRight) && tb.renderArrowW == 0 {
		t.Fatal("chevron shown without a reserved gutter — tabs will overlap it")
	}
	if tb.hasOverflowLeft && grid[1][1].Ch != '◀' {
		t.Fatalf("left chevron cell overwritten by a tab, got '%c'", grid[1][1].Ch)
	}
}

// TestTabBarNoOverScrollAfterClose: closing tabs must not leave the strip scrolled
// past the last tab (only the final tab visible with empty space to its right).
func TestTabBarNoOverScrollAfterClose(t *testing.T) {
	tb := NewTabBarWidget()
	tb.MoreButton = NewMoreButtonWidget()

	many := make([]Tab, 20)
	for i := range many {
		many[i] = Tab{Name: "untitled-" + string(rune('a'+i)) + ".go"}
	}
	many[19].Active = true
	tb.SetTabs(many)
	tb.SetRect(Rect{X: 0, Y: 0, W: 40, H: 3})
	tb.Render(NewRenderSurface(makeGrid(40, 3), Rect{X: 0, Y: 0, W: 40, H: 3}))
	if tb.ScrollOffset == 0 {
		t.Fatal("test setup: expected a non-zero scroll offset with 20 tabs at width 40")
	}

	// Close down to three tabs, first active (like closing everything to the right).
	tb.SetTabs([]Tab{
		{Name: "untitled-a.go", Active: true},
		{Name: "untitled-b.go"},
		{Name: "untitled-c.go"},
	})
	tb.SetRect(Rect{X: 0, Y: 0, W: 40, H: 3})
	tb.Render(NewRenderSurface(makeGrid(40, 3), Rect{X: 0, Y: 0, W: 40, H: 3}))

	if tb.ScrollOffset != 0 {
		t.Fatalf("offset should snap back to 0 once the tabs fit, got %d", tb.ScrollOffset)
	}
	if tb.hasOverflowLeft {
		t.Fatal("no left overflow expected once the tabs fit")
	}
}

// TestTabBarGutterClickDoesNotSpawnTab: at the first tab the ◀ is hidden but its
// gutter is still reserved. Clicking that empty gutter must be a no-op — it
// must NOT fall through to the empty-space click handler and spawn a tab (which
// looked like "jumping to the other side").
func TestTabBarGutterClickDoesNotSpawnTab(t *testing.T) {
	tb := NewTabBarWidget()
	tb.MoreButton = NewMoreButtonWidget()

	const n = 8
	tabs := make([]Tab, n)
	for i := range tabs {
		tabs[i] = Tab{Name: fmt.Sprintf("untitled-%d.go", i+1), Active: i == 0, Closable: i == 0}
	}
	tb.SetTabs(tabs)

	emptySpaceClicks, prevTabs := 0, 0
	tb.OnEmptySpaceClick = func() { emptySpaceClicks++ }
	tb.OnPrevTab = func() { prevTabs++ }

	r := Rect{X: 0, Y: 0, W: 40, H: 3}
	tb.SetRect(r)
	tb.Render(NewRenderSurface(makeGrid(40, 3), r))
	if tb.hasOverflowLeft || tb.renderArrowW == 0 {
		t.Fatalf("test setup: want hidden ◀ with reserved gutter, got ovL=%v arrowW=%d",
			tb.hasOverflowLeft, tb.renderArrowW)
	}

	// Click the empty left gutter where ◀ would be.
	tb.HandleEvent(tcell.NewEventMouse(r.X+1, 1, tcell.Button1, 0))
	tb.HandleEvent(tcell.NewEventMouse(r.X+1, 1, tcell.ButtonNone, 0))

	if emptySpaceClicks != 0 {
		t.Fatalf("gutter click spawned a tab (OnEmptySpaceClick fired %d times)", emptySpaceClicks)
	}
	if prevTabs != 0 {
		t.Fatalf("gutter click on a hidden ◀ scrolled (OnPrevTab fired %d times)", prevTabs)
	}
}

func TestTabBarNoOverflowWhenFits(t *testing.T) {
	tb := NewTabBarWidget()
	tb.SetTabs([]Tab{
		{Name: "a.go", Active: true},
		{Name: "b.go"},
	})
	tb.SetRect(Rect{X: 0, Y: 0, W: 40, H: 3})

	grid := makeGrid(40, 3)
	surface := NewRenderSurface(grid, Rect{X: 0, Y: 0, W: 40, H: 3})
	tb.Render(surface)

	if tb.hasOverflowLeft || tb.hasOverflowRight {
		t.Fatal("should not have overflow when all tabs fit")
	}
}
