# API Consistency & Test Coverage Audit

## API Inconsistencies

### 1. Error handling pattern mismatch across fs module

- **Location**: `internal/plugin/lua_fs.go:31-65`
- **Issue**: `fs.read()` and `fs.list()` return `nil, error_string` on failure (Go/Lua multi-return convention), but `fs.write()` returns just `error_string` on failure and zero values on success. This is a different contract within the same module.
- **Expected**: All fallible functions in the same module should use the same error return convention.
- **Fix**: Change `fs.write()` to return `nil, error_string` on failure, or `true` on success and `nil, error_string` on failure. This would be a breaking API change, so document it if applied.

### 2. Error handling pattern mismatch across modules

- **Location**: `internal/plugin/lua_fs.go`, `internal/plugin/lua_system.go`, `internal/plugin/lua_net.go`, `internal/plugin/lua_editor.go`
- **Issue**: Each module reports errors differently:
  - **fs**: `nil, error_string` (multi-return) for reads; single `error_string` for writes
  - **system.exec**: Always returns a table with `exit_code=-1` and `stderr=error` on failure (never nil/error)
  - **net.get/post**: Always returns a table with `status=0` and `error=error_msg` on failure (never nil/error)
  - **editor**: All read functions silently return empty string/empty table when editor is nil; all write functions silently no-op
- **Expected**: At minimum, document the error handling pattern for each module. Ideally, converge on one or two patterns (e.g., "read operations return `nil, err`; write operations return `nil, err`; batch result operations return a result table with an `error` field").
- **Fix**: This is a design decision. The current system module and network module patterns (table with error field) are arguably better for async callbacks. The fs module pattern (multi-return) is more idiomatic Lua. Document which pattern to use where and why.

### 3. "API not available" error handling is inconsistent

- **Location**: `internal/plugin/lua_fs.go:33-35`, `internal/plugin/lua_system.go:33-35`, `internal/plugin/lua_net.go:62-64`, `internal/plugin/lua_editor.go:43-47`
- **Issue**: When the backing Go API interface is nil (e.g., `p.Editor == nil`):
  - **fs module**: Returns `nil, "filesystem API not available"` (multi-return error)
  - **system module**: Calls `L.ArgError()` which raises a Lua error (aborts execution)
  - **network module**: Calls `L.ArgError()` which raises a Lua error
  - **editor module**: Returns empty string/empty table silently (no error at all)
- **Expected**: Same behavior when the API interface is nil across all modules.
- **Fix**: Pick one approach. Raising a Lua error (system/net pattern) is safest since a nil API indicates a bug in the host, not a user error. Apply it consistently.

### 4. `sys.exec_async` args parameter is required but `sys.exec` args is optional

- **Location**: `internal/plugin/lua_system.go:44-48` (exec) vs `internal/plugin/lua_system.go:82-86` (exec_async)
- **Issue**: `sys.exec("binary")` works without args (args default to nil). `sys.exec_async("binary", callback)` will fail because it tries to interpret the callback function as the args table. The docs say "pass `{}` if none" for `exec_async`, but `exec` silently accepts a missing args parameter.
- **Expected**: Both should handle missing args the same way.
- **Fix**: Either make `exec` also require args (breaking change), or make `exec_async` detect whether arg 2 is a function or table (like `net.get_async` does). The latter is more user-friendly.

### 5. `net.get_async` callback position is flexible, but `net.post_async` is not

- **Location**: `internal/plugin/lua_net.go:99-128` (get_async) vs `internal/plugin/lua_net.go:131-163` (post_async)
- **Issue**: `net.get_async(url, callback)` works (callback at position 2). `net.get_async(url, opts, callback)` also works (callback at position 3). But `net.post_async` always requires `opts` as position 2 and `callback` as position 3.
- **Expected**: This is documented correctly but worth noting as a potential surprise.
- **Fix**: Already documented. Low priority. `post_async` requiring opts makes sense since you usually need to send headers/body with POST.

### 6. Box model support inconsistency between code and documentation

