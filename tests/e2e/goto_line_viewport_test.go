package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoToLineViewportPosition(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	var lines []string
	for i := 1; i <= 100; i++ {
		lines = append(lines, "line "+string(rune('0'+i/100))+string(rune('0'+(i/10)%10))+string(rune('0'+i%10)))
	}
	f := filepath.Join(h.dir, "big.txt")
	os.WriteFile(f, []byte(strings.Join(lines, "\n")+"\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	h.app.EditorGroup.GoToLine(50)
	h.redraw()

	topLine := h.app.EditorGroup.Editor.Viewport.TopLine
	cursorLine := h.app.EditorGroup.Editor.Cursor.Line
	vpHeight := h.app.EditorGroup.Editor.Viewport.Height

	if cursorLine != 49 {
		t.Fatalf("expected cursor on line 49 (0-indexed), got %d", cursorLine)
	}

	margin := vpHeight / 3
	expectedTop := cursorLine - margin
	if topLine != expectedTop {
		t.Errorf("expected TopLine ~%d (1/3 margin), got %d", expectedTop, topLine)
	}

	if cursorLine < topLine || cursorLine >= topLine+vpHeight {
		t.Errorf("cursor line %d not visible in viewport [%d, %d)", cursorLine, topLine, topLine+vpHeight)
	}
}
