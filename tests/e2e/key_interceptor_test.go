package e2e

import (
	"testing"

	"github.com/gdamore/tcell/v2"
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
