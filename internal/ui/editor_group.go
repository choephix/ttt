package ui

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/core/buffer"
	"github.com/eugenioenko/ttt/internal/core/clipboard"
	"github.com/eugenioenko/ttt/internal/core/cursor"
	"github.com/eugenioenko/ttt/internal/core/diff"
	"github.com/eugenioenko/ttt/internal/core/highlight"
	"github.com/eugenioenko/ttt/internal/core/selection"
	"github.com/eugenioenko/ttt/internal/core/undo"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/view"

	"github.com/gdamore/tcell/v2"
)

type DiagnosticSeverity int

const (
	DiagError       DiagnosticSeverity = 1
	DiagWarning     DiagnosticSeverity = 2
	DiagInformation DiagnosticSeverity = 3
	DiagHint        DiagnosticSeverity = 4
)

type Diagnostic struct {
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
	Severity  DiagnosticSeverity
	Message   string
	Source    string
}

type editorTab struct {
	FilePath    string
	Buf         *buffer.Buffer
	Cur         *cursor.Cursor
	Vp          *view.Viewport
	Undo        *undo.UndoStack
	Sel         *selection.Selection
	Highlighter *highlight.Highlighter
	TabSize     int
	Content     Widget
	Pinned      bool
}

type EditorGroupWidget struct {
	BaseWidget
	TabBar       *TabBarWidget
	Editor       *EditorPaneWidget
	Autocomplete  *AutocompleteWidget
	Hover         *HoverWidget
	SignatureHelp *SignatureHelpWidget
	tabs         []editorTab
	active       int
	TabSize      int
	LineNumbers  bool
	Borders      *term.BorderSet
	OnFileOpen   func(path, lang, text string)
	OnFileChange func(path, lang, text string)
	OnFileClose  func(path, lang string)
	OnError      func(msg string)
}

func NewEditorGroupWidget(borders *term.BorderSet, tabSize int, lineNumbers bool) *EditorGroupWidget {
	editor := NewEditorPaneWidget(
		&buffer.Buffer{Lines: []string{""}},
		&cursor.Cursor{},
		&view.Viewport{},
	)
	editor.TabSize = tabSize
	editor.LineNumbers = lineNumbers

	tabBar := NewTabBarWidget()
	tabBar.Borders = borders
	tabBar.MoreButton = NewMoreButtonWidget()

	g := &EditorGroupWidget{
		TabBar:      tabBar,
		Editor:      editor,
		TabSize:     tabSize,
		LineNumbers: lineNumbers,
		Borders:     borders,
	}
	tabBar.OnTabClick = func(index int) {
		g.SwitchTab(index)
	}
	undoStack := &undo.UndoStack{}
	sel := &selection.Selection{}
	editor.Undo = undoStack
	editor.Selection = sel
	g.tabs = []editorTab{{
		FilePath: "untitled",
		Buf:      editor.Buf,
		Cur:      editor.Cursor,
		Vp:       editor.Viewport,
		Undo:     undoStack,
		Sel:      sel,
	}}
	g.syncTabs()
	return g
}

func (g *EditorGroupWidget) Focusable() bool { return true }

func (g *EditorGroupWidget) activeTab() *editorTab {
	if len(g.tabs) == 0 || g.active < 0 || g.active >= len(g.tabs) {
		return nil
	}
	return &g.tabs[g.active]
}

func (g *EditorGroupWidget) reportError(msg string) {
	if g.OnError != nil {
		g.OnError(msg)
	} else {
		slog.Error(msg)
	}
}

func (g *EditorGroupWidget) OpenFile(path string) {
	for i := range g.tabs {
		if g.tabs[i].FilePath == path {
			g.tabs[i].Pinned = true
			if g.tabs[i].Buf != nil && !g.tabs[i].Buf.Dirty {
				g.tabs[i].Buf.LoadFile(path)
			}
			g.SwitchTab(i)
			return
		}
	}
	newBuf := &buffer.Buffer{Lines: []string{""}}
	if err := newBuf.LoadFile(path); err != nil {
		g.reportError(fmt.Sprintf("Failed to open %s: %v", path, err))
		return
	}
	tabSize := g.TabSize
	ec := config.LoadEditorConfig(path)
	if ec.IndentSize > 0 {
		tabSize = ec.IndentSize
	} else if detected := buffer.DetectIndent(newBuf.Lines); detected.Size > 0 {
		tabSize = detected.Size
	}
	newTab := editorTab{
		FilePath:    path,
		Buf:         newBuf,
		Cur:         &cursor.Cursor{},
		Vp:          &view.Viewport{},
		Undo:        &undo.UndoStack{},
		Sel:         &selection.Selection{},
		Highlighter: highlight.New(path),
		TabSize:     tabSize,
	}
	if t := g.activeTab(); t != nil && !t.Pinned && t.Content == nil && t.Buf != nil && !t.Buf.Dirty {
		g.tabs[g.active] = newTab
		g.syncTabs()
	} else {
		g.tabs = append(g.tabs, newTab)
		g.SwitchTab(len(g.tabs) - 1)
	}
	if g.OnFileOpen != nil && newTab.Highlighter != nil {
		g.OnFileOpen(path, newTab.Highlighter.Language(), strings.Join(newBuf.Lines, "\n"))
	}
}

