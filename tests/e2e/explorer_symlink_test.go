package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v3"
)

// findLabel returns the column and row of text on screen. Explorer labels are
// plain ASCII at the left of the row, so a cell's column and its offset in the
// reassembled row string line up; the right-aligned glyphs that could be wide
// sit past the label and cannot shift it.
func findLabel(t *testing.T, cells []term.SimCell, w, h int, text string) (int, int) {
	t.Helper()
	for y := range h {
		if x := strings.Index(rowString(cells[y*w:(y+1)*w]), text); x >= 0 {
			return x, y
		}
	}
	t.Fatalf("%q not found on screen", text)
	return 0, 0
}

// The unit tests assert the semantic term.StyleSymlink slot, which stays green
// even if the theme never gives that slot a colour. This covers the rest of the
// chain: ResolveColors -> BuildStyleMap -> the cell the explorer actually draws.
func TestExplorerRendersSymlinkInSymlinkColour(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	if err := os.Symlink(filepath.Join(h.dir, "alpha.txt"), filepath.Join(h.dir, "linked.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	h.exec("sidebar.explorer")
	// Keep the selection on the root so the symlink row renders with its own
	// label style rather than the selection style.
	h.app.Explorer.Tree.SetSelectedIndex(0)
	h.pressRune('r')
	h.redraw()

	cells, w, ht := h.screen.GetContents()
	linkX, linkY := findLabel(t, cells, w, ht, "linked.txt")
	plainX, plainY := findLabel(t, cells, w, ht, "alpha.txt")

	theme := config.DefaultTheme()
	theme.ResolveColors()
	want := tcell.GetColor(theme.Sidebar.Symlink.Fg)

	if got := cells[linkY*w+linkX].Style.GetForeground(); got != want {
		t.Errorf("symlink label fg = %v, want %v", got, want)
	}
	if got := cells[plainY*w+plainX].Style.GetForeground(); got == want {
		t.Errorf("regular file label must not use the symlink colour, got %v", got)
	}
	if row := rowString(cells[linkY*w : (linkY+1)*w]); !strings.Contains(row, "→") {
		t.Errorf("symlink row is missing the → glyph: %q", row)
	}
}
