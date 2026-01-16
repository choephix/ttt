package cursor

import "testing"

func TestMoveLeft(t *testing.T) {
	c := &Cursor{Line: 0, Col: 2}
	c.MoveLeft(5)
	if c.Col != 1 {
		t.Errorf("expected Col=1, got %d", c.Col)
	}
	c.MoveLeft(5)
	c.MoveLeft(5)
	if c.Col != 0 {
		t.Errorf("expected Col=0, got %d", c.Col)
	}
}

func TestMoveRight(t *testing.T) {
	c := &Cursor{Line: 0, Col: 0}
	c.MoveRight(3)
	if c.Col != 1 {
		t.Errorf("expected Col=1, got %d", c.Col)
	}
	c.MoveRight(3)
	c.MoveRight(3)
	if c.Col != 3 {
		t.Errorf("expected Col=3, got %d", c.Col)
	}
}

func TestMoveUpDown(t *testing.T) {
	c := &Cursor{Line: 1, Col: 2, Goal: 2}
	c.MoveUp(1)
	if c.Line != 0 || c.Col != 1 {
		t.Errorf("expected Line=0, Col=1, got Line=%d, Col=%d", c.Line, c.Col)
	}
	c.MoveDown(3, 2)
	if c.Line != 1 || c.Col != 2 {
		t.Errorf("expected Line=1, Col=2, got Line=%d, Col=%d", c.Line, c.Col)
	}
}

func TestSetGoal(t *testing.T) {
	c := &Cursor{Col: 5}
	c.SetGoal()
	if c.Goal != 5 {
		t.Errorf("expected Goal=5, got %d", c.Goal)
	}
}
