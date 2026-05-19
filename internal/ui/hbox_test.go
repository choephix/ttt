package ui

import (
	"testing"
)

func TestHBoxFixedAndFlex(t *testing.T) {
	h := &HBox{}
	w1 := &BaseWidget{}
	w2 := &BaseWidget{}
	w3 := &BaseWidget{}

	h.AddChild(w1, LayoutConstraint{Type: Fixed, Value: 2})
	h.AddChild(w2, LayoutConstraint{Type: Fixed, Value: 30})
	h.AddChild(w3, LayoutConstraint{Type: Flex, Value: 1})

	h.SetRect(Rect{X: 0, Y: 0, W: 80, H: 24})
	h.Layout()

	r1 := w1.GetRect()
	r2 := w2.GetRect()
	r3 := w3.GetRect()

	if r1.W != 2 {
		t.Fatalf("w1 width: expected 2, got %d", r1.W)
	}
	if r2.W != 30 {
		t.Fatalf("w2 width: expected 30, got %d", r2.W)
	}
	if r3.W != 48 {
		t.Fatalf("w3 width: expected 48, got %d", r3.W)
	}
	if r2.X != 2 {
		t.Fatalf("w2 X: expected 2, got %d", r2.X)
	}
	if r3.X != 32 {
		t.Fatalf("w3 X: expected 32, got %d", r3.X)
	}

	// All should have full height
	if r1.H != 24 || r2.H != 24 || r3.H != 24 {
		t.Fatalf("expected height 24 for all children")
	}
}

func TestHBoxHidden(t *testing.T) {
	h := &HBox{}
	w1 := &BaseWidget{}
	w2 := &BaseWidget{}
	w3 := &BaseWidget{}

	h.AddChild(w1, LayoutConstraint{Type: Fixed, Value: 2})
	h.AddChild(w2, LayoutConstraint{Type: Hidden})
	h.AddChild(w3, LayoutConstraint{Type: Flex, Value: 1})

	h.SetRect(Rect{X: 0, Y: 0, W: 80, H: 24})
	h.Layout()

	r2 := w2.GetRect()
	r3 := w3.GetRect()

	if r2.W != 0 {
		t.Fatalf("hidden widget width: expected 0, got %d", r2.W)
	}
	if r3.W != 78 {
		t.Fatalf("flex widget width: expected 78, got %d", r3.W)
	}
}

func TestHBoxSetChildConstraint(t *testing.T) {
	h := &HBox{}
	w1 := &BaseWidget{}
	w2 := &BaseWidget{}

	h.AddChild(w1, LayoutConstraint{Type: Fixed, Value: 30})
	h.AddChild(w2, LayoutConstraint{Type: Flex, Value: 1})

	h.SetRect(Rect{X: 0, Y: 0, W: 80, H: 24})
	h.Layout()

	if w2.GetRect().W != 50 {
		t.Fatalf("flex width before toggle: expected 50, got %d", w2.GetRect().W)
	}

	h.SetChildConstraint(0, LayoutConstraint{Type: Hidden})
	h.Layout()

	if w2.GetRect().W != 80 {
		t.Fatalf("flex width after hiding: expected 80, got %d", w2.GetRect().W)
	}
}
