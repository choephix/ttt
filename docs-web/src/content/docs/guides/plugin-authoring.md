---
title: Plugin Authoring
description: Complete guide to creating plugins for TTT.
---

TTT supports Lua plugins that can render panels in the sidebar and bottom panel using either a widget-based API or raw cell drawing. Plugins run inside a sandboxed Lua 5.1 VM with no file system or network access by default — all capabilities are gated behind a permission system that users approve on first load.

## Getting Started

### Directory Structure

Plugins live in one of two locations:

- **Global:** `~/.config/ttt/plugins/<plugin-name>/` — available in all sessions.
- **Workspace-local:** `<workspace-root>/plugins/<plugin-name>/` — scoped to a specific project. The workspace root is the primary folder opened by ttt.

If a plugin with the same name exists in both locations, the global one takes precedence.

Each plugin is a directory containing a manifest file and one or more Lua source files:

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
  "api": 1,
  "permissions": {
    "panel.sidebar": true
  }
}
```

| Field         | Type   | Required | Description                                         |
|---------------|--------|----------|-----------------------------------------------------|
| `name`        | string | yes      | Unique plugin identifier. Must match directory name. |
| `description` | string | no       | Shown in the plugin list dialog.                    |
| `version`     | string | no       | Semver version string.                              |
| `author`      | string | no       | Plugin author name.                                 |
| `entry`       | string | yes      | Path to the Lua entry point, relative to the plugin directory. |
| `api`         | number | no       | Plugin API version the plugin targets. Defaults to `1` when omitted. The editor refuses to load plugins that target a newer API than it supports. |
| `permissions` | object | no       | Object declaring required permissions (see [Permissions Reference](#permissions-reference)). |

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

## Plugin Lifecycle

### Loading

1. On startup, ttt scans `~/.config/ttt/plugins/` and `<workspace-root>/plugins/` for directories containing `plugin.ttt.json`.
2. For each plugin, the manifest is parsed and permissions are compared against the user's stored approvals in `~/.config/ttt/plugins.ttt.json`.
3. If the plugin has been approved and its permissions haven't changed, it loads immediately: a sandboxed Lua VM is created, the entry file is executed, and `ttt.register()` wires up the panels.
4. If the plugin is new or its permissions have changed since approval, an approval dialog appears showing the requested permissions. The user can Allow or Cancel.

### Approval

When a plugin requests permissions the user hasn't approved yet, ttt shows a dialog listing each permission. Once approved, the permissions are saved to `~/.config/ttt/plugins.ttt.json` and the plugin won't ask again unless its `plugin.ttt.json` requests new permissions.

### Shutdown

When ttt exits, all plugin Lua VMs are closed. Plugins don't need cleanup logic unless they hold external resources.

### Reloading

Run **Plugins: Reload** from the command palette to reload a plugin without restarting ttt. This destroys the plugin's Lua VM, re-reads the manifest, and re-initializes. State is reset — local variables start fresh. Use **Plugins: Reload All** to reload every loaded plugin at once.

## Registration

The entry point Lua file must call `ttt.register()` to declare what the plugin provides. `ttt.register()` accepts a single table with `sidebar`, `bottom`, `commands`, and/or `keybindings` fields. A plugin can register any combination:

```lua
local ttt = require("ttt")

ttt.register({
  sidebar = {
    title = "Panel Title",
    render = function(panel) end,
    on_event = function(event) end,
    actions = { ... },
    on_action = function(command) end,
  },
  bottom = {
    title = "Output",
    render = function(panel) end,
    on_event = function(event) end,
  },
  commands = {
    { id = "myplugin.doSomething", title = "My Plugin: Do Something", handler = function() end },
  },
  keybindings = {
    { key = "ctrl+k d", command = "myplugin.doSomething" },
  },
})
```

### Sidebar Panel

Sidebar panels appear in the left sidebar alongside the file explorer. Requires the `panel.sidebar` permission.

| Field       | Type     | Required | Description                                                             |
|-------------|----------|----------|-------------------------------------------------------------------------|
| `title`     | string   | yes      | Displayed as the panel's tab label.                                     |
| `render`    | function | yes      | Called each render frame. Receives a [panel proxy](#widget-api) object. |
| `on_event`  | function | no       | Fallback handler for key/mouse events not consumed by widgets.          |
| `actions`   | table    | no       | Array of menu entries for the panel's "..." header menu.                |
| `on_action` | function | no       | Callback when a header menu action is selected. Receives the command string. |

The `actions` array defines entries for the sidebar panel's header menu (the "..." button). Each entry uses the [menu entry format](#menu-entry-format):

```lua
sidebar = {
  title = "Docker",
  actions = {
    { label = "Refresh All", command = "refresh" },
    { separator = true },
    { label = "Prune Containers", command = "prune_containers" },
  },
  on_action = function(command)
    if command == "refresh" then
      refresh()
    elseif command == "prune_containers" then
      prune()
    end
  end,
  render = function(panel) ... end,
}
```

### Bottom Panel

Bottom panels appear in the bottom panel alongside the terminal. Requires the `panel.bottom` permission.

| Field      | Type     | Required | Description                                                    |
|------------|----------|----------|----------------------------------------------------------------|
| `title`    | string   | yes      | Displayed as the panel's tab label.                            |
| `render`   | function | yes      | Called each render frame. Receives a panel proxy object.       |
| `on_event` | function | no       | Fallback handler for key/mouse events not consumed by widgets. |

### Commands

Plugins can register commands that appear in the command palette. Requires the `commands` permission.

Each command entry is a table with:

| Field     | Type     | Required | Description                              |
|-----------|----------|----------|------------------------------------------|
| `id`      | string   | yes      | Unique command ID (e.g. `myplugin.run`). |
| `title`   | string   | yes      | Display title in the command palette.     |
| `handler` | function | yes      | Function called when the command runs.    |

### Keybindings

Plugins can register keyboard shortcuts for their commands. Requires the `keybindings` permission.

Each keybinding entry is a table with:

| Field     | Type   | Required | Description                                           |
|-----------|--------|----------|-------------------------------------------------------|
| `key`     | string | yes      | Key combination string (e.g. `ctrl+k d` for a chord). |
| `command` | string | yes      | Command ID to execute when the key is pressed.        |

Use `ctrl+k <key>` chords for plugin keybindings to avoid conflicts with built-in shortcuts. Avoid `ctrl+shift` combos — they are unreliable in many terminals.

## ttt Module Functions

These functions are available on the `ttt` module directly:

```lua
local ttt = require("ttt")
```

### `ttt.register(config)`

Registers the plugin's panels, commands, and keybindings. See [Registration](#registration) for the full config table format.

### `ttt.log(message)` / `ttt.log(level, message)`

Log a message to the OUTPUT panel. No permission required.

```lua
ttt.log("plugin loaded")                -- defaults to "info" level
ttt.log("warn", "config not found")     -- explicit level
ttt.log("error", "connection failed")   -- error level
```

See [Logging](#logging) for details.

### `ttt.confirm(message, callback)`

Show a confirmation dialog. The callback is called (with no arguments) only if the user clicks "Allow" / confirms.

| Parameter  | Type     | Description                                       |
|------------|----------|---------------------------------------------------|
| `message`  | string   | Question or warning text shown in the dialog.     |
| `callback` | function | Called with no arguments if the user confirms.    |

```lua
ttt.confirm("Remove container 'web-app'?", function()
  -- only runs if user confirmed
  sys.exec_async("docker", {"rm", "-f", "web-app"}, function(result)
    ttt.log("Removed container")
    panel:redraw()
  end)
end)
```

### `ttt.show_info(title, entries)`

Show an informational dialog with key-value pairs.

| Parameter | Type   | Description                                          |
|-----------|--------|------------------------------------------------------|
| `title`   | string | Dialog title.                                        |
| `entries` | table  | Array of `{key = string, value = string}` tables.    |

```lua
ttt.show_info("Shortcuts", {
  { key = "r", value = "Refresh all" },
  { key = "Enter", value = "Expand / collapse" },
  { key = "Ctrl+K r", value = "Refresh (global)" },
})
```

### `ttt.notify(message[, level])`

Show a transient message in the status bar.

| Parameter | Type   | Required | Description                                                       |
|-----------|--------|----------|-------------------------------------------------------------------|
| `message` | string | yes      | The text to show.                                                 |
| `level`   | string | no       | `"info"` (default), `"warn"`, or `"error"` — controls the colour. |

```lua
ttt.notify("Formatted 3 files")
ttt.notify("aspell not found in PATH", "warn")
ttt.notify("Could not reach the server", "error")
```

Use `notify` for lightweight, non-blocking feedback (a linter's "binary missing"
hint, "added to dictionary", and so on). For anything the user must respond to,
use [`ttt.confirm`](#ttt-confirm) or [`ttt.show_info`](#ttt-show_info) instead.

### `ttt.open_drawer(config)`

Open a drawer panel anchored to the left or right side of the editor. Requires `panel.drawer` permission.

| Parameter   | Type     | Required | Description                                      |
|-------------|----------|----------|--------------------------------------------------|
| `width`     | number   | no       | Initial drawer width in columns. Default: `40`.  |
| `min_width` | number   | no       | Minimum resize width in columns. Default: `20`.  |
| `side`      | string   | no       | `"left"` or `"right"`. Default: `"right"`.       |
| `render`    | function | yes      | Render function. Receives a panel proxy object.  |

```lua
ttt.open_drawer({
  width = 50,
  min_width = 30,
  side = "right",
  render = function(panel)
    panel:label("Drawer content")
  end,
})
```

### `ttt.close_drawer()`

Close the current drawer panel. No arguments. Requires `panel.drawer` permission.

```lua
ttt.close_drawer()
```

### `ttt.open_tab(config)`

Open a custom editor tab with plugin-rendered content. Requires `panel.editor` permission.

| Parameter  | Type     | Required | Description                                       |
|------------|----------|----------|---------------------------------------------------|
| `title`    | string   | yes      | Tab title displayed in the editor tab bar.        |
| `render`   | function | yes      | Render function. Receives a panel proxy object.   |
| `on_event` | function | no       | Fallback handler for key/mouse events not consumed by widgets. |

```lua
ttt.open_tab({
  title = "Preview",
  render = function(panel)
    panel:label("Tab content here")
  end,
  on_event = function(ev) end,
})
```

### `ttt.close_tab(id)`

Close a plugin editor tab by its ID (string). Requires `panel.editor` permission.

```lua
ttt.close_tab("my-tab-id")
```

### `ttt.open_file(path, [line])`

Open a file in the editor, optionally jumping to a specific line.

| Parameter | Type   | Required | Description                                    |
|-----------|--------|----------|------------------------------------------------|
| `path`    | string | yes      | File path (absolute or relative to the workspace root). |
| `line`    | number | no       | 1-based line number to jump to after opening.  |

```lua
-- Open a file
ttt.open_file("src/main.go")

