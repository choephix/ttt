package main

import (
	"path/filepath"
	"strings"
	"ttt/internal/command"
	"ttt/internal/config"
	"ttt/internal/core/buffer"
	"ttt/internal/core/diff"
	"ttt/internal/git"
	"ttt/internal/term"
	"ttt/internal/ui"
	"ttt/internal/view"
)

type appWidgets struct {
	root         *ui.Root
	editorGroup  *ui.EditorGroupWidget
	sidebar      *ui.SidebarWidget
	splitPanel   *ui.SplitPanelWidget
	contentSplit *ui.ContentSplitWidget
	bottomPanel  *ui.BottomPanelWidget
	explorer     *ui.ExplorerWidget
	search       *ui.SearchWidget
	changes      *ui.ChangesWidget
	menuBar      *ui.MenuBarWidget
	statusBar    *ui.StatusBarWidget
	status       *view.StatusBar
	borders      *term.BorderSet
	screen       *term.TcellScreen
	renderer     interface{ Clear() }
	cwd          string

	showSidebar    func()
	hideSidebar    func()
	setSidebarWidth func(int)
}

func registerCommands(reg *command.Registry, app *appWidgets, running *bool, quitPending *bool) {
	reg.Register(command.Command{
		ID: "sidebar.toggle", Title: "Toggle Sidebar",
		Handler: func() {
			if app.sidebar.Visible {
				app.hideSidebar()
			} else {
				app.showSidebar()
			}
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.explorer", Title: "Show Explorer",
		Handler: func() {
			app.sidebar.SetActivePanel("explorer")
			if !app.sidebar.Visible {
				app.showSidebar()
			}
			app.root.SetFocus(app.explorer)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.search", Title: "Show Search",
		Handler: func() {
			app.sidebar.SetActivePanel("search")
			if !app.sidebar.Visible {
				app.showSidebar()
			}
			app.root.SetFocus(app.search)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.changes", Title: "Show Changes",
		Handler: func() {
			app.changes.Refresh()
			app.sidebar.SetActivePanel("changes")
			if !app.sidebar.Visible {
				app.showSidebar()
			}
			app.root.SetFocus(app.changes)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.wider", Title: "Increase Sidebar Width",
		Handler: func() {
			if app.sidebar.Visible {
				app.setSidebarWidth(app.splitPanel.DividerPos + 1)
			}
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.narrower", Title: "Decrease Sidebar Width",
		Handler: func() {
			if app.sidebar.Visible {
				app.setSidebarWidth(app.splitPanel.DividerPos - 1)
			}
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.focus", Title: "Focus Sidebar",
		Handler: func() {
			if !app.sidebar.Visible {
				app.showSidebar()
			}
			if w := app.sidebar.ActiveWidget(); w != nil {
				app.root.SetFocus(w)
			}
		},
	})

	reg.Register(command.Command{
		ID: "editor.focus", Title: "Focus Editor",
		Handler: func() {
			if len(app.root.Overlays) > 0 {
				app.root.PopOverlay()
			}
			app.root.SetFocus(app.editorGroup)
		},
	})

	reg.Register(command.Command{
		ID: "tab.next", Title: "Next Tab",
		Handler: func() { app.editorGroup.NextTab() },
	})

	reg.Register(command.Command{
		ID: "tab.prev", Title: "Previous Tab",
		Handler: func() { app.editorGroup.PrevTab() },
	})

	reg.Register(command.Command{
		ID: "tab.close", Title: "Close Tab",
		Handler: func() { app.editorGroup.CloseTab() },
	})

	reg.Register(command.Command{
		ID: "file.new", Title: "New File",
		Handler: func() {
			app.editorGroup.OpenBuffer("untitled", &buffer.Buffer{Lines: []string{""}})
			app.root.SetFocus(app.editorGroup)
		},
	})

	reg.Register(command.Command{
		ID: "file.save", Title: "Save File",
		Handler: func() { app.editorGroup.Save() },
	})

	reg.Register(command.Command{
		ID: "editor.undo", Title: "Undo",
		Handler: func() { app.editorGroup.Undo() },
	})

	reg.Register(command.Command{
		ID: "editor.redo", Title: "Redo",
		Handler: func() { app.editorGroup.Redo() },
	})

	reg.Register(command.Command{
		ID: "editor.selectAll", Title: "Select All",
		Handler: func() { app.editorGroup.SelectAll() },
	})

	reg.Register(command.Command{
		ID: "editor.copy", Title: "Copy",
		Handler: func() { app.editorGroup.Copy() },
	})

	reg.Register(command.Command{
		ID: "editor.cut", Title: "Cut",
		Handler: func() { app.editorGroup.Cut() },
	})

	reg.Register(command.Command{
		ID: "editor.paste", Title: "Paste",
		Handler: func() { app.editorGroup.Paste() },
	})

	reg.Register(command.Command{
		ID: "editor.quit", Title: "Quit",
		Handler: func() {
			if !app.editorGroup.AnyDirty() || *quitPending {
				*running = false
				return
			}
			*quitPending = true
			app.status.Message = "Unsaved changes. Press Ctrl+Q again to quit."
		},
	})

	reg.Register(command.Command{
		ID: "panel.toggle", Title: "Toggle Panel",
		Handler: func() {
			app.contentSplit.ShowBottom = !app.contentSplit.ShowBottom
		},
	})

	reg.Register(command.Command{
		ID: "panel.focus", Title: "Focus Panel",
		Handler: func() {
			if !app.contentSplit.ShowBottom {
				app.contentSplit.ShowBottom = true
			}
			if w := app.bottomPanel.ActiveWidget(); w != nil {
				app.root.SetFocus(w)
			}
		},
	})

	reg.Register(command.Command{
		ID: "editor.goToLine", Title: "Go to Line",
		Handler: func() {
			dialog := ui.NewGoToLineWidget()
			dialog.Borders = app.borders
			dialog.OnSubmit = func(line int) {
				app.editorGroup.GoToLine(line)
				app.root.PopOverlay()
				app.root.SetFocus(app.editorGroup)
			}
			dialog.OnDismiss = func() {
				app.root.PopOverlay()
				app.root.SetFocus(app.editorGroup)
			}
			app.root.PushOverlay(ui.Overlay{Widget: dialog, Modal: true})
			app.root.SetFocus(dialog)
		},
	})

	reg.Register(command.Command{
		ID: "search.find", Title: "Find",
		Handler: func() {
			findBar := ui.NewFindBarWidget()
			findBar.Borders = app.borders
			findBar.OnSearch = func(query string) []ui.FindMatch {
				app.editorGroup.SetSearchQuery(query)
				matches := ui.FindInLines(app.editorGroup.Editor.Buf.Lines, query)
				app.editorGroup.StoreSearchMatches(query, matches)
				return matches
			}
			findBar.OnNavigate = func(match ui.FindMatch) {
				app.editorGroup.SetSearchActive(findBar.Current)
				app.editorGroup.Editor.Cursor.Line = match.Line
				app.editorGroup.Editor.Cursor.Col = match.Col
			}
			findBar.OnDismiss = func() {
				app.root.PopOverlay()
				app.root.SetFocus(app.editorGroup)
			}
			app.root.PushOverlay(ui.Overlay{Widget: findBar, Modal: true})
			app.root.SetFocus(findBar)
		},
	})

	reg.Register(command.Command{
		ID: "search.findNext", Title: "Find Next",
		Handler: func() { app.editorGroup.FindNext() },
	})

	reg.Register(command.Command{
		ID: "search.findPrev", Title: "Find Previous",
		Handler: func() { app.editorGroup.FindPrev() },
	})

	reg.Register(command.Command{
		ID: "search.clearFind", Title: "Clear Find Highlights",
		Handler: func() { app.editorGroup.ClearSearch() },
	})

	reg.Register(command.Command{
		ID: "search.replace", Title: "Find and Replace",
		Handler: func() {
			bar := ui.NewReplaceBarWidget()
			bar.Borders = app.borders
			bar.OnSearch = func(query string) []ui.FindMatch {
				app.editorGroup.SetSearchQuery(query)
				matches := ui.FindInLines(app.editorGroup.Editor.Buf.Lines, query)
				app.editorGroup.StoreSearchMatches(query, matches)
				return matches
			}
			bar.OnNavigate = func(match ui.FindMatch) {
				app.editorGroup.SetSearchActive(bar.Current)
				app.editorGroup.Editor.Cursor.Line = match.Line
				app.editorGroup.Editor.Cursor.Col = match.Col
			}
			bar.OnReplace = func(match ui.FindMatch, replacement string) {
				app.editorGroup.ReplaceMatch(match, replacement)
			}
			bar.OnReplaceAll = func(query, replacement string) {
				app.editorGroup.ReplaceAll(query, replacement)
			}
			bar.OnDismiss = func() {
				app.root.PopOverlay()
				app.root.SetFocus(app.editorGroup)
			}
			app.root.PushOverlay(ui.Overlay{Widget: bar, Modal: true})
			app.root.SetFocus(bar)
		},
	})

	reg.Register(command.Command{
		ID: "command.palette", Title: "Command Palette",
		Handler: func() {
			palette := ui.NewCommandPaletteWidget(reg.List())
			palette.Borders = app.borders
			palette.OnExecute = func(id string) {
				app.root.PopOverlay()
				app.root.SetFocus(app.editorGroup)
				reg.Execute(id)
			}
			palette.OnDismiss = func() {
				app.root.PopOverlay()
				app.root.SetFocus(app.editorGroup)
			}
			app.root.PushOverlay(ui.Overlay{Widget: palette, Modal: true})
		},
	})

	reg.Register(command.Command{
		ID: "theme.switch", Title: "Switch Theme",
		Handler: func() {
			files := config.ListThemeFiles()
			if len(files) == 0 {
				return
			}
			var cmds []command.Command
			for _, f := range files {
				name := filepath.Base(f)
				name = strings.TrimPrefix(name, "theme.")
				name = strings.TrimSuffix(name, ".json")
				cmds = append(cmds, command.Command{ID: f, Title: name})
			}
			picker := ui.NewCommandPaletteWidget(cmds)
			picker.Borders = app.borders
			picker.OnExecute = func(path string) {
				app.root.PopOverlay()
				app.root.SetFocus(app.editorGroup)
				theme, err := config.LoadThemeFromFile(path)
				if err != nil {
					return
				}
				app.screen.SetStyleMap(buildStyleMap(theme))
				app.renderer.Clear()
			}
			picker.OnDismiss = func() {
				app.root.PopOverlay()
				app.root.SetFocus(app.editorGroup)
			}
			app.root.PushOverlay(ui.Overlay{Widget: picker, Modal: true})
		},
	})

	reg.Register(command.Command{
		ID: "changes.openDiff", Title: "Open Diff",
		Handler: func() {
			if app.changes.Selected >= 0 && app.changes.Selected < len(app.changes.Files) {
				status := app.changes.Files[app.changes.Selected]
				app.changes.OnOpenDiff(status)
			}
		},
	})

	reg.Register(command.Command{
		ID: "changes.openFile", Title: "Open File",
		Handler: func() {
			if app.changes.Selected >= 0 && app.changes.Selected < len(app.changes.Files) {
				status := app.changes.Files[app.changes.Selected]
				fullPath := filepath.Join(app.cwd, status.Path)
				app.editorGroup.OpenFile(fullPath)
				app.root.SetFocus(app.editorGroup)
			}
		},
	})

	reg.Register(command.Command{
		ID: "tab.closeOthers", Title: "Close Other Tabs",
		Handler: func() { app.editorGroup.CloseOtherTabs() },
	})

	reg.Register(command.Command{
		ID: "tab.closeAll", Title: "Close All Tabs",
		Handler: func() { app.editorGroup.CloseAllTabs() },
	})

	reg.Register(command.Command{
		ID: "explorer.open", Title: "Open",
		Handler: func() {
			app.explorer.ActivateSelected()
		},
	})

	reg.Register(command.Command{
		ID: "explorer.newFile", Title: "New File",
		Handler: func() {
			app.status.Message = "New File: not yet implemented"
		},
	})

	reg.Register(command.Command{
		ID: "explorer.newFolder", Title: "New Folder",
		Handler: func() {
			app.status.Message = "New Folder: not yet implemented"
		},
	})

	reg.Register(command.Command{
		ID: "explorer.rename", Title: "Rename",
		Handler: func() {
			app.status.Message = "Rename: not yet implemented"
		},
	})

	reg.Register(command.Command{
		ID: "explorer.delete", Title: "Delete",
		Handler: func() {
			app.status.Message = "Delete: not yet implemented"
		},
	})

	reg.Register(command.Command{
		ID: "about", Title: "About ttt",
		Handler: func() {
			app.status.Message = "ttt — Terminal Text Tool"
		},
	})

	app.menuBar.OnSelect = func(index int) {
		openMenuBarDropdown(app, reg, index)
	}

	app.editorGroup.TabBar.OnTabRightClick = func(index, sx, sy int) {
		app.editorGroup.SwitchTab(index)
		tabContextMenu := []ui.ContextMenuItem{
			{Label: "Close", Shortcut: "Ctrl+W", Command: "tab.close"},
			{Label: "Close Others", Shortcut: "", Command: "tab.closeOthers"},
			{Label: "Close All", Shortcut: "", Command: "tab.closeAll"},
		}
		openContextMenu(app, reg, tabContextMenu, sx, sy)
	}

	// Widget callbacks
	app.explorer.OnOpenFile = func(path string) {
		app.editorGroup.OpenFile(path)
		app.root.SetFocus(app.editorGroup)
	}

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
		openContextMenu(app, reg, items, sx, sy)
	}

	app.changes.OnRightClick = func(status git.FileStatus, sx, sy int) {
		openContextMenu(app, reg, changesContextMenu, sx, sy)
	}

	app.changes.OnOpenDiff = func(status git.FileStatus) {
		fullPath := filepath.Join(app.cwd, status.Path)
		if status.Status == "??" {
			app.editorGroup.OpenFile(fullPath)
			app.root.SetFocus(app.editorGroup)
			return
		}
		diffText, err := git.DiffFile(app.cwd, status.Path)
		if err != nil || diffText == "" {
			app.editorGroup.OpenFile(fullPath)
			app.root.SetFocus(app.editorGroup)
			return
		}
		parsed := diff.Parse(diffText)
		app.editorGroup.OpenDiff(status.Path, parsed)
		app.root.SetFocus(app.editorGroup)
	}

	app.contentSplit.OnResize = func(height int) {
		if height <= 0 {
			app.contentSplit.ShowBottom = false
		} else {
			app.contentSplit.ShowBottom = true
			app.contentSplit.BottomH = height
		}
	}

	app.splitPanel.OnResize = func(width int) {
		app.setSidebarWidth(width)
	}
}

func bindKeys(root *ui.Root, reg *command.Registry, keybindings []config.KeyBinding) {
	for _, kb := range keybindings {
		if len(kb.Steps) == 0 {
			continue
		}
		cmdID := kb.Command
		if kb.IsChord() {
			steps := make([]ui.GlobalKeyBinding, len(kb.Steps))
			for i, step := range kb.Steps {
				key, mod, rn := comboToTcell(step)
				steps[i] = ui.GlobalKeyBinding{Key: key, Mod: mod, Rune: rn}
			}
			root.AddChordKey(steps, func() {
				reg.Execute(cmdID)
			})
		} else {
			key, mod, rn := comboToTcell(kb.Steps[0])
			root.AddGlobalKey(key, mod, rn, func() {
				reg.Execute(cmdID)
			})
		}
	}
}
