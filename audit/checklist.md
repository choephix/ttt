# Plugin System Audit Checklist

## Critical

- [x] **`load()` available in sandbox** [Security] — allows arbitrary Lua code compilation at runtime, defeating the module allowlist. See `security.md` §Critical C1
- [x] **`package.loaders` bypass** [Security] — plugins can call the filesystem loader directly to load and execute any `.lua` file on disk, bypassing the `require` wrapper. See `security.md` §Critical C2
- [x] **Filesystem API has zero path restrictions** [Security] — `fs.read`/`fs.write` accept any path (`~/.ssh/id_rsa`, `/etc/shadow`, etc.) with no sandboxing. See `security.md` §Critical C3
- [x] **Command injection via allowed binary arguments** [Security] — `system.exec` validates binary names but not arguments; `git -c core.fsmonitor=!cmd status` executes arbitrary commands. See `security.md` §Critical C4
- [x] **Nil pointer dereference in async callbacks after plugin destroy** [Bug] — `p.State` is nil when `resultFn` runs if plugin was destroyed mid-async, causing panic. See `bugs.md` §Critical #1
- [x] **Data race on plugin fields from goroutines** [Bug] — async goroutines read `p.PostAsync`/`p.State` without synchronization while `Destroy()` nils them concurrently. See `bugs.md` §Critical #2

## High

- [x] **Path traversal in manifest `entry` field** [Security] — `../` in `entry` causes `DoFile` to execute arbitrary `.lua` files outside the plugin directory. See `security.md` §High H1
- [x] **No SSRF protection in network API** [Security] — plugins can hit `localhost`, internal IPs, cloud metadata (`169.254.169.254`), and `file://` URLs. See `security.md` §High H2
- [x] **Unbounded HTTP response body** [Security] — `io.ReadAll` with no size limit; a malicious server can OOM the editor. See `security.md` §High H3
- [x] **Environment variable access is completely unscoped** [Security] — `system.env` exposes all env vars (`AWS_SECRET_ACCESS_KEY`, `GITHUB_TOKEN`, etc.). See `security.md` §High H4
- [x] **Plugin install via `git clone` with limited URL validation** [Security] — accepts `file://`, SSH, and custom protocols; cloned hooks could run on `git pull` during update. See `security.md` §High H5
- [x] **Plugin loses all app callbacks after Update (no-approval path)** [Bug] — `Destroy()` nils all callbacks, `Init()` does not restore them, caller does not call `wirePlugin()`. See `bugs.md` §High #3
- [x] **Plugin not wired after `SetEnabled(true)`** [Bug] — new plugin object has nil callbacks and API interfaces; `wirePlugin()` is never called. See `bugs.md` §High #4
- [x] **Duplicate panel registrations accumulate after Update** [Bug] — `collectRegistrations` appends without removing old entries, growing lists unboundedly with stale refs. See `bugs.md` §High #5
- [ ] **Plugin package directly imports tcell** [Architecture] — `event_convert.go`, `panel_widget.go`, `widget_builder.go` violate the rule that only `internal/term/` imports tcell. See `architecture.md` §V1
- [ ] **Duplicate style maps across files** [Architecture/Duplication] — `styleMap` and `reverseStyleMap` are manually mirrored in `lua_panel.go` and `sandbox.go`; adding a style requires editing both. See `architecture.md` §V2, `duplication.md` §High #5

## Medium

