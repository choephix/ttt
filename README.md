# Pico

Pico is a modular, test-driven terminal text editor written in Go. It is designed for extensibility, clean architecture, and a modern TUI experience inspired by classic editors.

## Features

- Line-based buffer with Unicode support
- Visual cursor navigation (arrow keys, Home/End, PageUp/PageDown)
- Insert, delete, and split lines (Enter, Backspace)
- Status bar with filename and cursor position
- Syntax highlighting (regex-based, easily extensible)
- Undo/redo system (command-based)
- Multiple buffers and window management (split panes, modal dialogs)
- Diff-based renderer for efficient terminal updates
- Fully unit-tested core modules

## Getting Started

### Prerequisites

- Go 1.18 or newer

### Build and Run

```sh
make build
./bin/editor
```

Or use:

```sh
make run
```

### Controls

- **Arrow keys**: Move cursor
- **Enter**: Split line
- **Backspace**: Delete character/merge lines
- **Ctrl+C**: Quit

## Project Structure

```
cmd/editor/           # Main entry point
internal/core/buffer/ # Buffer logic
internal/core/cursor/ # Cursor logic
internal/core/undo/   # Undo/redo system
internal/core/highlight/ # Syntax highlighting
internal/core/buffers/   # Multiple buffer management
internal/view/        # Viewport, status bar
internal/term/        # Terminal abstraction (tcell)
internal/render/      # Diff-based renderer
internal/ui/          # Window manager, panes, dialogs
```

## Testing

Run all unit tests:

```sh
make test
```

## Extending Pico

- Add new syntax highlighters in `internal/core/highlight/`
- Implement new UI panes or dialogs in `internal/ui/`
- Add new commands or keybindings in `cmd/editor/main.go`

## Roadmap

- File open/save dialogs
- Configurable keybindings
- Mouse support
- Advanced syntax highlighting
- Plugin system

## License

MIT
