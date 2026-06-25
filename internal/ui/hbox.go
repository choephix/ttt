package ui

import "github.com/gdamore/tcell/v2"

type HBox struct {
	BaseWidget
	Children []ChildEntry
}

func (h *HBox) AddChild(w Widget, c LayoutConstraint) {
	h.Children = append(h.Children, ChildEntry{Widget: w, Constraint: c})
}

func (h *HBox) SetChildConstraint(index int, c LayoutConstraint) {
	if index >= 0 && index < len(h.Children) {
		h.Children[index].Constraint = c
	}
}

func (h *HBox) Layout() {
	r := h.GetRect()
	usedWidth := 0
	totalFlex := 0

	for _, child := range h.Children {
		switch child.Constraint.Type {
		case Fixed:
			usedWidth += child.Constraint.Value
		case Flex:
			totalFlex += child.Constraint.Value
		}
	}

	remaining := r.W - usedWidth
	if remaining < 0 {
		remaining = 0
	}

	x := 0
	for _, child := range h.Children {
		var w int
		switch child.Constraint.Type {
		case Fixed:
			w = child.Constraint.Value
		case Flex:
			if totalFlex > 0 {
				w = remaining * child.Constraint.Value / totalFlex
			}
		case Hidden:
			w = 0
		}
		child.Widget.SetRect(Rect{X: r.X + x, Y: r.Y, W: w, H: r.H})
		x += w
	}
}

func (h *HBox) Render(surface *RenderSurface) {
	h.Layout()
	for _, child := range h.Children {
		if child.Constraint.Type == Hidden {
			continue
		}
		cr := child.Widget.GetRect()
		sub := surface.sub(Rect{X: cr.X - h.rect.X, Y: cr.Y - h.rect.Y, W: cr.W, H: cr.H})
		child.Widget.Render(sub)
	}
}

func (h *HBox) HandleEvent(ev tcell.Event) EventResult {
	for _, child := range h.Children {
		if child.Constraint.Type == Hidden {
			continue
		}
		if child.Widget.HandleEvent(ev) == EventConsumed {
			return EventConsumed
		}
	}
	return EventIgnored
}