- [x] **`getfenv`/`setfenv` allow environment manipulation** [Security] — plugins can inspect sandbox wrapper internals via `getfenv(require)`. See `security.md` §Medium M1
- [x] **`rawset`/`rawget` bypass metatables** [Security] — could defeat future access-control metatables; low risk now due to per-plugin VM isolation. See `security.md` §Medium M2
- [ ] **No execution timeout on Lua VM** [Security] — infinite loops in `init()` or callbacks freeze the editor permanently. See `security.md` §Medium M3
- [ ] **No memory limits on Lua VM** [Security] — unbounded allocation via `string.rep` or large tables can OOM-kill the process. See `security.md` §Medium M4
- [ ] **No limit on concurrent async operations** [Security] — a plugin can spawn unlimited goroutines via `exec_async`/`get_async`/`post_async`, exhausting PIDs and FDs. See `security.md` §Medium M6
- [x] **`print` writes to stdout, corrupting terminal state** [Security] — `print()` from plugins bypasses tcell and corrupts the TUI display. See `security.md` §Medium M7
- [x] **`event.mod == nil` check never matches** [Bug] — `mod` is always a string (never nil), so the `r` key shortcut in go-test-runner, docker-manager, and todo-scanner never fires. See `bugs.md` §Medium #6
- [ ] **HTTP client plugin calls `editor.insert()` with wrong argument count** [Bug] — passes `(text)` instead of `(line, col, text)`, causing a type error on every click. See `bugs.md` §Medium #7
- [ ] **Nested widget reconciliation ignores child type mismatches** [Bug] — children matched by index only; type changes within containers keep the old widget. See `bugs.md` §Medium #8
- [x] **`keyEventToLua` drops combined modifiers** [Bug] — `else if` chain reports only the first modifier; `Ctrl+Shift` events lose the shift. See `bugs.md` §Medium #9
- [x] **`ScrollViewWidget` WheelDown doesn't clamp `scrollY`** [Bug] — `scrollY` can exceed bounds between event and render; corrected at draw time but intermediate state is inconsistent. See `bugs.md` §Medium #10
- [ ] **Plugin struct has 10 callback fields (god struct)** [Architecture] — every host capability is a separate field; `wirePlugin` is 100+ lines; `Destroy()` has 14 nil-assignments. See `architecture.md` §D1
- [ ] **Lua callback fields leak into `WidgetDesc`** [Architecture] — `WidgetDesc` stores `*lua.LFunction` pointers, coupling descriptors to gopher-lua types. See `architecture.md` §D2
- [ ] **Manager imports `internal/config` for a single path call** [Architecture] — uses `config.ConfigFilePath()` only for registry path; prevents isolated testing. See `architecture.md` §D3
- [ ] **Manager shells out to `git` directly** [Architecture] — `Install()`/`Update()` use `os/exec` with no interface; impossible to unit test. See `architecture.md` §D4
- [ ] **`sandbox.go` imports `internal/markdown`** [Architecture] — creates transitive dependency chain `plugin -> markdown -> core/highlight + term`. See `architecture.md` §D5
- [ ] **`OpenDrawer`/`OpenTab` callbacks take `*lua.LFunction` parameters** [Architecture] — Lua types leak through Plugin struct into the app layer. See `architecture.md` §D6
- [ ] **App package imports gopher-lua** [Architecture] — `commands_plugin.go` imports `lua` because callbacks expose `*lua.LFunction`. See `architecture.md` §Coupling C1
- [ ] **`PluginPanelWidget` lives in plugin package instead of widgets** [Architecture] — a UI widget that handles rendering and events sits in the plugin package, forcing a tcell import. See `architecture.md` §Coupling C2
- [ ] **Two separate widget builder systems** [Architecture] — `widgets/builder.go` (JSON) and `plugin/widget_builder.go` (Lua) duplicate construction patterns; new widget types need changes in both. See `architecture.md` §Coupling C3
- [ ] **`PanelProxy` couples Lua parsing with descriptor construction** [Architecture] — each `panel*Widget` function interleaves Lua table extraction and `WidgetDesc` building, making isolated testing impossible. See `architecture.md` §Coupling C4
- [ ] **Event dispatch uses type assertions in the event loop** [Architecture] — each new async plugin operation adds another case to the type switch; central choke point. See `architecture.md` §Coupling C5
- [ ] **Children-update pattern duplicated 4 times in `updateWidget`** [Duplication] — identical reconciliation loop for VStack, HStack, ScrollView, and Box. See `duplication.md` §High #1
- [ ] **`checkPanelProxy` nil-guard boilerplate in every widget parser** [Duplication] — same 4-line block repeated 18 times in `lua_panel.go`. See `duplication.md` §High #2
- [ ] **API nil-guard boilerplate across all Lua bindings** [Duplication] — `p.Editor == nil` / `p.Filesystem == nil` / etc. guards repeated 25 times across 4 files. See `duplication.md` §High #3
- [ ] **`hasFocusedChild` and `collectFocusable` duplicate type-switch tree** [Duplication] — 7 identical cases in parallel; every new container widget needs both updated. See `duplication.md` §High #4
- [ ] **Async callback pattern repeated 3 times** [Duplication] — `sysExecAsync`, `netGetAsync`, `netPostAsync` share identical goroutine/PostAsync/error-logging structure. See `duplication.md` §Medium #6
- [ ] **Tree/List widget parsers share most field extraction** [Duplication] — `panelTreeWidget` and `panelListWidget` parse the same fields; List is Tree without `indent`/`on_expand`. See `duplication.md` §Medium #7
- [ ] **Container parsers share `render` + `collectChildren` extraction** [Duplication] — VStack, HStack, ScrollView, Box all start with the same child-collection block. See `duplication.md` §Medium #8
- [ ] **`Set*API` boilerplate in Manager** [Duplication] — 5 consecutive methods with identical `for _, p := range m.plugins { p.X = api }` pattern. See `duplication.md` §Medium #9
- [ ] **`last_panel` + `initialized` + lazy-init boilerplate in every plugin** [Duplication] — 6 plugins repeat 6-8 lines of identical panel lifecycle boilerplate. See `duplication.md` §Medium #10
- [ ] **Notepad plugin rolls its own JSON encoder/decoder (106 lines)** [Duplication] — hand-written JSON parser; provide a built-in `ttt.json` module instead. See `duplication.md` §Medium #11
- [x] **`fs.write()` error return differs from `fs.read()`/`fs.list()`** [API/Tests] — write returns single error string; reads return `nil, error_string`. See `api-consistency.md` §API #1
- [ ] **Error handling pattern differs across modules** [API/Tests] — fs uses multi-return, system uses table with `exit_code`, net uses table with `error`, editor silently no-ops. See `api-consistency.md` §API #2
- [ ] **"API not available" handling inconsistent** [API/Tests] — fs returns nil+error, system/net raise Lua error, editor returns empty silently. See `api-consistency.md` §API #3
- [x] **`sys.exec_async` requires args but `sys.exec` does not** [API/Tests] — `exec("binary")` works, but `exec_async("binary", callback)` misinterprets callback as args table. See `api-consistency.md` §API #4
- [ ] **Box model documentation mismatch** [API/Tests] — code supports box model on `box` widget, but PLUGINS.md and CLAUDE.md omit it. See `api-consistency.md` §API #6
- [ ] **Divider widget has dead `applyBoxModel` call** [API/Tests] — `createDividerWidget` calls `applyBoxModel` but parser never reads box model fields. See `api-consistency.md` §API #7
- [ ] **`ttt.open_drawer()` and `ttt.close_drawer()` undocumented** [API/Tests] — implemented in `sandbox.go` with `panel.drawer` permission but missing from PLUGINS.md. See `api-consistency.md` §Doc #1
- [ ] **`ttt.open_tab()` and `ttt.close_tab()` undocumented** [API/Tests] — implemented with `panel.editor` permission but missing from PLUGINS.md. See `api-consistency.md` §Doc #2
- [ ] **`ttt.markdown()` undocumented** [API/Tests] — renders markdown to styled spans but missing from PLUGINS.md. See `api-consistency.md` §Doc #3
- [ ] **`key_commands` on tree/list widgets undocumented** [API/Tests] — maps single-char keys to commands via `on_command`, not mentioned in PLUGINS.md. See `api-consistency.md` §Doc #4
- [ ] **Extra styles available but undocumented** [API/Tests] — `bold`, `code`, and 11 `syntax_*` styles exist in `styleMap` but are not listed in PLUGINS.md. See `api-consistency.md` §Doc #5
- [ ] **HStack `height` field undocumented** [API/Tests] — parsed in code but not listed in PLUGINS.md config fields. See `api-consistency.md` §Doc #6
- [ ] **Label `width` field undocumented** [API/Tests] — parsed in code but not listed in PLUGINS.md config fields. See `api-consistency.md` §Doc #7
- [ ] **`panel.editor` and `panel.drawer` permissions not fully documented** [API/Tests] — permissions are listed but their corresponding APIs are undocumented. See `api-consistency.md` §Doc #8
- [ ] **CLAUDE.md widget API section is outdated** [API/Tests] — missing `keyvalue`, `hstack`, `scrollview`, `divider` widgets and `key_commands` feature. See `api-consistency.md` §Doc #9
- [ ] **No unit tests for widget descriptor building** [API/Tests] — `lua_panel.go` widget-building functions untested; primary plugin authoring surface. See `api-consistency.md` §Test #4
- [ ] **No tests for async functions** [API/Tests] — `sysExecAsync`, `netGetAsync`, `netPostAsync` goroutine/callback paths untested. See `api-consistency.md` §Test #5
- [ ] **No tests for critical widget files** [API/Tests] — `tree.go`, `scrollview.go`, `box.go`, `hstack.go`, `vstack.go` have no dedicated tests. See `api-consistency.md` §Test #7
- [ ] **No tests for drawer/tab APIs** [API/Tests] — `ttt.open_drawer`, `ttt.close_drawer`, `ttt.open_tab`, `ttt.close_tab` untested. See `api-consistency.md` §Test #1
- [ ] **No tests for `ttt.show_info` and `ttt.confirm`** [API/Tests] — dialog callbacks in `sandbox.go` untested. See `api-consistency.md` §Test #2
- [ ] **No tests for `Manager` lifecycle methods** [API/Tests] — Install, Uninstall, Update, Reload, SetEnabled have no tests. See `api-consistency.md` §Test #6
- [ ] **No e2e tests for bottom panel plugins** [API/Tests] — only sidebar panels are e2e-tested; bottom panel rendering and tab switching untested. See `api-consistency.md` §Test #8
- [ ] **No e2e tests for plugin commands and keybindings** [API/Tests] — command palette appearance and keybinding triggering untested at e2e level. See `api-consistency.md` §Test #9

