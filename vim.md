# Vim Plugin — Implementation Plan

Vim keybinding mode implemented as a Lua plugin, not in core.
The core needs several plugin API additions first — all general-purpose, not Vim-specific.

## Core Plugin API Additions

### 1. Key Interception Hook

Plugins currently cannot intercept key events before the editor processes them.
In Vim Normal mode, pressing `j` must be a motion, not insert a character.

**What to add:**
- `ttt.events.on("key.press", fn)` event where the callback receives the key event and returns `true` to suppress default handling.
- Hook point in `Root.HandleEvent` (`internal/ui/root.go`) — after ForceKeys and overlays, before the editor widget consumes the key.
- Must only intercept when the editor pane is focused (not terminal, not plugin panels).

### 2. Persistent Status Bar Section ✅

**Done.** Status bar refactored from flat struct fields to a segment-based model. Both core and plugins contribute segments with ID, side, priority, text, style, and click handler.

- `ttt.set_status_item(side, id, text, opts)` — add/update a segment. `opts`: `{priority, on_click}`. Default priority 1000; core uses 100–500.
- `ttt.remove_status_item(id)` — remove a segment. IDs are scoped per plugin (`pluginName:id`).
- `view.StatusBar` now uses `SetSegment(StatusSegment)` / `RemoveSegment(id)` / `LeftSegments()` / `RightSegments()`.
- Segments sorted by priority (lower = closer to edge). Same priority tiebreaks by registration order.
- Core segments: branch(L:100), blame(L:200), position(R:100), indent(R:200), encoding(R:300), eol(R:400), language(R:500).

### 3. Command Execution API

Plugins cannot programmatically invoke editor commands. A Vim plugin needs undo, redo, save, delete-line, join-lines, etc.

**What to add:**
- `ttt.exec(command_id)` — execute any registered command by ID (e.g., `"editor.undo"`, `"editor.joinLines"`, `"file.save"`).
- Expose through the Lua sandbox in `internal/plugin/sandbox.go`.

### 4. Single-Line Buffer Access

`editor.get_line(n)` is missing — must copy the entire buffer via `buffer_lines()` to read one line. A Vim plugin checks character-at-cursor on every keystroke.

**What to add:**
- `editor.get_line(n)` — returns the text of line `n` (1-based) as a string.
- `editor.line_count()` — returns total number of lines in the buffer.

### 5. Viewport and Scroll Control

No way to query or control the visible viewport. Vim's `Ctrl+D`, `Ctrl+U`, `zz`, `zt`, `zb` all need this.

**What to add:**
- `editor.viewport()` — returns `{top_line, bottom_line, height}`.
- `editor.scroll_to(line)` — scroll so that `line` is at the top of the viewport.
- `editor.scroll_by(delta)` — scroll up/down by `delta` lines.

### 6. Undo Transaction Grouping

Plugin edits push individual undo entries. A Vim `cw` (delete word + enter insert mode) should undo as one step.

**What to add:**
- `editor.begin_undo_group()` — start grouping subsequent edits.
- `editor.end_undo_group()` — close the group; all edits since `begin` undo/redo as one operation.

### 7. Multi-Cursor API (nice-to-have)

Internal multi-cursor support exists but is not exposed to plugins. Would enable Vim visual-block mode.

**What to add (later):**
- `editor.add_cursor(line, col)`
- `editor.get_cursors()` — returns list of `{line, col}` tables.
- `editor.clear_cursors()` — back to single cursor.
