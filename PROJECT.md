# ttt — Project Plan

ttt is a fully-featured code editor that lives in the terminal. Not a simplified terminal editor — a real alternative to VS Code, Zed, and Sublime that happens to run in your terminal.

With AI-driven development, developers spend less time in editors manually. When they do open one, they want something fast and capable that's already where they are — not a full GUI app booting up. ttt fills that gap: split panes, file explorer, tabs, command palette, menus, dialogs, syntax highlighting, mouse support, and a secure plugin system — all in a single Go binary.

> **Note:** Project was renamed from pico/macro to **ttt** (Terminal Text Tool). Module name is `ttt`, binary is `ttt`, repo is `eugenioenko/ttt`.

---

## Layout

The target screen layout mirrors VS Code:

```
┌──────────────────────────────────────────────────────┐
│  Menu Bar  (File  Edit  View  Help)                  │
├────────┬─────────────────────────────────────────────┤
│        │  Tab Bar  [main.go] [buffer.go] [+]         │
│  File  ├─────────────────────┬───────────────────────┤
│  Tree  │                     │                       │
│        │   Editor Pane 1     │   Editor Pane 2       │
│        │   (with gutter)     │   (with gutter)       │
│        │                     │                       │
│        ├─────────────────────┴───────────────────────┤
│        │  Bottom Panel (output / find results)       │
├────────┴─────────────────────────────────────────────┤
│  Status Bar  (file · ln · col · lang · encoding)     │
└──────────────────────────────────────────────────────┘
```

Each region is a UI component that owns its own rect, receives routed input, and renders into the cell grid. The window manager handles layout, focus, z-order (for floating dialogs), and resize propagation.

---

## UI Architecture

This section defines the component system that every visual feature is built on. Get this right and tabs, splits, sidebar, dialogs, and menus are all instances of the same abstractions. Get it wrong and each feature is a special case bolted onto the side.

### Widget Interface

Every UI element — editor pane, sidebar, tab bar, menu, dialog, status bar — implements a common `Widget` interface:

```go
type Widget interface {
    SetRect(Rect)
    Rect() Rect
    HandleEvent(Event) EventResult
    Render(surface *RenderSurface)
    Focusable() bool
}
```

`EventResult` tells the parent whether the event was consumed:

```go
type EventResult int
const (
    EventIgnored  EventResult = iota
    EventConsumed
)
```

Widgets do not know their absolute screen position. They render relative to (0,0) and the `RenderSurface` translates to absolute coordinates. This makes widgets reusable and testable in isolation.

### RenderSurface (Clipping Abstraction)

This is the single most important UI abstraction. It wraps the `Screen` interface and clips all writes to the widget's bounds:

```go
type RenderSurface struct {
    screen Screen
    clip   Rect   // absolute bounds on screen
}

func (s *RenderSurface) SetCell(x, y int, c Cell) {
    absX := s.clip.X + x
    absY := s.clip.Y + y
    if absX < s.clip.X || absX >= s.clip.X+s.clip.W { return }
    if absY < s.clip.Y || absY >= s.clip.Y+s.clip.H { return }
    s.screen.SetCell(absX, absY, c)
}

func (s *RenderSurface) Sub(r Rect) *RenderSurface {
    // Returns a new surface whose clip is the intersection of s.clip and r
    // offset relative to s.clip. Children call Sub() to create their own surface.
}
```

Without this, every widget does manual coordinate math and bounds-checking. With it, a widget can render as if it owns a full screen, and nesting just works.

### Containers and Layout

Containers are widgets that hold children and divide their Rect among them. Three container types cover the entire layout:

**VBox** — stacks children vertically. Each child is either fixed-height or flex (takes remaining space).

```
VBox
├── MenuBar        (fixed: 1 row)
├── MainArea       (flex: fill)
└── StatusBar      (fixed: 1 row)
```

**HBox** — stacks children horizontally. Each child is either fixed-width or flex.

```
HBox
├── Sidebar        (fixed: 30 cols, toggleable)
└── EditorArea     (flex: fill)
```

**Split** — exactly two children with a draggable divider. Orientation is vertical or horizontal. The divider position is stored as a ratio (0.0–1.0) so it adapts to resize.

```
Split (vertical)
├── EditorPane1    (ratio: 0.5)
└── EditorPane2    (ratio: 0.5)
```

Layout sizing for children uses a simple model:

```go
type LayoutConstraint struct {
    Type  ConstraintType  // Fixed, Flex, Hidden
    Value int             // pixels for Fixed, weight for Flex
}
```

Hidden means the child is collapsed (sidebar toggled off, bottom panel closed). Its space is redistributed to flex children.

### The Widget Tree

The full VS Code layout expressed as a widget tree:

```
Root (VBox)
├── MenuBar                        fixed: 1 row
├── MainArea (HBox)                flex: fill
│   ├── Sidebar                    fixed: 30 cols, toggleable
│   │   └── FileTree
│   └── EditorArea (VBox)          flex: fill
│       ├── TabBar                 fixed: 1 row
│       ├── PaneContainer (Split)  flex: fill
│       │   ├── EditorPane
│       │   └── EditorPane         (created on split, removed on close)
│       └── BottomPanel            fixed: 10 rows, toggleable
├── StatusBar                      fixed: 1 row
└── [Overlay Layer]
    ├── CommandPalette             (modal)
    ├── Dialog                     (modal)
    └── ContextMenu                (non-modal, positional)
```

Containers call `SetRect` on their children during layout, then delegate `Render` by creating a `Sub()` surface for each child.

