package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func makeTestTree() *ExplorerWidget {
	root := &TreeNode{
		Name:     "project",
		Path:     "/tmp/project",
		IsDir:    true,
		Expanded: true,
		Depth:    0,
		Children: []*TreeNode{
			{Name: "cmd", Path: "/tmp/project/cmd", IsDir: true, Depth: 1, Children: []*TreeNode{
				{Name: "main.go", Path: "/tmp/project/cmd/main.go", IsDir: false, Depth: 2},
			}},
			{Name: "go.mod", Path: "/tmp/project/go.mod", IsDir: false, Depth: 1},
		},
	}

	e := &ExplorerWidget{Roots: []*TreeNode{root}}
	e.flatten()
	return e
}

func TestExplorerFlatten(t *testing.T) {
	e := makeTestTree()

	// Root expanded, cmd collapsed by default
	// project -> cmd -> go.mod
	if len(e.FlatList) != 3 {
		t.Fatalf("expected 3 nodes in flat list, got %d", len(e.FlatList))
	}
	if e.FlatList[0].Name != "project" {
		t.Fatalf("expected 'project', got '%s'", e.FlatList[0].Name)
	}
}

func TestExplorerExpandCollapse(t *testing.T) {
	e := makeTestTree()
	// Select "cmd" (index 1), expand it
	e.Selected = 1
	e.expandSelected()

	if len(e.FlatList) != 4 {
		t.Fatalf("after expanding cmd, expected 4 nodes, got %d", len(e.FlatList))
	}
	if e.FlatList[2].Name != "main.go" {
		t.Fatalf("expected 'main.go' at index 2, got '%s'", e.FlatList[2].Name)
	}

	// Collapse it
	e.collapseSelected()
	if len(e.FlatList) != 3 {
		t.Fatalf("after collapsing cmd, expected 3 nodes, got %d", len(e.FlatList))
	}
}

func TestExplorerNavigation(t *testing.T) {
	e := makeTestTree()

	e.HandleEvent(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	if e.Selected != 1 {
		t.Fatalf("expected selected 1, got %d", e.Selected)
	}

	e.HandleEvent(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	if e.Selected != 2 {
		t.Fatalf("expected selected 2, got %d", e.Selected)
	}

	// Can't go past end
	e.HandleEvent(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	if e.Selected != 2 {
		t.Fatalf("expected selected 2 (clamped), got %d", e.Selected)
	}

	e.HandleEvent(tcell.NewEventKey(tcell.KeyUp, 0, 0))
	if e.Selected != 1 {
		t.Fatalf("expected selected 1, got %d", e.Selected)
	}
}

func TestExplorerOpenFile(t *testing.T) {
	e := makeTestTree()
	opened := ""
	e.OnOpenFile = func(path string) { opened = path }

	// Select go.mod (index 2)
	e.Selected = 2
	e.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))

	if opened != "/tmp/project/go.mod" {
		t.Fatalf("expected '/tmp/project/go.mod', got '%s'", opened)
	}
}

func TestSidebarPanelSwitching(t *testing.T) {
	sidebar := NewSidebarWidget()
	explorer := makeTestTree()
	search := NewSearchWidget()

	sidebar.AddPanel("explorer", "EXPLORER", explorer)
	sidebar.AddPanel("search", "SEARCH", search)

	if sidebar.ActivePanel != "explorer" {
		t.Fatal("first added panel should be active")
	}

	sidebar.SetActivePanel("search")
	if sidebar.ActivePanel != "search" {
		t.Fatal("should have switched to search")
	}

	sidebar.SetActivePanel("nonexistent")
	if sidebar.ActivePanel != "search" {
		t.Fatal("switching to nonexistent should not change active panel")
	}
}
