package main

import (
	"ttt/internal/command"
	"ttt/internal/config"
	"ttt/internal/core/diff"
	"ttt/internal/git"
	"ttt/internal/render"
	"ttt/internal/term"
	"ttt/internal/ui"
	"ttt/internal/view"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
)

func main() {
	cfg := config.Load()
	config.ParseKeyBindings(cfg.Keybindings)

	screen, err := term.NewTcellScreen()
	if err != nil {
		panic(err)
	}
	defer screen.Fini()

	screen.SetStyleMap(buildStyleMap(cfg.Theme))

	renderer := &render.Renderer{}
	cmdRegistry := command.NewRegistry()
	borders := buildBorderSet(cfg.Theme.Borders)

	// Editor group: tabs + editor pane
	editorGroup := ui.NewEditorGroupWidget(&borders, cfg.Settings.TabSize)
	if len(os.Args) > 1 {
		editorGroup.OpenFile(os.Args[1])
	}

	// Bottom panel: plugin tabs
	bottomPanel := ui.NewBottomPanelWidget(&borders)
	bottomPanel.AddPanel("output", "OUTPUT", ui.NewPlaceholderWidget(""))
	bottomPanel.AddPanel("debug", "DEBUG", ui.NewPlaceholderWidget(""))
	bottomPanel.AddPanel("terminal", "TERMINAL", ui.NewPlaceholderWidget(""))
	bottomPanel.AddPanel("ports", "PORTS", ui.NewPlaceholderWidget(""))

	// Content split: editor on top, bottom panel below
	contentSplit := ui.NewContentSplitWidget()
	contentSplit.Top = editorGroup
	contentSplit.Bottom = bottomPanel
	contentSplit.Borders = &borders
	contentSplit.ShowBottom = false

	status := &view.StatusBar{FileName: editorGroup.ActiveFilePath()}
	statusBar := ui.NewStatusBarWidget(status)

	// Menu bar
	menuBar := ui.NewMenuBarWidget([]ui.MenuItem{
		{Name: "File"},
		{Name: "Edit"},
		{Name: "Selection"},
		{Name: "View"},
		{Name: "Help"},
	})

	// Sidebar
	cwd, _ := os.Getwd()
	explorer := ui.NewExplorerWidget(cwd)
	search := ui.NewSearchWidget()
	changes := ui.NewChangesWidget(cwd)

	sidebar := ui.NewSidebarWidget()
	sidebar.AddPanel("explorer", "Files", explorer)
	sidebar.AddPanel("search", "Search", search)
	sidebar.AddPanel("changes", "Changes", changes)
	sidebar.Visible = cfg.Settings.SidebarVisible
	sidebar.Borders = &borders

	sidebarWidth := cfg.Settings.SidebarWidth
	if sidebarWidth <= 0 {
		sidebarWidth = 30
	}

	// Layout
	splitPanel := ui.NewSplitPanelWidget()
	splitPanel.Left = sidebar
	splitPanel.Right = contentSplit
	splitPanel.Borders = &borders
	splitPanel.DividerPos = sidebarWidth
	splitPanel.ShowLeft = sidebar.Visible
	splitPanel.RightBorderStartY = 2

	rootBox := &ui.VBox{}
	rootBox.AddChild(menuBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})
	rootBox.AddChild(splitPanel, ui.LayoutConstraint{Type: ui.Flex, Value: 1})
	rootBox.AddChild(statusBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})

	root := ui.NewRoot(rootBox)
	root.SetFocus(editorGroup)

	// Sidebar helpers
	showSidebar := func() {
		sidebar.Visible = true
		splitPanel.ShowLeft = true
	}
	hideSidebar := func() {
		sidebar.Visible = false
		splitPanel.ShowLeft = false
	}
	setSidebarWidth := func(w int) {
		if w <= 0 {
			hideSidebar()
			return
		}
		if !sidebar.Visible {
			showSidebar()
		}
		sidebarWidth = w
		splitPanel.DividerPos = sidebarWidth
	}

	splitPanel.OnResize = func(width int) {
		setSidebarWidth(width)
	}

	// Commands
	cmdRegistry.Register(command.Command{
		ID: "sidebar.toggle", Title: "Toggle Sidebar",
		Handler: func() {
			if sidebar.Visible {
				hideSidebar()
			} else {
				showSidebar()
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "sidebar.explorer", Title: "Show Explorer",
		Handler: func() {
			sidebar.SetActivePanel("explorer")
			if !sidebar.Visible {
				showSidebar()
			}
			root.SetFocus(explorer)
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "sidebar.search", Title: "Show Search",
		Handler: func() {
			sidebar.SetActivePanel("search")
			if !sidebar.Visible {
				showSidebar()
			}
			root.SetFocus(search)
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "sidebar.changes", Title: "Show Changes",
		Handler: func() {
			changes.Refresh()
			sidebar.SetActivePanel("changes")
			if !sidebar.Visible {
				showSidebar()
			}
			root.SetFocus(changes)
		},
	})

	changes.OnOpenDiff = func(status git.FileStatus) {
		fullPath := filepath.Join(cwd, status.Path)
		if status.Status == "??" {
			editorGroup.OpenFile(fullPath)
			root.SetFocus(editorGroup)
			return
		}
		diffText, err := git.DiffFile(cwd, status.Path)
		if err != nil || diffText == "" {
			editorGroup.OpenFile(fullPath)
			root.SetFocus(editorGroup)
			return
		}
		parsed := diff.Parse(diffText)
		editorGroup.OpenDiff(status.Path, parsed)
		root.SetFocus(editorGroup)
	}

	cmdRegistry.Register(command.Command{
		ID: "sidebar.wider", Title: "Increase Sidebar Width",
		Handler: func() {
			if sidebar.Visible {
				setSidebarWidth(sidebarWidth + 1)
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "sidebar.narrower", Title: "Decrease Sidebar Width",
		Handler: func() {
			if sidebar.Visible {
				setSidebarWidth(sidebarWidth - 1)
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "sidebar.focus", Title: "Focus Sidebar",
		Handler: func() {
			if !sidebar.Visible {
				showSidebar()
			}
			if w := sidebar.ActiveWidget(); w != nil {
				root.SetFocus(w)
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "editor.focus", Title: "Focus Editor",
		Handler: func() {
			if len(root.Overlays) > 0 {
				root.PopOverlay()
			}
			root.SetFocus(editorGroup)
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "tab.next", Title: "Next Tab",
		Handler: func() { editorGroup.NextTab() },
	})

	cmdRegistry.Register(command.Command{
		ID: "tab.prev", Title: "Previous Tab",
		Handler: func() { editorGroup.PrevTab() },
	})

	cmdRegistry.Register(command.Command{
		ID: "tab.close", Title: "Close Tab",
		Handler: func() { editorGroup.CloseTab() },
	})

	cmdRegistry.Register(command.Command{
		ID: "file.save", Title: "Save File",
		Handler: func() { editorGroup.Save() },
	})

	cmdRegistry.Register(command.Command{
		ID: "editor.undo", Title: "Undo",
		Handler: func() { editorGroup.Undo() },
	})

	cmdRegistry.Register(command.Command{
		ID: "editor.redo", Title: "Redo",
		Handler: func() { editorGroup.Redo() },
	})

	cmdRegistry.Register(command.Command{
		ID: "editor.selectAll", Title: "Select All",
		Handler: func() { editorGroup.SelectAll() },
	})

	cmdRegistry.Register(command.Command{
		ID: "editor.copy", Title: "Copy",
		Handler: func() { editorGroup.Copy() },
	})

	cmdRegistry.Register(command.Command{
		ID: "editor.cut", Title: "Cut",
		Handler: func() { editorGroup.Cut() },
	})

	cmdRegistry.Register(command.Command{
		ID: "editor.paste", Title: "Paste",
		Handler: func() { editorGroup.Paste() },
	})

	quitPending := false
	running := true
	cmdRegistry.Register(command.Command{
		ID: "editor.quit", Title: "Quit",
		Handler: func() {
			if !editorGroup.AnyDirty() || quitPending {
				running = false
				return
			}
			quitPending = true
			status.Message = "Unsaved changes. Press Ctrl+Q again to quit."
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "panel.toggle", Title: "Toggle Panel",
		Handler: func() {
			contentSplit.ShowBottom = !contentSplit.ShowBottom
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "panel.focus", Title: "Focus Panel",
		Handler: func() {
			if !contentSplit.ShowBottom {
				contentSplit.ShowBottom = true
			}
			if w := bottomPanel.ActiveWidget(); w != nil {
				root.SetFocus(w)
			}
		},
	})

	contentSplit.OnResize = func(height int) {
		if height <= 0 {
			contentSplit.ShowBottom = false
		} else {
			contentSplit.ShowBottom = true
			contentSplit.BottomH = height
		}
	}

	cmdRegistry.Register(command.Command{
		ID: "editor.goToLine", Title: "Go to Line",
		Handler: func() {
			dialog := ui.NewGoToLineWidget()
			dialog.Borders = &borders
			dialog.OnSubmit = func(line int) {
				editorGroup.GoToLine(line)
				root.PopOverlay()
				root.SetFocus(editorGroup)
			}
			dialog.OnDismiss = func() {
				root.PopOverlay()
				root.SetFocus(editorGroup)
			}
			root.PushOverlay(ui.Overlay{Widget: dialog, Modal: true})
			root.SetFocus(dialog)
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "search.find", Title: "Find",
		Handler: func() {
			findBar := ui.NewFindBarWidget()
			findBar.Borders = &borders
			findBar.OnSearch = func(query string) []ui.FindMatch {
				editorGroup.SetSearchQuery(query)
				return ui.FindInLines(editorGroup.Editor.Buf.Lines, query)
			}
			findBar.OnNavigate = func(match ui.FindMatch) {
				editorGroup.SetSearchActive(findBar.Current)
				editorGroup.Editor.Cursor.Line = match.Line
				editorGroup.Editor.Cursor.Col = match.Col
			}
			findBar.OnDismiss = func() {
				editorGroup.ClearSearch()
				root.PopOverlay()
				root.SetFocus(editorGroup)
			}
			root.PushOverlay(ui.Overlay{Widget: findBar, Modal: true})
			root.SetFocus(findBar)
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "command.palette", Title: "Command Palette",
		Handler: func() {
			palette := ui.NewCommandPaletteWidget(cmdRegistry.List())
			palette.Borders = &borders
			palette.OnExecute = func(id string) {
				root.PopOverlay()
				root.SetFocus(editorGroup)
				cmdRegistry.Execute(id)
			}
			palette.OnDismiss = func() {
				root.PopOverlay()
				root.SetFocus(editorGroup)
			}
			root.PushOverlay(ui.Overlay{Widget: palette, Modal: true})
		},
	})

	explorer.OnOpenFile = func(path string) {
		editorGroup.OpenFile(path)
		root.SetFocus(editorGroup)
	}

	// Keybindings
	for _, kb := range cfg.Keybindings {
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
				cmdRegistry.Execute(cmdID)
			})
		} else {
			key, mod, rn := comboToTcell(kb.Steps[0])
			root.AddGlobalKey(key, mod, rn, func() {
				cmdRegistry.Execute(cmdID)
			})
		}
	}

	// Initial layout
	w, h := screen.Size()
	root.SetSize(w, h)

	syncStatus := func() {
		line, col := editorGroup.ActiveCursor()
		status.FileName = editorGroup.ActiveFilePath()
		status.Line = line
		status.Col = col
		status.Dirty = editorGroup.IsDirty()
	}

	redraw := func() {
		cells := make([][]term.Cell, root.Height)
		for y := range cells {
			cells[y] = make([]term.Cell, root.Width)
		}
		root.Render(cells)
		renderer.SetCurrent(cells)
		if editorGroup.IsEditorActive() {
			screen.ShowCursor(editorGroup.Editor.CursorX, editorGroup.Editor.CursorY)
		} else {
			screen.HideCursor()
		}
		renderer.Render(screen)
	}

	redraw()

	// Event loop
	for running {
		ev := screen.PollEvent()
		switch tev := ev.(type) {
		case *tcell.EventKey:
			if quitPending && !(tev.Key() == tcell.KeyCtrlQ) {
				quitPending = false
				status.Message = ""
			}
			root.HandleEvent(tev)
			syncStatus()
			redraw()

		case *tcell.EventMouse:
			mx, my := tev.Position()
			btn := tev.Buttons()

			if splitPanel.HandleEvent(tev) == ui.EventConsumed {
				redraw()
				continue
			}

			if contentSplit.HandleEvent(tev) == ui.EventConsumed {
				redraw()
				continue
			}

			if btn&tcell.Button1 != 0 {
				panelRect := splitPanel.GetRect()
				inPanel := my >= panelRect.Y && my < panelRect.Y+panelRect.H &&
					mx >= panelRect.X && mx < panelRect.X+panelRect.W
				if inPanel {
					if sidebar.Visible {
						divX := splitPanel.DividerScreenX()
						if mx < divX {
							sidebar.HandleEvent(tev)
							cmdRegistry.Execute("sidebar.focus")
						} else {
							editorGroup.HandleEvent(tev)
							cmdRegistry.Execute("editor.focus")
						}
					} else {
						editorGroup.HandleEvent(tev)
						cmdRegistry.Execute("editor.focus")
					}
					redraw()
				}
			}

		case *tcell.EventResize:
			w, h := screen.Size()
			root.SetSize(w, h)
			renderer.Clear()
			redraw()
		}
	}
}
