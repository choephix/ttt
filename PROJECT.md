# Go Terminal Editor — Project Plan

This is a concrete, opinionated starting architecture for a Go terminal editor. It begins with text + highlighting and is deliberately shaped so you can later add Turbo Vision–style panes, dialogs, and movable windows without rewriting the core.

This is not abstract theory — this is something you could start coding tonight.

---

## 1. High-level architecture (non-negotiable split)

Keep the repository layout simple and modular so the core can be tested without a terminal:

```
cmd/editor/
  main.go

internal/
├── core/        ← editor engine (UI-agnostic)
│   ├── buffer/
│   ├── cursor/
│   ├── undo/
│   └── highlight/
│
├── view/        ← viewport logic (scrolling, wrapping)
│   └── viewport.go
│
├── ui/          ← windows, panes, dialogs (later)
│   ├── window.go
│   ├── manager.go
│   └── events.go
│
├── term/        ← terminal backend
│   ├── screen.go
│   └── input.go
│
└── render/      ← diff-based renderer
    └── renderer.go
```

Rule #1: `core/` must compile and be testable without a terminal. That’s how you avoid painting yourself into a corner.

---

## 2. Editor core (start here)

### 2.1 Buffer (keep it boring)

Start with line-oriented storage. Don’t optimize prematurely.

```go
// core/buffer/buffer.go
type Buffer struct {
    Lines []string
    Dirty bool
}

// Operations to implement:
// InsertRune(line, col, r)
// DeleteRune(line, col)
// InsertLine(idx, text)
// DeleteLine(idx)
```

Ropes and gap buffers are better in some cases, but a lines-first approach is simple and good enough for an initial version.

### 2.2 Cursor (visual, not byte-based)

```go
// core/cursor/cursor.go
type Cursor struct {
    Line int
    Col  int // visual column
    Goal int // preserved for vertical movement
}
```

Key rules:

- `Col` is a visual width, not a byte index.
- Preserve `Goal` on up/down movements.
- Clamp `Col` safely when line lengths change.

This is where many editors subtly break — get this right early.

### 2.3 Undo (command-based)

Use command objects instead of snapshotting buffers:

```go
type EditCommand interface {
    Apply(b *Buffer)
    Undo(b *Buffer)
}

// Examples: InsertRuneCommand, DeleteRangeCommand, InsertLineCommand

type UndoStack struct {
    undo []EditCommand
    redo []EditCommand
}
```

Command-based undo scales cleanly when panes and windows are added.

---

## 3. Highlighting (keep it dumb first)

Phase 1: regex-based, per-line highlighting. No multi-line state yet.

```go
// core/highlight/highlighter.go
type Span struct {
    Start int
    End   int
    Style Style
}

type Highlighter interface {
    Highlight(line string) []Span
}
```

Start with support for:

- comments
- strings
- keywords

This is intentionally simple so you can swap in a better highlighter later without touching rendering.

---

## 4. Viewport (the bridge)

The viewport translates the buffer into visible cells.

```go
// view/viewport.go
type Viewport struct {
    TopLine int
    LeftCol int
    Width   int
    Height  int
}
```

Responsibilities:

- Vertical and horizontal scrolling
- Optional soft wrap
- Mapping cursor → screen coordinates

Start with a single-pane viewport; it becomes pane-aware later.

---

## 5. Terminal abstraction (thin on purpose)

Use a library like `tcell`, but hide it behind a minimal interface in `term/` so the rest of the code never imports `tcell` directly.

```go
// term/screen.go
type Cell struct {
    Ch    rune
    Style Style
}

type Screen interface {
    Size() (w, h int)
    SetCell(x, y int, c Cell)
    Show()
    Clear()
}
```

The terminal backend should be narrow and replaceable.

---

## 6. Renderer (diff-based from day one)

Never redraw the whole screen if you can avoid it.

```go
// render/renderer.go
type Renderer struct {
    prev [][]Cell
    curr [][]Cell
}

// Flow:
// - Build curr
// - Diff against prev
// - Emit minimal updates
// - Swap buffers
```

This approach keeps the editor fast and usable over SSH.

---

## 7. Event loop (simple, deterministic)

Keep it straightforward and single-threaded at first.

```go
for {
    ev := term.ReadEvent()

    switch ev := ev.(type) {
    case KeyEvent:
        core.HandleKey(ev)
    case ResizeEvent:
        viewport.Resize(...)
    }

    renderer.Render()
}
```

No goroutines yet — get correctness before concurrency.

---

## 8. How this grows into Turbo Vision later

When the core is stable, add windows and a manager without changing the fundamentals.

```go
// ui/window.go
type Window struct {
    Rect Rect
    View *Viewport
    Buf  *Buffer
}

// ui/manager.go
type WindowManager struct {
    Windows [] *Window
    Focus   int
}
```

Then:

- Each window owns a viewport
- Renderer clips by window rect
- Input is routed by focus
- Dialogs are windows with modal flags

No rewrite required — just composition.

---

## 9. Milestones (realistic and motivating)

- Milestone 1 (1–2 weeks)

  - Open file
  - Edit text
  - Move cursor
  - Basic highlighting

- Milestone 2 (3–4 weeks)

  - Undo/redo
  - Scrolling
  - Status bar
  - Save file

- Milestone 3 (after core is solid)
  - Multiple buffers
  - Split panes (tiled)
  - THEN: floating windows / dialogs

## 10. One hard-earned warning ⚠️

Do not implement movable windows before editing feels good. Turbo Vision worked because the editor core was solid. Many modern TUIs fail because they invert this priority.

---

## Final takeaway

You now have:

- A Go-first architecture
- A path from a simple editor → Turbo Vision–style UI
- Minimal, non-throwaway core code
- Clear stopping points if life gets busy

If you want next, I can:

- Write a minimal `main.go` skeleton
- Implement the buffer + cursor code with you
- Show how to add the first dialog safely
- Help you pick exact libraries + versions

Say which piece you want to build first.
