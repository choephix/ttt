package main

import (
	"log/slog"
	"os"
	"ttt/internal/command"
	"ttt/internal/config"
	"ttt/internal/render"
	"ttt/internal/term"
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

	renderer := &render.Renderer{}
	cmdRegistry := command.NewRegistry()
	borders := buildBorderSet(cfg.Theme.Borders)

	app := buildWidgets(&cfg, &borders)
	app.screen = screen
	app.renderer = renderer

	quitPending := false
	running := true
	registerCommands(cmdRegistry, app, &running, &quitPending)
	bindKeys(app.root, cmdRegistry, cfg.Keybindings)

	w, h := screen.Size()
	app.root.SetSize(w, h)

	runEventLoop(screen, renderer, app, &running, &quitPending)
}
