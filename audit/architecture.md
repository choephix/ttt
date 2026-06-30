# Architecture Audit: Plugin System

## Violations (breaks stated architecture rules)

### V1: Plugin package directly imports tcell

- **File**: `internal/plugin/event_convert.go:4`
- **File**: `internal/plugin/panel_widget.go:6`
- **File**: `internal/plugin/widget_builder.go:6`
- **Issue**: The plugin package imports `github.com/gdamore/tcell/v2` directly. The project's architecture rule states that only `internal/term/` should import tcell. The `internal/widgets/` package already violates this rule too (every widget file imports tcell for `HandleEvent`), but the plugin package adds a third violator.
- **Impact**: The plugin system cannot be tested without pulling in the tcell dependency. It also creates a transitive coupling path: any change to tcell's event types ripples through the plugin package.
- **Fix**: For `event_convert.go`, define an abstract event type in `internal/term/` or `internal/widgets/` (e.g., `KeyEvent`, `MouseEvent` interfaces) and convert at the boundary in `internal/app/`. For `panel_widget.go` and `widget_builder.go`, the tcell import is only needed because `HandleEvent(tcell.Event)` and the `OnKey` callback use tcell types -- the same abstraction in `widgets.Widget` would solve both. Note: this is a systemic issue -- `internal/widgets/` already breaks the same rule, so the real fix is to define abstract event types in `internal/term/` that `widgets` and `plugin` use, with tcell conversion happening only in `internal/term/` or `cmd/ttt/`.

### V2: Duplicate style maps (lua_panel.go and sandbox.go)

- **File**: `internal/plugin/lua_panel.go:13-37` (`styleMap`)
- **File**: `internal/plugin/sandbox.go:13-37` (`reverseStyleMap`)
- **Issue**: Two separate maps maintain the same style name <-> `term.Style` mapping. `styleMap` maps `string -> term.Style` and `reverseStyleMap` maps `term.Style -> string`. They must be kept in sync manually. Adding a new named style requires editing both maps.
- **Impact**: A forgotten update in one map silently breaks either Lua-to-Go or Go-to-Lua style resolution. This is a maintenance trap.
- **Fix**: Define a single canonical list (e.g., `var styleEntries = []struct{ Name string; Style term.Style }{...}`) and derive both maps from it using an `init()` function or package-level var assignments.

## Design Concerns (not violations but potential problems)

### D1: Plugin struct has 10 callback fields -- "god struct" tendency

- **File**: `internal/plugin/plugin.go:23-65`
- **Issue**: The `Plugin` struct holds 10 function-valued callback fields (`RequestRedraw`, `PostAsync`, `Log`, `ShowContextMenu`, `ShowInfoDialog`, `ShowConfirmDialog`, `OpenDrawer`, `CloseDrawer`, `OpenTab`, `CloseTab`) plus 4 API interfaces and ~10 data fields. Every capability the host provides is injected individually. The `wirePlugin` function in `commands_plugin.go:272-376` is 100+ lines of point-by-point wiring.
- **Impact**: Adding a new host capability requires: (1) add a field to Plugin, (2) wire it in `wirePlugin`, (3) nil-check it in the Lua binding, (4) clean it up in `Destroy()`. This is error-prone; `Destroy()` already has 14 nil-assignments.
- **Fix**: Group related callbacks into interfaces. For example, a `PluginHost` interface:
  ```go
  type PluginHost interface {
      RequestRedraw()
      PostAsync(*PluginAsyncResult)
      Log(level, message string)
      ShowContextMenu(entries []widgets.MenuEntry, x, y int, onCommand func(string))
      ShowInfoDialog(title string, entries []widgets.KeyValueEntry)
      ShowConfirmDialog(message string, onConfirm func())
      OpenDrawer(renderFunc *lua.LFunction, width, minWidth int)
      CloseDrawer()
      OpenTab(id string, renderFunc, eventFunc *lua.LFunction)
      CloseTab(id string)
  }
  ```
  Then `Plugin` holds a single `Host PluginHost` field, and the app implements that interface. This reduces the wiring surface, makes nil-checking unnecessary (the host is always set or not), and makes the contract explicit.

### D2: Lua callback fields leak into WidgetDesc