-- Open a file and jump to line 42
ttt.open_file("src/main.go", 42)
```

No special permission is required — any plugin can open files.

### `ttt.plugin_dir()`

Returns the absolute path of the plugin's install directory. Use it to read and write plugin-local files (cached data, state) with the `ttt.fs` API — the plugin directory is always a writable filesystem root for the plugin, alongside the workspace folders. No permission required to call it; reading or writing files there still requires `fs.read` / `fs.write`.

```lua
local ttt = require("ttt")
local fs = require("ttt.fs")

local state_file = ttt.plugin_dir() .. "/state.json"
local ok, err = fs.write(state_file, json.encode(state))
```

Note: the plugin directory is a git clone that `Plugins: Update` pulls into, and it is deleted on uninstall — keep only regenerable state there.

### `ttt.set_timeout(ms, callback)` / `ttt.set_interval(ms, callback)`

Schedule a callback to run later on the editor's main loop. `set_timeout` fires once after `ms` milliseconds; `set_interval` fires every `ms` milliseconds until cleared. Both return a numeric timer id. No permission required.

Callbacks run on the main thread (the same place render and event handlers run), so they can safely touch plugin state and call any `ttt` API — no need for `panel:redraw()` juggling as with raw goroutines. Because they share the UI thread, `set_interval` enforces a minimum interval of 50ms.

```lua
local ttt = require("ttt")

-- Refresh a data panel every 5 seconds
local timer = ttt.set_interval(5000, function()
  refresh_data()
  panel:redraw()
end)

-- Run something once, shortly after load
ttt.set_timeout(200, function()
  ttt.log("plugin warmed up")
end)
```

### `ttt.clear_timeout(id)` / `ttt.clear_interval(id)`

Cancel a pending timeout or a running interval by its id. The two are interchangeable (a single timer registry backs both). Cancelling an unknown or already-fired id is a no-op.

```lua
local id = ttt.set_interval(1000, poll)
-- later...
ttt.clear_interval(id)
```

All of a plugin's timers are automatically stopped when the plugin is disabled, reloaded, or uninstalled — you don't need to clear them on shutdown.

### `ttt.on_install(callback)`

Register a function that runs once when the plugin is installed (from the Plugins panel or **Plugins: Install from URL**). Use this for one-time setup like writing default settings. No permission required to register the hook; whatever the callback does is still permission-checked.

```lua
local ttt = require("ttt")
local settings = require("ttt.settings")

ttt.on_install(function()
  settings.set("formatters.go", "gofmt")
end)
```

### `ttt.on_uninstall(callback)`

Register a cleanup function that runs when the plugin is uninstalled. Use this to remove settings, files, or other state the plugin created during its lifetime. No permission required.

```lua
local ttt = require("ttt")
local settings = require("ttt.settings")

settings.set("formatters.go", "gofmt")

ttt.on_uninstall(function()
  settings.set("formatters.go", nil)
end)
```

### `ttt.markdown(text)`

Parse a markdown string into styled spans. Returns a table of lines, where each line is a table of spans with `text` and `style` fields. No permission required.

```lua
local lines = ttt.markdown("# Hello\nSome **bold** text")
-- lines[1] = { {text = "Hello", style = "bold"} }
-- lines[2] = { {text = "Some ", style = "default"}, {text = "bold", style = "bold"}, ... }
```

### Debug helpers

These drive and capture the editor for automated testing (see [Testing Plugins](/guides/plugin-testing/)). They are the Lua equivalents of the `--exec` script commands. No permission required.

| Function | Description |
|----------|-------------|
| `ttt.screenshot(path)` | Write the current screen (plain text) to a file. |
| `ttt.debug(path)` | Write the editor's full state as JSON to a file (cursor, panels, widget tree, OUTPUT log). |
| `ttt.click(x, y)` | Simulate a mouse click at screen coordinates. |
| `ttt.drag(x1, y1, x2, y2)` | Simulate a mouse drag between two points. |
| `ttt.quit()` | Exit the editor. |

### `ttt.json` Module

The `ttt.json` module provides JSON encoding and decoding. No permission required.

```lua
local json = require("ttt.json")
```

#### `json.encode(value)`

Encode a Lua table or value to a JSON string.

```lua
local str = json.encode({name = "test", value = 42})
-- '{"name":"test","value":42}'
```

#### `json.decode(str)`

Decode a JSON string to a Lua table. Returns `nil, error_string` on failure.

```lua
local tbl, err = json.decode('{"name":"test"}')
if tbl then
  -- tbl.name == "test"
else
  ttt.log("error", "JSON decode failed: " .. err)
