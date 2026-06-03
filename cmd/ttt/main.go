package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/github"
	"github.com/eugenioenko/ttt/internal/lsp"
	"github.com/eugenioenko/ttt/internal/render"
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

var version = "dev"

func initLogger(debug bool) *os.File {
	if !debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))
		return nil
	}
	logPath := config.ConfigFilePath("ttt.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
		return nil
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return f
}

func handlePanic(screen *term.TcellScreen) {
	r := recover()
	if r == nil {
		return
	}
	if screen != nil {
		screen.Fini()
	}
	stack := debug.Stack()
	crashMsg := fmt.Sprintf("ttt crashed: %v\n\n%s", r, stack)
	crashPath := config.ConfigFilePath("crash.log")
	header := fmt.Sprintf("ttt crash report — %s\nVersion: %s\n\n", time.Now().Format(time.RFC3339), version)
	os.WriteFile(crashPath, []byte(header+crashMsg), 0644)
	fmt.Fprintf(os.Stderr, "ttt crashed. Crash log saved to %s\n", crashPath)
	fmt.Fprintln(os.Stderr, crashMsg)
	os.Exit(1)
}

func findConfigFlag() string {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func main() {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			fmt.Printf(`ttt %s - Terminal Text Tool, an IDE for your terminal

Usage: ttt [options] [files/folders/URLs...]

Arguments:
  files               Open one or more files
  folders             Open directories as workspace roots
  .                   Open the current directory
  PR URL              Open a GitHub pull request for review

Options:
  --help, -h          Show this help message
  --version, -v       Show version
  --workspace <file>  Open a saved workspace (.ttt file)
  --config <file>     Use a custom config file

Examples:
  ttt                                           Open current directory
  ttt main.go utils.go                          Open specific files
  ttt ~/projectA ~/projectB                     Multi-root workspace
  ttt . https://github.com/o/r/pull/123         Review a PR with repo tree

Docs: https://tttedit.dev
`, version)
			os.Exit(0)
		case "--version", "-v":
			fmt.Println("ttt " + version)
			os.Exit(0)
		}
	}

	cfg := config.Load(findConfigFlag())
	config.ParseKeyBindings(cfg.Keybindings)

	logFile := initLogger(cfg.Settings.DebugMode)
	if logFile != nil {
		defer logFile.Close()
	}
	slog.Info("starting", "debugMode", cfg.Settings.DebugMode)

	screen, err := term.NewTcellScreen()
	if err != nil {
		panic(err)
	}
	defer screen.Fini()
	defer handlePanic(screen)

	screen.SetStyleMap(buildStyleMap(cfg.Theme))
	screen.SetCursorStyle(term.ParseCursorStyle(cfg.Settings.CursorStyle))

	lspManager := lsp.NewManager(cfg.Settings.LSP)
	defer lspManager.Shutdown()
	lspManager.OnDiagnostics = func(params lsp.PublishDiagnosticsParams) {
		path := uriToPath(params.URI)
		diags := lspToUIDiagnostics(params.Diagnostics)
		slog.Debug("lsp diagnostics", "path", path, "count", len(diags))
		screen.PostEvent(tcell.NewEventInterrupt(&diagnosticsResult{
			path:        path,
			diagnostics: diags,
		}))
	}

	renderer := &render.Renderer{}
	cmdRegistry := command.NewRegistry()
	borders := buildBorderSet(cfg.Theme.Borders)

	app, prURLs := buildApp(&cfg, &borders)
	app.screen = screen
	app.renderer = renderer
	app.lspManager = lspManager

	app.editorGroup.OnError = func(msg string) {
		app.StatusError(msg)
	}
	app.editorGroup.OnFileOpen = func(path, lang, text string) {
		app.NotifyLSPOpen(path, lang, text)
	}
	app.editorGroup.OnFileClose = func(path, lang string) {
		app.NotifyLSPClose(path, lang)
	}
	if path := app.editorGroup.ActiveFilePath(); path != "" {
		if app.editorGroup.Editor != nil && app.editorGroup.Editor.Highlighter != nil {
			lang := app.editorGroup.Editor.Highlighter.Language()
			text := strings.Join(app.editorGroup.Editor.Buf.Lines, "\n")
			app.NotifyLSPOpen(path, lang, text)
		}
	}
	app.problems.OnNavigate = func(file string, line, col int) {
		app.editorGroup.OpenFile(file)
		app.editorGroup.GoToLine(line + 1)
		app.root.SetFocus(app.editorGroup)
	}
	app.references.OnNavigate = func(file string, line, col int) {
		app.editorGroup.OpenFile(file)
		app.editorGroup.GoToLine(line + 1)
		app.root.SetFocus(app.editorGroup)
	}

	app.editorGroup.Editor.OnChange = func() {
		path := app.editorGroup.ActiveFilePath()
		lang := ""
		if app.editorGroup.Editor.Highlighter != nil {
			lang = app.editorGroup.Editor.Highlighter.Language()
		}
		text := strings.Join(app.editorGroup.Editor.Buf.Lines, "\n")
		app.NotifyLSPChange(path, lang, text)
		app.ScheduleAutocomplete()
		app.CheckSignatureHelpTrigger()
	}

	app.keybindings = cfg.Keybindings
	quitPending := false
	running := true
	registerCommands(cmdRegistry, app, &running, &quitPending)
	bindKeys(app.root, cmdRegistry, cfg.Keybindings)

	if len(prURLs) > 0 {
		if !github.IsGHInstalled() {
			app.StatusError("GitHub CLI (gh) is required. Install from https://cli.github.com/")
		} else {
			app.ShowSidebar()
			app.sidebar.SetActivePanel("changes")
			for _, url := range prURLs {
				app.fetchAndOpenPR(url)
			}
		}
	}

	w, h := screen.Size()
	app.root.SetSize(w, h)

	runEventLoop(screen, renderer, app, &running, &quitPending, app.CloseTerminal)
}
