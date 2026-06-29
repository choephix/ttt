package app

import (
	"strings"

	"github.com/eugenioenko/ttt/internal/plugin"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
)

type PluginsPanel struct {
	SearchInput   *widgets.InputWidget
	SearchTree    *widgets.TreeWidget
	InstalledTree *widgets.TreeWidget
	Dropdown      *widgets.DropdownWidget
	Adapter       *ui.WidgetAdapter
	manager       *plugin.Manager

	OnInstall      func(repoURL, repoPath, name string)
	OnUninstall    func(name string)
	OnToggle       func(name string, enabled bool)
	OnUpdate       func(name string)
	OnOpenDetail   func(entry plugin.RemoteRegistryEntry)
	OnDropdownMenu func(entries []widgets.MenuEntry, screenX, screenY int)

	available   []plugin.RemoteRegistryEntry
	searchQuery string
}

func NewPluginsPanel(mgr *plugin.Manager) *PluginsPanel {
	pp := &PluginsPanel{
		manager: mgr,
	}

	pp.SearchInput = widgets.NewInputWidget(widgets.InputConfig{
		Placeholder: "Search plugins",
		OnChange: func(text string) {
			pp.searchQuery = strings.TrimSpace(text)
			pp.refreshAvailable()
		},
	})

	pp.SearchTree = widgets.NewTreeWidget(widgets.TreeConfig{
		EmptyText: "Loading plugins...",
		Indent:    1,
		OnCommand: func(cmd string, node *widgets.TreeNode) {
			if cmd == "activate" {
				for _, entry := range pp.available {
					if node.ID == "available."+entry.Name {
						if pp.OnOpenDetail != nil {
							pp.OnOpenDetail(entry)
						}
						return
					}
				}
			}
		},
	})

	pp.InstalledTree = widgets.NewTreeWidget(widgets.TreeConfig{
		EmptyText: "No plugins installed",
		Indent:    1,
		OnCommand: func(cmd string, node *widgets.TreeNode) {
			pp.handleCommand(cmd, node)
		},
	})

	pp.Dropdown = widgets.NewDropdownWidget(widgets.DropdownConfig{
		Entries: []widgets.MenuEntry{
			{Label: "Update All", Command: "updateAll"},
			{Label: "Refresh", Command: "refresh"},
		},
		OnMenu: func(entries []widgets.MenuEntry, screenX, screenY int) {
			if pp.OnDropdownMenu != nil {
				pp.OnDropdownMenu(entries, screenX, screenY)
			}
		},
	})

	titleLabel := widgets.NewLabelWidget(widgets.LabelConfig{
		Text:  "Installed",
		Style: term.StyleDefault,
	})

	titleLabel.Box.PaddingLeft = 1

	divider := widgets.NewDividerWidget(widgets.DividerConfig{})

	titleRow := widgets.NewHStackWidget(titleLabel, pp.Dropdown)
	titleRow.FixedHeight = 1

	divSearch := widgets.NewDividerWidget(widgets.DividerConfig{})
	divInstalled := widgets.NewDividerWidget(widgets.DividerConfig{})

	vstack := widgets.NewVStackWidget(
		pp.SearchInput,
		divSearch,
		pp.SearchTree,
		divInstalled,
		titleRow,
		divider,
		pp.InstalledTree,
	)

	pp.Adapter = ui.NewWidgetAdapter(vstack)
	pp.Refresh()
	return pp
}

func (pp *PluginsPanel) handleCommand(cmd string, node *widgets.TreeNode) {
	switch cmd {
	case "activate", "toggle":
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
	case "uninstall":
		if pp.OnUninstall != nil {
			pp.OnUninstall(node.ID)
		}
	case "update":
		if pp.OnUpdate != nil {
			pp.OnUpdate(node.ID)
		}
	}
}

func (pp *PluginsPanel) Refresh() {
	var installed []*widgets.TreeNode

	reg := pp.manager.Registry()
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
		installed = append(installed, node)
	}

	pp.InstalledTree.SetItems(installed)
	pp.refreshAvailable()
}

func (pp *PluginsPanel) refreshAvailable() {
	if len(pp.available) == 0 {
		pp.SearchTree.SetItems(nil)
		return
	}

	installedSet := make(map[string]bool)
	for _, name := range pp.manager.InstalledPluginNames() {
		installedSet[name] = true
	}

	var items []*widgets.TreeNode
	for _, entry := range pp.available {
		if installedSet[entry.Name] {
			continue
		}
		if pp.searchQuery != "" && !matchesSearch(entry, pp.searchQuery) {
			continue
		}
		items = append(items, &widgets.TreeNode{
			ID:    "available." + entry.Name,
			Label: entry.Name,
			Badge: entry.Description,
		})
	}

	pp.SearchTree.SetItems(items)
}

func matchesSearch(entry plugin.RemoteRegistryEntry, query string) bool {
	query = strings.ToLower(query)
	terms := strings.Fields(query)
	for _, t := range terms {
		if !termMatches(entry, t) {
			return false
		}
	}
	return true
}

func termMatches(entry plugin.RemoteRegistryEntry, t string) bool {
	if strings.Contains(strings.ToLower(entry.Name), t) {
		return true
	}
	if strings.Contains(strings.ToLower(entry.Description), t) {
		return true
	}
	if strings.Contains(strings.ToLower(entry.Author), t) {
		return true
	}
	for _, tag := range entry.Tags {
		if strings.Contains(strings.ToLower(tag), t) {
			return true
		}
	}
	return false
}

func (pp *PluginsPanel) SetAvailable(entries []plugin.RemoteRegistryEntry) {
	pp.available = entries
	pp.SearchTree.Config.EmptyText = "Type to search plugins"
	pp.Refresh()
}
