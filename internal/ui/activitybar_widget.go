package ui

import (
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ActivityItem struct {
	Icon rune
	ID   string
}

type ActivityBarWidget struct {
	BaseWidget
	Items    []ActivityItem
	ActiveID string
	Selected int
	OnSelect func(id string)
	Borders  *term.BorderSet
}

func NewActivityBarWidget() *ActivityBarWidget {
	return &ActivityBarWidget{
		Items: []ActivityItem{
			{Icon: 'E', ID: "explorer"},
			{Icon: 'S', ID: "search"},
			{Icon: 'G', ID: "git"},
			{Icon: 'T', ID: "test"},
		},
	}
}

func (a *ActivityBarWidget) Focusable() bool { return true }

func (a *ActivityBarWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	surface.Fill(term.Cell{Ch: ' ', Style: term.StyleActivityBar})

	borderCol := w - 1
	borderCh := '║'
	if a.Borders != nil {
		borderCh = a.Borders.Vertical
	}

	for i, item := range a.Items {
		if i >= h {
			break
		}
		style := term.StyleActivityBar
		if item.ID == a.ActiveID {
			style = term.StyleActivityBarActive
		}

		surface.SetCell(0, i, term.Cell{Ch: ' ', Style: style})
		surface.SetCell(1, i, term.Cell{Ch: item.Icon, Style: style})
		for x := 2; x < borderCol; x++ {
			surface.SetCell(x, i, term.Cell{Ch: ' ', Style: style})
		}
	}

	for y := 0; y < h; y++ {
		surface.SetCell(borderCol, y, term.Cell{Ch: borderCh, Style: term.StyleBorder})
	}
}

func (a *ActivityBarWidget) HandleEvent(ev tcell.Event) EventResult {
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventIgnored
	}

	switch kev.Key() {
	case tcell.KeyUp:
		if a.Selected > 0 {
			a.Selected--
			a.activate()
		}
		return EventConsumed
	case tcell.KeyDown:
		if a.Selected < len(a.Items)-1 {
			a.Selected++
			a.activate()
		}
		return EventConsumed
	case tcell.KeyEnter:
		a.activate()
		return EventConsumed
	}

	return EventIgnored
}

func (a *ActivityBarWidget) activate() {
	if a.Selected >= 0 && a.Selected < len(a.Items) {
		a.ActiveID = a.Items[a.Selected].ID
		if a.OnSelect != nil {
			a.OnSelect(a.ActiveID)
		}
	}
}

func (a *ActivityBarWidget) SetActiveByID(id string) {
	a.ActiveID = id
	for i, item := range a.Items {
		if item.ID == id {
			a.Selected = i
			break
		}
	}
}
