package main

import (
	"fmt"
	"log/slog"
	"ttt/internal/config"
	"ttt/internal/render"
	"ttt/internal/term"
	"ttt/internal/terminal"
	"ttt/internal/ui"
	"ttt/internal/view"

	"github.com/gdamore/tcell/v2"
)

type terminalTab struct {
	term   *terminal.Terminal
	widget *ui.TerminalWidget
}

type App struct {
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
	renderer     *render.Renderer
	settings     *config.Settings
	cwd          string
	palette      *ui.TerminalColorPalette
	terminals    []terminalTab
}

func (a *App) ShowSidebar() {
	a.sidebar.Visible = true
	a.splitPanel.ShowLeft = true
}

func (a *App) HideSidebar() {
	a.sidebar.Visible = false
	a.splitPanel.ShowLeft = false
}

func (a *App) ToggleSidebar() {
	if a.sidebar.Visible {
		a.HideSidebar()
	} else {
		a.ShowSidebar()
	}
}

func (a *App) SetSidebarWidth(w int) {
	if w <= 0 {
		a.HideSidebar()
		return
	}
	if !a.sidebar.Visible {
		a.ShowSidebar()
	}
	a.splitPanel.DividerPos = w
}

func (a *App) FocusEditor() {
	a.root.SetFocus(a.editorGroup)
}

func (a *App) FocusSidebar() {
	if !a.sidebar.Visible {
		a.ShowSidebar()
	}
	if w := a.sidebar.ActiveWidget(); w != nil {
		a.root.SetFocus(w)
	}
}

func (a *App) ShowBottomPanel() {
	a.contentSplit.ShowBottom = true
}

func (a *App) HideBottomPanel() {
	a.contentSplit.ShowBottom = false
	a.FocusEditor()
}

func (a *App) ToggleBottomPanel() {
	if a.contentSplit.ShowBottom {
		a.HideBottomPanel()
	} else {
		a.ShowBottomPanel()
	}
}

func (a *App) SpawnTerminal() {
	r := a.contentSplit.GetRect()
	cols := r.W
	rows := r.H - 3
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	t, err := terminal.New(a.settings.Terminal.Shell, cols, rows, nil)
	if err != nil {
		slog.Error("terminal.New", "err", err)
		a.status.Message = "Failed to open terminal: " + err.Error()
		return
	}

	tw := ui.NewTerminalWidget(t, a.palette)
	idx := len(a.terminals)
	a.terminals = append(a.terminals, terminalTab{term: t, widget: tw})

	panelID := fmt.Sprintf("terminal-%d", idx)
	label := fmt.Sprintf("[>_%d]", idx+1)
	a.bottomPanel.AddPanel(panelID, label, tw)
	a.bottomPanel.SetActivePanel(panelID)

	if !a.contentSplit.ShowBottom {
		a.contentSplit.ShowBottom = true
	}
	a.root.SetFocus(tw)

	t.OnUpdate = func() {
		a.screen.PostEvent(tcell.NewEventInterrupt(nil))
	}
	t.OnExit = func() {
		a.screen.PostEvent(tcell.NewEventInterrupt(panelID))
	}
}

func (a *App) CloseTerminal(panelID string) {
	for i, tt := range a.terminals {
		if fmt.Sprintf("terminal-%d", i) == panelID {
			tt.term.Close()
			break
		}
	}
	a.bottomPanel.RemovePanel(panelID)
	if a.bottomPanel.PanelCount() == 0 {
		a.contentSplit.ShowBottom = false
		a.FocusEditor()
	} else if w := a.bottomPanel.ActiveWidget(); w != nil {
		a.root.SetFocus(w)
	}
}

func (a *App) CloseAllTerminals() {
	panels := a.bottomPanel.PanelIDs()
	for i := len(panels) - 1; i >= 0; i-- {
		a.CloseTerminal(panels[i])
	}
}

func (a *App) ShowDialog(w ui.Widget) {
	a.root.PushOverlay(ui.Overlay{Widget: w, Modal: true})
	a.root.SetFocus(w)
}

func (a *App) DismissDialog() {
	a.root.PopOverlay()
	a.FocusEditor()
}
