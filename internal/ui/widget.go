package ui

import "github.com/gdamore/tcell/v2"

type EventResult int

const (
	EventIgnored  EventResult = iota
	EventConsumed
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
