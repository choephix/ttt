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

type tabState struct {
	FilePath string
	Buf      *buffer.Buffer
	Cur      *cursor.Cursor
	Vp       *view.Viewport
}

func main() {
	cfg := config.Load()
	config.ParseKeyBindings(cfg.Keybindings)

	initialBuf := &buffer.Buffer{Lines: []string{""}}
	initialPath := "untitled"
	if len(os.Args) > 1 {
		if err := initialBuf.LoadFile(os.Args[1]); err != nil {
			initialBuf = &buffer.Buffer{Lines: []string{"Error: " + err.Error()}}
		}
		initialPath = os.Args[1]
	}

	tabs := []tabState{{
		FilePath: initialPath,
		Buf:      initialBuf,
		Cur:      &cursor.Cursor{},
		Vp:       &view.Viewport{},
	}}
	activeTab := 0

	status := &view.StatusBar{FileName: initialPath, Dirty: false}

	screen, err := term.NewTcellScreen()
	if err != nil {
		panic(err)
	}
	defer screen.Fini()

	screen.SetStyleMap(buildStyleMap(cfg.Theme))

	renderer := &render.Renderer{}
	cmdRegistry := command.NewRegistry()
	borders := buildBorderSet(cfg.Theme.Borders)

	editorPane := ui.NewEditorPaneWidget(tabs[0].Buf, tabs[0].Cur, tabs[0].Vp)
	editorPane.TabSize = cfg.Settings.TabSize
	statusBar := ui.NewStatusBarWidget(status)

	tabBar := ui.NewTabBarWidget()
	tabBar.Borders = &borders

	syncTabs := func() {
		t := tabs[activeTab]
		editorPane.Buf = t.Buf
		editorPane.Cursor = t.Cur
		editorPane.Viewport = t.Vp
		status.FileName = t.FilePath
		status.Dirty = t.Buf.Dirty
		var uiTabs []ui.Tab
		for i, ts := range tabs {
			uiTabs = append(uiTabs, ui.Tab{
				Name:   ts.FilePath,
				Active: i == activeTab,
				Dirty:  ts.Buf.Dirty,
			})
		}
		tabBar.SetTabs(uiTabs)
	}

	switchTab := func(idx int) {
		if idx >= 0 && idx < len(tabs) {
			activeTab = idx
			syncTabs()
		}
	}

	openFile := func(path string) {
		for i, t := range tabs {
			if t.FilePath == path {
				switchTab(i)
				return
			}
		}
		newBuf := &buffer.Buffer{Lines: []string{""}}
		if err := newBuf.LoadFile(path); err != nil {
			return
		}
		tabs = append(tabs, tabState{
			FilePath: path,
			Buf:      newBuf,
			Cur:      &cursor.Cursor{},
			Vp:       &view.Viewport{},
		})
		switchTab(len(tabs) - 1)
	}

	syncTabs()

	menuBar := ui.NewMenuBarWidget([]ui.MenuItem{
		{Name: "File"},
		{Name: "Edit"},
		{Name: "Selection"},
		{Name: "View"},
		{Name: "Help"},
	})

	cwd, _ := os.Getwd()
	explorer := ui.NewExplorerWidget(cwd)
	search := ui.NewSearchWidget()

	sidebar := ui.NewSidebarWidget()
	sidebar.AddPanel("explorer", explorer)
	sidebar.AddPanel("search", search)
	sidebar.Visible = cfg.Settings.SidebarVisible
	sidebar.Title = "EXPLORER"
	sidebar.Borders = &borders

	sidebarWidth := cfg.Settings.SidebarWidth
	if sidebarWidth <= 0 {
		sidebarWidth = 30
	}
	const resizeStep = 1

	editorArea := &ui.VBox{}
	editorArea.AddChild(tabBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 3})
	editorArea.AddChild(editorPane, ui.LayoutConstraint{Type: ui.Flex, Value: 1})

	splitPanel := ui.NewSplitPanelWidget()
	splitPanel.Left = sidebar
	splitPanel.Right = editorArea
	splitPanel.Borders = &borders
	splitPanel.DividerPos = sidebarWidth
	splitPanel.ShowLeft = sidebar.Visible
	splitPanel.LeftTitle = ""
	splitPanel.RightTitle = ""
	splitPanel.RightBorderStartY = 2

	var setSidebarWidth func(int)

	rootBox := &ui.VBox{}
	rootBox.AddChild(menuBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})
	rootBox.AddChild(splitPanel, ui.LayoutConstraint{Type: ui.Flex, Value: 1})
	rootBox.AddChild(statusBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})

	root := ui.NewRoot(rootBox)
	root.SetFocus(editorPane)

	// Commands
	showSidebar := func() {
		sidebar.Visible = true
		splitPanel.ShowLeft = true
	}
	hideSidebar := func() {
		sidebar.Visible = false
		splitPanel.ShowLeft = false
	}

	setSidebarWidth = func(w int) {
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
			sidebar.Title = "EXPLORER"
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
			sidebar.Title = "SEARCH"
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
		ID: "tab.next", Title: "Next Tab",
		Handler: func() {
			if len(tabs) > 1 {
				switchTab((activeTab + 1) % len(tabs))
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "tab.prev", Title: "Previous Tab",
		Handler: func() {
			if len(tabs) > 1 {
				switchTab((activeTab - 1 + len(tabs)) % len(tabs))
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "tab.close", Title: "Close Tab",
		Handler: func() {
			if len(tabs) <= 1 {
				return
			}
			tabs = append(tabs[:activeTab], tabs[activeTab+1:]...)
			if activeTab >= len(tabs) {
				activeTab = len(tabs) - 1
			}
			syncTabs()
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "file.save", Title: "Save File",
		Handler: func() {
			t := &tabs[activeTab]
			if t.FilePath != "untitled" {
				t.Buf.SaveFile(t.FilePath)
				syncTabs()
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "command.palette", Title: "Command Palette",
		Handler: func() {
			palette := ui.NewCommandPaletteWidget(cmdRegistry.List())
			palette.Borders = &borders
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
		openFile(path)
		root.SetFocus(editorPane)
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

	for {
		ev := screen.PollEvent()
		switch tev := ev.(type) {
		case *tcell.EventKey:
			if tev.Key() == tcell.KeyCtrlC {
				return
			}
			root.HandleEvent(tev)

			t := tabs[activeTab]
			status.Line = t.Cur.Line
			status.Col = t.Cur.Col
			status.FileName = t.FilePath
			status.Dirty = t.Buf.Dirty
			syncTabs()
			redraw()

		case *tcell.EventMouse:
			mx, my := tev.Position()
			btn := tev.Buttons()

			panelRect := splitPanel.GetRect()
			inPanel := my >= panelRect.Y && my < panelRect.Y+panelRect.H &&
				mx >= panelRect.X && mx < panelRect.X+panelRect.W

			if draggingSidebar {
				if btn&tcell.Button1 != 0 {
					newWidth := mx - panelRect.X - 1
					setSidebarWidth(newWidth)
					redraw()
				} else {
					draggingSidebar = false
				}
			} else if btn&tcell.Button1 != 0 && inPanel {
				if sidebar.Visible {
					divX := splitPanel.DividerScreenX()
					if mx == divX {
						draggingSidebar = true
					} else if mx < divX {
						cmdRegistry.Execute("sidebar.focus")
						active := sidebar.ActiveWidget()
						if active != nil {
							active.HandleEvent(tev)
						}
						redraw()
					} else {
						cmdRegistry.Execute("editor.focus")
						redraw()
					}
				} else {
					cmdRegistry.Execute("editor.focus")
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
	applyStyleDef(&m, term.StyleMenuBar, theme.MenuBar)
	applyStyleDef(&m, term.StyleMenuBarActive, theme.MenuBarActive)
	applyStyleDef(&m, term.StyleBorder, theme.Border)
	return m
}

func firstRune(s string, fallback rune) rune {
	for _, r := range s {
		return r
	}
	return fallback
}

func buildBorderSet(bc config.BorderChars) term.BorderSet {
	d := term.SingleBorderSet()
	return term.BorderSet{
		Horizontal:  firstRune(bc.Horizontal, d.Horizontal),
		Vertical:    firstRune(bc.Vertical, d.Vertical),
		TopLeft:     firstRune(bc.TopLeft, d.TopLeft),
		TopRight:    firstRune(bc.TopRight, d.TopRight),
		BottomLeft:  firstRune(bc.BottomLeft, d.BottomLeft),
		BottomRight: firstRune(bc.BottomRight, d.BottomRight),
		TopTee:      firstRune(bc.TopTee, d.TopTee),
		BottomTee:   firstRune(bc.BottomTee, d.BottomTee),
		LeftTee:     firstRune(bc.LeftTee, d.LeftTee),
		RightTee:    firstRune(bc.RightTee, d.RightTee),
	}
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
