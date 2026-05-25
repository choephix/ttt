package ui

import "github.com/eugenioenko/ttt/internal/term"

type panelEntry struct {
	ID    string
	Title string
	W     Widget
	Dirty bool
}

type TabbedPanel struct {
	panels      []panelEntry
	ActivePanel string
	TabBar      *PanelTabBarWidget
	Borders     *term.BorderSet
}

func NewTabbedPanel() TabbedPanel {
	return TabbedPanel{
		TabBar: NewPanelTabBarWidget(),
	}
}

func (tp *TabbedPanel) InitTabClick() {
	tp.TabBar.OnTabClick = func(index int) {
		if index >= 0 && index < len(tp.panels) {
			tp.ActivePanel = tp.panels[index].ID
			tp.syncTabs()
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

func (tp *TabbedPanel) PanelIDs() []string {
	ids := make([]string, len(tp.panels))
	for i, p := range tp.panels {
		ids[i] = p.ID
	}
	return ids
}

func (tp *TabbedPanel) SetPanelDirty(id string, dirty bool) {
	for i := range tp.panels {
		if tp.panels[i].ID == id {
			tp.panels[i].Dirty = dirty
			tp.syncTabs()
			return
		}
	}
}

func (tp *TabbedPanel) syncTabs() {
	var tabs []Tab
	for _, p := range tp.panels {
		tabs = append(tabs, Tab{
			Name:   p.Title,
			Active: p.ID == tp.ActivePanel,
			Dirty:  p.Dirty,
		})
	}
	tp.TabBar.SetTabs(tabs)
}
