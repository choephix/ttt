package ui

import (
	"macro/internal/config"
	"macro/internal/core/buffer"
	"macro/internal/core/clipboard"
	"macro/internal/core/cursor"
	"macro/internal/core/selection"
	"macro/internal/core/undo"
	"macro/internal/term"
	"macro/internal/view"

	"github.com/gdamore/tcell/v2"
)

type editorTab struct {
	FilePath string
	Buf      *buffer.Buffer
	Cur      *cursor.Cursor
	Vp       *view.Viewport
	Undo     *undo.UndoStack
	Sel      *selection.Selection
	TabSize  int
}

type EditorGroupWidget struct {
	BaseWidget
	TabBar  *TabBarWidget
	Editor  *EditorPaneWidget
	tabs    []editorTab
	active  int
	TabSize int
	Borders *term.BorderSet
}

func NewEditorGroupWidget(borders *term.BorderSet, tabSize int) *EditorGroupWidget {
	editor := NewEditorPaneWidget(
		&buffer.Buffer{Lines: []string{""}},
		&cursor.Cursor{},
		&view.Viewport{},
	)
	editor.TabSize = tabSize

	tabBar := NewTabBarWidget()
	tabBar.Borders = borders

	g := &EditorGroupWidget{
		TabBar:  tabBar,
		Editor:  editor,
		TabSize: tabSize,
		Borders: borders,
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
	}
	g.tabs = append(g.tabs, editorTab{
		FilePath: path,
		Buf:      newBuf,
		Cur:      &cursor.Cursor{},
		Vp:       &view.Viewport{},
		Undo:     &undo.UndoStack{},
		Sel:      &selection.Selection{},
		TabSize:  tabSize,
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
	if len(g.tabs) <= 1 {
		return
	}
	g.tabs = append(g.tabs[:g.active], g.tabs[g.active+1:]...)
	if g.active >= len(g.tabs) {
		g.active = len(g.tabs) - 1
	}
	g.syncTabs()
}

func (g *EditorGroupWidget) Save() {
	t := &g.tabs[g.active]
	if t.FilePath != "untitled" {
		t.Buf.SaveFile(t.FilePath)
	}
}

func (g *EditorGroupWidget) ActiveFilePath() string {
	return g.tabs[g.active].FilePath
}

func (g *EditorGroupWidget) ActiveCursor() (line, col int) {
	t := g.tabs[g.active]
	return t.Cur.Line, t.Cur.Col
}

func (g *EditorGroupWidget) IsDirty() bool {
	return g.tabs[g.active].Buf.Dirty
}

func (g *EditorGroupWidget) AnyDirty() bool {
	for _, t := range g.tabs {
		if t.Buf.Dirty {
			return true
		}
	}
	return false
}

func (g *EditorGroupWidget) Undo() {
	t := &g.tabs[g.active]
	if t.Undo != nil {
		t.Undo.Undo(t.Buf)
	}
}

func (g *EditorGroupWidget) Redo() {
	t := &g.tabs[g.active]
	if t.Undo != nil {
		t.Undo.Redo(t.Buf)
	}
}

func (g *EditorGroupWidget) SelectAll() {
	t := &g.tabs[g.active]
	if t.Sel == nil {
		return
	}
	t.Sel.Start(0, 0)
	lastLine := len(t.Buf.Lines) - 1
	t.Cur.Line = lastLine
	t.Cur.Col = len([]rune(t.Buf.Lines[lastLine]))
}

func (g *EditorGroupWidget) SetSearchQuery(query string) {
	g.Editor.SearchQuery = query
}

func (g *EditorGroupWidget) SetSearchActive(idx int) {
	g.Editor.SearchActive = idx
}

func (g *EditorGroupWidget) GoToLine(line int) {
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
	g.Editor.SearchQuery = ""
	g.Editor.SearchActive = 0
}

func (g *EditorGroupWidget) Copy() {
	t := &g.tabs[g.active]
	if t.Sel == nil || !t.Sel.Active {
		return
	}
	text := t.Sel.Text(t.Buf.Lines, t.Cur.Line, t.Cur.Col)
	clipboard.Set(text)
}

func (g *EditorGroupWidget) Cut() {
	t := &g.tabs[g.active]
	if t.Sel == nil || !t.Sel.Active {
		return
	}
	text := t.Sel.Text(t.Buf.Lines, t.Cur.Line, t.Cur.Col)
	clipboard.Set(text)
	g.Editor.deleteSelection()
}

func (g *EditorGroupWidget) Paste() {
	text := clipboard.Get()
	if text == "" {
		return
	}
	g.Editor.pasteText(text)
}

func (g *EditorGroupWidget) syncTabs() {
	t := g.tabs[g.active]
	g.Editor.Buf = t.Buf
	g.Editor.Cursor = t.Cur
	g.Editor.Viewport = t.Vp
	g.Editor.Undo = t.Undo
	g.Editor.Selection = t.Sel
	if t.TabSize > 0 {
		g.Editor.TabSize = t.TabSize
	}
	var uiTabs []Tab
	for i, ts := range g.tabs {
		uiTabs = append(uiTabs, Tab{
			Name:   ts.FilePath,
			Active: i == g.active,
			Dirty:  ts.Buf.Dirty,
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

	editorH := h - tabBarH
	g.Editor.SetRect(Rect{X: r.X, Y: r.Y + tabBarH, W: r.W, H: editorH})
	editorSurface := surface.Sub(Rect{X: 0, Y: tabBarH, W: w, H: editorH})
	g.Editor.Render(editorSurface)
}

func (g *EditorGroupWidget) HandleEvent(ev tcell.Event) EventResult {
	if g.TabBar.HandleEvent(ev) == EventConsumed {
		return EventConsumed
	}
	return g.Editor.HandleEvent(ev)
}
