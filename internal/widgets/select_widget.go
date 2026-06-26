package widgets

import (
	"strings"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type SelectItem struct {
	ID    string
	Label string
}

type SelectConfig struct {
	Items       []SelectItem
	Placeholder string
	ShowDivider bool
	Collapsible bool
	OnSelect    func(id string)
	OnChange    func(id string)
	OnDismiss   func()
}

type SelectWidget struct {
	BaseWidget
	Config   SelectConfig
	input    *InputWidget
	divider  *DividerWidget
	filtered []int
	selected int
	focused  bool

	scrollTop int
	scrollbar scrollbar
}

func NewSelectWidget(config SelectConfig) *SelectWidget {
	s := &SelectWidget{
		Config:  config,
		divider: NewDividerWidget(DividerConfig{}),
	}
	s.input = NewInputWidget(InputConfig{
		Placeholder: config.Placeholder,
		OnChange: func(_ string) {
			s.filter()
			s.selected = 0
			s.scrollTop = 0
			if s.Config.OnChange != nil && len(s.filtered) > 0 {
				s.Config.OnChange(s.Config.Items[s.filtered[0]].ID)
			}
		},
	})
	s.filter()
	return s
}

func (s *SelectWidget) Height() int {
	if s.Config.Collapsible {
		return 1
	}
	h := len(s.Config.Items) + 1
	if s.Config.ShowDivider {
		h += 2
	}
	max := 11
	if h > max {
		h = max
	}
	return h
}

func (s *SelectWidget) popupHeight() int {
	h := len(s.filtered)
	max := 8
	if h > max {
		h = max
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (s *SelectWidget) HasPopup() bool {
	return s.Config.Collapsible && s.focused
}

func (s *SelectWidget) PopupRect() Rect {
	r := s.GetRect()
	return Rect{X: r.X, Y: r.Y + 1, W: r.W, H: s.popupHeight()}
}

func (s *SelectWidget) RenderPopup(surface Surface) {
	pr := s.PopupRect()
	w, h := pr.W, pr.H

	s.ensureVisible(h)

	s.scrollbar.X = pr.X + w - 1
	s.scrollbar.Y = pr.Y
	s.scrollbar.Height = h
	s.scrollbar.TotalItems = len(s.filtered)
	s.scrollbar.TopItem = s.scrollTop

	for i := range h {
		idx := s.scrollTop + i
		if idx >= len(s.filtered) {
			break
		}
		item := s.Config.Items[s.filtered[idx]]
		style := term.StylePaletteItem
		if idx == s.selected {
			style = term.StylePaletteSelected
		}
		for x := range w {
			surface.SetCell(x, i, term.Cell{Ch: ' ', Style: style})
		}
		surface.DrawText(1, i, item.Label, w-1, style)
	}

	s.scrollbar.Render(surface, w-1, 0)
}
func (s *SelectWidget) Width() int  { return 0 }

func (s *SelectWidget) Focusable() bool   { return true }
func (s *SelectWidget) SetFocused(f bool) { s.focused = f; s.input.SetFocused(f) }
func (s *SelectWidget) IsFocused() bool   { return s.focused }

func (s *SelectWidget) CursorPosition() (x, y int, visible bool) {
	return s.input.CursorPosition()
}

func (s *SelectWidget) filter() {
	query := strings.ToLower(strings.TrimSpace(s.input.Text()))
	s.filtered = s.filtered[:0]
	for i, item := range s.Config.Items {
		if query == "" || strings.Contains(strings.ToLower(item.Label), query) {
			s.filtered = append(s.filtered, i)
		}
	}
}

func (s *SelectWidget) visibleListH() int {
	if s.Config.Collapsible {
		return s.popupHeight()
	}
	listStart := 1
	if s.Config.ShowDivider {
		listStart = 3
	}
	return s.rect.H - listStart
}

func (s *SelectWidget) ensureVisible(visibleH int) {
	if visibleH <= 0 {
		return
	}
	if s.selected < s.scrollTop {
		s.scrollTop = s.selected
	}
	if s.selected >= s.scrollTop+visibleH {
		s.scrollTop = s.selected - visibleH + 1
	}
}

func (s *SelectWidget) Render(surface Surface) {
	w, h := surface.Size()
	if w <= 0 || h <= 0 {
		return
	}

	y := 0
	if s.Config.ShowDivider {
		s.divider.SetRect(Rect{X: s.rect.X, Y: s.rect.Y + y, W: s.rect.W, H: 1})
		divSurface := surface.Sub(Rect{X: 0, Y: y, W: w, H: 1})
		s.divider.Render(divSurface)
		y++
	}

	s.input.SetRect(Rect{X: s.rect.X, Y: s.rect.Y + y, W: s.rect.W, H: 1})
	inputSurface := surface.Sub(Rect{X: 0, Y: y, W: w, H: 1})
	s.input.Render(inputSurface)

	chevron := '▼'
	if s.focused {
		chevron = '▲'
	}
	surface.SetCell(w-2, y, term.Cell{Ch: chevron, Style: term.StyleMuted})
	y++

	if s.Config.Collapsible {
		return
	}

	if s.Config.ShowDivider {
		s.divider.SetRect(Rect{X: s.rect.X, Y: s.rect.Y + y, W: s.rect.W, H: 1})
		divSurface := surface.Sub(Rect{X: 0, Y: y, W: w, H: 1})
		s.divider.Render(divSurface)
		y++
	}

	listStart := y
	listH := h - listStart
	if listH <= 0 {
		return
	}

	s.scrollbar.X = s.rect.X + w - 1
	s.scrollbar.Y = s.rect.Y + listStart
	s.scrollbar.Height = listH
	s.scrollbar.TotalItems = len(s.filtered)
	s.scrollbar.TopItem = s.scrollTop

	for i := range listH {
		idx := s.scrollTop + i
		ly := listStart + i
		if idx >= len(s.filtered) {
			break
		}
		item := s.Config.Items[s.filtered[idx]]
		style := term.StylePaletteItem
		if idx == s.selected {
			style = term.StylePaletteSelected
		}
		for x := range w {
			surface.SetCell(x, ly, term.Cell{Ch: ' ', Style: style})
		}
		surface.DrawText(1, ly, item.Label, w-1, style)
	}

	s.scrollbar.Render(surface, w-1, listStart)
}

func (s *SelectWidget) HandleEvent(ev tcell.Event) EventResult {
	if newTop, consumed := s.scrollbar.HandleEvent(ev); consumed {
		s.scrollTop = newTop
		return EventConsumed
	}

	switch tev := ev.(type) {
	case *tcell.EventKey:
		return s.handleKey(tev)
	case *tcell.EventMouse:
		return s.handleMouse(tev)
	}
	return EventIgnored
}

func (s *SelectWidget) handleKey(ev *tcell.EventKey) EventResult {
	switch ev.Key() {
	case tcell.KeyUp:
		if s.selected > 0 {
			s.selected--
			s.ensureVisible(s.visibleListH())
			s.notifyChange()
		}
		return EventConsumed
	case tcell.KeyDown:
		if s.selected < len(s.filtered)-1 {
			s.selected++
			s.ensureVisible(s.visibleListH())
			s.notifyChange()
		}
		return EventConsumed
	case tcell.KeyEnter:
		if s.selected >= 0 && s.selected < len(s.filtered) && s.Config.OnSelect != nil {
			s.Config.OnSelect(s.Config.Items[s.filtered[s.selected]].ID)
		}
		return EventConsumed
	case tcell.KeyEscape:
		if s.Config.OnDismiss != nil {
			s.Config.OnDismiss()
		}
		return EventConsumed
	default:
		return s.input.HandleEvent(ev)
	}
}

func (s *SelectWidget) handleMouse(ev *tcell.EventMouse) EventResult {
	btn := ev.Buttons()
	_, my := ev.Position()
	r := s.rect

	if btn&tcell.WheelUp != 0 {
		s.scrollTop -= 3
		if s.scrollTop < 0 {
			s.scrollTop = 0
		}
		return EventConsumed
	}
	if btn&tcell.WheelDown != 0 {
		s.scrollTop += 3
		maxScroll := len(s.filtered) - s.visibleListH()
		if maxScroll < 0 {
			maxScroll = 0
		}
		if s.scrollTop > maxScroll {
			s.scrollTop = maxScroll
		}
		return EventConsumed
	}

	if btn&tcell.Button1 != 0 {
		if s.Config.Collapsible && s.focused {
			pr := s.PopupRect()
			if my >= pr.Y && my < pr.Y+pr.H {
				idx := s.scrollTop + (my - pr.Y)
				if idx >= 0 && idx < len(s.filtered) {
					s.selected = idx
					s.notifyChange()
					if s.Config.OnSelect != nil {
						s.Config.OnSelect(s.Config.Items[s.filtered[s.selected]].ID)
					}
				}
				return EventConsumed
			}
			return s.input.HandleEvent(ev)
		}
		listStart := r.Y + 1
		if s.Config.ShowDivider {
			listStart += 2
		}
		if my < listStart {
			return s.input.HandleEvent(ev)
		}
		idx := s.scrollTop + (my - listStart)
		if idx >= 0 && idx < len(s.filtered) {
			s.selected = idx
			s.notifyChange()
			if s.Config.OnSelect != nil {
				s.Config.OnSelect(s.Config.Items[s.filtered[s.selected]].ID)
			}
			return EventConsumed
		}
	}
	return EventIgnored
}

func (s *SelectWidget) notifyChange() {
	if s.Config.OnChange != nil && s.selected >= 0 && s.selected < len(s.filtered) {
		s.Config.OnChange(s.Config.Items[s.filtered[s.selected]].ID)
	}
}
