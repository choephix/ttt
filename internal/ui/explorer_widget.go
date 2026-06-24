package ui

import (
	"path/filepath"
	"strings"

	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type TreeNode struct {
	Name       string
	Path       string
	IsDir      bool
	Expanded   bool
	Children   []*TreeNode
	Depth      int
	GitIgnored bool
}

type ExplorerWidget struct {
	BaseWidget
	SelectableList
	Roots        []*TreeNode
	FlatList     []*TreeNode
	ActiveFile   string
	Settings     config.ExplorerSettings
	OnOpenFile   func(path string)
	OnRightClick func(node *TreeNode, screenX, screenY int)
	OnRootMenu   func(node *TreeNode, screenX, screenY int)
	scrollbar Scrollbar
}

func NewExplorerWidget(settings config.ExplorerSettings, rootPaths ...string) *ExplorerWidget {
	e := &ExplorerWidget{Settings: settings}
	multiRoot := len(rootPaths) > 1
	for _, p := range rootPaths {
		root := &TreeNode{
			Name:     filepath.Base(p),
			Path:     p,
			IsDir:    true,
			Expanded: !multiRoot,
			Depth:    0,
		}
		e.loadChildren(root)
		e.Roots = append(e.Roots, root)
	}
	e.flatten()
	return e
}

func (e *ExplorerWidget) Focusable() bool { return true }

func (e *ExplorerWidget) SelectedNode() *TreeNode {
	if e.Selected >= 0 && e.Selected < len(e.FlatList) {
		return e.FlatList[e.Selected]
	}
	return nil
}

func (e *ExplorerWidget) IsRoot(node *TreeNode) bool {
	for _, root := range e.Roots {
		if root == node {
			return true
		}
	}
	return false
}

func (e *ExplorerWidget) Reload() {
	for _, root := range e.Roots {
		e.loadChildren(root)
		e.reloadExpanded(root)
	}
	e.flatten()
	e.ClampSelected(len(e.FlatList))
}

func (e *ExplorerWidget) AddRoot(path string) {
	root := &TreeNode{
		Name:     filepath.Base(path),
		Path:     path,
		IsDir:    true,
		Expanded: true,
		Depth:    0,
	}
	e.loadChildren(root)
	e.Roots = append(e.Roots, root)
	e.flatten()
}

func (e *ExplorerWidget) RemoveRoot(path string) {
	for i, root := range e.Roots {
		if root.Path == path {
			e.Roots = append(e.Roots[:i], e.Roots[i+1:]...)
			e.flatten()
			e.ClampSelected(len(e.FlatList))
			return
		}
	}
}

func (e *ExplorerWidget) reloadExpanded(node *TreeNode) {
	for _, child := range node.Children {
		if child.IsDir && child.Expanded {
			e.loadChildren(child)
			e.reloadExpanded(child)
		}
	}
}

func (e *ExplorerWidget) loadChildren(node *TreeNode) {
	entries := LoadDirEntries(node.Path, e.Settings)
	node.Children = nil
	for _, de := range entries {
		child := &TreeNode{
			Name:       de.Name,
			Path:       de.Path,
			IsDir:      de.IsDir,
			Depth:      node.Depth + 1,
			GitIgnored: de.GitIgnored,
		}
		node.Children = append(node.Children, child)
	}
}

func (e *ExplorerWidget) flatten() {
	e.FlatList = nil
	for _, root := range e.Roots {
		e.flattenNode(root)
	}
}

func (e *ExplorerWidget) flattenNode(node *TreeNode) {
	e.FlatList = append(e.FlatList, node)
	if node.IsDir && node.Expanded {
		for _, child := range node.Children {
			e.flattenNode(child)
		}
	}
}

func (e *ExplorerWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	surface.Fill(term.Cell{Ch: ' '})

	visibleHeight := h
	if visibleHeight <= 0 {
		return
	}
	e.EnsureVisible(visibleHeight)

	r := e.GetRect()
	e.scrollbar.X = r.X + w - 1
	e.scrollbar.Y = r.Y
	e.scrollbar.Height = visibleHeight
	e.scrollbar.TotalItems = len(e.FlatList)
	e.scrollbar.TopItem = e.ScrollTop

	for i := 0; i < visibleHeight; i++ {
		idx := e.ScrollTop + i
		if idx >= len(e.FlatList) {
			break
		}
		node := e.FlatList[idx]
		y := i

		style := term.StyleDefault
		if idx == e.Selected {
			style = term.StyleSidebarSelected
		} else if !node.IsDir && node.Path == e.ActiveFile {
			style = term.StyleSidebarSelected
		}

		// Fill background for selected item
		for x := 0; x < w; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}

		// Indent
		indent := node.Depth * 2
		x := indent

		// Chevron for dirs
		if node.IsDir {
			chevron := '▶'
			if node.Expanded {
				chevron = '▼'
			}
			if x < w {
				surface.SetCell(x, y, term.Cell{Ch: chevron, Style: style})
			}
			x++
			if x < w {
				surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
			}
			x++
		}

		// Name
		nameStyle := style
		if idx != e.Selected && (strings.HasPrefix(node.Name, ".") || node.GitIgnored) {
			nameStyle = term.StyleMuted
		}
		for _, ch := range node.Name {
			if x >= w {
				break
			}
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: nameStyle})
			x++
		}

		if e.IsRoot(node) && w >= 3 {
			surface.SetCell(w-2, y, term.Cell{Ch: '⋮', Style: style})
		}
	}

	e.scrollbar.Render(surface, w-1, 0)
}

