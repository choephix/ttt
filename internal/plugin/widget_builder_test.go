package plugin

import (
	"testing"

	"github.com/eugenioenko/ttt/internal/widgets"
	lua "github.com/yuin/gopher-lua"
)

func TestReconcileCreatesWidgets(t *testing.T) {
	ws := NewWidgetState()
	p := &Plugin{Name: "test", State: lua.NewState()}
	defer p.State.Close()

	descs := []WidgetDesc{
		{Kind: WidgetLabel, Key: "label:0", Text: "Hello"},
		{Kind: WidgetTree, Key: "tree:0", Items: []*widgets.TreeNode{
			{ID: "a", Label: "Alpha"},
		}},
		{Kind: WidgetButton, Key: "button:0", Label: "OK"},
	}

	root := ws.Reconcile(descs, p)
	if root == nil {
		t.Fatal("expected non-nil root")
	}
	if len(root.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(root.Children))
	}

	if _, ok := root.Children[0].(*widgets.LabelWidget); !ok {
		t.Error("expected child 0 to be LabelWidget")
	}
	if _, ok := root.Children[1].(*widgets.TreeWidget); !ok {
		t.Error("expected child 1 to be TreeWidget")
	}
	if _, ok := root.Children[2].(*widgets.ButtonWidget); !ok {
		t.Error("expected child 2 to be ButtonWidget")
	}
}

func TestReconcilePreservesTreeState(t *testing.T) {
	ws := NewWidgetState()
	p := &Plugin{Name: "test", State: lua.NewState()}
	defer p.State.Close()

	descs := []WidgetDesc{
		{Kind: WidgetTree, Key: "tree:0", Items: []*widgets.TreeNode{
			{ID: "a", Label: "Alpha", Expandable: true, Children: []*widgets.TreeNode{
				{ID: "a1", Label: "Alpha-1"},
			}},
			{ID: "b", Label: "Beta"},
		}},
	}

	ws.Reconcile(descs, p)

	tw := ws.items[0].(*widgets.TreeWidget)
	tw.Config.Items[0].Expanded = true

	descs2 := []WidgetDesc{
		{Kind: WidgetTree, Key: "tree:0", Items: []*widgets.TreeNode{
			{ID: "a", Label: "Alpha Updated", Expandable: true, Children: []*widgets.TreeNode{
				{ID: "a1", Label: "Alpha-1"},
				{ID: "a2", Label: "Alpha-2"},
			}},
			{ID: "b", Label: "Beta"},
		}},
	}

	ws.Reconcile(descs2, p)

	tw2 := ws.items[0].(*widgets.TreeWidget)
	if !tw2.Config.Items[0].Expanded {
		t.Error("expected node 'a' to remain expanded after reconcile")
	}
	if tw2.Config.Items[0].Label != "Alpha Updated" {
		t.Error("expected label to be updated")
	}
}

func TestReconcilePreservesInputText(t *testing.T) {
	ws := NewWidgetState()
	p := &Plugin{Name: "test", State: lua.NewState()}
	defer p.State.Close()

	descs := []WidgetDesc{
		{Kind: WidgetInput, Key: "input:0", Placeholder: "Type..."},
	}

	ws.Reconcile(descs, p)

	iw := ws.items[0].(*widgets.InputWidget)
	iw.SetText("user typed this")

	descs2 := []WidgetDesc{
		{Kind: WidgetInput, Key: "input:0", Placeholder: "New placeholder"},
	}

	ws.Reconcile(descs2, p)

	iw2 := ws.items[0].(*widgets.InputWidget)
	if iw2.Text() != "user typed this" {
		t.Errorf("expected text preserved, got %q", iw2.Text())
	}
	if iw2.Config.Placeholder != "New placeholder" {
		t.Errorf("expected placeholder updated, got %q", iw2.Config.Placeholder)
	}
}

func TestReconcileHandlesTypeChange(t *testing.T) {
	ws := NewWidgetState()
	p := &Plugin{Name: "test", State: lua.NewState()}
	defer p.State.Close()

	descs1 := []WidgetDesc{
		{Kind: WidgetLabel, Key: "label:0", Text: "Hello"},
		{Kind: WidgetTree, Key: "tree:0"},
	}
	ws.Reconcile(descs1, p)

	descs2 := []WidgetDesc{
		{Kind: WidgetLabel, Key: "label:0", Text: "Hello"},
		{Kind: WidgetButton, Key: "button:0", Label: "Click"},
	}
	ws.Reconcile(descs2, p)

	if _, ok := ws.items[1].(*widgets.ButtonWidget); !ok {
		t.Error("expected child 1 to be replaced with ButtonWidget")
	}
}

func TestReconcileEmptyDescriptors(t *testing.T) {
	ws := NewWidgetState()
	p := &Plugin{Name: "test", State: lua.NewState()}
	defer p.State.Close()

	root := ws.Reconcile(nil, p)
	if root == nil {
		t.Fatal("expected non-nil root even with empty descriptors")
	}
	if len(root.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(root.Children))
	}
}
