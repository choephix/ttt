package main

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
	"github.com/eugenioenko/ttt/internal/command"
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
	problems           *ui.ProblemsWidget
	allDiagnostics     map[string][]ui.Diagnostic
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
	cols := r.W - terminalStripWidth
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

func (a *App) checkDiagnosticHover(mx, my int) {
	if a.editorGroup.Editor == nil {
		return
	}
	r := a.editorGroup.Editor.GetRect()
	if mx < r.X || mx >= r.X+r.W || my < r.Y || my >= r.Y+r.H {
		a.DismissHover()
		return
	}
	gw := a.editorGroup.Editor.GutterWidth()
	line := my - r.Y + a.editorGroup.Editor.Viewport.TopLine
	col := mx - r.X - gw + a.editorGroup.Editor.Viewport.LeftCol
	if col < 0 {
		a.DismissHover()
		return
	}
	d := a.editorGroup.Editor.DiagnosticAt(line, col)
	if d != nil {
		a.ShowHover(d.Message)
	} else {
		a.DismissHover()
	}
}

func (a *App) ShowHover(text string) {
	if text == "" {
		return
	}
	a.editorGroup.Hover = ui.NewHoverWidget(text, 0, 0)
}

func (a *App) DismissHover() {
	a.editorGroup.Hover = nil
}

func (a *App) ShowAutocomplete(items []ui.CompletionItem, lspItems []lsp.CompletionItem) {
	a.completionItems = items
	a.lspCompletionItems = lspItems
	prefix := a.currentPrefix()
	filtered := ui.FilterCompletions(items, prefix)
	if len(filtered) == 0 {
		return
	}
	ac := ui.NewAutocompleteWidget(filtered, 0, 0)
	ac.OnSelect = func(item ui.CompletionItem) {
		a.resolveAndInsert(item)
		a.DismissAutocomplete()
	}
	ac.OnDismiss = func() {
		a.DismissAutocomplete()
	}
	a.editorGroup.Autocomplete = ac
}

func (a *App) RefreshAutocomplete() {
	if a.editorGroup.Autocomplete == nil || len(a.completionItems) == 0 {
		return
	}
	prefix := a.currentPrefix()
	if prefix == "" {
		a.DismissAutocomplete()
		return
	}
	filtered := ui.FilterCompletions(a.completionItems, prefix)
	if len(filtered) == 0 {
		a.DismissAutocomplete()
		return
	}
	a.editorGroup.Autocomplete.SetItems(filtered)
}

func (a *App) identStart() (line, start, col int) {
	if !a.editorGroup.IsEditorActive() {
		return 0, 0, 0
	}
	editor := a.editorGroup.Editor
	line = editor.Cursor.Line
	col = editor.Cursor.Col
	if line >= len(editor.Buf.Lines) {
		return 0, 0, 0
	}
	runes := []rune(editor.Buf.Lines[line])
	start = col
	if start > len(runes) {
		start = len(runes)
	}
	for start > 0 && isIdentRune(runes[start-1]) {
		start--
	}
	return line, start, col
}

func (a *App) currentPrefix() string {
	if !a.editorGroup.IsEditorActive() {
		return ""
	}
	editor := a.editorGroup.Editor
	line, start, col := a.identStart()
	runes := []rune(editor.Buf.Lines[line])
	if col > len(runes) {
		col = len(runes)
	}
	if start > col {
		return ""
	}
	return string(runes[start:col])
}

func (a *App) DismissAutocomplete() {
	a.editorGroup.Autocomplete = nil
	a.completionItems = nil
	a.lspCompletionItems = nil
}

func (a *App) IsAutocompleteActive() bool {
	return a.editorGroup.Autocomplete != nil
}

