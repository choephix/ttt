package widgets

import (
	"strings"

	"github.com/eugenioenko/ttt/internal/markdown"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type MarkdownWidget struct {
	BaseWidget
	lines     []markdown.Line
	wrapWidth int
	wrapped   []markdown.Line
}

func NewMarkdownWidget() *MarkdownWidget {
	return &MarkdownWidget{}
}

func (m *MarkdownWidget) SetContent(text string) {
	m.lines = markdown.Render(text)
	m.wrapWidth = 0
	m.wrapped = nil
}

func (m *MarkdownWidget) Height() int { return 0 }
func (m *MarkdownWidget) Width() int  { return 0 }

func (m *MarkdownWidget) ScrollSize() (int, int) {
	r := m.GetRect()
	w := r.W
	if w <= 0 {
		w = 80
	}
	m.rewrap(w)
	return w, len(m.wrapped)
}

func (m *MarkdownWidget) rewrap(width int) {
	if width == m.wrapWidth && m.wrapped != nil {
		return
	}
	m.wrapWidth = width
	m.wrapped = nil
	for _, line := range m.lines {
		text := line.Text()
		if text == "" {
			m.wrapped = append(m.wrapped, markdown.Line{
				Spans: []markdown.Span{{Text: "", Style: term.StyleDefault}},
			})
			continue
		}
		if strings.HasPrefix(text, "---") {
			m.wrapped = append(m.wrapped, line)
			continue
		}
		if len([]rune(text)) <= width {
			m.wrapped = append(m.wrapped, line)
			continue
		}
		m.wrapped = append(m.wrapped, wrapMarkdownLine(line, width)...)
	}
}

func wrapMarkdownLine(line markdown.Line, width int) []markdown.Line {
	styles := flattenMarkdownStyles(line)
	runes := []rune(line.Text())
	var result []markdown.Line
	for len(runes) > 0 {
		end := width
		if end > len(runes) {
			end = len(runes)
		}
		if end < len(runes) {
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
		var spans []markdown.Span
		pos := 0
		chunk := runes[:end]
		chunkStyles := styles[:end]
		for pos < len(chunk) {
			st := chunkStyles[pos]
			start := pos
			for pos < len(chunk) && chunkStyles[pos] == st {
				pos++
			}
			spans = append(spans, markdown.Span{Text: string(chunk[start:pos]), Style: st})
		}
		result = append(result, markdown.Line{Spans: spans})
		runes = runes[end:]
		styles = styles[end:]
	}
	return result
}

func flattenMarkdownStyles(line markdown.Line) []term.Style {
	var styles []term.Style
	for _, span := range line.Spans {
		for range []rune(span.Text) {
			styles = append(styles, span.Style)
		}
	}
	return styles
}

func (m *MarkdownWidget) Render(surface Surface) {
	w, h := surface.Size()
	if w <= 0 || h <= 0 {
		return
	}

	m.rewrap(w)

	for y := 0; y < h && y < len(m.wrapped); y++ {
		line := m.wrapped[y]
		x := 0
		for _, span := range line.Spans {
			for _, ch := range span.Text {
				if x >= w {
					break
				}
				surface.SetCell(x, y, term.Cell{Ch: ch, Style: span.Style})
				x++
			}
		}
	}
}

func (m *MarkdownWidget) HandleEvent(ev tcell.Event) EventResult {
	return EventIgnored
}
