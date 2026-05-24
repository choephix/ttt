package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"ttt/internal/command"
	"ttt/internal/config"
	"ttt/internal/render"
	"ttt/internal/term"
	"ttt/internal/ui"
	"ttt/internal/workspace"

	"github.com/gdamore/tcell/v2"
)

type testHarness struct {
	t        *testing.T
	app      *App
	screen   tcell.SimulationScreen
	reg      *command.Registry
	renderer *render.Renderer
	running  bool
}

func newTestHarness(t *testing.T, w, h int) *testHarness {
	t.Helper()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "alpha.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "beta.txt"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(dir, "gamma.txt"), []byte("c"), 0644)
	os.WriteFile(filepath.Join(dir, "delta.txt"), []byte("d"), 0644)
	os.WriteFile(filepath.Join(dir, "epsilon.txt"), []byte("e"), 0644)
	os.WriteFile(filepath.Join(dir, "zeta.txt"), []byte("f"), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "subdir", "nested.txt"), []byte("nested"), 0644)

	sim := tcell.NewSimulationScreen("")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	sim.SetSize(w, h)

	cfg := config.AppConfig{
		Keybindings: config.DefaultKeybindings(),
		Settings:    config.DefaultSettings(),
		Theme:       config.DefaultTheme(),
	}
	config.ParseKeyBindings(cfg.Keybindings)

	screen := term.NewTcellScreenFrom(sim)
	screen.SetStyleMap(buildStyleMap(cfg.Theme))

	borders := buildBorderSet(cfg.Theme.Borders)

	ws := workspace.New([]string{dir})
	app := buildAppFromConfig(&cfg, &borders, ws, nil)
	app.screen = screen
	app.renderer = &render.Renderer{}

	reg := command.NewRegistry()
	quitPending := false
	running := true
	registerCommands(reg, app, &running, &quitPending)
	bindKeys(app.root, reg, cfg.Keybindings)

	app.root.SetSize(w, h)

	h2 := &testHarness{
		t:        t,
		app:      app,
		screen:   sim,
		reg:      reg,
		renderer: app.renderer,
		running:  running,
	}
	h2.redraw()
	return h2
}

func (h *testHarness) redraw() {
	h.t.Helper()
	cells := make([][]term.Cell, h.app.root.Height)
	for y := range cells {
		cells[y] = make([]term.Cell, h.app.root.Width)
	}
	h.app.root.Render(cells)
	h.renderer.SetCurrent(cells)
	h.renderer.Render(h.app.screen)
}

func (h *testHarness) pressKey(key tcell.Key, mod tcell.ModMask) {
	h.t.Helper()
	ev := tcell.NewEventKey(key, 0, mod)
	h.app.root.HandleEvent(ev)
	h.redraw()
}

func (h *testHarness) pressRune(r rune) {
	h.t.Helper()
	ev := tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
	h.app.root.HandleEvent(ev)
	h.redraw()
}

func (h *testHarness) pressCtrl(key tcell.Key) {
	h.t.Helper()
	h.pressKey(key, tcell.ModCtrl)
}

func (h *testHarness) click(x, y int) {
	h.t.Helper()
	ev := tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone)
	h.app.root.HandleEvent(ev)
	h.redraw()
}

func (h *testHarness) exec(cmdID string) {
	h.t.Helper()
	h.reg.Execute(cmdID)
	h.redraw()
}

func (h *testHarness) screenText() string {
	cells, w, ht := h.screen.GetContents()
	var lines []string
	for y := 0; y < ht; y++ {
		var line strings.Builder
		for x := 0; x < w; x++ {
			sc := cells[y*w+x]
			ch := ' '
			if len(sc.Runes) > 0 {
				ch = sc.Runes[0]
			}
			line.WriteRune(ch)
		}
		lines = append(lines, line.String())
	}
	return strings.Join(lines, "\n")
}

func (h *testHarness) screenRow(y int) string {
	cells, w, _ := h.screen.GetContents()
	var line strings.Builder
	for x := 0; x < w; x++ {
		sc := cells[y*w+x]
		ch := ' '
		if len(sc.Runes) > 0 {
			ch = sc.Runes[0]
		}
		line.WriteRune(ch)
	}
	return line.String()
}

func (h *testHarness) containsText(text string) bool {
	return strings.Contains(h.screenText(), text)
}

func (h *testHarness) assertContains(text string) {
	h.t.Helper()
	if !h.containsText(text) {
		h.t.Errorf("expected screen to contain %q, got:\n%s", text, h.screenText())
	}
}

func (h *testHarness) assertNotContains(text string) {
	h.t.Helper()
	if h.containsText(text) {
		h.t.Errorf("expected screen NOT to contain %q, got:\n%s", text, h.screenText())
	}
}

