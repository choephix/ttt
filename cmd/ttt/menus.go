package main

import (
	"ttt/internal/command"
	"ttt/internal/ui"

	"github.com/gdamore/tcell/v2"
)

var menuBarLabels = []string{
	"menu.file", "menu.edit", "menu.selection", "menu.view", "menu.help",
}

var menuBarMenus = [][]ui.ContextMenuItem{
	// File
	{
		{Label: "New File", Shortcut: "", Command: "file.new"},
		ui.MenuSep(),
		{Label: "Save", Shortcut: "Ctrl+S", Command: "file.save"},
		{Label: "Save As...", Shortcut: "", Command: "file.saveAs"},
		ui.MenuSep(),
		{Label: "Add Folder to Workspace", Shortcut: "", Command: "workspace.addFolder"},
		{Label: "Save Workspace As...", Shortcut: "", Command: "workspace.saveAs"},
		ui.MenuSep(),
		{Label: "Quit", Shortcut: "Ctrl+Q", Command: "editor.quit"},
	},
	// Edit
	{
		{Label: "Undo", Shortcut: "Ctrl+Z", Command: "editor.undo"},
		{Label: "Redo", Shortcut: "Ctrl+Y", Command: "editor.redo"},
		ui.MenuSep(),
		{Label: "Cut", Shortcut: "Ctrl+X", Command: "editor.cut"},
		{Label: "Copy", Shortcut: "Ctrl+C", Command: "editor.copy"},
		{Label: "Paste", Shortcut: "Ctrl+V", Command: "editor.paste"},
		ui.MenuSep(),
		{Label: "Find", Shortcut: "Ctrl+F", Command: "search.find"},
		{Label: "Replace", Shortcut: "Ctrl+H", Command: "search.replace"},
	},
	// Selection
	{
		{Label: "Select All", Shortcut: "Ctrl+A", Command: "editor.selectAll"},
	},
	// View
	{
		{Label: "Command Palette", Shortcut: "Ctrl+P", Command: "command.palette"},
		ui.MenuSep(),
		{Label: "Explorer", Shortcut: "Ctrl+E", Command: "sidebar.explorer"},
		{Label: "Search", Shortcut: "Ctrl+Shift+F", Command: "sidebar.search"},
		{Label: "Changes", Shortcut: "Ctrl+D", Command: "sidebar.changes"},
		ui.MenuSep(),
		{Label: "Toggle Sidebar", Shortcut: "Ctrl+B", Command: "sidebar.toggle"},
		{Label: "Toggle Terminal", Shortcut: "Ctrl+`", Command: "terminal.toggle"},
		{Label: "New Terminal", Shortcut: "", Command: "terminal.new"},
		ui.MenuSep(),
		{Label: "Toggle Panel", Shortcut: "", Command: "panel.toggle"},
		ui.MenuSep(),
		{Label: "Switch Theme", Shortcut: "Ctrl+K T", Command: "theme.switch"},
	},
	// Help
	{
		{Label: "About", Shortcut: "", Command: "about"},
	},
}

var editorContextMenu = []ui.ContextMenuItem{
	{Label: "Undo", Shortcut: "Ctrl+Z", Command: "editor.undo"},
	{Label: "Redo", Shortcut: "Ctrl+Y", Command: "editor.redo"},
	ui.MenuSep(),
	{Label: "Cut", Shortcut: "Ctrl+X", Command: "editor.cut"},
	{Label: "Copy", Shortcut: "Ctrl+C", Command: "editor.copy"},
	{Label: "Paste", Shortcut: "Ctrl+V", Command: "editor.paste"},
	ui.MenuSep(),
	{Label: "Select All", Shortcut: "Ctrl+A", Command: "editor.selectAll"},
	ui.MenuSep(),
	{Label: "Find", Shortcut: "Ctrl+F", Command: "search.find"},
	{Label: "Replace", Shortcut: "Ctrl+H", Command: "search.replace"},
	{Label: "Go to Line", Shortcut: "Ctrl+G", Command: "editor.goToLine"},
}

var changesContextMenu = []ui.ContextMenuItem{
	{Label: "Open Diff", Shortcut: "", Command: "changes.openDiff"},
	{Label: "Open File", Shortcut: "", Command: "changes.openFile"},
}

func openContextMenu(app *App, reg *command.Registry, items []ui.ContextMenuItem, x, y int) {
	menu := ui.NewContextMenuWidget(items, x, y)
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
	menu := ui.NewContextMenuWidget(menuBarMenus[index], anchorX, 1)
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
