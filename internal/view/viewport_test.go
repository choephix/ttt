package view

import "testing"

func TestScrollVertical(t *testing.T) {
	v := &Viewport{TopLine: 0, Height: 5}
	v.ScrollVertical(3, 10)
	if v.TopLine != 3 {
		t.Errorf("expected TopLine=3, got %d", v.TopLine)
	}
	v.ScrollVertical(-2, 10)
	if v.TopLine != 1 {
		t.Errorf("expected TopLine=1, got %d", v.TopLine)
	}
	v.ScrollVertical(-5, 10)
	if v.TopLine != 0 {
		t.Errorf("expected TopLine=0, got %d", v.TopLine)
	}
	v.ScrollVertical(100, 10)
	if v.TopLine != 5 {
		t.Errorf("expected TopLine=5, got %d", v.TopLine)
	}
}

func TestScrollHorizontal(t *testing.T) {
	v := &Viewport{LeftCol: 0, Width: 10}
	v.ScrollHorizontal(4, 20)
	if v.LeftCol != 4 {
		t.Errorf("expected LeftCol=4, got %d", v.LeftCol)
	}
	v.ScrollHorizontal(-2, 20)
	if v.LeftCol != 2 {
		t.Errorf("expected LeftCol=2, got %d", v.LeftCol)
	}
	v.ScrollHorizontal(-5, 20)
	if v.LeftCol != 0 {
		t.Errorf("expected LeftCol=0, got %d", v.LeftCol)
	}
	v.ScrollHorizontal(100, 20)
	if v.LeftCol != 10 {
		t.Errorf("expected LeftCol=10, got %d", v.LeftCol)
	}
}

func TestCursorScreenCoords(t *testing.T) {
	v := &Viewport{TopLine: 2, LeftCol: 5, Width: 10, Height: 5}
	row, col, visible := v.CursorScreenCoords(4, 8)
	if row != 2 || col != 3 || !visible {
		t.Errorf("expected row=2, col=3, visible=true; got row=%d, col=%d, visible=%v", row, col, visible)
	}
	_, _, visible = v.CursorScreenCoords(1, 8)
	if visible {
		t.Error("expected not visible for line above viewport")
	}
	_, _, visible = v.CursorScreenCoords(4, 20)
	if visible {
		t.Error("expected not visible for col outside viewport")
	}
}
