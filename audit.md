# UX Bug Audit

Started 2026-07-11 on branch `audit/bug-hunt`. Goal: **discover and document** UX bugs — no fixes on this branch. Each confirmed finding gets a ledger entry below and, where feasible, a repro test marked as expected-failure (`t.Skip("BUG-NNN")` in Go, `it.fails(...)` in vitest) so the suite stays green until the bug is fixed.

Process: one hunting agent at a time, scoped to an area from the coverage matrix. Orchestrator re-verifies every repro before it enters this file. LSP is out of scope.

## Coverage matrix

| Area | Status | Findings |
|---|---|---|
| Editing commands × selection | pending | |
| Multicursor interactions | pending | |
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
