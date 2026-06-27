# Code Duplication Audit

## High Impact (consolidation would significantly reduce maintenance burden)

### 1. Children-update pattern duplicated 4 times in updateWidget

- **Locations**:
  - `internal/plugin/widget_builder.go:126-137` (WidgetVStack)
  - `internal/plugin/widget_builder.go:139-150` (WidgetHStack)
  - `internal/plugin/widget_builder.go:154-167` (WidgetScrollView, nested in ScrollView -> VStack)
  - `internal/plugin/widget_builder.go:169-187` (WidgetBox, nested in Box -> VStack)
- **Pattern**: The same reconciliation loop is copy-pasted 4 times:
  ```go
  children := make([]widgets.Widget, len(desc.Children))
  for i, cd := range desc.Children {
      if i < len(existingChildren) {
          updateWidget(existingChildren[i], cd, p)
          children[i] = existingChildren[i]
      } else {
          children[i] = createWidget(cd, p)
      }
  }
  container.Children = children
  ```
  The only difference is unwrapping the container type (VStack vs HStack vs ScrollView.Child vs Box.Child).
- **Occurrences**: 4 times
- **Suggestion**: Extract a `reconcileChildren(existing []widgets.Widget, descs []WidgetDesc, p *Plugin) []widgets.Widget` helper. Each case then becomes a 2-line unwrap + assign. This would also centralize the reconciliation logic if it needs to handle removals or key-based matching in the future.

### 2. `checkPanelProxy` nil-guard boilerplate in every widget parser

- **Locations**:
  - `internal/plugin/lua_panel.go:203-206` (panelLabelWidget)
  - `internal/plugin/lua_panel.go:237-240` (panelTitleWidget)
  - `internal/plugin/lua_panel.go:262-266` (panelKeyValueWidget)
  - `internal/plugin/lua_panel.go:291-295` (panelTreeWidget)
  - `internal/plugin/lua_panel.go:326-330` (panelListWidget)
  - `internal/plugin/lua_panel.go:355-359` (panelButtonWidget)
  - `internal/plugin/lua_panel.go:375-379` (panelInputWidget)
  - `internal/plugin/lua_panel.go:471-475` (panelVStackWidget)
  - `internal/plugin/lua_panel.go:491-495` (panelHStackWidget)
  - `internal/plugin/lua_panel.go:514-518` (panelDividerWidget)
  - `internal/plugin/lua_panel.go:525-529` (panelScrollViewWidget)
  - `internal/plugin/lua_panel.go:542-546` (panelBoxWidget)
  - `internal/plugin/lua_panel.go:578-582` (panelDropdownWidget)
  - `internal/plugin/lua_panel.go:141-145` (panelSize)
  - `internal/plugin/lua_panel.go:152-156` (panelCell)
  - `internal/plugin/lua_panel.go:170-175` (panelText)
  - `internal/plugin/lua_panel.go:186-190` (panelClear)
  - `internal/plugin/lua_panel.go:603-607` (panelRedraw)
- **Pattern**: Every panel method starts with the identical 4-line block:
  ```go
  proxy := checkPanelProxy(L)
  if proxy == nil {
      return 0
  }
  ```
- **Occurrences**: 18 times
- **Suggestion**: This is intrinsic to gopher-lua's method dispatch pattern; the boilerplate cannot be easily eliminated without a wrapper that loses type safety. **Acceptable duplication** given the framework constraints -- but the repetition is worth noting because if the error handling ever changes (e.g., returning an error string to Lua), all 18 sites need updating. A possible mitigation: a higher-order function `withProxy(fn func(*PanelProxy, *lua.LState) int) lua.LGFunction` that encapsulates the nil-check, reducing each widget method to a single wrapper line.

### 3. `p.Editor == nil` / `p.Filesystem == nil` / `p.System == nil` / `p.Network == nil` guard in every API binding

- **Locations**:
  - `internal/plugin/lua_editor.go:43,54,70,81,100,118,129,140,151,162,175,190,202,216` (14 occurrences)
  - `internal/plugin/lua_fs.go:33,52,69,81` (4 occurrences)
  - `internal/plugin/lua_system.go:34,72,124` (3 occurrences)
  - `internal/plugin/lua_net.go:62,79,101,133` (4 occurrences)
