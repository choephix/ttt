package main

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/lsp"
	"github.com/eugenioenko/ttt/internal/render"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/terminal"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/view"
	"github.com/eugenioenko/ttt/internal/workspace"

	"github.com/gdamore/tcell/v2"
)

const terminalStripWidth = ui.VerticalTabBarWidth

type terminalTab struct {
	id     string
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
	workspace    *workspace.Workspace
	palette       *ui.TerminalColorPalette
	terminalPanel *ui.TerminalPanelWidget
	terminals     []terminalTab
	lspManager         *lsp.Manager
	docVersionsMu      sync.Mutex
	docVersions        map[string]int
	completionItems    []ui.CompletionItem
	lspCompletionItems []lsp.CompletionItem
	autocompleteTimer  *time.Timer
	hoverTimer         *time.Timer
	hoverGen           uint64
	lastHoverLine      int
	lastHoverCol       int
	problems           *ui.ProblemsWidget
	references         *ui.ReferencesWidget
	allDiagnostics     map[string][]ui.Diagnostic
	keybindings        []config.KeyBinding
	lspNotified        map[string]bool
	reg                *command.Registry
	running            *bool
	quitPending        *bool
}

func (a *App) KeyFor(cmd string) string {
	for _, kb := range a.keybindings {
		if kb.Command == cmd {
			return formatKeyDisplay(kb.Key)
		}
	}
	return ""
}

func formatKeyDisplay(key string) string {
	parts := strings.Fields(key)
	for i, part := range parts {
		tokens := strings.Split(part, "+")
		for j, tok := range tokens {
			if len(tok) > 0 {
				tokens[j] = strings.ToUpper(tok[:1]) + tok[1:]
			}
		}
		parts[i] = strings.Join(tokens, "+")
	}
	return strings.Join(parts, " ")
}

func (a *App) ShowSidebar() {
	a.sidebar.Visible = true
	a.splitPanel.ShowLeft = true
	a.applySearchHighlights()
}

func (a *App) HideSidebar() {
	a.sidebar.Visible = false
	a.splitPanel.ShowLeft = false
	a.editorGroup.ClearSearch()
}

func (a *App) applySearchHighlights() {
	if a.sidebar.ActivePanel == "search" && a.search.Input.Text != "" {
		matches, _ := ui.FindInLines(a.editorGroup.Editor.Buf.Lines, a.search.Input.Text, a.search.Options)
		a.editorGroup.SetSearch(a.search.Input.Text, matches)
	}
}

func (a *App) ShowPanel(id string, widget ui.Widget) {
	a.sidebar.SetActivePanel(id)
	if !a.sidebar.Visible {
		a.ShowSidebar()
	}
	a.root.SetFocus(widget)
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

func (a *App) showTerminalPanel() {
	a.contentSplit.ShowBottom = true
	if len(a.terminals) == 0 {
		a.SpawnTerminal()
	} else {
		a.bottomPanel.SetActivePanel("terminal")
		a.root.SetFocus(a.terminalPanel)
	}
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
	cols := r.W - terminalStripWidth
	rows := r.H - 3
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	t, err := terminal.New(a.settings.Terminal.Shell, cols, rows, a.settings.Terminal.Scrollback, nil, a.workspace.Primary())
	if err != nil {
		slog.Error("terminal.New", "err", err)
		a.StatusError("Failed to open terminal: " + err.Error())
		return
	}

	tw := ui.NewTerminalWidget(t, a.palette)
	panelID := fmt.Sprintf("terminal-%d", len(a.terminals))
	a.terminals = append(a.terminals, terminalTab{id: panelID, term: t, widget: tw})
	a.terminalPanel.AddTerminal(tw)
	a.bottomPanel.SetActivePanel("terminal")

	if !a.contentSplit.ShowBottom {
		a.contentSplit.ShowBottom = true
	}
	a.root.SetFocus(a.terminalPanel)

	t.OnUpdate = func() {
		a.screen.PostEvent(tcell.NewEventInterrupt(nil))
	}
	t.OnExit = func() {
		a.screen.PostEvent(tcell.NewEventInterrupt(panelID))
	}
}

func (a *App) CloseTerminal(panelID string) {
	for i, tt := range a.terminals {
		if tt.id == panelID {
			tt.term.Close()
			a.terminals = append(a.terminals[:i], a.terminals[i+1:]...)
			a.terminalPanel.RemoveTerminal(i)
			break
		}
	}
	if a.terminalPanel.Count() == 0 {
		a.FocusEditor()
	} else {
		a.root.SetFocus(a.terminalPanel)
	}
}

func (a *App) CloseAllTerminals() {
	for i := len(a.terminals) - 1; i >= 0; i-- {
		a.CloseTerminal(a.terminals[i].id)
	}
}

func (a *App) refreshWorkspaceWidgets() {
	paths := a.workspace.Paths()

	existing := make(map[string]bool)
	for _, r := range a.explorer.Roots {
		existing[r.Path] = true
	}
	wanted := make(map[string]bool)
	for _, p := range paths {
		wanted[p] = true
		if !existing[p] {
			a.explorer.AddRoot(p)
		}
	}
	for _, r := range a.explorer.Roots {
		if !wanted[r.Path] {
			a.explorer.RemoveRoot(r.Path)
		}
	}

	a.search.SetWorkDirs(paths)
	a.changes.SetDirs(paths)
	a.changes.Refresh()
}

func (a *App) refreshProblems() {
	var items []ui.ProblemItem
	for path, diags := range a.allDiagnostics {
		for _, d := range diags {
			items = append(items, ui.ProblemItem{
				File:     path,
				Line:     d.StartLine,
				Col:      d.StartCol,
				Severity: d.Severity,
				Message:  d.Message,
				Source:   d.Source,
			})
		}
	}
	a.problems.SetItems(items)
}

func (a *App) checkMouseHover(mx, my int) {
	if a.editorGroup.Editor == nil {
		return
	}
	r := a.editorGroup.Editor.GetRect()
	if mx < r.X || mx >= r.X+r.W || my < r.Y || my >= r.Y+r.H {
		a.DismissHover()
		a.cancelHoverTimer()
		return
	}
	gw := a.editorGroup.Editor.GutterWidth()
	line := my - r.Y + a.editorGroup.Editor.Viewport.TopLine
	col := mx - r.X - gw + a.editorGroup.Editor.Viewport.LeftCol
	if col < 0 {
		a.DismissHover()
		a.cancelHoverTimer()
		return
	}
	if line == a.lastHoverLine && col == a.lastHoverCol {
		return
	}
	a.lastHoverLine = line
	a.lastHoverCol = col
	a.DismissHover()
	a.cancelHoverTimer()
	delay := a.settings.LSP.HoverDelay
	if delay <= 0 {
		delay = 400
	}
	path := a.editorGroup.ActiveFilePath()
	lang := ""
	if a.editorGroup.Editor.Highlighter != nil {
		lang = a.editorGroup.Editor.Highlighter.Language()
	}
	a.hoverTimer = time.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
		a.RequestHover(path, lang, line, col, mx, my)
	})
}

