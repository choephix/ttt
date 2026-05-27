package diff

import "testing"

const sampleDiff = `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,5 @@
 package main

-import "fmt"
+import "log"

 func main() {
@@ -10,3 +10,4 @@
 	x := 1
-	y := 2
+	y := 3
+	z := 4
`

func TestParseFileNames(t *testing.T) {
	fd := Parse(sampleDiff)
	if fd.OldName != "main.go" {
		t.Errorf("expected old name 'main.go', got %q", fd.OldName)
	}
	if fd.NewName != "main.go" {
		t.Errorf("expected new name 'main.go', got %q", fd.NewName)
	}
}

func TestParseHunkCount(t *testing.T) {
	fd := Parse(sampleDiff)
	if len(fd.Hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(fd.Hunks))
	}
}

func TestParseContextLines(t *testing.T) {
	fd := Parse(sampleDiff)
	h := fd.Hunks[0]
	if h.Lines[0].Left.Kind != Context {
		t.Errorf("first line should be context, got %d", h.Lines[0].Left.Kind)
	}
	if h.Lines[0].Left.Text != "package main" {
		t.Errorf("expected 'package main', got %q", h.Lines[0].Left.Text)
	}
}

func TestParseDeletedAdded(t *testing.T) {
	fd := Parse(sampleDiff)
	h := fd.Hunks[0]
	// Line at index 2 should be the change: -import "fmt" / +import "log"
	found := false
	for _, dl := range h.Lines {
		if dl.Left.Kind == Deleted && dl.Right.Kind == Added {
			if dl.Left.Text == `import "fmt"` && dl.Right.Text == `import "log"` {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected paired delete/add for import line")
	}
}

func TestParseUnmatchedAdd(t *testing.T) {
	fd := Parse(sampleDiff)
	h := fd.Hunks[1]
	// Second hunk has 1 delete and 2 adds, so last row should have blank left
	found := false
	for _, dl := range h.Lines {
		if dl.Left.Kind == Blank && dl.Right.Kind == Added {
			if dl.Right.Text == "\tz := 4" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected unmatched add line for z := 4")
	}
}

func TestAllLines(t *testing.T) {
	fd := Parse(sampleDiff)
	all := fd.AllLines()
	if len(all) == 0 {
		t.Fatal("AllLines returned empty")
	}
	// Should have lines from both hunks plus a separator
	hunk1Lines := len(fd.Hunks[0].Lines)
	hunk2Lines := len(fd.Hunks[1].Lines)
	expected := hunk1Lines + 1 + hunk2Lines // +1 for separator
	if len(all) != expected {
		t.Errorf("expected %d all lines, got %d", expected, len(all))
	}
}

func TestParseEmpty(t *testing.T) {
	fd := Parse("")
	if len(fd.Hunks) != 0 {
		t.Errorf("expected 0 hunks for empty input, got %d", len(fd.Hunks))
	}
}

func TestGenerate(t *testing.T) {
	old := []string{"a", "b", "c", "d"}
	new := []string{"a", "x", "c", "d"}
	result := Generate(old, new, "test.txt")
	if result == "" {
		t.Fatal("expected non-empty diff")
	}

	fd := Parse(result)
	if fd.OldName != "test.txt" || fd.NewName != "test.txt" {
		t.Errorf("wrong names: %q %q", fd.OldName, fd.NewName)
	}
	if len(fd.Hunks) == 0 {
		t.Fatal("expected at least 1 hunk")
	}

	foundDel := false
	foundAdd := false
	for _, h := range fd.Hunks {
		for _, dl := range h.Lines {
			if dl.Left.Kind == Deleted && dl.Left.Text == "b" {
				foundDel = true
			}
			if dl.Right.Kind == Added && dl.Right.Text == "x" {
				foundAdd = true
			}
		}
	}
	if !foundDel {
		t.Error("expected deleted line 'b'")
	}
	if !foundAdd {
		t.Error("expected added line 'x'")
	}
}

func TestGenerateIdentical(t *testing.T) {
	lines := []string{"a", "b", "c"}
	result := Generate(lines, lines, "test.txt")
	if result != "" {
		t.Errorf("expected empty diff for identical files, got: %s", result)
	}
}

func TestGenerateAddition(t *testing.T) {
	old := []string{"a", "c"}
	new := []string{"a", "b", "c"}
	result := Generate(old, new, "test.txt")
	fd := Parse(result)

	foundAdd := false
	for _, h := range fd.Hunks {
		for _, dl := range h.Lines {
			if dl.Right.Kind == Added && dl.Right.Text == "b" {
				foundAdd = true
			}
		}
	}
	if !foundAdd {
		t.Error("expected added line 'b'")
	}
}
