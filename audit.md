# ttt Feature Audit — Gaps to a Daily-Driver Professional Editor

Audit date: 2026-06-10. Updated: 2026-06-11 with cross-reference against VS Code, Zed, Sublime
Text, and developer community feedback (HN, Reddit, Lobsters).

Scope: essential features for daily professional use. **Out of scope: DAP (debugger).**

## Verdict

The foundation is strong: multi-cursor, find/replace (in-file + project-wide with ripgrep),
fuzzy file open, command palette, LSP (completions, hover, go-to-def, references, rename, format,
diagnostics), git integration (stage/unstage/commit/diff/blame/PR review), integrated terminal,
EditorConfig, themes, menus, context menus, sidebar panels, and syntax highlighting all exist.

Since PR #55 was opened, several gaps have been closed:
- Atomic save + permission/symlink preservation (PR #56)
- External file change detection with silent reload (PR #58)
- Line-ending detection/preservation with status bar toggle (PR #60)
- Trim trailing whitespace on save (PR #61)
- Toggle comment with Ctrl+/ and selection-aware multi-line support (PR #62)

The remaining gaps cluster into: **navigation & orientation** (no back/forward, no symbol outline,
no code folding), **editor layout** (no split panes, no session restore), **LSP depth** (no code
actions UI, no inlay hints, no snippets), and **visual polish** (no indent guides, no git gutter).

## Feature Matrix

Features are ranked by cross-source consensus (how many of VS Code, Zed, Sublime Text, HN/Reddit
flagged them) and by impact in an **agent-driven workflow** where reviewing and navigating code
matters more than writing it.

### Tier 1 — High Impact (4-5 sources agree, transforms the editor)

| Feature | Status | Consensus | Notes |
|---|---|---|---|
| Split editor panes | MISSING | VS Code, Zed, Sublime, HN, Reddit | Side-by-side viewing for comparing files, reviewing agent output. Fundamental layout feature. |
| Code folding | MISSING | VS Code, Zed, Sublime, HN, Reddit | Collapse/expand code blocks. Indent-based works immediately; tree-sitter ideal long-term. |
| Session restore | MISSING | VS Code, Zed, Sublime, HN, Reddit | Reopen with same tabs, cursors, scroll positions. `.ttt` workspace files are the persistence vehicle. |
| Navigation history (back/forward) | MISSING | VS Code, Zed, Sublime, HN, Reddit | Alt+Left/Right after go-to-definition. Essential companion to existing LSP navigation. |
| Go to Symbol (file + workspace) | MISSING | VS Code, Zed, Sublime, Reddit | Ctrl+Shift+O for file outline, Ctrl+T project-wide. Uses LSP `documentSymbol`/`workspace/symbol`. |
| Code actions / lightbulb UI | MISSING | VS Code, Zed, HN, Reddit | Interactive quick-fix picker (Ctrl+.). LSP `textDocument/codeAction`. Have code actions on save, but no interactive UI. |

### Tier 2 — Developer Expectations (3-4 sources, expected by power users)

| Feature | Status | Consensus | Notes |
|---|---|---|---|
| Inlay hints | MISSING | VS Code, Zed, HN, Reddit | Inline type/parameter annotations via LSP `textDocument/inlayHint`. High value for Go/TS/Rust. |
| Snippet expansion | MISSING | VS Code, Zed, Sublime, Reddit | LSP completions return snippets with tab stops/placeholders. Without this, many completions are degraded. |
| Indent guides | MISSING | VS Code, Zed, Sublime, Reddit | Vertical lines at indent levels. Pure rendering, no AST needed. Quick win. |
| Git gutter indicators | MISSING | VS Code, Zed, HN, Reddit | Green/blue/red marks in gutter for added/modified/deleted lines vs HEAD. High visibility. |
| Outline / symbol sidebar | MISSING | VS Code, Zed, Reddit | Tree of functions/classes in sidebar panel. Leverages existing LSP + sidebar. |
| Column/box selection | MISSING | VS Code, Sublime, HN, Reddit | Rectangular selection with Ctrl+Alt+Up/Down. Complements existing multi-cursor. |
| Macro recording/playback | MISSING | Sublime, HN, Reddit | Record/replay editing sequences. Covers sequential cases multi-cursor can't. |

### Tier 3 — Differentiators (2-3 sources, sets ttt apart)

| Feature | Status | Consensus | Notes |
|---|---|---|---|
| Sticky scroll | MISSING | VS Code, Reddit | Pin enclosing scope headers at viewport top. No terminal editor has this. |
| Breadcrumbs | MISSING | VS Code, Zed, Reddit | File > Class > Method path showing current location. |
| Semantic tokens | MISSING | VS Code, Zed | LSP `textDocument/semanticTokens` for accurate syntax highlighting beyond regex. |
| Bracket pair colorization | MISSING | VS Code, Zed | Color-code nested bracket pairs. Readability improvement. |
| Bookmarks / marks | MISSING | Sublime, HN | Toggle bookmarks on lines, jump between them. Simple but useful in large files. |
| Word wrap rendering | PARTIAL | VS Code, Sublime, HN | `wordWrap` setting exists but is a no-op. Important for markdown/prose/logs. |
| Merge conflict resolution UI | MISSING | VS Code, Zed | Inline accept current/incoming/both above conflict markers. |

### Tier 4 — Polish & Nice-to-Have

| Feature | Status | Consensus | Notes |
|---|---|---|---|
| Minimap | MISSING | VS Code, Zed, Sublime | Character-based code overview. Polarizing — some love, some hide. |
| Zen/distraction-free mode | MISSING | VS Code, Sublime | Hide all chrome, center text. Simple to implement. |
| Sort/reverse/unique lines | MISSING | Sublime | Small text manipulation commands. Quick wins. |
| Case transforms (upper/lower/title) | MISSING | Sublime | Ctrl+K Ctrl+U/L. Quick wins. |
| Join lines | MISSING | VS Code, Sublime | Ctrl+J to merge next line up. |
| Split selection into lines | MISSING | Sublime | Ctrl+Shift+L — selection to per-line cursors. |
| Goto matching bracket | MISSING | VS Code, Sublime | `findMatchingBracket()` exists but no navigation command. |
| Recent files / reopen closed tab | MISSING | VS Code, Zed | MRU list, Ctrl+Shift+T. |
| Relative line numbers | MISSING | Zed, Sublime | Useful for vim-style workflows. |
| Vim keybindings mode | MISSING | Zed, Sublime, HN | Large audience but massive implementation effort. |
| Autosave | MISSING | VS Code, Zed | Save on focus change, timer, etc. |
| Settings hot reload | MISSING | Zed, Reddit | Change settings without restart. |
| Preview/transient tabs | MISSING | VS Code, Zed, Sublime | Single-click opens preview; editing pins the tab. |
| Task runner | MISSING | VS Code, Zed | Run build/test commands with error parsing. Terminal covers most of this. |
| Plugin/extension system | MISSING | HN, Reddit | Long-term multiplier but massive effort. |
| AI inline completion (ghost text) | MISSING | HN, Reddit | Growing expectation; integration point for Ollama/OpenAI. |
| Tree-sitter integration | MISSING | HN, Reddit | Architectural investment. Unlocks accurate highlighting, text objects, structural editing. |

### Completed (since original audit)

| Feature | Status | PR |
|---|---|---|
| Atomic save + permission/symlink preservation | DONE | #56 |
| External file change detection + silent reload | DONE | #58 |
| Line-ending detection/preservation + status bar toggle | DONE | #60 |
| Trim trailing whitespace on save | DONE | #61 |
| Toggle comment (Ctrl+/, selection-aware, multi-language) | DONE | #62 |

### Already Existed (confirmed working)

Multi-cursor (Ctrl+D, Alt+Click, select all occurrences), find/replace (in-file + project-wide),
fuzzy file open, command palette, LSP suite, git (stage/unstage/commit/diff/blame/PR review),
integrated terminal, EditorConfig, themes, auto-indent, auto-pair, smart home, move/dup/delete
line, indent/dedent selection, double/triple-click, menus, context menus, sidebar panels,
problems panel, bracket highlighting.

## Recommended Next Steps

Prioritized for an agent-driven workflow (reviewing and navigating > writing):

1. **Split editor panes** — side-by-side review of agent output vs existing code
2. **Code folding** (indent-based) — collapse irrelevant sections in large files
3. **Session restore** — resume context between sessions via `.ttt` workspace files
4. **Navigation history** (back/forward) — jump around code without losing position
5. **Go to Symbol** — quickly orient in unfamiliar/generated code
6. **Indent guides** — quick visual win, pure rendering
7. **Git gutter indicators** — high-visibility improvement, diff infra exists
8. **Code actions UI** (Ctrl+.) — interactive quick-fix picker
9. **Inlay hints** — inline type/parameter annotations
10. **Snippet expansion** — fix degraded LSP completions

## Sources

- VS Code official documentation and feature comparison
- Zed editor documentation and blog posts
- Sublime Text documentation and feature pages
- Hacker News discussions on terminal editors (2024-2026)
- Reddit, Lobsters, and developer blog posts on editor wishlists
- Helix, Kakoune, and Micro editor comparisons
