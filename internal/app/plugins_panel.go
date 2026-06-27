package app

import (
	"github.com/eugenioenko/ttt/internal/plugin"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
)

type PluginsPanel struct {
	Tree    *widgets.TreeWidget
	Adapter *ui.WidgetAdapter
	manager *plugin.Manager

	OnInstall   func(repoURL string)
	OnUninstall func(name string)
	OnToggle    func(name string, enabled bool)
	OnUpdate    func(name string)

	available []plugin.RemoteRegistryEntry
}

func NewPluginsPanel(mgr *plugin.Manager) *PluginsPanel {
	pp := &PluginsPanel{
		manager: mgr,
	}

	pp.Tree = widgets.NewTreeWidget(widgets.TreeConfig{
		EmptyText: "No plugins installed",
		Indent:    1,
		OnCommand: func(cmd string, node *widgets.TreeNode) {
			pp.handleCommand(cmd, node)
		},
	})

	pp.Adapter = ui.NewWidgetAdapter(pp.Tree)
	pp.Refresh()
	return pp
}

func (pp *PluginsPanel) handleCommand(cmd string, node *widgets.TreeNode) {
	switch cmd {
	case "activate":
		if node.ID != "" && node.Expandable {
			return
		}
		for _, entry := range pp.available {
			if node.ID == "available."+entry.Name {
				if pp.OnInstall != nil {
					pp.OnInstall(entry.Repo)
				}
				return
			}
		}
	case "uninstall":
		if pp.OnUninstall != nil {
			pp.OnUninstall(node.ID)
		}
	case "toggle":
		reg := pp.manager.Registry()
		if reg == nil {
			return
		}
		entry := reg.Find(node.ID)
		if entry == nil {
			return
		}
		if pp.OnToggle != nil {
			pp.OnToggle(node.ID, !entry.Enabled)
		}
	case "update":
		if pp.OnUpdate != nil {
			pp.OnUpdate(node.ID)
		}
	}
}

func (pp *PluginsPanel) Refresh() {
	var items []*widgets.TreeNode

	installedSection := &widgets.TreeNode{
		ID:         "_installed",
		Label:      "INSTALLED",
		Expandable: true,
		Expanded:   true,
	}

	reg := pp.manager.Registry()
	plugins := pp.manager.Plugins()
	names := pp.manager.InstalledPluginNames()

	for _, name := range names {
		var badge string
		var icon string
		var iconStyle term.Style

		p := pp.manager.FindPlugin(name)
		if p != nil && p.Enabled {
			badge = "v" + p.Manifest.Version
			icon = "●"
			iconStyle = term.StyleSuccess
		} else {
			if reg != nil {
				if entry := reg.Find(name); entry != nil {
					badge = "v" + entry.Version
				}
			}
			icon = "○"
			iconStyle = term.StyleMuted
		}

		node := &widgets.TreeNode{
			ID:        name,
			Label:     name,
			Badge:     badge,
			Icon:      icon,
			IconStyle: iconStyle,
			Actions: []widgets.Action{
				{Icon: "↑", Command: "update"},
				{Icon: "×", Command: "uninstall"},
			},
		}
		installedSection.Children = append(installedSection.Children, node)
	}

	_ = plugins
	items = append(items, installedSection)

	if len(pp.available) > 0 {
		availableSection := &widgets.TreeNode{
			ID:         "_available",
			Label:      "AVAILABLE",
			Expandable: true,
			Expanded:   true,
		}

		installed := make(map[string]bool)
		for _, name := range names {
			installed[name] = true
		}

		for _, entry := range pp.available {
			if installed[entry.Name] {
				continue
			}
			availableSection.Children = append(availableSection.Children, &widgets.TreeNode{
				ID:    "available." + entry.Name,
				Label: entry.Name,
				Badge: entry.Description,
			})
		}

		if len(availableSection.Children) > 0 {
			items = append(items, availableSection)
		}
	}

	pp.Tree.SetItems(items)
}

func (pp *PluginsPanel) SetAvailable(entries []plugin.RemoteRegistryEntry) {
	pp.available = entries
	pp.Refresh()
}
