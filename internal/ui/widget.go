package ui

import (
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/gdamore/tcell/v2"
)

type Rect = widgets.Rect

type EventResult = widgets.EventResult

const (
	EventIgnored   = widgets.EventIgnored
	EventConsumed  = widgets.EventConsumed
	EventDismissed = widgets.EventDismissed
	EventCaptured  = widgets.EventCaptured
)

type ConstraintType int

const (
	Fixed  ConstraintType = iota
	Flex
	Hidden
)

type LayoutConstraint struct {
	Type  ConstraintType
	Value int
}

type Widget interface {
	SetRect(r Rect)
	GetRect() Rect
	HandleEvent(ev tcell.Event) EventResult
	Render(surface *RenderSurface)
	Focusable() bool
}

type CursorProvider interface {
	CursorPosition() (x, y int, visible bool)
}

// RawKeyConsumer indicates a widget that wants all key events
// sent directly to it, bypassing global key bindings.
// Used by the terminal widget.
type RawKeyConsumer interface {
	WantsRawKeys() bool
}
