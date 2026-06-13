package ui

import (
	"testing"
)

func TestInputDialogHitRegionsStoredFromRender(t *testing.T) {
	d := NewInputDialogWidget("Title", "placeholder", "")
	surface := makeSurface(80, 24)
	d.Render(surface)

	// After render, hit regions should be populated with valid dimensions
	if d.saveHit.W <= 0 {
		t.Fatal("save hit region has zero or negative width")
	}
	if d.cancelHit.W <= 0 {
		t.Fatal("cancel hit region has zero or negative width")
	}

	// Both buttons should be on the same row
	if d.saveHit.Y != d.cancelHit.Y {
		t.Fatalf("buttons on different rows: save Y=%d, cancel Y=%d", d.saveHit.Y, d.cancelHit.Y)
	}

	// Cancel should be to the left of save
	if d.cancelHit.X >= d.saveHit.X {
		t.Fatalf("cancel should be left of save: cancel X=%d, save X=%d", d.cancelHit.X, d.saveHit.X)
	}

	// Buttons should not overlap
	if d.cancelHit.X+d.cancelHit.W > d.saveHit.X {
		t.Fatalf("buttons overlap: cancel end=%d, save start=%d", d.cancelHit.X+d.cancelHit.W, d.saveHit.X)
	}
}

func TestInputDialogCustomConfirmLabel(t *testing.T) {
	d := NewInputDialogWidget("Title", "placeholder", "")
	d.ConfirmLabel = "Apply"
	surface := makeSurface(80, 24)
	d.Render(surface)

	// Save hit region width should include the custom label + padding
	// " Apply " = 7 runes
	expectedW := len([]rune(" Apply "))
	if d.saveHit.W != expectedW {
		t.Fatalf("save hit width: expected %d, got %d", expectedW, d.saveHit.W)
	}
}
