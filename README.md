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
- Integrated terminal emulator with multiple tabs
- Diff view with syntax highlighting on top of diff background colors

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

### Usage

```sh
ttt                    # opens the current directory
ttt /path/to/dir       # opens that directory as the workspace
ttt /path/to/file.go   # opens the file; workspace is the git repo root
                       # (falls back to the file's parent dir if not in a repo)
```

When a directory is opened, the file explorer and git changes panel are rooted there. When a file is opened directly, ttt finds the enclosing git repository and uses that as the workspace root.

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
| Ctrl+` | Toggle integrated terminal |
| Ctrl+PgDn / Ctrl+PgUp | Next / Previous tab |
| Ctrl+W | Close tab |

### Integrated Terminal

TTT includes a built-in terminal emulator. Press Ctrl+` to toggle the terminal panel. Features:

- Multiple terminal tabs with a `+` button to spawn new ones
- Tab labels show the shell name (e.g. "zsh 1", "bash 2")
- Full VT escape sequence support via `hinshun/vt10x` and PTY management via `creack/pty`
- 256-color rendering with direct RGB color support
- When the terminal is focused, all keys go to the PTY except Ctrl+` (to toggle the panel)
- Terminal shell and scrollback are configurable in `settings.json`
- Terminal ANSI colors are theme-configurable via the `terminal` field in `theme.json`

### Configuration

Config files are loaded from `.config/` (cwd), `<exe-dir>/config/`, or `~/.config/ttt/`:

- `keybindings.json` — custom keybindings
- `settings.json` — editor settings (tabSize, sidebarWidth, terminal shell/scrollback, etc.)
- `theme.json` — colors and styles (including terminal ANSI colors)

The default theme sets explicit foreground/background colors (`#d4d4d4` on `#1e1e1e`). To use your terminal's native colors instead, set them to empty in `theme.json`:

```json
{
  "defaultFg": "",
  "defaultBg": ""
}
```

## License

MIT