- **Pattern**: Every API function starts with a nil check on the corresponding API interface, returning an empty/error value. In `lua_editor.go`, 14 functions all begin:
  ```go
  if p.Editor == nil {
      L.Push(lua.LString(""))  // or L.Push(L.NewTable()) etc
      return 1
  }
  ```
- **Occurrences**: 25 times total across 4 files
- **Suggestion**: Create a generic guard wrapper per API domain:
  ```go
  func editorGuard(p *Plugin, L *lua.LState, fallback lua.LValue, fn func()) int {
      if p.Editor == nil { L.Push(fallback); return 1 }
      fn()
      return ...
  }
  ```
  Or, since the permission check already gates whether the function is even registered, the nil check is a defense-in-depth belt-and-suspenders. If the API wiring is reliable, these checks could be removed entirely (making this dead code). If they must stay, a higher-order function wrapper would reduce the per-function overhead from 4 lines to 1.

### 4. `hasFocusedChild` and `collectFocusable` duplicate the same type-switch tree

- **Locations**:
  - `internal/widgets/surface.go:46-84` (`hasFocusedChild`)
  - `internal/widgets/focus.go:41-79` (`collectFocusable`)
- **Pattern**: Both functions have identical type-switch structures to walk the widget tree:
  ```go
  switch v := w.(type) {
  case *VStackWidget:
      for _, child := range v.Children { recurse(child) }
  case *HStackWidget:
      for _, child := range v.Children { recurse(child) }
  case *BoxWidget:
      if v.Child != nil { recurse(v.Child) }
  case *ScrollViewWidget:
      if v.Child != nil { recurse(v.Child) }
  case *TabbedWidget: ...
  case *DialogWidget: ...
  case *DrawerWidget: ...
  }
  ```
  Every new container widget requires updating both functions.
- **Occurrences**: 2 parallel type-switches with 7 identical cases each
- **Suggestion**: Add a `Children() []Widget` method to a `ContainerWidget` interface (or add it to `Widget` with a default nil return via `BaseWidget`). Both tree walkers would then use a single generic traversal:
  ```go
  func walkWidgetTree(w Widget, visit func(Widget)) {
      visit(w)
      if c, ok := w.(ContainerWidget); ok {
          for _, child := range c.Children() { walkWidgetTree(child, visit) }
      }
  }
  ```
  This also future-proofs against forgetting to update one of the two switch statements when adding new container types.

### 5. `styleMap` and `reverseStyleMap` are manually-maintained mirror copies

- **Locations**:
  - `internal/plugin/lua_panel.go:13-37` (`styleMap`: `map[string]term.Style`)
  - `internal/plugin/sandbox.go:13-37` (`reverseStyleMap`: `map[term.Style]string`)
- **Pattern**: Both maps contain the same 22 style entries, one as name->constant and the other as constant->name. Adding a new style requires updating both maps in two different files.
- **Occurrences**: 2 maps, 22 entries each
- **Suggestion**: Define one canonical list (e.g., `var styleEntries = []struct{ Name string; Style term.Style }{...}`) and generate both maps from it at init time. Alternatively, define only `styleMap` and generate `reverseStyleMap` by iterating:
  ```go
  var reverseStyleMap = func() map[term.Style]string {
      m := make(map[term.Style]string, len(styleMap))
      for name, style := range styleMap { m[style] = name }
      return m
  }()
  ```

## Medium Impact

### 6. Async callback pattern repeated across `sysExecAsync`, `netGetAsync`, `netPostAsync`

- **Locations**:
  - `internal/plugin/lua_system.go:91-118` (`sysExecAsync`)
  - `internal/plugin/lua_net.go:114-128` (`netGetAsync`)
  - `internal/plugin/lua_net.go:149-162` (`netPostAsync`)
- **Pattern**: All three async functions follow the same structure:
  ```go
  go func() {
      // do work
      resultFn := func() {
          tbl := buildResultTable(...)
          if callErr := p.CallLuaFunc(callback, tbl); callErr != nil {
              slog.Error("plugin async ... callback error", "plugin", p.Name, "error", callErr)
          }
      }
      if p.PostAsync != nil {
          p.PostAsync(&PluginAsyncResult{Plugin: p, Callback: resultFn})
      }
  }()
  ```
  The goroutine launch, error logging, and PostAsync dispatch are identical.