func (h *testHarness) stop() {
	h.screen.Fini()
}

type emptyWidget struct{ ui.BaseWidget }

func newEmptyWidget() *emptyWidget                                       { return &emptyWidget{} }
func (e *emptyWidget) Focusable() bool                                   { return false }
func (e *emptyWidget) Render(surface *ui.RenderSurface)                  {}
func (e *emptyWidget) HandleEvent(ev tcell.Event) ui.EventResult { return ui.EventIgnored }

func TestSidebarTabClick(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	if h.app.sidebar.ActivePanel != "explorer" {
		t.Fatalf("expected active panel 'explorer', got %q", h.app.sidebar.ActivePanel)
	}

	// Sidebar tabs render at y=1 (row below menu bar).
	// Tabs are: " Files " (x=0..6), " Search " (x=7..14), " Changes " (x=15..23)
	sidebarY := h.app.sidebar.GetRect().Y
	sidebarX := h.app.sidebar.GetRect().X

	// Click on "Search" tab (within x=7..14)
	h.click(sidebarX+10, sidebarY)
	if h.app.sidebar.ActivePanel != "search" {
		t.Errorf("expected active panel 'search' after click, got %q", h.app.sidebar.ActivePanel)
	}

	// Click on "Changes" tab (within x=15..23)
	h.click(sidebarX+18, sidebarY)
	if h.app.sidebar.ActivePanel != "changes" {
		t.Errorf("expected active panel 'changes' after click, got %q", h.app.sidebar.ActivePanel)
	}

	// Click back on "Files" tab (within x=0..6)
	h.click(sidebarX+3, sidebarY)
	if h.app.sidebar.ActivePanel != "explorer" {
		t.Errorf("expected active panel 'explorer' after click, got %q", h.app.sidebar.ActivePanel)
	}
}

func TestBottomPanelTabClick(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	// Add two panels to bottom panel
	h.app.bottomPanel.AddPanel("test-a", "Alpha", newEmptyWidget())
	h.app.bottomPanel.AddPanel("test-b", "Beta", newEmptyWidget())
	h.app.contentSplit.ShowBottom = true
	h.app.contentSplit.BottomH = 10
	h.redraw()

	if h.app.bottomPanel.ActivePanel != "test-a" {
		t.Fatalf("expected active panel 'test-a', got %q", h.app.bottomPanel.ActivePanel)
	}

	// Bottom panel tabs at the top of the bottom panel area
	panelY := h.app.bottomPanel.GetRect().Y
	panelX := h.app.bottomPanel.GetRect().X

	// Click on "Beta" tab: " Alpha " is 7 chars, " Beta " starts at x=7
	h.click(panelX+9, panelY)
	if h.app.bottomPanel.ActivePanel != "test-b" {
		t.Errorf("expected active panel 'test-b' after click, got %q", h.app.bottomPanel.ActivePanel)
	}

	// Click back on "Alpha" tab
	h.click(panelX+3, panelY)
	if h.app.bottomPanel.ActivePanel != "test-a" {
		t.Errorf("expected active panel 'test-a' after click, got %q", h.app.bottomPanel.ActivePanel)
	}
}

func TestTabbedPanelRemovePanel(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.app.bottomPanel.AddPanel("p1", "One", newEmptyWidget())
	h.app.bottomPanel.AddPanel("p2", "Two", newEmptyWidget())
	h.app.bottomPanel.AddPanel("p3", "Three", newEmptyWidget())
	h.app.bottomPanel.SetActivePanel("p2")

	if h.app.bottomPanel.PanelCount() != 3 {
		t.Fatalf("expected 3 panels, got %d", h.app.bottomPanel.PanelCount())
	}

	// Remove active panel, should switch to next
	h.app.bottomPanel.RemovePanel("p2")
	if h.app.bottomPanel.PanelCount() != 2 {
		t.Fatalf("expected 2 panels, got %d", h.app.bottomPanel.PanelCount())
	}
	if h.app.bottomPanel.ActivePanel == "p2" {
		t.Error("active panel should have changed after removing it")
	}

	// Remove all
	h.app.bottomPanel.RemovePanel("p1")
	h.app.bottomPanel.RemovePanel("p3")
	if h.app.bottomPanel.PanelCount() != 0 {
		t.Fatalf("expected 0 panels, got %d", h.app.bottomPanel.PanelCount())
	}
}

func TestStartup(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.assertContains("File")
	h.assertContains("Edit")
	h.assertContains("View")
	h.assertContains("Files")
}

func TestToggleSidebar(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.assertContains("Files")

	h.exec("sidebar.toggle")
	h.assertNotContains("Files")

	h.exec("sidebar.toggle")
	h.assertContains("Files")
}

