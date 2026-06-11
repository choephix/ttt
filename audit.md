# ttt Feature Audit — Gaps to a Daily-Driver Professional Editor

Audit date: 2026-06-10. Scope: essential quality-of-life features for daily professional use,
compared against editors like VS Code. **Explicitly out of scope: LSP-depth features and DAP.**

## Verdict

The foundation is strong: multi-cursor, find/replace (in-file and project-wide with ripgrep),
fuzzy file open, command palette, git changes panel with PR review, integrated terminal with
tabs, EditorConfig, themes, menus, and context menus all exist and work. The gaps cluster into
three areas: **data safety** (no external-change detection, non-atomic saves, no session
restore), **navigation memory** (no back/forward history, no recent files), and **git depth**
(no gutter indicators, no branch switching, no conflict resolution).

## Priority 1 — Data safety & trust (blockers for daily driving)

These are the features whose absence loses user data or breaks trust in the tool.

| Feature | Status | Notes |
|---|---|---|
| External file change detection | MISSING | No fsnotify/polling. `git checkout`, formatters, or other tools editing a file silently desync the buffer. Editing then saving overwrites external changes. |
| Atomic save + permission preservation | MISSING | `SaveFile()` uses `os.Create()` directly (`internal/core/buffer/io.go:39`) — truncate-then-write loses data on crash mid-save, and resets file permissions to 0644. |
| Hot exit / session restore | MISSING | Open tabs, cursor positions, and unsaved changes are lost on quit/crash. Only workspace folder paths persist (`.ttt` files). |
| Line-ending preservation | MISSING | Loader strips endings via `bufio.Scanner`, saver always writes LF. Opening and saving a CRLF file silently rewrites every line. Status bar hardcodes "LF". |
| Read-only file handling | MISSING | No permission check before save; failure surfaces as a raw error. |
| Crash recovery / backup files | PARTIAL | Crash stack trace logged to `crash.log`, but no buffer recovery. |
| Autosave | MISSING | No setting, no timer, no focus-loss save. |
| Large file handling | RISK | Whole file loaded into `[]string`; no guard or special casing for very large files. |

## Priority 2 — Navigation memory (the "feels slow without it" tier)

| Feature | Status | Notes |
|---|---|---|
| Back/forward cursor history (Alt+Left/Right) | MISSING | No location stack. After go-to-definition or a search jump, there is no way back. |
| Recent files / reopen closed tab | MISSING | No MRU list, no Ctrl+Shift+T equivalent. |
| Reveal active file in tree | PARTIAL | Active file highlights if visible, but parent folders don't auto-expand; no explicit "reveal" command. |
| Bookmarks / marks | MISSING | |
| Outline / symbol list | MISSING | No outline view, even regex-based. |
| Breadcrumbs | MISSING | |

## Priority 3 — Editing polish

| Feature | Status | Notes |
|---|---|---|
| Toggle line comment keybinding | PARTIAL | `ToggleLineComment()` exists (`internal/ui/editor_widget.go:1167`) with per-language prefixes, but is **unbound** (no Ctrl+/) and only handles a single line, not a selection. |
| Trim trailing whitespace on save | PARTIAL | EditorConfig `trim_trailing_whitespace` is parsed but never applied on save. |
| Word wrap | PARTIAL | `wordWrap` setting exists but rendering never wraps — setting is a no-op. |
| Whole-word toggle in find | MISSING | Find bar has case + regex toggles, but no whole-word. |
| Jump to matching bracket | MISSING | Highlighting exists (`findMatchingBracket()`), but no navigation command. |
| Add cursor above/below | MISSING | Alt+Click, Ctrl+D, select-all-occurrences exist; column-wise cursor stacking does not. |
| Column/block selection | MISSING | Selection model is linear anchor-to-cursor only. |
| Wrap selection in quotes/brackets | MISSING | Typing `"` with a selection replaces it instead of surrounding it. |
| Case transforms / join lines / sort lines | MISSING | None of the classic text commands exist. |
| Per-language indent settings | MISSING | Only global settings + EditorConfig. |
| Smart-home, auto-indent, auto-pair, indent/dedent selection, move/dup/delete line, double/triple-click selection | EXISTS | Solid coverage. |

## Priority 4 — Git depth

| Feature | Status | Notes |
|---|---|---|
| Gutter change indicators | MISSING | No modified/added/deleted marks next to line numbers — one of the most-looked-at signals in VS Code. |
| Branch switching / creation UI | MISSING | Branch shows in status bar but isn't clickable; no checkout/create commands. |
| Merge conflict detection & resolution | MISSING | No conflict marker highlighting, no accept ours/theirs. |
| Commit history / log view | MISSING | |
| Stash support | MISSING | |
| Stage/unstage/discard/commit, push/pull/sync, diff views, PR review, inline blame | EXISTS | Comprehensive. |

## Priority 5 — Workbench

| Feature | Status | Notes |
|---|---|---|
| Side-by-side editor splits | DEFERRED | Current splits are sidebar/editor and editor/terminal only. As a terminal editor, side-by-side is well served by tmux/multiplexer panes running separate instances; in-editor splits only add value for same-file dual views and shared buffer state. Not worth the structural lift now. |
| Tab drag-reorder / move between splits | MISSING | Tabs scroll and pin, but can't be reordered. |
| Move files in explorer | MISSING | Create/rename/delete exist; no move (and delete is permanent — `os.RemoveAll`, no trash). |
| Settings hot reload | MISSING | `settings.json` edits require restart. |
| Zen / distraction-free mode | MISSING | Terminal has fullscreen; editor doesn't. |
| Minimap | MISSING | Arguably optional in a terminal editor. |
| Explorer file watching | MISSING | Tree only updates on manual refresh. |
| Menus, context menus, command palette, sidebar panels, problems panel, terminal tabs+selection | EXISTS | Comprehensive. |

## Suggested attack order

1. **Atomic save + permission preservation** — small, contained (`internal/core/buffer/io.go`), eliminates the worst data-loss risk.
2. **External change detection** (fsnotify or stat-on-focus) with reload prompt — the other half of data trust.
3. **Line-ending detection/preservation** — quiet correctness fix; status bar already has the slot.
4. **Bind Ctrl+/ and make toggle-comment selection-aware** — the code is 90% there.
5. **Back/forward navigation history** — touches cursor/jump paths but is self-contained state.
6. **Session restore (open tabs + cursor positions)** — workspace files already provide the persistence vehicle.
7. **Recent files / reopen closed tab** — small, pairs with quick-open.
8. **Git gutter indicators** — high visibility payoff; diff infrastructure already exists.
9. **Trim trailing whitespace on save** — parser already reads the flag; just apply it.

Editor side-by-side splits are intentionally deferred: tmux panes with separate instances cover
the terminal-native workflow, and the structural cost inside the editor is the highest on this list.
