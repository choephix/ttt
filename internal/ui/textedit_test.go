package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func keyRune(r rune) *tcell.EventKey {
	return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
}

func keySpecial(k tcell.Key) *tcell.EventKey {
	return tcell.NewEventKey(k, 0, tcell.ModNone)
}

func TestHandleTextEditInsert(t *testing.T) {
	r := HandleTextEdit(keyRune('a'), "hello", 5)
	if r.Text != "helloa" || r.CurPos != 6 || !r.Changed {
		t.Fatalf("insert at end: got %q pos=%d changed=%v", r.Text, r.CurPos, r.Changed)
	}

	r = HandleTextEdit(keyRune('x'), "hello", 0)
	if r.Text != "xhello" || r.CurPos != 1 {
		t.Fatalf("insert at start: got %q pos=%d", r.Text, r.CurPos)
	}

	r = HandleTextEdit(keyRune('m'), "hello", 2)
	if r.Text != "hemllo" || r.CurPos != 3 {
		t.Fatalf("insert mid: got %q pos=%d", r.Text, r.CurPos)
	}
}

func TestHandleTextEditBackspace(t *testing.T) {
	r := HandleTextEdit(keySpecial(tcell.KeyBackspace2), "hello", 3)
	if r.Text != "helo" || r.CurPos != 2 || !r.Changed {
		t.Fatalf("backspace: got %q pos=%d", r.Text, r.CurPos)
	}

	r = HandleTextEdit(keySpecial(tcell.KeyBackspace2), "hello", 0)
	if r.Text != "hello" || r.CurPos != 0 {
		t.Fatalf("backspace at 0: got %q pos=%d", r.Text, r.CurPos)
	}
}

func TestHandleTextEditDelete(t *testing.T) {
	r := HandleTextEdit(keySpecial(tcell.KeyDelete), "hello", 2)
	if r.Text != "helo" || r.CurPos != 2 || !r.Changed {
		t.Fatalf("delete: got %q pos=%d", r.Text, r.CurPos)
	}

	r = HandleTextEdit(keySpecial(tcell.KeyDelete), "hello", 5)
	if r.Text != "hello" || r.CurPos != 5 {
		t.Fatalf("delete at end: got %q pos=%d", r.Text, r.CurPos)
	}
}

func TestHandleTextEditCursorMovement(t *testing.T) {
	r := HandleTextEdit(keySpecial(tcell.KeyLeft), "hello", 3)
	if r.CurPos != 2 {
		t.Fatalf("left: pos=%d", r.CurPos)
	}

	r = HandleTextEdit(keySpecial(tcell.KeyRight), "hello", 3)
	if r.CurPos != 4 {
		t.Fatalf("right: pos=%d", r.CurPos)
	}

	r = HandleTextEdit(keySpecial(tcell.KeyHome), "hello", 3)
	if r.CurPos != 0 {
		t.Fatalf("home: pos=%d", r.CurPos)
	}

	r = HandleTextEdit(keySpecial(tcell.KeyEnd), "hello", 3)
	if r.CurPos != 5 {
		t.Fatalf("end: pos=%d", r.CurPos)
	}
}

func TestHandleTextEditBoundary(t *testing.T) {
	r := HandleTextEdit(keySpecial(tcell.KeyLeft), "hello", 0)
	if r.CurPos != 0 {
		t.Fatalf("left at 0: pos=%d", r.CurPos)
	}

	r = HandleTextEdit(keySpecial(tcell.KeyRight), "hello", 5)
	if r.CurPos != 5 {
		t.Fatalf("right at end: pos=%d", r.CurPos)
	}
}
