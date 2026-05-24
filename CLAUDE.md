# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ttt is a terminal text editor written in Go, using tcell for terminal rendering. The Go module name is `ttt` (in go.mod).

## Build & Test Commands

```sh
make build        # builds to bin/ttt
make run          # build + run
make test         # go test ./...
make fmt          # gofmt -w .
make lint         # golint ./...
go test ./internal/core/buffer/   # run tests for a single package
```

## Architecture

The codebase follows a strict layered architecture: **core → view → render → term → ui**. The core layer has zero terminal dependencies and is fully unit-testable in isolation.

### Layers

- **`internal/core/`** — UI-agnostic editor engine. Must never import terminal or rendering packages.
  - `buffer/` — Line-based text storage (`[]string`), rune-level insert/delete, file I/O (load/save)
  - `cursor/` — Visual column cursor with goal-column preservation for vertical movement
  - `undo/` — Command-pattern undo/redo via `EditCommand` interface (InsertRune, DeleteRange, InsertLine)
  - `highlight/` — Regex-based per-line syntax highlighting (`Highlighter` interface with `Span` output)
  - `buffers/` — Multiple buffer management (list + active index)

- **`internal/view/`** — Viewport (scrolling, cursor-to-screen mapping) and status bar rendering

- **`internal/render/`** — Diff-based renderer: compares prev/curr cell grids and emits minimal updates

- **`internal/terminal/`** — Integrated terminal emulator. Wraps `hinshun/vt10x` for VT escape sequence parsing and `creack/pty` for PTY lifecycle management. Provides the backing state for terminal tabs.

- **`internal/term/`** — Terminal abstraction via `Screen` interface. `TcellScreen` is the real implementation; `MockScreen` is used in tests. Only this package imports `tcell`. Also defines `DirectColor` and `CellAttr` types for direct RGB color rendering (used by the terminal emulator to bypass the style map for 256-color support).

- **`internal/ui/`** — Window manager and pane system. `Window` binds a `Rect`, `Viewport`, and `Buffer` together. `WindowManager` tracks focus across windows. Also contains `terminal_widget.go` (renders vt10x grid as direct-color cells, handles key-to-VT translation), `root.go` (ForceKeys and RawKeyConsumer interface for terminal key routing), and `content_split.go` (OnTopClick/OnBottomClick for focus routing between editor and bottom panel).

- **`cmd/ttt/main.go`** — Entry point with event loop. Wires all components together, handles key dispatch, viewport scrolling, and redraw.

### Key Design Constraints

- Cursor `Col` is a visual column (rune-based), not a byte index — all line-length calculations use `[]rune()`.
- The renderer uses double-buffering (prev/curr cell grids) to minimize terminal writes.
- `Screen` interface keeps tcell isolated — the rest of the codebase never imports tcell directly (except `cmd/ttt/main.go` for event types).
- **Never hardcode colors.** All colors must go through the theme system (`internal/config/theme.go` → `StyleDef` → `term.Style` constants → `buildStyleMap`). Add a new `StyleDef` field to `ThemeConfig`, a `term.Style` constant, and wire it in `buildStyleMap()`. Widgets reference `term.Style*` constants, never color values. The one exception is the integrated terminal, which uses direct RGB color rendering via `DirectColor`/`CellAttr` to support 256-color output.
- **Terminal colors** are configured via the `terminal` field in `ThemeConfig` (`TerminalColors`), which holds 16 ANSI colors plus foreground/background defaults.
- The diff view layers syntax highlighting on top of diff background colors using `BgStyle` layering.
- **RawKeyConsumer interface**: when the integrated terminal is focused, all key events are routed directly to the PTY. Only force-keys (Ctrl+`) bypass this to allow toggling the terminal panel.
- Async PTY output wakes the event loop via `PostEvent`/`EventInterrupt`.

### Dependencies

Key external dependencies beyond the Go standard library:

- `github.com/gdamore/tcell/v2` — terminal rendering
- `github.com/creack/pty` — PTY management for the integrated terminal
- `github.com/hinshun/vt10x` — VT escape sequence parsing for the integrated terminal
