---
title: Default Keybindings
description: Complete list of default keyboard shortcuts.
sidebar:
  order: 2
---

All keybindings can be customized in `~/.config/ttt/keybindings.json`.

## General

| Shortcut | Command | Description |
|----------|---------|-------------|
| Ctrl+Q | `editor.quit` | Quit the editor |
| Ctrl+P | `command.palette` | Open command palette |
| Alt+P | `file.quickOpen` | Quick open file |
| Escape | `editor.focus` | Focus the editor |

## File

| Shortcut | Command | Description |
|----------|---------|-------------|
| Ctrl+N | `file.new` | New file |
| Ctrl+S | `file.save` | Save |
| Ctrl+K S | `file.saveAs` | Save as |

## Editor

| Shortcut | Command | Description |
|----------|---------|-------------|
| Ctrl+Z | `editor.undo` | Undo |
| Ctrl+Y | `editor.redo` | Redo |
| Ctrl+A | `editor.selectAll` | Select all |
| Ctrl+C | `editor.copy` | Copy |
| Ctrl+X | `editor.cut` | Cut |
| Ctrl+V | `editor.paste` | Paste |
| Ctrl+G | `editor.goToLine` | Go to line |
| Alt+Up | `editor.moveLineUp` | Move line up |
| Alt+Down | `editor.moveLineDown` | Move line down |
| Alt+Shift+Up | `editor.duplicateLine` | Duplicate line |
| Alt+Shift+Down | `editor.duplicateLine` | Duplicate line |
| Ctrl+K K | `editor.deleteLine` | Delete line |
| Ctrl+Enter | `editor.insertLineBelow` | Insert line below |
| Alt+Backspace | `editor.deleteWordLeft` | Delete word left |
| Alt+Delete | `editor.deleteWordRight` | Delete word right |
| Ctrl+Delete | `editor.deleteWordRight` | Delete word right |

## Search

| Shortcut | Command | Description |
|----------|---------|-------------|
| Ctrl+F | `search.find` | Find |
| Ctrl+H | `search.replace` | Find and replace |
| F3 | `search.findNext` | Find next |
| Shift+F3 | `search.findPrev` | Find previous |

In find/replace dialogs, use Alt+C to toggle case sensitivity and Alt+R (or Alt+X in replace) for regex.

## View

| Shortcut | Command | Description |
|----------|---------|-------------|
| Ctrl+B | `sidebar.toggle` | Toggle sidebar |
| Ctrl+J | `panel.toggle` | Toggle bottom panel |
| Ctrl+K E | `sidebar.explorer` | Show file explorer |
| Ctrl+K F | `sidebar.search` | Search across files |
| Ctrl+K H | `sidebar.searchReplace` | Search and replace in files |
| Ctrl+K C | `sidebar.changes` | Show changes |
| Ctrl+0 | `sidebar.focus` | Focus sidebar |
| Ctrl+K Ctrl+T | `theme.switch` | Switch theme |

## Tabs

| Shortcut | Command | Description |
|----------|---------|-------------|
| Ctrl+PgDn | `tab.next` | Next tab |
| Ctrl+PgUp | `tab.prev` | Previous tab |
| Ctrl+W | `tab.close` | Close tab |

## LSP

| Shortcut | Command | Description |
|----------|---------|-------------|
| Ctrl+U | `editor.autocomplete` | Trigger autocomplete |
| F12 | `editor.goToDefinition` | Go to definition |
| Shift+F12 | `editor.goToImplementation` | Go to implementation |
| F2 | `editor.rename` | Rename symbol |
| Ctrl+K I | `editor.hover` | Hover info |
| Ctrl+L F | `editor.formatDocument` | Format document |
| Ctrl+L S | `editor.formatSelection` | Format selection |
| Ctrl+L O | `editor.organizeImports` | Organize imports |
| Ctrl+L X | `editor.fixAll` | Fix all |
| Ctrl+L R | `editor.findReferences` | Find references |
| Ctrl+L I | `editor.goToImplementation` | Go to implementation |
| Ctrl+L T | `editor.goToTypeDefinition` | Go to type definition |

## Terminal

| Shortcut | Command | Description |
|----------|---------|-------------|
| Ctrl+` | `terminal.toggle` | Toggle terminal |
| Ctrl+K T | `terminal.new` | New terminal tab |

## Menu Bar

| Shortcut | Command | Description |
|----------|---------|-------------|
| F10 / Alt+F | `menu.file` | File menu |
| Alt+E | `menu.edit` | Edit menu |
| Alt+S | `menu.selection` | Selection menu |
| Alt+V | `menu.view` | View menu |
| Alt+H | `menu.help` | Help menu |