end
```

## Widget API

The `render` function receives a **panel proxy** object. Call methods on it to build a declarative widget tree. Widgets are stacked vertically in the order they are declared.

The render function is called every time the panel needs to redraw. Widget state (selection, expanded nodes, input text, cursor position) is **automatically preserved** between render calls — you don't need to manage it yourself.

### Label

Display a line of text, optionally styled with a badge.

**Simple string form:**

```lua
panel:label("Status: OK")
```

**Table form (with style and badge):**

```lua
panel:label({ text = "Error occurred!", style = "danger" })
panel:label({ text = "Containers", style = "muted", badge = "5" })
panel:label({ text = "Hint: press Enter", style = "muted", padding_left = 1 })
```

| Field   | Type   | Required | Description                                       |
|---------|--------|----------|---------------------------------------------------|
| `text`  | string | yes      | Label text to display.                            |
| `style` | string | no       | Named style (see [Styles](#styles)). Default: `"default"`. |
| `badge` | string | no       | Badge text displayed after the label.             |
| `width` | number | no       | Fixed width in columns. Text is truncated or padded to fit. |
| `border` / `border_top` / `border_bottom` / `border_left` / `border_right` | boolean | no | Draw a border around the label (same semantics as [Box](#box)). |
| + [box model fields](#box-model) | | | Margin and padding. |

Labels are not focusable — they display text only and don't receive keyboard events.

### Title

Bold heading text with an optional right-aligned badge and dropdown menu.

**Simple string form:**

```lua
panel:title("Section Header")
```

**Table form:**

```lua
panel:title({ text = "CONTAINERS", margin_top = 1, margin_bottom = 1 })