func (g *EditorGroupWidget) OpenBuffer(path string, buf *buffer.Buffer) {
	for i, t := range g.tabs {
		if t.FilePath == path {
			g.SwitchTab(i)
			return
		}
	}
	g.tabs = append(g.tabs, editorTab{
		FilePath: path,
		Buf:      buf,
		Cur:      &cursor.Cursor{},
		Vp:       &view.Viewport{},
		Undo:     &undo.UndoStack{},
		Sel:      &selection.Selection{},
	})
	g.SwitchTab(len(g.tabs) - 1)
}

func (g *EditorGroupWidget) OpenDiff(path string, fd diff.FileDiff) {
	tabName := path + " (diff)"
	for i, t := range g.tabs {
		if t.FilePath == tabName {
			t.Content = NewDiffViewWidget(path, fd)
			g.tabs[i] = t
			g.SwitchTab(i)
			return
		}
	}
	widget := NewDiffViewWidget(path, fd)
	g.tabs = append(g.tabs, editorTab{
		FilePath: tabName,
		Content:  widget,
	})
	g.SwitchTab(len(g.tabs) - 1)
}

func (g *EditorGroupWidget) ReloadFile(path string) {
	for i := range g.tabs {
		if g.tabs[i].FilePath == path && g.tabs[i].Buf != nil {
			g.tabs[i].Buf.LoadFile(path)
			g.tabs[i].Buf.Dirty = false
			if i == g.active {
				g.syncTabs()
			}
			return
		}
	}
}

func (g *EditorGroupWidget) IsEditorActive() bool {
	t := g.activeTab()
	return t != nil && t.Content == nil
}

func (g *EditorGroupWidget) CursorPosition() (int, int, bool) {
	if g.IsEditorActive() {
		return g.Editor.CursorX, g.Editor.CursorY, true
	}
	return 0, 0, false
}

func (g *EditorGroupWidget) SetTabSize(size int) {
	if t := g.activeTab(); t != nil {
		t.TabSize = size
	}
	g.Editor.TabSize = size
}

func (g *EditorGroupWidget) SwitchTab(idx int) {
	if idx >= 0 && idx < len(g.tabs) {
		g.active = idx
		g.syncTabs()
	}
}

func (g *EditorGroupWidget) NextTab() {
	if len(g.tabs) > 1 {
		g.SwitchTab((g.active + 1) % len(g.tabs))
	}
}

func (g *EditorGroupWidget) PrevTab() {
	if len(g.tabs) > 1 {
		g.SwitchTab((g.active - 1 + len(g.tabs)) % len(g.tabs))
	}
}

func (g *EditorGroupWidget) CloseTab() {
	if len(g.tabs) == 0 {
		return
	}
	closing := g.tabs[g.active]
	if g.OnFileClose != nil && closing.Highlighter != nil && closing.FilePath != "untitled" {
		g.OnFileClose(closing.FilePath, closing.Highlighter.Language())
	}
	g.tabs = append(g.tabs[:g.active], g.tabs[g.active+1:]...)
	if len(g.tabs) == 0 {
		g.tabs = []editorTab{{
			FilePath: "untitled",
			Buf:      &buffer.Buffer{Lines: []string{""}},
			Cur:      &cursor.Cursor{},
			Vp:       &view.Viewport{},
			Undo:     &undo.UndoStack{},
			Sel:      &selection.Selection{},
		}}
		g.active = 0
	} else if g.active >= len(g.tabs) {
		g.active = len(g.tabs) - 1
	}
	g.syncTabs()
}

func (g *EditorGroupWidget) CloseOtherTabs() {
	t := g.activeTab()
	if t == nil || len(g.tabs) <= 1 {
		return
	}
	g.tabs = []editorTab{*t}
	g.active = 0
	g.syncTabs()
}

