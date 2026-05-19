package ui

import (
	"macro/internal/term"
	"testing"
)

func TestTabBarRender(t *testing.T) {
	tb := NewTabBarWidget()
	tb.SetTabs([]Tab{
		{Name: "main.go", Active: true},
		{Name: "buf.go", Dirty: true},
	})
	tb.SetRect(Rect{X: 0, Y: 0, W: 30, H: 1})

	grid := makeGrid(30, 1)
	surface := NewRenderSurface(grid, Rect{X: 0, Y: 0, W: 30, H: 1})
	tb.Render(surface)

	// First tab should start with space then 'm'
	if grid[0][0].Ch != ' ' {
		t.Fatalf("expected space at start, got '%c'", grid[0][0].Ch)
	}
	if grid[0][1].Ch != 'm' {
		t.Fatalf("expected 'm' at pos 1, got '%c'", grid[0][1].Ch)
	}
	// Active tab should have active style
	if grid[0][1].Style != term.StyleActiveTab {
		t.Fatal("active tab should have StyleActiveTab")
	}
}

func TestTabBarNotFocusable(t *testing.T) {
	tb := NewTabBarWidget()
	if tb.Focusable() {
		t.Fatal("tab bar should not be focusable")
	}
}
