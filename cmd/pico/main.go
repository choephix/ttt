package main

import (
	"macro/internal/command"
	"macro/internal/config"
	"macro/internal/core/buffer"
	"macro/internal/core/cursor"
	"macro/internal/render"
	"macro/internal/term"
	"macro/internal/ui"
	"macro/internal/view"
	"os"

	"github.com/gdamore/tcell/v2"
)

func main() {
	cfg := config.Load()
	config.ParseKeyBindings(cfg.Keybindings)

	buf := &buffer.Buffer{Lines: []string{""}}
	if len(os.Args) > 1 {
		if err := buf.LoadFile(os.Args[1]); err != nil {
			buf = &buffer.Buffer{Lines: []string{"Error: " + err.Error()}}
		}
	}

	cur := &cursor.Cursor{Line: 0, Col: 0}
	vp := &view.Viewport{}
	status := &view.StatusBar{FileName: "untitled", Dirty: false}
	if len(os.Args) > 1 {
		status.FileName = os.Args[1]
	}

	screen, err := term.NewTcellScreen()
	if err != nil {
		panic(err)
	}
	defer screen.Fini()

	screen.SetStyleMap(buildStyleMap(cfg.Theme))

	renderer := &render.Renderer{}
	cmdRegistry := command.NewRegistry()

	editorPane := ui.NewEditorPaneWidget(buf, cur, vp)
	editorPane.TabSize = cfg.Settings.TabSize
	statusBar := ui.NewStatusBarWidget(status)
	tabBar := ui.NewTabBarWidget()
	tabBar.SetTabs([]ui.Tab{{Name: status.FileName, Active: true, Dirty: false}})

	activityBar := ui.NewActivityBarWidget()

	cwd, _ := os.Getwd()
	explorer := ui.NewExplorerWidget(cwd)
	search := ui.NewSearchWidget()

	sidebar := ui.NewSidebarWidget()
	sidebar.AddPanel("explorer", explorer)
	sidebar.AddPanel("search", search)
	sidebar.Visible = cfg.Settings.SidebarVisible
	activityBar.ActiveID = "explorer"

	editorArea := &ui.VBox{}
	editorArea.AddChild(tabBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})
	editorArea.AddChild(editorPane, ui.LayoutConstraint{Type: ui.Flex, Value: 1})

	resizeHandle := ui.NewResizeHandleWidget()

	mainArea := &ui.HBox{}
	mainArea.AddChild(activityBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 2})
	sidebarIdx := 1
	handleIdx := 2
	sidebarWidth := cfg.Settings.SidebarWidth
	if sidebarWidth <= 0 {
		sidebarWidth = 30
	}
	const minSidebarWidth = 10
	const maxSidebarWidth = 80
	const resizeStep = 2
	if sidebar.Visible {
		mainArea.AddChild(sidebar, ui.LayoutConstraint{Type: ui.Fixed, Value: sidebarWidth})
		mainArea.AddChild(resizeHandle, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})
	} else {
		mainArea.AddChild(sidebar, ui.LayoutConstraint{Type: ui.Hidden})
		mainArea.AddChild(resizeHandle, ui.LayoutConstraint{Type: ui.Hidden})
	}
	mainArea.AddChild(editorArea, ui.LayoutConstraint{Type: ui.Flex, Value: 1})

	setSidebarWidth := func(w int) {
		if w < minSidebarWidth {
			w = minSidebarWidth
		}
		if w > maxSidebarWidth {
			w = maxSidebarWidth
		}
		sidebarWidth = w
		mainArea.SetChildConstraint(sidebarIdx, ui.LayoutConstraint{Type: ui.Fixed, Value: sidebarWidth})
	}

	rootBox := &ui.VBox{}
	rootBox.AddChild(mainArea, ui.LayoutConstraint{Type: ui.Flex, Value: 1})
	rootBox.AddChild(statusBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})

	root := ui.NewRoot(rootBox)
	root.SetFocus(editorPane)

	// Commands
	showSidebar := func() {
		sidebar.Visible = true
		mainArea.SetChildConstraint(sidebarIdx, ui.LayoutConstraint{Type: ui.Fixed, Value: sidebarWidth})
		mainArea.SetChildConstraint(handleIdx, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})
	}
	hideSidebar := func() {
		sidebar.Visible = false
		mainArea.SetChildConstraint(sidebarIdx, ui.LayoutConstraint{Type: ui.Hidden})
		mainArea.SetChildConstraint(handleIdx, ui.LayoutConstraint{Type: ui.Hidden})
	}

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
			activityBar.SetActiveByID("explorer")
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
			activityBar.SetActiveByID("search")
			if !sidebar.Visible {
				showSidebar()
			}
			root.SetFocus(search)
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "sidebar.wider", Title: "Increase Sidebar Width",
		Handler: func() {
			if sidebar.Visible {
				setSidebarWidth(sidebarWidth + resizeStep)
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "sidebar.narrower", Title: "Decrease Sidebar Width",
		Handler: func() {
			if sidebar.Visible {
				setSidebarWidth(sidebarWidth - resizeStep)
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "sidebar.focus", Title: "Focus Sidebar",
		Handler: func() {
			if !sidebar.Visible {
				showSidebar()
			}
			active := sidebar.ActiveWidget()
			if active != nil {
				root.SetFocus(active)
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "editor.focus", Title: "Focus Editor",
		Handler: func() {
			if len(root.Overlays) > 0 {
				root.PopOverlay()
			}
			root.SetFocus(editorPane)
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "file.save", Title: "Save File",
		Handler: func() {
			if len(os.Args) > 1 {
				buf.SaveFile(os.Args[1])
				status.Dirty = false
				tabBar.SetTabs([]ui.Tab{{Name: status.FileName, Active: true, Dirty: false}})
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "command.palette", Title: "Command Palette",
		Handler: func() {
			palette := ui.NewCommandPaletteWidget(cmdRegistry.List())
			palette.OnExecute = func(id string) {
				root.PopOverlay()
				root.SetFocus(editorPane)
				cmdRegistry.Execute(id)
			}
			palette.OnDismiss = func() {
				root.PopOverlay()
				root.SetFocus(editorPane)
			}
			root.PushOverlay(ui.Overlay{Widget: palette, Modal: true})
		},
	})

	explorer.OnOpenFile = func(path string) {
		if err := buf.LoadFile(path); err != nil {
			return
		}
		cur.Line = 0
		cur.Col = 0
		vp.TopLine = 0
		vp.LeftCol = 0
		status.FileName = path
		status.Dirty = false
		tabBar.SetTabs([]ui.Tab{{Name: path, Active: true}})
		root.SetFocus(editorPane)
	}

	activityBar.OnSelect = func(id string) {
		switch id {
		case "explorer":
			cmdRegistry.Execute("sidebar.explorer")
		case "search":
			cmdRegistry.Execute("sidebar.search")
		}
	}

	// Apply keybindings from config
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

	w, h := screen.Size()
	root.SetSize(w, h)

	redraw := func() {
		cells := make([][]term.Cell, root.Height)
		for y := range cells {
			cells[y] = make([]term.Cell, root.Width)
		}
		root.Render(cells)
		renderer.SetCurrent(cells)
		screen.ShowCursor(editorPane.CursorX, editorPane.CursorY)
		renderer.Render(screen)
	}

	redraw()

	draggingSidebar := false
	activityBarWidth := 2

	for {
		ev := screen.PollEvent()
		switch tev := ev.(type) {
		case *tcell.EventKey:
			if tev.Key() == tcell.KeyCtrlC {
				return
			}
			root.HandleEvent(tev)

			status.Line = cur.Line
			status.Col = cur.Col
			status.Dirty = buf.Dirty
			tabBar.SetTabs([]ui.Tab{{Name: status.FileName, Active: true, Dirty: buf.Dirty}})
			redraw()

		case *tcell.EventMouse:
			mx, my := tev.Position()
			btn := tev.Buttons()

			if draggingSidebar {
				if btn&tcell.Button1 != 0 {
					newWidth := mx - activityBarWidth
					setSidebarWidth(newWidth)
					redraw()
				} else {
					draggingSidebar = false
				}
			} else if btn&tcell.Button1 != 0 {
				_, screenH := screen.Size()
				statusRow := screenH - 1

				if my < statusRow {
					if sidebar.Visible {
						handleX := activityBarWidth + sidebarWidth
						if mx == handleX {
							draggingSidebar = true
						} else if mx >= activityBarWidth && mx < handleX {
							cmdRegistry.Execute("sidebar.focus")
							redraw()
						} else if mx > handleX {
							cmdRegistry.Execute("editor.focus")
							redraw()
						}
					} else if mx >= activityBarWidth {
						cmdRegistry.Execute("editor.focus")
						redraw()
					}
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

var canonicalKeyToTcell = map[string]tcell.Key{
	"Escape":    tcell.KeyEscape,
	"Enter":     tcell.KeyEnter,
	"Tab":       tcell.KeyTab,
	"Backspace": tcell.KeyBackspace,
	"Delete":    tcell.KeyDelete,
	"Insert":    tcell.KeyInsert,
	"Up":        tcell.KeyUp,
	"Down":      tcell.KeyDown,
	"Left":      tcell.KeyLeft,
	"Right":     tcell.KeyRight,
	"Home":      tcell.KeyHome,
	"End":       tcell.KeyEnd,
	"PgUp":      tcell.KeyPgUp,
	"PgDn":      tcell.KeyPgDn,
	"Space":     tcell.KeyRune,
	"F1":        tcell.KeyF1,
	"F2":        tcell.KeyF2,
	"F3":        tcell.KeyF3,
	"F4":        tcell.KeyF4,
	"F5":        tcell.KeyF5,
	"F6":        tcell.KeyF6,
	"F7":        tcell.KeyF7,
	"F8":        tcell.KeyF8,
	"F9":        tcell.KeyF9,
	"F10":       tcell.KeyF10,
	"F11":       tcell.KeyF11,
	"F12":       tcell.KeyF12,
}

func comboToTcell(combo config.KeyCombo) (tcell.Key, tcell.ModMask, rune) {
	var mod tcell.ModMask
	if combo.Ctrl {
		mod |= tcell.ModCtrl
	}
	if combo.Alt {
		mod |= tcell.ModAlt
	}
	if combo.Shift {
		mod |= tcell.ModShift
	}

	if combo.KeyName != "" {
		if combo.KeyName == "Space" {
			return tcell.KeyRune, mod, ' '
		}
		key := canonicalKeyToTcell[combo.KeyName]
		return key, mod, 0
	}

	// Ctrl+letter → tcell.KeyCtrl<Letter>
	if combo.Ctrl && combo.Rune >= 'a' && combo.Rune <= 'z' {
		key := tcell.KeyCtrlA + tcell.Key(combo.Rune-'a')
		return key, mod, 0
	}

	return tcell.KeyRune, mod, combo.Rune
}

func buildStyleMap(theme config.ThemeConfig) term.StyleMap {
	m := term.DefaultStyleMap()
	applyStyleDef(&m, term.StyleStatusBar, theme.StatusBar)
	applyStyleDef(&m, term.StyleActiveTab, theme.ActiveTab)
	applyStyleDef(&m, term.StyleInactiveTab, theme.InactiveTab)
	applyStyleDef(&m, term.StyleActivityBar, theme.ActivityBar)
	applyStyleDef(&m, term.StyleActivityBarActive, theme.ActivityBarActive)
	applyStyleDef(&m, term.StyleSidebarHeader, theme.SidebarHeader)
	applyStyleDef(&m, term.StyleSidebarItem, theme.SidebarItem)
	applyStyleDef(&m, term.StyleSidebarSelected, theme.SidebarSelected)
	applyStyleDef(&m, term.StylePaletteBorder, theme.PaletteBorder)
	applyStyleDef(&m, term.StylePaletteInput, theme.PaletteInput)
	applyStyleDef(&m, term.StylePaletteItem, theme.PaletteItem)
	applyStyleDef(&m, term.StylePaletteSelected, theme.PaletteSelected)
	applyStyleDef(&m, term.StyleLineNumber, theme.LineNumber)
	applyStyleDef(&m, term.StyleResizeHandle, theme.ResizeHandle)
	return m
}

func applyStyleDef(m *term.StyleMap, idx term.Style, def config.StyleDef) {
	base := tcell.StyleDefault
	if def.Fg != "" {
		base = base.Foreground(tcell.GetColor(def.Fg))
	}
	if def.Bg != "" {
		base = base.Background(tcell.GetColor(def.Bg))
	}
	if def.Bold {
		base = base.Bold(true)
	}
	m[idx] = base
}
