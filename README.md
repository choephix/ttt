# TTT — Terminal Text Tool

A fully-featured code editor that lives in the terminal. Not a simplified terminal editor — a real alternative to VS Code, Zed, and Sublime that happens to run in your terminal. Single Go binary, zero config.

## Features

### Editor

- **Syntax highlighting** via [chroma](https://github.com/alecthomas/chroma) — supports hundreds of languages with automatic detection
- **Bracket matching** with highlighted pairs
- **Find and Replace** — inline find bar (Ctrl+F) with match navigation, replace bar (Ctrl+H) with replace-one and replace-all
- **Go to Line** (Ctrl+G)
- **Selection, copy/cut/paste** (Ctrl+C/X/V) with system clipboard support
- **Undo/redo** (Ctrl+Z/Y) via a command-pattern undo stack
- **`.editorconfig` support** — indent size is picked up automatically per file
- **Indent detection** — auto-detects indentation from file content; manual override via the status bar indent picker
- **Mouse support** — click to position cursor, click tabs, drag sidebar/panel dividers, right-click context menus
- **Git blame** — inline blame info for the current line shown in the status bar (author, relative time, summary)
- **Line numbers** with current-line highlighting
- **Integrated terminal emulator** via [vt10x](https://github.com/hinshun/vt10x) — multiple tabs, full VT escape sequence support
- **Diff-based renderer** for efficient terminal updates (double-buffered cell grid)

### Multi-Folder Workspaces

Open multiple project directories in a single session. Each root appears as a collapsible group in the explorer, search, and changes panels.

```sh
ttt                             # opens the current directory
ttt /path/to/dir                # opens that directory as the workspace
ttt /path/to/file.go            # opens the file; workspace is the git repo root
                                # (falls back to the file's parent dir if not in a repo)
ttt dir1 dir2                   # opens multiple folders as a multi-root workspace
ttt --workspace project.ttt     # loads a saved workspace file
```

Workspace files use the `.ttt` extension and store a list of folders as relative paths:

```json
{
  "folders": [
    { "path": "." },
    { "path": "../other-project" }
  ]
}
```

- **Save Workspace As...** from the File menu to create a workspace file
- **Add Folder to Workspace** and **Remove Folder from Workspace** via the command palette
- The git branch in the status bar switches automatically based on which workspace folder the active file belongs to

### File Explorer

Multi-root file tree in the sidebar (Ctrl+Shift+E). When multiple folders are open, each root is shown as a collapsible group.

- Directories sorted before files, both alphabetically
- Expand/collapse with Enter or arrow keys
- Right-click context menu: **New File**, **New Folder**, **Rename**, **Delete**
- Sidebar actions button for **Refresh** and **New File**

### Search

Sidebar search panel (Ctrl+Shift+F) powered by [ripgrep](https://github.com/BurntSushi/ripgrep). Results are grouped by file with match counts.

- **Smart-case** matching by default
- **Include/Exclude glob filters** — click the toggle arrow to reveal filter inputs (e.g. `*.go`, `vendor/**`)
- Tab between search, include, and exclude inputs
- Searches across all workspace folders simultaneously
- Click a result to jump to the file and line

### Git Integration

Changes panel in the sidebar (Ctrl+Shift+G) with full staging workflow.

**Staging:**
- **Spacebar** — toggle stage/unstage on the selected file
- **`a`** — stage all unstaged files
- **`u`** — unstage all staged files
- **`+` button** on the "Changes" section header — stage all files in that section
- **`-` button** on the "Staged" section header — unstage all files in that section

**Committing:**
- Inline commit message input at the top of each group
- Type a message and press Enter to commit all staged files

**Remote operations:**
- **Pull**, **Push**, **Sync** (pull then push) from the sidebar actions button
- Per-repo actions via the group header menu button in multi-root workspaces

**Diff view:**
- Select a changed file to open a diff with syntax highlighting layered on diff backgrounds
- Untracked files open directly in the editor

**Multi-root:**
- Changes are grouped by repository, each with its own collapsible Staged/Changes sections
- Each group has a commit input and a menu button for pull/push/sync on that specific repo
- File status badges: **M** (modified), **A** (added), **D** (deleted), **R** (renamed), **U** (untracked)

### Command Palette

- **Ctrl+Shift+P** — opens the command palette with all available commands
- **Ctrl+P** — opens quick file open (searches all files across workspace folders)
- Type `>` in quick-open mode to switch to command mode
- Delete the `>` in command mode to switch to file mode
- Menu shortcuts resolve dynamically from your keybindings

### Integrated Terminal

Built-in terminal emulator. Press Ctrl+` to toggle the terminal panel.

- **Ctrl+Shift+`** to spawn a new terminal tab
- Multiple terminal tabs with a `+` button and tab labels (`[>_1]`, `[>_2]`, ...)
- Full VT escape sequence support via `hinshun/vt10x` and PTY management via `creack/pty`
- 256-color rendering with direct RGB color support
- When the terminal is focused, all keys go to the PTY except Ctrl+` (to toggle the panel)
- Terminal shell and scrollback are configurable in `settings.json`
- Terminal ANSI colors are theme-configurable via the `terminal` field in `theme.json`
- Close all terminals from the panel actions menu

### LSP (Language Server Protocol)

TTT has built-in LSP support for language-aware editing features. Language servers run as external processes and communicate over JSON-RPC 2.0 via stdio.

#### Configuring Language Servers

Add language servers to `~/.config/ttt/settings.json` under the `lsp.servers` key. Each entry maps a language identifier to a command array:

```json
{
  "lsp": {
    "servers": {
      "go": { "command": ["gopls"] },
      "typescript": { "command": ["typescript-language-server", "--stdio"] },
      "javascript": { "command": ["typescript-language-server", "--stdio"] },
      "python": { "command": ["pyright-langserver", "--stdio"] }
    }
  }
}
```

The language identifier must match the language ID that TTT assigns to the file (e.g. `go`, `typescript`, `javascript`, `python`, `rust`, `c`, `cpp`). The server is started lazily on first use and shut down when the editor exits.

#### Supported Features

| Feature | Keybinding | Description |
|---------|-----------|-------------|
| Autocomplete | Ctrl+U | Trigger completion at cursor position |
| Go to Definition | F12 | Jump to the definition of the symbol under the cursor |
| Go to Implementation | Shift+F12 | Jump to the implementation of the symbol under the cursor |
| Go to Type Definition | *(command palette)* | Jump to the type definition of the symbol under the cursor |
| Hover | Ctrl+K I | Show type information and documentation for the symbol under the cursor |

Go to Type Definition is available via the command palette (Ctrl+Shift+P, then search for "Go to Type Definition").

### Tabs

Tabs follow a pin-on-reclick model similar to VS Code:

- Opening a file from the explorer or search replaces the current **unpinned** tab
- Clicking on an already-open tab (or opening the same file again) **pins** it
- **Ctrl+W** to close a tab, **Ctrl+PgDn/PgUp** to switch tabs
- Right-click a tab for **Close**, **Close Others**, **Close All**
- Tab bar actions button for **Close All**

### Theming

TTT supports fully customizable themes via JSON files. You can change every color in the editor — from syntax highlighting and diff backgrounds to the sidebar, tabs, status bar, terminal ANSI colors, borders, and semantic colors (success, danger, warning).

#### Built-in Themes

10 themes ship in the `sample-config/` directory:

- Aurora
- Bubblegum
- Default Dark
- Default Light
- Hotline
- Monokai
- One Dark
- Solarized Dark
- Solarized Light
- Virtru Dark

#### Switching Themes

Press **Ctrl+K Ctrl+T** (or use **View > Switch Theme** from the menu bar) to open the theme picker with a live preview.

#### Customizing

To create a custom theme, copy one of the built-in theme files to your config directory and edit it:

```sh
cp sample-config/theme.monokai.json ~/.config/ttt/theme.json
```

Restart TTT (or switch themes) to pick up changes.

To use your terminal's native colors instead of the theme's, set foreground/background to empty strings in your theme file.

### Menu Bar

File, Edit, Selection, View, and Help menus accessible via the menu bar or keyboard shortcuts. Menus display resolved keybindings next to each command. Navigate between menus with left/right arrow keys.

## Installation

### Prerequisites

- [Go](https://go.dev/) 1.18 or newer
- [Git](https://git-scm.com/) — required for source control features
- [ripgrep](https://github.com/BurntSushi/ripgrep) (`rg`) — required for workspace search

### Go Install

```sh
go install github.com/eugenioenko/ttt/cmd/ttt@latest
```

This installs the `ttt` binary to your `$GOPATH/bin` (or `$HOME/go/bin` by default). Make sure that directory is in your `PATH`.

### From Source

```sh
git clone https://github.com/eugenioenko/ttt.git
cd ttt
make build
```

This produces an optimized binary at `bin/ttt`. Add it to your `PATH` or copy it somewhere convenient:

```sh
cp bin/ttt ~/.local/bin/
```

### Configuration

Config files are loaded from `.config/` (cwd), `<exe-dir>/config/`, or `~/.config/ttt/`:

| File | Purpose |
|------|---------|
| `keybindings.json` | Custom keybindings (VS Code key format) |
| `settings.json` | Editor settings (tabSize, insertSpaces, wordWrap, lineNumbers, sidebarWidth, terminal, theme, lsp) |
| `theme.json` | Colors and styles (or use `theme.<name>.json` for named themes) |

#### Settings

```json
{
  "tabSize": 4,
  "insertSpaces": true,
  "wordWrap": false,
  "lineNumbers": true,
  "sidebarVisible": true,
  "sidebarWidth": 30,
  "theme": "default-dark",
  "terminal": {
    "shell": "",
    "scrollback": 1000
  },
  "lsp": {
    "servers": {
      "go": { "command": ["gopls"] },
      "typescript": { "command": ["typescript-language-server", "--stdio"] }
    }
  }
}
```

Settings and keybindings can also be opened from the command palette: **Preferences: Open Settings** and **Preferences: Open Keyboard Shortcuts**.

### Keybindings

All keybindings are customizable via `keybindings.json`. Supports chord sequences (e.g. `ctrl+k ctrl+t`). These are the defaults:

| Shortcut | Action |
|----------|--------|
| | **General** |
| Ctrl+Q | Quit |
| Ctrl+P | Command palette |
| Alt+P | Quick open file |
| Escape | Focus editor |
| | **File** |
| Ctrl+N | New file |
| Ctrl+S | Save |
| Ctrl+K S | Save as |
| | **Editor** |
| Ctrl+Z | Undo |
| Ctrl+Y | Redo |
| Ctrl+A | Select all |
| Ctrl+C | Copy |
| Ctrl+X | Cut |
| Ctrl+V | Paste |
| Ctrl+G | Go to line |
| | **Search** |
| Ctrl+F | Find |
| Ctrl+H | Find and replace |
| F3 / Shift+F3 | Find next / previous |
| | **View** |
| Ctrl+B | Toggle sidebar |
| Ctrl+J | Toggle bottom panel |
| Ctrl+K E | Show file explorer |
| Ctrl+K F | Search across files |
| Ctrl+K C | Show changes |
| Ctrl+0 | Focus sidebar |
| Ctrl+K Ctrl+T | Switch theme |
| | **Tabs** |
| Ctrl+PgDn / PgUp | Next / previous tab |
| Ctrl+W | Close tab |
| | **LSP** |
| Ctrl+U | Autocomplete |
| F12 | Go to definition |
| Shift+F12 | Go to implementation |
| Ctrl+K I | Hover info |
| | **Terminal** |
| Ctrl+` | Toggle terminal |
| Ctrl+K T | New terminal tab |
| | **Changes Panel** |
| Space | Toggle stage/unstage file |
| A | Stage all |
| U | Unstage all |
| R | Refresh |
| Enter | Open diff / activate |
| | **Menu Bar** |
| F10 / Alt+F | File menu |
| Alt+E / S / V / H | Edit / Selection / View / Help |

## Architecture

The codebase follows a strict layered architecture: **core -> view -> render -> term -> ui**.

- **`internal/core/`** — UI-agnostic editor engine (buffer, cursor, undo, syntax highlighting, multi-buffer management)
- **`internal/view/`** — Viewport scrolling and status bar
- **`internal/render/`** — Diff-based double-buffered renderer for minimal terminal writes
- **`internal/terminal/`** — Integrated terminal emulator (vt10x + pty)
- **`internal/term/`** — Terminal abstraction via `Screen` interface (only package that imports tcell)
- **`internal/ui/`** — Window manager, pane system, and all widgets
- **`internal/lsp/`** — Language Server Protocol client (JSON-RPC 2.0 over stdio, per-language server management)
- **`internal/workspace/`** — Multi-folder workspace management with git detection
- **`internal/git/`** — Git operations (status, stage, unstage, commit, pull, push, diff, blame)
- **`internal/config/`** — Configuration loading (settings, themes, keybindings, editorconfig)

## License

MIT
