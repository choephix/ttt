package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/core/buffer"
	"github.com/eugenioenko/ttt/internal/core/diff"
	"github.com/eugenioenko/ttt/internal/git"
	"github.com/eugenioenko/ttt/internal/ui"
)

func registerCommands(reg *command.Registry, app *App, running *bool, quitPending *bool) {
	registerViewCommands(reg, app)
	registerEditorCommands(reg, app, running, quitPending)
	registerSearchCommands(reg, app)
	registerPaletteCommands(reg, app)
	registerExplorerCommands(reg, app)
	registerGitCommands(reg, app)
	registerWorkspaceCommands(reg, app)
	registerWidgetCallbacks(reg, app)
}

func registerViewCommands(reg *command.Registry, app *App) {
	reg.Register(command.Command{
		ID: "sidebar.toggle", Title: "Toggle Sidebar",
		Handler: app.ToggleSidebar,
	})

	reg.Register(command.Command{
		ID: "sidebar.explorer", Title: "Show Explorer",
		Handler: func() {
			app.explorer.Reload()
			app.sidebar.SetActivePanel("explorer")
			if !app.sidebar.Visible {
				app.ShowSidebar()
			}
			app.root.SetFocus(app.explorer)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.search", Title: "Show Search",
		Handler: func() {
			app.sidebar.SetActivePanel("search")
			if !app.sidebar.Visible {
				app.ShowSidebar()
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
				app.ShowSidebar()
			}
			app.root.SetFocus(app.changes)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.wider", Title: "Increase Sidebar Width",
		Handler: func() {
			if app.sidebar.Visible {
				app.SetSidebarWidth(app.splitPanel.DividerPos + 1)
			}
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.narrower", Title: "Decrease Sidebar Width",
		Handler: func() {
			if app.sidebar.Visible {
				app.SetSidebarWidth(app.splitPanel.DividerPos - 1)
			}
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.focus", Title: "Focus Sidebar",
		Handler: app.FocusSidebar,
	})

	reg.Register(command.Command{
		ID: "panel.toggle", Title: "Toggle Panel",
		Handler: app.ToggleBottomPanel,
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
		ID: "terminal.new", Title: "New Terminal",
		Handler: app.SpawnTerminal,
	})

	reg.Register(command.Command{
		ID: "terminal.toggle", Title: "Toggle Terminal",
		Handler: func() {
			if !app.contentSplit.ShowBottom {
				r := app.contentSplit.GetRect()
				half := r.H / 2
				if half > r.H-4 {
					half = r.H - 4
				}
				app.contentSplit.BottomH = half
				app.contentSplit.ShowBottom = true
				if len(app.terminals) == 0 {
					app.SpawnTerminal()
				} else {
					app.bottomPanel.SetActivePanel("terminal")
					app.root.SetFocus(app.terminalPanel)
				}
			} else {
				app.HideBottomPanel()
			}
		},
	})

	reg.Register(command.Command{
		ID: "terminal.closeAll", Title: "Close All Terminals",
		Handler: app.CloseAllTerminals,
	})

	reg.Register(command.Command{
		ID: "about", Title: "About ttt",
		Handler: func() {
			url := "https://github.com/eugenioenko/ttt"
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				cmd = exec.Command("open", url)
			case "windows":
				cmd = exec.Command("cmd", "/c", "start", url)
			default:
				cmd = exec.Command("xdg-open", url)
			}
			if err := cmd.Start(); err != nil {
				app.StatusNotify("ttt — Terminal Text Tool")
			}
		},
	})
}

func registerEditorCommands(reg *command.Registry, app *App, running *bool, quitPending *bool) {
	reg.Register(command.Command{
		ID: "editor.focus", Title: "Focus Editor",
		Handler: func() {
			if app.IsAutocompleteActive() {
				app.DismissAutocomplete()
				return
			}
			app.FocusEditor()
		},
	})

	reg.Register(command.Command{
		ID: "editor.autocomplete", Title: "Trigger Autocomplete",
		Handler: func() {
			if !app.settings.Autocomplete.Enabled {
				return
			}
			if app.IsAutocompleteActive() {
				app.DismissAutocomplete()
				return
			}
			path := app.editorGroup.ActiveFilePath()
			lang := ""
			if app.editorGroup.Editor != nil && app.editorGroup.Editor.Highlighter != nil {
				lang = app.editorGroup.Editor.Highlighter.Language()
			}
			if lang == "" {
				app.StatusWarn("No language detected for this file")
			} else if app.lspManager == nil || !app.lspManager.HasServer(strings.ToLower(lang)) {
				app.StatusWarn(lang + " language server is not configured. Add it to settings.json under lsp.servers")
			} else {
				line, col := app.editorGroup.ActiveCursor()
				app.RequestCompletions(path, lang, line, col)
			}
		},
	})

	lspAction := func(action func(path, lang string, line, col int)) {
		path := app.editorGroup.ActiveFilePath()
		lang := ""
		if app.editorGroup.Editor != nil && app.editorGroup.Editor.Highlighter != nil {
			lang = app.editorGroup.Editor.Highlighter.Language()
		}
		line, col := app.editorGroup.ActiveCursor()
		action(path, lang, line, col)
	}

	reg.Register(command.Command{
		ID: "editor.hover", Title: "Show Hover",
		Handler: func() { lspAction(app.RequestHover) },
	})

	reg.Register(command.Command{
		ID: "editor.goToDefinition", Title: "Go to Definition",
		Handler: func() { lspAction(app.RequestDefinition) },
	})

	reg.Register(command.Command{
		ID: "editor.goToImplementation", Title: "Go to Implementation",
		Handler: func() { lspAction(app.RequestImplementation) },
	})

	reg.Register(command.Command{
		ID: "editor.goToTypeDefinition", Title: "Go to Type Definition",
		Handler: func() { lspAction(app.RequestTypeDefinition) },
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
		Handler: func() {
			if !app.editorGroup.IsDirty() {
				app.editorGroup.CloseTab()
				return
			}
			name := app.editorGroup.ActiveFileName()
			dialog := ui.NewConfirmDialogWidget3(
				"Save changes to "+name+"?",
				"Discard", "Cancel", "Save",
			)
			dialog.Borders = app.borders
			dialog.OnButton[0] = func() {
				app.DismissDialog()
				app.editorGroup.CloseTab()
			}
			dialog.OnButton[1] = func() {
				app.DismissDialog()
			}
			dialog.OnButton[2] = func() {
				app.DismissDialog()
				reg.Execute("file.save")
				app.editorGroup.CloseTab()
			}
			dialog.OnDismiss = func() {
				app.DismissDialog()
			}
			app.ShowDialog(dialog)
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
		ID: "file.new", Title: "New File",
		Handler: func() {
			app.editorGroup.OpenBuffer("untitled", &buffer.Buffer{Lines: []string{""}})
			app.root.SetFocus(app.editorGroup)
		},
	})

	saveAs := func() {
		current := app.editorGroup.ActiveFilePath()
		initial := ""
		if current != "untitled" {
			initial = current
		}
		app.ShowInputDialog("Save As", initial, func(path string) {
			if path != "" {
				app.editorGroup.SaveAs(path)
			}
		})
	}

	reg.Register(command.Command{
		ID: "file.save", Title: "Save File",
		Handler: func() {
			if !app.editorGroup.Save() {
				saveAs()
				return
			}
			path := app.editorGroup.ActiveFilePath()
			if app.editorGroup.Editor != nil && app.editorGroup.Editor.Highlighter != nil {
				lang := app.editorGroup.Editor.Highlighter.Language()
				text := strings.Join(app.editorGroup.Editor.Buf.Lines, "\n")
				app.NotifyLSPSave(path, lang, text)
			}
		},
	})

	reg.Register(command.Command{
		ID: "file.saveAs", Title: "Save As...",
		Handler: saveAs,
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
				for _, tt := range app.terminals {
					tt.term.Close()
				}
				*running = false
				return
			}
			*quitPending = true
			app.StatusWarn("Unsaved changes. Press Ctrl+Q again to quit.")
		},
	})
}

func registerSearchCommands(reg *command.Registry, app *App) {
	reg.Register(command.Command{
		ID: "search.find", Title: "Find",
		Handler: func() {
			findBar := ui.NewFindBarWidget()
			findBar.Borders = app.borders
			findBar.OnSearch = func(query string) []ui.FindMatch {
				matches := ui.FindInLines(app.editorGroup.Editor.Buf.Lines, query)
				app.editorGroup.SetSearch(query, matches)
				return matches
			}
			findBar.OnNavigate = func(match ui.FindMatch) {
				app.editorGroup.SetSearchActive(findBar.Current)
				app.editorGroup.Editor.Cursor.Line = match.Line
				app.editorGroup.Editor.Cursor.Col = match.Col
			}
			findBar.OnDismiss = func() {
				app.DismissDialog()
				app.editorGroup.ClearSearch()
			}
			app.ShowDialog(findBar)
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
				matches := ui.FindInLines(app.editorGroup.Editor.Buf.Lines, query)
				app.editorGroup.SetSearch(query, matches)
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
				app.DismissDialog()
				app.editorGroup.ClearSearch()
			}
			app.ShowDialog(bar)
		},
	})

	reg.Register(command.Command{
		ID: "search.clear", Title: "Clear Search Results",
		Handler: func() {
			app.search.Input.Text = ""
			app.search.Input.CursorPos = 0
			app.search.Groups = nil
			app.search.FlatList = nil
			app.search.Selected = 0
			app.search.ScrollTop = 0
		},
	})
}

func registerPaletteCommands(reg *command.Registry, app *App) {
	openPalette := func(fileMode bool, initialText ...string) {
		palette := ui.NewCommandPaletteWidget(reg.List())
		palette.Borders = app.borders
		palette.SetFiles(app.workspace.Paths())
		if len(initialText) > 0 {
			palette.Input.SetText(initialText[0])
		} else if fileMode {
			palette.Input.SetText("")
		}
		palette.OnExecute = func(id string) {
			app.DismissDialog()
			reg.Execute(id)
		}
		palette.OnOpenFile = func(absPath string) {
			app.DismissDialog()
			app.editorGroup.OpenFile(absPath)
		}
		palette.OnGoToLine = func(line int) {
			app.DismissDialog()
			app.editorGroup.GoToLine(line)
		}
		palette.OnDismiss = func() {
			app.DismissDialog()
		}
		app.ShowDialog(palette)
	}

	reg.Register(command.Command{
		ID: "command.palette", Title: "Command Palette",
		Handler: func() { openPalette(false) },
	})

	reg.Register(command.Command{
		ID: "file.quickOpen", Title: "Go to File",
		Handler: func() { openPalette(true) },
	})

	reg.Register(command.Command{
		ID: "editor.goToLine", Title: "Go to Line",
		Handler: func() { openPalette(false, ":") },
	})

	reg.Register(command.Command{
		ID: "theme.switch", Title: "Switch Theme",
		Handler: func() {
			names := config.ListThemes()
			if len(names) == 0 {
				return
			}
			var cmds []command.Command
			for _, name := range names {
				cmds = append(cmds, command.Command{ID: name, Title: name})
			}
			picker := ui.NewCommandPaletteWidget(cmds)
			picker.Borders = app.borders
			originalStyleMap := app.screen.GetStyleMap()
			originalPalette := *app.palette
			applyTheme := func(theme config.ThemeConfig) {
				app.screen.SetStyleMap(buildStyleMap(theme))
				*app.palette = buildTerminalPalette(theme)
				app.renderer.Clear()
			}
			picker.OnSelectionChange = func(name string) {
				theme, err := config.LoadTheme(name)
				if err != nil {
					return
				}
				applyTheme(theme)
			}
			picker.OnExecute = func(name string) {
				app.DismissDialog()
				theme, err := config.LoadTheme(name)
				if err != nil {
					return
				}
				applyTheme(theme)
				app.settings.Theme = name
				config.SaveSettings(*app.settings)
			}
			picker.OnDismiss = func() {
				app.DismissDialog()
				app.screen.SetStyleMap(originalStyleMap)
				*app.palette = originalPalette
				app.renderer.Clear()
			}
			app.ShowDialog(picker)
		},
	})

	openIndentPicker := func() {
		var cmds []command.Command
		sizes := []int{1, 2, 3, 4, 6, 8}
		for _, s := range sizes {
			label := fmt.Sprintf("Spaces: %d", s)
			cmds = append(cmds, command.Command{ID: strconv.Itoa(s), Title: label})
		}
		cmds = append(cmds, command.Command{ID: "tabs", Title: "Indent Using Tabs"})
		cmds = append(cmds, command.Command{ID: "detect", Title: "Detect from Content"})
		app.ShowPicker(cmds, func(id string) {
			if id == "detect" {
				if app.editorGroup.Editor != nil && app.editorGroup.Editor.Buf != nil {
					if info := buffer.DetectIndent(app.editorGroup.Editor.Buf.Lines); info.Size > 0 {
						app.editorGroup.SetTabSize(info.Size)
					}
				}
			} else if id == "tabs" {
				app.editorGroup.SetTabSize(4)
			} else if size, err := strconv.Atoi(id); err == nil {
				app.editorGroup.SetTabSize(size)
			}
		})
	}

	app.statusBar.OnIndentClick = openIndentPicker

	reg.Register(command.Command{
		ID: "editor.indentation", Title: "Change Indentation",
		Handler: openIndentPicker,
	})

	reg.Register(command.Command{
		ID: "settings.open", Title: "Preferences: Open Settings",
		Handler: func() {
			path := config.ConfigFilePath("settings.json")
			app.editorGroup.OpenFile(path)
		},
	})

	reg.Register(command.Command{
		ID: "keybindings.open", Title: "Preferences: Open Keyboard Shortcuts",
		Handler: func() {
			path := config.ConfigFilePath("keybindings.json")
			app.editorGroup.OpenFile(path)
		},
	})
}

func registerExplorerCommands(reg *command.Registry, app *App) {
	reg.Register(command.Command{
		ID: "explorer.refresh", Title: "Refresh Explorer",
		Handler: func() { app.explorer.Reload() },
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
			node := app.explorer.SelectedNode()
			if node == nil {
				return
			}
			parentDir := node.Path
			if !node.IsDir {
				parentDir = filepath.Dir(node.Path)
			}
			app.ShowInputDialog("New File", "", func(name string) {
				newPath := filepath.Join(parentDir, name)
				if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
					app.StatusError("Error: " + err.Error())
					return
				}
				if err := os.WriteFile(newPath, []byte{}, 0644); err != nil {
					app.StatusError("Error: " + err.Error())
					return
				}
				app.explorer.Reload()
				app.editorGroup.OpenFile(newPath)
				app.FocusEditor()
			})
		},
	})

	reg.Register(command.Command{
		ID: "explorer.newFolder", Title: "New Folder",
		Handler: func() {
			node := app.explorer.SelectedNode()
			if node == nil {
				return
			}
			parentDir := node.Path
			if !node.IsDir {
				parentDir = filepath.Dir(node.Path)
			}
			app.ShowInputDialog("New Folder", "", func(name string) {
				newPath := filepath.Join(parentDir, name)
				if err := os.MkdirAll(newPath, 0755); err != nil {
					app.StatusError("Error: " + err.Error())
					return
				}
				app.explorer.Reload()
			})
		},
	})

	reg.Register(command.Command{
		ID: "explorer.rename", Title: "Rename",
		Handler: func() {
			node := app.explorer.SelectedNode()
			if node == nil {
				return
			}
			app.ShowInputDialog("Rename", node.Name, func(newName string) {
				dir := filepath.Dir(node.Path)
				newPath := filepath.Join(dir, newName)
				if err := os.Rename(node.Path, newPath); err != nil {
					app.StatusError("Error: " + err.Error())
					return
				}
				app.explorer.Reload()
			})
		},
	})

	reg.Register(command.Command{
		ID: "explorer.delete", Title: "Delete",
		Handler: func() {
			node := app.explorer.SelectedNode()
			if node == nil {
				return
			}
			app.ShowConfirmDialog("Delete "+node.Name+"?",
				[]string{"Yes", "No"},
				[]func(){
					func() {
						app.DismissDialog()
						if err := os.RemoveAll(node.Path); err != nil {
							app.StatusError("Error: " + err.Error())
							return
						}
						app.explorer.Reload()
					},
					func() { app.DismissDialog() },
				},
			)
		},
	})
}

