package ui

import (
	"github.com/eugenioenko/ttt/internal/term"
	"testing"
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

func TestTabBarNotFocusable(t *testing.T) {
	tb := NewTabBarWidget()
	if tb.Focusable() {
		t.Fatal("tab bar should not be focusable")
	}
}
