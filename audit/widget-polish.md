# Widget Polish & Pre-Release Checklist

## 1. Tree View Ellipsis

When tree node labels exceed available width, they clip mid-character with no visual indicator.

**What to do:**
- In `internal/widgets/tree.go` `renderNode()`, the label loop (line ~338) breaks at `maxX` silently.
- When label runes would exceed `maxX`, stop 1 char early and render `…` (U+2026) at the truncation point.
- Same treatment for badge text (line ~355) and icon text.
- `maxX` is already computed as `w - 2 - rightSideWidth(node)`, accounting for scrollbar and right-side elements (dropdown menu icon). No layout changes needed.

**Test:** `--exec` test with a narrow `--size` that forces truncation. Verify `…` appears.

---

## 2. Dialog Button Left/Right Navigation

Dialog footer buttons (OK, Cancel) respond to Tab but not Left/Right arrows. VS Code uses both.

**What to do:**
- In `internal/widgets/dialog.go` `HandleEvent()` (line ~170), before delegating to `d.footer.HandleEvent(e)`:
  - On `KeyLeft`: find the focused button in `d.footer.Children`, focus the previous one.
  - On `KeyRight`: focus the next one.
- The footer is an HStack of ButtonWidgets. Need a small focus helper — iterate `d.footer.Children`, find the one where `IsFocused() == true`, then `SetFocused(false)` on it and `SetFocused(true)` on the neighbor.
- Don't wrap around (Left on first button = no-op, Right on last = no-op).

**Test:** Unit test: build a dialog with 2-3 buttons, send KeyRight/KeyLeft events, verify focus moves.

---

## 3. Box Model Consistency

Every widget embeds `BaseWidget` which has `BoxModel` and `RenderBox()`. Most widgets already call `RenderBox()` in their `Render()` — three don't.

**Already using `RenderBox()` (box model works):**
label, title, button, input, checkbox, divider, hstack, vstack, paragraph, tabbed, tabs, box

**Dropdown** delegates to an inner ButtonWidget which calls `RenderBox()` — box model works transitively.

**Missing `RenderBox()` — these three need fixing:**

**Tree** (`tree.go`): Render directly draws at `x=0, y=0`. Wrap with `RenderBox()` so margins/padding work. After this change, `panel:tree({margin_top = 1})` will work. Also need `Height()` to account for `BoxOverheadH()`.

**ScrollView** (`scrollview.go`): Renders child directly at hardcoded positions. Wrap with `RenderBox()` for outer margin/padding.

**KeyValueList** (`keyvaluelist.go`): Renders at hardcoded positions. Wrap with `RenderBox()`.

**Lua parser gap:** `lua_panel.go` calls `parseBoxModel()` for label, title, keyvalue, input, box — but NOT for tree, list, button, vstack, hstack, divider, scrollview, dropdown. Extend `parseBoxModel()` calls to all widget parsers so the full API is exposed from Lua. The Go side (`widget_builder.go` `applyBoxModel`) already applies to any widget that embeds `BaseWidget`.

**Test:** Lua plugin test: render a tree with `margin_top = 2`, verify the tree content starts 2 rows lower than without margin.

---

## 4. New Widgets (Progress Bar, Table)

Build the two missing widget primitives in `internal/widgets/` and wire them into the Lua plugin API. See #7a and #7b for full specs.

- **Progress bar** — ~50 lines, straightforward. New `progress.go`, Lua wiring in `lua_panel.go` + `widget_builder.go`.
- **Table** — ~200-300 lines, medium effort. New `table.go`, Lua wiring. Column layout, row selection, keyboard nav, scrolling.

Both get box model via `RenderBox()` from day one.

---

## 5. Global Focus Traversal

Two new commands to cycle focus between major UI regions with F6 / Shift+F6.

**Commands:**
- `focus.nextGroup` (F6) — cycle forward through visible regions
- `focus.prevGroup` (Shift+F6) — cycle backward
- `focus.menuBar` (F10) — focus menu bar directly (matching VS Code, not part of F6 cycle)
- `focus.terminal` (no shortcut, command palette only) — focus terminal if bottom panel is open, otherwise open it. Escape returns to editor.

