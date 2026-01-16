package highlight

import "regexp"

// Style represents a highlighting style (placeholder for now).
type Style int

const (
	StyleNormal Style = iota
	StyleComment
	StyleString
	StyleKeyword
)

// Span represents a highlighted region in a line.
type Span struct {
	Start int
	End   int
	Style Style
}

// Highlighter provides per-line syntax highlighting.
type Highlighter interface {
	Highlight(line string) []Span
}

// RegexHighlighter is a simple regex-based highlighter.
type RegexHighlighter struct {
	comment *regexp.Regexp
	str     *regexp.Regexp
	keyword *regexp.Regexp
}

func NewRegexHighlighter() *RegexHighlighter {
	return &RegexHighlighter{
		comment: regexp.MustCompile(`//.*`),
		str:     regexp.MustCompile(`"[^"]*"`),
		keyword: regexp.MustCompile(`\b(func|package|import|type|var|const|if|else|for|return)\b`),
	}
}

func (h *RegexHighlighter) Highlight(line string) []Span {
	spans := []Span{}
	// Comments
	if loc := h.comment.FindStringIndex(line); loc != nil {
		spans = append(spans, Span{Start: loc[0], End: loc[1], Style: StyleComment})
	}
	// Strings
	for _, loc := range h.str.FindAllStringIndex(line, -1) {
		spans = append(spans, Span{Start: loc[0], End: loc[1], Style: StyleString})
	}
	// Keywords
	for _, loc := range h.keyword.FindAllStringIndex(line, -1) {
		spans = append(spans, Span{Start: loc[0], End: loc[1], Style: StyleKeyword})
	}
	return spans
}
