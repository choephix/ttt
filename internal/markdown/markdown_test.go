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

func TestWrapLineShortLine(t *testing.T) {
	line := Line{Spans: []Span{{Text: "hello", Style: term.StylePaletteItem}}}
	result := wrapLine(line, 60)
	if len(result) != 1 {
		t.Fatalf("expected 1 line, got %d", len(result))
	}
	if result[0].Text() != "hello" {
		t.Errorf("expected 'hello', got %q", result[0].Text())
	}
}

func TestWrapLineExactWidth(t *testing.T) {
	text := "abcde"
	line := Line{Spans: []Span{{Text: text, Style: term.StylePaletteItem}}}
	result := wrapLine(line, 5)
	if len(result) != 1 {
		t.Fatalf("expected 1 line for exact width, got %d", len(result))
	}
}

func TestWrapLineLongLineBreaksAtSpace(t *testing.T) {
	line := Line{Spans: []Span{{Text: "hello world foo", Style: term.StylePaletteItem}}}
	result := wrapLine(line, 12)
	if len(result) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(result))
	}
	// First line should break at a space within width
	first := result[0].Text()
	if len([]rune(first)) > 12 {
		t.Errorf("first line should be at most 12 runes, got %d: %q", len([]rune(first)), first)
	}
	// Reconstruct full text
	var full string
	for _, l := range result {
		full += l.Text()
	}
	if full != "hello world foo" {
		t.Errorf("expected full text 'hello world foo', got %q", full)
	}
}

func TestWrapLineNoSpaceForces(t *testing.T) {
	// A very long word with no spaces — should break at width boundary
	line := Line{Spans: []Span{{Text: "abcdefghijklmnop", Style: term.StylePaletteItem}}}
	result := wrapLine(line, 5)
	if len(result) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(result))
	}
	// Each line except maybe the last should be exactly 5 runes
	for i := 0; i < len(result)-1; i++ {
		if len([]rune(result[i].Text())) != 5 {
			t.Errorf("line %d: expected 5 runes, got %d: %q", i, len([]rune(result[i].Text())), result[i].Text())
		}
	}
}

func TestWrapLinePreservesStyles(t *testing.T) {
	// "hello world" with "hello" in one style and " world" in another
	line := Line{Spans: []Span{
		{Text: "hello ", Style: term.StyleHoverBold},
		{Text: "world again", Style: term.StyleHoverCode},
	}}
	result := wrapLine(line, 8)
	if len(result) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(result))
	}

	// Verify styles are preserved across wrapped lines
	allText := ""
	for _, l := range result {
		allText += l.Text()
	}
	if allText != "hello world again" {
		t.Errorf("expected full text 'hello world again', got %q", allText)
	}
}

func TestFlattenStylesEmpty(t *testing.T) {
	line := Line{Spans: []Span{}}
	styles := flattenStyles(line)
	if len(styles) != 0 {
		t.Errorf("expected 0 styles, got %d", len(styles))
	}
}

func TestFlattenStylesSingleSpan(t *testing.T) {
	line := Line{Spans: []Span{{Text: "abc", Style: term.StyleHoverBold}}}
	styles := flattenStyles(line)
	if len(styles) != 3 {
		t.Fatalf("expected 3 styles, got %d", len(styles))
	}
	for i, s := range styles {
		if s != term.StyleHoverBold {
			t.Errorf("style[%d] = %d, want %d", i, s, term.StyleHoverBold)
		}
	}
}

func TestFlattenStylesMultipleSpans(t *testing.T) {
	line := Line{Spans: []Span{
		{Text: "ab", Style: term.StyleHoverBold},
		{Text: "cd", Style: term.StyleHoverCode},
	}}
	styles := flattenStyles(line)
	if len(styles) != 4 {
		t.Fatalf("expected 4 styles, got %d", len(styles))
	}
	if styles[0] != term.StyleHoverBold || styles[1] != term.StyleHoverBold {
		t.Error("first two styles should be StyleHoverBold")
	}
	if styles[2] != term.StyleHoverCode || styles[3] != term.StyleHoverCode {
		t.Error("last two styles should be StyleHoverCode")
	}
}

func TestFlattenStylesUnicode(t *testing.T) {
	line := Line{Spans: []Span{{Text: "日本語", Style: term.StyleDefault}}}
	styles := flattenStyles(line)
	if len(styles) != 3 {
		t.Fatalf("expected 3 styles for 3 runes, got %d", len(styles))
	}
}

func TestLineText(t *testing.T) {
	line := Line{Spans: []Span{
		{Text: "hello", Style: term.StyleDefault},
		{Text: " ", Style: term.StyleDefault},
		{Text: "world", Style: term.StyleHoverBold},
	}}
	if line.Text() != "hello world" {
		t.Errorf("expected 'hello world', got %q", line.Text())
	}
}

func TestLineTextEmpty(t *testing.T) {
	line := Line{Spans: []Span{}}
	if line.Text() != "" {
		t.Errorf("expected empty string, got %q", line.Text())
	}
}

func TestRenderDividerBlanksStripped(t *testing.T) {
	// Extra blank lines around --- should be stripped
	lines := Render("above\n\n\n---\n\n\nbelow")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].Text() != "above" {
		t.Errorf("expected 'above', got %q", lines[0].Text())
	}
	if lines[1].Spans[0].Style != term.StyleBorder {
		t.Error("expected divider with StyleBorder")
	}
	if lines[2].Text() != "below" {
		t.Errorf("expected 'below', got %q", lines[2].Text())
	}
}

func TestRenderUnmatchedBacktick(t *testing.T) {
	lines := Render("unmatched ` backtick")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	text := lines[0].Text()
	if text != "unmatched ` backtick" {
		t.Errorf("expected 'unmatched ` backtick', got %q", text)
	}
}

func TestRenderUnmatchedBold(t *testing.T) {
	lines := Render("unmatched ** bold")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	text := lines[0].Text()
	if text != "unmatched ** bold" {
		t.Errorf("expected literal **, got %q", text)
	}
}

func TestRenderUnclosedCodeBlock(t *testing.T) {
	// Code block without closing ```
	lines := Render("```\ncode line\nno close")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].Text() != "code line" {
		t.Errorf("expected 'code line', got %q", lines[0].Text())
	}
	if lines[1].Text() != "no close" {
		t.Errorf("expected 'no close', got %q", lines[1].Text())
	}
}