**Focus regions** (in cycle order, only if visible):
1. Editor (EditorGroup)
2. Sidebar (if visible)
3. Bottom panel active tab (if visible — terminal, problems, output, plugin panel)

**Tab switching within panels:**
Currently the tab bars (sidebar tabs, bottom panel tabs) are mouse-only. Instead of making tab pills focusable, add context-aware commands:
- `panel.nextTab` / `panel.prevTab` — checks `Root.Focused` type: if `*ui.SidebarWidget` cycles sidebar tabs, if `*ui.BottomPanelWidget` cycles bottom panel tabs, if editor cycles editor tabs
- One shortcut each (e.g. `Ctrl+PageDown` / `Ctrl+PageUp`, or a chord if those conflict)
- Focus stays on the content — the command just calls `SetActivePanel` on the next/prev tab

**Implementation:** A helper in `app.go` that:
1. Builds a list of visible regions: `[EditorGroup, Sidebar?, BottomPanel?]`
2. Checks which region currently has focus (is `Root.Focused` equal to or inside the region)
3. Calls `Root.SetFocus()` on the next/prev region

For bottom panel, focus the active tab's widget — same logic `OnBottomClick` already uses. No changes to Root or FocusManager needed — just two command handlers and one helper function. Tab bar focus requires making the tab widget focusable and wiring it into the panel's focus group.

**Risk:** Low-medium. F6 cycling is simple. Tab bar focus adds a new focusable element to each panel.

---

## 6. Widget Testing

**Missing unit tests:** box, button, checkbox, dialog, dropdown, hstack, scrollview, tabs, tree, vstack.

**Approach:** Go unit tests with `VirtualSurface` for both rendering and interaction testing. For each widget, first audit all functionality, user interactions, and click areas, then write tests covering every one. The `VirtualSurface` lets us verify rendered cell content at exact coordinates, then send mouse events at those same coordinates to confirm hit regions match rendering — 100% precision, no timing issues.

For Lua-to-widget integration testing (callbacks fire, panel renders correctly), use `--exec --plugin` with small Lua fixture scripts and debug JSON assertions. These are secondary — the Go unit tests are the primary coverage.

**Process per widget:**
1. Read the widget source and catalog: all render regions, all keyboard handlers, all mouse click areas, all callbacks, all edge cases (empty data, overflow, boundary)
2. Write tests for every item in the catalog
3. Use `VirtualSurface` cell scanning to find where elements rendered, then click those exact coordinates

**Priority order and test coverage:**

