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

- **`internal/term/`** — Terminal abstraction via `Screen` interface. `TcellScreen` is the real implementation; `MockScreen` is used in tests. Only this package imports `tcell`.

- **`internal/ui/`** — Window manager and pane system. `Window` binds a `Rect`, `Viewport`, and `Buffer` together. `WindowManager` tracks focus across windows.

- **`cmd/ttt/main.go`** — Entry point with event loop. Wires all components together, handles key dispatch, viewport scrolling, and redraw.

### Key Design Constraints

- Cursor `Col` is a visual column (rune-based), not a byte index — all line-length calculations use `[]rune()`.
- The renderer uses double-buffering (prev/curr cell grids) to minimize terminal writes.
- `Screen` interface keeps tcell isolated — the rest of the codebase never imports tcell directly (except `cmd/ttt/main.go` for event types).
