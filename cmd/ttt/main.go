package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"path/filepath"

	"github.com/eugenioenko/ttt/internal/app"
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/core/clipboard"
	"github.com/eugenioenko/ttt/internal/github"
	"github.com/eugenioenko/ttt/internal/lsp"
	"github.com/eugenioenko/ttt/internal/plugin"
	"github.com/eugenioenko/ttt/internal/render"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/ui"

	"github.com/gdamore/tcell/v2"
)

var (
	version         = "dev"
	profilerEnabled bool
)

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
			fmt.Printf(`ttt %s - TTT Editor, Terminal Text Tool, an IDE for your terminal

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

	if profilerEnabled {
		defer startProfiler()()
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

	screen.SetStyleMap(app.BuildStyleMap(cfg.Theme))
	screen.SetCursorStyle(term.ParseCursorStyle(cfg.Settings.Editor.CursorStyle))

	// Route OSC 52 clipboard writes through the tty, not raw stderr
	if tty, ok := screen.Tty(); ok {
		clipboard.SetOSCWriter(tty)
	}

	lspManager := lsp.NewManager(cfg.Settings.LSP)
	defer lspManager.Shutdown()

	renderer := &render.Renderer{}
	cmdRegistry := command.NewRegistry()
	borders := app.BuildBorderSet(cfg.Theme.Borders)

	editor, prURLs := app.BuildApp(&cfg, &borders)
	editor.ApplyBorderStyle()
	editor.Init(screen, renderer, lspManager)

	editor.Version = version
	editor.Keybindings = cfg.Keybindings
	editor.Reg = cmdRegistry
	running := true
	editor.Running = &running
	app.RegisterCommands(editor)
	app.BindKeys(editor.Root, cmdRegistry, cfg.Keybindings)

	pluginsDir := filepath.Join(filepath.Dir(config.ConfigFilePath("plugins.ttt.json")), "plugins")
	localPluginsDir := filepath.Join(editor.Workspace.Primary(), "plugins")
	pluginManager := plugin.NewManager(pluginsDir, localPluginsDir)
	pendingApprovals := pluginManager.LoadAll()
	editor.PluginManager = pluginManager
	defer pluginManager.Shutdown()

	for _, reg := range pluginManager.SidebarPanels {
		editor.Sidebar.AddPanel(reg.ID, reg.Title, ui.NewWidgetAdapter(reg.Widget))
	}
	for _, reg := range pluginManager.BottomPanels {
		editor.BottomPanel.AddPanel(reg.ID, reg.Title, ui.NewWidgetAdapter(reg.Widget))
	}
	pluginManager.SetEditorAPI(app.NewPluginEditorAPI(editor))
	pluginManager.SetFilesystemAPI(app.NewPluginFilesystemAPI())
	pluginManager.SetSystemAPI(app.NewPluginSystemAPI())
	pluginManager.SetNetworkAPI(app.NewPluginNetworkAPI())

	for _, p := range pluginManager.Plugins() {
		p.RequestRedraw = func() {
			screen.PostEvent(tcell.NewEventInterrupt(nil))
		}
		p.PostAsync = func(result *plugin.PluginAsyncResult) {
			screen.PostEvent(tcell.NewEventInterrupt(result))
		}
	}
	pluginManager.SetLogFactory(func(pluginName string) func(string, string) {
		return func(level, message string) {
			editor.Output.AddLine(ui.OutputLine{
				Time:       time.Now().Format("15:04:05"),
				PluginName: pluginName,
				Level:      level,
				Message:    message,
			})
			screen.PostEvent(tcell.NewEventInterrupt(nil))
		}
	})

	editor.RegisterStartupPluginCommands()

	pluginsPanel := app.NewPluginsPanel(pluginManager)
	editor.Sidebar.AddPanel("plugins", "Plugins", pluginsPanel.Adapter)
	editor.PluginsPanel = pluginsPanel
	pluginsPanel.OnInstall = func(repoURL string) {
		editor.PluginInstallFromURL(repoURL)
	}
	pluginsPanel.OnUninstall = func(name string) {
		editor.PluginUninstallByName(name)
	}
	pluginsPanel.OnToggle = func(name string, enabled bool) {
		if err := pluginManager.SetEnabled(name, enabled); err != nil {
			slog.Error("toggle plugin", "error", err)
		}
		pluginsPanel.Refresh()
	}
	pluginsPanel.OnUpdate = func(name string) {
		editor.PluginUpdateByName(name)
	}

	go func() {
		entries, err := plugin.FetchRemoteRegistry(plugin.DefaultRegistryURL)
		if err != nil {
			slog.Debug("fetch plugin registry", "error", err)
			return
		}
		screen.PostEvent(tcell.NewEventInterrupt(&app.RemoteRegistryResult{Entries: entries}))
	}()

	if len(pendingApprovals) > 0 {
		editor.PendingPluginApprovals = pendingApprovals
	}

	if len(prURLs) > 0 {
		if !github.IsGHInstalled() {
			editor.StatusError("GitHub CLI (gh) is required. Install from https://cli.github.com/")
		} else {
			editor.ShowSidebar()
			editor.Sidebar.SetActivePanel("changes")
			for _, url := range prURLs {
				editor.FetchAndOpenPR(url)
			}
		}
	}

	w, h := screen.Size()
	editor.Root.SetSize(w, h)

	app.RunEventLoop(screen, renderer, editor, &running, editor.CloseTerminal)
}