func TestTogglePanel(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	if h.app.contentSplit.ShowBottom {
		t.Error("bottom panel should start hidden")
	}

	h.exec("panel.toggle")
	if !h.app.contentSplit.ShowBottom {
		t.Error("bottom panel should be visible after toggle")
	}

	h.exec("panel.toggle")
	if h.app.contentSplit.ShowBottom {
		t.Error("bottom panel should be hidden after second toggle")
	}
}

func TestSidebarPanelSwitching(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")
	h.assertContains("Files")

	h.exec("sidebar.search")
	h.assertContains("Search")

	h.exec("sidebar.changes")
	h.assertContains("Changes")
}

func TestCommandPaletteOpenClose(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("command.palette")
	if len(h.app.root.Overlays) != 1 {
		t.Fatalf("expected 1 overlay, got %d", len(h.app.root.Overlays))
	}

	h.pressKey(tcell.KeyEscape, tcell.ModNone)
	if len(h.app.root.Overlays) != 0 {
		t.Fatalf("expected 0 overlays after Escape, got %d", len(h.app.root.Overlays))
	}
}

func TestGoToLineDialog(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("editor.goToLine")
	if len(h.app.root.Overlays) != 1 {
		t.Fatalf("expected 1 overlay, got %d", len(h.app.root.Overlays))
	}

	h.pressKey(tcell.KeyEscape, tcell.ModNone)
	if len(h.app.root.Overlays) != 0 {
		t.Fatalf("expected 0 overlays after Escape, got %d", len(h.app.root.Overlays))
	}
}

func TestFindDialog(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("search.find")
	if len(h.app.root.Overlays) != 1 {
		t.Fatalf("expected 1 overlay, got %d", len(h.app.root.Overlays))
	}

	h.pressKey(tcell.KeyEscape, tcell.ModNone)
	if len(h.app.root.Overlays) != 0 {
		t.Fatalf("expected 0 overlays after Escape, got %d", len(h.app.root.Overlays))
	}
}

func TestNewFile(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("file.new")
	if h.app.editorGroup.ActiveFilePath() != "untitled" {
		t.Errorf("expected path 'untitled', got %q", h.app.editorGroup.ActiveFilePath())
	}
	h.assertContains("untitled")
}

func TestSidebarWidth(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	initial := h.app.splitPanel.DividerPos

	h.exec("sidebar.wider")
	if h.app.splitPanel.DividerPos != initial+1 {
		t.Errorf("expected width %d, got %d", initial+1, h.app.splitPanel.DividerPos)
	}

	h.exec("sidebar.narrower")
	if h.app.splitPanel.DividerPos != initial {
		t.Errorf("expected width %d, got %d", initial, h.app.splitPanel.DividerPos)
	}
}

func TestMenuBarRendered(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	row := h.screenRow(0)
	if !strings.Contains(row, "File") {
		t.Errorf("menu bar should contain 'File', got: %s", row)
	}
	if !strings.Contains(row, "Help") {
		t.Errorf("menu bar should contain 'Help', got: %s", row)
	}
}

func TestThemeSwitchDialog(t *testing.T) {
	// theme.switch returns early if no theme files found,
	// so this test only verifies it doesn't crash
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("theme.switch")
	// If theme files exist, overlay opens; otherwise it's a no-op
	if len(h.app.root.Overlays) == 1 {
		h.pressKey(tcell.KeyEscape, tcell.ModNone)
		if len(h.app.root.Overlays) != 0 {
			t.Fatalf("expected 0 overlays after Escape, got %d", len(h.app.root.Overlays))
		}
	}
}

func TestExplorerKeyNavigation(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")

	if len(h.app.explorer.FlatList) < 3 {
		t.Skipf("expected at least 3 explorer items, got %d", len(h.app.explorer.FlatList))
	}

	h.app.explorer.Selected = 0

	h.pressKey(tcell.KeyDown, tcell.ModNone)
	if h.app.explorer.Selected != 1 {
		t.Errorf("expected Selected 1 after Down, got %d", h.app.explorer.Selected)
	}

	h.pressKey(tcell.KeyDown, tcell.ModNone)
	if h.app.explorer.Selected != 2 {
		t.Errorf("expected Selected 2 after second Down, got %d", h.app.explorer.Selected)
	}

	h.pressKey(tcell.KeyUp, tcell.ModNone)
	if h.app.explorer.Selected != 1 {
		t.Errorf("expected Selected 1 after Up, got %d", h.app.explorer.Selected)
	}

	h.app.explorer.Selected = 0
	h.pressKey(tcell.KeyUp, tcell.ModNone)
	if h.app.explorer.Selected != 0 {
		t.Errorf("expected Selected 0 (clamped at top), got %d", h.app.explorer.Selected)
	}
}

