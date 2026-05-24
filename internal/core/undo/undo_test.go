package undo

import (
	"github.com/eugenioenko/ttt/internal/core/buffer"
	"testing"
)

func TestInsertRuneCommand(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{"abc"}}
	cmd := &InsertRuneCommand{Line: 0, Col: 1, Rune: 'X'}
	cmd.Apply(b)
	if b.Lines[0] != "aXbc" {
		t.Errorf("expected 'aXbc', got '%s'", b.Lines[0])
	}
	cmd.Undo(b)
	if b.Lines[0] != "abc" {
		t.Errorf("expected 'abc', got '%s'", b.Lines[0])
	}
}

func TestDeleteRangeCommand(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{"abcdef"}}
	cmd := &DeleteRangeCommand{Line: 0, Start: 2, End: 4}
	cmd.Apply(b)
	if b.Lines[0] != "abef" {
		t.Errorf("expected 'abef', got '%s'", b.Lines[0])
	}
	cmd.Undo(b)
	if b.Lines[0] != "abcdef" {
		t.Errorf("expected 'abcdef', got '%s'", b.Lines[0])
	}
}

func TestInsertLineCommand(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{"a", "c"}}
	cmd := &InsertLineCommand{Idx: 1, Text: "b"}
	cmd.Apply(b)
	if len(b.Lines) != 3 || b.Lines[1] != "b" {
		t.Errorf("expected line 'b' at index 1, got '%v'", b.Lines)
	}
	cmd.Undo(b)
	if len(b.Lines) != 2 || b.Lines[1] != "c" {
		t.Errorf("expected lines [a c], got '%v'", b.Lines)
	}
}

func TestPasteCommandSingleLine(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{"hello world"}}
	cmd := &PasteCommand{Line: 0, Col: 5, Text: " beautiful", Suffix: " world"}
	cmd.Apply(b)
	if b.Lines[0] != "hello beautiful world" {
		t.Errorf("expected 'hello beautiful world', got '%s'", b.Lines[0])
	}
	cmd.Undo(b)
	if b.Lines[0] != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", b.Lines[0])
	}
}

func TestPasteCommandMultiLine(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{"ab"}}
	cmd := &PasteCommand{Line: 0, Col: 1, Text: "X\nY\nZ", Suffix: "b"}
	cmd.Apply(b)
	if len(b.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(b.Lines), b.Lines)
	}
	if b.Lines[0] != "aX" || b.Lines[1] != "Y" || b.Lines[2] != "Zb" {
		t.Errorf("unexpected lines: %v", b.Lines)
	}
	cmd.Undo(b)
	if len(b.Lines) != 1 || b.Lines[0] != "ab" {
		t.Errorf("expected ['ab'], got %v", b.Lines)
	}
}

func TestDeleteSelectionCommand(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{"hello", "world", "foo"}}
	cmd := &DeleteSelectionCommand{StartLine: 0, StartCol: 3, EndLine: 1, EndCol: 2}
	cmd.Apply(b)
	if len(b.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(b.Lines), b.Lines)
	}
	if b.Lines[0] != "helrld" {
		t.Errorf("expected 'helrld', got '%s'", b.Lines[0])
	}
	cmd.Undo(b)
	if len(b.Lines) != 3 || b.Lines[0] != "hello" || b.Lines[1] != "world" {
		t.Errorf("undo failed: %v", b.Lines)
	}
}

func TestUndoStack(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{"abc"}}
	s := &UndoStack{}
	cmd := &InsertRuneCommand{Line: 0, Col: 3, Rune: '!'}
	s.Push(cmd)
	cmd.Apply(b)
	s.Undo(b)
	if b.Lines[0] != "abc" {
		t.Errorf("expected 'abc', got '%s'", b.Lines[0])
	}
	s.Redo(b)
	if b.Lines[0] != "abc!" {
		t.Errorf("expected 'abc!', got '%s'", b.Lines[0])
	}
}
