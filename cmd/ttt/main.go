package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/lsp"
	"github.com/eugenioenko/ttt/internal/render"
	"github.com/eugenioenko/ttt/internal/term"
)

func initLogger(debug bool) *os.File {
	level := slog.LevelError
	if debug {
		level = slog.LevelDebug
	}
	f, err := os.OpenFile("ttt.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
		return nil
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})))
	return f
}

func main() {
	cfg := config.Load()
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

	screen.SetStyleMap(buildStyleMap(cfg.Theme))
	screen.SetCursorStyle(term.ParseCursorStyle(cfg.Settings.CursorStyle))

	lspManager := lsp.NewManager(cfg.Settings.LSP)
	defer lspManager.Shutdown()

	renderer := &render.Renderer{}
	cmdRegistry := command.NewRegistry()
	borders := buildBorderSet(cfg.Theme.Borders)

	app := buildApp(&cfg, &borders)
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
	app.editorGroup.Editor.OnChange = func() {
		path := app.editorGroup.ActiveFilePath()
		lang := ""
		if app.editorGroup.Editor.Highlighter != nil {
			lang = app.editorGroup.Editor.Highlighter.Language()
		}
		text := strings.Join(app.editorGroup.Editor.Buf.Lines, "\n")
		app.NotifyLSPChange(path, lang, text)
	}

	quitPending := false
	running := true
	registerCommands(cmdRegistry, app, &running, &quitPending)
	bindKeys(app.root, cmdRegistry, cfg.Keybindings)

	w, h := screen.Size()
	app.root.SetSize(w, h)

	runEventLoop(screen, renderer, app, &running, &quitPending, app.CloseTerminal)
}
