package ui

import "github.com/gdamore/tcell/v2"

type ChildEntry struct {
	Widget     Widget
	Constraint LayoutConstraint
}

type VBox struct {
	BaseWidget
	Children []ChildEntry
}

func (v *VBox) AddChild(w Widget, c LayoutConstraint) {
	v.Children = append(v.Children, ChildEntry{Widget: w, Constraint: c})
}

func (v *VBox) SetChildConstraint(index int, c LayoutConstraint) {
	if index >= 0 && index < len(v.Children) {
		v.Children[index].Constraint = c
	}
}

func (v *VBox) Layout() {
	r := v.GetRect()
	usedHeight := 0
	totalFlex := 0

	for _, child := range v.Children {
		switch child.Constraint.Type {
		case Fixed:
			usedHeight += child.Constraint.Value
		case Flex:
			totalFlex += child.Constraint.Value
		}
	}

	remaining := r.H - usedHeight
	if remaining < 0 {
		remaining = 0
	}

	y := 0
	for _, child := range v.Children {
		var h int
		switch child.Constraint.Type {
		case Fixed:
			h = child.Constraint.Value
		case Flex:
			if totalFlex > 0 {
				h = remaining * child.Constraint.Value / totalFlex
			}
		case Hidden:
			h = 0
		}
		child.Widget.SetRect(Rect{X: r.X, Y: r.Y + y, W: r.W, H: h})
		y += h
	}
}

func (v *VBox) Render(surface Surface) {
	v.Layout()
	for _, child := range v.Children {
		if child.Constraint.Type == Hidden {
			continue
		}
		cr := child.Widget.GetRect()
		sub := surface.Sub(Rect{X: cr.X - v.rect.X, Y: cr.Y - v.rect.Y, W: cr.W, H: cr.H})
		child.Widget.Render(sub)
	}
}

func (v *VBox) HandleEvent(ev tcell.Event) EventResult {
	for _, child := range v.Children {
		if child.Constraint.Type == Hidden {
			continue
		}
		if child.Widget.HandleEvent(ev) == EventConsumed {
			return EventConsumed
		}
	}
	return EventIgnored
}
