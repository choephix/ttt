package ui

import (
	"testing"

	"github.com/eugenioenko/ttt/internal/core/buffer"
	"github.com/eugenioenko/ttt/internal/core/cursor"
	"github.com/eugenioenko/ttt/internal/view"

	"github.com/gdamore/tcell/v2"
)

func newEditorWithLines(lines ...string) *EditorPaneWidget {
	buf := &buffer.Buffer{Lines: lines}
	cur := &cursor.Cursor{Line: 0, Col: 0}
	vp := &view.Viewport{TopLine: 0, LeftCol: 0, Width: 40, Height: 10}
	return NewEditorPaneWidget(buf, cur, vp)
}

func TestAutoIndentEnterAfterOpenBrace(t *testing.T) {
	e := newEditorWithLines("{")
	e.Cursor.Col = 1

	e.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))

	if e.Buf.Lines[1] != "    " {
		t.Fatalf("expected new line indented 4 spaces, got %q", e.Buf.Lines[1])
	}
	if e.Cursor.Line != 1 || e.Cursor.Col != 4 {
		t.Fatalf("expected cursor at (1,4), got (%d,%d)", e.Cursor.Line, e.Cursor.Col)
	}
}

func TestAutoIndentEnterBetweenBracesSplitsClosing(t *testing.T) {
	e := newEditorWithLines("{}")
	e.Cursor.Col = 1

	e.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))

	if e.Buf.Lines[0] != "{" || e.Buf.Lines[1] != "    " || e.Buf.Lines[2] != "}" {
		t.Fatalf("expected {, 4-space line, }, got %q", e.Buf.Lines)
	}
	if e.Cursor.Line != 1 || e.Cursor.Col != 4 {
		t.Fatalf("expected cursor at (1,4), got (%d,%d)", e.Cursor.Line, e.Cursor.Col)
	}
}

func TestAutoIndentDedentOnCloseBrace(t *testing.T) {
	e := newEditorWithLines("{", "    ")
	e.Cursor.Line = 1
	e.Cursor.Col = 4

	e.HandleEvent(tcell.NewEventKey(tcell.KeyRune, '}', 0))

	if e.Buf.Lines[1] != "}" {
		t.Fatalf("expected closing brace dedented to column 0, got %q", e.Buf.Lines[1])
	}
	if e.Cursor.Col != 1 {
		t.Fatalf("expected cursor at col 1, got %d", e.Cursor.Col)
	}
}

func TestAutoIndentDedentNestedRemovesOneLevel(t *testing.T) {
	e := newEditorWithLines("        ") // two levels of indent
	e.Cursor.Col = 8

	e.HandleEvent(tcell.NewEventKey(tcell.KeyRune, '}', 0))

	if e.Buf.Lines[0] != "    }" {
		t.Fatalf("expected one indent level removed leaving '    }', got %q", e.Buf.Lines[0])
	}
	if e.Cursor.Col != 5 {
		t.Fatalf("expected cursor at col 5, got %d", e.Cursor.Col)
	}
}

func TestAutoIndentDedentSkipsMidLine(t *testing.T) {
	e := newEditorWithLines("    foo")
	e.Cursor.Col = 7

	e.HandleEvent(tcell.NewEventKey(tcell.KeyRune, '}', 0))

	if e.Buf.Lines[0] != "    foo}" {
		t.Fatalf("expected no dedent for mid-line brace, got %q", e.Buf.Lines[0])
	}
}

func TestAutoIndentDisabledPlainNewline(t *testing.T) {
	e := newEditorWithLines("{")
	e.AutoIndent = false
	e.Cursor.Col = 1

	e.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))

	if e.Buf.Lines[1] != "" {
		t.Fatalf("expected empty new line when auto-indent off, got %q", e.Buf.Lines[1])
	}
	if e.Cursor.Line != 1 || e.Cursor.Col != 0 {
		t.Fatalf("expected cursor at (1,0), got (%d,%d)", e.Cursor.Line, e.Cursor.Col)
	}
}

func TestAutoIndentDisabledNoDedent(t *testing.T) {
	e := newEditorWithLines("    ")
	e.AutoIndent = false
	e.Cursor.Col = 4

	e.HandleEvent(tcell.NewEventKey(tcell.KeyRune, '}', 0))

	if e.Buf.Lines[0] != "    }" {
		t.Fatalf("expected brace appended without dedent, got %q", e.Buf.Lines[0])
	}
}

func TestAutoIndentEnabledByDefault(t *testing.T) {
	e := newEditorWithLines("")
	if !e.AutoIndent {
		t.Fatal("expected auto-indent to be enabled by default")
	}
}
