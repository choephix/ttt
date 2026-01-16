package term

import "testing"

func TestMockScreen_SetAndGetCell(t *testing.T) {
	s := NewMockScreen(10, 5)
	c := Cell{Ch: 'A', Style: 1}
	s.SetCell(2, 3, c)
	got, ok := s.Cells[[2]int{2, 3}]
	if !ok || got.Ch != 'A' || got.Style != 1 {
		t.Errorf("expected cell at (2,3) to be {A,1}, got {%c,%d}", got.Ch, got.Style)
	}
}

func TestMockScreen_Clear(t *testing.T) {
	s := NewMockScreen(5, 5)
	s.SetCell(1, 1, Cell{Ch: 'X'})
	s.Clear()
	if len(s.Cells) != 0 {
		t.Error("expected all cells cleared")
	}
}

func TestMockScreen_Size(t *testing.T) {
	s := NewMockScreen(7, 8)
	w, h := s.Size()
	if w != 7 || h != 8 {
		t.Errorf("expected size 7x8, got %dx%d", w, h)
	}
}
