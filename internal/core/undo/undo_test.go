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

func TestBatchCommand(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{"abc"}}
	batch := &BatchCommand{
		Commands: []EditCommand{
			&InsertRuneCommand{Line: 0, Col: 0, Rune: 'X'},
			&InsertRuneCommand{Line: 0, Col: 1, Rune: 'Y'},
			&InsertRuneCommand{Line: 0, Col: 2, Rune: 'Z'},
		},
	}
	batch.Apply(b)
	if b.Lines[0] != "XYZabc" {
		t.Errorf("expected 'XYZabc', got '%s'", b.Lines[0])
	}
	batch.Undo(b)
	if b.Lines[0] != "abc" {
		t.Errorf("expected 'abc' after undo, got '%s'", b.Lines[0])
	}
}

func TestBatchCommandUndoStack(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{"hello"}}
	s := &UndoStack{}
	batch := &BatchCommand{
		Commands: []EditCommand{
			&InsertRuneCommand{Line: 0, Col: 5, Rune: '!'},
			&InsertRuneCommand{Line: 0, Col: 6, Rune: '!'},
		},
	}
	batch.Apply(b)
	s.Push(batch)
	if b.Lines[0] != "hello!!" {
		t.Errorf("expected 'hello!!', got '%s'", b.Lines[0])
	}
	s.Undo(b)
	if b.Lines[0] != "hello" {
		t.Errorf("expected 'hello' after single undo, got '%s'", b.Lines[0])
	}
	s.Redo(b)
	if b.Lines[0] != "hello!!" {
		t.Errorf("expected 'hello!!' after redo, got '%s'", b.Lines[0])
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

func TestUndoGroupsConsecutiveInserts(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{""}}
	s := &UndoStack{}
	for i, r := range []rune("hello") {
		cmd := &InsertRuneCommand{Line: 0, Col: i, Rune: r}
		cmd.Apply(b)
		s.Push(cmd)
	}
	if b.Lines[0] != "hello" {
		t.Fatalf("expected 'hello', got '%s'", b.Lines[0])
	}
	s.Undo(b)
	if b.Lines[0] != "" {
		t.Errorf("expected '' after single undo, got '%s'", b.Lines[0])
	}
	s.Redo(b)
	if b.Lines[0] != "hello" {
		t.Errorf("expected 'hello' after redo, got '%s'", b.Lines[0])
	}
}

func TestUndoBreaksGroupOnSpace(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{""}}
	s := &UndoStack{}
	for i, r := range []rune("hi there") {
		cmd := &InsertRuneCommand{Line: 0, Col: i, Rune: r}
		cmd.Apply(b)
		s.Push(cmd)
	}
	if b.Lines[0] != "hi there" {
		t.Fatalf("expected 'hi there', got '%s'", b.Lines[0])
	}
	s.Undo(b)
	if b.Lines[0] != "hi " {
		t.Errorf("expected 'hi ' after first undo, got '%s'", b.Lines[0])
	}
	s.Undo(b)
	if b.Lines[0] != "hi" {
		t.Errorf("expected 'hi' after second undo, got '%s'", b.Lines[0])
	}
	s.Undo(b)
	if b.Lines[0] != "" {
		t.Errorf("expected '' after third undo, got '%s'", b.Lines[0])
	}
}

func TestUndoBreaksGroupOnBreakGroup(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{""}}
	s := &UndoStack{}
	for i, r := range []rune("ab") {
		cmd := &InsertRuneCommand{Line: 0, Col: i, Rune: r}
		cmd.Apply(b)
		s.Push(cmd)
	}
	s.BreakGroup()
	for i, r := range []rune("cd") {
		cmd := &InsertRuneCommand{Line: 0, Col: 2 + i, Rune: r}
		cmd.Apply(b)
		s.Push(cmd)
	}
	if b.Lines[0] != "abcd" {
		t.Fatalf("expected 'abcd', got '%s'", b.Lines[0])
	}
	s.Undo(b)
	if b.Lines[0] != "ab" {
		t.Errorf("expected 'ab' after first undo, got '%s'", b.Lines[0])
	}
	s.Undo(b)
	if b.Lines[0] != "" {
		t.Errorf("expected '' after second undo, got '%s'", b.Lines[0])
	}
}

func TestUndoGroupsConsecutiveDeletes(t *testing.T) {
	b := &buffer.Buffer{Lines: []string{"hello"}}
	s := &UndoStack{}
	for i := 4; i >= 0; i-- {
		cmd := &DeleteRuneCommand{Line: 0, Col: i}
		cmd.Apply(b)
		s.Push(cmd)
	}
	if b.Lines[0] != "" {
		t.Fatalf("expected '', got '%s'", b.Lines[0])
	}
	s.Undo(b)
	if b.Lines[0] != "hello" {
		t.Errorf("expected 'hello' after single undo, got '%s'", b.Lines[0])
	}
}