func registerGitCommands(reg *command.Registry, app *App) {
	reg.Register(command.Command{
		ID: "changes.openDiff", Title: "Open Diff",
		Handler: func() {
			dir, status, ok := app.changes.SelectedFile()
			if ok && app.changes.OnOpenDiff != nil {
				app.changes.OnOpenDiff(dir, status)
			}
		},
	})

	reg.Register(command.Command{
		ID: "changes.openFile", Title: "Open File",
		Handler: func() {
			fullPath := app.changes.SelectedFullPath()
			if fullPath != "" {
				app.editorGroup.OpenFile(fullPath)
				app.root.SetFocus(app.editorGroup)
			}
		},
	})

	reg.Register(command.Command{
		ID: "changes.refresh", Title: "Refresh Changes",
		Handler: func() { app.changes.Refresh() },
	})

	registerGitCmd := func(id, title string, ops []func(string) error, verb string) {
		reg.Register(command.Command{
			ID: id, Title: title,
			Handler: func() {
				for _, dir := range app.changes.Dirs {
					for _, op := range ops {
						if err := op(dir); err != nil {
							app.StatusError(fmt.Sprintf("%s failed: %v", verb, err))
							return
						}
					}
				}
				app.StatusNotify(verb + " successfully")
				app.changes.Refresh()
			},
		})
	}
	registerGitCmd("git.pull", "Git Pull", []func(string) error{git.Pull}, "Pulled")
	registerGitCmd("git.push", "Git Push", []func(string) error{git.Push}, "Pushed")
	registerGitCmd("git.sync", "Git Sync", []func(string) error{git.Pull, git.Push}, "Synced")
}