func (a *App) resolveAndInsert(item ui.CompletionItem) {
	var lspItem *lsp.CompletionItem
	for i, li := range a.lspCompletionItems {
		if li.Label == item.Label {
			lspItem = &a.lspCompletionItems[i]
			break
		}
	}

	if lspItem != nil && len(lspItem.AdditionalTextEdits) == 0 {
		path := a.editorGroup.ActiveFilePath()
		lang := ""
		if a.editorGroup.Editor != nil && a.editorGroup.Editor.Highlighter != nil {
			lang = a.editorGroup.Editor.Highlighter.Language()
		}
		langKey, ok := a.lspReady(lang)
		if ok {
			workDir := a.lspWorkDir(path)
			client, err := a.lspManager.ClientForLanguage(langKey, workDir)
			if err == nil {
				resolved, err := client.ResolveCompletion(*lspItem)
				if err == nil && resolved != nil {
					for _, edit := range resolved.AdditionalTextEdits {
						item.AdditionalEdits = append(item.AdditionalEdits, ui.AdditionalEdit{
							StartLine: edit.Range.Start.Line,
							StartCol:  edit.Range.Start.Character,
							EndLine:   edit.Range.End.Line,
							EndCol:    edit.Range.End.Character,
							NewText:   edit.NewText,
						})
					}
				}
			}
		}
	}

	a.insertCompletion(item)
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
	line, start, col := a.identStart()
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

	for i := len(item.AdditionalEdits) - 1; i >= 0; i-- {
		edit := item.AdditionalEdits[i]
		linesBefore := len(editor.Buf.Lines)

		if edit.StartLine != edit.EndLine || edit.StartCol != edit.EndCol {
			editor.ExecCommand(&undo.DeleteSelectionCommand{
				StartLine: edit.StartLine, StartCol: edit.StartCol,
				EndLine: edit.EndLine, EndCol: edit.EndCol,
			})
		}
		if edit.NewText != "" {
			suffix := ""
			if edit.StartLine < len(editor.Buf.Lines) {
				runes := []rune(editor.Buf.Lines[edit.StartLine])
				if edit.StartCol < len(runes) {
					suffix = string(runes[edit.StartCol:])
				}
			}
			editor.ExecCommand(&undo.PasteCommand{
				Line: edit.StartLine, Col: edit.StartCol,
				Text: edit.NewText, Suffix: suffix,
			})
		}

		linesAdded := len(editor.Buf.Lines) - linesBefore
		if linesAdded != 0 && edit.StartLine <= editor.Cursor.Line {
			editor.Cursor.Line += linesAdded
		}
	}
}

func (a *App) ScheduleAutocomplete() {
	if !a.settings.Autocomplete.Enabled || !a.settings.Autocomplete.AutoSuggest {
		return
	}
	if a.autocompleteTimer != nil {
		a.autocompleteTimer.Stop()
	}
	delay := time.Duration(a.settings.Autocomplete.Debounce) * time.Millisecond
	a.autocompleteTimer = time.AfterFunc(delay, func() {
		a.screen.PostEvent(tcell.NewEventInterrupt(&autocompleteTrigger{}))
	})
}

func (a *App) CheckSignatureHelpTrigger() {
	if !a.settings.Autocomplete.Enabled || !a.settings.Autocomplete.SignatureHelp {
		return
	}
	if !a.editorGroup.IsEditorActive() {
		return
	}
	editor := a.editorGroup.Editor
	line := editor.Cursor.Line
	col := editor.Cursor.Col
	if col <= 0 || line >= len(editor.Buf.Lines) {
		return
	}
	runes := []rune(editor.Buf.Lines[line])
	if col > len(runes) {
		return
	}
	ch := runes[col-1]
	if ch == '(' || ch == ',' {
		path := a.editorGroup.ActiveFilePath()
		lang := ""
		if editor.Highlighter != nil {
			lang = editor.Highlighter.Language()
		}
		a.RequestSignatureHelp(path, lang, line, col)
	} else if ch == ')' {
		a.DismissSignatureHelp()
	}
}

func (a *App) RequestSignatureHelp(path, lang string, line, col int) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		return
	}
	workDir := a.lspWorkDir(path)
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			slog.Error("lsp client", "err", err)
			return
		}
		sig, err := client.SignatureHelp(fileURI(path), line, col)
		if err != nil {
			slog.Error("lsp signatureHelp", "err", err)
			return
		}
		if sig != nil && len(sig.Signatures) > 0 {
			result := lspToSignatureHelpResult(sig)
			if result.label != "" {
				a.screen.PostEvent(tcell.NewEventInterrupt(result))
			}
		}
	}()
}

func (a *App) ShowSignatureHelp(result *signatureHelpResult) {
	w := ui.NewSignatureHelpWidget(result.label, result.paramStart, result.paramEnd)
	a.editorGroup.SignatureHelp = w
}