panel:title({
  text = "CONTAINERS",
  badge = "3",
  menu = {
    { label = "Refresh", command = "refresh" },
    { separator = true },
    { label = "Prune", command = "prune" },
  },
  on_menu = function(command)
    if command == "refresh" then refresh() end
  end,
})
```

| Field     | Type     | Required | Description            |
|-----------|----------|----------|------------------------|
| `text`    | string   | yes      | Title text to display. |
| `badge`   | string   | no       | Right-aligned badge text, rendered muted. |
| `menu`    | table    | no       | Array of [menu entries](#menu-entry-format). Adds a dropdown button on the right edge. |
| `on_menu` | function | no       | Callback when a menu item is selected. Receives the command string. |
| `icon`    | string   | no       | Overrides the dropdown button icon (default `⋮`). |
| `padded`  | boolean  | no       | Adds horizontal padding around the dropdown button. |
| + [box model fields](#box-model) | | | Margin and padding. |

Titles are not keyboard-focusable; the dropdown menu button is operated with the mouse.

### Key-Value List

Display a list of key-value pairs, aligned in two columns.

```lua
panel:keyvalue({
  { key = "Status", value = "Running" },
  { key = "Image",  value = "nginx:latest" },
  { key = "Port",   value = "8080:80" },
})
```

The argument is an array of tables, each with:

| Field   | Type   | Required | Description   |
|---------|--------|----------|---------------|
| `key`   | string | yes      | Left column.  |
| `value` | string | yes      | Right column. |

The outer table also supports [box model fields](#box-model) for margin/padding.

Key-value lists are not focusable.

### Tree

Hierarchical list with expandable nodes, keyboard navigation (arrows, Enter), and mouse support.

```lua
panel:tree({
  items = {
    {
      id = "src",
      label = "src/",
      icon = "\xF0\x9F\x93\x81",
      expandable = true,
      expanded = true,
      children = {
        { id = "main.go", label = "main.go", icon = "\xF0\x9F\x93\x84" },
        { id = "util.go", label = "util.go", icon = "\xF0\x9F\x93\x84", muted = true },
      },
    },
    { id = "readme", label = "README.md", icon = "\xF0\x9F\x93\x84", badge = "modified" },
  },
  indent = 2,
  on_select = function(node)
    -- called when a node is activated (Enter key or double-click)
  end,
  on_expand = function(node)
    -- called when a node is expanded (use it to lazy-load children)
  end,
  on_command = function(command, node)
    -- called when a context menu command or inline action is triggered
  end,
  node_menu = {
    { label = "Open", command = "open" },
    { label = "Delete", command = "delete" },
  },
})
```

**Tree config fields:**

| Field        | Type     | Default | Description                                    |
|--------------|----------|---------|------------------------------------------------|
| `items`      | table    | `{}`    | Array of [tree node tables](#tree-node-format). |
| `indent`     | number   | `2`     | Number of spaces per nesting level.            |
| `on_select`  | function | nil     | Callback when a node is activated (Enter or double-click). Receives the node table. |
| `on_expand`  | function | nil     | Callback when a node is expanded (not fired on collapse — intended for lazy-loading children). Receives the node table. |
| `on_command`   | function | nil     | Callback when a context menu command or key command is selected. Receives `(command, node)`. |
| `node_menu`    | table    | nil     | Array of [menu entries](#menu-entry-format) for right-click context menu on nodes. |
| `key_commands` | table    | nil     | Map of single-char keys to command strings. When pressed, triggers `on_command(command, selected_node)`. |

**Keyboard navigation:** When focused, Up/Down arrows move selection, Enter activates `on_select`, Left/Right collapse/expand nodes. Shift+Enter opens the context menu on the selected node.

### List

Flat list — functionally identical to a tree but without nesting or expand/collapse behavior.

```lua
panel:list({
  items = {
    { id = "item1", label = "First item", icon = "o" },
    { id = "item2", label = "Second item", icon = "o", badge = "new" },
    { id = "item3", label = "Third item", muted = true },
  },
  on_select = function(node)
    -- called when an item is activated
  end,
  on_command = function(command, node)
    -- called when a context menu command is triggered
  end,
  node_menu = {
    { label = "Start", command = "start" },
    { label = "Stop", command = "stop" },
    { separator = true },
    { label = "Remove", command = "remove" },
  },
})
```

**List config fields:**

| Field        | Type     | Default | Description                                            |
|--------------|----------|---------|--------------------------------------------------------|
| `items`        | table    | `{}`    | Array of [tree node tables](#tree-node-format).        |
| `on_select`    | function | nil     | Callback when an item is activated. Receives the node table. |
| `on_command`   | function | nil     | Callback when a context menu command or key command is selected. Receives `(command, node)`. |
| `node_menu`    | table    | nil     | Array of [menu entries](#menu-entry-format) for the right-click context menu. |
| `key_commands` | table    | nil     | Map of single-char keys to command strings. When pressed, triggers `on_command(command, selected_node)`. |

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

| Field      | Type     | Required | Description                                    |
|------------|----------|----------|------------------------------------------------|
| `label`    | string   | yes      | Button text. Use `&` for an accelerator: `"&Save"` underlines S. |
| `on_click` | function | no       | Callback when the button is pressed.           |

### Input

Single-line text input with placeholder text and optional prefix.

```lua
panel:input({
  placeholder = "Type to search...",
  prefix = "# ",
  on_change = function(text)
    -- called on every keystroke with the current text
  end,
  on_submit = function(text)
    -- called when Enter is pressed with the current text
  end,
})
```

**Input config fields:**

| Field              | Type     | Default | Description                                    |
|--------------------|----------|---------|------------------------------------------------|
| `placeholder`      | string   | `""`    | Grayed-out hint text shown when input is empty.|
| `prefix`           | string   | `""`    | Non-editable text displayed before the input.  |
| `clear_on_submit`  | boolean  | `false` | When `true`, the input text is cleared after `on_submit` fires. |
| `on_change`        | function | nil     | Called after every text change. Receives the current text (string). |
| `on_submit`        | function | nil     | Called when Enter is pressed. Receives the current text (string). |

The input text and cursor position are automatically preserved across re-renders.

### VStack

Vertical stack layout container. Groups child widgets with an optional gap between them.

```lua
panel:vstack({
  gap = 1,
  render = function(p)
    p:label({ text = "Section Title", style = "muted" })
    p:button({ label = "&Action", on_click = function() end })
  end,
})
```

**VStack config fields:**

| Field    | Type     | Required | Description                                                |
|----------|----------|----------|------------------------------------------------------------|
| `render` | function | yes      | Builder function that receives a child panel proxy. Call widget methods on it to add children. |
| `gap`    | number   | no       | Vertical gap (in rows) between children. Default: `0`.     |

VStack is useful for grouping related widgets into sections. The child panel proxy supports all the same widget methods.

### HStack

Horizontal stack layout container. Lays out child widgets side by side. The first child grows to fill available space; remaining children get fixed width.

```lua
panel:hstack({
  gap = 1,
  render = function(p)
    p:label({ text = "Left (grows)", style = "default" })
    p:button({ label = "&Action", on_click = function() end })
  end,
})
```

**HStack config fields:**

| Field    | Type     | Required | Description                                                |
|----------|----------|----------|------------------------------------------------------------|
| `render` | function | yes      | Builder function that receives a child panel proxy. Call widget methods on it to add children. |
| `gap`    | number   | no       | Horizontal gap (in columns) between children. Default: `0`. |
| `height` | number   | no       | Fixed height in rows.                                       |

The first child in the hstack expands to fill remaining horizontal space. All subsequent children are given their natural/fixed width. This is useful for toolbar-style layouts with a label or spacer on the left and action buttons on the right.

### ScrollView

Scrollable container for content that may exceed the available height. Wraps children in a scroll view with mouse wheel scrolling and a scrollbar when content overflows.

```lua
panel:scrollview({
  render = function(p)
    for i = 1, 100 do
      p:label("Line " .. i)
    end
  end,
})
```

**ScrollView config fields:**

| Field    | Type     | Required | Description                                                |
|----------|----------|----------|------------------------------------------------------------|
| `render` | function | yes      | Builder function that receives a child panel proxy. Call widget methods on it to add children. |

Use `scrollview` when the content may exceed the panel height. The scroll position is preserved across re-renders. A scrollbar indicator appears on the right edge when content overflows.

### Box

Container with optional borders and fixed height. Wraps child widgets.

```lua
panel:box({
  border = true,
  render = function(p)
    p:list({
      items = items,
      on_select = function(node) end,
    })
  end,
})
```

**Box config fields:**

| Field           | Type     | Default | Description                                    |
|-----------------|----------|---------|------------------------------------------------|
| `render`        | function | yes     | Builder function that receives a child panel proxy. |
| `border`        | boolean  | false   | Draw a full border around the box.             |
| `border_top`    | boolean  | false   | Draw only the top border.                      |
| `border_bottom` | boolean  | false   | Draw only the bottom border.                   |
| `border_left`   | boolean  | false   | Draw only the left border.                     |
| `border_right`  | boolean  | false   | Draw only the right border.                    |
| `height`        | number   | 0       | Fixed height in rows. `0` means auto-size.     |

Individual `border_*` flags can be combined. If `border` is true, all four sides are drawn regardless of individual flags.

**Example with sections:**

```lua
panel:vstack({
  render = function(p)
    p:label({ text = "Containers", badge = "3", padding_left = 1 })
    p:box({
      border = true,
      render = function(bp)
        bp:list({ items = container_items })
      end,
    })
  end,
})
```

### Divider

Horizontal divider line. Renders a single-line separator across the panel width. Useful for visually separating sections.

```lua
panel:label("Section 1")
panel:divider()
panel:label("Section 2")
```

No configuration fields. The divider takes no arguments and is not focusable.

### Dropdown

A button that opens a context menu when clicked.

```lua
panel:dropdown({
  label = "Actions",
  entries = {
    { label = "Start", command = "start" },
    { label = "Stop", command = "stop" },
    { separator = true },
    { label = "Remove", command = "remove" },
  },
  on_menu = function(command)
    if command == "start" then ... end
  end,
})
```

**Dropdown config fields:**

| Field     | Type     | Required | Description                                                |
|-----------|----------|----------|------------------------------------------------------------|
| `label`   | string   | yes      | Button text displayed.                                     |
| `entries` | table    | yes      | Array of [menu entries](#menu-entry-format) for the popup. |
| `on_menu` | function | no       | Callback when a menu item is selected. Receives the command string. |

### Progress

Horizontal progress bar filling the panel width.

```lua
panel:progress({ value = 0.65 })
panel:progress({ value = 0.9, style = "success", char = "█" })
```

**Progress config fields:**

| Field   | Type   | Required | Description                                       |
|---------|--------|----------|---------------------------------------------------|
| `value` | number | yes      | Fill ratio between `0.0` and `1.0`.               |
| `style` | string | no       | Named style for the filled portion. Default: `"default"`. |
| `char`  | string | no       | Fill character (first rune used). Default: `▄`.   |
| + [box model fields](#box-model) | | | Margin and padding. |

Progress bars are not focusable.

### Table

Tabular data with column headers, row selection, keyboard navigation, and context menu support.

```lua
panel:table({
  columns = {
    { label = "Name" },
    { label = "Status", width = 10 },
    { label = "Port", width = 6, align = "right" },
  },
  rows = {
    { "web-app", "running", "8080" },
    { "database", "stopped", "5432" },
  },
  on_select = function(row_index)
    -- 1-based index into rows
  end,
  on_command = function(command, row_index)
    -- context menu command or key command on a row
  end,
  node_menu = {
    { label = "Restart", command = "restart" },
  },
  key_commands = { r = "restart" },
})
```

**Table config fields:**

| Field          | Type     | Required | Description                                                 |
|----------------|----------|----------|-------------------------------------------------------------|
| `columns`      | table    | yes      | Array of column tables: `{label, width, align}`. `width` is fixed columns (omit to auto-size); `align` is `"right"` or default left. |
| `rows`         | table    | yes      | Array of rows; each row is an array of cell strings.        |
| `on_select`    | function | no       | Callback when the selected row changes (arrow keys, click) or Enter is pressed. Receives the 1-based row index. |
| `on_command`   | function | no       | Callback for context menu or key commands. Receives `(command, row_index)`. |
| `node_menu`    | table    | no       | Array of [menu entries](#menu-entry-format) for the right-click context menu on rows. |
| `key_commands` | table    | no       | Map of single-char keys to command strings. Triggers `on_command(command, selected_row_index)`. |
| + [box model fields](#box-model) | | | Margin and padding. |

### Markdown

Render a markdown string as rich text with text selection and copy support. The content is wrapped in a scroll view automatically.

**Simple string form:**

```lua
panel:markdown("# Title\n\nSome **bold** and `code` text.")
```

**Table form:**

```lua
panel:markdown({ text = readme_content, margin_top = 1 })
```

| Field  | Type   | Required | Description               |
|--------|--------|----------|---------------------------|
| `text` | string | yes      | Markdown source to render. |
| + [box model fields](#box-model) | | | Margin and padding. |

Prose lines wrap at the `markdown.wrapWidth` setting (default 80). Headings, bold, italic, code spans, fenced code blocks, lists, and links are styled using the theme's styles. Text inside the widget can be selected with the mouse and copied.

### Tree Node Format

Used in `items` arrays for both `tree` and `list` widgets.

| Field        | Type    | Default | Description                                  |
|--------------|---------|---------|----------------------------------------------|
| `id`         | string  | `""`    | Unique identifier. **Required for state preservation.** |
| `label`      | string  | `""`    | Display text.                                |
| `icon`       | string  | `""`    | Icon string displayed before the label.      |
| `badge`      | string  | `""`    | Badge text displayed after the label.        |
| `muted`      | boolean | `false` | Render the label in a dimmed style.          |
| `expandable` | boolean | `false` | Show expand/collapse chevron indicator. Auto-set to `true` if `children` is non-empty. |
| `expanded`   | boolean | `false` | Initial expanded state (only used on first render — see [Reconciliation and State Preservation](#reconciliation-and-state-preservation)). |
| `children`   | table   | `{}`    | Array of child node tables (recursive).      |

**Callback argument:** `on_select` and `on_expand` callbacks receive a Lua table with the same fields as the node: `id`, `label`, `icon` (if non-empty), `badge` (if non-empty), `expanded`, `muted`, and `children` (if present). The `on_command` callback receives two arguments: `(command_string, node_table)`.

### Menu Entry Format

Used in `actions` (sidebar header menu), `node_menu` (tree/list context menu), and `entries` (dropdown).

| Field       | Type    | Required | Description                             |
|-------------|---------|----------|-----------------------------------------|
| `label`     | string  | yes*     | Display text of the menu item.          |
| `command`   | string  | yes*     | Command identifier passed to the callback. |
| `separator` | boolean | no       | If `true`, renders as a separator line instead of an item. When `true`, `label` and `command` are ignored. |

```lua
{
  { label = "Start", command = "start" },
  { label = "Stop", command = "stop" },
  { separator = true },
  { label = "Remove", command = "remove" },
}
```

### Box Model

Many widgets support margin and padding fields for spacing control. These fields can be included in the widget's configuration table.

| Field            | Type   | Default | Description               |
|------------------|--------|---------|---------------------------|
| `margin_top`     | number | `0`     | Space above the widget.   |
| `margin_bottom`  | number | `0`     | Space below the widget.   |
| `margin_left`    | number | `0`     | Space to the left.        |
| `margin_right`   | number | `0`     | Space to the right.       |
| `padding_top`    | number | `0`     | Internal top padding.     |
| `padding_bottom` | number | `0`     | Internal bottom padding.  |
| `padding_left`   | number | `0`     | Internal left padding.    |
| `padding_right`  | number | `0`     | Internal right padding.   |

All widgets support box model fields except `divider` (which takes no configuration).

```lua
panel:label({ text = "Indented label", padding_left = 2, margin_top = 1 })
panel:title({ text = "SECTION", margin_bottom = 1 })
panel:tree({ items = items, margin_top = 1 })
```

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

`panel:redraw()` posts an event to the editor's event loop. The render function will be called again on the next frame, and the updated state will be reflected. This is safe to call from any callback, including async callbacks.

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

| Parameter | Type            | Required | Description                                |
|-----------|-----------------|----------|--------------------------------------------|
| `x`       | number          | yes      | Column (0-based).                          |
| `y`       | number          | yes      | Row (0-based).                             |
| `char`    | string          | yes      | Single character to draw (first rune used).|
| `style`   | string or table | no       | Named style string or table with `style` field. |

### `panel:text(x, y, text, [style])`

Draw a text string starting at the given position. Text is clipped to the panel width.

```lua
panel:text(0, 0, "Hello World")                -- default style
panel:text(0, 1, "Error!", "danger")           -- named style as string
panel:text(0, 2, "Hint", { style = "muted" }) -- named style as table
```

| Parameter | Type            | Required | Description                                |
|-----------|-----------------|----------|--------------------------------------------|
| `x`       | number          | yes      | Starting column (0-based).                 |
| `y`       | number          | yes      | Row (0-based).                             |
| `text`    | string          | yes      | Text to draw.                              |
| `style`   | string or table | no       | Named style.                               |

### `panel:clear(x, y, w, h)`

Clear a rectangular region, filling it with spaces in the default style.

```lua
panel:clear(0, 0, 40, 10)
```

| Parameter | Type   | Required | Description         |
|-----------|--------|----------|---------------------|
| `x`       | number | yes      | Left column.        |
| `y`       | number | yes      | Top row.            |
| `w`       | number | yes      | Width in cells.     |
| `h`       | number | yes      | Height in cells.    |

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
    panel:cell(x, 2, "-", "border")
  end
  panel:text(0, 3, "Custom rendered content")
end
```

