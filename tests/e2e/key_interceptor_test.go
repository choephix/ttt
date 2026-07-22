package e2e

import (
	"testing"

	"github.com/gdamore/tcell/v3"

	"github.com/eugenioenko/ttt/internal/ui"
)

func TestKeyInterceptorBlocksRune(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("file.new")

	intercepted := false
	h.app.Root.KeyInterceptor = func(ev *tcell.EventKey) bool {
		if ev.Key() == tcell.KeyRune && ev.Rune() == 'j' {
			intercepted = true
			return true
		}
		return false
	}

	h.pressRune('j')

	if !intercepted {
		t.Fatal("expected interceptor to be called")
	}
	if h.containsText("j") {
		t.Fatal("expected 'j' to be suppressed by interceptor")
	}
}

func TestKeyInterceptorPassthrough(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("file.new")

	h.app.Root.KeyInterceptor = func(ev *tcell.EventKey) bool {
		return false
	}

	h.pressRune('x')

	h.assertContains("x")
}

// Modal plugins (Vim mode) need Esc to leave insert mode, so the interceptor
// runs before EscapeDismissers.
func TestKeyInterceptorConsumesEscape(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("file.new")

	dismissed := false
	h.app.Root.EscapeDismissers = append(h.app.Root.EscapeDismissers, func() bool {
		dismissed = true
		return true
	})

	intercepted := false
	h.app.Root.KeyInterceptor = func(ev *tcell.EventKey) bool {
		if ev.Key() == tcell.KeyEscape {
			intercepted = true
			return true
		}
		return false
	}

	h.pressKey(tcell.KeyEscape, tcell.ModNone)

	if !intercepted {
		t.Fatal("expected interceptor to receive Escape")
	}
	if dismissed {
		t.Fatal("expected interceptor to preempt EscapeDismissers")
	}
}

func TestKeyInterceptorEscapePassthrough(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("file.new")

	dismissed := false
	h.app.Root.EscapeDismissers = append(h.app.Root.EscapeDismissers, func() bool {
		dismissed = true
		return true
	})

	h.app.Root.KeyInterceptor = func(ev *tcell.EventKey) bool {
		return false
	}

	h.pressKey(tcell.KeyEscape, tcell.ModNone)

	if !dismissed {
		t.Fatal("expected EscapeDismissers to run when the interceptor declines")
	}
}

// A chord in flight outranks the interceptor: its continuation keys are plain
// runes that a modal plugin would otherwise swallow.
func TestKeyInterceptorDoesNotBreakChords(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("file.new")

	// Stand in for a modal plugin that consumes every printable key.
	h.app.Root.KeyInterceptor = func(ev *tcell.EventKey) bool {
		return ev.Key() == tcell.KeyRune
	}

	fired := false
	h.app.Root.AddChordKey([]ui.GlobalKeyBinding{
		{Key: tcell.KeyCtrlK, Mod: tcell.ModCtrl},
		{Key: tcell.KeyRune, Rune: 'z'},
	}, func() { fired = true })

	h.pressCtrl(tcell.KeyCtrlK)
	h.pressRune('z')

	if !fired {
		t.Fatal("expected chord to fire despite an interceptor that consumes runes")
	}
}

func TestKeyInterceptorNotCalledForOverlays(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("file.new")

	called := false
	h.app.Root.KeyInterceptor = func(ev *tcell.EventKey) bool {
		called = true
		return true
	}

	h.exec("command.palette")
	h.pressRune('a')

	if called {
		t.Fatal("interceptor should not be called when an overlay is active")
	}
}
