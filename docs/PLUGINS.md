# Plugin Authoring Guide

ttt supports Lua plugins that can render panels in the sidebar and bottom panel using either a widget-based API or raw cell drawing. Plugins run inside a sandboxed Lua 5.1 VM with no file system or network access by default — all capabilities are gated behind a permission system that users approve on first load.

## Table of Contents

- [Getting Started](#getting-started)
- [Plugin Lifecycle](#plugin-lifecycle)
- [Registration](#registration)
- [Widget API](#widget-api)
  - [Label](#label)
  - [Tree](#tree)
  - [List](#list)
  - [Button](#button)
  - [Input](#input)
  - [Requesting Redraws](#requesting-redraws)
- [Raw Cell API](#raw-cell-api)
- [Mixing Widgets and Raw Cells](#mixing-widgets-and-raw-cells)
- [State Management](#state-management)
- [Reconciliation and State Preservation](#reconciliation-and-state-preservation)
- [Focus and Keyboard Navigation](#focus-and-keyboard-navigation)
- [Event Handling](#event-handling)
- [Editor API](#editor-api)
- [Filesystem API](#filesystem-api)
- [System API](#system-api)
- [Network API](#network-api)
- [Events API](#events-api)
- [Styles](#styles)
- [Error Handling and Debugging](#error-handling-and-debugging)
- [Permissions Reference](#permissions-reference)
- [Lua Sandbox](#lua-sandbox)
- [Managing Plugins](#managing-plugins)
- [Examples](#examples)

---

## Getting Started

### Directory Structure

Plugins live in `~/.config/ttt/plugins/<plugin-name>/`. Each plugin is a directory containing a manifest file and one or more Lua source files:

```
~/.config/ttt/plugins/my-plugin/
  plugin.ttt.json    # manifest (required)
  init.lua           # entry point (referenced by manifest)
```

### Manifest

Every plugin requires a `plugin.ttt.json` manifest file at the root of its directory:

```json
{
  "name": "my-plugin",
  "description": "A short description of what this plugin does",
  "version": "1.0.0",
  "author": "Your Name",
  "entry": "init.lua",
  "permissions": {
    "panel.sidebar": true
  }
}
```

| Field         | Required | Description                                         |
|---------------|----------|-----------------------------------------------------|
| `name`        | yes      | Unique plugin identifier. Must match directory name. |
| `description` | no       | Shown in the plugin list dialog.                    |
| `version`     | no       | Semver version string.                              |
| `author`      | no       | Plugin author name.                                 |
| `entry`       | yes      | Path to the Lua entry point, relative to the plugin directory. |
| `permissions` | no       | Object declaring required permissions (see [Permissions Reference](#permissions-reference)). |

### Minimal Example

`~/.config/ttt/plugins/hello/plugin.ttt.json`:

```json
{
  "name": "hello",
  "entry": "init.lua",
  "permissions": { "panel.sidebar": true }
}
```

`~/.config/ttt/plugins/hello/init.lua`:

```lua
local ttt = require("ttt")

ttt.register({
  sidebar = {
    title = "Hello",
    render = function(panel)
      panel:label("Hello from a plugin!")
    end,
  },
})
```

---

## Plugin Lifecycle

### Loading

1. On startup, ttt scans `~/.config/ttt/plugins/` for directories containing `plugin.ttt.json`.
2. For each plugin, the manifest is parsed and permissions are compared against the user's stored approvals in `~/.config/ttt/plugins.ttt.json`.
3. If the plugin has been approved and its permissions haven't changed, it loads immediately: a sandboxed Lua VM is created, the entry file is executed, and `ttt.register()` wires up the panels.
4. If the plugin is new or its permissions have changed since approval, an approval dialog appears showing the requested permissions. The user can Allow or Cancel.

### Approval

When a plugin requests permissions the user hasn't approved yet, ttt shows a dialog listing each permission. Once approved, the permissions are saved to `~/.config/ttt/plugins.ttt.json` and the plugin won't ask again unless its `plugin.ttt.json` requests new permissions.

### Shutdown

When ttt exits, all plugin Lua VMs are closed. Plugins don't need cleanup logic unless they hold external resources.

### Reloading

Currently, plugins are only loaded at startup. To reload a plugin after editing its Lua files, restart ttt.

---

## Registration

The entry point Lua file must call `ttt.register()` to declare what the plugin provides. This is the only function available on the `ttt` module.

`ttt.register()` accepts a table with `sidebar` and/or `bottom` fields. A plugin can register both:

```lua
local ttt = require("ttt")

ttt.register({
  sidebar = {
    title = "Panel Title",       -- tab label
    render = function(panel)     -- called each render frame
      -- build UI here
    end,
    on_event = function(event)   -- optional: fallback event handler
      -- handle key/mouse events not consumed by widgets
    end,
  },
  bottom = {
    title = "Output",
    render = function(panel) end,
    on_event = function(event) end,
  },
})
```

| Field      | Required | Description                                                    |
|------------|----------|----------------------------------------------------------------|
| `title`    | yes      | Displayed as the panel's tab label.                            |
| `render`   | yes      | Function called each render frame. Receives a panel proxy object. |
| `on_event` | no       | Fallback handler for key/mouse events not consumed by widgets. |

**Sidebar** panels appear in the left sidebar alongside the file explorer. They require the `panel.sidebar` permission.

**Bottom** panels appear in the bottom panel alongside the terminal. They require the `panel.bottom` permission.

---

## Widget API

The `render` function receives a **panel proxy** object. Call methods on it to build a declarative widget tree. Widgets are stacked vertically in the order they are declared.

The render function is called every time the panel needs to redraw. Widget state (selection, expanded nodes, input text, cursor position) is **automatically preserved** between render calls — you don't need to manage it yourself.

### Label

Display a line of text, optionally styled.

**Simple string form:**

```lua
panel:label("Status: OK")
panel:label("Some text")
```

**Table form (with style):**

```lua
panel:label({ text = "Error occurred!", style = "danger" })
panel:label({ text = "Hint: press Enter", style = "muted" })
```

| Parameter | Type           | Description                          |
|-----------|----------------|--------------------------------------|
| argument  | string or table | String for simple text. Table with `text` and optional `style` fields. |

Labels are not focusable — they display text only and don't receive keyboard events.

---

### Tree

Hierarchical list with expandable nodes, keyboard navigation (arrows, Enter), and mouse support.

```lua
panel:tree({
  items = {
    {
      id = "src",
      label = "src/",
      icon = "📁",
      expandable = true,
      expanded = true,
      children = {
        { id = "main.go", label = "main.go", icon = "📄" },
        { id = "util.go", label = "util.go", icon = "📄", muted = true },
      },
    },
    { id = "readme", label = "README.md", icon = "📄", badge = "modified" },
  },
  indent = 2,
  on_select = function(node)
    -- called when a node is activated (Enter key or double-click)
  end,
  on_expand = function(node)
    -- called when a node is expanded
  end,
})
```

**Tree config fields:**

| Field       | Type     | Default | Description                                    |
|-------------|----------|---------|------------------------------------------------|
| `items`     | table    | `{}`    | Array of tree node tables (see below).         |
| `indent`    | number   | `2`     | Number of spaces per nesting level.            |
| `on_select` | function | nil     | Callback when a node is activated.             |
| `on_expand` | function | nil     | Callback when a node is expanded.              |

**Tree node fields:**

| Field        | Type    | Default | Description                                  |
|--------------|---------|---------|----------------------------------------------|
| `id`         | string  | —       | Unique identifier. Required for state preservation. |
| `label`      | string  | `""`    | Display text.                                |
| `icon`       | string  | `""`    | Icon string displayed before the label.      |
| `badge`      | string  | `""`    | Badge text displayed after the label.        |
| `muted`      | bool    | `false` | Render the label in a dimmed style.          |
| `expandable` | bool    | `false` | Show expand/collapse chevron indicator.      |
| `expanded`   | bool    | `false` | Initial expanded state (only used on first render — see [Reconciliation](#reconciliation-and-state-preservation)). |
| `children`   | table   | `{}`    | Array of child node tables.                  |

**Callback argument:** Both `on_select` and `on_expand` receive a Lua table with the same fields as the node that was interacted with (`id`, `label`, `icon`, `badge`, `muted`, `expanded`), plus a `children` table if the node has children.

**Keyboard navigation:** When focused, arrow keys move selection, Enter activates `on_select`, Left/Right collapse/expand nodes.

---

### List

Flat list — functionally identical to a tree but without nesting or expand/collapse behavior.

```lua
panel:list({
  items = {
    { id = "item1", label = "First item", icon = "•" },
    { id = "item2", label = "Second item", icon = "•", badge = "new" },
    { id = "item3", label = "Third item", muted = true },
  },
  on_select = function(node)
    -- called when an item is activated
  end,
})
```

**List config fields:**

| Field       | Type     | Description                                    |
|-------------|----------|------------------------------------------------|
| `items`     | table    | Array of item tables (same fields as tree nodes). |
| `on_select` | function | Callback when an item is activated.            |

Items use the same field format as [tree nodes](#tree).

---

### Button

Clickable button that responds to Enter, Space, and mouse click when focused.

```lua
panel:button({
  label = "Run Tests",
  on_click = function()
    -- handle the button press
  end,
})
```

**Button config fields:**

| Field      | Type     | Description                                    |
|------------|----------|------------------------------------------------|
| `label`    | string   | Button text. Use `&` for an accelerator: `"&Save"` underlines S. |
| `on_click` | function | Callback when the button is pressed.           |

---

### Input

Single-line text input with placeholder text and optional prefix.

```lua
panel:input({
  placeholder = "Type to search...",
  prefix = "🔍 ",
  on_change = function(text)
    -- called on every keystroke with the current text
  end,
  on_submit = function(text)
    -- called when Enter is pressed with the current text
  end,
})
```

**Input config fields:**

| Field         | Type     | Default | Description                                    |
|---------------|----------|---------|------------------------------------------------|
| `placeholder` | string   | `""`    | Grayed-out hint text shown when input is empty.|
| `prefix`      | string   | `""`    | Non-editable text displayed before the input.  |
| `on_change`   | function | nil     | Called after every text change. Receives the current text as a string. |
| `on_submit`   | function | nil     | Called when Enter is pressed. Receives the current text as a string. |

The input text and cursor position are automatically preserved across re-renders.

---

### Requesting Redraws

The editor only redraws when events occur (key press, mouse, resize). If your plugin's state changes outside of event handling — for example, after modifying a data table in a callback — call `panel:redraw()` to request a redraw:

```lua
local items = { { id = "1", label = "Loading..." } }

ttt.register({
  sidebar = {
    title = "Data",
    render = function(panel)
      panel:list({
        items = items,
        on_select = function(node)
          -- modify the data
          table.insert(items, { id = tostring(#items + 1), label = "New item" })
          -- request the panel to redraw with updated data
          panel:redraw()
        end,
      })
    end,
  },
})
```

`panel:redraw()` posts an event to the editor's event loop. The render function will be called again on the next frame, and the updated state will be reflected. This is safe to call from any callback.

---

## Raw Cell API

For full control over pixel-level rendering, use the low-level cell API. This draws directly to the panel surface without the widget abstraction.

### `panel:size()`

Returns the panel's width and height in cells.

```lua
local w, h = panel:size()
```

**Returns:** two numbers — width, height.

### `panel:cell(x, y, char, [style])`

Draw a single character at the given position.

```lua
panel:cell(0, 0, "X")                         -- default style
panel:cell(1, 0, "Y", "success")              -- named style as string
panel:cell(2, 0, "Z", { style = "danger" })   -- named style as table
```

| Parameter | Type            | Description                                |
|-----------|-----------------|--------------------------------------------|
| `x`       | number          | Column (0-based).                          |
| `y`       | number          | Row (0-based).                             |
| `char`    | string          | Single character to draw (first rune used).|
| `style`   | string or table | Optional. Named style string or table with `style` field. |

### `panel:text(x, y, text, [style])`

Draw a text string starting at the given position.

```lua
panel:text(0, 0, "Hello World")                -- default style
panel:text(0, 1, "Error!", "danger")           -- named style as string
panel:text(0, 2, "Hint", { style = "muted" }) -- named style as table
```

| Parameter | Type            | Description                                |
|-----------|-----------------|--------------------------------------------|
| `x`       | number          | Starting column (0-based).                 |
| `y`       | number          | Row (0-based).                             |
| `text`    | string          | Text to draw.                              |
| `style`   | string or table | Optional. Named style.                     |

### `panel:clear(x, y, w, h)`

Clear a rectangular region, filling it with spaces.

```lua
panel:clear(0, 0, 40, 10)
```

| Parameter | Type   | Description         |
|-----------|--------|---------------------|
| `x`       | number | Left column.        |
| `y`       | number | Top row.            |
| `w`       | number | Width in cells.     |
| `h`       | number | Height in cells.    |

---

## Mixing Widgets and Raw Cells

You can use both the widget API and the raw cell API in the same render function. When mixed:

- **Widgets** are rendered from the top of the panel in a vertical stack.
- **Raw cells** are drawn directly to the surface and can overlap or fill areas below the widgets.

```lua
render = function(panel)
  -- Widget at the top
  panel:label({ text = "Status Bar", style = "muted" })

  -- Raw drawing below
  local w, h = panel:size()
  for x = 0, w - 1 do
    panel:cell(x, 2, "─", "border")
  end
  panel:text(0, 3, "Custom rendered content")
end
```

Note that raw cell coordinates are relative to the full panel surface, not offset by widget height. If your widgets take up 2 rows, drawing at y=0 will overlap them.

---

## State Management

Plugin state lives in **Lua local variables** in your entry file. The Lua VM persists for the lifetime of the ttt process, so locals defined at the top level retain their values between render calls:

```lua
local ttt = require("ttt")

-- This table persists across renders
local counter = 0
local items = {}

ttt.register({
  sidebar = {
    title = "Counter",
    render = function(panel)
      panel:label("Count: " .. counter)
      panel:button({
        label = "&Increment",
        on_click = function()
          counter = counter + 1
          table.insert(items, { id = tostring(counter), label = "Item " .. counter })
          panel:redraw()
        end,
      })
      panel:list({ items = items })
    end,
  },
})
```

**Pattern: reactive updates.** Modify your state in callbacks, then call `panel:redraw()` to trigger a re-render with the new state. The render function reads from the current state each time it's called.

---

## Reconciliation and State Preservation

The widget system uses a **reconciliation** algorithm to preserve interactive state across re-renders:

1. Each render call builds a list of widget descriptors (label, tree, button, etc.).
2. After the render function returns, ttt compares the new descriptors against the previous frame.
3. If a widget at the same position has the same type, it is **updated in place** — preserving user-facing state.
4. If the type changed, the old widget is replaced with a new one.

**What is preserved:**
- **Tree/List:** Expanded/collapsed state of nodes (matched by `id`), selected item, scroll position.
- **Input:** Text content, cursor position.
- **Button:** Focus state.

**What is NOT preserved:**
- Anything that changes when you provide different data (labels, items, badges, icons — these update to reflect the new values).
- State across type changes (if position 0 was a label and becomes a tree, the tree starts fresh).

**Important:** Node `id` fields are critical for tree state preservation. If you change a node's `id`, it will be treated as a new node and lose its expanded state. Use stable, unique IDs.

---

## Focus and Keyboard Navigation

The plugin panel manages its own focus system. When the plugin panel is focused:

- **Tab** / **Shift+Tab** cycles focus between focusable widgets (trees, lists, inputs, buttons).
- **Arrow keys** navigate within the focused widget (e.g., up/down in a tree).
- **Enter** / **Space** activate the focused widget (select tree node, press button, submit input).
- Events not consumed by the focused widget fall through to the `on_event` handler.

Labels are not focusable. If your panel has only labels and raw cells, all events go directly to `on_event`.

---

## Event Handling

There are two levels of event handling:

### 1. Widget Callbacks (Automatic)

Widget-specific events are routed automatically to the callbacks you provide:

| Widget | Event        | Callback     | Argument                            |
|--------|--------------|--------------|-------------------------------------|
| Tree   | node activated | `on_select` | Node table (`{id, label, ...}`)     |
| Tree   | node expanded  | `on_expand` | Node table                          |
| List   | item activated | `on_select` | Node table                          |
| Button | pressed      | `on_click`   | (none)                              |
| Input  | text changed | `on_change`  | Current text (string)               |
| Input  | Enter pressed| `on_submit`  | Current text (string)               |

### 2. Fallback Event Handler (`on_event`)

For key and mouse events not consumed by widgets, the `on_event` function receives a Lua table describing the event:

**Key event:**

```lua
{
  type = "key",
  key = "Enter",     -- key name: "Enter", "Tab", "Escape", "Up", "Down", "Left", "Right",
                     -- "Backspace", "Delete", "Home", "End", "PgUp", "PgDn",
                     -- or the character itself ("a", "A", "1", "/", etc.)
  mod = "ctrl",      -- modifier: "ctrl", "shift", "alt", or nil for no modifier
}
```

**Mouse event:**

```lua
{
  type = "mouse",
  x = 5,             -- column (0-based, relative to panel)
  y = 10,            -- row (0-based, relative to panel)
  button = "left",   -- "left", "right", "middle"
}
```

**Example:**

```lua
on_event = function(event)
  if event.type == "key" then
    if event.key == "r" and event.mod == nil then
      -- refresh data
      reload_data()
      panel:redraw()
    end
  end
end
```

---

## Editor API

The `ttt.editor` module provides read and write access to the active editor buffer. All line and column numbers are **1-based** in Lua.

```lua
local editor = require("ttt.editor")
```

### Read Functions

Require the `editor.read` permission.

| Function               | Returns                                                   | Description                          |
|------------------------|-----------------------------------------------------------|--------------------------------------|
| `editor.buffer_text()` | string                                                    | Full buffer content as a single string. |
| `editor.buffer_lines()`| table of strings                                          | Array of lines.                      |
| `editor.current_line()`| string                                                    | Text of the line at cursor.          |
| `editor.cursor()`      | `{line, col}`                                             | Current cursor position (1-based).   |
| `editor.selection()`   | `{active, start_line, start_col, end_line, end_col}`     | Selection state (1-based). `active` is boolean. |
| `editor.selection_text()` | string                                                 | Selected text (empty if no selection).|
| `editor.file_path()`   | string                                                    | Absolute path of the active file.    |
| `editor.file_name()`   | string                                                    | Filename only.                       |
| `editor.language()`    | string                                                    | Detected language (e.g. `"go"`, `"lua"`). |

### Write Functions

Require the `editor.write` permission.

| Function                                          | Description                                    |
|---------------------------------------------------|------------------------------------------------|
| `editor.insert(line, col, text)`                  | Insert text at position.                       |
| `editor.replace(start_line, start_col, end_line, end_col, text)` | Replace a range with text.     |
| `editor.set_cursor(line, col)`                    | Move the cursor.                               |
| `editor.set_selection(start_line, start_col, end_line, end_col)` | Set selection range.           |
| `editor.clear_selection()`                        | Clear the current selection.                   |

All write operations go through the undo system — they can be undone with Ctrl+Z.

**Example:**

```lua
local editor = require("ttt.editor")

-- Read cursor position and current line
local pos = editor.cursor()
local line = editor.current_line()

-- Insert text at cursor
editor.insert(pos.line, pos.col, "// TODO: ")
```

---

## Filesystem API

The `ttt.fs` module provides file system access.

```lua
local fs = require("ttt.fs")
```

| Function             | Permission | Returns                    | Description                    |
|----------------------|------------|----------------------------|--------------------------------|
| `fs.read(path)`      | `fs.read`  | string, or nil + error     | Read file contents.            |
| `fs.write(path, content)` | `fs.write` | nil, or error string  | Write content to file.         |
| `fs.exists(path)`    | `fs.read`  | boolean                    | Check if path exists.          |
| `fs.list(path)`      | `fs.read`  | table of `{name, is_dir}`, or nil + error | List directory entries. |

**Example:**

```lua
local fs = require("ttt.fs")

if fs.exists("/tmp/config.json") then
  local content = fs.read("/tmp/config.json")
  -- process content
end

local entries = fs.list("/home/user/project")
for _, entry in ipairs(entries) do
  if entry.is_dir then
    -- it's a directory
  end
end
```

---

## System API

The `ttt.system` module provides command execution and environment variable access.

```lua
local sys = require("ttt.system")
```

### `sys.exec(binary, args)`

Execute a command synchronously. Requires `system.exec` permission with the binary listed in the allowlist.

**Returns** a table: `{stdout, stderr, exit_code}`.

```lua
local result = sys.exec("git", {"status", "--porcelain"})
if result.exit_code == 0 then
  -- parse result.stdout
end
```

### `sys.exec_async(binary, args, callback)`

Execute a command asynchronously. The callback receives the same result table and is called on the main thread when the command completes. The UI remains responsive during execution.

```lua
sys.exec_async("docker", {"ps", "--format", "{{.Names}}"}, function(result)
  -- process result.stdout
  panel:redraw()
end)
```

### `sys.env(name)`

Read an environment variable. Requires `system.env` permission.

```lua
local home = sys.env("HOME")
```

---

## Network API

The `ttt.net` module provides HTTP request capabilities. Requires the `network.http` permission.

```lua
local net = require("ttt.net")
```

### `net.get(url, [opts])`

Synchronous HTTP GET.

```lua
local resp = net.get("https://api.example.com/data")
-- resp.status (number), resp.body (string), resp.headers (table), resp.error (string or nil)

-- With custom headers:
local resp = net.get("https://api.example.com/data", {
  headers = { ["Authorization"] = "Bearer token" },
})
```

### `net.post(url, opts)`

Synchronous HTTP POST.

```lua
local resp = net.post("https://api.example.com/data", {
  headers = { ["Content-Type"] = "application/json" },
  body = '{"key": "value"}',
})
```

### Async Variants

`net.get_async(url, [opts], callback)` and `net.post_async(url, opts, callback)` work like their sync counterparts but don't block the UI. The callback receives the response table.

```lua
net.get_async("https://api.example.com/data", function(resp)
  if resp.status == 200 then
    -- process resp.body
    panel:redraw()
  end
end)
```

---

## Events API

The `ttt.events` module lets plugins react to editor lifecycle events.

```lua
local events = require("ttt.events")
```

### `events.on(event_name, callback)`

Register an event listener. Multiple listeners can be registered for the same event.

**File events** (require `events.file` permission):

| Event         | Callback argument | Description                 |
|---------------|-------------------|-----------------------------|
| `file.open`   | path (string)     | A file was opened.          |
| `file.close`  | path (string)     | A file tab was closed.      |
| `file.save`   | path (string)     | A file was saved.           |

**Editor events** (require `events.editor` permission):

| Event           | Callback argument | Description                 |
|-----------------|-------------------|-----------------------------|
| `editor.change` | path (string)     | Buffer content changed.     |

**Example:**

```lua
local events = require("ttt.events")

events.on("file.save", function(path)
  -- auto-format or run linter after save
end)

events.on("file.open", function(path)
  -- load file-specific config
end)
```

---

## Styles

Named styles available for both widget and raw cell rendering. Actual colors depend on the user's theme.

| Name       | Typical Usage                  |
|------------|--------------------------------|
| `default`  | Normal text                    |
| `muted`    | Dimmed, secondary text         |
| `border`   | Panel borders, separators      |
| `success`  | Green, positive status         |
| `danger`   | Red, errors                    |
| `warning`  | Yellow, caution                |
| `selected` | Highlighted/selected item      |
| `item`     | List/palette item              |
| `line`     | Line numbers                   |
| `input`    | Input field text               |

Styles can be passed as a string or a table:

```lua
panel:text(0, 0, "OK", "success")                -- string form
panel:cell(0, 0, "X", { style = "danger" })       -- table form
panel:label({ text = "Note", style = "muted" })   -- in widget config
```

If an unrecognized style name is used, it falls back to `default`.

---

## Error Handling and Debugging

### Render Errors

If your `render` function throws a Lua error, the plugin panel displays the error message in red (`danger` style) instead of the normal content. The plugin stays loaded and will retry rendering on the next frame — so fixing the error in your Lua file and restarting ttt will recover.

Errors are also logged to stderr, so if you run ttt from a terminal you'll see them there.

### Callback Errors

If a callback function (`on_select`, `on_click`, etc.) throws an error, it is caught and logged. The error is stored on the plugin's `LastError` field and displayed in the plugin list (accessible via the command palette: **Plugins: List Installed**).

### Debugging Tips

- **Start simple.** Begin with a single `panel:label()` to verify your plugin loads, then add complexity.
- **Use labels for debugging.** Since there's no `print()` to the console, use `panel:label()` to display variable values in your panel.
- **Check the plugin list.** Open the command palette and run **Plugins: List Installed** to see status (enabled/disabled/error) and version for each plugin.
- **Watch stderr.** Run ttt from a terminal to see error logs: `ttt 2>plugin-errors.log`.

---

## Permissions Reference

Permissions are declared in the manifest's `permissions` object. Boolean permissions are set to `true`; array permissions list specific values.

| Permission       | Type     | Description                                       |
|------------------|----------|---------------------------------------------------|
| `panel.sidebar`  | bool     | Register a sidebar panel.                         |
| `panel.bottom`   | bool     | Register a bottom panel.                          |
| `panel.drawer`   | bool     | Register a drawer panel. (Reserved for future use.) |
| `commands`       | bool     | Register commands in the command palette.         |
| `keybindings`    | bool     | Bind keyboard shortcuts.                          |
| `editor.read`    | bool     | Read the contents of editor buffers.              |
| `editor.write`   | bool     | Modify the contents of editor buffers.            |
| `fs.read`        | bool     | Read files from the file system.                  |
| `fs.write`       | bool     | Write files to the file system.                   |
| `system.exec`    | string[] | Execute specific system commands. List each binary. |
| `system.env`     | bool     | Read environment variables.                       |
| `network.http`   | bool     | Make outbound HTTP requests.                      |
| `events.file`    | bool     | Receive file system change events.                |
| `events.editor`  | bool     | Receive editor events (cursor move, file open, etc.). |

**Example with multiple permissions:**

```json
{
  "permissions": {
    "panel.sidebar": true,
    "panel.bottom": true,
    "system.exec": ["git", "npm"],
    "fs.read": true
  }
}
```

Currently implemented: `panel.sidebar`, `panel.bottom`, `editor.read`, `editor.write`, `fs.read`, `fs.write`, `system.exec`, `system.env`, `network.http`, `events.file`, `events.editor`.

---

## Lua Sandbox

Plugins run in a sandboxed Lua 5.1 environment. Only safe standard library modules are available:

| Module   | Available | Notes                        |
|----------|-----------|------------------------------|
| `base`   | yes       | `print`, `type`, `tostring`, `tonumber`, `pairs`, `ipairs`, `select`, `unpack`, `error`, `pcall`, `xpcall`, `assert`, `rawget`, `rawset`, `rawequal`, `setmetatable`, `getmetatable` |
| `string` | yes       | Full string library (`format`, `find`, `gsub`, `match`, `sub`, `rep`, `upper`, `lower`, `byte`, `char`, `len`, `reverse`) |
| `table`  | yes       | Full table library (`insert`, `remove`, `sort`, `concat`, `maxn`) |
| `math`   | yes       | Full math library (`floor`, `ceil`, `sqrt`, `sin`, `cos`, `random`, `pi`, etc.) |
| `os`     | **no**    | Blocked entirely.            |
| `io`     | **no**    | Blocked entirely.            |
| `debug`  | **no**    | Not loaded.                  |

**Removed globals:** `dofile`, `loadfile` are set to nil.

**Module loading:** `require()` allows `"ttt"`, `"ttt.editor"`, `"ttt.fs"`, `"ttt.system"`, `"ttt.net"`, and `"ttt.events"`. Any other module name raises an error.

---

## Managing Plugins

### Installing

Copy or clone the plugin directory into `~/.config/ttt/plugins/`:

```sh
cp -r my-plugin ~/.config/ttt/plugins/
```

Restart ttt. If the plugin is new, you'll see an approval dialog.

### Plugin List

Open the command palette (`Ctrl+Shift+P`) and run **Plugins: List Installed** to see all installed plugins with their status and version.

### Removing

Delete the plugin directory:

```sh
rm -rf ~/.config/ttt/plugins/my-plugin
```

The approval record in `~/.config/ttt/plugins.ttt.json` can be left — it's harmless and will be ignored.

---

## Examples

### File Tree Browser

A sidebar panel that shows a static file tree:

```json
{
  "name": "file-tree",
  "entry": "init.lua",
  "permissions": { "panel.sidebar": true }
}
```

```lua
local ttt = require("ttt")

local tree = {
  {
    id = "src", label = "src/", icon = "📁", expandable = true, expanded = true,
    children = {
      { id = "src/main.go", label = "main.go", icon = "📄" },
      { id = "src/server.go", label = "server.go", icon = "📄" },
      {
        id = "src/handlers", label = "handlers/", icon = "📁", expandable = true,
        children = {
          { id = "src/handlers/auth.go", label = "auth.go", icon = "📄" },
          { id = "src/handlers/api.go", label = "api.go", icon = "📄" },
        },
      },
    },
  },
  { id = "go.mod", label = "go.mod", icon = "📄", muted = true },
  { id = "README.md", label = "README.md", icon = "📄", muted = true },
}

local selected_file = nil

ttt.register({
  sidebar = {
    title = "Files",
    render = function(panel)
      if selected_file then
        panel:label({ text = "Selected: " .. selected_file, style = "muted" })
      end
      panel:tree({
        items = tree,
        indent = 2,
        on_select = function(node)
          selected_file = node.label
          panel:redraw()
        end,
      })
    end,
  },
})
```

### Search Panel with Input

A bottom panel with a search input and results list:

```json
{
  "name": "search-panel",
  "entry": "init.lua",
  "permissions": { "panel.bottom": true }
}
```

```lua
local ttt = require("ttt")

local results = {}
local query = ""

local function do_search(text)
  query = text
  results = {}
  -- In a real plugin with fs.read permission, you'd search files here.
  -- For demo, generate fake results:
  if #text > 0 then
    for i = 1, 5 do
      table.insert(results, {
        id = tostring(i),
        label = "Result " .. i .. " for '" .. text .. "'",
        badge = "line " .. (i * 10),
      })
    end
  end
end

ttt.register({
  bottom = {
    title = "Search",
    render = function(panel)
      panel:input({
        placeholder = "Search...",
        prefix = "🔍 ",
        on_change = function(text)
          do_search(text)
          panel:redraw()
        end,
        on_submit = function(text)
          do_search(text)
          panel:redraw()
        end,
      })
      if #results > 0 then
        panel:label({ text = #results .. " results for '" .. query .. "'", style = "muted" })
        panel:list({
          items = results,
          on_select = function(node)
            -- In a real plugin, you'd open the file at the matching line
          end,
        })
      elseif #query > 0 then
        panel:label({ text = "No results", style = "warning" })
      end
    end,
  },
})
```

### Raw Cell Drawing — Progress Bar

A sidebar panel using only the raw cell API:

```json
{
  "name": "progress",
  "entry": "init.lua",
  "permissions": { "panel.sidebar": true }
}
```

```lua
local ttt = require("ttt")

local progress = 0.65  -- 65%

ttt.register({
  sidebar = {
    title = "Progress",
    render = function(panel)
      local w, h = panel:size()

      panel:text(0, 0, "Build Progress", "muted")

      -- Draw progress bar
      local bar_w = w - 2
      local filled = math.floor(bar_w * progress)

      for x = 0, bar_w - 1 do
        if x < filled then
          panel:cell(x + 1, 2, "█", "success")
        else
          panel:cell(x + 1, 2, "░", "muted")
        end
      end

      -- Draw percentage
      local pct = math.floor(progress * 100) .. "%"
      panel:text(1, 4, pct, "default")
    end,
  },
})
```

### TODO List (Sidebar + Bottom)

A plugin that registers both a sidebar list and a bottom detail view:

```json
{
  "name": "todo-list",
  "entry": "init.lua",
  "version": "0.1.0",
  "permissions": {
    "panel.sidebar": true,
    "panel.bottom": true
  }
}
```

```lua
local ttt = require("ttt")

local todos = {
  { id = "1", label = "Write plugin docs", done = false },
  { id = "2", label = "Add tests", done = false },
  { id = "3", label = "Ship it!", done = false },
}

local selected = nil

local function todo_items()
  local items = {}
  for _, t in ipairs(todos) do
    table.insert(items, {
      id = t.id,
      label = t.label,
      muted = t.done,
      badge = t.done and "✓" or "",
    })
  end
  return items
end

ttt.register({
  sidebar = {
    title = "TODOs",
    render = function(panel)
      panel:label({ text = #todos .. " tasks", style = "muted" })
      panel:list({
        items = todo_items(),
        on_select = function(node)
          selected = node.id
          -- toggle done
          for _, t in ipairs(todos) do
            if t.id == node.id then
              t.done = not t.done
              break
            end
          end
          panel:redraw()
        end,
      })
      panel:button({
        label = "&Add Task",
        on_click = function()
          local id = tostring(#todos + 1)
          table.insert(todos, { id = id, label = "New task " .. id, done = false })
          panel:redraw()
        end,
      })
    end,
  },
  bottom = {
    title = "Task Detail",
    render = function(panel)
      if selected == nil then
        panel:label({ text = "Select a task in the sidebar", style = "muted" })
        return
      end
      for _, t in ipairs(todos) do
        if t.id == selected then
          panel:label("Task: " .. t.label)
          panel:label({ text = "Status: " .. (t.done and "Done" or "Pending"),
                        style = t.done and "success" or "warning" })
          return
        end
      end
      panel:label({ text = "Task not found", style = "danger" })
    end,
  },
})
```

### Git Status Panel (Editor + System + Events)

A sidebar panel that shows `git status` output and refreshes on file save:

```json
{
  "name": "git-status",
  "entry": "init.lua",
  "version": "0.1.0",
  "permissions": {
    "panel.sidebar": true,
    "system.exec": ["git"],
    "events.file": true
  }
}
```

```lua
local ttt = require("ttt")
local sys = require("ttt.system")
local events = require("ttt.events")

local files = {}
local error_msg = nil

local function refresh(panel)
  local result = sys.exec("git", {"status", "--porcelain"})
  if result.exit_code ~= 0 then
    error_msg = result.stderr
    files = {}
  else
    error_msg = nil
    files = {}
    for line in result.stdout:gmatch("[^\n]+") do
      local status = line:sub(1, 2):match("%S+") or "?"
      local path = line:sub(4)
      table.insert(files, {
        id = path,
        label = path,
        badge = status,
        muted = status == "?",
      })
    end
  end
  if panel then panel:redraw() end
end

ttt.register({
  sidebar = {
    title = "Git",
    render = function(panel)
      if error_msg then
        panel:label({ text = error_msg, style = "danger" })
        return
      end
      if #files == 0 then
        panel:label({ text = "Working tree clean", style = "muted" })
        return
      end
      panel:label({ text = #files .. " changed files", style = "muted" })
      panel:list({
        items = files,
        on_select = function(node)
          -- could open the file here with editor API
        end,
      })
      panel:button({
        label = "&Refresh",
        on_click = function()
          refresh(panel)
        end,
      })
    end,
  },
})

-- Refresh on startup
refresh(nil)

-- Auto-refresh after file saves
events.on("file.save", function(path)
  -- Use exec_async to avoid blocking the UI
  sys.exec_async("git", {"status", "--porcelain"}, function(result)
    if result.exit_code == 0 then
      error_msg = nil
      files = {}
      for line in result.stdout:gmatch("[^\n]+") do
        local status = line:sub(1, 2):match("%S+") or "?"
        local fpath = line:sub(4)
        table.insert(files, {
          id = fpath,
          label = fpath,
          badge = status,
          muted = status == "?",
        })
      end
    end
  end)
end)
```
