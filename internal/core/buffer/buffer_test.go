package buffer

import "testing"

func TestInsertRune(t *testing.T) {
	b := &Buffer{Lines: []string{"hello"}}
	b.InsertRune(0, 5, '!')
	if b.Lines[0] != "hello!" {
		t.Errorf("expected 'hello!', got '%s'", b.Lines[0])
	}
}

func TestDeleteRune(t *testing.T) {
	b := &Buffer{Lines: []string{"hello!"}}
	b.DeleteRune(0, 5)
	if b.Lines[0] != "hello" {
		t.Errorf("expected 'hello', got '%s'", b.Lines[0])
	}
}

func TestInsertLine(t *testing.T) {
	b := &Buffer{Lines: []string{"a", "c"}}
	b.InsertLine(1, "b")
	if len(b.Lines) != 3 || b.Lines[1] != "b" {
		t.Errorf("expected line 'b' at index 1, got '%v'", b.Lines)
	}
}

func TestDetectIndent2Spaces(t *testing.T) {
	lines := []string{
		"function foo() {",
		"  if (true) {",
		"    return 1",
		"  }",
		"}",
	}
	info := DetectIndent(lines)
	if info.Size != 2 {
		t.Errorf("expected indent size 2, got %d", info.Size)
	}
	if info.UseTabs {
		t.Error("expected spaces, got tabs")
	}
}

func TestDetectIndent4Spaces(t *testing.T) {
	lines := []string{
		"func main() {",
		"    fmt.Println()",
		"    if true {",
		"        return",
		"    }",
		"}",
	}
	info := DetectIndent(lines)
	if info.Size != 4 {
		t.Errorf("expected indent size 4, got %d", info.Size)
	}
}

func TestDetectIndentTabs(t *testing.T) {
	lines := []string{
		"func main() {",
		"\tfmt.Println()",
		"\tif true {",
		"\t\treturn",
		"\t}",
		"}",
	}
	info := DetectIndent(lines)
	if !info.UseTabs {
		t.Error("expected tabs")
	}
}

func TestDeleteLine(t *testing.T) {
	b := &Buffer{Lines: []string{"a", "b", "c"}}
	b.DeleteLine(1)
	if len(b.Lines) != 2 || b.Lines[1] != "c" {
		t.Errorf("expected lines [a c], got '%v'", b.Lines)
	}
}
