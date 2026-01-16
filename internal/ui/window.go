package ui

import (
	"macro/internal/core/buffer"
	"macro/internal/view"
)

type Rect struct {
	X, Y, W, H int
}

type Window struct {
	Rect  Rect
	View  *view.Viewport
	Buf   *buffer.Buffer
	Focus bool
	Modal bool
}
