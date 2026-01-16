package buffer

import "testing"

func TestInsertRune(t *testing.T) {
	b := &Buffer{Lines: []string{"hello"}}
	b.InsertRune(0, 5, '!')
	if b.Lines[0] != "hello!" {
		t.Errorf("expected 'hello!', got '%s'", b.Lines[0])
	}
}

func TestDeleteRune(t *testing.T) {
	b := &Buffer{Lines: []string{"hello!"}}
	b.DeleteRune(0, 5)
	if b.Lines[0] != "hello" {
		t.Errorf("expected 'hello', got '%s'", b.Lines[0])
	}
}

func TestInsertLine(t *testing.T) {
	b := &Buffer{Lines: []string{"a", "c"}}
	b.InsertLine(1, "b")
	if len(b.Lines) != 3 || b.Lines[1] != "b" {
		t.Errorf("expected line 'b' at index 1, got '%v'", b.Lines)
	}
}

func TestDeleteLine(t *testing.T) {
	b := &Buffer{Lines: []string{"a", "b", "c"}}
	b.DeleteLine(1)
	if len(b.Lines) != 2 || b.Lines[1] != "c" {
		t.Errorf("expected lines [a c], got '%v'", b.Lines)
	}
}
