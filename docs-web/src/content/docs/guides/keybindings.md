---
title: Keybindings
description: Keyboard shortcuts and customization.
---

All keybindings are customizable via `keybindings.json`. TTT supports chord sequences (e.g. `ctrl+k ctrl+t`) for two-step shortcuts.

## Default Keybindings

### General

| Shortcut | Action |
|----------|--------|
| Ctrl+Q | Quit |
| Ctrl+P | Command palette |
| Alt+P | Quick open file |
| Escape | Focus editor |

### File

| Shortcut | Action |
|----------|--------|
| Ctrl+N | New file |
| Ctrl+S | Save |
| Ctrl+K S | Save as |

### Editor

| Shortcut | Action |
|----------|--------|
| Ctrl+Z | Undo |
| Ctrl+Y | Redo |
| Ctrl+A | Select all |
| Ctrl+C | Copy |
| Ctrl+X | Cut |
| Ctrl+V | Paste |
| Ctrl+G | Go to line |

### Multi-Cursor

| Shortcut | Action |
|----------|--------|
| Ctrl+D | Select next occurrence |
| Ctrl+K L | Select all occurrences |
| Alt+Click | Add cursor at click position |
| Ctrl+K U | Undo last cursor addition |
| Escape | Collapse to single cursor |

### Search

| Shortcut | Action |
|----------|--------|
| Ctrl+F | Find |
| Ctrl+H | Find and replace |
| F3 / Shift+F3 | Find next / previous |

### View

| Shortcut | Action |
|----------|--------|
| Ctrl+B | Toggle sidebar |
| Ctrl+J | Toggle bottom panel |
| Ctrl+K E | Show file explorer |
| Ctrl+K F | Search across files |
| Ctrl+K C | Show changes |
| Ctrl+0 | Focus sidebar |
| Ctrl+K Ctrl+T | Switch theme |

### Tabs

| Shortcut | Action |
|----------|--------|
| Ctrl+PgDn / PgUp | Next / previous tab |
| Ctrl+W | Close tab |

### LSP

| Shortcut | Action |
|----------|--------|
| Ctrl+U | Autocomplete |
| F12 | Go to definition |
| Shift+F12 | Go to implementation |
| F2 | Rename symbol |
| Ctrl+K I | Hover info |
| Ctrl+L F | Format document |
| Ctrl+L S | Format selection |
| Ctrl+L O | Organize imports |
| Ctrl+L X | Fix all |
| Ctrl+L R | Find references |
| Ctrl+L I | Go to implementation |
| Ctrl+L T | Go to type definition |

### Terminal

| Shortcut | Action |
|----------|--------|
| Ctrl+` | Toggle terminal |
| Ctrl+K T | New terminal tab |

### Changes Panel

| Shortcut | Action |
|----------|--------|
| Space | Toggle stage/unstage file |
| A | Stage all |
| U | Unstage all |
| R | Refresh |
| Enter | Open diff / activate |

### Menu Bar

| Shortcut | Action |
|----------|--------|
| F10 / Alt+F | File menu |
| Alt+E / S / V / H | Edit / Selection / View / Help |

## Customizing Keybindings

Create a `keybindings.json` in `~/.config/ttt/` to override defaults. The format follows VS Code conventions:

```json
[
  { "key": "ctrl+d", "command": "editor.goToDefinition" },
  { "key": "ctrl+k ctrl+s", "command": "file.saveAs" }
]
```

### Chord Sequences

TTT supports two-step chord sequences. Press the first key combination, then the second:

- `ctrl+k e` means press Ctrl+K, release, then press E
- `ctrl+l f` means press Ctrl+L, release, then press F

### Keyboard Tester

Use **View > Keyboard Tester** from the menu bar to see what key combinations your terminal supports and which commands they are bound to.
