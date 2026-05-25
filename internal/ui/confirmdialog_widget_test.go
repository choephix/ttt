package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestConfirmDialog2ButtonDefaults(t *testing.T) {
	d := NewConfirmDialogWidget("Delete?")
	if len(d.Buttons) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(d.Buttons))
	}
	if d.Selected != 1 {
		t.Fatalf("expected default selected=1 (No), got %d", d.Selected)
	}
}

func TestConfirmDialog3ButtonDefaults(t *testing.T) {
	d := NewConfirmDialogWidget3("Save?", "Discard", "Cancel", "Save")
	if len(d.Buttons) != 3 {
		t.Fatalf("expected 3 buttons, got %d", len(d.Buttons))
	}
	if d.Selected != 2 {
		t.Fatalf("expected default selected=2, got %d", d.Selected)
	}
}

func TestConfirmDialogKeyboardNav(t *testing.T) {
	d := NewConfirmDialogWidget3("msg", "A", "B", "C")

	d.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	if d.Selected != 1 {
		t.Fatalf("left from 2: expected 1, got %d", d.Selected)
	}

	d.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	if d.Selected != 2 {
		t.Fatalf("right from 1: expected 2, got %d", d.Selected)
	}

	d.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	if d.Selected != 0 {
		t.Fatalf("right wraps: expected 0, got %d", d.Selected)
	}

	d.HandleEvent(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	if d.Selected != 1 {
		t.Fatalf("tab from 0: expected 1, got %d", d.Selected)
	}
}

func TestConfirmDialogEnter(t *testing.T) {
	d := NewConfirmDialogWidget3("msg", "A", "B", "C")
	pressed := -1
	d.OnButton[0] = func() { pressed = 0 }
	d.OnButton[1] = func() { pressed = 1 }
	d.OnButton[2] = func() { pressed = 2 }

	d.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if pressed != 2 {
		t.Fatalf("enter on default(2): expected 2, got %d", pressed)
	}

	d.Selected = 0
	d.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if pressed != 0 {
		t.Fatalf("enter on 0: expected 0, got %d", pressed)
	}
}

func TestConfirmDialogEscape(t *testing.T) {
	d := NewConfirmDialogWidget("msg")
	dismissed := false
	d.OnDismiss = func() { dismissed = true }

	d.HandleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if !dismissed {
		t.Fatal("escape should call OnDismiss")
	}
}

func TestConfirmDialogLetterShortcut(t *testing.T) {
	d := NewConfirmDialogWidget3("msg", "Discard", "Cancel", "Save")
	pressed := -1
	d.OnButton[0] = func() { pressed = 0 }
	d.OnButton[1] = func() { pressed = 1 }
	d.OnButton[2] = func() { pressed = 2 }

	d.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
	if pressed != 2 {
		t.Fatalf("'s' shortcut: expected 2 (Save), got %d", pressed)
	}

	d.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'D', tcell.ModNone))
	if pressed != 0 {
		t.Fatalf("'D' shortcut: expected 0 (Discard), got %d", pressed)
	}
}