- **Occurrences**: 3 times
- **Suggestion**: Extract a `p.runAsync(fn func() (lua.LValue, error), callback *lua.LFunction)` helper that handles the goroutine, PostAsync wiring, and error logging. Each async binding then only needs to provide the work function and the callback.

### 7. Tree/List widget parsers in `lua_panel.go` share most field extraction

- **Locations**:
  - `internal/plugin/lua_panel.go:291-323` (`panelTreeWidget`)
  - `internal/plugin/lua_panel.go:326-352` (`panelListWidget`)
- **Pattern**: Both parse the same fields from Lua tables: `items`, `on_select`, `on_command`, `node_menu`, `key_commands`. The List version is just Tree without `indent` and `on_expand`.
  ```go
  // Both have:
  if items, ok := L.GetField(tbl, "items").(*lua.LTable); ok { desc.Items = ... }
  if fn, ok := L.GetField(tbl, "on_select").(*lua.LFunction); ok { desc.OnSelect = fn }
  if fn, ok := L.GetField(tbl, "on_command").(*lua.LFunction); ok { desc.OnCommand = fn }
  if menu, ok := L.GetField(tbl, "node_menu").(*lua.LTable); ok { desc.NodeMenu = ... }
  if kc, ok := L.GetField(tbl, "key_commands").(*lua.LTable); ok { desc.KeyCommands = ... }
  ```
- **Occurrences**: 2 (Tree and List parsers)
- **Suggestion**: Extract a `parseTreeListCommon(L, tbl, desc)` helper and call it from both. The Tree parser would additionally parse `indent` and `on_expand`.

### 8. VStack/HStack/ScrollView/Box container parsers share `render` + `collectChildren` extraction

- **Locations**:
  - `internal/plugin/lua_panel.go:471-488` (`panelVStackWidget`)
  - `internal/plugin/lua_panel.go:491-512` (`panelHStackWidget`)
  - `internal/plugin/lua_panel.go:525-540` (`panelScrollViewWidget`)
  - `internal/plugin/lua_panel.go:542-576` (`panelBoxWidget`)
- **Pattern**: All four start with the same child-collection block:
  ```go
  tbl := L.CheckTable(2)
  desc := WidgetDesc{}
  if fn, ok := L.GetField(tbl, "render").(*lua.LFunction); ok {
      desc.Children = collectChildren(L, proxy, fn)
  }
  ```
  Then each adds its own specific fields (`gap`, `height`, `border`, etc.).
- **Occurrences**: 4 times
- **Suggestion**: Extract a `parseContainerBase(L, proxy, tbl) WidgetDesc` helper that handles `render` -> `collectChildren`, and optionally `gap` / `height`. Each widget parser then extends the base desc.

### 9. `SetEditorAPI` / `SetFilesystemAPI` / `SetSystemAPI` / `SetNetworkAPI` / `SetLogFactory` boilerplate

- **Locations**:
  - `internal/plugin/manager.go:153-181` (5 methods)
- **Pattern**: Five consecutive methods follow the exact same pattern:
  ```go
  func (m *Manager) SetXxxAPI(api XxxAPI) {
      for _, p := range m.plugins {
          p.Xxx = api
      }
  }
  ```
- **Occurrences**: 5 methods (4 API setters + 1 log factory setter)
- **Suggestion**: Replace with a single `ForEachPlugin(fn func(*Plugin))` method, or consolidate all APIs into a single `PluginHost` struct that gets assigned once. The current approach means adding a new API domain requires adding yet another `Set*API` method + Manager method. However, the current code is only 30 lines total and each method is trivially simple, so this is moderate impact.

### 10. Lua plugin `last_panel` + `initialized` + lazy-init-on-first-render pattern