- **File**: `internal/plugin/widget_desc.go:77-103`
- **Issue**: `WidgetDesc` stores `*lua.LFunction` pointers (`OnSelect`, `OnExpand`, `OnCommand`, `OnClick`, `OnChange`, `OnSubmit`, `OnMenu`). This means the descriptor layer -- which is conceptually a data description of what to render -- is tightly coupled to gopher-lua types.
- **Impact**: Cannot serialize/deserialize widget descriptors, cannot use them from non-Lua contexts, and the descriptor layer becomes untestable without a Lua state. If you ever wanted to support a second scripting language, the descriptor types would need to change.
- **Fix**: Store callbacks as opaque `interface{}` or use a callback registry (map of string ID -> function) so the descriptor only holds callback IDs. The widget builder would resolve IDs to actual functions. This decouples the descriptor from the Lua runtime.

### D3: Manager imports `internal/config` for a single function call

- **File**: `internal/plugin/manager.go:11` (imports `config`)
- **File**: `internal/plugin/manager.go:44` (uses `config.ConfigFilePath`)
- **Issue**: The Manager uses `config.ConfigFilePath("plugins.ttt.json")` to locate the registry file. This creates a dependency on the config package just for path resolution.
- **Impact**: The Manager cannot be instantiated in tests without the config package's side effects. The `pluginsDir` is already passed as a constructor argument, but the registry path is discovered internally.
- **Fix**: Pass the registry path as a parameter to `NewManager()` or `LoadAll()`, or add a `RegistryPath` field. The caller (`internal/app/`) already knows the config directory and can compute this path.

### D4: Manager imports `os/exec` -- install/update shell out to `git`

- **File**: `internal/plugin/manager.go:8` (imports `os/exec`)
- **File**: `internal/plugin/manager.go:215-218` (`git clone`)
- **File**: `internal/plugin/manager.go:275-278` (`git pull`)
- **Issue**: `Manager.Install()` and `Manager.Update()` directly shell out to `git`. This is not injectable/mockable, making these methods impossible to unit test without a real git repo.
- **Impact**: No tests exist for Install/Update (the test file only tests Registry). Any change to the install flow can only be verified by running it for real.
- **Fix**: Extract a `PluginInstaller` interface (with `Clone(repoURL, targetDir string) error` and `Pull(dir string) error` methods). Inject it into Manager. The default implementation calls git; tests can use a mock.

### D5: `sandbox.go` imports `internal/markdown`

- **File**: `internal/plugin/sandbox.go:6`
- **Issue**: The plugin package imports the `markdown` package to provide `ttt.markdown()` to Lua plugins. The markdown package itself imports `internal/core/highlight` and `internal/term`. This creates a dependency chain: `plugin -> markdown -> core/highlight + term`.
- **Impact**: The plugin package transitively depends on the syntax highlighting engine. Changes to the highlighter could break the plugin system. The markdown rendering is a nice-to-have utility, not core plugin infrastructure.
- **Fix**: This is acceptable if markdown support is considered essential to the plugin API. If it's optional, it could be registered as an extension module from `internal/app/` rather than being hardwired into `sandbox.go`.

### D6: `OpenDrawer` callback takes `*lua.LFunction` as parameter

- **File**: `internal/plugin/plugin.go:52`
- **Issue**: The `OpenDrawer` and `OpenTab` callback fields take `*lua.LFunction` parameters. This means the app-side wiring code (in `commands_plugin.go`) must know about Lua function types. The host-side code creates `PluginPanelWidget` instances that hold these Lua functions.
- **Impact**: The boundary between plugin-internal Lua details and the host application is blurred. The app package imports `gopher-lua` specifically for these callback types.
- **Fix**: Have the plugin package create the `PluginPanelWidget` internally and pass a `widgets.Widget` to the host. The `OpenDrawer` callback would take `(widget widgets.Widget, width, minWidth int)` instead. The Lua function wrapping stays inside the plugin package where it belongs.

## Coupling Issues

### C1: App package imports gopher-lua

- **File**: `internal/app/commands_plugin.go:14` (`lua "github.com/yuin/gopher-lua"`)
- **Issue**: The app package imports gopher-lua because `wirePlugin` assigns callbacks that take `*lua.LFunction` parameters (for `OpenDrawer` and `OpenTab`). The app shouldn't need to know about Lua types.
- **Impact**: If you ever replace or supplement gopher-lua with another Lua implementation, app code must change.
- **Fix**: See D6 above -- restructure callbacks so Lua types don't leak through the Plugin struct into the app layer.