func (g *EditorGroupWidget) CloseAllTabs() {
	g.tabs = []editorTab{{
		FilePath: "untitled",
		Buf:      &buffer.Buffer{Lines: []string{""}},
		Cur:      &cursor.Cursor{},
		Vp:       &view.Viewport{},
		Undo:     &undo.UndoStack{},
		Sel:      &selection.Selection{},
	}}
	g.active = 0
	g.syncTabs()
}

func (g *EditorGroupWidget) Save() bool {
	t := g.activeTab()
	if t == nil || t.Content != nil {
		return false
	}
	if t.FilePath == "untitled" {
		return false
	}
	if err := t.Buf.SaveFile(t.FilePath); err != nil {
		g.reportError(fmt.Sprintf("Failed to save %s: %v", t.FilePath, err))
		return false
	}
	return true
}

func (g *EditorGroupWidget) SaveAs(path string) {
	t := g.activeTab()
	if t == nil || t.Content != nil {
		return
	}
	if err := t.Buf.SaveFile(path); err != nil {
		g.reportError(fmt.Sprintf("Failed to save %s: %v", path, err))
		return
	}
	t.FilePath = path
	t.Highlighter = highlight.New(path)
	g.syncTabs()
}

func (g *EditorGroupWidget) ActiveFilePath() string {
	if t := g.activeTab(); t != nil {
		return t.FilePath
	}
	return ""
}

func (g *EditorGroupWidget) ActiveCursor() (line, col int) {
	t := g.activeTab()
	if t == nil || t.Content != nil {
		return 0, 0
	}
	return t.Cur.Line, t.Cur.Col
}

func (g *EditorGroupWidget) ActiveFileName() string {
	t := g.activeTab()
	if t == nil {
		return "untitled"
	}
	return filepath.Base(t.FilePath)
}

func (g *EditorGroupWidget) IsDirty() bool {
	t := g.activeTab()
	if t == nil || t.Content != nil {
		return false
	}
	return t.Buf.Dirty
}

func (g *EditorGroupWidget) AnyDirty() bool {
	for _, t := range g.tabs {
		if t.Content != nil {
			continue
		}
		if t.Buf.Dirty {
			return true
		}
	}
	return false
}

func (g *EditorGroupWidget) Undo() {
	t := g.activeTab()
	if t == nil || t.Content != nil {
		return
	}
	if t.Undo != nil {
		t.Undo.Undo(t.Buf)
	}
}

func (g *EditorGroupWidget) Redo() {
	t := g.activeTab()
	if t == nil || t.Content != nil {
		return
	}
	if t.Undo != nil {
		t.Undo.Redo(t.Buf)
	}
}

func (g *EditorGroupWidget) SelectAll() {
	t := g.activeTab()
	if t == nil || t.Content != nil || t.Sel == nil {
		return
	}
	t.Sel.Start(0, 0)
	lastLine := len(t.Buf.Lines) - 1
	t.Cur.Line = lastLine
	t.Cur.Col = len([]rune(t.Buf.Lines[lastLine]))
}

func (g *EditorGroupWidget) SetSearch(query string, matches []FindMatch) {
	if !g.IsEditorActive() {
		return
	}
	g.Editor.SearchQuery = query
	g.Editor.SearchMatches = matches
	g.Editor.SearchActive = 0
}

func (g *EditorGroupWidget) SetSearchActive(idx int) {
	if !g.IsEditorActive() {
		return
	}
	g.Editor.SearchActive = idx
}

func (g *EditorGroupWidget) GoToLine(line int) {
	if !g.IsEditorActive() {
		return
	}
	if line < 1 {
		line = 1
	}
	if line > len(g.Editor.Buf.Lines) {
		line = len(g.Editor.Buf.Lines)
	}
	g.Editor.Cursor.Line = line - 1
	g.Editor.Cursor.Col = 0
	g.Editor.scrollViewport()
}

func (g *EditorGroupWidget) ScrollToCursor() {
	if g.IsEditorActive() {
		g.Editor.scrollViewport()
	}
}

func (g *EditorGroupWidget) ClearSearch() {
	if !g.IsEditorActive() {
		return
	}
	g.Editor.SearchQuery = ""
	g.Editor.SearchMatches = nil
	g.Editor.SearchActive = 0
}

