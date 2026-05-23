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
	Root       *TreeNode
	FlatList   []*TreeNode
	Selected   int
	ScrollTop  int
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
	if e.Selected >= len(e.FlatList) {
		e.Selected = len(e.FlatList) - 1
	}
	if e.Selected < 0 {
		e.Selected = 0
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

	// Ensure scroll follows selected
	visibleHeight := h
	if visibleHeight <= 0 {
		return
	}
	if e.Selected < e.ScrollTop {
		e.ScrollTop = e.Selected
	}
	if e.Selected >= e.ScrollTop+visibleHeight {
		e.ScrollTop = e.Selected - visibleHeight + 1
	}

	for i := 0; i < visibleHeight; i++ {
		idx := e.ScrollTop + i
		if idx >= len(e.FlatList) {
			break
		}
		node := e.FlatList[idx]
		y := i

		style := term.StyleSidebarItem
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
	switch tev := ev.(type) {
	case *tcell.EventMouse:
		btn := tev.Buttons()
		if btn&tcell.Button1 != 0 {
			_, my := tev.Position()
			r := e.GetRect()
			localY := my - r.Y
			idx := e.ScrollTop + localY
			if idx >= 0 && idx < len(e.FlatList) {
				e.Selected = idx
				e.ActivateSelected()
			}
			return EventConsumed
		}
		if btn&tcell.Button3 != 0 && e.OnRightClick != nil {
			mx, my := tev.Position()
			r := e.GetRect()
			localY := my - r.Y
			idx := e.ScrollTop + localY
			if idx >= 0 && idx < len(e.FlatList) {
				e.Selected = idx
				e.OnRightClick(e.FlatList[idx], mx, my)
			}
			return EventConsumed
		}
		if btn&tcell.WheelUp != 0 {
			e.ScrollTop -= 3
			if e.ScrollTop < 0 {
				e.ScrollTop = 0
			}
			return EventConsumed
		}
		if btn&tcell.WheelDown != 0 {
			r := e.GetRect()
			max := len(e.FlatList) - r.H
			if max < 0 {
				max = 0
			}
			e.ScrollTop += 3
			if e.ScrollTop > max {
				e.ScrollTop = max
			}
			return EventConsumed
		}
	case *tcell.EventKey:
		switch tev.Key() {
		case tcell.KeyUp:
			if e.Selected > 0 {
				e.Selected--
			}
			return EventConsumed
		case tcell.KeyDown:
			if e.Selected < len(e.FlatList)-1 {
				e.Selected++
			}
			return EventConsumed
		case tcell.KeyEnter:
			e.ActivateSelected()
			return EventConsumed
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
