package ui

import (
	"macro/internal/core/buffer"
	"macro/internal/core/cursor"
	"macro/internal/core/selection"
	"macro/internal/term"
	"macro/internal/view"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func newTestEditor() *EditorPaneWidget {
	buf := &buffer.Buffer{Lines: []string{"Hello", "World", "Test"}}
	cur := &cursor.Cursor{Line: 0, Col: 0}
	vp := &view.Viewport{TopLine: 0, LeftCol: 0, Width: 20, Height: 10}
	return NewEditorPaneWidget(buf, cur, vp)
}

func TestEditorRender(t *testing.T) {
	e := newTestEditor()
	e.SetRect(Rect{X: 0, Y: 0, W: 20, H: 10})

	grid := makeGrid(20, 10)
	surface := NewRenderSurface(grid, Rect{X: 0, Y: 0, W: 20, H: 10})
	e.Render(surface)

	if grid[0][0].Ch != 'H' {
		t.Fatalf("expected 'H' at (0,0), got '%c'", grid[0][0].Ch)
	}
	if grid[1][0].Ch != 'W' {
		t.Fatalf("expected 'W' at (0,1), got '%c'", grid[1][0].Ch)
	}
	// Line past buffer should show '~'
	if grid[3][0].Ch != '~' {
		t.Fatalf("expected '~' at (0,3), got '%c'", grid[3][0].Ch)
	}
}

func TestEditorArrowKeys(t *testing.T) {
	e := newTestEditor()
	e.SetRect(Rect{X: 0, Y: 0, W: 20, H: 10})

	e.HandleEvent(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	if e.Cursor.Line != 1 {
		t.Fatalf("expected line 1, got %d", e.Cursor.Line)
	}

	e.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if e.Cursor.Col != 1 {
		t.Fatalf("expected col 1, got %d", e.Cursor.Col)
	}

	e.HandleEvent(tcell.NewEventKey(tcell.KeyUp, 0, 0))
	if e.Cursor.Line != 0 {
		t.Fatalf("expected line 0, got %d", e.Cursor.Line)
	}
}

func TestEditorTypeCharacter(t *testing.T) {
	e := newTestEditor()
	e.SetRect(Rect{X: 0, Y: 0, W: 20, H: 10})

	e.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'X', 0))
	if e.Buf.Lines[0] != "XHello" {
		t.Fatalf("expected 'XHello', got '%s'", e.Buf.Lines[0])
	}
	if e.Cursor.Col != 1 {
		t.Fatalf("expected col 1, got %d", e.Cursor.Col)
	}
}

func TestEditorCursorPosition(t *testing.T) {
	e := newTestEditor()
	e.SetRect(Rect{X: 5, Y: 3, W: 20, H: 10})
	e.Cursor.Line = 1
	e.Cursor.Col = 2

	grid := makeGrid(25, 13)
	surface := NewRenderSurface(grid, Rect{X: 5, Y: 3, W: 20, H: 10})
	e.Render(surface)

	if e.CursorX != 7 {
		t.Fatalf("expected CursorX 7, got %d", e.CursorX)
	}
	if e.CursorY != 4 {
		t.Fatalf("expected CursorY 4, got %d", e.CursorY)
	}
}

func TestEditorFocusable(t *testing.T) {
	e := newTestEditor()
	if !e.Focusable() {
		t.Fatal("editor should be focusable")
	}
}

func TestEditorHomeEnd(t *testing.T) {
	e := newTestEditor()
	e.Cursor.Col = 3

	e.HandleEvent(tcell.NewEventKey(tcell.KeyHome, 0, 0))
	if e.Cursor.Col != 0 {
		t.Fatalf("Home: expected col 0, got %d", e.Cursor.Col)
	}

	e.HandleEvent(tcell.NewEventKey(tcell.KeyEnd, 0, 0))
	if e.Cursor.Col != 5 {
		t.Fatalf("End: expected col 5, got %d", e.Cursor.Col)
	}
}

func TestEditorBackspace(t *testing.T) {
	e := newTestEditor()
	e.Cursor.Col = 2

	e.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace2, 0, 0))
	if e.Buf.Lines[0] != "Hllo" {
		t.Fatalf("expected 'Hllo', got '%s'", e.Buf.Lines[0])
	}
}

func TestEditorEnter(t *testing.T) {
	e := newTestEditor()
	e.Cursor.Col = 3

	e.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if e.Buf.Lines[0] != "Hel" {
		t.Fatalf("expected 'Hel', got '%s'", e.Buf.Lines[0])
	}
	if e.Buf.Lines[1] != "lo" {
		t.Fatalf("expected 'lo', got '%s'", e.Buf.Lines[1])
	}
	if e.Cursor.Line != 1 || e.Cursor.Col != 0 {
		t.Fatalf("expected cursor at (1,0), got (%d,%d)", e.Cursor.Line, e.Cursor.Col)
	}
}

func TestEditorSelectionHighlight(t *testing.T) {
	e := newTestEditor()
	e.SetRect(Rect{X: 0, Y: 0, W: 20, H: 10})
	e.Selection = &selection.Selection{}

	// Select "ell" in "Hello" via Shift+Right from col 1
	e.Cursor.Col = 1
	e.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModShift))
	e.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModShift))
	e.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModShift))

	if !e.Selection.Active {
		t.Fatal("selection should be active")
	}

	grid := makeGrid(20, 10)
	surface := NewRenderSurface(grid, Rect{X: 0, Y: 0, W: 20, H: 10})
	e.Render(surface)

	// Cols 1,2,3 should be StyleSelection (anchor=1, cursor=4)
	for col := 1; col <= 3; col++ {
		if grid[0][col].Style != term.StyleSelection {
			t.Errorf("col %d: expected StyleSelection, got %d", col, grid[0][col].Style)
		}
	}
	// Col 0 and col 4 should NOT be selected
	if grid[0][0].Style == term.StyleSelection {
		t.Error("col 0 should not be selected")
	}
	if grid[0][4].Style == term.StyleSelection {
		t.Error("col 4 should not be selected")
	}
}

// ensure term import is used
var _ = term.Cell{}
