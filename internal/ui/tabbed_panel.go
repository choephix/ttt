package ui

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/widgets"
)

type panelEntry struct {
	ID    string
	Title string
	W     Widget
}

type TabbedPanel struct {
	ActivePanel   string
	Tabs          *widgets.TabsWidget
	OnPanelChange func(id string)

	panels []panelEntry
}

func NewTabbedPanel() TabbedPanel {
	return TabbedPanel{
		Tabs: widgets.NewTabsWidget(widgets.TabsConfig{}),
	}
}

func (tp *TabbedPanel) InitTabClick() {
	tp.Tabs.Config.OnTabClick = func(index int) {
		if index >= 0 && index < len(tp.panels) {
			tp.SetActivePanel(tp.panels[index].ID)
		}
	}
}

func (tp *TabbedPanel) AddPanel(id, title string, w Widget) {
	tp.panels = append(tp.panels, panelEntry{ID: id, Title: title, W: w})
	if tp.ActivePanel == "" {
		tp.ActivePanel = id
	}
	tp.syncTabs()
}

func (tp *TabbedPanel) RemovePanel(id string) {
	for i, p := range tp.panels {
		if p.ID == id {
			tp.panels = append(tp.panels[:i], tp.panels[i+1:]...)
			if tp.ActivePanel == id {
				if len(tp.panels) > 0 {
					idx := i
					if idx >= len(tp.panels) {
						idx = len(tp.panels) - 1
					}
					tp.ActivePanel = tp.panels[idx].ID
				} else {
					tp.ActivePanel = ""
				}
			}
			tp.syncTabs()
			return
		}
	}
}

func (tp *TabbedPanel) SetActivePanel(id string) {
	for _, p := range tp.panels {
		if p.ID == id {
			tp.ActivePanel = id
			tp.syncTabs()
			if tp.OnPanelChange != nil {
				tp.OnPanelChange(id)
			}
			return
		}
	}
}

func (tp *TabbedPanel) ActiveWidget() Widget {
	for _, p := range tp.panels {
		if p.ID == tp.ActivePanel {
			return p.W
		}
	}
	return nil
}

func (tp *TabbedPanel) PanelCount() int {
	return len(tp.panels)
}

func (tp *TabbedPanel) NextPanel() {
	if len(tp.panels) <= 1 {
		return
	}
	for i, p := range tp.panels {
		if p.ID == tp.ActivePanel {
			next := (i + 1) % len(tp.panels)
			tp.SetActivePanel(tp.panels[next].ID)
			return
		}
	}
}

func (tp *TabbedPanel) PrevPanel() {
	if len(tp.panels) <= 1 {
		return
	}
	for i, p := range tp.panels {
		if p.ID == tp.ActivePanel {
			prev := i - 1
			if prev < 0 {
				prev = len(tp.panels) - 1
			}
			tp.SetActivePanel(tp.panels[prev].ID)
			return
		}
	}
}

func (tp *TabbedPanel) HasPanel(id string) bool {
	for _, p := range tp.panels {
		if p.ID == id {
			return true
		}
	}
	return false
}

func (tp *TabbedPanel) PanelIDs() []string {
	ids := make([]string, len(tp.panels))
	for i, p := range tp.panels {
		ids[i] = p.ID
	}
	return ids
}

type PanelInfo struct {
	ID    string
	Title string
}

func (tp *TabbedPanel) PanelEntries() []PanelInfo {
	entries := make([]PanelInfo, len(tp.panels))
	for i, p := range tp.panels {
		entries[i] = PanelInfo{ID: p.ID, Title: p.Title}
	}
	return entries
}

func (tp *TabbedPanel) SetPanelDirty(id string, dirty bool) {
	tp.Tabs.SetDirty(id, dirty)
}

func (tp *TabbedPanel) HiddenTabs() ([]string, []string) {
	var ids, titles []string
	for _, idx := range tp.Tabs.HiddenTabs() {
		if idx >= 0 && idx < len(tp.panels) {
			ids = append(ids, tp.panels[idx].ID)
			titles = append(titles, tp.panels[idx].Title)
		}
	}
	return ids, titles
}

func (tp *TabbedPanel) syncTabs() {
	dirty := make(map[string]bool)
	for _, item := range tp.Tabs.Config.Items {
		if item.Dirty {
			dirty[item.ID] = true
		}
	}
	items := make([]widgets.TabItem, len(tp.panels))
	for i, p := range tp.panels {
		items[i] = widgets.TabItem{
			ID:     p.ID,
			Label:  p.Title,
			Active: p.ID == tp.ActivePanel,
			Dirty:  dirty[p.ID],
		}
	}
	tp.Tabs.Config.Items = items
}

func (tp *TabbedPanel) RenderTabs(surface Surface, r Rect) {
	tp.Tabs.SetRect(Rect{X: r.X, Y: r.Y, W: r.W, H: 1})
	tp.Tabs.Render(surface.Sub(Rect{X: 0, Y: 0, W: r.W, H: 1}))
}

func (tp *TabbedPanel) RenderDivider(surface Surface, y, w int, borders *term.BorderSet) {
	horizontal := '─'
	if borders != nil {
		horizontal = borders.Horizontal
	}
	for x := 0; x < w; x++ {
		surface.SetCell(x, y, term.Cell{Ch: horizontal, Style: term.StyleBorder})
	}
}
