---
title: Editor Basics
description: Core editing features in TTT.
---

## Syntax Highlighting

TTT uses [chroma](https://github.com/alecthomas/chroma) for syntax highlighting, supporting hundreds of languages with automatic detection based on file extension.

## Bracket Matching

Matching brackets are highlighted automatically when the cursor is on a bracket character.

## Find and Replace

- **Ctrl+F** opens the find bar with match navigation
- **Ctrl+H** opens find and replace with replace-one and replace-all
- **F3 / Shift+F3** to jump between matches

## Go to Line

Press **Ctrl+G** to open the Go to Line dialog.

## Selection and Clipboard

- **Ctrl+A** to select all
- **Ctrl+C / Ctrl+X / Ctrl+V** for copy, cut, paste with system clipboard support

## Undo and Redo

- **Ctrl+Z** to undo, **Ctrl+Y** to redo
- Uses a command-pattern undo stack for reliable history tracking

## Multi-Cursor Editing

TTT supports editing with multiple cursors simultaneously. Place cursors at multiple locations and type, delete, or insert lines — all cursors act in parallel.

### Adding Cursors

- **Ctrl+D** — select the current word (or extend the current selection) and add a cursor at the next occurrence
- **Ctrl+K L** — select all occurrences of the current word/selection at once
- **Alt+Click** — add a cursor at the clicked position

### Removing Cursors

- **Ctrl+K U** — undo the last cursor addition
- **Escape** — collapse back to a single cursor (when multiple cursors are active)

### Behavior

- Typing, backspace, delete, and enter work at all cursor positions simultaneously
- Undo/redo groups multi-cursor edits into a single action
- The status bar shows the cursor count (e.g., "3 cursors") when multiple cursors are active
- All cursors render as solid blocks

## Indentation

TTT supports `.editorconfig` files and picks up indent size automatically per file. It also auto-detects indentation from file content. You can manually override indentation via the status bar indent picker.

## Mouse Support

- Click to position the cursor
- Click tabs to switch files
- Drag sidebar and panel dividers to resize
- Right-click for context menus

## Line Numbers

Line numbers are shown in the gutter with current-line highlighting. Toggle with `lineNumbers` in settings.

## Git Blame

Inline blame info for the current line is shown in the status bar, including the author, relative time, and commit summary.