### 6.1 Tree (`tree.go`) — highest complexity
**Rendering:**
- Basic: items render at correct Y positions with correct indentation per depth
- Icons: expand/collapse chevrons at correct X position
- Actions: action icons render at right edge, correct X positions per action
- Dropdown menu icon: renders at rightmost position when `NodeMenu` configured
- Selection: selected row renders with highlight style
- Scrollbar: appears when items exceed height
- Ellipsis: label truncation with `…` when text exceeds available width (after item #1)
- Badge: badge text renders at correct position
- Muted: muted nodes use muted style

**Keyboard:**
- Up/Down: move selection, wrap or clamp at boundaries
- Left: collapse expanded node, or move to parent
- Right: expand collapsed node
- Enter: activate selected node (fires `OnSelect`)
- `key_commands`: single-char keys fire `OnCommand` with correct command string
- Page Up/Down: scroll by page
- Home/End: jump to first/last item

**Mouse clicks:**
- Click on row label: selects row and activates
- Click on action icon (exact X): fires `OnCommand` with correct action command
- Click on dropdown menu icon (exact X): fires `OnMenu`
- Right-click on row: fires `OnMenu` (context menu)
- Mouse wheel up/down: scrolls, clamps at bounds
- Scrollbar drag: updates scroll position
- Click outside bounds: ignored

**Edge cases:**
- Empty tree (no items)
- Single item
- Deeply nested items (3+ levels)
- Scroll to last item, then collapse parent (scroll should adjust)

### 6.2 Dialog (`dialog.go`)
**Rendering:**
- Title renders centered
- Content widget renders in body area
- Footer buttons render horizontally
- Border renders around dialog

**Keyboard:**
- Tab: cycles focus between footer buttons
- Left/Right: moves focus between footer buttons (after item #2)
- Enter: activates focused button (fires callback)
- Escape: dismisses dialog (returns `EventDismissed`)

**Mouse clicks:**
- Click on button (exact coordinates): fires that button's callback
- Click on close button (if present): dismisses

### 6.3 ScrollView (`scrollview.go`)
**Rendering:**
- Child content renders within bounds
- Scrollbar appears when content overflows
- Content clips at container boundary

**Keyboard/Mouse:**
- Mouse wheel up/down: scrolls content, clamps at 0 and max
- Scrollbar click: jumps to position
- Scroll at top + wheel up: no-op (stays at 0)
- Scroll at bottom + wheel down: no-op (stays at max)

**Edge cases:**
- Content smaller than container (no scrollbar)
- Content exactly fits (no scrollbar)
- Content 1 row taller than container

### 6.4 HStack / VStack (`hstack.go`, `vstack.go`)
**Rendering:**
- Children positioned correctly with gap spacing
- HStack: first child grows to fill, others fixed width
- VStack: children stack vertically with correct Y offsets
- Box model applied correctly (margins, padding affect child layout)

**Mouse clicks:**
- Click on child N: correct child receives the event
- Click on gap between children: no child receives event

### 6.5 Button (`button.go`)
**Rendering:**
- Label renders centered
- Accelerator character (`&` prefix) renders underlined
- Focus ring visible when focused

**Interactions:**
- Click: fires `OnClick` callback
- Enter key when focused: fires `OnClick`
- Accelerator key: fires `OnClick`

### 6.6 Dropdown (`dropdown.go`)
**Rendering:**
- Label renders on button
- Caret/indicator visible

**Interactions:**
- Click: opens menu (fires callback or returns popup)
- Enter when focused: opens menu

### 6.7 Box (`box.go`)
**Rendering:**
- Border renders when enabled (all 4 sides, corners correct)
- Fixed height respected
- Child renders inside border/padding area

### 6.8 Checkbox (`checkbox.go`)
**Rendering:**
- Unchecked: `[ ]` + label
- Checked: `[x]` + label

**Interactions:**
- Click: toggles checked state, fires `OnChange`
- Enter/Space when focused: toggles

### 6.9 Tabs (`tabs.go`)
**Rendering:**
- Tab labels render horizontally
- Active tab has highlight style
- Inactive tabs have default style

**Mouse clicks:**
- Click on tab N (exact X range): switches to that tab, fires `OnSelect`
- Click between tabs: no-op or nearest tab

**Test pattern:**
```go
// Render + click precision test
surface := widgets.NewVirtualSurface(40, 10)
tree := widgets.NewTreeWidget(config) // config has 3 items with actions
tree.SetRect(widgets.Rect{X: 0, Y: 0, W: 40, H: 10})
tree.Render(surface)

// Scan surface to find where action icon "×" was rendered
actionX, actionY := surface.FindCell('×')

// Click that exact coordinate
clickedCmd := ""
config.OnCommand = func(cmd string, n *TreeNode) { clickedCmd = cmd }
ev := tcell.NewEventMouse(actionX, actionY, tcell.Button1, tcell.ModNone)
tree.HandleEvent(ev)
assert(clickedCmd == "close") // the action at that position
```

---

## 7. New Widgets

The widget system is missing a few primitives that tview/bubbles offer and that plugins would benefit from. These also matter if the widget system is ever extracted as a standalone library — the box model is the differentiator, but the primitive set needs to be complete.

### 7a. Progress Bar

A horizontal bar showing completion percentage. Common in plugins (build progress, download, scan).

**API:**
```lua
panel:progress({
  value = 0.65,          -- 0.0 to 1.0
  style = "success",     -- bar fill color
  height = 1,            -- always 1 row
})
```

**Implementation:** `internal/widgets/progress.go`. Simple: fill `value * width` cells with a block character (e.g. `█`), rest with a dimmed character (e.g. `░`). Call `RenderBox()` for box model. About 40-50 lines.

**Lua wiring:** New `panelProgressWidget` in `lua_panel.go`, new `WidgetProgress` kind, create/update in `widget_builder.go`.

### 7b. Table

A columnar data widget with headers, sortable columns, and row selection. The highest-effort new widget but high value — many plugin use cases (docker containers, test results, git log, HTTP headers).

**API:**
```lua
panel:table({
  columns = {
    { label = "Name", width = 20 },
    { label = "Status", width = 10 },
    { label = "CPU", width = 8, align = "right" },
  },
  rows = {
    { "web-app", "Running", "12%" },
    { "db", "Stopped", "0%" },
  },
  on_select = function(row_index) end,
  on_command = function(command, row_index) end,
  node_menu = { ... },
})
```

**Implementation:** `internal/widgets/table.go`. Needs:
- Column layout: fixed widths, optional grow column, header row
- Row rendering with selection highlight
- Keyboard: Up/Down for selection, Enter for activate, context menu
- Scrolling: reuse scroll logic from TreeWidget
- Box model via `RenderBox()`

Estimate: 200-300 lines. Could reuse `SelectableList` logic for keyboard/scroll.

### 7c. Menu Bar (Skip)

MenuBarWidget is 109 lines, only used once, and the menu system's complexity is in the app-level wiring (`menus.go` — overlay management, left/right navigation between menus, shortcut resolution), not the bar widget itself. Plugins already have `dropdown` for menu trigger buttons. Not worth moving.

### 7d. Split Widget (Deferred)

SplitPanelWidget and ContentSplitWidget could be unified into a generic `SplitWidget`, but the border corner rendering is very app-specific — `RightBorderStartY`, `BottomTee` junction, tab bar row click routing, scrollbar column avoidance. Generalizing these details would add complexity without clear benefit. Keep them in `internal/ui/` for now. Revisit if a third split variant is needed or if the widget library is extracted.

---

## 8. Testing Infrastructure

Two gaps found during `--exec` testing that make plugin panel testing painful:

### 8a. `panel.show <id>` command

No way to switch to a specific bottom panel tab programmatically. Currently the only way is clicking the tab, which requires knowing exact coordinates and the tab being visible (not behind `»` overflow). Add a command that takes a panel ID (e.g. `plugin.test_ellipsis`) and activates it — calls `BottomPanel.SetActivePanel(id)` and `ContentSplit.ShowBottom = true`.

### 8b. `ttt.open_drawer()` at init time

Calling `ttt.open_drawer()` during plugin init silently does nothing because the event loop hasn't started and `OpenDrawer` callback isn't wired yet. Buffer the call and replay it after wiring, or document that drawers must be opened from a command/timer.

---

## 9. Markdown Parser (Defer)

Keep the custom parser in `internal/markdown/`. Surface area is small (headings, bold, italic, code, code blocks, lists, links). Only revisit if users report rendering bugs in the markdown preview plugin.

---

## Execution Order

| Phase | Items | Scope |
|-------|-------|-------|
| A | Tree ellipsis (#1) | `tree.go` — single function change |
| A | Dialog button nav (#2) | `dialog.go` — single function change |
| A | Testing infra (#8a, #8b) | `commands_view.go`, `sandbox.go` — small additions |
| B | Box model on tree, scrollview, keyvaluelist (#3) | `tree.go`, `scrollview.go`, `keyvaluelist.go`, `lua_panel.go` |
| C | Progress bar widget (#4, #7a) | New `progress.go`, Lua wiring |
| D | Table widget (#4, #7b) | New `table.go`, Lua wiring |
| E | Widget tests — tree, dialog, scrollview (#6) | New test files, no behavior changes |
| F | Widget tests — hstack, vstack, button, remaining (#6) | New test files |
| G | Global focus (#5) | Root, app wiring — biggest risk |
