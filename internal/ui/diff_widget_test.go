package ui

import (
	"testing"

	"github.com/eugenioenko/ttt/internal/core/diff"
)

func TestDiffWidgetLeftRightLines(t *testing.T) {
	fd := diff.Parse("--- a/test.go\n+++ b/test.go\n@@ -1,3 +1,3 @@\n hello world\n-old line\n+new line\n context\n")
	dv := NewDiffViewWidget("test.go", fd, nil, nil, false)

	left := dv.LeftLines()
	right := dv.RightLines()

	if len(left) == 0 {
		t.Fatal("LeftLines returned empty")
	}
	if len(right) == 0 {
		t.Fatal("RightLines returned empty")
	}
	if len(left) != len(right) {
		t.Fatalf("left/right length mismatch: %d vs %d", len(left), len(right))
	}

	foundOld := false
	foundNew := false
	for _, l := range left {
		if l == "old line" {
			foundOld = true
		}
	}
	for _, r := range right {
		if r == "new line" {
			foundNew = true
		}
	}
	if !foundOld {
		t.Errorf("expected 'old line' in left lines, got: %v", left)
	}
	if !foundNew {
		t.Errorf("expected 'new line' in right lines, got: %v", right)
	}
}

func TestDiffWidgetSearchFindsMatches(t *testing.T) {
	fd := diff.Parse("--- a/test.go\n+++ b/test.go\n@@ -1,3 +1,3 @@\n hello world\n-old line\n+new line\n context\n")
	dv := NewDiffViewWidget("test.go", fd, nil, nil, false)

	leftMatches, err := FindInLines(dv.LeftLines(), "old", SearchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	rightMatches, err := FindInLines(dv.RightLines(), "new", SearchOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(leftMatches) == 0 {
		t.Errorf("expected matches for 'old' in left, got none. Left lines: %v", dv.LeftLines())
	}
	if len(rightMatches) == 0 {
		t.Errorf("expected matches for 'new' in right, got none. Right lines: %v", dv.RightLines())
	}
}

func TestDiffWidgetSetSearchMatches(t *testing.T) {
	fd := diff.Parse("--- a/test.go\n+++ b/test.go\n@@ -1,3 +1,3 @@\n hello world\n-old line\n+new line\n context\n")
	dv := NewDiffViewWidget("test.go", fd, nil, nil, false)

	leftMatches, _ := FindInLines(dv.LeftLines(), "line", SearchOptions{})
	rightMatches, _ := FindInLines(dv.RightLines(), "line", SearchOptions{})

	merged := dv.SetSearchMatches(leftMatches, rightMatches)

	if len(merged) == 0 {
		t.Fatal("expected merged matches, got none")
	}
	if len(merged) != len(leftMatches)+len(rightMatches) {
		t.Errorf("merged count %d != left %d + right %d", len(merged), len(leftMatches), len(rightMatches))
	}

	dv.SetActiveMatch(0)
	if dv.searchActiveSideIdx < 0 {
		t.Error("expected active side index >= 0 after SetActiveMatch(0)")
	}
}

func TestDiffWidgetExtendedMode(t *testing.T) {
	fd := diff.Parse("--- a/test.go\n+++ b/test.go\n@@ -2,3 +2,3 @@\n hello world\n-old line\n+new line\n context\n")
	oldLines := []string{"first", "hello world", "old line", "context", "last line", "another"}
	newLines := []string{"first", "hello world", "new line", "context", "last line", "another"}
	dv := NewDiffViewWidget("test.go", fd, oldLines, newLines, false)

	compactCount := len(dv.Lines)

	dv.SetExtended(true)
	if !dv.IsExtended() {
		t.Error("expected extended mode to be true")
	}
	extendedCount := len(dv.Lines)
	if extendedCount <= compactCount {
		t.Errorf("extended should have more lines than compact: %d vs %d", extendedCount, compactCount)
	}

	dv.SetExtended(false)
	if dv.IsExtended() {
		t.Error("expected extended mode to be false")
	}
	if len(dv.Lines) != compactCount {
		t.Errorf("compact line count changed: %d vs %d", len(dv.Lines), compactCount)
	}
}

func TestDiffWidgetSearchContext(t *testing.T) {
	fd := diff.Parse("--- a/test.go\n+++ b/test.go\n@@ -1,3 +1,3 @@\n hello world\n-old line\n+new line\n context\n")
	dv := NewDiffViewWidget("test.go", fd, nil, nil, false)

	leftMatches, _ := FindInLines(dv.LeftLines(), "hello", SearchOptions{})
	rightMatches, _ := FindInLines(dv.RightLines(), "hello", SearchOptions{})

	total := len(leftMatches) + len(rightMatches)
	if total == 0 {
		t.Errorf("expected matches for 'hello' in context lines, got none. Left: %v, Right: %v", dv.LeftLines(), dv.RightLines())
	}
}
