package main

import (
	"fmt"
	"log/slog"
	"strings"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/core/undo"
	"github.com/eugenioenko/ttt/internal/lsp"
	"github.com/eugenioenko/ttt/internal/render"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/terminal"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/view"
	"github.com/eugenioenko/ttt/internal/workspace"

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
	workspace    *workspace.Workspace
	palette      *ui.TerminalColorPalette
	terminals    []terminalTab
	lspManager   *lsp.Manager
	docVersions  map[string]int
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

func (a *App) ShowAutocomplete(items []ui.CompletionItem) {
	ac := ui.NewAutocompleteWidget(items, 0, 0)
	ac.OnSelect = func(item ui.CompletionItem) {
		a.DismissAutocomplete()
		a.insertCompletion(item)
	}
	ac.OnDismiss = func() {
		a.DismissAutocomplete()
	}
	a.editorGroup.Autocomplete = ac
}

func (a *App) DismissAutocomplete() {
	a.editorGroup.Autocomplete = nil
}

func (a *App) IsAutocompleteActive() bool {
	return a.editorGroup.Autocomplete != nil
}

func (a *App) insertCompletion(item ui.CompletionItem) {
	if !a.editorGroup.IsEditorActive() {
		return
	}
	text := item.InsertText
	if text == "" {
		text = item.Label
	}
	editor := a.editorGroup.Editor
	line := editor.Cursor.Line
	col := editor.Cursor.Col
	runes := []rune(editor.Buf.Lines[line])
	start := col
	for start > 0 && isIdentRune(runes[start-1]) {
		start--
	}
	if start < col {
		editor.ExecCommand(&undo.DeleteSelectionCommand{
			StartLine: line, StartCol: start,
			EndLine: line, EndCol: col,
		})
	}
	editor.ExecCommand(&undo.InsertStringCommand{
		Line: line, Col: start, Text: text,
	})
	editor.Cursor.Line = line
	editor.Cursor.Col = start + len([]rune(text))
}

func isIdentRune(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func (a *App) RequestCompletions(path, lang string, line, col int) {
	if a.lspManager == nil || lang == "" {
		return
	}
	langKey := strings.ToLower(lang)
	if !a.lspManager.HasServer(langKey) {
		return
	}
	workDir := a.workspace.Primary()
	if folder := a.workspace.FolderForFile(path); folder != nil {
		workDir = folder.Path
	}
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			slog.Error("lsp client", "err", err)
			return
		}
		items, err := client.Completion(fileURI(path), line, col)
		if err != nil {
			slog.Error("lsp completion", "err", err)
			return
		}
		uiItems := lspToUICompletions(items)
		if len(uiItems) > 0 {
			a.screen.PostEvent(tcell.NewEventInterrupt(&completionResult{items: uiItems}))
		}
	}()
}

func (a *App) NotifyLSPOpen(path, lang, text string) {
	if a.lspManager == nil || lang == "" {
		return
	}
	langKey := strings.ToLower(lang)
	if !a.lspManager.HasServer(langKey) {
		return
	}
	workDir := a.workspace.Primary()
	if folder := a.workspace.FolderForFile(path); folder != nil {
		workDir = folder.Path
	}
	if a.docVersions == nil {
		a.docVersions = make(map[string]int)
	}
	a.docVersions[path] = 1
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			return
		}
		client.DidOpen(fileURI(path), langKey, text)
	}()
}

func (a *App) NotifyLSPChange(path, lang, text string) {
	if a.lspManager == nil || lang == "" {
		return
	}
	langKey := strings.ToLower(lang)
	if !a.lspManager.HasServer(langKey) {
		return
	}
	if a.docVersions == nil {
		a.docVersions = make(map[string]int)
	}
	a.docVersions[path]++
	version := a.docVersions[path]
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, "")
		if err != nil {
			return
		}
		client.DidChange(fileURI(path), text, version)
	}()
}

func (a *App) NotifyLSPClose(path, lang string) {
	if a.lspManager == nil || lang == "" {
		return
	}
	langKey := strings.ToLower(lang)
	if !a.lspManager.HasServer(langKey) {
		return
	}
	delete(a.docVersions, path)
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, "")
		if err != nil {
			return
		}
		client.DidClose(fileURI(path))
	}()
}

func (a *App) ShowDialog(w ui.Widget) {
	a.root.PushOverlay(ui.Overlay{Widget: w, Modal: true})
	a.root.SetFocus(w)
}

func (a *App) DismissDialog() {
	a.root.PopOverlay()
	a.FocusEditor()
}
