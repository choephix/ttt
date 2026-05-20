# TTT — Terminal Text Tool

A fully-featured code editor that lives in the terminal. Not a simplified terminal editor — a real alternative to VS Code, Zed, and Sublime that happens to run in your terminal. Single Go binary, zero config.

## Features

- Tabbed editor with multi-buffer support
- File explorer sidebar with clickable panel tabs (Files, Search, Changes)
- Git changes panel with side-by-side diff viewer
- Command palette (Ctrl+P)
- Find bar with incremental search (Ctrl+F)
- Selection, copy/cut/paste (Ctrl+C/X/V)
- Undo/redo (Ctrl+Z/Y)
- Syntax highlighting (regex-based)
- Configurable themes, keybindings, and settings via JSON
- Chord keybindings (e.g. Ctrl+K Ctrl+C)
- Mouse support: click to position, click tabs, drag dividers
- Diff-based renderer for efficient terminal updates

## Getting Started

### Prerequisites

- Go 1.18 or newer

### Build and Run

```sh
make build
./bin/ttt
```

Or:

```sh
make run
```

Open a file:

```sh
./bin/ttt path/to/file.go
```

### Keybindings

| Key | Action |
|-----|--------|
| Ctrl+S | Save |
| Ctrl+Q | Quit |
| Ctrl+Z / Ctrl+Y | Undo / Redo |
| Ctrl+F | Find |
| Ctrl+G | Go to line |
| Ctrl+P | Command palette |
| Ctrl+B | Toggle sidebar |
| Ctrl+E | Show file explorer |
| Ctrl+D | Show git changes |
| Ctrl+A | Select all |
| Ctrl+C / Ctrl+X / Ctrl+V | Copy / Cut / Paste |
| Ctrl+PgDn / Ctrl+PgUp | Next / Previous tab |
| Ctrl+W | Close tab |

### Configuration

Config files are loaded from `.config/` (cwd), `<exe-dir>/config/`, or `~/.config/ttt/`:

- `keybindings.json` — custom keybindings
- `settings.json` — editor settings (tabSize, sidebarWidth, etc.)
- `theme.json` — colors and styles

The default theme sets explicit foreground/background colors (`#d4d4d4` on `#1e1e1e`). To use your terminal's native colors instead, set them to empty in `theme.json`:

```json
{
  "defaultFg": "",
  "defaultBg": ""
}
```

## License

MIT
