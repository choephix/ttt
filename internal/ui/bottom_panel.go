package ui

import (
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

type panelEntry struct {
	ID    string
	Title string
	W     Widget
}

type BottomPanelWidget struct {
	BaseWidget
	TabBar      *PanelTabBarWidget
	panels      []panelEntry
	ActivePanel string
	Visible     bool
	Borders     *term.BorderSet
}

func NewBottomPanelWidget(borders *term.BorderSet) *BottomPanelWidget {
	tabBar := NewPanelTabBarWidget()
	tabBar.Borders = borders
	return &BottomPanelWidget{
		TabBar:  tabBar,
		Visible: true,
		Borders: borders,
	}
}

func (bp *BottomPanelWidget) AddPanel(id, title string, w Widget) {
	bp.panels = append(bp.panels, panelEntry{ID: id, Title: title, W: w})
	if bp.ActivePanel == "" {
		bp.ActivePanel = id
	}
	bp.syncTabs()
}

func (bp *BottomPanelWidget) SetActivePanel(id string) {
	for _, p := range bp.panels {
		if p.ID == id {
			bp.ActivePanel = id
			bp.syncTabs()
			return
		}
	}
}

func (bp *BottomPanelWidget) ActiveWidget() Widget {
	for _, p := range bp.panels {
		if p.ID == bp.ActivePanel {
			return p.W
		}
	}
	return nil
}

func (bp *BottomPanelWidget) Focusable() bool { return true }

func (bp *BottomPanelWidget) syncTabs() {
	var tabs []Tab
	for _, p := range bp.panels {
		tabs = append(tabs, Tab{
			Name:   p.Title,
			Active: p.ID == bp.ActivePanel,
		})
	}
	bp.TabBar.SetTabs(tabs)
}

func (bp *BottomPanelWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	r := bp.GetRect()

	if h < 3 {
		return
	}

	bs := term.StyleBorder
	horizontal := '─'
	if bp.Borders != nil {
		horizontal = bp.Borders.Horizontal
	}

	// Tab bar: 1 row
	bp.TabBar.SetRect(Rect{X: r.X, Y: r.Y, W: r.W, H: 1})
	tabSurface := surface.Sub(Rect{X: 0, Y: 0, W: w, H: 1})
	bp.TabBar.Render(tabSurface)

	// Divider line below tabs
	for x := 0; x < w; x++ {
		surface.SetCell(x, 1, term.Cell{Ch: horizontal, Style: bs})
	}

	// Content area below divider
	contentH := h - 2
	active := bp.ActiveWidget()
	if active != nil && contentH > 0 {
		active.SetRect(Rect{X: r.X, Y: r.Y + 2, W: r.W, H: contentH})
		contentSurface := surface.Sub(Rect{X: 0, Y: 2, W: w, H: contentH})
		active.Render(contentSurface)
	}
}

func (bp *BottomPanelWidget) HandleEvent(ev tcell.Event) EventResult {
	active := bp.ActiveWidget()
	if active != nil {
		return active.HandleEvent(ev)
	}
	return EventIgnored
}