func (a *App) cancelHoverTimer() {
	if a.hoverTimer != nil {
		a.hoverTimer.Stop()
		a.hoverTimer = nil
	}
}

func (a *App) ShowHover(text string, anchorX, anchorY int) {
	if text == "" {
		return
	}
	a.editorGroup.Hover = ui.NewHoverWidget(text, anchorX, anchorY)
}

func (a *App) isMouseOverHover(mx, my int) bool {
	h := a.editorGroup.Hover
	if h == nil {
		return false
	}
	r := h.GetRect()
	return mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H
}

func (a *App) DismissHover() {
	a.editorGroup.Hover = nil
	a.cancelHoverTimer()
	a.hoverGen++
}

func (a *App) statusMessage(msg string, level view.NotifyLevel) {
	a.status.SetNotification(msg, level, 5*time.Second)
	time.AfterFunc(5*time.Second, func() {
		a.screen.PostEvent(tcell.NewEventInterrupt(nil))
	})
}

func (a *App) StatusNotify(msg string) { a.statusMessage(msg, view.NotifyInfo) }
func (a *App) StatusWarn(msg string)   { a.statusMessage(msg, view.NotifyWarning) }
func (a *App) StatusError(msg string)  { a.statusMessage(msg, view.NotifyError) }

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

func (a *App) ShowDialog(w ui.Widget) {
	a.root.PushOverlay(ui.Overlay{Widget: w, Modal: true})
	a.root.SetFocus(w)
}

func (a *App) ShowFindBar(w ui.Widget) {
	a.root.PushOverlay(ui.Overlay{Widget: w, Modal: false})
}

func (a *App) DismissDialog() {
	a.root.PopOverlay()
	a.FocusEditor()
}

func (a *App) ShowInputDialog(title, placeholder, initial string, onSubmit func(string)) {
	dialog := ui.NewInputDialogWidget(title, placeholder, initial)
	dialog.Borders = a.borders
	dialog.OnSubmit = func(value string) {
		a.DismissDialog()
		onSubmit(value)
	}
	dialog.OnDismiss = func() {
		a.DismissDialog()
	}
	a.ShowDialog(dialog)
}

func (a *App) ShowConfirmDialog(message string, buttons []string, callbacks []func()) {
	var dialog *ui.ConfirmDialogWidget
	if len(buttons) == 3 {
		dialog = ui.NewConfirmDialogWidget3(message, buttons[0], buttons[1], buttons[2])
	} else if len(buttons) == 2 {
		dialog = ui.NewConfirmDialogWidget2(message, buttons[0], buttons[1])
	} else {
		dialog = ui.NewConfirmDialogWidget(message)
	}
	dialog.Borders = a.borders
	for i, cb := range callbacks {
		if i < len(dialog.OnButton) {
			dialog.OnButton[i] = cb
		}
	}
	dialog.OnDismiss = func() {
		a.DismissDialog()
	}
	a.ShowDialog(dialog)
}

func (a *App) showDiffFindBar(dv *ui.DiffViewWidget) {
	findBar := ui.NewFindBarWidget()
	findBar.Borders = a.borders
	findBar.OnSearch = func(query string, opts ui.SearchOptions) []ui.FindMatch {
		leftMatches, err := ui.FindInLines(dv.LeftLines(), query, opts)
		if err != nil {
			a.StatusWarn("Invalid regex: " + err.Error())
			return nil
		}
		rightMatches, _ := ui.FindInLines(dv.RightLines(), query, opts)
		return dv.SetSearchMatches(leftMatches, rightMatches)
	}
	findBar.OnNavigate = func(match ui.FindMatch) {
		dv.SetActiveMatch(findBar.Current)
		dv.ScrollToLine(match.Line)
	}
	findBar.OnDismiss = func() {
		a.DismissDialog()
		dv.ClearSearch()
	}
	a.ShowFindBar(findBar)
}

func (a *App) ShowPicker(items []command.Command, onSelect func(id string)) {
	picker := ui.NewCommandPaletteWidget(items)
	picker.Borders = a.borders
	picker.OnExecute = func(id string) {
		a.DismissDialog()
		onSelect(id)
	}
	picker.OnDismiss = func() {
		a.DismissDialog()
	}
	a.ShowDialog(picker)
}