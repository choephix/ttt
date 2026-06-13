# Code Folding Spec

## Overview

Add code folding to the editor, allowing users to collapse and expand regions of code. Fold ranges come from two sources: LSP `textDocument/foldingRange` (when available) and indentation-based fallback.

## Fold Range Sources

### 1. Indentation-based (fallback, always available)

A fold range starts on any line whose next non-blank line has greater indentation. The range extends until indentation returns to the starting level or lower. Minimum fold size: 2 lines.

```
func main() {        ← fold starts here (indent 0)
    fmt.Println()    ← indent 1
    if true {        ← nested fold starts here (indent 1)
        doStuff()    ← indent 2
    }                ← nested fold ends here
}                    ← outer fold ends here
```

### 2. LSP `textDocument/foldingRange` (preferred when available)

Request fold ranges from the language server after document open and on document change (debounced). Falls back to indentation if the server doesn't support `foldingRangeProvider`.

## Data Model

### New package: `internal/core/fold/`

```go
type Range struct {
    StartLine int // 0-based, inclusive (the line with the fold marker)
    EndLine   int // 0-based, inclusive (last line hidden when collapsed)
}

type State struct {
    ranges    []Range // sorted by StartLine
    collapsed map[int]bool // keyed by StartLine
}
```

Key methods:
- `SetRanges(ranges []Range)` — replace all ranges (from LSP or indentation recompute)
- `Toggle(line int)` — collapse/expand the range starting at or containing `line`
- `CollapseAll()` / `ExpandAll()`
- `IsCollapsed(startLine int) bool`
- `FoldAt(line int) *Range` — returns the range starting at this line, if any
- `VisibleLines(totalLines int) []int` — returns ordered list of visible buffer line indices
- `BufferToScreen(bufLine int) int` — map buffer line to screen line
- `ScreenToBuffer(screenLine int) int` — map screen line to buffer line
- `HiddenCount() int` — total hidden lines (for scrollbar calculation)

## Rendering Changes

### Editor widget (`editor_widget.go`)

The render loop currently does `lineIdx := Viewport.TopLine + y`. With folding:

1. Pre-compute visible line list from `fold.State.VisibleLines()`
2. `TopLine` becomes an index into the visible line list, not the buffer
3. For each screen row `y`, look up `visibleLines[topLineIdx + y]` to get the buffer line
4. If a line starts a collapsed fold, render it normally but append ` ... (+N lines)` after the line content
5. Fold chevrons appear in the gutter, to the right of line numbers:
   - `⏵` on collapsed fold start lines — **always visible**
   - `⏷` on expanded fold start lines — **only visible when mouse hovers over the gutter column**
   - No indicator on non-fold lines
6. The editor must track `gutterHover bool` and `gutterHoverY int` from mouse move events to know when to show expanded chevrons
7. Clicking a chevron toggles the fold

### Viewport changes (`view/viewport.go`)

- `TopLine` stays as buffer line index for persistence
- Add `VisibleLineIndex(bufLine int, foldState) int` to convert
- `CursorScreenCoords` must account for hidden lines
- Scrollbar thumb size and position must reflect visible line count vs total

### Cursor behavior

- Cursor never lands on a hidden line
- `MoveDown` from line above a collapsed range jumps to the line after the range
- `MoveUp` from line below a collapsed range jumps to the fold start line
- Typing on the fold start line expands the fold
- Clicking the chevron in the gutter toggles the fold
- Mouse hover over the gutter reveals chevrons on expanded fold lines; collapsed chevrons are always visible

## Commands