### Event Routing

Events flow through the tree in a specific order:

1. **Overlay layer first.** If a modal overlay exists, it captures all events. Non-modal overlays (context menu, autocomplete) only capture events that hit their bounds; misses dismiss them.
2. **Global keybindings.** Certain keys are handled before the focused widget: Ctrl+Shift+P (command palette), Ctrl+B (toggle sidebar), Ctrl+Q (quit). These are registered on the root.
3. **Focused widget.** The event is delivered to the widget that currently has focus.
4. **Bubble up.** If the focused widget returns `EventIgnored`, the event bubbles to its parent container, then grandparent, etc. This lets containers handle events their children don't care about (e.g., a split container handles Ctrl+Arrow to resize even though the editor pane ignores it).

Mouse events are routed differently: hit-test the overlay layer, then walk the widget tree to find which widget's Rect contains the click coordinates. That widget receives the event and gains focus.

### Focus Management

- Exactly one widget has keyboard focus at any time.
- Focus is tracked centrally by the root (not stored in individual widgets).
- Focus moves via: click, Tab/Shift+Tab within a focus group, programmatic set (opening a dialog focuses its first input).
- **Focus groups**: a container can define a focus group for Tab cycling. The main editor area is one group. A dialog is another. Tab cycles within the active group only.
- **Modal focus trap**: when a modal dialog is open, focus is trapped inside it. Tab cycles only the dialog's widgets. Esc dismisses the modal and restores previous focus.

### Z-Order and Overlays

The rendering and event system uses a flat layer stack:

- **Layer 0**: the main widget tree (everything in the Root VBox)
- **Layer 1+**: overlays — dialogs, menus, command palette, tooltips

Rendering order: layer 0 first, then overlays in order. Event order: reverse — topmost overlay gets first shot.

Overlays are not part of the main widget tree. They're managed by a separate list on the root:

```go
type Overlay struct {
    Widget Widget
    Modal  bool
}

type Root struct {
    Main     Widget       // the VBox tree
    Overlays []Overlay    // rendered on top, events routed first
}
```

Opening a dialog pushes an Overlay. Closing it pops. The main tree doesn't know overlays exist.

### Rendering Pipeline

Each frame follows this sequence:

1. Root calls `layout()` — propagates Rects down the tree based on constraints and current terminal size.
2. Root calls `Render()` on the main tree — each container creates `Sub()` surfaces for children and delegates.
3. Root calls `Render()` on each overlay, from bottom to top.
4. The diff-based `Renderer` compares the new cell grid to the previous one and emits minimal updates to tcell.

Step 4 already exists. Steps 1–3 are what the Widget/Container system provides.

### Theming

A `Theme` is a named map of style tokens to tcell styles:

```go
type Theme struct {
    Name   string
    Styles map[string]Style  // "editor.bg", "gutter.fg", "tab.active", "menu.highlight", etc.
}
```

Widgets look up styles by token name from the active theme. Color choices never appear in widget code. Theme is passed through the RenderSurface or a context object so every widget has access.

### Event System (EventBus)

Widgets and plugins need to react to things happening elsewhere without being directly coupled. The editor pane shouldn't know the status bar exists, but the status bar needs to update when the cursor moves. A plugin shouldn't know about the tab bar, but the tab bar needs to show a dirty indicator when a plugin modifies a buffer.

The event bus is the decoupling layer:

```go
type EventType string

const (
    EventBufferChanged    EventType = "buffer.changed"      // content edited
    EventBufferDirty      EventType = "buffer.dirty"        // dirty flag changed
    EventFileOpened       EventType = "file.opened"         // file loaded into a buffer
    EventFileSaved        EventType = "file.saved"          // buffer written to disk
    EventFileClosed       EventType = "file.closed"         // buffer closed
    EventCursorMoved      EventType = "cursor.moved"        // cursor position changed
    EventSelectionChanged EventType = "selection.changed"   // selection changed
    EventFocusChanged     EventType = "focus.changed"       // keyboard focus moved
    EventThemeChanged     EventType = "theme.changed"       // active theme switched
    EventConfigChanged    EventType = "config.changed"      // settings changed
    EventLayoutChanged    EventType = "layout.changed"      // pane split/closed/resized
)

type Event struct {
    Type    EventType
    Payload any       // event-specific data (file path, cursor pos, etc.)
}

type EventBus struct {
    subscribers map[EventType][]func(Event)
}

func (bus *EventBus) Subscribe(eventType EventType, handler func(Event))
func (bus *EventBus) Publish(event Event)
```

**How it flows:**

1. User types a character → editor pane modifies the buffer → publishes `buffer.changed`
2. Status bar is subscribed to `buffer.changed` → updates dirty indicator
3. Tab bar is subscribed to `buffer.dirty` → shows dot on the tab
4. Plugin host is subscribed to `buffer.changed` → forwards to service plugins that registered for `textDocument/didChange`

The event bus is **synchronous and in-process** for internal subscribers (widgets). The plugin host bridges events to external plugin processes asynchronously over JSON-RPC.

**Relationship to input events vs. domain events:**

- **Input events** (key press, mouse click, resize) flow through the widget tree via `HandleEvent` — top-down, focused routing as described in the Event Routing section.
- **Domain events** (buffer changed, file saved, cursor moved) flow through the EventBus — pub/sub, any subscriber can listen.

These are two separate systems. A key press is an input event that gets routed to the editor pane. The editor pane handles it by modifying the buffer, then publishes a domain event on the bus. Other widgets and plugins react to the domain event.