func registerWorkspaceCommands(reg *command.Registry, app *App) {
	reg.Register(command.Command{
		ID: "workspace.addFolder", Title: "Add Folder to Workspace",
		Handler: func() {
			app.ShowInputDialog("Add Folder", "", func(path string) {
				if path == "" {
					return
				}
				abs, err := filepath.Abs(path)
				if err != nil {
					app.StatusError("Error: " + err.Error())
					return
				}
				info, err := os.Stat(abs)
				if err != nil || !info.IsDir() {
					app.StatusError("Not a directory: " + abs)
					return
				}
				app.workspace.AddFolder(abs)
				app.refreshWorkspaceWidgets()
			})
		},
	})

	reg.Register(command.Command{
		ID: "workspace.removeFolder", Title: "Remove Folder from Workspace",
		Handler: func() {
			paths := app.workspace.Paths()
			if len(paths) <= 1 {
				app.StatusWarn("Cannot remove the last folder")
				return
			}
			var cmds []command.Command
			for _, p := range paths {
				cmds = append(cmds, command.Command{ID: p, Title: filepath.Base(p)})
			}
			app.ShowPicker(cmds, func(path string) {
				app.workspace.RemoveFolder(path)
				app.refreshWorkspaceWidgets()
			})
		},
	})

	reg.Register(command.Command{
		ID: "workspace.saveAs", Title: "Save Workspace As...",
		Handler: func() {
			app.ShowInputDialog("Save Workspace", "workspace.ttt", func(path string) {
				if path == "" {
					return
				}
				if err := app.workspace.SaveFile(path); err != nil {
					app.StatusError("Error: " + err.Error())
				} else {
					app.StatusNotify("Workspace saved: " + path)
				}
			})
		},
	})
}