func (g *EditorGroupWidget) SetDiagnostics(path string, diags []Diagnostic) {
	t := g.activeTab()
	if t != nil && t.FilePath == path {
		g.Editor.Diagnostics = diags
	}
}

func (g *EditorGroupWidget) FindNext() {
	if !g.IsEditorActive() || len(g.Editor.SearchMatches) == 0 {
		return
	}
	cur := g.Editor.SearchActive
	cur = (cur + 1) % len(g.Editor.SearchMatches)
	g.Editor.SearchActive = cur
	m := g.Editor.SearchMatches[cur]
	g.Editor.Cursor.Line = m.Line
	g.Editor.Cursor.Col = m.Col
	g.Editor.scrollViewport()
}

func (g *EditorGroupWidget) ReplaceMatch(match FindMatch, replacement string) {
	if !g.IsEditorActive() {
		return
	}
	runes := []rune(g.Editor.Buf.Lines[match.Line])
	endCol := match.Col + match.Len
	if endCol > len(runes) {
		endCol = len(runes)
	}
	g.Editor.exec(&undo.DeleteSelectionCommand{
		StartLine: match.Line, StartCol: match.Col,
		EndLine: match.Line, EndCol: endCol,
	})
	if replacement != "" {
		g.Editor.exec(&undo.InsertStringCommand{
			Line: match.Line, Col: match.Col, Text: replacement,
		})
	}
	g.Editor.Cursor.Line = match.Line
	g.Editor.Cursor.Col = match.Col + len([]rune(replacement))
	g.Editor.scrollViewport()
}

func (g *EditorGroupWidget) ReplaceAll(query, replacement string) {
	if !g.IsEditorActive() || query == "" {
		return
	}
	matches, _ := FindInLines(g.Editor.Buf.Lines, query, SearchOptions{})
	for i := len(matches) - 1; i >= 0; i-- {
		g.ReplaceMatch(matches[i], replacement)
	}
}

func (g *EditorGroupWidget) FindPrev() {
	if !g.IsEditorActive() || len(g.Editor.SearchMatches) == 0 {
		return
	}
	cur := g.Editor.SearchActive
	cur = (cur - 1 + len(g.Editor.SearchMatches)) % len(g.Editor.SearchMatches)
	g.Editor.SearchActive = cur
	m := g.Editor.SearchMatches[cur]
	g.Editor.Cursor.Line = m.Line
	g.Editor.Cursor.Col = m.Col
	g.Editor.scrollViewport()
}

func (g *EditorGroupWidget) MoveLineUp() {
	if g.IsEditorActive() {
		g.Editor.MoveLineUp()
	}
}

func (g *EditorGroupWidget) MoveLineDown() {
	if g.IsEditorActive() {
		g.Editor.MoveLineDown()
	}
}

func (g *EditorGroupWidget) DuplicateLine() {
	if g.IsEditorActive() {
		g.Editor.DuplicateLine()
	}
}

func (g *EditorGroupWidget) DeleteLine() {
	if g.IsEditorActive() {
		g.Editor.DeleteLine()
	}
}

func (g *EditorGroupWidget) InsertLineBelow() {
	if g.IsEditorActive() {
		g.Editor.InsertLineBelow()
	}
}

func (g *EditorGroupWidget) InsertLineAbove() {
	if g.IsEditorActive() {
		g.Editor.InsertLineAbove()
	}
}

func (g *EditorGroupWidget) ToggleLineComment() {
	if g.IsEditorActive() {
		g.Editor.ToggleLineComment()
	}
}

func (g *EditorGroupWidget) DeleteWordLeft() {
	if g.IsEditorActive() {
		g.Editor.DeleteWordLeft()
	}
}

func (g *EditorGroupWidget) DeleteWordRight() {
	if g.IsEditorActive() {
		g.Editor.DeleteWordRight()
	}
}

func (g *EditorGroupWidget) Copy() {
	t := g.activeTab()
	if t == nil || t.Content != nil || t.Sel == nil || !t.Sel.Active {
		return
	}
	text := t.Sel.Text(t.Buf.Lines, t.Cur.Line, t.Cur.Col)
	clipboard.Set(text)
}

func (g *EditorGroupWidget) Cut() {
	t := g.activeTab()
	if t == nil || t.Content != nil || t.Sel == nil || !t.Sel.Active {
		return
	}
	text := t.Sel.Text(t.Buf.Lines, t.Cur.Line, t.Cur.Col)
	clipboard.Set(text)
	g.Editor.deleteSelection()
}

