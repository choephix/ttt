package ui

import "unicode"

// wordAt returns the maximal run of word characters (letters, digits, or '_')
// surrounding col in lineText, or "" if col is not on a word character.
func wordAt(lineText string, col int) string {
	runes := []rune(lineText)
	if len(runes) == 0 || col < 0 {
		return ""
	}
	if col >= len(runes) {
		col = len(runes) - 1
	}
	isWord := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
	}
	if !isWord(runes[col]) {
		return ""
	}
	start, end := col, col
	for start > 0 && isWord(runes[start-1]) {
		start--
	}
	for end < len(runes)-1 && isWord(runes[end+1]) {
		end++
	}
	return string(runes[start : end+1])
}

func leadingIndentWidth(line string, tabSize int) int {
	runes := []rune(line)
	if len(runes) > 0 && runes[0] == '\t' {
		return 1
	}
	remove := 0
	for remove < tabSize && remove < len(runes) && runes[remove] == ' ' {
		remove++
	}
	return remove
}

func leadingWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
}

func bufColToVisualCol(line string, bufCol, tabW int) int {
	visCol := 0
	ri := 0
	for _, ch := range line {
		if ri >= bufCol {
			break
		}
		if ch == '\t' {
			visCol = ((visCol / tabW) + 1) * tabW
		} else {
			visCol++
		}
		ri++
	}
	return visCol
}

func visualColToBufCol(line string, targetVisCol, tabW int) int {
	visCol := 0
	ri := 0
	for _, ch := range line {
		if visCol >= targetVisCol {
			return ri
		}
		if ch == '\t' {
			nextStop := ((visCol / tabW) + 1) * tabW
			if targetVisCol < nextStop {
				return ri
			}
			visCol = nextStop
		} else {
			visCol++
		}
		ri++
	}
	return len([]rune(line))
}

func isEditorIdentRune(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
