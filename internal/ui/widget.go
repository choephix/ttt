package ui

import "github.com/eugenioenko/ttt/internal/widgets"

type Rect = widgets.Rect

type Surface = widgets.Surface

type BoxModel = widgets.BoxModel

type EventResult = widgets.EventResult

type Widget = widgets.Widget

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

type CursorProvider interface {
	CursorPosition() (x, y int, visible bool)
}

// RawKeyConsumer indicates a widget that wants all key events
// sent directly to it, bypassing global key bindings.
// Used by the terminal widget.
type RawKeyConsumer interface {
	WantsRawKeys() bool
}
