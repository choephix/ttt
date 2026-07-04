# QA Sweep — v1.0.0 (2026-07-04)

Full regression sweep before the 1.0.0 release (everything except LSP), driven via the
`--exec` debug harness, a widget-torture plugin, and a chaos-monkey Docker soak.
Binary under test: `bin/ttt` built from main plus the Phase 1 doc/API fixes.

Severity: 🔴 blocker (fix before release) · 🟡 should-fix · ⚪ minor/cosmetic

| # | Sev | Area | Status | Summary |
|---|-----|------|--------|---------|
| 1 | ⚪ | Docs/Workspace | FIXED (docs) | Opening a file directly creates NO workspace — that is intended behavior (a workspace requires a folder argument). The README/docs-web wrongly promised "workspace = git repo root" for file opens; docs corrected. The real repo-root fallback is for **folders**: git features (changes panel, branch) resolve the enclosing repo root when the opened folder is nested in a repo — verified working. An initial code change implementing the README's wrong promise was reverted. |
| 2 | 🔴 | Tests/Settings | FIXED | Functional tests executed Options toggles that persist to the developer's real `~/.config/ttt/settings.json` — parallel test files raced on it (reproducible syntax-fold failure) and a crashed run would permanently flip user settings. Tests also loaded the user's real installed plugins. |
| 3 | 🔴 | Editor | FIXED | Chaos crash: `UpperCase`/`LowerCase`/`TitleCase` panic (`slice bounds out of range`) when the selection anchor column is beyond the line's rune count. |
| 4 | 🔴 | Widgets | FIXED | `p:scrollview` containing any grow widget (tree, list, table, markdown) rendered completely empty — `VStack.HeightForWidth` collapsed to 0 when any child was unmeasurable. |
| 5 | 🟡 | Widgets | FIXED | Tab focus inside a plugin scrollview never scrolled the focused widget into view — keyboard users tabbed into invisible widgets. |
| 6 | 🔴 | Widgets | FIXED | All mouse interaction inside `p:scrollview` was dead (title menus, dropdowns, buttons, tree clicks): scroll views forwarded screen-coordinate events to children holding content-space rects, and popups opened at wrong positions. |
| 7 | 🔴 | Widgets | FIXED | Infinite render loop: tree state restoration during reconcile re-fired `on_expand` for every expanded node on every render — any plugin whose `on_expand` logs or redraws looped forever. |
| 8 | ⚪ | Docs | FIXED | `on_expand` documented as firing on expand and collapse; it only fires on expand (lazy-load design). Docs corrected. |
| 9 | 🟡 | Plugins/Drawer | FIXED | `ttt.open_drawer` while a drawer was open stacked both drawers; `ttt.close_drawer` blind-popped the top overlay (could close an unrelated dialog). Now tracked and replaced/removed specifically (`Root.RemoveOverlay`). |
| 10 | 🔴 | Widgets/Table | FIXED | Table columns without an explicit `width` rendered nothing (auto-width was unimplemented — `renderCell` bailed on width 0). Auto columns now share remaining width. |
| 11 | 🟡 | Widgets/Table | FIXED | Table `node_menu` was completely unwired — no right-click, no Shift+Enter, the right-click handler was an empty stub checking the wrong button. Now opens the context menu (right-click + Shift+Enter) and routes to `on_command`, like trees. |
| 12 | 🔴 | Tests/Chaos | FIXED | Chaos monkey and chaos REPLAYS ran against the developer's real `~/.config/ttt` — random commands persist settings/keybindings. A local replay during this sweep wiped all 76 keybindings in the real config (restored by deleting the damaged file; it contained no customizations). Harness now sets `config.OverrideConfigDir`. |
| 13 | 🟡 | Docs/Keybindings | FIXED | Three divergent keymaps (compiled defaults vs `config/keybindings.json` vs README/docs-web). Decision: compiled `DefaultKeybindings()` is canonical. `config/keybindings.json` regenerated from it (79 entries); README keybindings table rewritten; feature-section key refs fixed (Ctrl+Shift+E/F/G→Ctrl+K E/F/C, Ctrl+H→Ctrl+R, Ctrl+J→Ctrl+K B, `` Ctrl+` ``→Ctrl+T, Ctrl+PgDn/Up→Alt+./,, Ctrl+K Ctrl+T theme→palette/menu); docs-web tab rows + formatter key fixed. Also fixed a real conflict inside the defaults: `ctrl+k f` was bound to BOTH `sidebar.search` (winner) and `editor.formatExternal` (dead) — external format moved to `ctrl+l e`, grouped with the other `ctrl+l` format chords. CLAUDE.md keymap notes corrected. |
| 14 | 🟡 | Plugins panel | FIXED | "Plugins: Show Panel" opened the panel without focusing it — typing went into the editor buffer. Now routes through `ShowPanel` like the other sidebar commands. |
| 15 | ⚪ | Tabs | OPEN | Tab overflow at narrow widths shows a clipped tab fragment next to the ◀ indicator (e.g. "◀ .go"). Cosmetic. |

---

## Findings

### #1 — File opens create no workspace — INTENDED; docs fixed

- **Original report**: `ttt README.md` leaves the workspace with zero folders (empty Explorer). The README promised "workspace is the git repo root (falls back to the file's parent dir)".
- **Resolution**: per the author, this is by design — opening a file must NOT open a workspace; a workspace requires a folder argument. The README/docs-web workspace examples were corrected. The genuine repo-root fallback applies to **folder** opens: git integration (changes panel, status-bar branch) resolves the enclosing repo root when the opened folder sits inside a repo (verified: opening `repo/src` shows repo-wide changes and the repo branch).
- **Reverted**: the interim code change (implicit workspace from file's repo root) and its functional tests were removed; `resolveArgs` now carries a comment documenting the intent.

### #2 — Functional tests mutate the real user config — FIXED

- **Symptom**: `syntax-fold.test.js` failed reproducibly in the full run, passed in isolation.
- **Cause**: every Options toggle calls `config.SaveSettings()` which always writes `~/.config/ttt/settings.json` (the `--config` flag only affects *loading*). `options-toggle.test.js` runs "Toggle Syntax Highlight" concurrently with other test files, so parallel binaries loaded syntax-highlight-off mid-window. Worse: an aborted test run leaves the developer's settings permanently flipped, and all test binaries loaded the developer's real plugins (with their exec permissions).
- **Fix**: new `TTT_CONFIG_DIR` env var wires into the existing `config.OverrideConfigDir`; `tests/functional/tui.js` sets it to a per-test temp dir, isolating settings, plugin registry, and plugins.
- **Verification**: full functional suite green (188/188) three runs in a row locally; user settings verified unmutated.

### #3 — Case-transform panic on stale selection anchor — FIXED

- **Found by**: chaos monkey soak (`chaos-output/crash-1783175956353591268-1581.json`), 1 crash in 3000 iterations.
- **Cause**: `transformSelection` (internal/ui/editor_widget.go) sliced the first selected line with `[:start.Col]` without clamping — the selection anchor can legitimately hold a column beyond the line's rune count (anchor set on a longer line, buffer changed underneath).
- **Fix**: clamp the start column to the line's rune length; cursor/selection restore uses the clamped value.
- **Tests**: unit regression `TestTransformSelectionStaleAnchorBeyondLine` (internal/ui); chaos replay of the crash file passes; the 4 older crash reports (June 30) also replay clean.
- **Note**: chaos soak restarted with the fixed binary.

### #4 — ScrollView with grow widgets renders empty — FIXED

- **Repro**: torture plugin sidebar/bottom panel whose `p:scrollview` contains a tree/list/table/markdown widget → entire panel blank (only a scrollbar row).
- **Cause**: `ScrollViewWidget` measures content via `VStack.HeightForWidth`, which returned 0 for the whole stack when any child was a grow widget (`Height()==0` with no `HeightForWidth`).
- **Fix**: new `ContentHeighter` interface (`ContentHeight()` on TreeWidget = visible rows, TableWidget = header+rows, ScrollViewWidget = child content); `VStackWidget.MeasureGrow` mode (set only on plugin scrollview stacks) measures grow children by content and skips unmeasurable ones instead of collapsing. Plain VStack grow semantics unchanged (explorer/changes layouts unaffected).

### #5 — Focus doesn't scroll into view inside scrollviews — FIXED

- **Fix**: `FocusManager.OnFocusChange` hook + `widgets.ScrollIntoView(root, target)`; wired in the plugin panel's `WidgetState`. Tab/Shift+Tab now scroll the focused widget into view (verified via --exec).

### #6 — Mouse events broken inside scrollviews — FIXED

- **Repro**: title dropdown `⋮` click did nothing; same for any button/tree/dropdown inside `p:scrollview`.
- **Cause**: `ScrollViewWidget.Render` gave children content-space rects (`0,0`-based virtual surface) but `HandleEvent` forwarded raw screen-coordinate mouse events; popup positioning (dropdown/title/node menus) also derived screen positions from those rects.
- **Fix**: children now keep screen-space rects (viewport origin minus scroll offset) — consistent with the rest of the widget tree; rendering is unaffected because drawing goes through relative Sub surfaces. Mouse button events outside the viewport are gated so scrolled-out widgets can't catch stray clicks; `FocusManager` hit-tests use `VisibleRect` (rect clipped by enclosing scrollview viewports); `ScrollIntoView` converts to per-scrollview content coords. MarkdownWidget's existing screen-coordinate math is unchanged and still correct.
- **Verified**: title menu opens at the right position and `on_menu` fires; dropdown, button, tree expand/select all work by mouse inside the scrollview.

### #7 — Infinite on_expand loop from reconcile — FIXED

- **Repro**: click to expand a tree node whose `on_expand` calls `ttt.log` (or `panel:redraw()`) → endless `tree expand` callbacks (thousands per second), UI livelocked.
- **Cause**: plugin reconcile restores tree expansion every render via `RestoreExpanded`, which fired `Config.OnExpand` for every expanded node — each callback triggered another redraw → another reconcile.
- **Fix**: `RestoreExpandedSilent` (no callbacks) used in the reconcile path; explorer `Reload` keeps the notifying variant for lazy directory loading.

### #9 — Drawer stacking / close_drawer overlay bugs — FIXED

- **Repro**: run two commands that each call `ttt.open_drawer` → both drawers render side by side, the older one clipped. `ttt.close_drawer` popped whatever overlay was topmost.
- **Fix**: App tracks the active plugin drawer; opening a new one replaces it; closing removes that specific overlay via new `Root.RemoveOverlay` (dialogs stacked above are untouched). Verified via --exec.

### #10 — Table auto-width columns invisible — FIXED

- **Repro**: `p:table` column without `width` (e.g. `{label = "Name"}`) → column header and all its cells simply absent.
- **Cause**: no auto-width computation existed; `renderCell` returned immediately for `Width <= 0`.
- **Fix**: `effectiveWidths` — fixed widths honored, auto columns share the remaining content width (min 1 cell). Verified: long names truncate with `…`, right-align still works.

### #11 — Table node_menu unwired — FIXED

- **Fix**: `TableConfig.OnMenu` + builder wiring through `ShowContextMenu` (same pattern as trees); right-click (Button2) and Shift+Enter open the menu on the row; selection follows right-click. `on_select` semantics kept as selection-change + Enter (per existing unit test), docs updated to say so; reconcile now also refreshes `node_menu`/`key_commands` and box model.

---

## Areas checked

- [x] Widget torture plugin: sidebar (all widget types, unicode, empty, overflow content) — findings #4–#8
- [x] Widget torture: bottom panel table — findings #10, #11
- [x] Widget torture: drawer, editor tab, empty widgets — finding #9
- [x] Widget focus cycling (Tab/Shift+Tab into scrollviews, F6, hstack buttons, markdown drag/copy) — finding #5
- [x] Editor core: typing, selection (mouse drag), undo/redo to save point, clipboard — plus 191 functional tests green
- [x] Editor: multi-cursor, find/replace, folding, go-to-line, word wrap — covered by functional suite (191/191)
- [x] Editor: mouse click/drag; scrollbar drag covered by chaos drag events
- [x] Tabs: open/close/switch (alt+./,), overflow at 80w (◀ indicator, active-tab priority), dirty-close dialog (Discard/Cancel/Save)
- [x] Explorer: navigation, expand, Shift+Enter context menu, New File/Rename/Delete end-to-end (disk verified) — finding #1 (docs)
- [x] Changes panel: stage (space), stage-all (a), commit input (Tab→type→Enter, git log verified), diff view side-by-side
- [x] Search panel: rg query, grouped results, jump-to-line
- [x] Command palette (ctrl+p) + quick open (ctrl+k p) — finding #13 (README documents wrong keys)
- [x] Menus: F10 file menu, title menus, dropdowns, node context menus
- [x] Dialogs: confirm (delete), input (rename/new file), select (theme picker via command), dirty-close
- [x] Bottom panel: tabs, panel switching, OUTPUT log (exercised constantly by torture plugin)
- [x] Plugins panel + marketplace: search filter, detail tab with markdown README — finding #14
- [x] Terminal smoke: spawn (ctrl+k t), echo roundtrip, vertical tab bar
- [x] Chaos soak: 46+ min on fixed binary, 0 crashes (old binary: 1 crash/3000 iters = finding #3); all 5 historical crash reports replay clean
- [x] Test matrix: go test ./... green, functional 191/191, integration 39 passed/8 skipped, docs-web builds, gofmt clean
- [ ] NOT covered: LSP (excluded by scope), bottom-panel drag-resize, real-PTY terminal edge cases beyond integration suite
