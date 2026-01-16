# IMPLEMENTATION.md

A comprehensive, step-by-step implementation plan for the Go Terminal Editor. Each step is atomic and can be resumed or executed independently by an LLM or developer. Follow the order for best results.

---

## 0. Project Setup

1. Initialize a new Go module in the project root.
2. Create the directory structure as described in `PROJECT.md` (see architecture diagram).
3. Add a `.gitignore` for Go projects.
4. Set up a minimal `main.go` in `cmd/editor/` that prints "Hello, editor!".

---

## 1. Core Buffer Implementation

5. Create `internal/core/buffer/buffer.go` and define the `Buffer` struct with `Lines []string` and `Dirty bool`.
6. Implement `InsertRune(line, col, r)`, `DeleteRune(line, col)`, `InsertLine(idx, text)`, and `DeleteLine(idx)` methods for `Buffer`.
7. Write unit tests for all buffer operations.

---

## 2. Cursor Implementation

8. Create `internal/core/cursor/cursor.go` and define the `Cursor` struct (`Line`, `Col`, `Goal`).
9. Implement cursor movement methods (left, right, up, down) with visual column logic and goal preservation.
10. Write unit tests for cursor movement and edge cases.

---

## 3. Undo System

11. Define the `EditCommand` interface in `internal/core/undo/undo.go`.
12. Implement command types: `InsertRuneCommand`, `DeleteRangeCommand`, `InsertLineCommand`.
13. Implement the `UndoStack` struct with undo/redo stacks and methods.
14. Write unit tests for undo/redo functionality.

---

## 4. Syntax Highlighting (Phase 1)

15. Create `internal/core/highlight/highlighter.go` and define `Span` and `Highlighter` interface.
16. Implement a simple regex-based highlighter for comments, strings, and keywords.
17. Write unit tests for the highlighter.

---

## 5. Viewport

18. Create `internal/view/viewport.go` and define the `Viewport` struct.
19. Implement logic for vertical/horizontal scrolling, soft wrap (optional), and cursor-to-screen mapping.
20. Write unit tests for viewport logic.

---

## 6. Terminal Abstraction

21. Create `internal/term/screen.go` and define `Cell` and `Screen` interface.
22. Implement a minimal `tcell`-based backend (or stub for testing).
23. Write a test/mock implementation for `Screen`.

---

## 7. Renderer

24. Create `internal/render/renderer.go` and define the `Renderer` struct with `prev` and `curr` buffers.
25. Implement diffing and minimal update emission.
26. Write unit tests for the renderer diff logic.

---

## 8. Event Loop

27. In `main.go`, implement the event loop: read events, dispatch to core, update viewport, render.
28. Handle key events and resize events.
29. Integrate buffer, cursor, viewport, and renderer.

---

## 9. File I/O

30. Implement file open and save logic in the editor.
31. Add basic error handling for file operations.

---

## 10. Status Bar

32. Implement a simple status bar (e.g., file name, cursor position, dirty flag).
33. Render the status bar in the main event loop.

---

## 11. Multiple Buffers (Optional, after core is solid)

34. Add support for multiple open buffers.
35. Implement buffer switching logic.

---

## 12. Split Panes and Windows (Optional, after core is solid)

36. Implement the `Window` struct and `WindowManager` as described in `PROJECT.md`.
37. Add support for split panes and window focus.
38. Implement modal dialogs as special windows.

---

## 13. Final Polish

39. Refactor code for clarity and maintainability.
40. Add documentation and usage instructions.
41. Review and expand test coverage.

---

## 14. Next Steps

- Consider advanced syntax highlighting (multi-line, language-specific).
- Add configuration and keybinding support.
- Package and distribute the editor.

---

Each step is atomic and can be resumed or executed independently. LLMs or developers should mark each step as complete before proceeding to the next.
