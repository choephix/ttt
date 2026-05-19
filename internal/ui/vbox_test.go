package ui

import (
	"testing"
)

func TestVBoxFixedAndFlex(t *testing.T) {
	v := &VBox{}
	w1 := &BaseWidget{}
	w2 := &BaseWidget{}
	w3 := &BaseWidget{}

	v.AddChild(w1, LayoutConstraint{Type: Fixed, Value: 1})
	v.AddChild(w2, LayoutConstraint{Type: Flex, Value: 1})
	v.AddChild(w3, LayoutConstraint{Type: Fixed, Value: 1})

	v.SetRect(Rect{X: 0, Y: 0, W: 80, H: 24})
	v.Layout()

	r1 := w1.GetRect()
	r2 := w2.GetRect()
	r3 := w3.GetRect()

	if r1.H != 1 {
		t.Fatalf("w1 height: expected 1, got %d", r1.H)
	}
	if r2.H != 22 {
		t.Fatalf("w2 height: expected 22, got %d", r2.H)
	}
	if r3.H != 1 {
		t.Fatalf("w3 height: expected 1, got %d", r3.H)
	}
	if r2.Y != 1 {
		t.Fatalf("w2 Y: expected 1, got %d", r2.Y)
	}
	if r3.Y != 23 {
		t.Fatalf("w3 Y: expected 23, got %d", r3.Y)
	}

	// All should have full width
	if r1.W != 80 || r2.W != 80 || r3.W != 80 {
		t.Fatalf("expected width 80 for all children")
	}
}

func TestVBoxHidden(t *testing.T) {
	v := &VBox{}
	w1 := &BaseWidget{}
	w2 := &BaseWidget{}
	w3 := &BaseWidget{}

	v.AddChild(w1, LayoutConstraint{Type: Fixed, Value: 1})
	v.AddChild(w2, LayoutConstraint{Type: Hidden})
	v.AddChild(w3, LayoutConstraint{Type: Flex, Value: 1})

	v.SetRect(Rect{X: 0, Y: 0, W: 80, H: 24})
	v.Layout()

	r2 := w2.GetRect()
	r3 := w3.GetRect()

	if r2.H != 0 {
		t.Fatalf("hidden widget height: expected 0, got %d", r2.H)
	}
	if r3.H != 23 {
		t.Fatalf("flex widget height: expected 23, got %d", r3.H)
	}
}

func TestVBoxSetChildConstraint(t *testing.T) {
	v := &VBox{}
	w1 := &BaseWidget{}
	w2 := &BaseWidget{}

	v.AddChild(w1, LayoutConstraint{Type: Fixed, Value: 5})
	v.AddChild(w2, LayoutConstraint{Type: Flex, Value: 1})

	v.SetRect(Rect{X: 0, Y: 0, W: 80, H: 24})
	v.Layout()

	if w2.GetRect().H != 19 {
		t.Fatalf("flex height before toggle: expected 19, got %d", w2.GetRect().H)
	}

	// Hide the first child
	v.SetChildConstraint(0, LayoutConstraint{Type: Hidden})
	v.Layout()

	if w2.GetRect().H != 24 {
		t.Fatalf("flex height after hiding: expected 24, got %d", w2.GetRect().H)
	}
}

func TestVBoxMultipleFlex(t *testing.T) {
	v := &VBox{}
	w1 := &BaseWidget{}
	w2 := &BaseWidget{}

	v.AddChild(w1, LayoutConstraint{Type: Flex, Value: 1})
	v.AddChild(w2, LayoutConstraint{Type: Flex, Value: 1})

	v.SetRect(Rect{X: 0, Y: 0, W: 80, H: 20})
	v.Layout()

	if w1.GetRect().H != 10 {
		t.Fatalf("w1 flex height: expected 10, got %d", w1.GetRect().H)
	}
	if w2.GetRect().H != 10 {
		t.Fatalf("w2 flex height: expected 10, got %d", w2.GetRect().H)
	}
}