- **Location**: `internal/plugin/lua_panel.go:572` (box), `internal/plugin/lua_panel.go:404-422` (parseBoxModel)
- **Issue**: The `box` widget calls `parseBoxModel()` (line 572) and thus supports box model fields, but PLUGINS.md states "Widgets that support box model: `label`, `title`, `keyvalue`" -- omitting `box`. CLAUDE.md is even more outdated, saying only `label` and `title` support box model.
- **Expected**: Documentation lists all widgets that actually support box model.
- **Fix**: Update PLUGINS.md to add `box` to the box model support list. Update CLAUDE.md to add `keyvalue` and `box`.

### 7. Divider widget has dead `applyBoxModel` call

- **Location**: `internal/plugin/widget_builder.go:274-278` (createDividerWidget) and `internal/plugin/lua_panel.go:514-523` (panelDividerWidget)
- **Issue**: `createDividerWidget()` calls `applyBoxModel(&dw.Box, desc)`, but `panelDividerWidget()` never parses box model fields from Lua input (the divider takes no arguments). The box model values will always be zero. This is dead code.
- **Expected**: Either remove the `applyBoxModel` call, or accept optional box model arguments on divider.
- **Fix**: Remove the `applyBoxModel` call from `createDividerWidget` since divider is documented as taking no arguments. Or, if spacing around dividers is desired, make divider accept an optional table with box model fields.

## Documentation Gaps

### 1. `ttt.open_drawer()`, `ttt.close_drawer()` are undocumented

- **Location**: `internal/plugin/sandbox.go:224-258`
- **Issue**: The `open_drawer` and `close_drawer` functions are implemented in the `ttt` module but not documented in `docs/PLUGINS.md`. They require the `panel.drawer` permission. Parameters: `open_drawer({render, width, min_width})`.
- **Fix**: Add a "Drawer API" section to PLUGINS.md, or if these are intentionally hidden/experimental, mark them with a code comment.

### 2. `ttt.open_tab()`, `ttt.close_tab()` are undocumented

- **Location**: `internal/plugin/sandbox.go:260-290`
- **Issue**: The `open_tab` and `close_tab` functions are implemented but not documented in `docs/PLUGINS.md`. They require the `panel.editor` permission. Parameters: `open_tab({title, render, on_event})`, `close_tab(id)`.
- **Fix**: Add an "Editor Tab API" section to PLUGINS.md.

### 3. `ttt.markdown()` is undocumented

- **Location**: `internal/plugin/sandbox.go:293-309`
- **Issue**: The `markdown` function renders markdown text into styled spans but is not documented. It takes a string and returns a table of lines, each containing an array of `{text, style}` spans.
- **Fix**: Add documentation in the "ttt Module Functions" section of PLUGINS.md.

### 4. `key_commands` feature on tree/list widgets is undocumented

- **Location**: `internal/plugin/lua_panel.go:318-319` (tree), `internal/plugin/lua_panel.go:348-349` (list)
- **Issue**: Both tree and list widgets parse a `key_commands` field from the config table. This maps single-character keys to command strings, routing them through `on_command`. This feature is not mentioned in PLUGINS.md.
- **Fix**: Add `key_commands` to the tree and list widget documentation tables.

### 5. Extra styles available but undocumented

- **Location**: `internal/plugin/lua_panel.go:24-37`
- **Issue**: The `styleMap` includes styles not listed in the Styles section of PLUGINS.md: `bold`, `code`, `syntax_comment`, `syntax_string`, `syntax_keyword`, `syntax_number`, `syntax_operator`, `syntax_function`, `syntax_type`, `syntax_builtin`, `syntax_variable`, `syntax_tag`, `syntax_attribute`. These are available to plugins but not documented.
- **Fix**: Add these to the Styles table in PLUGINS.md, or explicitly document which are public API vs internal.

### 6. HStack `height` field is undocumented

- **Location**: `internal/plugin/lua_panel.go:506-508`
- **Issue**: `panelHStackWidget` parses a `height` field to set `desc.FixedHeight`, but the HStack documentation in PLUGINS.md only lists `render` and `gap` as config fields.
- **Fix**: Add `height` to the HStack config fields table.

### 7. Label `width` field is undocumented

