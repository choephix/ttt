package highlight

import "testing"

func TestRegexHighlighter_Comment(t *testing.T) {
	h := NewRegexHighlighter()
	spans := h.Highlight("let x = 1 // comment")
	found := false
	for _, s := range spans {
		if s.Style == StyleComment {
			found = true
		}
	}
	if !found {
		t.Error("expected comment span")
	}
}

func TestRegexHighlighter_String(t *testing.T) {
	h := NewRegexHighlighter()
	spans := h.Highlight("let s = \"hello\"")
	found := false
	for _, s := range spans {
		if s.Style == StyleString {
			found = true
		}
	}
	if !found {
		t.Error("expected string span")
	}
}

func TestRegexHighlighter_Keyword(t *testing.T) {
	h := NewRegexHighlighter()
	spans := h.Highlight("func main() {}")
	found := false
	for _, s := range spans {
		if s.Style == StyleKeyword {
			found = true
		}
	}
	if !found {
		t.Error("expected keyword span")
	}
}
