package ui

import (
	"log/slog"
	"ttt/internal/config"
	"ttt/internal/core/buffer"
	"ttt/internal/core/clipboard"
	"ttt/internal/core/cursor"
	"ttt/internal/core/diff"
	"ttt/internal/core/highlight"
	"ttt/internal/core/selection"
	"ttt/internal/core/undo"
	"ttt/internal/term"
	"ttt/internal/view"

	"github.com/gdamore/tcell/v2"
)

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
}

type EditorGroupWidget struct {
	BaseWidget
	TabBar       *TabBarWidget
	Editor       *EditorPaneWidget
	tabs         []editorTab
	active       int
	TabSize      int
	LineNumbers  bool
	Borders      *term.BorderSet
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

func (g *EditorGroupWidget) OpenFile(path string) {
	for i, t := range g.tabs {
		if t.FilePath == path {
			g.SwitchTab(i)
			return
		}
	}
	newBuf := &buffer.Buffer{Lines: []string{""}}
	if err := newBuf.LoadFile(path); err != nil {
		return
	}
	tabSize := g.TabSize
	ec := config.LoadEditorConfig(path)
	if ec.IndentSize > 0 {
		tabSize = ec.IndentSize
	} else if detected := buffer.DetectIndent(newBuf.Lines); detected.Size > 0 {
		tabSize = detected.Size
	}
	g.tabs = append(g.tabs, editorTab{
		FilePath:    path,
		Buf:         newBuf,
		Cur:         &cursor.Cursor{},
		Vp:          &view.Viewport{},
		Undo:        &undo.UndoStack{},
		Sel:         &selection.Selection{},
		Highlighter: highlight.New(path),
		TabSize:     tabSize,
	})
	g.SwitchTab(len(g.tabs) - 1)
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
	g.tabs = append(g.tabs[:g.active], g.tabs[g.active+1:]...)
	if g.active >= len(g.tabs) {
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
	t.Buf.SaveFile(t.FilePath)
	return true
}

func (g *EditorGroupWidget) SaveAs(path string) {
	t := g.activeTab()
	if t == nil || t.Content != nil {
		return
	}
	t.FilePath = path
	t.Buf.SaveFile(path)
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

func (g *EditorGroupWidget) ClearSearch() {
	if !g.IsEditorActive() {
		return
	}
	g.Editor.SearchQuery = ""
	g.Editor.SearchMatches = nil
	g.Editor.SearchActive = 0
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
	matches := FindInLines(g.Editor.Buf.Lines, query)
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
		uiTabs = append(uiTabs, Tab{
			Name:   ts.FilePath,
			Active: i == g.active,
			Dirty:  dirty,
		})
	}
	g.TabBar.SetTabs(uiTabs)
}

func (g *EditorGroupWidget) Render(surface *RenderSurface) {
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
}

func (g *EditorGroupWidget) HandleEvent(ev tcell.Event) EventResult {
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
