# Bug Audit

## Critical

### 1. Nil pointer dereference in async callbacks after plugin destroy
- **File**: `internal/plugin/lua_system.go:95`, `internal/plugin/lua_net.go:117`, `internal/plugin/lua_net.go:152`
- **Description**: In `sysExecAsync`, `netGetAsync`, and `netPostAsync`, a goroutine runs the async operation and constructs a `resultFn` callback that accesses `p.State` (e.g. `p.State.NewTable()`) without a nil check. This callback is posted to the main event loop via `PostAsync` and executed later. If the plugin is destroyed between when the goroutine starts and when `resultFn` executes on the main thread, `p.State` is nil (set to nil by `Destroy()`), causing a nil pointer dereference panic.
- **Impact**: Application crash. Any plugin using `exec_async`, `get_async`, or `post_async` can trigger this if the user uninstalls, disables, reloads, or updates the plugin while an async operation is in flight. The go-test-runner, docker-manager, http-client, and todo-scanner plugins all use async operations.
- **Fix**: Add a `p.State == nil` guard at the top of each `resultFn` closure before accessing `p.State`. For example in `lua_system.go`:
  ```go
  resultFn := func() {
      if p.State == nil {
          return
      }
      tbl := p.State.NewTable()
      // ...
  }
  ```

### 2. Data race on plugin fields accessed from goroutines
- **File**: `internal/plugin/lua_system.go:110`, `internal/plugin/lua_net.go:122-124`, `internal/plugin/lua_net.go:157-159`
- **Description**: The async goroutines read `p.PostAsync` (and construct closures over `p.State`) without synchronization, while `Destroy()` on the main thread sets these fields to nil concurrently. In Go, concurrent read/write of interface values without synchronization is a data race and can cause undefined behavior, including partial reads of interface values.
- **Impact**: Undefined behavior / potential crash under concurrent plugin destruction and async completion.
- **Fix**: Either (a) add a mutex to `Plugin` that protects `State` and `PostAsync`, locking in both the goroutine and `Destroy()`, or (b) use the `PostAsync` callback to safely check `State` on the main thread only (i.e., always post the result and check `p.State` inside the event-loop callback).

## High

