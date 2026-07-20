# Vim Plugin — Implementation Plan

Vim keybinding mode implemented as a Lua plugin, not in core.
The core needs several plugin API additions first — all general-purpose, not Vim-specific.

## Core Plugin API Additions

### 1. Key Interception Hook ✅

**Done.** Plugins can now intercept key events before the editor processes them.

- `ttt.events.on("key.press", fn)` — callback receives a key event table (`{type, key, rune, mod}`) and returns `true` to suppress default handling.
- Hook point in `Root.HandleEvent` (`internal/ui/root.go`) via `KeyInterceptor` — after ForceKeys, overlays, escape, chords, and RawKeyConsumer check, before the editor widget consumes the key.
- Only intercepts when the editor pane is focused (not terminal, not plugin panels).
- Gated on the existing `keybindings` permission.

### 2. Persistent Status Bar Section ✅

**Done.** Status bar refactored from flat struct fields to a segment-based model. Both core and plugins contribute segments with ID, side, priority, text, style, and click handler.

- `ttt.set_status_item(side, id, text, opts)` — add/update a segment. `opts`: `{priority, on_click}`. Default priority 1000; core uses 100–500.
- `ttt.remove_status_item(id)` — remove a segment. IDs are scoped per plugin (`pluginName:id`).
- `view.StatusBar` now uses `SetSegment(StatusSegment)` / `RemoveSegment(id)` / `LeftSegments()` / `RightSegments()`.
- Segments sorted by priority (lower = closer to edge). Same priority tiebreaks by registration order.
- Core segments: branch(L:100), blame(L:200), position(R:100), indent(R:200), encoding(R:300), eol(R:400), language(R:500).

### 3. Command Execution API ✅

**Done.** Plugins can now invoke any registered command by ID.

- `ttt.exec_command(id)` — execute a command by ID, returns `true` if found and executed, `false` otherwise.
- Gated on the existing `commands` permission (same as command registration).
- Wired via `Plugin.ExecCommand` callback → `App.Reg.Execute(id)`.

### 4. Single-Line Buffer Access ✅

**Done.** Plugins can now read individual lines without copying the entire buffer.

- `editor.get_line(n)` — returns the text of line `n` (1-based) as a string.
- `editor.line_count()` — returns total number of lines in the buffer.
- Both gated on the existing `editor.read` permission.

### 5. Viewport and Scroll Control ✅

**Done.** Plugins can now query and control the visible viewport.

- `editor.viewport()` — returns `{top_line, bottom_line, height}` (1-based). Gated on `editor.read`.
- `editor.scroll_to(line)` — scroll so that `line` (1-based) is at the top of the viewport. Gated on `editor.write`.
- `editor.scroll_by(delta)` — scroll up/down by `delta` lines. Gated on `editor.write`.

### 6. Undo Transaction Grouping ✅

**Done.** Plugins can group multiple edits into a single undo/redo step.

- `editor.begin_undo_group()` — start grouping subsequent edits. Gated on `editor.write`.
- `editor.end_undo_group()` — close the group; all edits since `begin` undo/redo as one operation. Gated on `editor.write`.
- Backed by `UndoStack.BeginTransaction()` / `EndTransaction()` which wraps accumulated commands in a `BatchCommand`.

### 7. Multi-Cursor API (nice-to-have)

Internal multi-cursor support exists but is not exposed to plugins. Would enable Vim visual-block mode.

**What to add (later):**
- `editor.add_cursor(line, col)`
- `editor.get_cursors()` — returns list of `{line, col}` tables.
- `editor.clear_cursors()` — back to single cursor.
