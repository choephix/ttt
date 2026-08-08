package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/widgets"
)

func TestExplorerMarksSymlinkChildren(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real.txt")
	if err := os.WriteFile(real, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(root, "linked.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "nowhere"), filepath.Join(root, "dangling.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	n := NewNavigationPanel(config.DefaultExplorerSettings(), root)
	children := map[string]*widgets.TreeNode{}
	for _, child := range n.Tree.Config.Items[0].Children {
		children[child.Label] = child
	}

	cases := []struct {
		name  string
		style term.Style
		icon  string
	}{
		{"real.txt", term.StyleDefault, ""},
		{"linked.txt", term.StyleSymlink, "→"},
		{"dangling.txt", term.StyleDanger, "↛"},
	}
	for _, tc := range cases {
		child, ok := children[tc.name]
		if !ok {
			t.Fatalf("%s missing from explorer tree", tc.name)
		}
		if child.LabelStyle != tc.style {
			t.Errorf("%s: LabelStyle = %v, want %v", tc.name, child.LabelStyle, tc.style)
		}
		if child.RightIcon != tc.icon {
			t.Errorf("%s: RightIcon = %q, want %q", tc.name, child.RightIcon, tc.icon)
		}
	}
}
