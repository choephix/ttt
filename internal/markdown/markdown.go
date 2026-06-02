package markdown

import (
	"regexp"
	"strings"

	"github.com/eugenioenko/ttt/internal/core/highlight"
	"github.com/eugenioenko/ttt/internal/term"
)

// Strip blank lines adjacent to --- dividers (LSP hover content often has extra newlines around them)
var reDividerBlanks = regexp.MustCompile(`\n+---`)
var reDividerBlanksAfter = regexp.MustCompile(`---\n+`)

type Span struct {
	Text  string
	Style term.Style
}

type Line struct {
	Spans []Span
}

func (l Line) Text() string {
	var b strings.Builder
	for _, s := range l.Spans {
		b.WriteString(s.Text)
	}
	return b.String()
}

const WrapWidth = 60

func Render(text string) []Line {
	text = strings.TrimRight(text, "\n")
	text = reDividerBlanks.ReplaceAllString(text, "\n---")
	text = reDividerBlanksAfter.ReplaceAllString(text, "---\n")
	raw := strings.Split(text, "\n")
	var lines []Line
	i := 0
	for i < len(raw) {
		line := raw[i]
		if lang, ok := strings.CutPrefix(line, "```"); ok {
			lang = strings.TrimSpace(lang)
			var block []string
			i++
			for i < len(raw) && !strings.HasPrefix(raw[i], "```") {
				block = append(block, raw[i])
				i++
			}
			if i < len(raw) {
				i++
			}
			lines = append(lines, renderCodeBlock(block, lang)...)
			continue
		}
		if line == "---" {
			lines = append(lines, Line{Spans: []Span{{Text: "---", Style: term.StyleBorder}}})
			i++
			continue
		}
		parsed := renderInline(line)
		lines = append(lines, wrapLine(parsed, WrapWidth)...)
		i++
	}
	return lines
}

func wrapLine(line Line, width int) []Line {
	text := line.Text()
	if len([]rune(text)) <= width {
		return []Line{line}
	}
	styles := flattenStyles(line)
	runes := []rune(text)
	var result []Line
	for len(runes) > 0 {
		end := width
		if end > len(runes) {
			end = len(runes)
		}
		if end < len(runes) {
			// break at last space within width
			bp := -1
			for j := end - 1; j > 0; j-- {
				if runes[j] == ' ' {
					bp = j
					break
				}
			}
			if bp > 0 {
				end = bp + 1
			}
		}
		var spans []Span
		pos := 0
		chunk := runes[:end]
		chunkStyles := styles[:end]
		for pos < len(chunk) {
			st := chunkStyles[pos]
			start := pos
			for pos < len(chunk) && chunkStyles[pos] == st {
				pos++
			}
			spans = append(spans, Span{Text: string(chunk[start:pos]), Style: st})
		}
		result = append(result, Line{Spans: spans})
		runes = runes[end:]
		styles = styles[end:]
	}
	return result
}

func flattenStyles(line Line) []term.Style {
	var styles []term.Style
	for _, span := range line.Spans {
		for range []rune(span.Text) {
			styles = append(styles, span.Style)
		}
	}
	return styles
}

func renderCodeBlock(block []string, lang string) []Line {
	var h *highlight.Highlighter
	if lang != "" {
		h = highlight.New("file." + lang)
	}
	lines := make([]Line, len(block))
	for i, text := range block {
		if h != nil {
			spans := h.HighlightLine(text)
			lines[i] = highlightToLine(text, spans)
		} else {
			lines[i] = Line{Spans: []Span{{Text: text, Style: term.StyleHoverCode}}}
		}
	}
	return lines
}

func highlightToLine(text string, spans []highlight.Span) Line {
	if len(spans) == 0 {
		return Line{Spans: []Span{{Text: text, Style: term.StyleHoverCode}}}
	}
	runes := []rune(text)
	var result []Span
	pos := 0
	for _, s := range spans {
		if s.Start > pos {
			result = append(result, Span{Text: string(runes[pos:s.Start]), Style: term.StyleHoverCode})
		}
		result = append(result, Span{Text: string(runes[s.Start:s.End]), Style: s.Style})
		pos = s.End
	}
	if pos < len(runes) {
		result = append(result, Span{Text: string(runes[pos:]), Style: term.StyleHoverCode})
	}
	return Line{Spans: result}
}

func renderInline(text string) Line {
	if text == "" {
		return Line{Spans: []Span{{Text: "", Style: term.StylePaletteItem}}}
	}
	var spans []Span
	runes := []rune(text)
	i := 0
	var buf []rune
	flush := func(style term.Style) {
		if len(buf) > 0 {
			spans = append(spans, Span{Text: string(buf), Style: style})
			buf = nil
		}
	}
	for i < len(runes) {
		ch := runes[i]
		if ch == '`' {
			flush(term.StylePaletteItem)
			end := indexOf(runes, '`', i+1)
			if end == -1 {
				buf = append(buf, ch)
				i++
				continue
			}
			spans = append(spans, Span{Text: string(runes[i+1 : end]), Style: term.StyleHoverCode})
			i = end + 1
			continue
		}
		if ch == '*' && i+1 < len(runes) && runes[i+1] == '*' {
			flush(term.StylePaletteItem)
			end := indexOfDouble(runes, '*', i+2)
			if end == -1 {
				buf = append(buf, ch)
				i++
				continue
			}
			spans = append(spans, Span{Text: string(runes[i+2 : end]), Style: term.StyleHoverBold})
			i = end + 2
			continue
		}
		if ch == '*' || ch == '_' {
			flush(term.StylePaletteItem)
			end := indexOf(runes, ch, i+1)
			if end == -1 || end == i+1 {
				buf = append(buf, ch)
				i++
				continue
			}
			spans = append(spans, Span{Text: string(runes[i+1 : end]), Style: term.StyleHoverBold})
			i = end + 1
			continue
		}
		if ch == '[' {
			flush(term.StylePaletteItem)
			closeBracket := indexOf(runes, ']', i+1)
			if closeBracket != -1 && closeBracket+1 < len(runes) && runes[closeBracket+1] == '(' {
				closeParen := indexOf(runes, ')', closeBracket+2)
				if closeParen != -1 {
					linkText := string(runes[i+1 : closeBracket])
					inner := renderInline(linkText)
					spans = append(spans, inner.Spans...)
					i = closeParen + 1
					continue
				}
			}
			buf = append(buf, ch)
			i++
			continue
		}
		buf = append(buf, ch)
		i++
	}
	flush(term.StylePaletteItem)
	if len(spans) == 0 {
		spans = []Span{{Text: "", Style: term.StylePaletteItem}}
	}
	return Line{Spans: spans}
}

func indexOf(runes []rune, ch rune, from int) int {
	for i := from; i < len(runes); i++ {
		if runes[i] == ch {
			return i
		}
	}
	return -1
}

func indexOfDouble(runes []rune, ch rune, from int) int {
	for i := from; i+1 < len(runes); i++ {
		if runes[i] == ch && runes[i+1] == ch {
			return i
		}
	}
	return -1
}