Note that raw cell coordinates are relative to the full panel surface, not offset by widget height. If your widgets take up 2 rows, drawing at y=0 will overlap them.

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

## Focus and Keyboard Navigation

The plugin panel manages its own focus system. When the plugin panel is focused:

- **Tab** / **Shift+Tab** cycles focus between focusable widgets (trees, lists, tables, inputs, buttons, dropdowns).
- **Arrow keys** navigate within the focused widget (e.g., up/down in a tree).
- **Enter** / **Space** activate the focused widget (select tree node, press button, submit input).
- **Shift+Enter** opens the context menu on the selected tree/list node (if `node_menu` is defined).
- Events not consumed by the focused widget fall through to the `on_event` handler.

Labels, titles, and key-value lists are not focusable. If your panel has only non-focusable widgets and raw cells, all events go directly to `on_event`.

## Event Handling

There are two levels of event handling:

### Widget Callbacks (Automatic)

Widget-specific events are routed automatically to the callbacks you provide:

| Widget   | Event              | Callback     | Argument(s)                         |
|----------|--------------------|--------------|-------------------------------------|
| Tree     | node activated     | `on_select`  | Node table (`{id, label, ...}`)     |
| Tree     | node expanded      | `on_expand`  | Node table                          |
| Tree     | context menu cmd   | `on_command` | `(command_string, node_table)`      |
| List     | item activated     | `on_select`  | Node table                          |
| List     | context menu cmd   | `on_command` | `(command_string, node_table)`      |
| Button   | pressed            | `on_click`   | (none)                              |
| Input    | text changed       | `on_change`  | Current text (string)               |
| Input    | Enter pressed      | `on_submit`  | Current text (string)               |
| Dropdown | menu item selected | `on_menu`    | Command string                      |
| Table    | row selected       | `on_select`  | Row index (number, 1-based)         |
| Table    | context menu cmd   | `on_command` | `(command_string, row_index)`       |
| Title    | menu item selected | `on_menu`    | Command string                      |

### Fallback Event Handler (`on_event`)

For key and mouse events not consumed by widgets, the `on_event` function receives a Lua table describing the event:

**Key event:**

```lua
{
  type = "key",
  key = "Enter",     -- key name: "Enter", "Tab", "Escape", "Up", "Down", "Left", "Right",
                     -- "Backspace", "Delete", "Home", "End", "PgUp", "PgDn",
                     -- or the character itself ("a", "A", "1", "/", etc.)
  rune = "a",        -- only set for printable character keys
  mod = "ctrl",      -- "ctrl", "alt", "shift", joined with "+" when combined
                     -- (e.g. "ctrl+shift"); nil when no modifier is held
}
```

**Mouse event:**

```lua
{
  type = "mouse",
  x = 5,             -- column (0-based, relative to panel)
  y = 10,            -- row (0-based, relative to panel)
  button = "left",   -- "left", "right", "middle", "wheel_up", "wheel_down", or "none"
}
```

**Example:**

```lua
on_event = function(event)
  if event.type == "key" then
    if event.key == "r" and not event.mod then
      -- refresh data
      reload_data()
      panel:redraw()
    end
  end
end
```

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
| `editor.insert(line, col, text)`                  | Insert text at position (1-based).             |
| `editor.replace(start_line, start_col, end_line, end_col, text)` | Replace a range with text (1-based). |
| `editor.set_cursor(line, col)`                    | Move the cursor (1-based).                     |
| `editor.set_selection(start_line, start_col, end_line, end_col)` | Set selection range (1-based).     |
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

## Filesystem API

The `ttt.fs` module provides file system access. Access is restricted to the workspace folders and the plugin's own directory — paths outside these roots (including via symlinks) return an "access denied" error.

```lua
local fs = require("ttt.fs")
```

### `fs.read(path)`

Read the contents of a file. Requires `fs.read` permission.

**Returns:** `content_string` on success, or `nil, error_string` on failure.

```lua
local content, err = fs.read("/tmp/config.json")
if content then
  -- process content
else
  ttt.log("error", "Failed to read: " .. err)
end
```

### `fs.write(path, content)`

Write content to a file. Requires `fs.write` permission.

**Returns:** `true` on success, or `nil, error_string` on failure.

```lua
local ok, err = fs.write("output.txt", "hello world")
if not ok then
  ttt.log("error", "Failed to write: " .. err)
end
```

### `fs.exists(path)`

Check if a file or directory exists. Requires `fs.read` permission.

**Returns:** boolean.

```lua
if fs.exists("/tmp/config.json") then
  -- file exists
end
```

### `fs.list(path)`

List entries in a directory. Requires `fs.read` permission.

**Returns:** array of `{name, is_dir}` tables on success, or `nil, error_string` on failure.

```lua
local entries, err = fs.list("/home/user/project")
if entries then
  for _, entry in ipairs(entries) do
    if entry.is_dir then
      ttt.log("dir: " .. entry.name)
    else
      ttt.log("file: " .. entry.name)
    end
  end
end
```

## System API

The `ttt.system` module provides command execution and environment variable access.

```lua
local sys = require("ttt.system")
```

### `sys.exec(binary, [args])`

Execute a command synchronously. Requires the `system.exec` permission with the binary listed in the allowlist.