func TestExplorerDirExpandCollapse(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")

	h.app.explorer.Selected = 0
	root := h.app.explorer.FlatList[0]
	if !root.IsDir {
		t.Fatal("expected root to be a directory")
	}

	initialCount := len(h.app.explorer.FlatList)

	h.pressKey(tcell.KeyLeft, tcell.ModNone)
	if root.Expanded {
		t.Error("expected root to be collapsed after Left")
	}
	if len(h.app.explorer.FlatList) >= initialCount {
		t.Error("expected fewer items after collapsing root")
	}

	h.pressKey(tcell.KeyRight, tcell.ModNone)
	if !root.Expanded {
		t.Error("expected root to be expanded after Right")
	}
	if len(h.app.explorer.FlatList) != initialCount {
		t.Errorf("expected %d items after re-expanding, got %d", initialCount, len(h.app.explorer.FlatList))
	}
}

func TestExplorerEnterOpensFile(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")

	fileIdx := -1
	for i, node := range h.app.explorer.FlatList {
		if !node.IsDir {
			fileIdx = i
			break
		}
	}
	if fileIdx < 0 {
		t.Skip("no file found in explorer")
	}

	h.app.explorer.Selected = fileIdx
	expectedPath := h.app.explorer.FlatList[fileIdx].Path

	h.pressKey(tcell.KeyEnter, tcell.ModNone)

	if h.app.editorGroup.ActiveFilePath() != expectedPath {
		t.Errorf("expected editor to open %q, got %q", expectedPath, h.app.editorGroup.ActiveFilePath())
	}
}

func TestExplorerEnterToggleDir(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")

	h.app.explorer.Selected = 0
	root := h.app.explorer.FlatList[0]
	if !root.IsDir || !root.Expanded {
		t.Fatal("expected root to be an expanded directory")
	}

	h.pressKey(tcell.KeyEnter, tcell.ModNone)
	if root.Expanded {
		t.Error("expected root to be collapsed after Enter")
	}

	h.pressKey(tcell.KeyEnter, tcell.ModNone)
	if !root.Expanded {
		t.Error("expected root to be expanded after second Enter")
	}
}

func TestExplorerClickOpensFile(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")
	h.redraw()

	fileIdx := -1
	for i, node := range h.app.explorer.FlatList {
		if !node.IsDir {
			fileIdx = i
			break
		}
	}
	if fileIdx < 0 {
		t.Skip("no file found in explorer")
	}

	r := h.app.explorer.GetRect()
	clickY := r.Y + (fileIdx - h.app.explorer.ScrollTop)
	h.click(r.X+5, clickY)

	if h.app.explorer.Selected != fileIdx {
		t.Errorf("expected Selected %d after click, got %d", fileIdx, h.app.explorer.Selected)
	}

	expectedPath := h.app.explorer.FlatList[fileIdx].Path
	if h.app.editorGroup.ActiveFilePath() != expectedPath {
		t.Errorf("expected editor to open %q, got %q", expectedPath, h.app.editorGroup.ActiveFilePath())
	}
}

func TestExplorerScrollFollowing(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")

	itemCount := len(h.app.explorer.FlatList)
	if itemCount < 5 {
		t.Skipf("need at least 5 items for scroll test, got %d", itemCount)
	}

	h.app.explorer.Selected = itemCount - 1
	r := h.app.explorer.GetRect()
	contentH := r.H

	h.redraw()

	if contentH > 0 && itemCount > contentH {
		if h.app.explorer.ScrollTop == 0 {
			t.Error("expected ScrollTop > 0 when selected item is past visible area")
		}
	}
}

func TestChangesKeyNavigation(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.changes")

	if h.app.changes.TotalChanges() == 0 {
		t.Skip("no changed files in working directory")
	}

	h.app.changes.Selected = 0
	h.pressKey(tcell.KeyDown, tcell.ModNone)
	h.pressKey(tcell.KeyUp, tcell.ModNone)
	if h.app.changes.Selected != 0 {
		t.Errorf("expected Selected 0 after Up, got %d", h.app.changes.Selected)
	}
}

func TestChangesRefreshKey(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.changes")

	h.pressRune('r')
	// Just verify refresh doesn't crash
}

func TestFocusEditor(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.focus")
	if h.app.root.Focused == h.app.editorGroup {
		t.Error("focus should not be on editor after sidebar.focus")
	}

	h.exec("editor.focus")
	if h.app.root.Focused != h.app.editorGroup {
		t.Error("focus should be on editor after editor.focus")
	}
}
