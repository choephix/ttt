package ui

import "github.com/gdamore/tcell/v2"

type BaseWidget struct {
	rect Rect
}

func (b *BaseWidget) SetRect(r Rect)                              { b.rect = r }
func (b *BaseWidget) GetRect() Rect                               { return b.rect }
func (b *BaseWidget) HandleEvent(ev tcell.Event) EventResult      { return EventIgnored }
func (b *BaseWidget) Render(surface Surface)                      {}
func (b *BaseWidget) Focusable() bool                             { return false }
