package undo

import (
	"macro/internal/core/buffer"
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
