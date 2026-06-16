package e2e

import (
	"strings"
	"testing"
)

func TestDefaultSettingsCommandRegistered(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	cmd, ok := h.reg.Get("options.defaultSettings")
	if !ok {
		t.Fatal("expected options.defaultSettings command to be registered")
	}
	if cmd.Title != "Preferences: Open Default Settings" {
		t.Errorf("expected title 'Preferences: Open Default Settings', got %q", cmd.Title)
	}
}

func TestDefaultSettingsOpensBuffer(t *testing.T) {
	h := newTestHarness(t, 120, 40)
	defer h.stop()

	h.exec("options.defaultSettings")

	// The tab bar should show the default settings tab
	h.assertContains("Default Settings")

	// The buffer should contain settings reference content
	screen := h.screenText()
	if !strings.Contains(screen, "Default Settings") {
		t.Errorf("expected screen to show default settings tab, got:\n%s", screen)
	}
}

func TestDefaultSettingsShowsSettingsSections(t *testing.T) {
	h := newTestHarness(t, 120, 40)
	defer h.stop()

	h.exec("options.defaultSettings")

	// The screen should show the beginning of the default settings reference
	screen := h.screenText()
	if !strings.Contains(screen, "Default Settings Reference") {
		t.Errorf("expected screen to contain 'Default Settings Reference', got:\n%s", screen)
	}
}

func TestDefaultSettingsReopensExistingTab(t *testing.T) {
	h := newTestHarness(t, 120, 40)
	defer h.stop()

	// Open default settings twice
	h.exec("options.defaultSettings")
	h.exec("options.defaultSettings")

	// Should still only have the default settings tab (plus the initial file tab)
	// The OpenBuffer method deduplicates by path, so a second call should
	// switch to the existing tab, not create a new one.
	screen := h.screenText()
	// Count occurrences of the tab name — should appear exactly once in the tab bar
	tabBar := strings.Split(screen, "\n")[1] // tab bar is typically the second row
	count := strings.Count(tabBar, "Default Settings")
	if count > 1 {
		t.Errorf("expected only one 'Default Settings' tab, found %d in tab bar: %q", count, tabBar)
	}
}
