package main

import (
	"macro/internal/command"
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

	renderer := &render.Renderer{}
	cmdRegistry := command.NewRegistry()

	// Build widgets
	editorPane := ui.NewEditorPaneWidget(buf, cur, vp)
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
	activityBar.ActiveID = "explorer"

	// Editor area: tab bar + editor pane
	editorArea := &ui.VBox{}
	editorArea.AddChild(tabBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})
	editorArea.AddChild(editorPane, ui.LayoutConstraint{Type: ui.Flex, Value: 1})

	// Main area: activity bar + sidebar + editor area
	mainArea := &ui.HBox{}
	mainArea.AddChild(activityBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 2})
	sidebarIdx := 1
	mainArea.AddChild(sidebar, ui.LayoutConstraint{Type: ui.Fixed, Value: 30})
	mainArea.AddChild(editorArea, ui.LayoutConstraint{Type: ui.Flex, Value: 1})

	// Root layout: main area + status bar
	rootBox := &ui.VBox{}
	rootBox.AddChild(mainArea, ui.LayoutConstraint{Type: ui.Flex, Value: 1})
	rootBox.AddChild(statusBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})

	root := ui.NewRoot(rootBox)
	root.SetFocus(editorPane)

	// Commands
	cmdRegistry.Register(command.Command{
		ID: "sidebar.toggle", Title: "Toggle Sidebar",
		Handler: func() {
			sidebar.Visible = !sidebar.Visible
			if sidebar.Visible {
				mainArea.SetChildConstraint(sidebarIdx, ui.LayoutConstraint{Type: ui.Fixed, Value: 30})
			} else {
				mainArea.SetChildConstraint(sidebarIdx, ui.LayoutConstraint{Type: ui.Hidden})
			}
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "sidebar.explorer", Title: "Show Explorer",
		Handler: func() {
			sidebar.SetActivePanel("explorer")
			activityBar.SetActiveByID("explorer")
			if !sidebar.Visible {
				sidebar.Visible = true
				mainArea.SetChildConstraint(sidebarIdx, ui.LayoutConstraint{Type: ui.Fixed, Value: 30})
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
				sidebar.Visible = true
				mainArea.SetChildConstraint(sidebarIdx, ui.LayoutConstraint{Type: ui.Fixed, Value: 30})
			}
			root.SetFocus(search)
		},
	})

	cmdRegistry.Register(command.Command{
		ID: "editor.focus", Title: "Focus Editor",
		Handler: func() {
			root.SetFocus(editorPane)
		},
	})

	// File open from explorer
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

	// Activity bar callback
	activityBar.OnSelect = func(id string) {
		switch id {
		case "explorer":
			cmdRegistry.Execute("sidebar.explorer")
		case "search":
			cmdRegistry.Execute("sidebar.search")
		}
	}

	// Command palette
	openPalette := func() {
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
	}

	// Global keybindings
	// Ctrl+B -> toggle sidebar
	root.AddGlobalKey(tcell.KeyCtrlB, tcell.ModCtrl, 0, func() {
		cmdRegistry.Execute("sidebar.toggle")
	})

	// Ctrl+E -> explorer
	root.AddGlobalKey(tcell.KeyCtrlE, tcell.ModCtrl, 0, func() {
		cmdRegistry.Execute("sidebar.explorer")
	})

	// Ctrl+F -> search (project-wide; in-file find will use Ctrl+/ later)
	root.AddGlobalKey(tcell.KeyCtrlF, tcell.ModCtrl, 0, func() {
		cmdRegistry.Execute("sidebar.search")
	})

	// Ctrl+P -> command palette
	root.AddGlobalKey(tcell.KeyCtrlP, tcell.ModCtrl, 0, func() {
		openPalette()
	})

	// Ctrl+S -> save
	root.AddGlobalKey(tcell.KeyCtrlS, tcell.ModCtrl, 0, func() {
		if len(os.Args) > 1 {
			buf.SaveFile(os.Args[1])
			status.Dirty = false
			tabBar.SetTabs([]ui.Tab{{Name: status.FileName, Active: true, Dirty: false}})
		}
	})

	// Escape -> focus editor (or dismiss overlay)
	root.AddGlobalKey(tcell.KeyEscape, 0, 0, func() {
		if len(root.Overlays) > 0 {
			root.PopOverlay()
		}
		root.SetFocus(editorPane)
	})

	// Initial size
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

		case *tcell.EventResize:
			w, h := screen.Size()
			root.SetSize(w, h)
			renderer.Clear()
			redraw()
		}
	}
}
