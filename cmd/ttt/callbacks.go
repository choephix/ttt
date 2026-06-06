package main

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/git"
	"github.com/eugenioenko/ttt/internal/ui"

	"github.com/gdamore/tcell/v2"
)

func (a *App) ShowSidebarMoreMenu(sx, sy int) {
	var items []ui.ContextMenuItem
	switch a.sidebar.ActivePanel {
	case "explorer":
		items = []ui.ContextMenuItem{
			{Label: "New File", Command: "file.new"},
			{Label: "Add Folder", Command: "workspace.addFolder"},
			{Label: "Refresh", Command: "explorer.refresh"},
		}
	case "search":
		replaceLabel := "Replace"
		if a.search.IsReplaceMode() {
			replaceLabel = "Search"
		}
		items = []ui.ContextMenuItem{
			{Label: replaceLabel, Shortcut: a.KeyFor("sidebar.searchReplace"), Command: "sidebar.searchReplace"},
			ui.MenuSep(),
			{Label: "Clear Results", Command: "search.clear"},
		}
	case "changes":
		items = []ui.ContextMenuItem{
			{Label: "Refresh", Command: "changes.refresh"},
			{Label: "Open Pull Request", Command: "pr.open"},
			ui.MenuSep(),
			{Label: "Pull", Command: "git.pull"},
			{Label: "Push", Command: "git.push"},
			{Label: "Sync", Command: "git.sync"},
		}
	}
	if len(items) > 0 {
		openContextMenu(a, items, sx, sy)
	}
}

func registerWidgetCallbacks(app *App) {
	reg := app.reg

	for i := range menuBarMenus {
		idx := i
		reg.Register(command.Command{
			ID:    menuBarLabels[idx],
			Title: "Open " + menuBarLabels[idx] + " Menu",
			Handler: func() {
				openMenuBarDropdown(app, idx)
			},
		})
	}

	app.menuBar.OnSelect = func(index int) {
		openMenuBarDropdown(app, index)
	}

	app.root.OnRightClick = func(mx, my int) {
		handleRightClick(app, mx, my)
	}

	app.splitPanel.OnLeftClick = func() {
		reg.Execute("sidebar.focus")
	}
	app.splitPanel.OnRightClick = func() {}

	app.sidebar.MoreButton.OnClick = app.ShowSidebarMoreMenu

	app.sidebar.OnPanelChange = func(id string) {
		if id == "search" {
			app.applySearchHighlights()
		} else {
			app.editorGroup.ClearSearch()
		}
		if id == "changes" {
			app.changes.Refresh()
		}
	}

	app.sidebar.OnTabOverflow = func(ids []string, titles []string, sx, sy int) {
		var items []ui.ContextMenuItem
		for i, id := range ids {
			panelID := id
			items = append(items, ui.ContextMenuItem{Label: titles[i], Command: "sidebar.overflow." + panelID})
			reg.Register(command.Command{
				ID:      "sidebar.overflow." + panelID,
				Title:   titles[i],
				Handler: func() { app.sidebar.SetActivePanel(panelID) },
			})
		}
		openContextMenu(app, items, sx, sy)
	}

	app.editorGroup.TabBar.OnTabClose = func(index int) {
		app.editorGroup.SwitchTab(index)
		reg.Execute("tab.close")
	}

	app.editorGroup.TabBar.MoreButton.OnClick = func(sx, sy int) {
		moreMenu := []ui.ContextMenuItem{
			{Label: "Close All", Command: "tab.closeAll"},
		}
		openContextMenu(app, moreMenu, sx, sy)
	}

	app.editorGroup.TabBar.OnTabRightClick = func(index, sx, sy int) {
		app.editorGroup.SwitchTab(index)
		tabContextMenu := []ui.ContextMenuItem{
			{Label: "Close", Shortcut: app.KeyFor("tab.close"), Command: "tab.close"},
			{Label: "Close Others", Shortcut: "", Command: "tab.closeOthers"},
			{Label: "Close All", Shortcut: "", Command: "tab.closeAll"},
		}
		openContextMenu(app, tabContextMenu, sx, sy)
	}

	app.explorer.OnOpenFile = func(path string) {
		app.editorGroup.OpenFile(path)
		app.root.SetFocus(app.editorGroup)
	}

	app.search.OnClear = func() {
		app.editorGroup.ClearSearch()
	}
	app.search.PostEvent = func() {
		app.screen.PostEvent(tcell.NewEventInterrupt(nil))
	}
	app.search.DiffSources = app.DiffSearchSources
	app.search.OnOpenMatch = app.NavigateToSearchMatch
	app.search.OnPreview = app.PreviewSearchReplace
	app.search.OnReplace = app.ApplySearchReplace
	app.search.OnReplaceAll = app.ApplySearchReplaceAll

	app.explorer.OnRightClick = func(node *ui.TreeNode, sx, sy int) {
		items := []ui.ContextMenuItem{
			{Label: "Open", Command: "explorer.open"},
			ui.MenuSep(),
			{Label: "New File", Command: "explorer.newFile"},
			{Label: "New Folder", Command: "explorer.newFolder"},
			ui.MenuSep(),
			{Label: "Rename", Command: "explorer.rename"},
			{Label: "Delete", Command: "explorer.delete"},
		}
		openContextMenu(app, items, sx, sy)
	}

	app.changes.OnRightClick = func(dir string, status git.FileStatus, sx, sy int) {
		if status.Staged {
			openContextMenu(app, changesContextMenuStaged, sx, sy)
		} else {
			openContextMenu(app, changesContextMenuUnstaged, sx, sy)
		}
	}

	app.changes.OnOpenDiff = app.OpenChangeDiff
	app.changes.OnOpenPRDiff = app.OpenPRDiff
	app.changes.OnPRGroupMenu = app.ShowPRGroupMenu
	app.changes.OnGroupMenu = app.ShowGroupMenu
	app.changes.OnCommit = app.CommitChanges
	app.changes.OnConfirmDiscard = app.ConfirmDiscard

	app.contentSplit.OnResize = func(height int) {
		if height <= 0 {
			app.contentSplit.ShowBottom = false
		} else {
			app.contentSplit.ShowBottom = true
			app.contentSplit.BottomH = height
			if len(app.terminals) == 0 {
				app.SpawnTerminal()
			}
		}
	}

	app.contentSplit.OnTopClick = func() {
		app.root.SetFocus(app.editorGroup)
	}

	app.contentSplit.OnBottomClick = func() {
		if w := app.bottomPanel.ActiveWidget(); w != nil {
			app.root.SetFocus(w)
		}
	}

	app.splitPanel.OnResize = func(width int) {
		app.SetSidebarWidth(width)
	}

	app.bottomPanel.TabBar.OnTabClick = func(index int) {
		panels := app.bottomPanel.PanelIDs()
		if index >= 0 && index < len(panels) {
			app.bottomPanel.SetActivePanel(panels[index])
			if w := app.bottomPanel.ActiveWidget(); w != nil {
				app.root.SetFocus(w)
			}
		}
	}

	app.bottomPanel.TabBar.OnAdd = func() {
		reg.Execute("terminal.new")
	}

	app.bottomPanel.TabBar.MoreButton = ui.NewMoreButtonWidget()
	app.bottomPanel.TabBar.MoreButton.OnClick = func(sx, sy int) {
		items := []ui.ContextMenuItem{
			{Label: "New Terminal", Command: "terminal.new"},
			ui.MenuSep(),
			{Label: "Close All Terminals", Command: "terminal.closeAll"},
		}
		openContextMenu(app, items, sx, sy)
	}
}
