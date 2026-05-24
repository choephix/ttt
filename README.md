# TTT — Terminal Text Tool

A fully-featured code editor that lives in the terminal. Not a simplified terminal editor — a real alternative to VS Code, Zed, and Sublime that happens to run in your terminal. Single Go binary, zero config.

## Features

- **Multi-folder workspaces** — open multiple directories in a single session, save/load workspace files (`.ttt`), or pass `--workspace` on the CLI
- **Tabbed editor** with multi-buffer support and tab reuse (unpinned tabs are replaced when opening a new file; clicking an existing tab pins it, like VS Code)
- **File explorer sidebar** with multi-root support — when multiple folders are open, each root is shown as a collapsible group
- **Search panel** powered by ripgrep with include/exclude glob filters (e.g. `*.go`, `!vendor/**`)
- **Git changes panel** with per-folder grouping in multi-root workspaces
- **Side-by-side diff viewer** with syntax highlighting layered on top of diff background colors
- **Command palette** (Ctrl+P) — type `>` for commands, or delete the `>` to switch to quick file open across all workspace folders
- **Find and Replace** — inline find bar (Ctrl+F) with match navigation, replace bar (Ctrl+H) with replace-one and replace-all
- **Go to Line** (Ctrl+G)
- **Selection, copy/cut/paste** (Ctrl+C/X/V) with system clipboard support
- **Undo/redo** (Ctrl+Z/Y) via command-pattern undo stack
- **Regex-based syntax highlighting** with language auto-detection
- **Configurable themes** with live preview switching (Ctrl+K T)
- **Custom keybindings** with chord support (e.g. Ctrl+K Ctrl+C)
- **`.editorconfig` support** — indent size is picked up automatically per file
- **Indent detection** — auto-detects indentation from file content; manual override via the indent picker in the status bar
- **Mouse support** — click to position cursor, click tabs, drag sidebar/panel dividers, right-click context menus
- **Menu bar** — File, Edit, Selection, View, Help menus
- **Diff-based renderer** for efficient terminal updates (double-buffered cell grid)
- **Integrated terminal emulator** with multiple tabs
- **Git blame** — inline blame info for the current line shown in the status bar
- **Dynamic status bar branch** — the git branch display switches automatically based on which workspace folder the active file belongs to

## Getting Started

### Prerequisites

- Go 1.18 or newer
- [ripgrep](https://github.com/BurntSushi/ripgrep) (`rg`) — required for workspace search

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
ttt                             # opens the current directory
ttt /path/to/dir                # opens that directory as the workspace
ttt /path/to/file.go            # opens the file; workspace is the git repo root
                                # (falls back to the file's parent dir if not in a repo)
ttt dir1 dir2                   # opens multiple folders as a multi-root workspace
ttt --workspace project.ttt     # loads a saved workspace file
```

When a directory is opened, the file explorer, search, and git changes panels are rooted there. When multiple directories are opened (or loaded from a `.ttt` workspace file), each one becomes a separate root in the explorer and changes panel.

#### Workspace Files

Workspace files use the `.ttt` extension and store a list of folders as relative paths:

```json
{
  "folders": [
    { "path": "." },
    { "path": "../other-project" }
  ]
}
```

Use **File > Save Workspace As...** to create one, or load it with `--workspace`.

You can also manage folders at runtime via the command palette: **Add Folder to Workspace** and **Remove Folder from Workspace**.

### Keybindings

| Key | Action |
|-----|--------|
| Ctrl+S | Save |
| Ctrl+Q | Quit |
| Ctrl+Z / Ctrl+Y | Undo / Redo |
| Ctrl+F | Find |
| Ctrl+H | Find and Replace |
| Ctrl+G | Go to line |
| Ctrl+P | Command palette / quick file open |
| Ctrl+B | Toggle sidebar |
| Ctrl+E | Show file explorer |
| Ctrl+Shift+F | Show search |
| Ctrl+D | Show git changes |
| Ctrl+A | Select all |
| Ctrl+C / Ctrl+X / Ctrl+V | Copy / Cut / Paste |
| Ctrl+` | Toggle integrated terminal |
| Ctrl+PgDn / Ctrl+PgUp | Next / Previous tab |
| Ctrl+W | Close tab |
| Ctrl+K T | Switch theme |

### Integrated Terminal

TTT includes a built-in terminal emulator. Press Ctrl+` to toggle the terminal panel. Features:

- Multiple terminal tabs with a `+` button to spawn new ones
- Tab labels show the shell name (e.g. `[>_1]`, `[>_2]`)
- Full VT escape sequence support via `hinshun/vt10x` and PTY management via `creack/pty`
- 256-color rendering with direct RGB color support
- When the terminal is focused, all keys go to the PTY except Ctrl+` (to toggle the panel)
- Terminal shell and scrollback are configurable in `settings.json`
- Terminal ANSI colors are theme-configurable via the `terminal` field in `theme.json`

### Tab Behavior

Tabs follow a pin-on-reclick model similar to VS Code:

- Opening a file from the explorer or search replaces the current **unpinned** tab.
- Clicking on an already-open tab (or opening the same file again) **pins** it, so it won't be replaced.
- You can close tabs with Ctrl+W, or right-click a tab for Close / Close Others / Close All.

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
