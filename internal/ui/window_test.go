package ui

import (
	"macro/internal/core/buffer"
	"macro/internal/view"
	"testing"
)

func TestWindow_Basic(t *testing.T) {
	win := &Window{
		Rect:  Rect{X: 0, Y: 0, W: 10, H: 5},
		View:  &view.Viewport{TopLine: 0, LeftCol: 0, Width: 10, Height: 5},
		Buf:   &buffer.Buffer{Lines: []string{"abc"}},
		Focus: true,
		Modal: false,
	}
	if win.Rect.W != 10 || win.View.Width != 10 {
		t.Error("window or viewport width incorrect")
	}
	if win.Buf.Lines[0] != "abc" {
		t.Error("buffer not set correctly")
	}
}