func (e *ExplorerWidget) HandleEvent(ev tcell.Event) EventResult {
	if newTop, consumed := e.scrollbar.HandleEvent(ev); consumed {
		e.ScrollTop = newTop
		if e.scrollbar.IsDragging() {
			return EventCaptured
		}
		return EventConsumed
	}

	r := e.GetRect()

	if tev, ok := ev.(*tcell.EventMouse); ok {
		if tev.Buttons()&tcell.Button1 != 0 {
			mx, my := tev.Position()
			idx := e.ScrollTop + (my - r.Y)
			if idx >= 0 && idx < len(e.FlatList) && mx >= r.X+r.W-3 {
				node := e.FlatList[idx]
				if e.IsRoot(node) && e.OnRootMenu != nil {
					e.Selected = idx
					e.OnRootMenu(node, mx, my)
					return EventConsumed
				}
			}
		}
	}

	res := e.SelectableList.HandleListEvent(ev, r, len(e.FlatList))
	if res.Result == EventConsumed {
		switch res.Action {
		case ListActionActivate:
			e.ActivateSelected()
		case ListActionContext:
			if e.OnRightClick != nil && e.Selected >= 0 && e.Selected < len(e.FlatList) {
				e.OnRightClick(e.FlatList[e.Selected], res.ScreenX, res.ScreenY)
			}
		}
		return EventConsumed
	}

	if tev, ok := ev.(*tcell.EventKey); ok {
		switch tev.Key() {
		case tcell.KeyRune:
			if tev.Rune() == ' ' {
				e.ActivateSelected()
				return EventConsumed
			}
		case tcell.KeyEnter:
			if tev.Modifiers()&tcell.ModShift != 0 {
				if e.OnRightClick != nil && e.Selected >= 0 && e.Selected < len(e.FlatList) {
					r := e.GetRect()
					e.OnRightClick(e.FlatList[e.Selected], r.X, r.Y+e.Selected-e.ScrollTop)
				}
				return EventConsumed
			}
		case tcell.KeyLeft:
			e.collapseSelected()
			return EventConsumed
		case tcell.KeyRight:
			e.expandSelected()
			return EventConsumed
		}
	}
	return EventIgnored
}

func (e *ExplorerWidget) ActivateSelected() {
	if e.Selected < 0 || e.Selected >= len(e.FlatList) {
		return
	}
	node := e.FlatList[e.Selected]
	if node.IsDir {
		node.Expanded = !node.Expanded
		if node.Expanded && len(node.Children) == 0 {
			e.loadChildren(node)
		}
		e.flatten()
	} else if e.OnOpenFile != nil {
		e.OnOpenFile(node.Path)
	}
}


func (e *ExplorerWidget) collapseSelected() {
	if e.Selected < 0 || e.Selected >= len(e.FlatList) {
		return
	}
	node := e.FlatList[e.Selected]
	if node.IsDir && node.Expanded {
		node.Expanded = false
		e.flatten()
	}
}

func (e *ExplorerWidget) expandSelected() {
	if e.Selected < 0 || e.Selected >= len(e.FlatList) {
		return
	}
	node := e.FlatList[e.Selected]
	if node.IsDir && !node.Expanded {
		node.Expanded = true
		if len(node.Children) == 0 {
			e.loadChildren(node)
		}
		e.flatten()
	}
}