| Command ID | Title | Keybinding | Action |
|---|---|---|---|
| `fold.toggle` | Toggle Fold | Ctrl+Shift+[ | Toggle fold at cursor line |
| `fold.collapseAll` | Fold All | Ctrl+Shift+0 | Collapse all fold ranges |
| `fold.expandAll` | Unfold All | Ctrl+Shift+9 | Expand all fold ranges |

## LSP Integration

### Protocol additions (`lsp/protocol.go`)

```go
type FoldingRange struct {
    StartLine      int    `json:"startLine"`
    StartCharacter int    `json:"startCharacter,omitempty"`
    EndLine        int    `json:"endLine"`
    EndCharacter   int    `json:"endCharacter,omitempty"`
    Kind           string `json:"kind,omitempty"` // "comment", "imports", "region"
}

type FoldingRangeParams struct {
    TextDocument TextDocumentIdentifier `json:"textDocument"`
}
```

### Client method (`lsp/client.go`)

```go
func (c *Client) FoldingRange(uri string) ([]FoldingRange, error)
```

### Integration flow

1. On document open: request fold ranges (async via PostEvent)
2. On document change: re-request fold ranges (debounced, same as completions)
3. Merge LSP ranges with fold state, preserving existing collapsed state

## Edge Cases

- **Search**: search highlights and jump-to-match must expand the containing fold
- **Go to line**: auto-expand the fold containing the target line
- **Go to definition**: auto-expand the fold at the destination
- **Selection**: selecting across a collapsed fold expands it
- **Copy/paste**: copying a collapsed fold copies all hidden lines
- **Undo/redo**: undo that modifies folded lines expands the fold
- **Diagnostics**: error on a hidden line auto-expands the fold (or shows indicator on fold line)
- **Multi-cursor**: cursors on hidden lines expand their containing folds
- **Word wrap**: only affects visible lines, no interaction

## Implementation Order

1. `internal/core/fold/` — data model and line mapping logic
2. Unit tests for fold state
3. Wire fold state into editor widget render loop
4. Indentation-based fold range computation
5. Fold toggle command + keybinding
6. Cursor navigation with folds
7. E2E tests
8. Fold all / expand all commands
9. LSP folding range integration
10. Functional tests
11. Edge cases (search expand, go-to-line expand, diagnostics)

---

## Test Plan

### Unit Tests (`internal/core/fold/fold_test.go`)

#### State management
- `TestSetRanges` — setting ranges sorts them by StartLine
- `TestToggle` — toggling a fold line flips collapsed state
- `TestToggleNonFoldLine` — toggling a non-fold line is a no-op
- `TestCollapseAll` — all ranges become collapsed
- `TestExpandAll` — all ranges become expanded
- `TestNestedRanges` — collapsing outer hides inner; expanding outer restores inner visibility

#### Line mapping
- `TestVisibleLines_NoFolds` — returns all lines 0..N-1
- `TestVisibleLines_OneFold` — collapsed range lines excluded from result
- `TestVisibleLines_FoldStartVisible` — fold start line is always visible
- `TestVisibleLines_MultipleFolds` — multiple collapsed ranges excluded correctly
- `TestVisibleLines_NestedFolds` — outer fold hides inner fold lines too
- `TestBufferToScreen` — buffer line maps to correct screen line with collapsed ranges
- `TestScreenToBuffer` — screen line maps to correct buffer line
- `TestHiddenCount` — returns total number of hidden lines

#### Indentation-based ranges
- `TestIndentRanges_Function` — function body creates a fold range
- `TestIndentRanges_Nested` — nested blocks create nested ranges
- `TestIndentRanges_BlankLines` — blank lines don't break fold ranges
- `TestIndentRanges_MinSize` — single-line indentation increase doesn't create a fold
- `TestIndentRanges_Tabs` — tab-indented code detected correctly
- `TestIndentRanges_FlatFile` — file with no indentation changes produces no ranges

### E2E Tests (`tests/e2e/fold_test.go`)

#### Rendering
- `TestFoldToggle_RendersFoldMarker` — collapsed fold shows `⏵` in gutter (always visible) and `...` on the line
- `TestFoldToggle_HidesLines` — lines inside collapsed fold not in screen text
- `TestFoldExpand_ShowsLines` — expanding fold restores hidden lines
- `TestFoldAll_CollapsesAllRanges` — fold all command hides all foldable regions
- `TestExpandAll_ShowsAllLines` — expand all restores all lines
- `TestFoldLineNumbers` — buffer line numbers shown correctly (not renumbered)

#### Cursor navigation
- `TestCursorDown_SkipsFold` — cursor down from above fold jumps past collapsed range
- `TestCursorUp_SkipsFold` — cursor up from below fold jumps to fold start line
- `TestCursorOnFoldStart_CanEdit` — typing on fold start line works normally

#### Scrolling
- `TestScroll_SkipsFoldedLines` — page down skips collapsed ranges
- `TestScrollbar_ReflectsFolds` — scrollbar size accounts for hidden lines

#### Interactions
- `TestGoToLine_ExpandsFold` — go-to-line targeting a hidden line expands its fold
- `TestSearch_ExpandsFold` — search match inside a fold expands it

### Functional Blackbox Tests (`tests/functional/code-folding.test.js`)

#### Basic folding
- `should show fold chevrons on gutter hover` — open a Go file, hover over gutter, verify `⏷` appears on foldable lines
- `should hide fold chevrons when not hovering gutter` — move mouse away from gutter, verify chevrons disappear
- `should collapse a fold with Ctrl+Shift+[` — press keybinding on a function line, verify body disappears
- `should expand a collapsed fold with Ctrl+Shift+[` — toggle again, verify body reappears
- `should always show chevron on collapsed folds` — collapsed fold shows `⏵` even without hover
- `should show hidden line count on collapsed fold` — collapsed fold line shows `(+N lines)`

#### Fold all / expand all
- `should collapse all folds with Ctrl+Shift+0` — all function bodies hidden
- `should expand all folds with Ctrl+Shift+9` — all function bodies restored

#### Cursor navigation
- `should skip folded lines when pressing down arrow` — cursor jumps past collapsed range
- `should navigate correctly with multiple folds` — cursor moves between visible lines only

#### Persistence within session
- `should preserve fold state when switching tabs and back` — fold a region, switch to another tab, switch back, fold still collapsed