- **Location**: `internal/plugin/lua_panel.go:225-227`
- **Issue**: Labels parse a `width` field to set `desc.FixedWidth`, but this is not documented in PLUGINS.md.
- **Fix**: Add `width` to the Label config fields table.

### 8. `panel.editor` and `panel.drawer` permissions mentioned in code but not fully documented

- **Location**: `internal/plugin/permissions.go:12-13`
- **Issue**: The Permissions Reference in PLUGINS.md mentions `panel.drawer` as "Reserved for future use" and `panel.editor` is listed. However, the actual APIs gated by these permissions (`open_drawer`, `open_tab`) are not documented. A plugin author wouldn't know what these permissions enable.
- **Fix**: Tie each permission to its corresponding API functions in the Permissions Reference.

### 9. CLAUDE.md widget API section is outdated

- **Location**: CLAUDE.md "Plugin Widget API" section
- **Issue**: CLAUDE.md says box model is "Currently supported on `label` and `title` widgets only" -- missing `keyvalue` and `box`. Also missing the `keyvalue`, `hstack`, `scrollview`, and `divider` widgets from the method table, and missing `key_commands` from tree/list.
- **Expected**: CLAUDE.md should reflect all currently implemented widgets and features.
- **Fix**: Update the CLAUDE.md Plugin Widget API section to match PLUGINS.md, then update PLUGINS.md to match the code.

## Test Coverage Gaps

### 1. No tests for `ttt.open_drawer`, `ttt.close_drawer`, `ttt.open_tab`, `ttt.close_tab`

- **Untested**: The drawer and tab APIs in `sandbox.go` lines 224-290 have no unit or e2e tests.
- **Risk**: Permission checks, parameter parsing, and callback routing could break silently.
- **Priority**: Medium -- these are newer APIs that may not be widely used yet, but they touch permissions and callback wiring.

### 2. No tests for `ttt.show_info` and `ttt.confirm`

- **Untested**: `sandbox.go` lines 187-222 define `show_info` and `confirm` with dialog callbacks. No unit tests verify parameter parsing or callback invocation.
- **Risk**: Dialog callbacks could fail to fire or pass wrong arguments.
- **Priority**: Medium

### 3. No tests for `ttt.markdown`

- **Untested**: `sandbox.go` lines 293-309 -- the markdown rendering function.
- **Risk**: Return format (nested tables of spans) could change without detection.
- **Priority**: Low -- depends on the underlying `markdown.Render()` which may have its own tests.

### 4. No unit tests for `lua_panel.go` widget descriptor building

- **Untested**: The individual widget-building functions (`panelLabelWidget`, `panelTreeWidget`, `panelListWidget`, etc.) are not unit-tested. The `panel_widget_test.go` tests raw cell API and render errors but does not test widget descriptor output.
- **Risk**: Regressions in field parsing (e.g., adding a new field but forgetting to parse it, or box model not being applied).
- **Priority**: High -- this is the primary plugin authoring surface. The `widget_builder_test.go` tests reconciliation but assumes correct descriptors as input.

### 5. No tests for `sys.exec_async` and `net.*_async` functions

- **Untested**: All async functions (`sysExecAsync`, `netGetAsync`, `netPostAsync`) involve goroutines and `PostAsync` callbacks. No tests exercise these paths.
- **Risk**: Goroutine panics, race conditions, or callback not being invoked. The `PostAsync` callback mechanism is critical for async plugin operation.
- **Priority**: High -- async operations are heavily used in real plugins (e.g., the Docker manager). Failures would be hard to diagnose.

### 6. No tests for `manager.go` (Install, Uninstall, Update, Reload, SetEnabled)

- **Untested**: The `Manager` type's lifecycle management methods have no tests. These shell out to `git clone`/`git pull` and manage the plugin registry.
- **Risk**: Plugin installation could fail silently, registry corruption, stale panel registrations after uninstall/reload.
- **Priority**: Medium -- these touch the filesystem and git, making them harder to test but also more failure-prone.

### 7. Widget files in `internal/widgets/` lacking tests