func (a *App) DismissSignatureHelp() {
	a.editorGroup.SignatureHelp = nil
}

func (a *App) editorTabSize() (int, bool) {
	tabSize := a.settings.TabSize
	insertSpaces := a.settings.InsertSpaces
	if a.editorGroup.Editor != nil && a.editorGroup.Editor.TabSize > 0 {
		tabSize = a.editorGroup.Editor.TabSize
	}
	return tabSize, insertSpaces
}

func (a *App) RequestFormatting(path, lang string) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		if lang != "" {
			a.StatusWarn(lang + " language server is not configured")
		}
		return
	}
	workDir := a.lspWorkDir(path)
	tabSize, insertSpaces := a.editorTabSize()
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			slog.Error("lsp client", "err", err)
			return
		}
		edits, err := client.Formatting(fileURI(path), tabSize, insertSpaces)
		if err != nil {
			slog.Error("lsp formatting", "err", err)
			return
		}
		if len(edits) > 0 {
			a.screen.PostEvent(tcell.NewEventInterrupt(&formattingResult{edits: edits}))
		}
	}()
}

func (a *App) RequestRangeFormatting(path, lang string, startLine, startCol, endLine, endCol int) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		if lang != "" {
			a.StatusWarn(lang + " language server is not configured")
		}
		return
	}
	workDir := a.lspWorkDir(path)
	tabSize, insertSpaces := a.editorTabSize()
	r := lsp.Range{
		Start: lsp.Position{Line: startLine, Character: startCol},
		End:   lsp.Position{Line: endLine, Character: endCol},
	}
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			slog.Error("lsp client", "err", err)
			return
		}
		edits, err := client.RangeFormatting(fileURI(path), r, tabSize, insertSpaces)
		if err != nil {
			slog.Error("lsp rangeFormatting", "err", err)
			return
		}
		if len(edits) > 0 {
			a.screen.PostEvent(tcell.NewEventInterrupt(&formattingResult{edits: edits}))
		}
	}()
}

func (a *App) FormatOnSave(path, lang string) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		return
	}
	workDir := a.lspWorkDir(path)
	tabSize, insertSpaces := a.editorTabSize()
	client, err := a.lspManager.ClientForLanguage(langKey, workDir)
	if err != nil {
		return
	}
	edits, err := client.Formatting(fileURI(path), tabSize, insertSpaces)
	if err != nil {
		slog.Error("lsp formatOnSave", "err", err)
		return
	}
	if len(edits) > 0 {
		a.ApplyTextEdits(edits)
	}
}

func (a *App) ApplyTextEdits(edits []lsp.TextEdit) {
	if !a.editorGroup.IsEditorActive() {
		return
	}
	editor := a.editorGroup.Editor

	sort.Slice(edits, func(i, j int) bool {
		if edits[i].Range.Start.Line != edits[j].Range.Start.Line {
			return edits[i].Range.Start.Line > edits[j].Range.Start.Line
		}
		return edits[i].Range.Start.Character > edits[j].Range.Start.Character
	})

	for _, edit := range edits {
		sl, sc := edit.Range.Start.Line, edit.Range.Start.Character
		el, ec := edit.Range.End.Line, edit.Range.End.Character

		if sl != el || sc != ec {
			editor.ExecCommand(&undo.DeleteSelectionCommand{
				StartLine: sl, StartCol: sc,
				EndLine: el, EndCol: ec,
			})
		}
		if edit.NewText != "" {
			suffix := ""
			if sl < len(editor.Buf.Lines) {
				runes := []rune(editor.Buf.Lines[sl])
				if sc < len(runes) {
					suffix = string(runes[sc:])
				}
			}
			editor.ExecCommand(&undo.PasteCommand{
				Line: sl, Col: sc,
				Text: edit.NewText, Suffix: suffix,
			})
		}
	}
}

func isIdentRune(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func (a *App) lspReady(lang string) (string, bool) {
	if a.lspManager == nil || lang == "" {
		return "", false
	}
	langKey := strings.ToLower(lang)
	if !a.lspManager.HasServer(langKey) {
		return "", false
	}
	return langKey, true
}

func (a *App) RequestCompletions(path, lang string, line, col int) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		return
	}
	workDir := a.lspWorkDir(path)
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
			a.screen.PostEvent(tcell.NewEventInterrupt(&completionResult{items: uiItems, lspItems: items}))
		}
	}()
}