| Parameter | Type   | Required | Description                              |
|-----------|--------|----------|------------------------------------------|
| `binary`  | string | yes      | Command to execute. Must be in the `system.exec` permission allowlist. |
| `args`    | table  | no       | Array of string arguments.               |

**Returns** a table:

| Field       | Type   | Description                                     |
|-------------|--------|-------------------------------------------------|
| `stdout`    | string | Standard output.                                |
| `stderr`    | string | Standard error.                                 |
| `exit_code` | number | Exit code (`0` for success, `-1` for exec errors). |

```lua
local result = sys.exec("git", {"status", "--porcelain"})
if result.exit_code == 0 then
  -- parse result.stdout
end
```

Arguments are validated before execution: shell-injection patterns and dangerous git flags (e.g. `--upload-pack`, `core.fsmonitor`) are rejected.

### `sys.exec_async(binary, [args], callback)`

Execute a command asynchronously. The callback receives the same result table and is called on the main thread when the command completes. The UI remains responsive during execution.

| Parameter  | Type     | Required | Description                                  |
|------------|----------|----------|----------------------------------------------|
| `binary`   | string   | yes      | Command to execute.                          |
| `args`     | table    | no       | Array of string arguments. May be omitted — the callback can be the second argument. |
| `callback` | function | yes      | Receives the result table when done.         |

```lua
sys.exec_async("docker", {"ps", "--format", "{{.Names}}"}, function(result)
  if result.exit_code == 0 then
    -- process result.stdout
  end
  panel:redraw()
end)

-- Without arguments:
sys.exec_async("uptime", function(result)
  panel:redraw()
end)
```

### `sys.env(name)`

Read an environment variable. Requires `system.env` permission.

**Returns:** the value as a string (empty string if not set).

```lua
local home = sys.env("HOME")
```

## Network API

The `ttt.net` module provides HTTP request capabilities. Requires the `network.http` permission.

Requests are restricted for safety: only `http` and `https` URLs are allowed; requests to localhost, loopback, link-local, and private network addresses are blocked; requests time out after 30 seconds; and response bodies are capped at 10 MB.

```lua
local net = require("ttt.net")
```

### Network host scoping

The `network.http` permission can be a boolean or an array of hostnames:

```json
"network.http": true                          // any host
"network.http": ["api.github.com", "cheat.sh"] // only these hosts
```

When an array is declared, `ttt.net` requests to any other host fail with a permission error in the response table (`status = 0`, `error` set) — the request never leaves the editor. Host matching is exact and case-insensitive (`api.github.com` does not cover `codeload.github.com`; list each host you need). The approval dialog shows the exact hosts, so users can see what a plugin talks to. Prefer an explicit list over `true` whenever your plugin knows its endpoints.

> This scopes the `ttt.net` client only. A plugin that also has `system.exec` for a network-capable binary (`curl`, `git`, `gh`, …) can still reach the network through that binary — `system.exec` is the permission that governs it.

### HTTP Response Table

All HTTP functions return (or pass to callbacks) a response table:

| Field     | Type   | Present      | Description                    |
|-----------|--------|--------------|--------------------------------|
| `status`  | number | always       | HTTP status code (`0` on error). |
| `body`    | string | always       | Response body (empty on error). |
| `headers` | table  | on success   | String-to-string map of response headers. |
| `error`   | string | on error only | Error message.                 |

### `net.get(url, [opts])`

Synchronous HTTP GET.

| Parameter | Type   | Required | Description                                |
|-----------|--------|----------|--------------------------------------------|
| `url`     | string | yes      | Request URL.                               |
| `opts`    | table  | no       | Options table with `headers` (string-to-string map). |

```lua
local resp = net.get("https://api.example.com/data")
if resp.error then
  ttt.log("error", resp.error)
elseif resp.status == 200 then
  -- process resp.body
end

-- With custom headers:
local resp = net.get("https://api.example.com/data", {
  headers = { ["Authorization"] = "Bearer token" },
})
```

### `net.post(url, [opts])`

Synchronous HTTP POST.

| Parameter | Type   | Required | Description                                |
|-----------|--------|----------|--------------------------------------------|
| `url`     | string | yes      | Request URL.                               |
| `opts`    | table  | no       | Options: `headers` (map) and `body` (string). |

```lua
local resp = net.post("https://api.example.com/data", {
  headers = { ["Content-Type"] = "application/json" },
  body = '{"key": "value"}',
})
```

### `net.get_async(url, [opts], callback)`

Asynchronous HTTP GET. The callback receives the response table. If `opts` is provided, callback is the third argument; otherwise it is the second.

```lua
-- Without options (callback is 2nd arg):
net.get_async("https://api.example.com/data", function(resp)
  if resp.status == 200 then
    -- process resp.body
    panel:redraw()
  end
end)

-- With options (callback is 3rd arg):
net.get_async("https://api.example.com/data", {
  headers = { ["Authorization"] = "Bearer token" },
}, function(resp)
  -- process response
  panel:redraw()
end)
```

### `net.post_async(url, opts, callback)`

Asynchronous HTTP POST. The `opts` table and callback are both required.

```lua
net.post_async("https://api.example.com/data", {
  headers = { ["Content-Type"] = "application/json" },
  body = '{"key": "value"}',
}, function(resp)
  if resp.status == 200 then
    -- process resp.body
    panel:redraw()
  end
end)
```

## Events API

The `ttt.events` module lets plugins react to editor lifecycle events.

```lua
local events = require("ttt.events")
```

### `events.on(event_name, callback)`

Register an event listener. Multiple listeners can be registered for the same event.

**File events** (require `events.file` permission):

| Event         | Description                 |
|---------------|-----------------------------|
| `file.open`   | A file was opened.          |
| `file.close`  | A file tab was closed.      |
| `file.save`   | A file was saved.           |

**Editor events** (require `events.editor` permission):

| Event           | Description                 |
|-----------------|-----------------------------|
| `editor.change` | Buffer content changed.     |
| `cursor.change` | Cursor position changed (line or column). |

All callbacks receive the file path of the affected file as their single argument.

**Example:**

```lua
local events = require("ttt.events")

events.on("file.save", function(path)
  -- auto-format or run linter after save
end)

events.on("file.open", function(path)
  -- load file-specific config
end)

events.on("cursor.change", function(path)
  -- update status display
  panel:redraw()
end)
```

Registering an unknown event name raises a Lua error.

### `ttt.settings` Module

The `ttt.settings` module provides scoped read/write access to editor settings. Requires the `settings` permission and `settings_keys` to specify which keys the plugin can access.

```lua
local settings = require("ttt.settings")
```

#### `settings.get(key)`

Read a setting value. Returns the value (string, number, boolean, or table depending on the setting), or `nil` if not set.

```lua
local formatter = settings.get("formatters.go")  -- "gofmt" or nil
local debounce = settings.get("autocomplete.debounce")  -- 150
```

#### `settings.set(key, value)`

Write a setting value. Persists to `settings.json` immediately. Any JSON-serializable value is accepted (string, number, boolean, table). Setting a key to `nil` deletes it.

```lua
settings.set("formatters.go", "gofmt")
settings.set("formatters.go", nil)  -- removes the key
```

**Key format:** Settings keys use dot notation (`group.name`, e.g. `formatters.go`, `lsp.servers.gopls`). Intermediate objects are created as needed when setting nested keys.

**Permission scoping:** The `settings_keys` manifest field controls which keys a plugin can access:

- `"formatters.*"` — wildcard, allows all keys under `formatters.`
- `"formatters.go"` — exact match, only allows `formatters.go`

```json
{
  "permissions": {
    "settings": true,
    "settings_keys": ["formatters.*"]
  }
}
```

**Example — formatter plugin:**

