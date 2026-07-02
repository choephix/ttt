//go:build chaos

package chaos

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/ttt/internal/app"
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/core/clipboard"
	"github.com/eugenioenko/ttt/internal/plugin"
	"github.com/eugenioenko/ttt/internal/render"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/workspace"

	"github.com/gdamore/tcell/v2"
)

type EventRecord struct {
	Type string `json:"type"`
	Desc string `json:"desc"`
}

type CrashReport struct {
	Seed       int64         `json:"seed"`
	Iteration  int           `json:"iteration"`
	EventCount int           `json:"event_count"`
	Events     []EventRecord `json:"events"`
	Panic      string        `json:"panic"`
	Stack      string        `json:"stack"`
}

type chaosHarness struct {
	app      *app.App
	screen   tcell.SimulationScreen
	reg      *command.Registry
	renderer *render.Renderer
	dir      string
	events   []EventRecord
	rng      *rand.Rand
}

func newChaosHarness(seed int64) *chaosHarness {
	// Prevent OSC 52 escape sequences from leaking into test output
	clipboard.DisableSystem()
	dir, _ := os.MkdirTemp("", "chaos-*")

	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Test\n\nSome content here.\n\n- item 1\n- item 2\n"), 0644)
	os.WriteFile(filepath.Join(dir, "data.txt"), []byte(strings.Repeat("The quick brown fox jumps over the lazy dog.\n", 20)), 0644)
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	os.WriteFile(filepath.Join(dir, "src", "lib.go"), []byte("package src\n\nfunc Add(a, b int) int { return a + b }\n"), 0644)

	w, h := 80, 24
	sim := tcell.NewSimulationScreen("")
	sim.Init()
	sim.SetSize(w, h)

	cfg := config.AppConfig{
		Keybindings: config.DefaultKeybindings(),
		Settings:    config.DefaultSettings(),
		Theme:       config.DefaultTheme(),
	}
	config.ParseKeyBindings(cfg.Keybindings)

	screen := term.NewTcellScreenFrom(sim)
	screen.SetStyleMap(app.BuildStyleMap(cfg.Theme))

	borders := app.BuildBorderSet(cfg.Theme.Borders)

	ws := workspace.New([]string{dir})
	editor := app.BuildAppFromConfig(&cfg, &borders, ws, nil)
	editor.Screen = screen
	editor.Renderer = &render.Renderer{}

	pluginsDir := filepath.Join(dir, "plugins")
	registryPath := filepath.Join(dir, "registry.json")
	editor.PluginManager = plugin.NewManager(pluginsDir, registryPath)

	reg := command.NewRegistry()
	editor.Reg = reg
	running := true
	editor.Running = &running
	app.RegisterCommands(editor)
	app.BindKeys(editor.Root, reg, cfg.Keybindings)
	editor.Root.SetSize(w, h)

	cells := make([][]term.Cell, h)
	for y := range cells {
		cells[y] = make([]term.Cell, w)
	}
	editor.Root.Render(cells)
	editor.Renderer.SetCurrent(cells)
	editor.Renderer.Render(screen)

	return &chaosHarness{
		app:      editor,
		screen:   sim,
		reg:      reg,
		renderer: editor.Renderer,
		dir:      dir,
		events:   nil,
		rng:      rand.New(rand.NewSource(seed)),
	}
}

func (h *chaosHarness) cleanup() {
	// Close terminals before removing the temp dir to avoid PTY fd leaks across iterations
	h.app.CloseAllTerminals()
	h.screen.Fini()
	os.RemoveAll(h.dir)
}

func (h *chaosHarness) redraw() {
	cells := make([][]term.Cell, h.app.Root.Height)
	for y := range cells {
		cells[y] = make([]term.Cell, h.app.Root.Width)
	}
	h.app.Root.Render(cells)
	h.renderer.SetCurrent(cells)
	h.renderer.Render(h.app.Screen)
}

func (h *chaosHarness) flushOnChange() {
	if h.app.EditorGroup.Editor != nil {
		h.app.EditorGroup.Editor.FlushOnChange()
	}
}

func (h *chaosHarness) dispatch(ev tcell.Event) {
	h.app.Root.HandleEvent(ev)
	h.flushOnChange()
	h.redraw()
}

func (h *chaosHarness) record(typ, desc string) {
	h.events = append(h.events, EventRecord{Type: typ, Desc: desc})
}

var printableRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 \t!@#$%^&*()_+-=[]{}|;':\",./<>?`~")

var specialKeys = []tcell.Key{
	tcell.KeyEscape, tcell.KeyEnter, tcell.KeyTab, tcell.KeyBacktab,
	tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete,
	tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight,
	tcell.KeyHome, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyPgDn,
	tcell.KeyF1, tcell.KeyF2, tcell.KeyF3, tcell.KeyF5,
}