func registerWidgetCallbacks(reg *command.Registry, app *App) {
	for i := range menuBarMenus {
		idx := i
		reg.Register(command.Command{
			ID:    menuBarLabels[idx],
			Title: "Open " + menuBarLabels[idx] + " Menu",
			Handler: func() {
				openMenuBarDropdown(app, reg, idx)
			},
		})
	}

	app.menuBar.OnSelect = func(index int) {
		openMenuBarDropdown(app, reg, index)
	}

	app.root.OnRightClick = func(mx, my int) {
		handleRightClick(app, reg, mx, my)
	}

	app.splitPanel.OnLeftClick = func() {
		reg.Execute("sidebar.focus")
	}
	app.splitPanel.OnRightClick = func() {}


	app.sidebar.MoreButton.OnClick = func(sx, sy int) {
		var items []ui.ContextMenuItem
		switch app.sidebar.ActivePanel {
		case "explorer":
			items = []ui.ContextMenuItem{
				{Label: "New File", Command: "file.new"},
				{Label: "Refresh", Command: "explorer.refresh"},
			}
		case "search":
			items = []ui.ContextMenuItem{
				{Label: "Clear Results", Command: "search.clear"},
			}
		case "changes":
			items = []ui.ContextMenuItem{
				{Label: "Refresh", Command: "changes.refresh"},
				ui.MenuSep(),
				{Label: "Pull", Command: "git.pull"},
				{Label: "Push", Command: "git.push"},
				{Label: "Sync", Command: "git.sync"},
			}
		}
		if len(items) > 0 {
			openContextMenu(app, reg, items, sx, sy)
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
		openContextMenu(app, reg, items, sx, sy)
	}

	app.editorGroup.TabBar.OnTabClose = func(index int) {
		app.editorGroup.SwitchTab(index)
		reg.Execute("tab.close")
	}

	app.editorGroup.TabBar.MoreButton.OnClick = func(sx, sy int) {
		moreMenu := []ui.ContextMenuItem{
			{Label: "Close All", Command: "tab.closeAll"},
		}
		openContextMenu(app, reg, moreMenu, sx, sy)
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

	app.explorer.OnOpenFile = func(path string) {
		app.editorGroup.OpenFile(path)
		app.root.SetFocus(app.editorGroup)
	}

	app.search.OnOpenMatch = func(path string, line, col int) {
		app.editorGroup.OpenFile(path)
		app.editorGroup.GoToLine(line)
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

	app.changes.OnRightClick = func(dir string, status git.FileStatus, sx, sy int) {
		openContextMenu(app, reg, changesContextMenu, sx, sy)
	}

	app.changes.OnOpenDiff = func(dir string, status git.FileStatus) {
		fullPath := filepath.Join(dir, status.Path)
		if status.Status == "?" {
			app.editorGroup.OpenFile(fullPath)
			app.root.SetFocus(app.editorGroup)
			return
		}
		diffText, err := git.DiffFile(dir, status.Path)
		if err != nil || diffText == "" {
			app.editorGroup.OpenFile(fullPath)
			app.root.SetFocus(app.editorGroup)
			return
		}
		parsed := diff.Parse(diffText)
		app.editorGroup.OpenDiff(status.Path, parsed)
		app.root.SetFocus(app.editorGroup)
	}

	app.changes.OnGroupMenu = func(dir string, sx, sy int) {
		items := []ui.ContextMenuItem{
			{Label: "Pull", Command: "git.pull." + dir},
			{Label: "Push", Command: "git.push." + dir},
			{Label: "Sync", Command: "git.sync." + dir},
		}
		registerDirGitCmd := func(id, title string, ops []func(string) error, verb string) {
			reg.Register(command.Command{
				ID: id, Title: title,
				Handler: func() {
					for _, op := range ops {
						if err := op(dir); err != nil {
							app.StatusError(fmt.Sprintf("%s failed: %v", verb, err))
							return
						}
					}
					app.StatusNotify(verb + " successfully")
					app.changes.Refresh()
				},
			})
		}
		registerDirGitCmd("git.pull."+dir, "Pull", []func(string) error{git.Pull}, "Pulled")
		registerDirGitCmd("git.push."+dir, "Push", []func(string) error{git.Push}, "Pushed")
		registerDirGitCmd("git.sync."+dir, "Sync", []func(string) error{git.Pull, git.Push}, "Synced")
		openContextMenu(app, reg, items, sx, sy)
	}

	app.changes.OnCommit = func(dir string, message string) {
		if err := git.Commit(dir, message); err != nil {
			app.StatusError("Commit failed: " + err.Error())
		} else {
			app.StatusNotify("Committed: " + message)
			app.changes.Refresh()
		}
	}

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
		openContextMenu(app, reg, items, sx, sy)
	}
}

// Commands that work even when terminal has raw key focus.
var forceKeyCommands = map[string]bool{
	"panel.toggle":    true,
	"terminal.toggle": true,
	"quit":            true,
}

func bindKeys(root *ui.Root, reg *command.Registry, keybindings []config.KeyBinding) {
	for _, kb := range keybindings {
		if len(kb.Steps) == 0 {
			continue
		}
		cmdID := kb.Command
		reg.SetShortcut(cmdID, formatKeyBinding(kb.Key))
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
			handler := func() { reg.Execute(cmdID) }
			root.AddGlobalKey(key, mod, rn, handler)
			if forceKeyCommands[cmdID] {
				root.AddForceKey(key, mod, rn, handler)
			}
		}
	}
}

func formatKeyBinding(key string) string {
	parts := strings.Fields(key)
	for i, part := range parts {
		tokens := strings.Split(part, "+")
		for j, t := range tokens {
			if t == "backtick" {
				tokens[j] = "`"
			} else if len(t) > 0 {
				tokens[j] = strings.ToUpper(t[:1]) + t[1:]
			}
		}
		parts[i] = strings.Join(tokens, "+")
	}
	return strings.Join(parts, " ")
}