```lua
local settings = require("ttt.settings")
settings.set("formatters.go", "gofmt")
```

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
| `bold`     | Bold/emphasized text           |
| `italic`   | Italic text                    |
| `code`     | Code/monospace text            |
| `syntax_comment`   | Syntax: comments       |
| `syntax_string`    | Syntax: string literals |
| `syntax_keyword`   | Syntax: keywords       |
| `syntax_number`    | Syntax: numeric literals |
| `syntax_operator`  | Syntax: operators      |
| `syntax_function`  | Syntax: function names |
| `syntax_type`      | Syntax: type names     |
| `syntax_builtin`   | Syntax: built-in identifiers |
| `syntax_variable`  | Syntax: variables      |
| `syntax_tag`       | Syntax: HTML/XML tags  |
| `syntax_attribute` | Syntax: HTML/XML attributes |

Styles can be passed as a string or a table:

```lua
panel:text(0, 0, "OK", "success")                -- string form
panel:cell(0, 0, "X", { style = "danger" })       -- table form
panel:label({ text = "Note", style = "muted" })   -- in widget config
```

If an unrecognized style name is used, it falls back to `default`.

## Logging

Plugins can write messages to the **OUTPUT** bottom panel using `ttt.log()`. No permission is required.

```lua
local ttt = require("ttt")

-- Single argument defaults to "info" level
ttt.log("plugin loaded successfully")

-- Two arguments: level and message
ttt.log("warn", "config file not found, using defaults")
ttt.log("error", "failed to connect to service")
```

| Level   | Display                          |
|---------|----------------------------------|
| `info`  | Default text color               |
| `warn`  | Warning color (yellow)           |
| `error` | Danger color (red)               |

Messages appear in the OUTPUT panel with a timestamp and plugin name prefix: `15:04:05 [my-plugin] message`. Open the OUTPUT panel via the bottom panel tabs.

Plugin errors (init failures, render crashes, callback errors) are automatically routed to the OUTPUT panel as error-level messages, so you don't need to wrap everything in `pcall` for visibility.

Use **Plugins: Clear Output** from the command palette to clear the OUTPUT panel.

## Error Handling and Debugging

### Render Errors

If your `render` function throws a Lua error, the plugin panel displays the error message in red (`danger` style) instead of the normal content. The plugin stays loaded and will retry rendering on the next frame — so fixing the error in your Lua file and reloading will recover.

### Callback Errors

If a callback function (`on_select`, `on_click`, etc.) throws an error, it is caught and logged. The error is stored on the plugin's `LastError` field and displayed in the plugin list (accessible via the command palette: **Plugins: List Installed**).

### Debugging Tips

- **Start simple.** Begin with a single `panel:label()` to verify your plugin loads, then add complexity.
- **Use `ttt.log()` for debugging.** Write diagnostic messages to the OUTPUT panel: `ttt.log("value is: " .. tostring(x))`.
- **Check the OUTPUT panel.** Plugin errors automatically appear here. Open it from the bottom panel tabs.
- **Check the plugin list.** Open the command palette and run **Plugins: List Installed** to see status (enabled/disabled/error) and version for each plugin.
- **Use reload for fast iteration.** Run **Plugins: Reload** from the command palette to reload your plugin without restarting ttt.
- **Watch stderr.** Run ttt from a terminal to see error logs: `ttt 2>plugin-errors.log`.

## Permissions Reference

Permissions are declared in the manifest's `permissions` object. Boolean permissions are set to `true`; array permissions list specific values.

| Permission       | Type     | Gates                                             |
|------------------|----------|---------------------------------------------------|
| `panel.sidebar`  | boolean  | Register a sidebar panel.                         |
| `panel.bottom`   | boolean  | Register a bottom panel.                          |
| `panel.drawer`   | boolean  | Open drawer panels via `ttt.open_drawer()`.         |
| `panel.editor`   | boolean  | Open editor tabs via `ttt.open_tab()`.              |
| `commands`       | boolean  | Register commands in the command palette.         |
| `keybindings`    | boolean  | Bind keyboard shortcuts.                          |
| `editor.read`    | boolean  | Read the contents of editor buffers (`ttt.editor` read functions). |
| `editor.write`   | boolean  | Modify editor buffers (`ttt.editor` write functions). |
| `fs.read`        | boolean  | Read files and list directories (`ttt.fs` read functions). |
| `fs.write`       | boolean  | Write files to the file system (`ttt.fs.write`).  |
| `system.exec`    | string[] | Execute specific system commands. List each allowed binary name. |
| `system.env`     | boolean  | Read environment variables (`ttt.system.env`).    |
| `network.http`   | boolean \| string[] | Make outbound HTTP requests (`ttt.net`). `true` allows any host; an array (`["api.github.com"]`) restricts requests to those hostnames. See [Network host scoping](#network-host-scoping). |
| `events.file`    | boolean  | Listen for file events: `file.open`, `file.close`, `file.save`. |
| `events.editor`  | boolean  | Listen for editor events: `editor.change`, `cursor.change`. |
| `settings`       | boolean  | Read/write editor settings (`ttt.settings`).      |
| `settings_keys`  | string[] | Allowed settings key patterns. Use `group.*` for prefix match or exact key. |

**Example with multiple permissions:**

```json
{
  "name": "docker-manager",
  "description": "Manage Docker containers, images and volumes",
  "version": "0.1.0",
  "author": "eugenioenko",
  "entry": "init.lua",
  "permissions": {
    "panel.sidebar": true,
    "commands": true,
    "keybindings": true,
    "system.exec": ["docker"]
  }
}
```

**How permissions work at runtime:** If a permission is not granted, the corresponding functions are simply not available on the module. For example, without `editor.read`, the `ttt.editor` module will not have `buffer_text`, `cursor`, etc. Calling a function that requires a non-granted permission raises a Lua error.

## Lua Sandbox

Plugins run in a sandboxed Lua 5.1 environment. Only safe standard library modules are available:

| Module      | Available | Notes                        |
|-------------|-----------|------------------------------|
| `base`      | yes       | `type`, `tostring`, `tonumber`, `pairs`, `ipairs`, `select`, `unpack`, `error`, `pcall`, `xpcall`, `assert`, `rawequal`, `setmetatable`, `getmetatable` |
| `string`    | yes       | Full string library (`format`, `find`, `gsub`, `match`, `sub`, `rep`, `upper`, `lower`, `byte`, `char`, `len`, `reverse`) |
| `table`     | yes       | Full table library (`insert`, `remove`, `sort`, `concat`, `maxn`) |
| `math`      | yes       | Full math library (`floor`, `ceil`, `sqrt`, `sin`, `cos`, `random`, `pi`, etc.) |
| `coroutine` | yes       | Full coroutine library (`create`, `resume`, `yield`, `status`, `wrap`) |
| `os`        | partial   | Safe subset only: `os.time()`, `os.clock()`, `os.date()`. Dangerous functions (`execute`, `remove`, `rename`, `exit`) are not available. |
| `crypto`    | yes       | `crypto.random_bytes(n)` returns `n` cryptographically secure random bytes as a hex string (max 1024). `crypto.uuid()` returns a random UUID v4. |
| `io`        | **no**    | Blocked entirely. Use `ttt.fs` instead. |
| `debug`     | **no**    | Not loaded.                  |

### `os` Module (safe subset)

| Function     | Returns | Description |
|--------------|---------|-------------|
| `os.time()`  | number  | Current Unix timestamp (seconds since epoch). |
| `os.clock()` | number  | Seconds elapsed since the editor started (wall-clock, fractional). |
| `os.date([format])` | string | Format current time. Supports `%Y`, `%m`, `%d`, `%H`, `%M`, `%S`, `%c`, `%A`, `%a`, `%B`, `%b`, `%p`, `%I`, `%Z`, `%%`. Default format is `%c`. |

```lua
local now = os.time()           -- 1719532800
local elapsed = os.clock()      -- 12.345
local date = os.date("%Y-%m-%d") -- "2026-06-28"
```

### `crypto` Module

| Function | Returns | Description |
|----------|---------|-------------|
| `crypto.random_bytes(n)` | string | `n` cryptographically secure random bytes as a hex string. `n` must be 1–1024. |
| `crypto.uuid()` | string | Random UUID v4 (e.g. `"550e8400-e29b-41d4-a716-446655440000"`). |

```lua
local bytes = crypto.random_bytes(16)  -- "a1b2c3d4e5f6..."  (32 hex chars)
local id = crypto.uuid()               -- "550e8400-e29b-41d4-a716-446655440000"
```

**Removed globals:** `dofile`, `loadfile`, `load`, `loadstring`, `getfenv`, `setfenv`, `rawset`, `rawget`, `print` are removed from the sandbox.

**Module loading:** `require()` only allows these modules:

| Module         | Description                    |
|----------------|--------------------------------|
| `ttt`          | Core module: `register`, `log`, `confirm`, `show_info`, `notify`, `open_drawer`, `close_drawer`, `open_tab`, `close_tab`, `open_file`, `plugin_dir`, `set_timeout`, `set_interval`, `clear_timeout`, `clear_interval`, `on_install`, `on_uninstall`, `markdown`, `screenshot`, `debug`, `click`, `drag`, `quit` |
| `ttt.json`     | JSON encode/decode             |
| `ttt.editor`   | Editor buffer read/write       |
| `ttt.fs`       | Filesystem access              |
| `ttt.system`   | Command execution, env vars    |
| `ttt.net`      | HTTP requests                  |
| `ttt.events`   | Event listeners                |
| `ttt.settings` | Read/write editor settings     |

Any other module name passed to `require()` raises an error.

## Managing Plugins

### Plugins Panel

Open the **Plugins** tab in the sidebar (`Plugins: Show Panel` from the command palette) to see installed and available plugins. The panel has two sections:

- **INSTALLED** -- Lists all installed plugins with their status and version. Action buttons let you update or remove plugins.
- **AVAILABLE** -- Fetched from the vetted plugin registry. Click to install.

### Installing

**From the command palette:**

1. Run **Plugins: Install from URL**
2. Enter a git repository URL (e.g. `https://github.com/author/ttt-docker`)
3. The plugin is cloned into `~/.config/ttt/plugins/<name>/`
4. An approval dialog shows the requested permissions
5. Click **Allow** to load the plugin immediately

**Manual installation:**

Copy or clone the plugin directory into `~/.config/ttt/plugins/`:

```sh
git clone https://github.com/author/ttt-my-plugin ~/.config/ttt/plugins/my-plugin
```

Restart ttt. If the plugin is new, you'll see an approval dialog.

### Plugin List

Open the command palette and run **Plugins: List Installed** to see all installed plugins with their status and version.

### Updating

Run **Plugins: Update** from the command palette and select a plugin. This runs `git pull` in the plugin directory. If the updated manifest requests new permissions, the plugin is disabled and you'll be prompted to re-approve.

### Removing

Run **Plugins: Uninstall** from the command palette and select a plugin. This removes the plugin directory and its registry entry.

Manual removal:

```sh
rm -rf ~/.config/ttt/plugins/my-plugin
```

### Enabling / Disabling

Run **Plugins: Enable** or **Plugins: Disable** from the command palette to toggle a plugin without removing it. Disabled plugins retain their permissions in the registry.

## Examples

### File Tree Browser

A sidebar panel that shows a static file tree:

`plugin.ttt.json`:
```json
{
  "name": "file-tree",
  "entry": "init.lua",
  "permissions": { "panel.sidebar": true }
}
```

`init.lua`:
```lua
local ttt = require("ttt")

local tree = {
  {
    id = "src", label = "src/", icon = ">", expandable = true, expanded = true,
    children = {
      { id = "src/main.go", label = "main.go", icon = " " },
      { id = "src/server.go", label = "server.go", icon = " " },
      {
        id = "src/handlers", label = "handlers/", icon = ">", expandable = true,
        children = {
          { id = "src/handlers/auth.go", label = "auth.go", icon = " " },
          { id = "src/handlers/api.go", label = "api.go", icon = " " },
        },
      },
    },
  },
  { id = "go.mod", label = "go.mod", icon = " ", muted = true },
  { id = "README.md", label = "README.md", icon = " ", muted = true },
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

`plugin.ttt.json`:
```json
{
  "name": "search-panel",
  "entry": "init.lua",
  "permissions": { "panel.bottom": true }
}
```

`init.lua`:
```lua
local ttt = require("ttt")

local results = {}
local query = ""

local function do_search(text)
  query = text
  results = {}
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
        prefix = "# ",
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
            -- open the file at the matching line
          end,
        })
      elseif #query > 0 then
        panel:label({ text = "No results", style = "warning" })
      end
    end,
  },
})
```

### Context Menu with Tree

A sidebar panel demonstrating context menus on tree items:

```json
{
  "name": "context-menu-demo",
  "entry": "init.lua",
  "permissions": { "panel.sidebar": true }
}
```

```lua
local ttt = require("ttt")

