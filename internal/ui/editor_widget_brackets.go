package ui

import "github.com/eugenioenko/ttt/internal/term"

var bracketPairs = map[rune]rune{
	'(': ')', ')': '(',
	'[': ']', ']': '[',
	'{': '}', '}': '{',
}

var closingBrackets = map[rune]bool{')': true, ']': true, '}': true}

var indentOpeners = map[rune]bool{'{': true, '(': true, '[': true, ':': true}

func (e *EditorPaneWidget) findMatchingBracket() (int, int, bool) {
	if e.Cursor.Line < 0 || e.Cursor.Line >= len(e.Buf.Lines) {
		return 0, 0, false
	}
	runes := []rune(e.Buf.Lines[e.Cursor.Line])
	if e.Cursor.Col < 0 || e.Cursor.Col >= len(runes) {
		return 0, 0, false
	}
	ch := runes[e.Cursor.Col]
	match, ok := bracketPairs[ch]
	if !ok {
		return 0, 0, false
	}
	dir := 1
	if closingBrackets[ch] {
		dir = -1
	}
	depth := 1
	line, col := e.Cursor.Line, e.Cursor.Col
	for {
		col += dir
		lr := []rune(e.Buf.Lines[line])
		if col < 0 {
			line--
			if line < 0 {
				return 0, 0, false
			}
			lr = []rune(e.Buf.Lines[line])
			col = len(lr) - 1
			if col < 0 {
				col = 0
				continue
			}
		} else if col >= len(lr) {
			line++
			if line >= len(e.Buf.Lines) {
				return 0, 0, false
			}
			col = 0
			lr = []rune(e.Buf.Lines[line])
			if len(lr) == 0 {
				continue
			}
		}
		c := lr[col]
		if c == ch {
			depth++
		} else if c == match {
			depth--
			if depth == 0 {
				return line, col, true
			}
		}
	}
}

func (e *EditorPaneWidget) GoToMatchingBracket() {
	line, col, ok := e.findMatchingBracket()
	if ok {
		e.ExpandFoldContaining(line)
		e.Cursor.Line = line
		e.Cursor.Col = col
		if e.Selection != nil && e.Selection.Active {
			e.Selection.Clear()
		}
		e.scrollViewport()
	}
}

var openBrackets = map[rune]bool{'(': true, '[': true, '{': true}

type bracketColorEntry struct {
	col   int
	style term.Style
}

type bracketColorMap [][]bracketColorEntry

func (e *EditorPaneWidget) computeBracketColors() bracketColorMap {
	totalLines := len(e.Buf.Lines)
	if len(e.BracketColorStyles) == 0 || totalLines == 0 {
		return nil
	}

	result := make(bracketColorMap, totalLines)
	bracketStyles := e.BracketColorStyles
	numStyles := len(bracketStyles)
	depth := 0

	inLineComment := false
	inBlockComment := false
	inString := false
	var stringChar rune

	for lineIdx := 0; lineIdx < totalLines; lineIdx++ {
		runes := []rune(e.Buf.Lines[lineIdx])
		inLineComment = false
		n := len(runes)

		for i := 0; i < n; i++ {
			ch := runes[i]
			next := rune(0)
			if i+1 < n {
				next = runes[i+1]
			}

			if inBlockComment {
				if ch == '*' && next == '/' {
					inBlockComment = false
					i++
				}
				continue
			}
			if inLineComment {
				continue
			}
			if inString {
				if ch == '\\' {
					i++
					continue
				}
				if ch == stringChar {
					inString = false
				}
				continue
			}

			if ch == '/' && next == '/' {
				inLineComment = true
				continue
			}
			if ch == '/' && next == '*' {
				inBlockComment = true
				i++
				continue
			}
			if ch == '"' || ch == '\'' || ch == '`' {
				inString = true
				stringChar = ch
				if ch == '`' {
					// backtick strings can span lines; handled by not resetting inString
				}
				continue
			}

			if openBrackets[ch] {
				style := bracketStyles[depth%numStyles]
				depth++
				result[lineIdx] = append(result[lineIdx], bracketColorEntry{col: i, style: style})
			} else if closingBrackets[ch] {
				depth--
				if depth < 0 {
					depth = 0
				}
				style := bracketStyles[depth%numStyles]
				result[lineIdx] = append(result[lineIdx], bracketColorEntry{col: i, style: style})
			}
		}

		if inString && stringChar != '`' {
			inString = false
		}
	}
	return result
}
