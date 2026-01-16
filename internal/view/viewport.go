package view

// Viewport represents the visible region of the buffer.
type Viewport struct {
	TopLine int // first visible line in the buffer
	LeftCol int // first visible column
	Width   int // width of the viewport (in columns)
	Height  int // height of the viewport (in lines)
}

// ScrollVertical scrolls the viewport vertically by n lines.
func (v *Viewport) ScrollVertical(n, maxLines int) {
	v.TopLine += n
	if v.TopLine < 0 {
		v.TopLine = 0
	}
	if v.TopLine > maxLines-v.Height {
		v.TopLine = maxLines - v.Height
		if v.TopLine < 0 {
			v.TopLine = 0
		}
	}
}

// ScrollHorizontal scrolls the viewport horizontally by n columns.
func (v *Viewport) ScrollHorizontal(n, maxCols int) {
	v.LeftCol += n
	if v.LeftCol < 0 {
		v.LeftCol = 0
	}
	if v.LeftCol > maxCols-v.Width {
		v.LeftCol = maxCols - v.Width
		if v.LeftCol < 0 {
			v.LeftCol = 0
		}
	}
}

// CursorScreenCoords maps a buffer cursor position to screen coordinates.
func (v *Viewport) CursorScreenCoords(line, col int) (row, colOnScreen int, visible bool) {
	row = line - v.TopLine
	colOnScreen = col - v.LeftCol
	visible = row >= 0 && row < v.Height && colOnScreen >= 0 && colOnScreen < v.Width
	return
}
