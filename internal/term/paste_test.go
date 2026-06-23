package term

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func key(k tcell.Key, r rune, mod tcell.ModMask) *tcell.EventKey {
	return tcell.NewEventKey(k, r, mod)
}

func TestCollectPasteText_Runes(t *testing.T) {
	events := []*tcell.EventKey{
		key(tcell.KeyRune, 'H', tcell.ModNone),
		key(tcell.KeyRune, 'i', tcell.ModNone),
	}
	got := CollectPasteText(events)
	if got != "Hi" {
		t.Errorf("got %q, want %q", got, "Hi")
	}
}

func TestCollectPasteText_Unicode(t *testing.T) {
	events := []*tcell.EventKey{
		key(tcell.KeyRune, '┌', tcell.ModNone),
		key(tcell.KeyRune, '─', tcell.ModNone),
		key(tcell.KeyRune, '┐', tcell.ModNone),
	}
	got := CollectPasteText(events)
	if got != "┌─┐" {
		t.Errorf("got %q, want %q", got, "┌─┐")
	}
}

func TestCollectPasteText_UnixNewlines(t *testing.T) {
	// Unix \n arrives as KeyCtrlJ
	events := []*tcell.EventKey{
		key(tcell.KeyRune, 'a', tcell.ModNone),
		key(tcell.KeyCtrlJ, 0, tcell.ModCtrl),
		key(tcell.KeyRune, 'b', tcell.ModNone),
	}
	got := CollectPasteText(events)
	if got != "a\nb" {
		t.Errorf("got %q, want %q", got, "a\nb")
	}
}

func TestCollectPasteText_CRLFNewlines(t *testing.T) {
	// Windows \r\n arrives as KeyEnter followed by KeyCtrlJ
	events := []*tcell.EventKey{
		key(tcell.KeyRune, 'a', tcell.ModNone),
		key(tcell.KeyEnter, 0, tcell.ModNone),
		key(tcell.KeyCtrlJ, 0, tcell.ModCtrl),
		key(tcell.KeyRune, 'b', tcell.ModNone),
	}
	got := CollectPasteText(events)
	if got != "a\nb" {
		t.Errorf("got %q, want %q", got, "a\nb")
	}
}

func TestCollectPasteText_CROnly(t *testing.T) {
	// Old Mac \r arrives as KeyEnter, normalized to \n
	events := []*tcell.EventKey{
		key(tcell.KeyRune, 'a', tcell.ModNone),
		key(tcell.KeyEnter, 0, tcell.ModNone),
		key(tcell.KeyRune, 'b', tcell.ModNone),
	}
	got := CollectPasteText(events)
	if got != "a\nb" {
		t.Errorf("got %q, want %q", got, "a\nb")
	}
}

func TestCollectPasteText_Tab(t *testing.T) {
	events := []*tcell.EventKey{
		key(tcell.KeyTab, 0, tcell.ModNone),
		key(tcell.KeyRune, 'x', tcell.ModNone),
	}
	got := CollectPasteText(events)
	if got != "\tx" {
		t.Errorf("got %q, want %q", got, "\tx")
	}
}

func TestCollectPasteText_ControlKeysSkipped(t *testing.T) {
	events := []*tcell.EventKey{
		key(tcell.KeyRune, 'a', tcell.ModNone),
		key(tcell.KeyEscape, 0, tcell.ModNone),
		key(tcell.KeyBackspace, 0, tcell.ModNone),
		key(tcell.KeyUp, 0, tcell.ModNone),
		key(tcell.KeyRune, 'b', tcell.ModNone),
	}
	got := CollectPasteText(events)
	if got != "ab" {
		t.Errorf("got %q, want %q", got, "ab")
	}
}

func TestCollectPasteText_Empty(t *testing.T) {
	got := CollectPasteText(nil)
	if got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestCollectPasteText_MixedLineEndings(t *testing.T) {
	// Mix of \r\n, \n, and \r in one paste
	events := []*tcell.EventKey{
		key(tcell.KeyRune, 'a', tcell.ModNone),
		key(tcell.KeyEnter, 0, tcell.ModNone),       // \r (part of \r\n)
		key(tcell.KeyCtrlJ, 0, tcell.ModCtrl),        // \n
		key(tcell.KeyRune, 'b', tcell.ModNone),
		key(tcell.KeyCtrlJ, 0, tcell.ModCtrl),        // standalone \n
		key(tcell.KeyRune, 'c', tcell.ModNone),
		key(tcell.KeyEnter, 0, tcell.ModNone),         // standalone \r
		key(tcell.KeyRune, 'd', tcell.ModNone),
	}
	got := CollectPasteText(events)
	if got != "a\nb\nc\nd" {
		t.Errorf("got %q, want %q", got, "a\nb\nc\nd")
	}
}
