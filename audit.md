# UX Bug Audit

Started 2026-07-11 on branch `audit/bug-hunt`. Goal: **discover and document** UX bugs — no fixes on this branch. Each confirmed finding gets a ledger entry below and, where feasible, a repro test marked as expected-failure (`t.Skip("BUG-NNN")` in Go, `it.fails(...)` in vitest) so the suite stays green until the bug is fixed.

Process: one hunting agent at a time, scoped to an area from the coverage matrix. Orchestrator re-verifies every repro before it enters this file. LSP is out of scope.

## Coverage matrix

| Area | Status | Findings |
|---|---|---|
| Editing commands × selection | swept (4 findings) | BUG-001..004 |
| Multicursor interactions | swept (4 findings) | BUG-005..008 |
| Undo/redo semantics | swept (6 findings) | BUG-020..025 |
| Code folding × editing | swept (2 findings) | BUG-026, BUG-027 |
| Find/replace + search highlights | swept (6 findings) | BUG-010..015 |
| Tabs & split panes | swept (1 finding; split panes N/A — feature doesn't exist) | BUG-016 |
| Explorer (file tree) | in progress | |
| Global search (sidebar, rg-based) | pending | |
| Mouse targets / click offsets | swept (2 findings) | BUG-018, BUG-019 |
| Resize & layout | pending | |
| Wide-char / edge content (CJK, emoji, tabs, long lines) | swept (1 finding) | BUG-009 |
| Keyboard navigation parity | partial (orchestrator probe, not a full sweep) | BUG-017 |
| Themes & rendering | pending | |
| Settings & options | pending | |
| Workspace (multi-folder) | pending | |
| Integrated terminal panel | pending (do last) | |
| Plugin widgets | pending | |

Status values: `pending` → `in progress` → `swept (N findings)` / `swept (clean)`.

## Findings

### BUG-001: Move Line Up/Down includes trailing col-0 selection line and swaps the invisible trailing empty line into the buffer
- **Area:** Editing commands × selection
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** file `line0\nline1\nline2\nline3\nline4\n`; `bin/ttt --size 120x40 --exec "wait 200; key down; key down; key shift+down; key shift+down; exec \"Move Line Down\"; screenshot /tmp/s.txt; quit" file.txt`
- **Expected:** selection line2→line4-col0 covers lines 2–3 (col-0 convention per JoinLines/ToggleLineComment); block swaps past line4 → `line0,line1,line4,line2,line3`
- **Actual:** a blank line appears between line1 and line2 (visible buffer grows 5→6 rows): `MoveLineDown`/`MoveLineUp` (`internal/ui/editor_widget_lines.go:33-54`) apply no col-0 adjustment, and the EOF guard uses the internal line count, which includes the invisible trailing empty line of any `\n`-terminated file — so that phantom line gets swapped into the middle of the file. Buffer marked modified; undo does restore correctly.
- **Test:** `tests/functional/audit-selection-bugs.test.js` (`it.fails`)

### BUG-002: Indent (Tab) with selection ending at col 0 indents one line too many
- **Area:** Editing commands × selection
- **Severity:** medium
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** file `line0\nline1\nline2\nline3\nline4\n`; `bin/ttt --size 120x40 --exec "wait 200; key down; key shift+down; key tab; screenshot /tmp/s.txt; quit" file.txt`
- **Expected:** selection line1→line2-col0 covers only line1 (control: `Toggle Line Comment` with the identical selection correctly comments only line1) → only line1 indented
- **Actual:** line1 AND line2 both indented — the KeyTab handler (`internal/ui/editor_widget_keyboard.go:238-247`) iterates `start.Line..end.Line` with no col-0 exclusion
- **Test:** `tests/functional/audit-selection-bugs.test.js` (`it.fails`)

### BUG-003: Duplicate Line and Delete Line ignore an active multi-line selection
- **Area:** Editing commands × selection
- **Severity:** medium
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** file `line0\nline1\nline2\nline3\nline4\n`; `bin/ttt --size 120x40 --exec "wait 200; key down; key shift+down; key shift+down; exec \"Duplicate Line\"; screenshot /tmp/s.txt; quit" file.txt` (same shape with `exec "Delete Line"`)
- **Expected:** per the project convention ("line-based commands operate on the selected lines"), with lines 1–2 selected: Duplicate Line duplicates the block; Delete Line deletes it
- **Actual:** `DuplicateLine()`/`DeleteLine()` (`internal/ui/editor_widget_lines.go:56-83`) never consult the selection — both act only on the cursor's line (line3, not even part of the selection per the col-0 convention). Delete Line additionally leaves a stale selection range pointing past the shifted buffer.
- **Test:** `tests/functional/audit-selection-bugs.test.js` (`it.fails`, 2 cases)

### BUG-004: Outdent (Backtab) with selection ending at col 0 outdents one line too many
- **Area:** Editing commands × selection
- **Severity:** medium
- **Status:** confirmed (agent code-inspection suspicion, orchestrator confirmed at runtime via e2e)
- **Repro:** not drivable via `--exec` (`key shift+tab` synthesizes `KeyTab+ModShift`, not `KeyBacktab` — harness gap, see below); e2e test injects `tcell.KeyBacktab` directly
- **Expected:** selection line0→line1-col0 covers only line0 → only line0 outdented
- **Actual:** line1 outdented as well — `KeyBacktab` handler (`internal/ui/editor_widget_keyboard.go:201-232`) iterates `start.Line..end.Line` with no col-0 exclusion, same defect as BUG-002
- **Test:** `tests/e2e/audit_selection_bugs_test.go` (`t.Skip`-marked; verified failing when unskipped)

### BUG-005: Line commands under multicursor leave `e.Multi` stale — next keystroke corrupts the buffer
- **Area:** Multicursor interactions
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** file `foo bar foo baz\nfoo qux\nbar foo end\n`; `bin/ttt --size 120x40 --exec 'wait 200; key ctrl+k l; exec "Duplicate Line"; type Y; screenshot /tmp/s.txt; quit' foo.txt`
- **Expected:** line commands while multicursor is active either shift `Multi.Cursors` consistently (like `multiExecEnter` does) or collapse multicursor mode; typing afterwards must not touch text no cursor covers
- **Actual:** buffer corrupted — `Yar foo baz` / `foo Y` / `bar foo end` (characters clobbered in words that were never selected). Same pattern with Delete Line and Move Line Down. Root cause: `DuplicateLine`/`DeleteLine`/`MoveLineUp`/`MoveLineDown` (`internal/ui/editor_widget_lines.go`) never read or update `e.Multi.Cursors`.
- **Test:** `tests/functional/audit-multicursor-bugs.test.js` (`it.fails`)

### BUG-006: Case transforms under multicursor only affect the primary cursor
- **Area:** Multicursor interactions
- **Severity:** medium
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** same file; `bin/ttt --size 120x40 --exec 'wait 200; key ctrl+k l; exec "Transform to Uppercase"; screenshot /tmp/s.txt; quit' foo.txt`
- **Expected:** all 4 selected "foo" occurrences become "FOO"
- **Actual:** only the first is uppercased; the other 3 are silently ignored while the status bar reports "(4 cursors)". Root cause: `transformSelection` (`internal/ui/editor_widget_text.go:163`) reads only the primary `e.Selection`.
- **Test:** `tests/functional/audit-multicursor-bugs.test.js` (`it.fails`)

### BUG-007: Paste (and cut/copy) under multicursor only applies to the primary selection
- **Area:** Multicursor interactions
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** same file; copy "bar", `key ctrl+k l`, `key ctrl+v`
- **Expected:** paste replaces every cursor's selection (VS Code semantics), or is an explicit no-op under multicursor
- **Actual:** only the primary "foo" becomes "bar"; the other 3 untouched, status bar still "(4 cursors)". Root cause: `EditorGroupWidget.Paste`/`Copy`/`Cut` (`internal/ui/editor_group.go:1262-1322`) and `pasteText` (`internal/ui/editor_widget.go:290`) never consult `e.Multi`.
- **Test:** `tests/functional/audit-multicursor-bugs.test.js` (`it.fails`)

### BUG-008: Undo after a multicursor edit strands the cursor and leaves `e.Multi` stale — next keystroke corrupts
- **Area:** Multicursor interactions
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** same file; `bin/ttt --size 120x40 --exec 'wait 200; key ctrl+k l; type X; key ctrl+z; type Z; screenshot /tmp/s.txt; quit' foo.txt`
- **Expected:** undo restores text and either restores consistent multicursor selections or collapses to single cursor at the primary's pre-edit position
- **Actual:** text restores, but the cursor jumps to the last secondary cursor's stale post-edit position, "(4 cursors)" persists, and typing "Z" corrupts: `foo barZ foo baz` / `fZoo qux` / `bar fZoZo end` (two Z's from one keystroke). Root cause: undo (`internal/core/undo`) has no concept of `e.Multi`, so stale post-edit offsets survive into the reverted buffer.
- **Test:** `tests/functional/audit-multicursor-bugs.test.js` (`it.fails`)

