package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/core/clipboard"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/view"
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/gdamore/tcell/v3"
)

func newOpenFileTestApp(t *testing.T) *App {
	t.Helper()
	a := buildTestApp(t, config.DefaultSettings())
	// Status notifications schedule a redraw. A zero screen safely drops that
	// delayed event and keeps error-path tests independent of a terminal.
	a.Screen = &term.TcellScreen{}
	return a
}

func writeOpenFileFixture(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOpenFilePathResolvesRelativeAndAbsolutePaths(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	relativePath := filepath.Join("nested", "relative.txt")
	relativeAbs := filepath.Join(dir, relativePath)
	absolutePath := filepath.Join(dir, "absolute.txt")
	writeOpenFileFixture(t, relativeAbs)
	writeOpenFileFixture(t, absolutePath)

	a := newOpenFileTestApp(t)
	a.openFilePath(relativePath)
	if got := a.EditorGroup.ActiveFilePath(); got != relativeAbs {
		t.Fatalf("relative path: got %q, want %q", got, relativeAbs)
	}

	a.openFilePath(absolutePath)
	if got := a.EditorGroup.ActiveFilePath(); got != absolutePath {
		t.Fatalf("absolute path: got %q, want %q", got, absolutePath)
	}
	if a.Root.Focused != a.EditorGroup {
		t.Fatal("opening a file should focus the editor group")
	}
}

func TestOpenFilePathExpandsTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, "notes", "tilde.txt")
	writeOpenFileFixture(t, path)

	a := newOpenFileTestApp(t)
	a.openFilePath("~/notes/tilde.txt")
	if got := a.EditorGroup.ActiveFilePath(); got != path {
		t.Fatalf("tilde path: got %q, want %q", got, path)
	}
}

func TestOpenFilePathRejectsMissingPath(t *testing.T) {
	a := newOpenFileTestApp(t)
	missing := filepath.Join(t.TempDir(), "missing.txt")

	a.openFilePath(missing)

	if got := a.EditorGroup.OpenFilePaths(); len(got) != 0 {
		t.Fatalf("missing path opened tabs: %v", got)
	}
	if want := "File not found: " + missing; a.Status.Notification != want {
		t.Fatalf("status: got %q, want %q", a.Status.Notification, want)
	}
	if a.Status.NotifyLevel != view.NotifyError {
		t.Fatalf("status level: got %v, want error", a.Status.NotifyLevel)
	}
}

func TestOpenFilePathRejectsDirectory(t *testing.T) {
	a := newOpenFileTestApp(t)
	dir := t.TempDir()

	a.openFilePath(dir)

	if got := a.EditorGroup.OpenFilePaths(); len(got) != 0 {
		t.Fatalf("directory opened tabs: %v", got)
	}
	if want := "Not a regular file: " + dir; a.Status.Notification != want {
		t.Fatalf("status: got %q, want %q", a.Status.Notification, want)
	}
	if a.Status.NotifyLevel != view.NotifyError {
		t.Fatalf("status level: got %v, want error", a.Status.NotifyLevel)
	}
}

func TestOpenFileCommandRegistrationAndDialog(t *testing.T) {
	a := newOpenFileTestApp(t)
	a.Reg = command.NewRegistry()
	registerEditorCommands(a)

	cmd, ok := a.Reg.Get("file.open")
	if !ok {
		t.Fatal("file.open command is not registered")
	}
	if cmd.Title != "Open File..." {
		t.Fatalf("command title: got %q, want %q", cmd.Title, "Open File...")
	}
	if found, ok := a.Reg.FindByTitle("Open File..."); !ok || found.ID != "file.open" {
		t.Fatalf("command palette lookup: got %+v, found=%v", found, ok)
	}

	cmd.Handler()
	adapter, ok := a.Root.TopOverlayWidget().(*ui.WidgetAdapter)
	if !ok {
		t.Fatalf("dialog overlay has type %T", a.Root.TopOverlayWidget())
	}
	dialog, ok := adapter.W.(*widgets.DialogWidget)
	if !ok {
		t.Fatalf("dialog widget has type %T", adapter.W)
	}
	if dialog.Title != "Open File" {
		t.Fatalf("dialog title: got %q, want %q", dialog.Title, "Open File")
	}
	input, ok := dialog.Content.(*widgets.InputWidget)
	if !ok {
		t.Fatalf("dialog content has type %T", dialog.Content)
	}
	if input.Config.Placeholder != "File path" {
		t.Fatalf("placeholder: got %q, want %q", input.Config.Placeholder, "File path")
	}
	if len(dialog.Buttons) != 2 || dialog.Buttons[1].Label != "&Open" {
		t.Fatalf("dialog buttons: got %+v", dialog.Buttons)
	}
}