### C2: PluginPanelWidget lives in plugin package but acts as a UI widget

- **File**: `internal/plugin/panel_widget.go`
- **Issue**: `PluginPanelWidget` implements the `widgets.Widget` interface and handles rendering, event dispatch, and focus management. It is essentially a UI component that lives in the plugin package rather than in `internal/widgets/` or `internal/ui/`.
- **Impact**: The plugin package takes on UI responsibilities. The widget needs tcell for `HandleEvent`, which forces the tcell import (V1). If widget rendering patterns change, both `internal/widgets/` and `internal/plugin/` must be updated.
- **Fix**: Move `PluginPanelWidget` to `internal/widgets/` as a generic "scriptable panel widget" that takes a render callback and event callback as Go functions (not Lua-specific). The plugin package would provide the Lua-specific adapter that wraps Lua functions into Go callbacks.

### C3: Widget builder in plugin package duplicates widget creation patterns

- **File**: `internal/plugin/widget_builder.go`
- **Issue**: The `createWidget`/`updateWidget` functions in the plugin package duplicate patterns that the `internal/widgets/builder.go` package already provides. There are two separate widget construction systems: one for JSON-defined widgets (`widgets/builder.go`) and one for Lua-defined widgets (`plugin/widget_builder.go`).
- **Impact**: Adding a new widget type requires changes in both builders. The reconciliation logic (preserving expanded state for trees, etc.) is only in the plugin builder, not available to other consumers.
- **Fix**: Unify widget construction. Define a common `WidgetSpec` type in `internal/widgets/` and have both JSON and Lua descriptors convert to it. The reconciler/builder would live in `internal/widgets/` and serve all consumers.

### C4: PanelProxy tightly couples Lua parsing with widget descriptor construction

- **File**: `internal/plugin/lua_panel.go`
- **Issue**: Each `panel*Widget` function (e.g., `panelTreeWidget`, `panelInputWidget`) does two things: (1) parses Lua table fields, and (2) constructs a `WidgetDesc`. These are interleaved in the same function body, making it hard to test the descriptor construction independently from Lua parsing.
- **Impact**: Testing widget descriptor construction requires a Lua state. The ~600-line file mixes two concerns.
- **Fix**: Separate Lua parsing (extract a typed struct from a Lua table) from descriptor construction (convert the typed struct to a `WidgetDesc`). This allows testing each layer independently.

### C5: Event dispatch uses type assertions in the event loop

- **File**: `internal/app/eventloop.go:305-314`
- **Issue**: The event loop uses `tcell.EventInterrupt` with type-asserted payloads (`*plugin.PluginAsyncResult`, `*pluginInstallResult`, `*pluginUpdateResult`, `*RemoteRegistryResult`). Each new async plugin operation requires adding another case to the event loop's type switch.
- **Impact**: The event loop grows linearly with the number of async plugin operations. It's a central choke point.
- **Fix**: Use a single `PluginEvent` type with a callback field, or use a channel-based approach where plugin results are consumed by a dedicated handler, not the main event loop.

## Recommendations

### R1: Define a PluginHost interface (high priority)
Collapse the 10 callback fields on `Plugin` into a single `PluginHost` interface. This is the highest-leverage change: it clarifies the contract, reduces wiring code, eliminates nil-check proliferation, and makes the boundary testable.

### R2: Stop tcell from leaking into plugin (high priority)
The `HandleEvent(tcell.Event)` method on widgets is the root cause. Define an abstract event type (`widgets.Event` or `term.Event`) and convert at the boundary. This fixes V1 and brings `internal/plugin/` in line with the architecture rules.

### R3: Unify style map definitions (quick win)
Combine `styleMap` and `reverseStyleMap` into a single source of truth. This is a 15-minute change that eliminates a maintenance hazard.

### R4: Make Manager dependencies injectable (medium priority)
Pass the registry path to the constructor. Extract git operations behind an interface. This makes the Manager fully unit-testable.

### R5: Move PluginPanelWidget to widgets package (medium priority)
It belongs with the other widget types. The plugin package should only contain Lua-specific glue code, not widget implementations.

### R6: Consider unifying widget builders (long-term)
Having two separate widget construction systems (`widgets/builder.go` for JSON, `plugin/widget_builder.go` for Lua) will diverge over time. A common `WidgetSpec` type would keep them aligned.