## Low/Informational

- [ ] **Plugin keybinding hijacking** [Security] — plugins can override critical keybindings like `ctrl+s`; no protected set exists. See `security.md` §Informational I1
- [ ] **Plugin command ID collisions** [Security] — plugins can register IDs matching built-in commands (e.g., `editor.save`). See `security.md` §Informational I2
- [ ] **No content-type validation on HTTP responses** [Security] — binary responses converted to string consume excess memory. See `security.md` §Informational I3
- [ ] **Registry file permissions too broad** [Security] — `plugins.ttt.json` written with `0644` instead of `0600`. See `security.md` §Informational I4
- [ ] **No cryptographic verification of plugins** [Security] — `git clone` with no signature or checksum validation. See `security.md` §Informational I5
- [ ] **`string.rep` as memory exhaustion vector** [Security] — subset of M4 (no memory limits); `string.rep("A", 2^30)` allocates 1GB. See `security.md` §Informational I6
- [x] **Each plugin gets a separate Lua VM** [Security] — positive finding: strong cross-plugin isolation. See `security.md` §Informational I7
- [x] **`Protect: true` prevents Lua panics from crashing Go** [Security] — positive finding: all `CallByParam` uses `Protect: true`. See `security.md` §Informational I8
- [ ] **`net.get_async` callback position flexible but `net.post_async` is not** [API/Tests] — minor asymmetry; `post_async` always requires opts at position 2. See `api-consistency.md` §API #5
- [ ] **No tests for `ttt.markdown`** [API/Tests] — markdown rendering return format untested. See `api-consistency.md` §Test #3
- [ ] **No tests for `Manager.DispatchEvent` type conversion** [API/Tests] — string/int/bool to lua.LValue conversion in dispatch untested. See `api-consistency.md` §Test #10
- [ ] **No functional (blackbox) tests for plugin features** [API/Tests] — `tests/functional/` has no plugin system tests. See `api-consistency.md` §Test #11
- [ ] **Focusable/SetFocused/IsFocused 3-line boilerplate on every widget** [Duplication] — idiomatic Go pattern; could embed `FocusableBase` but some widgets override. See `duplication.md` §Low #12
- [ ] **`Height()`/`Width()` returning 0 defaults repeated across widgets** [Duplication] — `BaseWidget` could provide defaults but savings would be minimal. See `duplication.md` §Low #13
- [ ] **`HandleEvent` delegation pattern in container widgets** [Duplication] — 3 containers share 5-line delegation loop; may diverge. See `duplication.md` §Low #14
- [ ] **`createTreeWidget` and `createListWidget` nearly identical** [Duplication] — only difference is `Indent` field; keeping separate aids readability. See `duplication.md` §Low #15
- [ ] **Docker plugin `cmd_prune_*` handlers repeat confirm-exec-refresh** [Duplication] — 3 identical patterns within one plugin file. See `duplication.md` §Low #16
- [ ] **Docker plugin `*_items` list-building functions share pattern** [Duplication] — 3 similar loops in one plugin; structures differ enough that abstraction hurts readability. See `duplication.md` §Low #17
- [ ] **Docker plugin `refresh_*` functions share fetch-parse-store pattern** [Duplication] — 3 async fetch loops in one plugin; parsing differs per resource type. See `duplication.md` §Low #18
- [ ] **Editor API getters follow uniform closure-nil-check-push pattern** [Duplication] — 6 identical-shaped simple string getters could use a helper. See `duplication.md` §Low #19
