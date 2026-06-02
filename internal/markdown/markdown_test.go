package markdown

import (
	"testing"

	"github.com/eugenioenko/ttt/internal/term"
)

func TestRenderPlainText(t *testing.T) {
	lines := Render("hello world")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Text() != "hello world" {
		t.Errorf("expected 'hello world', got %q", lines[0].Text())
	}
}

func TestRenderInlineCode(t *testing.T) {
	lines := Render("use `fmt.Println` here")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Text() != "use fmt.Println here" {
		t.Errorf("expected 'use fmt.Println here', got %q", lines[0].Text())
	}
	found := false
	for _, s := range lines[0].Spans {
		if s.Text == "fmt.Println" && s.Style == term.StyleHoverCode {
			found = true
		}
	}
	if !found {
		t.Error("expected inline code span with StyleHoverCode")
	}
}

func TestRenderBold(t *testing.T) {
	lines := Render("this is **bold** text")
	if lines[0].Text() != "this is bold text" {
		t.Errorf("expected 'this is bold text', got %q", lines[0].Text())
	}
	found := false
	for _, s := range lines[0].Spans {
		if s.Text == "bold" && s.Style == term.StyleHoverBold {
			found = true
		}
	}
	if !found {
		t.Error("expected bold span with StyleHoverBold")
	}
}

func TestRenderLink(t *testing.T) {
	lines := Render("see [docs](https://example.com) here")
	if lines[0].Text() != "see docs here" {
		t.Errorf("expected 'see docs here', got %q", lines[0].Text())
	}
}

func TestRenderCodeBlock(t *testing.T) {
	input := "before\n```go\nfunc main() {}\n```\nafter"
	lines := Render(input)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].Text() != "before" {
		t.Errorf("line 0: expected 'before', got %q", lines[0].Text())
	}
	if lines[1].Text() != "func main() {}" {
		t.Errorf("line 1: expected 'func main() {}', got %q", lines[1].Text())
	}
	if lines[2].Text() != "after" {
		t.Errorf("line 2: expected 'after', got %q", lines[2].Text())
	}
}

func TestRenderDivider(t *testing.T) {
	lines := Render("above\n---\nbelow")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[1].Spans[0].Style != term.StyleBorder {
		t.Error("expected divider line to have StyleBorder")
	}
}

func TestRenderMultilineCodeBlock(t *testing.T) {
	input := "```\nline1\nline2\nline3\n```"
	lines := Render(input)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	for i, expected := range []string{"line1", "line2", "line3"} {
		if lines[i].Text() != expected {
			t.Errorf("line %d: expected %q, got %q", i, expected, lines[i].Text())
		}
	}
}

func TestRenderItalic(t *testing.T) {
	lines := Render("this is *italic* text")
	if lines[0].Text() != "this is italic text" {
		t.Errorf("expected 'this is italic text', got %q", lines[0].Text())
	}
}

func TestRenderLinkWithInlineCode(t *testing.T) {
	lines := Render("[`string` on pkg.go.dev](https://pkg.go.dev/builtin#string)")
	text := lines[0].Text()
	if text != "string on pkg.go.dev" {
		t.Errorf("expected 'string on pkg.go.dev', got %q", text)
	}
	found := false
	for _, s := range lines[0].Spans {
		if s.Text == "string" && s.Style == term.StyleHoverCode {
			found = true
		}
	}
	if !found {
		t.Error("expected inline code span inside link")
	}
}

func TestRenderEmptyText(t *testing.T) {
	lines := Render("")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}
