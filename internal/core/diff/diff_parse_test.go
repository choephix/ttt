package diff

import (
	"fmt"
	"testing"
)

func TestDiffParseRealOutput(t *testing.T) {
	unified := "diff --git a/test.go b/test.go\n--- a/test.go\n+++ b/test.go\n@@ -1,5 +1,5 @@\n package main\n \n-func oldFunction() {\n-    fmt.Println(\"hello\")\n+func newFunction() {\n+    fmt.Println(\"world\")\n }\n"
	fd := Parse(unified)
	lines := fd.AllLines()
	fmt.Printf("Total diff lines: %d\n", len(lines))
	for i, dl := range lines {
		fmt.Printf("[%d] L: kind=%d num=%d text=%q\n", i, dl.Left.Kind, dl.Left.Num, dl.Left.Text)
		fmt.Printf("     R: kind=%d num=%d text=%q\n", dl.Right.Kind, dl.Right.Num, dl.Right.Text)
	}
	if len(lines) == 0 {
		t.Fatal("no lines parsed")
	}
	// Check that context lines have text
	foundPackage := false
	for _, dl := range lines {
		if dl.Left.Text == "package main" || dl.Right.Text == "package main" {
			foundPackage = true
		}
	}
	if !foundPackage {
		t.Error("expected 'package main' in context lines")
	}
}
