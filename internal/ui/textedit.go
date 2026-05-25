package ui

import "github.com/gdamore/tcell/v2"

type TextEditResult struct {
	Text    string
	CurPos  int
	Changed bool
}

func HandleTextEdit(kev *tcell.EventKey, text string, curPos int) TextEditResult {
	runes := []rune(text)
	switch kev.Key() {
	case tcell.KeyRune:
		if kev.Modifiers() != 0 {
			return TextEditResult{Text: text, CurPos: curPos}
		}
		runes = append(runes[:curPos], append([]rune{kev.Rune()}, runes[curPos:]...)...)
		return TextEditResult{Text: string(runes), CurPos: curPos + 1, Changed: true}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if curPos > 0 {
			runes = append(runes[:curPos-1], runes[curPos:]...)
			return TextEditResult{Text: string(runes), CurPos: curPos - 1, Changed: true}
		}
	case tcell.KeyDelete:
		if curPos < len(runes) {
			runes = append(runes[:curPos], runes[curPos+1:]...)
			return TextEditResult{Text: string(runes), CurPos: curPos, Changed: true}
		}
	case tcell.KeyLeft:
		if curPos > 0 {
			return TextEditResult{Text: text, CurPos: curPos - 1}
		}
	case tcell.KeyRight:
		if curPos < len(runes) {
			return TextEditResult{Text: text, CurPos: curPos + 1}
		}
	case tcell.KeyHome:
		return TextEditResult{Text: text, CurPos: 0}
	case tcell.KeyEnd:
		return TextEditResult{Text: text, CurPos: len(runes)}
	}
	return TextEditResult{Text: text, CurPos: curPos}
}