### How This Maps to Existing Code

| Current code | Becomes |
|---|---|
| `ui.Window` | One implementation of `Widget` (the editor pane widget) |
| `ui.WindowManager` | Evolves into `Root` — owns the widget tree, overlay stack, and focus |
| `ui.Rect` | Stays as-is, used by all widgets |
| `view.Viewport` | Internal to the editor pane widget |
| `view.StatusBar` | Becomes a `Widget` that renders into its surface |
| `render.Renderer` | Unchanged — sits below the widget layer, receives the final cell grid |
| `term.Screen` | Unchanged — wrapped by `RenderSurface` |
| `term.MockScreen` | Used to test individual widgets in isolation |

The key refactor is: `main.go`'s monolithic event loop and manual coordinate math gets replaced by the widget tree. The loop becomes: poll event → route through tree → layout → render. All the per-widget logic moves into Widget implementations.

---

## Phase 1 — Core Editing ✅

- [x] Line-based buffer with rune-level insert/delete
- [x] File load/save
- [x] Cursor with visual column and goal-column preservation
- [x] Command-based undo/redo
- [x] Regex-based syntax highlighting (Go)
- [x] Viewport with scrolling
- [x] Status bar (filename, cursor position, dirty flag)
- [x] Diff-based renderer (double-buffered)
- [x] tcell abstraction with mock screen for testing
- [x] Window and WindowManager structs
- [x] Accept filename as CLI argument
- [x] Ctrl+S save, Ctrl+Q quit with dirty-file warning
- [x] Active line highlight — background highlight on cursor's current line (`StyleActiveLine`, default bg #282828)
- [x] Empty lines past buffer show blank space (removed vim-style `~` tildes)
- [x] Ctrl+Z / Ctrl+Y wired to the undo system
- [x] Tab key inserts spaces (configurable tab width via .editorconfig)
- [x] Home/End keys (line start/end)
- [x] PageUp/PageDown keys
- [x] Delete key support
- [x] Auto-indent (new line inherits whitespace)

---

## Phase 2 — Widget Framework ✅

- [x] `RenderSurface` — wraps cell grid with clipping and coordinate translation, supports `Sub()` for nesting
- [x] `Widget` interface — SetRect, GetRect, HandleEvent, Render, Focusable
- [x] `EventResult` type (Consumed / Ignored)
- [x] `VBox` — lays out children vertically with Fixed/Flex constraints
- [x] `HBox` — lays out children horizontally with Fixed/Flex constraints
- [x] `SplitPanelWidget` — sidebar/content split with draggable divider
- [x] `ContentSplitWidget` — editor/bottom-panel split with draggable divider
- [x] `Root` struct — owns widget tree, overlay stack, focus pointer, global/chord keybindings
- [x] Focus tracking via `Root.SetFocus()`
- [x] Event dispatch: modal overlays → chord bindings → global bindings → focused widget
- [x] Overlay stack with push/pop and Modal flag
- [x] `EditorPaneWidget`, `EditorGroupWidget` (tabs + editor), `StatusBarWidget`, `MenuBarWidget`
- [x] `ExplorerWidget`, `SearchWidget`, `SidebarWidget`
- [x] `CommandPaletteWidget` with fuzzy filtering
- [x] `TabBarWidget` with click support and dirty indicators
- [x] `BottomPanelWidget` with tab switching
- [x] main.go event loop: poll event → Root.HandleEvent → render
- [x] main.go refactored — split into main.go (36 lines), commands.go, widgets.go, eventloop.go, theme.go

---

## Phase 3 — Selection, Clipboard, and Search (in progress)

- [x] Selection model: anchor + cursor defining a range, rendered with inverted style
- [x] Shift+Arrow / Shift+Home / Shift+End / Shift+PgUp / Shift+PgDn to extend selection
- [x] Ctrl+A select all
- [x] Ctrl+C / Ctrl+X / Ctrl+V — copy, cut, paste (internal clipboard)
- [x] Typing or backspace with an active selection replaces the selected text
- [x] Ctrl+F opens a find bar with case-insensitive incremental search and match highlighting
- [x] Ctrl+G go-to-line dialog
- [x] System clipboard integration — OSC 52 escape sequence for copy (works over SSH), native fallbacks: xclip/xsel (X11), wl-copy/wl-paste (Wayland), pbcopy/pbpaste (macOS). Paste reads system clipboard first, falls back to internal buffer.
- [x] Ctrl+H opens find-and-replace bar
- [x] F3 / Shift+F3 for next/previous match

---

## Phase 4 — Line Numbers, Gutter, and Indentation (partially done)

- [x] Line number gutter on the left edge of each editor pane — dynamic width based on total line count, right-aligned numbers, 1-space left padding + 2-space right padding, `StyleLineNumber` for gutter, `StyleActiveLine` on active line's gutter. Controlled by `lineNumbers` setting (default true).
- [x] Gutter width adjusts dynamically based on total line count
- [x] Auto-indent: new line inherits indentation of the previous line
- [x] Tab/Shift+Tab indent/dedent selected lines
- [ ] Visible whitespace option (show tabs and trailing spaces)
- [ ] Soft wrap toggle (per-buffer setting)

---

## Phase 4b — Tabs and Multi-Buffer Workflow ✅

- [x] Tab bar rendered above the editor area, showing open buffer names
- [x] Ctrl+PgDn / Ctrl+PgUp to cycle tabs
- [x] Click tab to switch
- [x] Ctrl+W close current tab
- [x] Modified indicator on tab (dot)
- [x] Tab overflow: scroll with arrows when too many tabs to fit
- [x] Per-tab undo/redo, cursor, viewport, and selection state

