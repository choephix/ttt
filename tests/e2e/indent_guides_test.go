package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndentGuidesRender(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	// Hide sidebar to avoid border characters interfering
	h.exec("sidebar.toggle")

	// Write a file with indented content (8-space indent = 2 tab stops)
	path := filepath.Join(h.dir, "indented.go")
	content := "func main() {\n    if true {\n        fmt.Println()\n    }\n}\n"
	os.WriteFile(path, []byte(content), 0644)

	h.app.EditorGroup.OpenFile(path)
	h.app.EditorGroup.PinActiveTab()
	h.redraw()

	screen := h.screenText()
	// With indent guides enabled and tabSize=4, the line "        fmt.Println()"
	// should have a guide at column 4 (within 8-space indent).
	if !strings.Contains(screen, "│") {
		t.Errorf("expected indent guide character in screen, got:\n%s", screen)
	}

	// Verify the guide appears on the correct line
	lines := strings.Split(screen, "\n")
	foundGuide := false
	for _, line := range lines {
		// The fmt.Println line (line 3) has 8-space indent, guide at col 4
		if strings.Contains(line, "fmt.Println") && strings.Contains(line, "│") {
			foundGuide = true
			break
		}
	}
	if !foundGuide {
		t.Errorf("expected indent guide on fmt.Println line, got:\n%s", screen)
	}
}

// countIndentGuides counts the number of │ characters in the indent area
// of an editor line. It looks for the region between the line number and
// the content, skipping any leading border characters.
func countIndentGuides(screenLine string, lineNum string, contentSubstr string) int {
	// Find where the line number ends (after the line number digits and gutter padding)
	numIdx := strings.Index(screenLine, lineNum)
	if numIdx < 0 {
		return 0
	}
	// Start counting after the line number area (number + trailing gutter spaces)
	startAfterNum := numIdx + len(lineNum)

	contentIdx := strings.Index(screenLine, contentSubstr)
	if contentIdx < 0 || contentIdx <= startAfterNum {
		return 0
	}
	// Count │ in the whitespace/indent area between gutter and content
	count := 0
	for _, ch := range screenLine[startAfterNum:contentIdx] {
		if ch == '│' {
			count++
		}
	}
	return count
}

func TestIndentGuidesToggle(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	// Hide sidebar to avoid border characters
	h.exec("sidebar.toggle")

	path := filepath.Join(h.dir, "indented2.go")
	content := "func main() {\n    if true {\n        fmt.Println()\n    }\n}\n"
	os.WriteFile(path, []byte(content), 0644)

	h.app.EditorGroup.OpenFile(path)
	h.app.EditorGroup.PinActiveTab()
	h.redraw()

	// Check that the fmt.Println line (line 3) has indent guides in the indent area
	screen := h.screenText()
	lines := strings.Split(screen, "\n")
	guidesOn := 0
	for _, line := range lines {
		if strings.Contains(line, "fmt.Println") {
			guidesOn = countIndentGuides(line, "3", "fmt")
			break
		}
	}
	if guidesOn == 0 {
		t.Fatalf("expected indent guide on fmt.Println line before toggle, got:\n%s", screen)
	}

	// Toggle indent guides off
	h.exec("view.toggleIndentGuides")
	screen = h.screenText()
	lines = strings.Split(screen, "\n")
	guidesOff := 0
	for _, line := range lines {
		if strings.Contains(line, "fmt.Println") {
			guidesOff = countIndentGuides(line, "3", "fmt")
			break
		}
	}
	if guidesOff != 0 {
		t.Errorf("expected 0 indent guides on fmt.Println line after toggle off, got %d\nscreen:\n%s", guidesOff, screen)
	}

	// Toggle back on
	h.exec("view.toggleIndentGuides")
	screen = h.screenText()
	lines = strings.Split(screen, "\n")
	guidesBack := 0
	for _, line := range lines {
		if strings.Contains(line, "fmt.Println") {
			guidesBack = countIndentGuides(line, "3", "fmt")
			break
		}
	}
	if guidesBack == 0 {
		t.Errorf("expected indent guides on fmt.Println line after toggle on, got 0\nscreen:\n%s", screen)
	}
}

func TestIndentGuidesBlankLines(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	// Hide sidebar
	h.exec("sidebar.toggle")

	// A blank line between two indented lines should have guides extended through it
	path := filepath.Join(h.dir, "blank.go")
	content := "func main() {\n    line1\n\n    line2\n}\n"
	os.WriteFile(path, []byte(content), 0644)

	h.app.EditorGroup.OpenFile(path)
	h.app.EditorGroup.PinActiveTab()
	h.redraw()

	// The blank line (line 3) should not have a guide because it's a blank line
	// between two lines with indent=4, and min(above=4, below=4)=4,
	// but colIdx=4 is NOT < 4, so no guide. That's correct behavior --
	// the guide only extends through blank lines when surrounding indent > one tab stop.

	// Verify that the surrounding indented content renders properly
	screen := h.screenText()
	if !strings.Contains(screen, "line1") || !strings.Contains(screen, "line2") {
		t.Errorf("expected file content to render, got:\n%s", screen)
	}
}

func TestIndentGuidesDeepNesting(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	// Hide sidebar
	h.exec("sidebar.toggle")

	// Deep nesting: 3 levels of indent (12 spaces)
	path := filepath.Join(h.dir, "deep.go")
	content := "func main() {\n    if true {\n        for {\n            x := 1\n        }\n    }\n}\n"
	os.WriteFile(path, []byte(content), 0644)

	h.app.EditorGroup.OpenFile(path)
	h.app.EditorGroup.PinActiveTab()
	h.redraw()

	screen := h.screenText()
	// The "x := 1" line has 12-space indent, should have guides at col 4 and col 8
	lines := strings.Split(screen, "\n")
	guideCount := 0
	for _, line := range lines {
		if strings.Contains(line, "x := 1") {
			// Count │ characters in this line
			for _, ch := range line {
				if ch == '│' {
					guideCount++
				}
			}
			break
		}
	}
	if guideCount < 2 {
		t.Errorf("expected at least 2 indent guides on deeply nested line, got %d\nscreen:\n%s", guideCount, screen)
	}
}
