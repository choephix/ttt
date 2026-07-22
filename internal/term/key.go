package term

import "github.com/gdamore/tcell/v2"

// KeyRune returns the rune carried by a key event, or 0 if none.
// Isolates the tcell v2 EventKey.Rune() / v3 EventKey.Str() difference.
func KeyRune(ev *tcell.EventKey) rune {
	return ev.Rune()
}
