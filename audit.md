# UX Bug Audit

Started 2026-07-11 on branch `audit/bug-hunt`. Goal: **discover and document** UX bugs — no fixes on this branch. Each confirmed finding gets a ledger entry below and, where feasible, a repro test marked as expected-failure (`t.Skip("BUG-NNN")` in Go, `it.fails(...)` in vitest) so the suite stays green until the bug is fixed.

Process: one hunting agent at a time, scoped to an area from the coverage matrix. Orchestrator re-verifies every repro before it enters this file. LSP is out of scope.

## Coverage matrix

| Area | Status | Findings |
|---|---|---|
| Editing commands × selection | swept (4 findings) | BUG-001..004 |
| Multicursor interactions | in progress | |
| Undo/redo semantics | pending | |
| Code folding × editing | pending | |
| Find/replace + search highlights | pending | |
| Tabs & split panes | pending | |
| Explorer (file tree) | pending | |
| Mouse targets / click offsets | pending | |
| Resize & layout | pending | |
| Wide-char / edge content (CJK, emoji, tabs, long lines) | pending | |
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

### Harness gap (not a product bug): `--exec key shift+tab` cannot produce `KeyBacktab`
`comboToTcell("shift+tab")` yields `(KeyTab, ModShift)`; there is no `backtab` keyword in the key parser, so the `KeyBacktab` code path is unreachable from `--exec`/functional tests. Real terminals send Backtab as CSI Z. Consider adding a `backtab` keyword when convenient — until then, Backtab behavior is only testable via e2e event injection.

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
