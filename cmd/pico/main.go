package main

import (
	"fmt"
	"macro/internal/core/buffer"
	"macro/internal/core/cursor"
	"macro/internal/render"
	"macro/internal/term"
	"macro/internal/view"

	"github.com/gdamore/tcell/v2"
)

func main() {
	// Initialize core components
	buf := &buffer.Buffer{Lines: []string{"Hello, world!", "This is a test buffer.", "Line 3."}}
	cur := &cursor.Cursor{Line: 0, Col: 0}
	viewport := &view.Viewport{TopLine: 0, LeftCol: 0, Width: 40, Height: 5}
	status := &view.StatusBar{FileName: "demo.txt", Line: 0, Col: 0, Dirty: false}
	screen, err := term.NewTcellScreen()
	if err != nil {
		panic(err)
	}
	defer screen.Fini()
	renderer := &render.Renderer{}

	fmt.Println("Editor starting...")

	redraw := func() {
		cells := make([][]term.Cell, viewport.Height+1)
		for y := 0; y < viewport.Height; y++ {
			lineIdx := viewport.TopLine + y
			var line string
			if lineIdx < len(buf.Lines) {
				line = buf.Lines[lineIdx]
			}
			row := make([]term.Cell, viewport.Width)
			for x := 0; x < viewport.Width; x++ {
				ch := ' '
				if x < len([]rune(line)) {
					ch = []rune(line)[x]
				}
				row[x] = term.Cell{Ch: ch}
			}
			cells[y] = row
		}
		// Status bar
		bar := status.RenderStatusBar(viewport.Width)
		barRow := make([]term.Cell, viewport.Width)
		for i, ch := range bar {
			barRow[i] = term.Cell{Ch: ch}
		}
		cells[viewport.Height] = barRow

		renderer.SetCurrent(cells)
		// Show the cursor at the current position relative to viewport
		screenRow := cur.Line - viewport.TopLine
		screenCol := cur.Col - viewport.LeftCol
		screen.ShowCursor(screenCol, screenRow)
		// Now render and show the screen
		renderer.Render(screen)
	}

	redraw()

	// Example event loop structure
	for {
		ev := screen.PollEvent()
		switch tev := ev.(type) {
		case *tcell.EventKey:
			switch tev.Key() {
			case tcell.KeyCtrlC:
				return
			case tcell.KeyUp:
				if cur.Line > 0 {
					cur.Line--
					// Clamp to new line length
					lineLen := len([]rune(buf.Lines[cur.Line]))
					if cur.Col > lineLen {
						cur.Col = lineLen
					}
				}
			case tcell.KeyDown:
				if cur.Line < len(buf.Lines)-1 {
					cur.Line++
					// Clamp to new line length
					lineLen := len([]rune(buf.Lines[cur.Line]))
					if cur.Col > lineLen {
						cur.Col = lineLen
					}
				}
			case tcell.KeyLeft:
				if cur.Col > 0 {
					cur.Col--
				} else if cur.Line > 0 {
					// Move to end of previous line
					cur.Line--
					cur.Col = len([]rune(buf.Lines[cur.Line]))
				}
			case tcell.KeyRight:
				lineLen := len([]rune(buf.Lines[cur.Line]))
				if cur.Col < lineLen {
					cur.Col++
				} else if cur.Line < len(buf.Lines)-1 {
					// Move to start of next line
					cur.Line++
					cur.Col = 0
				}
			case tcell.KeyEnter:
				line := buf.Lines[cur.Line]
				rline := []rune(line)
				// Clamp cursor position
				if cur.Col < 0 {
					cur.Col = 0
				}
				if cur.Col > len(rline) {
					cur.Col = len(rline)
				}
				left := string(rline[:cur.Col])
				right := string(rline[cur.Col:])
				buf.Lines[cur.Line] = left
				buf.InsertLine(cur.Line+1, right)
				cur.Line++
				cur.Col = 0
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if cur.Col > 0 {
					buf.DeleteRune(cur.Line, cur.Col-1)
					cur.Col--
				} else if cur.Line > 0 {
					prevLen := len([]rune(buf.Lines[cur.Line-1]))
					buf.Lines[cur.Line-1] += buf.Lines[cur.Line]
					buf.DeleteLine(cur.Line)
					cur.Line--
					cur.Col = prevLen
				}
			default:
				r := tev.Rune()
				if r != 0 && tev.Modifiers() == 0 {
					buf.InsertRune(cur.Line, cur.Col, r)
					cur.Col++
				}
			}

			// Safety clamp - only if something went wrong
			if cur.Line < 0 {
				cur.Line = 0
			}
			if cur.Line >= len(buf.Lines) {
				cur.Line = len(buf.Lines) - 1
			}
			if cur.Col < 0 {
				cur.Col = 0
			}

			// Note: Don't clamp col to line length here for normal navigation
			// Only clamp when changing lines (which is done in the specific cases above)

			// Safety clamp - only if something went wrong
			if cur.Line < 0 {
				cur.Line = 0
			}
			if cur.Line >= len(buf.Lines) {
				cur.Line = len(buf.Lines) - 1
			}
			if cur.Col < 0 {
				cur.Col = 0
			}

			// Note: Don't clamp col to line length here for normal navigation
			// Only clamp when changing lines (which is done in the specific cases above)

			// Ensure viewport follows cursor
			if cur.Line < viewport.TopLine {
				viewport.TopLine = cur.Line
			}
			if cur.Line >= viewport.TopLine+viewport.Height {
				viewport.TopLine = cur.Line - viewport.Height + 1
			}
			if cur.Col < viewport.LeftCol {
				viewport.LeftCol = cur.Col
			}
			if cur.Col >= viewport.LeftCol+viewport.Width {
				viewport.LeftCol = cur.Col - viewport.Width + 1
			}

			// Update status bar with cursor position
			status.Line = cur.Line
			status.Col = cur.Col
			redraw()
		case *tcell.EventResize:
			w, h := screen.Size()
			viewport.Width = w
			viewport.Height = h - 1
			redraw()
		}
	}
}
