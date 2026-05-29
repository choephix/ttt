---
title: Quick Start
description: Get up and running with TTT in minutes.
sidebar:
  order: 3
---

## Opening Files and Folders

```sh
ttt                             # opens the current directory
ttt /path/to/dir                # opens that directory as the workspace
ttt /path/to/file.go            # opens the file
ttt dir1 dir2                   # opens multiple folders
ttt --workspace project.ttt     # loads a saved workspace file
```

## Basic Navigation

- **Ctrl+P** opens the command palette
- **Alt+P** opens quick file open
- **Ctrl+B** toggles the sidebar
- **Ctrl+T** toggles the terminal (half screen)
- **Alt+T** toggles the terminal fullscreen
- **Ctrl+G** opens Go to Line

## Editing

- **Ctrl+Z / Ctrl+Y** for undo/redo
- **Ctrl+C / Ctrl+X / Ctrl+V** for copy/cut/paste
- **Ctrl+F** to find, **Ctrl+H** to find and replace
- **Ctrl+A** to select all
- **Ctrl+D** to select the next occurrence (multi-cursor)
- **Alt+Click** to add cursors at multiple positions

## Tabs

- **Ctrl+PgDn / Ctrl+PgUp** to switch tabs
- **Ctrl+W** to close a tab
- Opening a file replaces the current unpinned tab
- Clicking an already-open tab pins it

## Command Palette

Press **Ctrl+P** to open the command palette. All available commands are listed there. Type `>` to switch between file search and command mode.

## Configuration

Config files live in `~/.config/ttt/`:

| File | Purpose |
|------|---------|
| `settings.json` | Editor settings |
| `keybindings.json` | Custom keybindings |
| `themes/*.json` | Custom themes |

You can also open these from the command palette: **Preferences: Open Settings** and **Preferences: Open Keyboard Shortcuts**.