- **Locations**:
  - `plugins/go-test-runner/plugin.lua:14,13,406-411`
  - `plugins/todo-scanner/plugin.lua:10,8,191-196`
  - `plugins/docker-manager/init.lua:204,9,281-285`
  - `plugins/cheat-sheet/plugin.lua:11,149`
  - `plugins/http-client/plugin.lua:62,69`
  - `plugins/notepad/plugin.lua:111,226`
- **Pattern**: Every sidebar/bottom plugin repeats the same boilerplate:
  ```lua
  local last_panel = nil
  local initialized = false
  -- ...
  render = function(panel)
      last_panel = panel
      if not initialized then
          initialized = true
          do_initial_work(panel)
      end
      -- ...
  end
  ```
  This is 6-8 lines of boilerplate per plugin, plus every command handler needs to reference `last_panel` for `panel:redraw()`.
- **Occurrences**: 6 plugins (all panel-based plugins)
- **Suggestion**: The plugin framework could support `on_init` and `on_activate` lifecycle callbacks that run once automatically. The panel reference could be stored on the plugin object itself, eliminating the need for a global `last_panel`. Alternatively, provide a Lua utility module (`ttt.utils`) with a `lazy_init(fn)` pattern, but lifecycle callbacks would be cleaner.

### 11. Notepad plugin rolls its own JSON encoder/decoder (106 lines)

- **Locations**:
  - `plugins/notepad/plugin.lua:8-105` (`json_escape`, `json_encode`, `json_decode`)