func TestOpenFileDialogTerminalPasteRoutesToInput(t *testing.T) {
	a := newOpenFileTestApp(t)
	a.OpenFile()

	adapter := a.Root.TopOverlayWidget().(*ui.WidgetAdapter)
	dialog := adapter.W.(*widgets.DialogWidget)
	input := dialog.Content.(*widgets.InputWidget)

	a.PasteText("/tmp/pasted.txt")

	if got := input.Text(); got != "/tmp/pasted.txt" {
		t.Fatalf("open file input text: got %q, want %q", got, "/tmp/pasted.txt")
	}
}

func TestGlobalClipboardCommandsRouteToAdapterInputSelection(t *testing.T) {
	clipboard.DisableSystem()
	a := newOpenFileTestApp(t)
	a.OpenFile()

	adapter := a.Root.TopOverlayWidget().(*ui.WidgetAdapter)
	dialog := adapter.W.(*widgets.DialogWidget)
	input := dialog.Content.(*widgets.InputWidget)
	input.SetText("alpha beta")

	for range 4 {
		adapter.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, "", tcell.ModShift))
	}
	clipboard.Set("sentinel")
	a.Copy()
	if got := clipboard.Get(); got != "beta" {
		t.Fatalf("copied text: got %q, want %q", got, "beta")
	}

	clipboard.Set("gamma")
	a.Paste()
	if got := input.Text(); got != "alpha gamma" {
		t.Fatalf("text after paste: got %q, want %q", got, "alpha gamma")
	}

	for range 5 {
		adapter.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, "", tcell.ModShift))
	}
	a.Cut()
	if got := input.Text(); got != "alpha " {
		t.Fatalf("text after cut: got %q, want %q", got, "alpha ")
	}
	if got := clipboard.Get(); got != "gamma" {
		t.Fatalf("cut text: got %q, want %q", got, "gamma")
	}
}

func TestSettingsContentAdapterRoutesPasteAndGlobalClipboardCommands(t *testing.T) {
	clipboard.DisableSystem()
	a := newOpenFileTestApp(t)
	a.ShowSettings()

	if a.Root.Focused != a.EditorGroup {
		t.Fatalf("focused widget is %T, want editor group", a.Root.Focused)
	}
	adapter := a.settingsView.adapter
	adapter.HandleEvent(tcell.NewEventKey(tcell.KeyTab, "", tcell.ModNone))
	input := adapter.FocusedInput()
	if input == nil {
		t.Fatal("settings adapter has no focused input")
	}

	input.SetText("")
	a.PasteText("12")
	if got := input.Text(); got != "12" {
		t.Fatalf("text after terminal paste: got %q, want %q", got, "12")
	}

	input.SetText("alpha beta")
	for range 4 {
		adapter.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, "", tcell.ModShift))
	}
	clipboard.Set("sentinel")
	a.Copy()
	if got := clipboard.Get(); got != "beta" {
		t.Fatalf("copied text: got %q, want %q", got, "beta")
	}

	clipboard.Set("gamma")
	a.Paste()
	if got := input.Text(); got != "alpha gamma" {
		t.Fatalf("text after clipboard paste: got %q, want %q", got, "alpha gamma")
	}

	for range 5 {
		adapter.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, "", tcell.ModShift))
	}
	a.Cut()
	if got := input.Text(); got != "alpha " {
		t.Fatalf("text after cut: got %q, want %q", got, "alpha ")
	}
	if got := clipboard.Get(); got != "gamma" {
		t.Fatalf("cut text: got %q, want %q", got, "gamma")
	}
}

func TestAdapterWithoutFocusedInputPreservesEditorPasteFallback(t *testing.T) {
	a := newOpenFileTestApp(t)
	button := widgets.NewButtonWidget(widgets.ButtonConfig{Label: "Action"})
	a.ShowDialog(ui.NewWidgetAdapter(button))

	a.PasteText("editor fallback")

	if got := a.EditorGroup.Editor.Buf.Lines[0]; got != "editor fallback" {
		t.Fatalf("editor text: got %q, want %q", got, "editor fallback")
	}
}

func TestFileMenuIncludesOpenFileNextToNewFile(t *testing.T) {
	fileMenu := menuBarMenus[0]
	for i, item := range fileMenu {
		if item.Command != "file.new" {
			continue
		}
		if i+1 >= len(fileMenu) {
			t.Fatal("New File is the last File menu item")
		}
		open := fileMenu[i+1]
		if open.Label != "Open File..." || open.Command != "file.open" {
			t.Fatalf("item after New File: got %+v", open)
		}
		return
	}
	t.Fatal("File menu does not contain New File")
}
