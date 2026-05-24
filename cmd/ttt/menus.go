package main

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/ui"

	"github.com/gdamore/tcell/v2"
)

var menuBarLabels = []string{
	"menu.file", "menu.edit", "menu.selection", "menu.view", "menu.help",
}

var menuBarMenus = [][]ui.ContextMenuItem{
	// File
	{
		{Label: "New File", Command: "file.new"},
		ui.MenuSep(),
		{Label: "Save", Command: "file.save"},
		{Label: "Save As...", Command: "file.saveAs"},
		ui.MenuSep(),
		{Label: "Add Folder to Workspace", Command: "workspace.addFolder"},
		{Label: "Save Workspace As...", Command: "workspace.saveAs"},
		ui.MenuSep(),
		{Label: "Quit", Command: "editor.quit"},
	},
	// Edit
	{
		{Label: "Undo", Command: "editor.undo"},
		{Label: "Redo", Command: "editor.redo"},
		ui.MenuSep(),
		{Label: "Cut", Command: "editor.cut"},
		{Label: "Copy", Command: "editor.copy"},
		{Label: "Paste", Command: "editor.paste"},
		ui.MenuSep(),
		{Label: "Find", Command: "search.find"},
		{Label: "Replace", Command: "search.replace"},
	},
	// Selection
	{
		{Label: "Select All", Command: "editor.selectAll"},
	},
	// View
	{
		{Label: "Command Palette", Command: "command.palette"},
		ui.MenuSep(),
		{Label: "Explorer", Command: "sidebar.explorer"},
		{Label: "Search", Command: "sidebar.search"},
		{Label: "Changes", Command: "sidebar.changes"},
		ui.MenuSep(),
		{Label: "Toggle Sidebar", Command: "sidebar.toggle"},
		{Label: "Toggle Terminal", Command: "terminal.toggle"},
		{Label: "New Terminal", Command: "terminal.new"},
		ui.MenuSep(),
		{Label: "Toggle Panel", Command: "panel.toggle"},
		ui.MenuSep(),
		{Label: "Switch Theme", Command: "theme.switch"},
	},
	// Help
	{
		{Label: "About", Command: "about"},
	},
}

var editorContextMenu = []ui.ContextMenuItem{
	{Label: "Undo", Command: "editor.undo"},
	{Label: "Redo", Command: "editor.redo"},
	ui.MenuSep(),
	{Label: "Cut", Command: "editor.cut"},
	{Label: "Copy", Command: "editor.copy"},
	{Label: "Paste", Command: "editor.paste"},
	ui.MenuSep(),
	{Label: "Select All", Command: "editor.selectAll"},
	ui.MenuSep(),
	{Label: "Find", Command: "search.find"},
	{Label: "Replace", Command: "search.replace"},
	{Label: "Go to Line", Command: "editor.goToLine"},
}

var changesContextMenu = []ui.ContextMenuItem{
	{Label: "Open Diff", Command: "changes.openDiff"},
	{Label: "Open File", Command: "changes.openFile"},
}

func resolveShortcuts(reg *command.Registry, items []ui.ContextMenuItem) []ui.ContextMenuItem {
	resolved := make([]ui.ContextMenuItem, len(items))
	for i, item := range items {
		resolved[i] = item
		if item.Command != "" {
			if cmd, ok := reg.Get(item.Command); ok && cmd.Shortcut != "" {
				resolved[i].Shortcut = cmd.Shortcut
			}
		}
	}
	return resolved
}

func openContextMenu(app *App, reg *command.Registry, items []ui.ContextMenuItem, x, y int) {
	menu := ui.NewContextMenuWidget(resolveShortcuts(reg, items), x, y)
	menu.Borders = app.borders
	menu.OnExec = func(cmd string) {
		app.root.PopOverlay()
		reg.Execute(cmd)
	}
	menu.OnDismiss = func() {
		app.root.PopOverlay()
	}
	app.root.PushOverlay(ui.Overlay{Widget: menu, Modal: true})
	app.root.SetFocus(menu)
}

func openMenuBarDropdown(app *App, reg *command.Registry, index int) {
	if index < 0 || index >= len(menuBarMenus) {
		return
	}
	app.menuBar.Selected = index
	anchorX := app.menuBar.ItemAnchorX(index)
	menu := ui.NewContextMenuWidget(resolveShortcuts(reg, menuBarMenus[index]), anchorX, 1)
	menu.Borders = app.borders
	menu.OnExec = func(cmd string) {
		app.root.PopOverlay()
		app.menuBar.Selected = -1
		reg.Execute(cmd)
	}
	menu.OnDismiss = func() {
		app.root.PopOverlay()
		app.menuBar.Selected = -1
	}
	menu.OnNavigate = func(dir int) {
		app.root.PopOverlay()
		next := (index + dir + len(menuBarMenus)) % len(menuBarMenus)
		openMenuBarDropdown(app, reg, next)
	}
	app.root.PushOverlay(ui.Overlay{Widget: menu, Modal: true})
	app.root.SetFocus(menu)
}

func handleRightClick(app *App, reg *command.Registry, mx, my int) {
	panelRect := app.splitPanel.GetRect()
	if my < panelRect.Y || my >= panelRect.Y+panelRect.H {
		return
	}

	if app.sidebar.Visible {
		divX := app.splitPanel.DividerScreenX()
		if mx < divX {
			sidebarR := app.sidebar.GetRect()
			if my > sidebarR.Y+1 {
				ev := tcell.NewEventMouse(mx, my, tcell.Button2, 0)
				if w := app.sidebar.ActiveWidget(); w != nil {
					w.HandleEvent(ev)
				}
			}
			return
		}
	}

	tabR := app.editorGroup.TabBar.GetRect()
	if my >= tabR.Y && my < tabR.Y+tabR.H {
		ev := tcell.NewEventMouse(mx, my, tcell.Button2, 0)
		app.editorGroup.TabBar.HandleEvent(ev)
		return
	}

	openContextMenu(app, reg, editorContextMenu, mx, my)
}
