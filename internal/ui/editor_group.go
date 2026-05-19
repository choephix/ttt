package ui

import (
	"macro/internal/core/buffer"
	"macro/internal/core/cursor"
	"macro/internal/term"
	"macro/internal/view"

	"github.com/gdamore/tcell/v2"
)

type editorTab struct {
	FilePath string
	Buf      *buffer.Buffer
	Cur      *cursor.Cursor
	Vp       *view.Viewport
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
	g.tabs = []editorTab{{
		FilePath: "untitled",
		Buf:      editor.Buf,
		Cur:      editor.Cursor,
		Vp:       editor.Viewport,
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
	g.tabs = append(g.tabs, editorTab{
		FilePath: path,
		Buf:      newBuf,
		Cur:      &cursor.Cursor{},
		Vp:       &view.Viewport{},
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

func (g *EditorGroupWidget) syncTabs() {
	t := g.tabs[g.active]
	g.Editor.Buf = t.Buf
	g.Editor.Cursor = t.Cur
	g.Editor.Viewport = t.Vp
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
