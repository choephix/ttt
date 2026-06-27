package app

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eugenioenko/ttt/internal/config"

	"github.com/gdamore/tcell/v2"
)

// RunExecScript parses a semicolon-separated script string and executes
// each command sequentially. Intended to be run in a goroutine after
// the event loop starts.
func RunExecScript(a *App, script string) {
	commands := strings.Split(script, ";")
	for _, raw := range commands {
		cmd := strings.TrimSpace(raw)
		if cmd == "" {
			continue
		}
		action, args := parseExecCommand(cmd)
		slog.Debug("exec_script", "action", action, "args", args)

		switch action {
		case "click":
			execClick(a, args)
		case "key":
			execKey(a, args)
		case "type":
			execType(a, args)
		case "exec":
			execCommand(a, args)
		case "screenshot":
			execScreenshot(a, args)
		case "debug":
			execDebug(a, args)
		case "wait":
			execWait(args)
		case "quit":
			execQuit(a)
		default:
			slog.Error("exec_script: unknown command", "action", action)
		}

		// Small implicit delay between commands to let the event loop process
		time.Sleep(50 * time.Millisecond)
	}
}

// parseExecCommand splits a command string into action and arguments.
// Handles quoted strings for the exec command (e.g., exec "Command Name").
func parseExecCommand(cmd string) (string, string) {
	// Find the first space to split action from args
	idx := strings.IndexByte(cmd, ' ')
	if idx < 0 {
		return cmd, ""
	}
	return cmd[:idx], strings.TrimSpace(cmd[idx+1:])
}

func execClick(a *App, args string) {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		slog.Error("exec_script: click requires X Y", "args", args)
		return
	}
	x, err := strconv.Atoi(parts[0])
	if err != nil {
		slog.Error("exec_script: invalid click X", "value", parts[0], "error", err)
		return
	}
	y, err := strconv.Atoi(parts[1])
	if err != nil {
		slog.Error("exec_script: invalid click Y", "value", parts[1], "error", err)
		return
	}

	// Press
	a.Screen.PostEvent(tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone))
	time.Sleep(50 * time.Millisecond)
	// Release
	a.Screen.PostEvent(tcell.NewEventMouse(x, y, tcell.ButtonNone, tcell.ModNone))
}

func execKey(a *App, args string) {
	combo := strings.TrimSpace(args)
	if combo == "" {
		slog.Error("exec_script: key requires a key combo")
		return
	}

	steps, err := config.ParseKeyString(combo)
	if err != nil {
		slog.Error("exec_script: invalid key combo", "combo", combo, "error", err)
		return
	}

	for _, step := range steps {
		key, mod, ch := comboToTcell(step)
		a.Screen.PostEvent(tcell.NewEventKey(key, ch, mod))
		time.Sleep(30 * time.Millisecond)
	}
}

func execType(a *App, args string) {
	text := stripQuotes(args)
	for _, r := range text {
		a.Screen.PostEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
		time.Sleep(10 * time.Millisecond)
	}
}

func execCommand(a *App, args string) {
	title := stripQuotes(strings.TrimSpace(args))
	if title == "" {
		slog.Error("exec_script: exec requires a command title")
		return
	}

	cmd, ok := a.Reg.FindByTitle(title)
	if !ok {
		slog.Error("exec_script: command not found", "title", title)
		return
	}
	a.Reg.Execute(cmd.ID)
}

func execScreenshot(a *App, args string) {
	path := stripQuotes(strings.TrimSpace(args))
	if path == "" {
		slog.Error("exec_script: screenshot requires a file path")
		return
	}
	// Trigger a redraw so the screen is up-to-date
	a.Screen.PostEvent(tcell.NewEventInterrupt(nil))
	time.Sleep(50 * time.Millisecond)

	if err := a.DumpScreenshot(path); err != nil {
		slog.Error("exec_script: screenshot failed", "path", path, "error", err)
	}
}

func execDebug(a *App, args string) {
	path := stripQuotes(strings.TrimSpace(args))
	if path == "" {
		slog.Error("exec_script: debug requires a file path")
		return
	}
	// Trigger a redraw so state is current
	a.Screen.PostEvent(tcell.NewEventInterrupt(nil))
	time.Sleep(50 * time.Millisecond)

	if err := a.DumpDebugState(path); err != nil {
		slog.Error("exec_script: debug dump failed", "path", path, "error", err)
	}
}

func execWait(args string) {
	ms := strings.TrimSpace(args)
	if ms == "" {
		slog.Error("exec_script: wait requires milliseconds")
		return
	}
	n, err := strconv.Atoi(ms)
	if err != nil {
		slog.Error("exec_script: invalid wait duration", "value", ms, "error", err)
		return
	}
	time.Sleep(time.Duration(n) * time.Millisecond)
}

func execQuit(a *App) {
	*a.Running = false
	a.Screen.PostEvent(tcell.NewEventInterrupt(nil))
}

// stripQuotes removes surrounding double quotes from a string if present.
func stripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// ExecScriptUsage returns the usage text for the --exec flag.
func ExecScriptUsage() string {
	return fmt.Sprintf(`--exec "commands"  Execute semicolon-separated commands after startup

Supported commands:
  click X Y          Simulate mouse click at coordinates
  key COMBO          Simulate key press (e.g., key ctrl+p, key enter)
  type TEXT           Type a string of text
  exec "Command"     Run a command palette command by title
  screenshot PATH    Save screen text to file
  debug PATH         Save debug state JSON to file
  wait MS            Wait milliseconds
  quit               Exit the editor

Example:
  %s --exec "wait 200; screenshot /tmp/s1.txt; quit"`, os.Args[0])
}