local items = {
  { id = "item1", label = "First item" },
  { id = "item2", label = "Second item" },
  { id = "item3", label = "Third item" },
}

ttt.register({
  sidebar = {
    title = "Context Menu",
    render = function(panel)
      panel:list({
        items = items,
        on_select = function(node)
          ttt.log("Selected: " .. node.label)
        end,
        on_command = function(command, node)
          if command == "delete" then
            ttt.confirm("Delete '" .. node.label .. "'?", function()
              for i, item in ipairs(items) do
                if item.id == node.id then
                  table.remove(items, i)
                  break
                end
              end
              panel:redraw()
            end)
          elseif command == "rename" then
            ttt.log("Rename " .. node.label)
          end
        end,
        node_menu = {
          { label = "Rename", command = "rename" },
          { separator = true },
          { label = "Delete", command = "delete" },
        },
      })
    end,
  },
})
```

### Layout with VStack and Box

Demonstrates using VStack and Box for structured layouts:

```json
{
  "name": "layout-demo",
  "entry": "init.lua",
  "permissions": { "panel.sidebar": true }
}
```

```lua
local ttt = require("ttt")

local containers = {
  { id = "web", label = "web-app", badge = "nginx:latest" },
  { id = "db", label = "postgres", badge = "postgres:15" },
}

ttt.register({
  sidebar = {
    title = "Layout",
    render = function(panel)
      -- Section 1: Containers
      panel:vstack({
        render = function(p)
          p:label({ text = "Containers", badge = tostring(#containers), padding_left = 1 })
          p:box({
            border = true,
            render = function(bp)
              bp:list({
                items = containers,
                on_select = function(node)
                  ttt.log("Selected: " .. node.label)
                end,
              })
            end,
          })
        end,
      })

      -- Section 2: Info
      panel:vstack({
        render = function(p)
          p:label({ text = "Details", padding_left = 1, margin_top = 1 })
          p:box({
            border = true,
            render = function(bp)
              bp:keyvalue({
                { key = "Status", value = "Running" },
                { key = "Uptime", value = "2h 30m" },
              })
            end,
          })
        end,
      })
    end,
  },
})
```

### Raw Cell Drawing

A sidebar panel using only the raw cell API. (For an actual progress bar, prefer the built-in [`panel:progress`](#progress) widget — this example just demonstrates raw drawing.)

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

      local bar_w = w - 2
      local filled = math.floor(bar_w * progress)

      for x = 0, bar_w - 1 do
        if x < filled then
          panel:cell(x + 1, 2, "#", "success")
        else
          panel:cell(x + 1, 2, ".", "muted")
        end
      end

      local pct = math.floor(progress * 100) .. "%"
      panel:text(1, 4, pct, "default")
    end,
  },
})
```

### Git Status Panel (Editor + System + Events)

A sidebar panel that shows `git status` output and refreshes on file save:

`plugin.ttt.json`:
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

`init.lua`:
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

### Reference Plugin

For a complete, production-quality example, see the [Docker Manager plugin](https://github.com/eugenioenko/ttt-plugins/tree/main/docker-manager) in the community plugins repository. It demonstrates:

- Sidebar panel with multiple sections using VStack and Box
- Header actions menu with `actions` and `on_action`
- Async system commands with `exec_async`
- Context menus on list items with `node_menu` and `on_command`
- Confirmation dialogs with `ttt.confirm`
- Command palette commands and keybindings
- Reactive state management with `panel:redraw()`
- Fallback key handling with `on_event`
