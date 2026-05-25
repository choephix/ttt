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

# Open a multi-folder workspace
bin/ttt --workspace project.ttt

# Open specific folders or files
bin/ttt ~/projectA ~/projectB file.go
```

## Architecture

The codebase follows a strict layered architecture: **core → view → render → term → ui**, with `workspace` sitting alongside as an independent support layer. The core layer has zero terminal dependencies and is fully unit-testable in isolation.

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

- **`internal/workspace/`** — Multi-folder workspace management. `Folder` and `Workspace` types track one or more project roots, with `IsRepo` git-detection, `FolderForFile` lookup (longest-prefix match), and JSON-based workspace file loading/saving (`.ttt` files). The editor falls back to `cwd` when no folders are explicitly provided.

- **`cmd/ttt/main.go`** — Entry point with event loop. Wires all components together, handles key dispatch, viewport scrolling, and redraw. Accepts a `--workspace <file>` flag to open a saved workspace, or folder/file paths as positional arguments.

### Key Design Constraints

- Cursor `Col` is a visual column (rune-based), not a byte index — all line-length calculations use `[]rune()`.
- The renderer uses double-buffering (prev/curr cell grids) to minimize terminal writes.
- `Screen` interface keeps tcell isolated — the rest of the codebase never imports tcell directly (except `cmd/ttt/main.go` for event types).
- **Never hardcode colors.** All colors must go through the theme system (`internal/config/theme.go` → `StyleDef` → `term.Style` constants → `buildStyleMap`). Add a new `StyleDef` field to `ThemeConfig`, a `term.Style` constant, and wire it in `buildStyleMap()`. Widgets reference `term.Style*` constants, never color values. The one exception is the integrated terminal, which uses direct RGB color rendering via `DirectColor`/`CellAttr` to support 256-color output.
- **Terminal colors** are configured via the `terminal` field in `ThemeConfig` (`TerminalColors`), which holds 16 ANSI colors plus foreground/background defaults.
- The diff view layers syntax highlighting on top of diff background colors using `BgStyle` layering.
- **RawKeyConsumer interface**: when the integrated terminal is focused, all key events are routed directly to the PTY. Only force-keys (Ctrl+`) bypass this to allow toggling the terminal panel.
- Async PTY output wakes the event loop via `PostEvent`/`EventInterrupt`.

### Keybinding System & tcell Key Mapping

Keybindings are defined in `internal/config/keybindings.go` (`DefaultKeybindings()`) and converted to tcell key constants via `comboToTcell()` in `cmd/ttt/keys.go`. The matching happens in `matchKey()` in `internal/ui/root.go`.

**Critical: tcell control key behavior.** For control keys (`r < ' '`), tcell posts events with **both** the `KeyCtrl*` constant **and** `ModCtrl` set (see `vendor/.../tcell/v2/input.go:452`). When registering control key bindings in `comboToTcell`, do NOT strip `ModCtrl` — the registered modifier must match what tcell delivers, otherwise `matchKey()` will fail silently.

**Ctrl+Backtick (`` ctrl+` ``):** Maps to `KeyCtrlSpace` (value 64) in tcell because Ctrl+` sends NUL (0x00), same as Ctrl+Space. This is a terminal-level constraint, not a bug. Both `ctrl+backtick` and `ctrl+space` produce the same tcell event — they cannot be bound to different commands. Currently `ctrl+backtick` is bound to `terminal.toggle`.

**Force keys:** Bindings in the `forceKeyCommands` map (`cmd/ttt/commands.go`) are registered via `root.AddForceKey()` and are checked even when a `RawKeyConsumer` (like the integrated terminal) has focus. `terminal.toggle` must remain a force key.

### LSP Integration

Language server support lives in `internal/lsp/`. Servers are configured per-language in `~/.config/ttt/extensions.json`. The LSP client uses JSON-RPC 2.0 over stdio with Content-Length framing — no external dependencies.

- `jsonrpc.go` — codec (send/receive with Content-Length framing)
- `protocol.go` — minimal LSP type definitions (initialize, document sync, completions)
- `client.go` — LSP client with async read loop and request/response channel matching
- `manager.go` — one client per language, lazy-started on first use
- `extensions.go` — config loading from `extensions.json`
- `cmd/ttt/lsp.go` — bridge converting `lsp.CompletionItem` → `ui.CompletionItem`

Async completions use the same `PostEvent(EventInterrupt)` pattern as git blame. Document sync is full-document (not incremental).

### Dependencies

Key external dependencies beyond the Go standard library:

- `github.com/gdamore/tcell/v2` — terminal rendering
- `github.com/creack/pty` — PTY management for the integrated terminal
- `github.com/hinshun/vt10x` — VT escape sequence parsing for the integrated terminal