### BUG-009: Cursor movement and backspace split ZWJ grapheme clusters
- **Area:** Wide-char / edge content
- **Severity:** medium
- **Status:** confirmed (orchestrator spot-check; the area agent's sweep missed it and reported clean)
- **Repro:** file `a👨‍👩‍👧‍👦b\n`; `bin/ttt --size 80x15 --exec 'wait 200; key right; key right; debug /tmp/d.json; key backspace; screenshot /tmp/s.txt; quit' zwj.txt`
- **Expected:** the family emoji (7 runes: 👨 ZWJ 👩 ZWJ 👧 ZWJ 👦) is one grapheme cluster — arrow keys cross it in one press, backspace deletes it whole (VS Code behavior)
- **Actual:** cursor stops mid-cluster (rune col 2 after two rights); backspace deletes only the 👨 rune, leaving a dangling ZWJ in the buffer and the emoji rendering exploded as `a 👩 👧 👦b`. Movement/deletion is rune-based everywhere, with no grapheme-cluster segmentation. Combining accents (e + U+0301) are presumably the same family — not separately verified.
- **Note:** rune-based `Col` is a documented design constraint, so the fix is a design decision (grapheme segmentation layer), not a one-liner. Plain CJK, skin-tone-free emoji, tabs, long lines, and boundary cases all passed the sweep — this is specifically about multi-rune clusters.
- **Test:** `tests/functional/audit-grapheme-bugs.test.js` (`it.fails`)

### BUG-010: Search matches go stale after buffer edits — Find Next jumps to non-matching lines
- **Area:** Find/replace
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** open find (`ctrl+f`, query `alpha`), click into the editor, insert a line above the matches, press F3
- **Expected:** matches shift with the edited text (or are recomputed); Find Next never lands on a non-matching line
- **Actual:** F3 jumps to the stale line index — cursor lands on "beta". `SearchMatches` is never recomputed after buffer edits while the bar is open.
- **Test:** `tests/functional/audit-findreplace-bugs.test.js` (`it.fails`)

### BUG-011: Replace All ignores the Case-Sensitive/Regex toggles the bar itself displays
- **Area:** Find/replace
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** `ctrl+r`, query `Foo`, `alt+c` (bar shows 1/1), replacement `X`, `alt+r`
- **Expected:** only the exact-case "Foo" replaced
- **Actual:** all of "Foo"/"foo"/"FOO" replaced — `EditorGroupWidget.ReplaceAll` (`internal/ui/editor_group.go`) re-runs `FindInLines` with a fresh default `SearchOptions{}` instead of the bar's current options
- **Test:** `tests/functional/audit-findreplace-bugs.test.js` (`it.fails`)

### BUG-012: Replace All is not atomic in undo — one Ctrl+Z leaves a never-seen garbled state
- **Area:** Find/replace × undo
- **Severity:** medium
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** replace-all `cat`→`dog` over 4 matches, escape, one `ctrl+z`
- **Expected:** single undo step reverts the whole replace-all (VS Code semantics)
- **Actual:** every replacement pushes 2 ungrouped commands (DeleteSelection + InsertString) — 8 undos to fully revert; one undo yields `" dog dog"`, a state that never existed on screen
- **Test:** `tests/functional/audit-findreplace-bugs.test.js` (`it.fails`)

### BUG-013: Search state survives tab switches — Find Next navigates a tab that has no matches
- **Area:** Find/replace × tabs
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** find `alpha` in tab A (1 match), switch to tab B (no matches), press F3
- **Expected:** bar clears or re-searches on tab switch; F3 in a matchless buffer is a no-op
- **Actual:** tab A's matches persist ("1/1" still shown for tab B); F3 clamps the stale line-5 target into tab B's 2-line buffer and moves the cursor to line 1 — meaningless navigation (no crash; line is clamped)
- **Test:** `tests/functional/audit-findreplace-bugs.test.js` (`it.fails`)

### BUG-014: Replace bar unconditionally swallows all keys — global bindings (tab-switch, tab-close) dead while open
- **Area:** Find/replace × keybindings
- **Severity:** medium
- **Status:** confirmed (agent-reported, orchestrator re-verified with find-bar control run)
- **Repro:** `ctrl+r` open, press `alt+.` (tab.next) — nothing happens; same key with only `ctrl+f` open switches tabs fine
- **Expected:** keys the replace input doesn't handle fall through to global handling, matching the find bar's behavior
- **Actual:** `ReplaceBarWidget.handleKey` (`internal/ui/replacebar_widget.go`) ends with an unconditional `return EventConsumed`
- **Test:** `tests/functional/audit-findreplace-bugs.test.js` (`it.fails`)

### BUG-015: Find does not seed the query from the active selection
- **Area:** Find/replace
- **Severity:** low (VS Code parity)
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** select "world", `ctrl+f`
- **Expected:** query box pre-filled with "world" (VS Code behavior; selection survives either way)
- **Actual:** empty "Search" placeholder; selection does survive
- **Test:** `tests/functional/audit-findreplace-bugs.test.js` (`it.fails`)

### BUG-016: Tab-bar overflow chevron switches the active tab instead of scrolling the strip
- **Area:** Tabs
- **Severity:** medium
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** open 5 files at `--size 50x20` (strip overflows); `click 2 2` on the `◀` chevron → `active_tab` goes 4→3
- **Expected:** chevrons scroll the tab strip to reveal hidden tabs without changing the active file (the code's own comment in `internal/ui/tabbar_widget.go` says "Scroll only when there is something hidden in that direction")
- **Actual:** chevron click calls `g.PrevTab()`/`g.NextTab()` (`internal/ui/editor_group.go:127-128`) — it changes the open file by one per click
- **Test:** `tests/functional/audit-tabbar-bugs.test.js` (`it.fails`)

### BUG-017: Ctrl+Home / Ctrl+End (document start/end) do nothing
- **Area:** Keyboard navigation parity
- **Severity:** medium
- **Status:** confirmed (orchestrator, found while validating the viewport dump field)
- **Repro:** 100-line file; `bin/ttt --size 80x20 --exec 'wait 200; key ctrl+end; debug /tmp/d.json; quit'` → cursor stays at 0,0
- **Expected:** Ctrl+End jumps to end of document, Ctrl+Home to start (plus shift variants for selection) — universal editor behavior, VS Code parity
- **Actual:** the `KeyHome`/`KeyEnd` handlers (`internal/ui/editor_widget_keyboard.go:125-151`) never check ModCtrl, no document start/end command exists in the registry, and nothing is bound — the keypress is silently dropped (not even line home/end fires)
- **Test:** `tests/functional/audit-navigation-bugs.test.js` (`it.fails`)

### BUG-020: Undo of line commands never restores the cursor to the edit site
- **Area:** Undo/redo
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** Delete Line at line 2, move away, `ctrl+z` → text restores but cursor stays put
- **Expected:** undo returns the cursor to where the edit happened (as it does for typed text, paste, join lines)
- **Actual:** `cursorAfterUndo`/`cursorAfterRedo` (`internal/core/undo/undo.go`) have no cases for `InsertLineCommand`/`DeleteLineCommand`/`SwapLineCommand`/`ReplaceLinesCommand`, so `Undo()` returns nil and `editor_group.go:816` skips the cursor update. Affects Delete/Duplicate/Move/Sort Lines.
- **Test:** `tests/functional/audit-undo-bugs.test.js` (`it.fails`)

### BUG-021: Multi-line indent/outdent is one undo step per line, not atomic
- **Area:** Undo/redo
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** select 3 lines, Tab, one `ctrl+z` → only the last line un-indents
- **Expected:** one undo reverts the whole indent (as Toggle Line Comment does via `BatchCommand`)
- **Actual:** KeyTab/KeyBacktab loop issues one top-level `exec` per line with no batch (`internal/ui/editor_widget_keyboard.go:240-247`)
- **Test:** `tests/functional/audit-undo-bugs.test.js` (`it.fails`)

### BUG-022: Enter with auto-indent takes 2–4 undos to revert; first undo can be a no-op
- **Area:** Undo/redo
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** Enter at end of an indented line, one `ctrl+z` → stray blank line remains (only the auto-indent whitespace was undone). Bracket-pair Enter (`{`|`}`) takes 4 undos, the first a visible no-op.
- **Expected:** one undo fully reverts one keypress
- **Actual:** `execEnter()` (`internal/ui/editor_widget_keyboard.go:259-301`) issues 2–4 separate top-level exec calls, unbatched
- **Test:** `tests/functional/audit-undo-bugs.test.js` (`it.fails`)

### BUG-023: Viewport does not scroll to an off-screen undo location
- **Area:** Undo/redo
- **Severity:** medium
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** edit line 90, jump to line 1, `ctrl+z` → `cursor.line` 89, `viewport.top_line` 0 — cursor invisible
- **Expected:** undo scrolls the restored cursor into view like every other cursor-moving path
- **Actual:** `EditorGroupWidget.Undo()`/`Redo()` (`internal/ui/editor_group.go:810-842`) never call `scrollViewport()`
- **Test:** `tests/functional/audit-undo-bugs.test.js` (`it.fails`)

### BUG-024: Undoing an edit on a folded header line silently unfolds the region
- **Area:** Undo/redo × folding
- **Severity:** medium
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** fold a function, type a char on the header line, `ctrl+z` → buffer byte-identical to pre-edit, fold expanded
- **Expected:** fold state survives an undo that restores the exact pre-edit text
- **Actual:** fold collapsed-state is dropped on the edit and not restored by undo
- **Test:** `tests/functional/audit-undo-bugs.test.js` (`it.fails`)

### BUG-025: Undo grouping only breaks on whitespace — punctuation runs undo as one blob
- **Area:** Undo/redo
- **Severity:** low (possibly intentional — design question)
- **Status:** confirmed behavior; whether it's a bug needs an owner decision
- **Repro:** type `a.b.c.d.e`, one `ctrl+z` → entire string removed
- **Expected (VS Code-ish):** punctuation acts as a group delimiter like whitespace
- **Actual:** `canGroup()` (`internal/core/undo/undo.go:73-96`) only breaks groups on space/tab
- **Test:** none — behavior is a design choice; test would prescribe an undecided policy

### Harness gap from the undo sweep
No `folds` field in the debug dump (collapsed ranges) — BUG-024 had to be confirmed via screenshot fold markers. Add fold state if the folding sweep needs it.

### BUG-026: Fold collapsed-state reattaches to an unrelated block after line-count edits
- **Area:** Folding × editing
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** fold the `if true {` block (line 4), insert a blank line at the top of the file → the fold marker now collapses `func outer()`'s entire body instead
- **Expected:** collapsed state follows the folded content (or at worst clears); a region the user never folded must never become collapsed
- **Actual:** `fold.State.SetRanges` (`internal/core/fold/fold.go:26-39`) recomputes ranges on every line-count change and preserves collapse purely by raw `StartLine` equality — after the shift, the outer function's new start line coincides with the old collapsed key and inherits the fold, silently hiding different code. Duplicate Line on a folded header makes the fold vanish by the same mechanism.
- **Test:** `tests/functional/audit-fold-bugs.test.js` (`it.fails`)

### BUG-027: Move Line on a folded header swaps the header with a HIDDEN line — silent code reordering
- **Area:** Folding × editing
- **Severity:** high
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** fold `if true {`, press `alt+down` → buffer becomes `func outer() { / \t\tfoo() / \tif true {` — `foo()` hoisted out of its block — while the fold marker still renders as if valid
- **Expected:** move the whole folded region as a unit (VS Code) or no-op while folded; never reorder invisible code
- **Actual:** `MoveLineDown`/`Up` issue a raw `SwapLineCommand` with no fold awareness; since line COUNT is unchanged, the `exec()` fold-recompute guard (`internal/ui/editor_widget.go:214-217`) never fires, so the stale marker keeps rendering over now-invalid structure
- **Test:** `tests/functional/audit-fold-bugs.test.js` (`it.fails`)

### Folding area notes (clean probes)
Delete Line on folded header (hidden lines revealed, no data loss), copy/paste of header, selection deletion across folds, Join Lines, arrow-skip over collapsed regions, go-to-line auto-expand, nested fold preservation, collapse/expand-all, save-with-folds — all correct. Syntax-highlight layering on collapsed headers not independently verified (no style info in dump).

### BUG-018: Clicking a second menu header closes the open menu instead of switching to it
- **Area:** Mouse / menu bar
- **Severity:** medium
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** `click 4 0` (File opens), `click 30 0` (View header) → `.overlay` becomes null, no menu open; a further click is needed to open View
- **Expected:** standard menu-bar UX — clicking another header while a menu is open switches directly to that menu
- **Actual:** the second click only dismisses the open menu
- **Test:** `tests/functional/audit-mouse-bugs.test.js` (`it.fails`)

### BUG-019: Rightmost column of explorer tree rows is click-dead
- **Area:** Mouse / sidebar explorer
- **Severity:** low
- **Status:** confirmed (agent-reported, orchestrator re-verified)
- **Repro:** explorer tree rect `{x:1,w:30}`; `click 30 6` on a file row does nothing; `click 29 6` opens the file
- **Expected:** the full row rect (x=1..30) is clickable
- **Actual:** the last column is dead — off-by-one in the tree's hit test
- **Test:** `tests/functional/audit-mouse-bugs.test.js` (`it.fails`)

### Mouse-area notes
- Core editor click mapping is clean: scrolled viewports (vertical + horizontal), tabs, CJK, clamping past line end/below last line, gutter, double-click word select (incl. CJK/punctuation), triple-click line select — no offset divergences found.
- The reported "stale wrap-map click after Toggle Word Wrap" is a **harness artifact, not a product bug**: synchronous `exec "..."` doesn't trigger a render pass, so an immediately following `click` resolves against the pre-toggle wrap map. Through the real key-dispatch path (palette + Enter) the same click resolves correctly. Same root as the stale-status-bar gap below.
- Additional dump gaps: `.overlay` reports `{"type":"unknown"}` for menus (can't distinguish overlay kinds); top-level `.focus` reports `"other"` for most real focus states (also seen in the first sweep) — per-widget `focused` flags in the widget tree are the workaround. `--exec` has no `drag` command, so drag-selection remains unprobed.

### Tabs area notes (clean probes)
Per-tab cursor/selection/scroll/multicursor/fold restoration, undo isolation, dirty-flag lifecycle, close-with-unsaved-changes dialog, duplicate-open reuse, new-file/save-as, overflow hit-testing on tab labels, wrap-around switching, and widget-tree leak checks all passed. **Editor split panes do not exist** in the codebase (`SplitPanelWidget`/`ContentSplitWidget` are the sidebar/bottom layout splits) — that sub-area is N/A until the feature exists.

### Harness gap from the find/replace sweep
`debug` JSON has no `search` section (query, options, match list, active index, bar focus) — match staleness had to be inferred via cursor movement. Screenshot carries no style info, so highlight-artifact checks are only indirect. Consider a `search` block in the dump.

### Harness gap (not a product bug): `--exec key shift+tab` cannot produce `KeyBacktab`
`comboToTcell("shift+tab")` yields `(KeyTab, ModShift)`; there is no `backtab` keyword in the key parser, so the `KeyBacktab` code path is unreachable from `--exec`/functional tests. Real terminals send Backtab as CSI Z. Consider adding a `backtab` keyword when convenient — until then, Backtab behavior is only testable via e2e event injection.

### Harness gaps from the multicursor sweep
- ~~`debug` JSON lacked buffer text and multicursor state~~ — **fixed on this branch** (`buffer.text` capped at 1000 lines, `multi_cursor[]` with selection anchors).
- `--exec click/hover` always post `ModNone` — Alt+Click (mouse add-cursor) cannot be exercised; that feature remains untested.
- `exec "Command Name"` runs synchronously and bypasses the event loop's `syncStatus()`, so a screenshot taken immediately after shows a stale status bar (cursor pos, cursor count, dirty flag) until the next real key/click event. Affects harness observations only, not real palette usage.
- "Add cursor above/below" does not exist as commands — cursor creation is `ctrl+d` / `ctrl+k l` / Alt+Click only. `editor.splitSelectionToLines` exists but was not deeply probed (budget).

<!-- Template:
### BUG-NNN: <one-line summary>
- **Area:**
- **Severity:** low / medium / high
- **Status:** confirmed / unconfirmed
- **Repro:** `bin/ttt --size 120x40 --exec "..."`
- **Expected:**
- **Actual:**
- **Test:** <path or "none — reason">
-->
