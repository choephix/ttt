package cursor

// Cursor represents the position in the buffer.
type Cursor struct {
	Line int // current line
	Col  int // visual column
	Goal int // goal column for vertical movement
}

// MoveLeft moves the cursor one position to the left.
func (c *Cursor) MoveLeft(lineLength int) {
	if c.Col > 0 {
		c.Col--
	}
}

// MoveRight moves the cursor one position to the right.
func (c *Cursor) MoveRight(lineLength int) {
	if c.Col < lineLength {
		c.Col++
	}
}

// MoveUp moves the cursor up one line, preserving the goal column.
func (c *Cursor) MoveUp(prevLineLength int) {
	if c.Line > 0 {
		c.Line--
		c.Col = clampCol(c.Goal, prevLineLength)
	}
}

// MoveDown moves the cursor down one line, preserving the goal column.
func (c *Cursor) MoveDown(nextLineLength, maxLine int) {
	if c.Line < maxLine {
		c.Line++
		c.Col = clampCol(c.Goal, nextLineLength)
	}
}

// SetGoal sets the goal column for vertical movement.
func (c *Cursor) SetGoal() {
	c.Goal = c.Col
}

// clampCol clamps the column to the line length.
func clampCol(goal, lineLength int) int {
	if goal > lineLength {
		return lineLength
	}
	return goal
}
