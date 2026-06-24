package app

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
)

type NavigationPanel struct {
	Tree     *widgets.TreeWidget
	Adapter  *ui.WidgetAdapter
	Settings config.ExplorerSettings
	Roots    []string

	OnOpenFile   func(path string)
	OnRightClick func(node *widgets.TreeNode, sx, sy int)
	OnRootMenu   func(node *widgets.TreeNode, sx, sy int)
}

func NewNavigationPanel(settings config.ExplorerSettings, paths ...string) *NavigationPanel {
	n := &NavigationPanel{
		Settings: settings,
		Roots:    paths,
	}

	items := make([]*widgets.TreeNode, len(paths))
	multiRoot := len(paths) > 1
	for i, p := range paths {
		root := &widgets.TreeNode{
			ID:         p,
			Label:      filepath.Base(p),
			Expanded:   !multiRoot,
			Expandable: true,
		}
		items[i] = root
	}

	tree := widgets.NewTreeWidget(widgets.TreeConfig{
		Items: items,
		OnExpand: func(node *widgets.TreeNode) {
			n.loadChildren(node)
		},
		OnCommand: func(cmd string, node *widgets.TreeNode) {
			if cmd == "activate" && n.OnOpenFile != nil {
				n.OnOpenFile(node.ID)
			}
		},
		OnMenu: func(_ []widgets.MenuEntry, node *widgets.TreeNode, sx, sy int) {
			if n.isRoot(node) {
				if n.OnRootMenu != nil {
					n.OnRootMenu(node, sx, sy)
				}
			} else if n.OnRightClick != nil {
				n.OnRightClick(node, sx, sy)
			}
		},
	})
	n.Tree = tree

	for _, root := range items {
		if root.Expanded {
			n.loadChildren(root)
		}
	}
	tree.SetItems(items)

	n.Adapter = ui.NewWidgetAdapter(tree)
	return n
}

func (n *NavigationPanel) isRoot(node *widgets.TreeNode) bool {
	return slices.Contains(n.Roots, node.ID)
}

func (n *NavigationPanel) SetActiveFile(path string) {
	n.Tree.SetActiveID(path)
}

func (n *NavigationPanel) Reload() {
	n.Tree.Reload()
}

func (n *NavigationPanel) loadChildren(node *widgets.TreeNode) {
	entries := ui.LoadDirEntries(node.ID, n.Settings)
	node.Children = nil
	for _, de := range entries {
		child := &widgets.TreeNode{
			ID:         de.Path,
			Label:      de.Name,
			Expandable: de.IsDir,
			Muted:      de.GitIgnored || strings.HasPrefix(de.Name, "."),
		}
		node.Children = append(node.Children, child)
	}
}

func (a *App) navigationNodePath() string {
	if a.NavigationContextNode != nil {
		return a.NavigationContextNode.ID
	}
	if node := a.Navigation.Tree.Selected(); node != nil {
		return node.ID
	}
	return ""
}

func (a *App) navigationReload() {
	a.Navigation.Reload()
	a.Explorer.Reload()
}

func (a *App) NavigateNewFile() {
	a.FileOpNewFile(a.navigationNodePath(), a.navigationReload)
}

func (a *App) NavigateNewFolder() {
	a.FileOpNewFolder(a.navigationNodePath(), a.navigationReload)
}

func (a *App) NavigateRename() {
	a.NavigationContextNode = nil
	a.FileOpRename(a.navigationNodePath(), a.navigationReload)
}

func (a *App) NavigateDelete() {
	a.NavigationContextNode = nil
	a.FileOpDelete(a.navigationNodePath(), a.navigationReload)
}

func (a *App) NavigateCopyAbsolutePath() {
	a.NavigationContextNode = nil
	a.FileOpCopyAbsolutePath(a.navigationNodePath())
}

func (a *App) NavigateCopyRelativePath() {
	a.NavigationContextNode = nil
	a.FileOpCopyRelativePath(a.navigationNodePath())
}

func (a *App) NavigateRemoveRoot() {
	a.NavigationContextNode = nil
	a.FileOpRemoveRoot(a.navigationNodePath())
}