var ctrlKeys = []tcell.Key{
	tcell.KeyCtrlA, tcell.KeyCtrlB, tcell.KeyCtrlC, tcell.KeyCtrlD,
	tcell.KeyCtrlE, tcell.KeyCtrlF, tcell.KeyCtrlG, tcell.KeyCtrlH,
	tcell.KeyCtrlK, tcell.KeyCtrlL, tcell.KeyCtrlN, tcell.KeyCtrlO,
	tcell.KeyCtrlP, tcell.KeyCtrlQ, tcell.KeyCtrlR, tcell.KeyCtrlS,
	tcell.KeyCtrlT, tcell.KeyCtrlU, tcell.KeyCtrlV, tcell.KeyCtrlW,
	tcell.KeyCtrlX, tcell.KeyCtrlY, tcell.KeyCtrlZ,
	tcell.KeyCtrlSpace,
}

var chordFollowRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func (h *chaosHarness) randomEvent() {
	w, hh := h.app.Root.Width, h.app.Root.Height

	n := h.rng.Intn(100)
	switch {
	case n >= 0 && n < 30:
		// 30%: printable rune
		r := printableRunes[h.rng.Intn(len(printableRunes))]
		h.record("rune", string(r))
		h.dispatch(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))

	case n >= 30 && n < 45:
		// 15%: special key
		k := specialKeys[h.rng.Intn(len(specialKeys))]
		h.record("special", fmt.Sprintf("key=%d", k))
		h.dispatch(tcell.NewEventKey(k, 0, tcell.ModNone))

	case n >= 45 && n < 55:
		// 10%: ctrl+key
		k := ctrlKeys[h.rng.Intn(len(ctrlKeys))]
		h.record("ctrl", fmt.Sprintf("key=%d", k))
		h.dispatch(tcell.NewEventKey(k, 0, tcell.ModCtrl))

	case n >= 55 && n < 65:
		// 10%: chord (ctrl+k followed by a rune)
		h.record("chord", "ctrl+k")
		h.dispatch(tcell.NewEventKey(tcell.KeyCtrlK, 0, tcell.ModCtrl))
		r := chordFollowRunes[h.rng.Intn(len(chordFollowRunes))]
		h.record("chord-follow", string(r))
		h.dispatch(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))

	case n >= 65 && n < 75:
		// 10%: mouse click
		mx := h.rng.Intn(w)
		my := h.rng.Intn(hh)
		h.record("click", fmt.Sprintf("x=%d,y=%d", mx, my))
		h.dispatch(tcell.NewEventMouse(mx, my, tcell.Button1, tcell.ModNone))
		h.dispatch(tcell.NewEventMouse(mx, my, tcell.ButtonNone, tcell.ModNone))

	case n >= 75 && n < 80:
		// 5%: mouse drag
		x1 := h.rng.Intn(w)
		y1 := h.rng.Intn(hh)
		x2 := h.rng.Intn(w)
		y2 := h.rng.Intn(hh)
		steps := 3 + h.rng.Intn(4)
		h.record("drag", fmt.Sprintf("(%d,%d)->(%d,%d) steps=%d", x1, y1, x2, y2, steps))
		h.dispatch(tcell.NewEventMouse(x1, y1, tcell.Button1, tcell.ModNone))
		for s := 1; s <= steps; s++ {
			ix := x1 + (x2-x1)*s/steps
			iy := y1 + (y2-y1)*s/steps
			h.dispatch(tcell.NewEventMouse(ix, iy, tcell.Button1, tcell.ModNone))
		}
		h.dispatch(tcell.NewEventMouse(x2, y2, tcell.ButtonNone, tcell.ModNone))

	case n >= 80 && n < 85:
		// 5%: mouse scroll
		mx := h.rng.Intn(w)
		my := h.rng.Intn(hh)
		btn := tcell.WheelUp
		dir := "up"
		if h.rng.Intn(2) == 0 {
			btn = tcell.WheelDown
			dir = "down"
		}
		h.record("scroll", fmt.Sprintf("x=%d,y=%d,dir=%s", mx, my, dir))
		h.dispatch(tcell.NewEventMouse(mx, my, btn, tcell.ModNone))

	case n >= 85 && n < 90:
		// 5%: resize
		nw := 40 + h.rng.Intn(120)
		nh := 10 + h.rng.Intn(50)
		h.record("resize", fmt.Sprintf("w=%d,h=%d", nw, nh))
		h.screen.SetSize(nw, nh)
		h.app.Root.SetSize(nw, nh)
		h.redraw()

	case n >= 90 && n < 95:
		// 5%: execute random command
		cmds := h.reg.List()
		if len(cmds) > 0 {
			cmd := cmds[h.rng.Intn(len(cmds))]
			h.record("command", cmd.ID)
			h.reg.Execute(cmd.ID)
			h.flushOnChange()
			h.redraw()
		}

	default:
		// 5%: shift+special key
		k := specialKeys[h.rng.Intn(len(specialKeys))]
		h.record("shift-special", fmt.Sprintf("key=%d", k))
		h.dispatch(tcell.NewEventKey(k, 0, tcell.ModShift))
	}
}