- **Untested**: The following widget implementation files have no dedicated test files:
  - `box.go` -- box container (used heavily by plugins)
  - `button.go` -- button widget
  - `dropdown.go` -- dropdown menu widget
  - `hstack.go` -- horizontal stack layout
  - `scrollview.go` -- scrollable container
  - `tree.go` -- tree widget (critical for plugin UI)
  - `vstack.go` -- vertical stack layout
  - `list_widget.go` -- list widget
- **Risk**: Layout calculations, event handling, focus management, and rendering in these widgets could regress. Tree widget is especially critical since it's the most complex interactive widget.
- **Priority**: High for tree.go and scrollview.go (complex interactive widgets); Medium for box.go, hstack.go, vstack.go (layout widgets); Low for button.go, dropdown.go (simpler widgets).

### 8. No e2e tests for bottom panel plugins

- **Untested**: The e2e test `plugin_panel_test.go` only tests sidebar panels. No e2e test verifies bottom panel plugin rendering, event routing, or tab switching.
- **Risk**: Bottom panel plugin rendering or event routing could break without detection.
- **Priority**: Medium

### 9. No e2e tests for plugin commands and keybindings

- **Untested**: While `lua_commands_test.go` tests registration at the unit level, there are no e2e tests verifying that registered plugin commands appear in the command palette or that plugin keybindings trigger their handlers.
- **Risk**: Command/keybinding wiring between the plugin system and the app could break.
- **Priority**: Medium

### 10. No tests for event dispatch through Manager.DispatchEvent

- **Untested**: `Manager.DispatchEvent()` converts Go values to Lua values and dispatches to all plugins. The type conversion (string/int/bool to lua.LValue) and multi-plugin dispatch are untested.
- **Risk**: Type conversion bugs (e.g., missing a type case) or dispatch ordering issues.
- **Priority**: Low -- the individual `Plugin.DispatchEvent` is tested; the manager layer adds type conversion.

### 11. No functional (blackbox) tests for any plugin features

- **Untested**: The `tests/functional/` directory has no tests exercising the plugin system with the real binary.
- **Risk**: Integration issues between the plugin system and the rest of the editor (sidebar/bottom panel wiring, command palette integration, theme interaction) would not be caught.
- **Priority**: Low for now -- e2e tests with `testHarness` provide reasonable coverage. Functional tests would be valuable once the plugin system stabilizes.

## Recommendations

### Immediate (before merging)

1. **Add unit tests for widget descriptor building** (`lua_panel.go`). Test that `panelLabelWidget`, `panelTreeWidget`, etc. produce correct `WidgetDesc` structs from Lua table input. This is the highest-risk gap since it's the primary plugin authoring surface.

2. **Add unit tests for async functions** (`sys.exec_async`, `net.get_async`, `net.post_async`). Use synchronous mocks with channel-based `PostAsync` to verify callback invocation without actual goroutine timing issues.

3. **Document `open_drawer`, `close_drawer`, `open_tab`, `close_tab`, `markdown`** in PLUGINS.md, or mark them as internal/experimental with code comments.

4. **Document `key_commands`** in the tree/list widget sections of PLUGINS.md.

5. **Fix the box model documentation** in both PLUGINS.md and CLAUDE.md to include `box` (and `keyvalue` in CLAUDE.md).

### Short-term

6. **Add tests for `tree.go` in `internal/widgets/`**. The tree widget is the most complex interactive widget and is used in most plugin UIs. Rendering, expand/collapse, selection, context menus, and keyboard navigation should all be tested.

7. **Add e2e tests for bottom panel plugins** and plugin command registration.

8. **Standardize error handling** across modules. At minimum, document the error handling contract for each module. Ideally, make `fs.write()` return `nil, error_string` to match `fs.read()` and `fs.list()`.

### Long-term

9. **Add functional tests** for the plugin system once it stabilizes.

10. **Consider making `sys.exec_async` detect callback position** like `net.get_async` does, to allow omitting the empty args table.

11. **Add unit tests for `Manager` lifecycle operations** (install, uninstall, update, reload). These could use a temporary directory structure and skip git operations.

12. **Remove dead `applyBoxModel` call** from `createDividerWidget`, or extend divider to accept box model arguments.
