package ui

import (
	"testing"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
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

// TestTabBarCloseHitInMoreButtonWindow guards issue #354: when the tabs overflow
// the inner zone (the ⋮ MoreButton reserves 4 cols) but NOT the full widget width,
// Render reserves no arrow gutter (arrowW 0). HandleEvent used to recompute arrowW
// from the overflow flags and land on 3, shifting every close-X hit test by 3 cells
// so clicking the active tab's × did nothing. The click must reuse Render's arrowW.
func TestTabBarCloseHitInMoreButtonWindow(t *testing.T) {
	tb := NewTabBarWidget()
	tb.MoreButton = NewMoreButtonWidget()
	tb.SetTabs([]Tab{
		{Name: "main.go", Active: true, Closable: true},
		{Name: "buffer.go"},
		{Name: "cursor.go"},
	})

	// Measure total tab width, then size the bar to exactly that width. With the
	// MoreButton stealing 4 cols this puts us in the divergent window: tabs fit the
	// full width (arrowW 0) but overflow the inner zone (hasOverflowRight true).
	tb.SetRect(Rect{X: 0, Y: 0, W: 200, H: 3})
	probe := makeGrid(200, 3)
	tb.Render(NewRenderSurface(probe, Rect{X: 0, Y: 0, W: 200, H: 3}))
	w := tb.totalTabWidth

	tb.SetRect(Rect{X: 0, Y: 0, W: w, H: 3})
	grid := makeGrid(w, 3)
	tb.Render(NewRenderSurface(grid, Rect{X: 0, Y: 0, W: w, H: 3}))

	if tb.renderArrowW != 0 {
		t.Fatalf("test setup: expected no arrow gutter, got renderArrowW=%d", tb.renderArrowW)
	}
	if !tb.hasOverflowRight {
		t.Fatal("test setup: expected right overflow inside the MoreButton window")
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