func writeCrashReport(report CrashReport) string {
	outputDir := os.Getenv("CHAOS_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "."
	}
	os.MkdirAll(outputDir, 0755)
	filename := filepath.Join(outputDir, fmt.Sprintf("crash-%d-%d.json", report.Seed, report.Iteration))
	data, _ := json.MarshalIndent(report, "", "  ")
	os.WriteFile(filename, data, 0644)
	return filename
}

func runIteration(seed int64, eventsPerRun int) *CrashReport {
	h := newChaosHarness(seed)
	defer h.cleanup()

	var report *CrashReport
	func() {
		defer func() {
			if r := recover(); r != nil {
				report = &CrashReport{
					Seed:       seed,
					Iteration:  0,
					EventCount: len(h.events),
					Events:     h.events,
					Panic:      fmt.Sprintf("%v", r),
					Stack:      string(debug.Stack()),
				}
			}
		}()

		for i := 0; i < eventsPerRun; i++ {
			h.randomEvent()
		}
	}()

	return report
}

func TestChaosMonkey(t *testing.T) {
	iterations := 50
	eventsPerRun := 500

	if v := os.Getenv("CHAOS_ITERATIONS"); v != "" {
		fmt.Sscanf(v, "%d", &iterations)
	}
	if v := os.Getenv("CHAOS_EVENTS"); v != "" {
		fmt.Sscanf(v, "%d", &eventsPerRun)
	}

	baseSeed := time.Now().UnixNano()
	if v := os.Getenv("CHAOS_SEED"); v != "" {
		fmt.Sscanf(v, "%d", &baseSeed)
	}

	var crashes []CrashReport

	for i := 0; i < iterations; i++ {
		seed := baseSeed + int64(i)
		report := runIteration(seed, eventsPerRun)
		if report != nil {
			report.Iteration = i
			file := writeCrashReport(*report)
			t.Errorf("CRASH at iteration %d (seed=%d): %s\n  saved to %s", i, seed, report.Panic, file)
			crashes = append(crashes, *report)
		}
	}

	if len(crashes) == 0 {
		t.Logf("OK: %d iterations x %d events = %d total events, no panics (base seed: %d)",
			iterations, eventsPerRun, iterations*eventsPerRun, baseSeed)
	} else {
		t.Errorf("FAILED: %d/%d iterations crashed", len(crashes), iterations)
	}
}

func TestChaosReplay(t *testing.T) {
	replayFile := os.Getenv("CHAOS_REPLAY")
	if replayFile == "" {
		t.Skip("set CHAOS_REPLAY=<crash-file.json> to replay")
	}

	data, err := os.ReadFile(replayFile)
	if err != nil {
		t.Fatal(err)
	}

	var report CrashReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}

	h := newChaosHarness(report.Seed)
	defer h.cleanup()

	defer func() {
		if r := recover(); r != nil {
			t.Logf("REPRODUCED panic: %v", r)
			t.Logf("Stack:\n%s", debug.Stack())
			t.FailNow()
		}
	}()

	for i := 0; i < report.EventCount; i++ {
		h.randomEvent()
	}

	t.Log("Replay completed without panic — may be non-deterministic or already fixed")
}

// TestChaosLoop runs continuously until stopped — designed for Docker.
func TestChaosLoop(t *testing.T) {
	if os.Getenv("CHAOS_LOOP") == "" {
		t.Skip("set CHAOS_LOOP=1 to run continuous chaos loop")
	}

	eventsPerRun := 500
	if v := os.Getenv("CHAOS_EVENTS"); v != "" {
		fmt.Sscanf(v, "%d", &eventsPerRun)
	}

	outputDir := os.Getenv("CHAOS_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "/output"
	}
	os.Setenv("CHAOS_OUTPUT_DIR", outputDir)

	iteration := 0
	totalCrashes := 0

	for {
		seed := time.Now().UnixNano()
		report := runIteration(seed, eventsPerRun)
		if report != nil {
			report.Iteration = iteration
			file := writeCrashReport(*report)
			totalCrashes++
			fmt.Fprintf(os.Stderr, "CRASH #%d at iteration %d (seed=%d): %s\n  → %s\n",
				totalCrashes, iteration, seed, report.Panic, file)
		}

		iteration++
		if iteration%100 == 0 {
			fmt.Fprintf(os.Stderr, "chaos: %d iterations, %d crashes\n", iteration, totalCrashes)
		}
	}
}