func (g *EditorGroupWidget) Paste() {
	if !g.IsEditorActive() {
		return
	}
	text := clipboard.Get()
	if text == "" {
		return
	}
	g.Editor.pasteText(text)
}

func (g *EditorGroupWidget) syncTabs() {
	t := g.activeTab()
	if t == nil {
		g.TabBar.SetTabs(nil)
		return
	}
	if t.Content == nil {
		g.Editor.Buf = t.Buf
		g.Editor.Cursor = t.Cur
		g.Editor.Viewport = t.Vp
		g.Editor.Undo = t.Undo
		g.Editor.Selection = t.Sel
		g.Editor.Highlighter = t.Highlighter
		if t.TabSize > 0 {
			g.Editor.TabSize = t.TabSize
		}
	}
	var uiTabs []Tab
	for i, ts := range g.tabs {
		dirty := false
		if ts.Buf != nil {
			dirty = ts.Buf.Dirty
		}
		closable := true
		if len(g.tabs) == 1 && ts.FilePath == "untitled" && ts.Buf != nil && !ts.Buf.Dirty && len(ts.Buf.Lines) <= 1 && (len(ts.Buf.Lines) == 0 || ts.Buf.Lines[0] == "") {
			closable = false
		}
		uiTabs = append(uiTabs, Tab{
			Name:     ts.FilePath,
			Active:   i == g.active,
			Dirty:    dirty,
			Closable: closable,
		})
	}
	g.TabBar.SetTabs(uiTabs)
}

func (g *EditorGroupWidget) Render(surface *RenderSurface) {
	g.syncTabs()
	w, h := surface.Size()
	r := g.GetRect()

	const tabBarH = 3
	if h <= tabBarH {
		return
	}

	g.TabBar.SetRect(Rect{X: r.X, Y: r.Y, W: r.W, H: tabBarH})
	tabSurface := surface.Sub(Rect{X: 0, Y: 0, W: w, H: tabBarH})
	g.TabBar.Render(tabSurface)

	contentH := h - tabBarH
	contentRect := Rect{X: r.X, Y: r.Y + tabBarH, W: r.W, H: contentH}
	contentSurface := surface.Sub(Rect{X: 0, Y: tabBarH, W: w, H: contentH})

	t := g.activeTab()
	if t == nil {
		return
	}
	if t.Content != nil {
		t.Content.SetRect(contentRect)
		t.Content.Render(contentSurface)
	} else {
		g.Editor.SetRect(contentRect)
		g.Editor.Render(contentSurface)
	}

	if g.SignatureHelp != nil && g.SignatureHelp.Label != "" {
		g.SignatureHelp.AnchorX = g.Editor.CursorX - r.X
		g.SignatureHelp.AnchorY = g.Editor.CursorY - r.Y
		g.SignatureHelp.Borders = g.Borders
		g.SignatureHelp.Render(surface)
	}

	if g.Autocomplete != nil && len(g.Autocomplete.Items) > 0 {
		g.Autocomplete.AnchorX = g.Editor.CursorX - r.X
		g.Autocomplete.AnchorY = g.Editor.CursorY - r.Y
		g.Autocomplete.Borders = g.Borders
		g.Autocomplete.Render(surface)
	}

	if g.Hover != nil && len(g.Hover.Lines) > 0 {
		g.Hover.OffsetX = r.X
		g.Hover.OffsetY = r.Y
		g.Hover.Borders = g.Borders
		g.Hover.Render(surface)
	}
}

func (g *EditorGroupWidget) HandleEvent(ev tcell.Event) EventResult {
	if g.Hover != nil {
		result := g.Hover.HandleEvent(ev)
		if result == EventDismissed {
			g.Hover = nil
		}
	}
	if g.SignatureHelp != nil {
		if kev, ok := ev.(*tcell.EventKey); ok && kev.Key() == tcell.KeyEscape && g.Autocomplete == nil {
			g.SignatureHelp = nil
			return EventConsumed
		}
	}
	if g.Autocomplete != nil {
		result := g.Autocomplete.HandleEvent(ev)
		if result == EventConsumed {
			return EventConsumed
		}
	}
	result := g.TabBar.HandleEvent(ev)
	slog.Debug("editorGroup", "tabBarResult", result)
	if result == EventConsumed {
		return EventConsumed
	}
	t := g.activeTab()
	if t == nil {
		return EventIgnored
	}
	if t.Content != nil {
		return t.Content.HandleEvent(ev)
	}
	return g.Editor.HandleEvent(ev)
}
