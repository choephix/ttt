package ui

import (
	"ttt/internal/term"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type TreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Expanded bool
	Children []*TreeNode
	Depth    int
}

type ExplorerWidget struct {
	BaseWidget
	SelectableList
	Root       *TreeNode
	FlatList   []*TreeNode
	ActiveFile string
	OnOpenFile   func(path string)
	OnRightClick func(node *TreeNode, screenX, screenY int)
}

func NewExplorerWidget(rootPath string) *ExplorerWidget {
	e := &ExplorerWidget{}
	e.Root = &TreeNode{
		Name:     filepath.Base(rootPath),
		Path:     rootPath,
		IsDir:    true,
		Expanded: true,
		Depth:    0,
	}
	e.loadChildren(e.Root)
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

func (e *ExplorerWidget) Reload() {
	e.loadChildren(e.Root)
	e.reloadExpanded(e.Root)
	e.flatten()
	e.ClampSelected(len(e.FlatList))
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
	entries, err := os.ReadDir(node.Path)
	if err != nil {
		return
	}

	node.Children = nil
	dirs := []*TreeNode{}
	files := []*TreeNode{}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		child := &TreeNode{
			Name:  entry.Name(),
			Path:  filepath.Join(node.Path, entry.Name()),
			IsDir: entry.IsDir(),
			Depth: node.Depth + 1,
		}
		if entry.IsDir() {
			dirs = append(dirs, child)
		} else {
			files = append(files, child)
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	node.Children = append(dirs, files...)
}

func (e *ExplorerWidget) flatten() {
	e.FlatList = nil
	e.flattenNode(e.Root)
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
		} else {
			x += 2
		}

		// Name
		for _, ch := range node.Name {
			if x >= w {
				break
			}
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
			x++
		}
	}
}

func (e *ExplorerWidget) HandleEvent(ev tcell.Event) EventResult {
	r := e.GetRect()
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