### 3. Plugin loses all app callbacks after Update (no-approval path)
- **File**: `internal/plugin/manager.go:306-317`, `internal/app/commands_plugin.go:209-226`
- **Description**: When `Manager.Update()` succeeds without requiring re-approval (permissions unchanged), it calls `p.Destroy()` then `p.Init()` on the same plugin object. `Destroy()` sets all app callbacks to nil (`RequestRedraw`, `PostAsync`, `Log`, `ShowContextMenu`, `ShowConfirmDialog`, `OpenDrawer`, `CloseDrawer`, `OpenTab`, `CloseTab`). `Init()` only sets up the Lua state and runs the plugin script; it does NOT restore any app callbacks. The caller `handlePluginUpdateResult` only calls `pluginsPanel.Refresh()` -- it does NOT call `wirePlugin()`. The `Editor`, `Filesystem`, `System`, and `Network` API interfaces survive (not cleared by `Destroy`), but all UI integration callbacks are nil.
- **Impact**: After updating a plugin (that doesn't need re-approval), the plugin silently loses: logging (`ttt.log` does nothing), redraw requests, context menus, confirm dialogs, drawer/tab operations, and all async result delivery. The plugin appears to work but UI interactions fail silently.
- **Fix**: In `handlePluginUpdateResult`, when `result.plugin == nil && result.err == nil` (the no-approval update path), look up the plugin by name and call `wirePlugin(p)` to re-establish all callbacks. Also need to remove and re-add sidebar/bottom panels in the UI.

### 4. Plugin not wired after SetEnabled(true)
- **File**: `internal/plugin/manager.go:323-364`, `cmd/ttt/main.go:207-211`
- **Description**: When `Manager.SetEnabled(name, true)` is called, it creates a new `Plugin` object, calls `Init()`, appends to `m.plugins`, and calls `collectRegistrations()`. However, the new plugin has nil values for: `RequestRedraw`, `PostAsync`, `Log`, `ShowContextMenu`, `ShowConfirmDialog`, `ShowInfoDialog`, `OpenDrawer`, `CloseDrawer`, `OpenTab`, `CloseTab`, `Editor`, `Filesystem`, `System`, `Network`, and `Borders`. The caller in `main.go` (the `OnToggle` handler) only calls `pluginsPanel.Refresh()` -- it does NOT call `wirePlugin()` or any of the `SetEditorAPI`/`SetFilesystemAPI`/etc methods.
- **Impact**: Re-enabling a previously disabled plugin creates a non-functional plugin. All API calls from Lua return empty/nil results. All UI callbacks do nothing. The plugin panel may render (if it only uses static widgets) but any interactive features fail silently.
- **Fix**: After `SetEnabled(name, true)`, the caller should call `wirePlugin(p)` on the newly enabled plugin to set up all app-level callbacks and API interfaces. Requires returning the new plugin from `SetEnabled` or looking it up afterward.

### 5. Duplicate panel registrations accumulate after Update
- **File**: `internal/plugin/manager.go:316`
- **Description**: In `Manager.Update()` (no-approval path, line 316), `collectRegistrations(p)` appends new `SidebarRegistration`/`BottomRegistration` entries to the Manager's panel lists without removing the old entries for the same plugin. Each update adds another set of registrations. The old registrations reference stale `PluginPanelWidget` objects whose render functions were from the previous Lua state.
- **Impact**: Manager's `SidebarPanels`/`BottomPanels` lists grow unboundedly with stale entries on each update. While the stale entries don't cause immediate crashes (old render funcs are nil, handled gracefully), they waste memory and could cause confusion if iterated over by other code (e.g., `wirePlugin` iterates `SidebarPanels` to find matching registrations).
- **Fix**: Before calling `collectRegistrations(p)` in `Update()`, remove existing entries for this plugin from `m.SidebarPanels` and `m.BottomPanels`, similar to how `Reload()` does it (lines 409-418).

## Medium

### 6. Lua `event.mod == nil` check never matches (on_event 'r' shortcut broken)
- **File**: `internal/plugin/event_convert.go:33-41`, `plugins/go-test-runner/plugin.lua:486`, `plugins/docker-manager/init.lua:367`, `plugins/todo-scanner/plugin.lua:244`
- **Description**: In `keyEventToLua`, the `mod` field is always set to a Lua string: `""` when no modifier is pressed, `"ctrl"`, `"alt"`, or `"shift"` otherwise. Three Lua plugins check `event.mod == nil` to detect "no modifier pressed". In Lua, an empty string `""` is NOT `nil`, so this comparison is always false. The condition `event.key == "r" and event.mod == nil` never evaluates to true.
- **Impact**: The 'r' keyboard shortcut for refreshing/rescanning within the sidebar/bottom panels of go-test-runner, docker-manager, and todo-scanner plugins does not work. Users must use the command palette instead.
- **Fix**: Either (a) change the Lua plugins to check `event.mod == ""` instead of `event.mod == nil`, or (b) change `event_convert.go` to not set the `mod` field when no modifier is present (so accessing it returns `nil` in Lua), or (c) change the Go code to set `lua.LNil` when mod is empty: `if mod == "" { L.SetField(tbl, "mod", lua.LNil) }`.

### 7. HTTP client plugin calls editor.insert() with wrong argument count
- **File**: `plugins/http-client/plugin.lua:153`
- **Description**: The plugin calls `editor.insert(text)` with a single string argument (the HTTP response body). The Go binding `editorInsert` expects three arguments: `(line, col, text)`, where line and col are numbers. `L.CheckNumber(1)` is called on the string argument, which fails because a string is not a number.
- **Impact**: Clicking the "Insert to editor" button in the HTTP client plugin always fails with a Lua type error. The error is caught by `Protect: true` and logged, but the insert never happens.
- **Fix**: Change line 153 to pass cursor position: `local pos = editor.cursor(); editor.insert(pos.line, pos.col, text)` (matching how `color-picker/plugin.lua:110` does it).

### 8. Nested widget reconciliation ignores child type mismatches
- **File**: `internal/plugin/widget_builder.go:127-187`
- **Description**: When `updateWidget` reconciles children of container widgets (VStack, HStack, Box, ScrollView), it matches children by index only: `if i < len(vs.Children)`. It does NOT check whether the existing child widget's type matches the descriptor's `Kind`. If the types differ, the type assertion in `updateWidget` silently fails (e.g., `if lw, ok := w.(*widgets.LabelWidget); ok` is false for an InputWidget), and the old widget is kept as-is in the new children list: `children[i] = vs.Children[i]`.
- **Impact**: If a Lua plugin conditionally renders different widget types within a container (e.g., showing a label when loading, then an input when loaded), the old widget type persists after the condition changes. The user sees stale widgets of the wrong type until the root-level reconciliation creates a new container. This primarily affects dynamic UIs within `vstack`, `hstack`, `box`, and `scrollview` containers.
- **Fix**: Check the widget kind before attempting an update. If the kinds don't match, create a new widget instead of reusing the old one:
  ```go
  if i < len(vs.Children) && widgetMatchesKind(vs.Children[i], cd.Kind) {
      updateWidget(vs.Children[i], cd, p)
      children[i] = vs.Children[i]
  } else {
      children[i] = createWidget(cd, p)
  }
  ```

### 9. keyEventToLua drops combined modifiers
- **File**: `internal/plugin/event_convert.go:33-41`
- **Description**: The modifier detection uses `else if` chains, so only the first matching modifier is reported. If a key event has both Ctrl and Shift (e.g., Ctrl+Shift+A), only `"ctrl"` is reported. The shift modifier is lost.
- **Impact**: Lua plugins cannot distinguish between `Ctrl+Key` and `Ctrl+Shift+Key` events. If a plugin needs to handle shifted control combinations, the modifier information is incomplete. In practice this is partially mitigated by the CLAUDE.md note that ctrl+shift combos are unreliable in terminals, but the event data is still incorrect.
- **Fix**: Build a composite modifier string or return a table of active modifiers. For example: `mod = "ctrl+shift"` or `{ ctrl = true, shift = true }`.

### 10. ScrollViewWidget WheelDown doesn't clamp scrollY
- **File**: `internal/widgets/scrollview.go:143-146`
- **Description**: When handling `WheelDown` mouse events, `scrollY` is incremented by 3 without any upper-bound clamping. While `clamp()` is called during `Render()` which corrects the value before drawing, between the event and the next render, `scrollY` can exceed `contentH - viewH`. If any code reads `scrollY` between event handling and rendering (e.g., `EnsureVisible`), it could compute incorrect visibility offsets.
- **Impact**: Minor visual glitch potential. The clamp in `Render` prevents out-of-bounds drawing, but intermediate state is inconsistent.
- **Fix**: Add clamping after incrementing:
  ```go
  sv.scrollY += 3
  // clamp will be done in Render, but for consistency:
  // max := contentH - viewH; if sv.scrollY > max { sv.scrollY = max }
  return EventConsumed
  ```
