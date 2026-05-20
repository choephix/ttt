package main

import (
	"ttt/internal/command"
	"ttt/internal/config"
	"ttt/internal/render"
	"ttt/internal/term"
)

func main() {
	cfg := config.Load()
	config.ParseKeyBindings(cfg.Keybindings)

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

	quitPending := false
	running := true
	registerCommands(cmdRegistry, app, &running, &quitPending)
	bindKeys(app.root, cmdRegistry, cfg.Keybindings)

	w, h := screen.Size()
	app.root.SetSize(w, h)

	runEventLoop(screen, renderer, cmdRegistry, app, &running, &quitPending)
}