func (a *App) RequestHover(path, lang string, line, col int) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		if lang != "" {
			a.StatusWarn(lang + " language server is not configured")
		}
		return
	}
	workDir := a.lspWorkDir(path)
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			slog.Error("lsp client", "err", err)
			return
		}
		hover, err := client.Hover(fileURI(path), line, col)
		if err != nil {
			slog.Error("lsp hover", "err", err)
			return
		}
		if hover != nil && hover.Contents.Value != "" {
			a.screen.PostEvent(tcell.NewEventInterrupt(&hoverResult{text: hover.Contents.Value}))
		}
	}()
}

func (a *App) RequestDefinition(path, lang string, line, col int) {
	a.requestLocation("textDocument/definition", path, lang, line, col)
}

func (a *App) RequestImplementation(path, lang string, line, col int) {
	a.requestLocation("textDocument/implementation", path, lang, line, col)
}

func (a *App) RequestTypeDefinition(path, lang string, line, col int) {
	a.requestLocation("textDocument/typeDefinition", path, lang, line, col)
}

func (a *App) requestLocation(method, path, lang string, line, col int) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		if lang != "" {
			a.StatusWarn(lang + " language server is not configured")
		}
		return
	}
	workDir := a.lspWorkDir(path)
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			slog.Error("lsp client", "err", err)
			return
		}
		var locs []lsp.Location
		var reqErr error
		switch method {
		case "textDocument/definition":
			locs, reqErr = client.Definition(fileURI(path), line, col)
		case "textDocument/implementation":
			locs, reqErr = client.Implementation(fileURI(path), line, col)
		case "textDocument/typeDefinition":
			locs, reqErr = client.TypeDefinition(fileURI(path), line, col)
		}
		if reqErr != nil {
			slog.Error("lsp "+method, "err", reqErr)
			return
		}
		if len(locs) > 0 {
			a.screen.PostEvent(tcell.NewEventInterrupt(&locationResult{locations: locs}))
		}
	}()
}

func (a *App) lspWorkDir(path string) string {
	workDir := a.workspace.Primary()
	if folder := a.workspace.FolderForFile(path); folder != nil {
		workDir = folder.Path
	}
	return workDir
}

func (a *App) NotifyLSPOpen(path, lang, text string) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		return
	}
	workDir := a.lspWorkDir(path)
	a.docVersionsMu.Lock()
	a.docVersions[path] = 1
	a.docVersionsMu.Unlock()
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			return
		}
		client.DidOpen(fileURI(path), langKey, text)
	}()
}

func (a *App) NotifyLSPChange(path, lang, text string) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		return
	}
	workDir := a.lspWorkDir(path)
	a.docVersionsMu.Lock()
	a.docVersions[path]++
	version := a.docVersions[path]
	a.docVersionsMu.Unlock()
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			return
		}
		client.DidChange(fileURI(path), text, version)
	}()
}

func (a *App) NotifyLSPSave(path, lang, text string) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		return
	}
	workDir := a.lspWorkDir(path)
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			return
		}
		client.DidSave(fileURI(path), text)
	}()
}

func (a *App) NotifyLSPClose(path, lang string) {
	langKey, ok := a.lspReady(lang)
	if !ok {
		return
	}
	workDir := a.lspWorkDir(path)
	a.docVersionsMu.Lock()
	delete(a.docVersions, path)
	a.docVersionsMu.Unlock()
	go func() {
		client, err := a.lspManager.ClientForLanguage(langKey, workDir)
		if err != nil {
			return
		}
		client.DidClose(fileURI(path))
	}()
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

func (a *App) ShowDialog(w ui.Widget) {
	a.root.PushOverlay(ui.Overlay{Widget: w, Modal: true})
	a.root.SetFocus(w)
}

func (a *App) DismissDialog() {
	a.root.PopOverlay()
	a.FocusEditor()
}

func (a *App) ShowInputDialog(title, initial string, onSubmit func(string)) {
	dialog := ui.NewInputDialogWidget(title, initial)
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