- **Pattern**: A complete hand-written JSON parser/serializer in Lua. Any plugin that needs to persist structured data would need to duplicate this or copy it.
- **Occurrences**: 1 currently, but will grow as more plugins need data persistence
- **Suggestion**: Provide a built-in `ttt.json` module (backed by Go's `encoding/json`) that plugins can `require("ttt.json")` to encode/decode Lua tables. This would eliminate 100+ lines from the notepad plugin and prevent future plugins from reimplementing JSON parsing. Go already has a robust JSON implementation -- there is no reason for each Lua plugin to reimplement it.

## Low Impact / Acceptable Duplication

### 12. Focusable/SetFocused/IsFocused 3-line boilerplate on every focusable widget

- **Locations**: `tree.go:79-81`, `input.go:57-59`, `button.go:57-59`, `scrollview.go:155-157`, `checkbox.go:28-30`, `tabs.go:48-55`, `select_widget.go:126-128`, `panel_widget.go:82-84`, plus `ui/` widgets (terminal_widget, references_widget, problems_widget, search_widget, etc.)
- **Pattern**:
  ```go
  func (x *Widget) Focusable() bool        { return true }
  func (x *Widget) SetFocused(f bool)       { x.focused = f }
  func (x *Widget) IsFocused() bool         { return x.focused }
  ```
- **Occurrences**: ~10 widget types in `internal/widgets/` + ~15 in `internal/ui/`
- **Suggestion**: Could be consolidated by embedding a `FocusableBase` struct that provides the default implementation. However, the 3-line boilerplate is idiomatic Go (interface methods on concrete types), some widgets override with custom behavior (e.g., `TabsWidget.SetFocused` does more), and embedding would add structural coupling. **Acceptable duplication** -- the Go language design makes this the expected pattern.

### 13. `Height() int { return 0 }` / `Width() int { return 0 }` grow-to-fill defaults

- **Locations**: `vstack.go:32`, `hstack.go:25`, `scrollview.go:22-23`, `tree.go:77-78`, `panel_widget.go:28-29`
- **Pattern**: Multiple widget types return 0 from Height/Width to signal "grow to fill available space."
- **Occurrences**: ~8 widget types
- **Suggestion**: `BaseWidget` could provide default `Height() int { return 0 }` and `Width() int { return 0 }` methods, eliminating the need for widgets to declare them explicitly. However, since many widgets override one or both with non-zero values, the savings would be minimal. **Acceptable duplication**.

### 14. HandleEvent delegation pattern in container widgets

- **Locations**:
  - `internal/widgets/vstack.go:153-160`
  - `internal/widgets/hstack.go:107-114`
  - `internal/widgets/box.go:93-98`
- **Pattern**:
  ```go
  func (v *VStackWidget) HandleEvent(ev tcell.Event) EventResult {
      for _, child := range v.Children {
          if child.HandleEvent(ev) == EventConsumed { return EventConsumed }
      }
      return EventIgnored
  }
  ```
- **Occurrences**: 3 container types
- **Suggestion**: Could extract a `delegateEvent(children []Widget, ev)` helper. However, each container may evolve to handle events differently (e.g., Box might add click-to-focus), and the function is only 5 lines. **Acceptable duplication**.

### 15. `createTreeWidget` and `createListWidget` are nearly identical

- **Locations**:
  - `internal/plugin/widget_builder.go:230-237` (`createTreeWidget`)
  - `internal/plugin/widget_builder.go:240-246` (`createListWidget`)
- **Pattern**: Both create a `TreeWidget` with a `TreeConfig`, the only difference being that Tree passes `Indent: desc.Indent` while List does not.
- **Occurrences**: 2
- **Suggestion**: Could be a single function with a `useIndent bool` parameter. However, keeping them separate makes the code more readable and allows them to diverge if List and Tree gain different behaviors. **Acceptable duplication**.

### 16. Docker plugin `cmd_prune_*` handlers follow identical confirm-exec-refresh pattern

- **Locations**:
  - `plugins/docker-manager/init.lua:210-222` (`cmd_prune_containers`)
  - `plugins/docker-manager/init.lua:224-236` (`cmd_prune_images`)
  - `plugins/docker-manager/init.lua:238-250` (`cmd_prune_volumes`)
- **Pattern**:
  ```lua
  local function cmd_prune_X()
    ttt.confirm("Prune all ... ?", function()
      ttt.log("info", "pruning ...")
      sys.exec_async("docker", {"X", "prune", "-f"}, function(result)
        if result.exit_code == 0 then ttt.log("info", "... prune: ok")
        else ttt.log("error", "... prune: " .. result.stderr) end
        refresh_X(last_panel)
      end)
    end)
  end
  ```
- **Occurrences**: 3 in one plugin
- **Suggestion**: Could extract a generic `prune(resource, args, refresh_fn)` helper within the plugin. Low priority since it is contained within a single plugin file.

### 17. Docker plugin `container_items` / `image_items` / `volume_items` follow the same list-building pattern

- **Locations**:
  - `plugins/docker-manager/init.lua:150-171`
  - `plugins/docker-manager/init.lua:173-186`
  - `plugins/docker-manager/init.lua:188-201`
- **Pattern**: Each builds a list of items from a data table using the same loop structure.
- **Occurrences**: 3 in one plugin
- **Suggestion**: Plugin-internal concern. Could use a generic mapper function but the structures differ enough that forced abstraction would reduce readability. **Acceptable duplication**.

### 18. Docker plugin `refresh_containers` / `refresh_images` / `refresh_volumes` follow the same fetch-parse-store pattern

- **Locations**:
  - `plugins/docker-manager/init.lua:22-42`
  - `plugins/docker-manager/init.lua:44-61`
  - `plugins/docker-manager/init.lua:63-80`
- **Pattern**: Each calls `sys.exec_async("docker", ...)`, parses the output line by line, and stores results in a module-level table.
- **Occurrences**: 3 in one plugin
- **Suggestion**: Plugin-internal concern. The parsing differs per resource type. **Acceptable duplication**.

### 19. Editor API getter functions follow a uniform closure-nil-check-push pattern

- **Locations**: `internal/plugin/lua_editor.go:41-158` (9 read-only getter functions)
- **Pattern**: Each getter follows:
  ```go
  func editorXxx(p *Plugin) lua.LGFunction {
      return func(L *lua.LState) int {
          if p.Editor == nil { L.Push(fallbackValue); return 1 }
          L.Push(lua.LString(p.Editor.Xxx()))
          return 1
      }
  }
  ```
  The 6 simple string getters (`BufferText`, `CurrentLine`, `FilePath`, `FileName`, `Language`, `SelectionText`) are completely interchangeable modulo the method name.
- **Occurrences**: 6 identical-shaped simple getters
- **Suggestion**: A `stringGetter(p *Plugin, get func(EditorAPI) string) lua.LGFunction` helper would reduce each to a one-liner. However, the current code is clear and each function is only ~8 lines. **Low impact**.