---

## Phase 5 — Split Panes

- [ ] Ctrl+\ vertical split, Ctrl+- horizontal split (or via command palette)
- [ ] Each pane is an independent editor with its own viewport, cursor, and tab bar
- [ ] Panes can show the same buffer (linked cursors are independent, buffer is shared)
- [ ] Focus switching: Ctrl+1/2/3 or click
- [ ] Drag divider to resize (with mouse) or keyboard shortcut to equalize
- [ ] Close pane returns focus to neighbor

---

## Phase 6 — File Explorer Sidebar (partially done)

- [x] Tree view of the working directory, rendered in a fixed-width left panel
- [x] Expand/collapse directories
- [x] Enter or click to open file in active pane
- [x] Ctrl+B toggle sidebar visibility
- [x] Visual indicators: directory chevrons (▶/▼)
- [x] Git changes panel (Ctrl+D) — lists modified/added/deleted files, opens diff in editor; untracked files open as regular files
- [x] Sidebar panel tabs — `PanelTabBarWidget` at top of sidebar with clickable tabs: Files, Search, Changes (title case)
- [ ] Changes panel: tree view mode (`changesView: "list" | "tree"` setting, default list) *(deferred)*
- [ ] Changes panel: show `filename  path/to/dir` with muted directory path as option (useful when multiple files share a name) *(deferred)*
- [x] Highlight the currently open file in the tree — active file highlighted with `StyleSidebarSelected`, synced on every editor status update
- [x] Basic file operations: new file, new folder, rename, delete (with confirmation dialog)
- [ ] File icons (using Unicode/Nerd Font glyphs if available)
- [x] Open folder support — `ttt` opens cwd, `ttt /path/to/dir` opens that directory, `ttt /path/to/file.go` opens file with workspace set to git repo root (falls back to file's parent dir if not in a repo)

---

## Phase 7 — Menu Bar and Command Palette (partially done)

### Menu Bar
- [x] Top-row menu bar: File, Edit, Selection, View, Help
- [x] Keyboard-driven: F10/Alt+F opens File menu, Alt+E/S/V/H for other menus, Left/Right arrows navigate between menus, Up/Down/Enter to select
- [x] Menus show keybinding hints on the right side
- [x] Dropdown menus with actions — File, Edit, Selection, View, Help menus with shortcut display

### Command Palette
- [x] Ctrl+P opens a fuzzy-search dialog listing all available commands
- [x] Commands are registered with name, keybinding, and handler
- [x] Typing filters the list; Enter executes; Esc dismisses
- [ ] Recently used commands float to the top

---

## Phase 8 — Dialogs and Modals (partially done)

- [x] Dialog system: floating, centered panels rendered on top of the editor (z-order)
- [x] Focus is trapped inside the dialog while open; Esc dismisses
- [x] Go to line dialog (Ctrl+G)
- [x] Command palette (modal overlay)
- [x] Find bar (modal overlay)
- [ ] Open file dialog (with path input and file list)
- [ ] Save as dialog
- [x] Confirm dialog (delete file)
- [ ] About/help dialog

---

## Phase 9 — Mouse Support (partially done)

- [x] Click to position cursor
- [x] Click on tab bar to switch tabs
- [x] Click on sidebar entries to open files
- [x] Drag sidebar divider to resize
- [x] Drag bottom panel divider to resize
- [x] Scroll wheel for vertical scrolling (editor, explorer, and changes widgets — 3 lines/items per tick)
- [x] Editor scrollbar — proportional scrollbar on right edge when content exceeds viewport, themed via `StyleScrollbar`/`StyleScrollbarThumb`
- [x] Click+drag to select text
- [x] Double-click to select word, triple-click to select line
- [x] Click on menu bar to open menus
- [x] Right-click context menu — editor (undo/redo/cut/copy/paste/find/replace/go-to-line), tab bar (close/close others/close all), explorer (open/new file/new folder/rename/delete), changes (open diff/open file)

---

## Phase 10 — Syntax Highlighting v2

The current highlighter is single-line regex. This phase makes it real.

- [ ] Language detection from file extension
- [ ] Multi-language support: Go, Python, JavaScript/TypeScript, Rust, C/C++, Markdown, JSON, YAML, TOML, Shell
- [ ] Multi-line constructs: block comments, multi-line strings, heredocs
- [ ] Theme system: named color schemes applied via a config file
- [ ] At minimum ship two themes: a dark theme and a light theme
- [ ] Highlighter runs incrementally (only re-highlight changed lines + propagate state changes)

---

## Phase 11 — Status Bar v2 and Notifications

- [ ] Rich status bar segments: filename, line:col, language mode, encoding (UTF-8), line ending (LF/CRLF), indentation (spaces/tabs + size)
- [ ] Clickable segments (with mouse): click language to change, click encoding to change
- [ ] Transient notification area: "File saved", "No results found", etc. — auto-dismiss after a few seconds

---

## Phase 12 — Configuration (partially done)

- [x] JSON config files: `settings.json`, `theme.json`, `keybindings.json`
- [x] Config search paths: `.config/` (cwd) → `<exe-dir>/config/` → `~/.config/ttt/`
- [x] Theme: explicit per-style defaults (fg/bg/bold) — `defaultFg` (#fafafa) and `defaultBg` (#1f1f1f) in theme config applied as base to all styles for guaranteed contrast; `lineNumber` fg (#999999); removed `accentColor` abstraction in favor of explicit values per `StyleDef`
- [x] Settings: tabSize, insertSpaces, sidebarVisible, sidebarWidth
- [x] .editorconfig support for per-file indent settings
- [ ] Word wrap setting
- [x] Line numbers on/off setting (`lineNumbers`, default true)
- [ ] Font/glyph preferences (Nerd Fonts yes/no)
- [ ] Default encoding and line endings settings
- [ ] Live reload on config file change

---

## Phase 13 — Keybinding System (partially done)

- [x] Key parser: `ParseKeyString("ctrl+shift+f")` into normalized KeyCombo
- [x] Chord support: multi-step key sequences (e.g. `ctrl+k ctrl+c`)
- [x] Command registry: register/lookup/execute commands by ID
- [x] Default keybindings compiled in, user overrides via `keybindings.json`
- [x] Config-driven keybinding loop (no hardcoded key-to-command mapping)
- [ ] When-clause evaluator: conditional keybindings based on context
- [ ] Keybinding hints shown in menus and command palette

### Default keybindings file

Defaults are compiled via `DefaultKeybindings()`. User overrides live at `~/.config/ttt/keybindings.json` (or `.config/keybindings.json`). User entries replace all defaults.

```json
[
  { "key": "ctrl+s",       "command": "file.save" },
  { "key": "ctrl+q",       "command": "editor.quit" },
  { "key": "ctrl+z",       "command": "editor.undo" },
  { "key": "ctrl+y",       "command": "editor.redo" },
  { "key": "ctrl+f",       "command": "search.find" },
  { "key": "ctrl+h",       "command": "search.replace" },
  { "key": "ctrl+shift+f", "command": "search.findInFiles" },
  { "key": "ctrl+shift+p", "command": "commandPalette.open" },
  { "key": "ctrl+b",       "command": "sidebar.toggle" },
  { "key": "ctrl+shift+e", "command": "sidebar.explorer" },
  { "key": "ctrl+shift+g", "command": "sidebar.git" },
  { "key": "ctrl+shift+t", "command": "sidebar.testRunner" },
  { "key": "ctrl+\\",      "command": "editor.splitRight" },
  { "key": "ctrl+-",       "command": "editor.splitDown" },
  { "key": "ctrl+w",       "command": "editor.closePane" },
  { "key": "ctrl+tab",     "command": "editor.nextTab" },
  { "key": "ctrl+shift+tab","command": "editor.prevTab" },
  { "key": "ctrl+g",       "command": "editor.goToLine" },
  { "key": "ctrl+a",       "command": "editor.selectAll" },
  { "key": "ctrl+c",       "command": "editor.copy" },
  { "key": "ctrl+x",       "command": "editor.cut" },
  { "key": "ctrl+v",       "command": "editor.paste" },
  { "key": "f3",           "command": "search.findNext" },
  { "key": "shift+f3",     "command": "search.findPrev" },
  { "key": "escape",       "command": "overlay.dismiss", "when": "overlayOpen" }
]
```

### Chords

A chord is a multi-key sequence: press the first key combo, release, then press the second. VS Code uses these extensively (`Ctrl+K Ctrl+C` to comment, `Ctrl+K Ctrl+U` to uncomment). ttt supports them.

In the JSON config, chords are written with a space between the two key combos:

```json
{ "key": "ctrl+k ctrl+c", "command": "editor.commentLine" },
{ "key": "ctrl+k ctrl+u", "command": "editor.uncommentLine" },
{ "key": "ctrl+k ctrl+s", "command": "file.saveAll" },
{ "key": "ctrl+k ctrl+w", "command": "editor.closeAllTabs" }
```

**How chord dispatch works:**

1. User presses `Ctrl+K`. The keybinding resolver finds that `ctrl+k` is a **chord prefix** (some binding starts with it).
2. The editor enters **chord-pending state**. The status bar shows `(Ctrl+K) was pressed. Waiting for second key...`.
3. User presses `Ctrl+C`. The resolver matches the full chord `ctrl+k ctrl+c` → executes the command.
4. If the second key doesn't match any chord (e.g., user presses `A`), the chord is cancelled, the pending key is discarded, and the `A` is treated normally.
5. If the user presses `Escape` during chord-pending, the chord is cancelled.

The chord state is a simple state machine:

```
Idle → (chord prefix pressed) → ChordPending → (second key) → Idle
                                              → (timeout/escape) → Idle
```

No timeout is enforced by default (VS Code doesn't timeout chords either). The user takes as long as they need.

### Conditional keybindings (`when` clauses)

Some bindings only apply in certain contexts:

- `"when": "editorFocus"` — only when an editor pane has focus
- `"when": "sidebarFocus"` — only when the sidebar has focus
- `"when": "overlayOpen"` — only when a modal/overlay is active
- `"when": "editorLangId == go"` — language-specific bindings

The `when` evaluator is a simple expression parser over a context map that the editor maintains (`{ editorFocus: true, overlayOpen: false, editorLangId: "go", ... }`).

### Command Registry

Every action in the editor is a named command. Keybindings, menu items, command palette entries, and plugin contributions all resolve to commands.

```go
type Command struct {
    ID      string
    Title   string           // shown in command palette and menus
    Handler func(args any)
}

type CommandRegistry struct {
    commands map[string]Command
}
```

The keybinding system resolves key events → command ID → registry lookup → execute handler. This indirection is what makes the whole system pluggable: a plugin registers commands, keybindings map to them, the palette lists them — all through the same registry.

### Implementation order

This should be built as part of Phase 2 (widget framework, step 3 — event routing). The event routing layer needs the keybinding resolver before global shortcuts can work. The steps:

- [ ] Key parser: parse `"ctrl+shift+f"` strings into a normalized key representation
- [ ] Keybinding loader: read defaults.json, merge with user overrides
- [ ] Command registry: register/lookup/execute commands by ID
- [ ] When-clause evaluator: parse and evaluate simple boolean expressions against a context map
- [ ] Integration: the Root's event dispatch checks keybindings before routing to the focused widget
- [ ] Unit tests: key parsing, merge priority (user > default), when-clause evaluation, command dispatch

---

## Phase 14 — Project-Wide Search

- [ ] Ctrl+Shift+F opens project-wide search in the bottom panel
- [ ] Search results grouped by file, showing matching line with context
- [ ] Click result to open file at that line
- [ ] Respect .gitignore patterns
- [ ] Find and replace across files (with preview/confirmation)

---

## Phase 15 — Plugin System

### Two Plugin Tiers

**Tool plugins** — CLI tools invoked on demand. No long-running process. The manifest declares how to call the tool and how to parse its output. Covers search, git, formatting, building, linting — anything stateless.

**Service plugins** — long-running processes for things that need persistent state or need to push events to the editor: LSP servers, file watchers, live diagnostics.

Both tiers use the same manifest format and register through the same system. The editor doesn't care how the plugin is implemented — it cares what the plugin contributes and how to talk to it.

### Communication Model

The editor is the orchestrator. Communication is **bidirectional** with the editor driving the lifecycle:

```
Plugin                                    Editor
  │                                         │
  │──── register ──────────────────────────>│  "I provide these commands,
  │                                         │   I subscribe to these events"
  │                                         │
  │<──── event: textDocument/didSave ───────│  editor pushes subscribed events
  │                                         │
  │──── editor/setDecorations ─────────────>│  plugin calls back to editor API
  │──── editor/showNotification ───────────>│  plugin calls back to editor API
  │                                         │
  │<──── command: go.build ─────────────────│  user triggered a command the
  │                                         │   plugin registered
  │                                         │
  │──── editor/appendOutput ───────────────>│  plugin calls back to editor API
  │                                         │
```

**The four message directions:**

| Direction | Purpose | Example |
|---|---|---|
| Plugin → Editor: **register** | Declare capabilities and event subscriptions | "I handle `go.build`, subscribe me to `onSave` for `.go` files" |
| Editor → Plugin: **event** | Push subscribed events | "user saved `main.go`", "user opened a `.go` file" |
| Editor → Plugin: **command** | Invoke a command the plugin registered | "user pressed Ctrl+Shift+B → `go.build`" |
| Plugin → Editor: **callback** | Call editor APIs to affect the UI | "show notification", "set decorations", "open file" |

The plugin never touches the UI directly. It calls editor APIs, and the editor decides how to render the result.

### Tool Plugins (CLI-based)

Tool plugins don't run persistently. The manifest declares CLI templates. The editor spawns the tool, captures output, and parses it.

```json
{
  "name": "search",
  "type": "tool",
  "commands": {
    "search.findInFiles": {
      "exec": "rg",
      "args": ["--json", "--smart-case", "${query}", "${workspaceDir}"],
      "output": "json-lines",
      "presentation": "searchResults"
    },
    "format.file": {
      "exec": "gofmt",
      "args": ["-w", "${filePath}"],
      "output": "none",
      "onComplete": "reloadFile"
    }
  }
}
```

Template variables (`${query}`, `${workspaceDir}`, `${filePath}`, `${selection}`) are resolved by the editor before spawning.

Output formats the editor knows how to parse:
- `json` — single JSON object
- `json-lines` — one JSON object per line (rg, go test -json)
- `lines` — plain text, one result per line
- `none` — ignore output (side-effect-only tools like formatters)
- `stream` — stream output to the bottom panel in real time (build, test)

No registration handshake needed — the manifest IS the registration. This means existing CLI tools work as plugins with zero code: just write a `plugin.json` declaring how to call them.

### Service Plugins (long-running, JSON-RPC)

For plugins that need persistent state or push events. Communication is JSON-RPC 2.0 over stdin/stdout.

```json
{
  "name": "go-lsp",
  "type": "service",
  "runtime": "go",
  "entry": "gopls",
  "activationEvents": ["onLanguage:go"],
  "contributes": {
    "commands": [
      { "id": "go.build", "title": "Go: Build Package" },
      { "id": "go.test", "title": "Go: Run Tests" }
    ],
    "sidebar": [
      { "id": "go.testExplorer", "title": "Test Explorer", "icon": "T" }
    ],
    "keybindings": [
      { "command": "go.build", "key": "ctrl+shift+b" }
    ],
    "events": ["textDocument/didOpen", "textDocument/didChange", "textDocument/didSave"]
  }
}
```

**Lifecycle:**

1. **Activation.** An activation event fires (file with `.go` extension opened). The editor spawns the plugin process.
2. **Handshake.** Editor sends `initialize` with workspace path, capabilities, and config. Plugin responds with its capabilities.
3. **Registration.** Plugin sends `register` messages declaring commands and subscribing to events. (Alternatively, the manifest's `contributes` and `events` fields serve as static registration, and the plugin can dynamically add more at runtime.)
4. **Steady state.** Editor pushes events. Plugin calls back to editor APIs. User triggers commands.
5. **Shutdown.** Editor sends `shutdown`, waits for acknowledgment, then kills the process.

**Events the editor can push (plugin subscribes via manifest or `register`):**

```
textDocument/didOpen      file opened: uri, languageId, content
textDocument/didChange    buffer edited: uri, changes
textDocument/didSave      file saved: uri
textDocument/didClose     file closed: uri
workspace/didChangeFiles  files created/deleted/renamed on disk
configuration/didChange   settings changed relevant to this plugin
```

**Editor APIs the plugin can call back to:**

```
editor/applyEdit          modify buffer contents
editor/showNotification   display a transient message
editor/showInputBox       prompt user for text, returns response
editor/showQuickPick      show a selection list, returns chosen item
editor/setDecorations     inline hints, squiggles, gutter markers
editor/registerTreeView   provide data for a sidebar tree panel
editor/updateTreeView     refresh sidebar tree data
editor/setStatusBarItem   add/update a status bar segment
editor/openFile           open a file in the editor
editor/revealRange        scroll to and highlight a range
editor/appendOutput       write to a named output channel in the bottom panel
diagnostics/publish       errors/warnings for a file (gutter + problems panel)
```

### Runtime Types

| Runtime | Entry | How it's spawned |
|---------|-------|-----------------|
| `go` | compiled binary path | exec directly |
| `node` | `.js` file | `node <entry>` |
| `deno` | `.ts` file | `deno run <entry>` |
| `python` | `.py` file | `python3 <entry>` |

### Extension Points (what plugins can contribute)

| Contribution | Description |
|---|---|
| **Commands** | Appear in command palette, can be bound to keys and menus |
| **Languages** | File extension → language ID mapping, syntax highlighting rules |
| **Sidebar panels** | New entries in the activity bar with tree/list views |
| **Bottom panel tabs** | Output channels, test results, diagnostics list |
| **Themes** | Color schemes (token → style mappings) |
| **Status bar items** | Segments in the status bar (e.g., branch name, linter status) |
| **Keybindings** | Additional key → command mappings |
| **Menu items** | Entries in the menu bar or context menus |
| **Editor decorations** | Inline hints, squiggles, gutter icons |
| **File decorations** | Icons/colors for files in the explorer tree |

### Built-in Features as Plugins

To keep the editor core small and validate the plugin API, several "built-in" features are implemented as internal plugins that use the same interfaces (compiled into the binary but designed as if they were external):

- **Explorer** — file tree sidebar. Uses the tree view API.
- **Search** — project-wide search. Tool plugin wrapping `rg`.
- **Git** — source control sidebar. Tool plugin wrapping `git`.
- **Test runner** — discovers and runs tests. Tool plugin wrapping `go test -json`, `npm test`, etc.

The activity bar panels from the UX mockups (Explorer, Search, Git, Test) are all plugin-provided. The editor core only knows about the widget system and the plugin host.

### Plugin Host

```go
type PluginHost struct {
    tools      map[string]*ToolManifest       // tool plugins (CLI-based)
    services   map[string]*ServiceProcess     // running service plugins
    manifests  map[string]*Manifest           // all loaded manifests
    pending    map[string]*Manifest           // loaded but not yet activated
}
```

Responsibilities:
- Scan `~/.config/ttt/plugins/` and parse manifests on startup
- For tool plugins: resolve CLI templates and spawn/parse on command invocation
- For service plugins: match activation events, spawn process, manage lifecycle
- Route events to subscribed plugins
- Route command invocations to the plugin that registered them
- Restart crashed service plugins (with backoff)
- Provide a mock plugin host for testing

### Plugin Permissions

Plugins declare what they need in the manifest. On install, the editor shows the user exactly what the plugin is requesting. The user approves or denies. Approved permissions are cached in `~/.config/ttt/plugin-permissions.json` so the prompt only appears once per plugin (or on upgrade if new permissions are requested).

**The install prompt:**

```
Installing "go-language" v0.1.0...

This plugin requests:
  ✓ Read file contents         (textDocument/didOpen, didChange)
  ✓ Run command: gopls         (exec)
  ✓ Modify buffers             (editor/applyEdit)
  ✓ Show UI elements           (sidebar panel, status bar item)
  ✗ Access filesystem
  ✗ Execute arbitrary commands
  ✗ Network access

Allow? [Y/n]
```

**Permission tiers:**

| Tier | What it covers | Behavior |
|---|---|---|
| **Read** | Receive events about open files, cursor, selections | Auto-granted — harmless, the plugin is just being informed |
| **UI** | Show notifications, status bar items, sidebar panels, decorations, output | Auto-granted — the plugin can show things but can't change data |
| **Write** | Modify buffer contents (`editor/applyEdit`) | Prompt on install |
| **Exec** | Spawn child processes — tool plugins declare exact binaries, service plugins run their entry | Prompt on install, shows the exact binary name |
| **Filesystem** | Read or write files beyond the currently open buffers | Prompt on install |
| **Network** | Outbound HTTP/TCP connections | Prompt on install |

The key insight: **most events are harmless** (cursor moved, file opened). The dangerous part is what the plugin can *do back to the editor*. So permissions gate the callbacks, not the events.

**Manifest declares permissions explicitly:**

```json
{
  "name": "go-language",
  "permissions": ["read", "ui", "write", "exec:gopls"],
  ...
}
```

The `exec` permission names the specific binary. A plugin requesting `exec:gopls` can only spawn `gopls`, not arbitrary commands. A plugin requesting `exec:*` (any command) gets a scarier prompt.

**Permission storage (`~/.config/ttt/plugin-permissions.json`):**

```json
{
  "go-language@0.1.0": {
    "granted": ["read", "ui", "write", "exec:gopls"],
    "denied": [],
    "grantedAt": "2026-05-18T12:00:00Z"
  }
}
```

**Version upgrades:** if a plugin updates and requests new permissions it didn't have before, the user is prompted again — but only for the new permissions. Existing grants carry over.

**Revocation:** the user can revoke permissions at any time via command palette (`Plugin: Manage Permissions`) or by editing the JSON file. Revoking a permission from a running service plugin sends it a `permissionRevoked` notification so it can degrade gracefully.

**Runtime enforcement:** the plugin host checks permissions before executing any callback. If a plugin calls `editor/applyEdit` without `write` permission, the call is rejected with an error and a notification is shown: `"go-language" tried to modify a buffer but doesn't have write permission.`

**Built-in plugins** (Explorer, Search, Git, Test) are trusted and skip the permission prompt. They're compiled into the binary and reviewed as part of the editor's own code.

### Plugin Development

**Tool plugin (zero code — just a manifest):**
```json
{
  "name": "my-formatter",
  "type": "tool",
  "commands": {
    "myFormatter.format": {
      "exec": "prettier",
      "args": ["--write", "${filePath}"],
      "output": "none",
      "onComplete": "reloadFile"
    }
  }
}
```

**Service plugin (Go):**
```go
// main.go — reads/writes JSON-RPC on stdin/stdout
func main() {
    conn := jsonrpc.NewStdioConn()
    conn.HandleRequest("initialize", onInitialize)
    conn.HandleNotification("textDocument/didSave", onSave)
    conn.Serve()
}
```

**Service plugin (TypeScript):**
```typescript
// index.ts — using @ttt/plugin-api for typed helpers
import { createPlugin } from "@ttt/plugin-api";

const plugin = createPlugin();
plugin.onSave(async (doc) => {
    const diagnostics = await lint(doc);
    plugin.publishDiagnostics(doc.uri, diagnostics);
});
plugin.start();
```

### What This Means for Earlier Phases

The plugin system is Phase 15, but it shapes earlier design:

- **Phase 8 (command palette)**: the command registry should accept dynamic registration from day one so plugins can add commands later.
- **Phase 7 (sidebar)**: the activity bar and sidebar panel system should use a generic tree/list view widget, not hard-code Explorer. When plugins arrive, Explorer is just a plugin providing tree data.
- **Phase 11 (highlighting)**: language support should be a pluggable provider, even if initially built-in.
- **Status bar**: segments should be a generic list that core or plugins can add to.

Design for these extension points from the start. The plugin host comes later; the seams should already be there.

---

## Git Worktree Workspaces

Git worktrees are powerful but the UX is terrible — raw CLI commands, no editor understands them natively. VS Code treats each worktree as a separate window with no connection between them. ttt makes worktrees a first-class workspace concept.

This should be implementable as a plugin, not a core feature. The plugin API already provides everything needed: sidebar tree views, commands, status bar items, `editor/openFile` for cross-directory file access, and `exec:git` permission for worktree operations. If it can't be built as a plugin, that's a signal the plugin API is missing something.

### Features

- **Worktree sidebar panel** — activity bar entry showing all worktrees for the current repo. Each entry shows branch name, dirty state, and path.
- **Create worktree** — command palette: "Worktree: New from Branch". Picks a branch (or creates one), runs `git worktree add`, opens it.
- **Switch worktree** — click in sidebar or command palette. Opens the worktree's directory as the active workspace, preserving the other worktree's state.
- **Cross-worktree diff** — compare the same file across two worktrees side by side. Each pane shows the file from its own branch, both editable. No stashing, no branch switching.
- **Worktree status overview** — see dirty state, unpushed commits, and CI status across all active worktrees in one view.
- **Clean up** — list stale worktrees, delete them with one command. No remembering `git worktree remove --force`.

### Workflow

You're working on a feature. A bug report comes in. You hit a keybinding → "Worktree: New from Branch" → type the hotfix branch name → ttt creates the worktree and opens it in a split pane. You fix the bug, commit, push, close the worktree pane. Your feature branch is untouched — no stash, no interrupted state.

### Core API requirement

One thing the core needs to support: `editor/openFile` must accept absolute paths outside the current workspace root. A worktree plugin needs to open files from sibling worktree directories. This is a reasonable core capability regardless (useful for any plugin that references external files) and falls under the `filesystem` permission tier.

---

## Future Ideas (unscoped)

- Integrated terminal panel (bottom panel runs a shell)
- Git integration: gutter indicators for added/modified/deleted lines, branch name in status bar
- Multi-cursor editing (Ctrl+D to select next occurrence)
- Minimap
- Bracket matching and auto-close
- LSP client for autocomplete, diagnostics, go-to-definition
- Session restore (reopen last files and layout)
- Snippet support
- Macro recording and playback
- Plugin marketplace / registry

---

## Design Principles

1. **Core stays terminal-free.** `internal/core/` must compile and test without tcell. All terminal interaction goes through the `Screen` interface.
2. **Composition over inheritance.** Windows, panes, dialogs, and sidebar are all composed from the same primitives: Rect, Viewport, Buffer, Renderer.
3. **Test without a terminal.** MockScreen exists for this reason. New UI components should be testable against it.
4. **Render efficiently.** The diff-based renderer exists from day one. Never rebuild the full screen when a partial update will do.
5. **Rune-correct.** Cursor positions, selections, and display widths are always rune-based, not byte-based. CJK and emoji support is a goal, not an afterthought.
