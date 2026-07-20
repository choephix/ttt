# Vim Mode

A Vim compatibility layer for ttt, implemented entirely as a Lua plugin.
Tracks [issue #386](https://github.com/eugenioenko/ttt/issues/386).

## Status

Under construction, built in phases. Currently at **Phase 2**.

| Phase | Scope | State |
|---|---|---|
| 0 | Mode state machine, normal/insert, Esc, status indicator | ✅ |
| 1 | Motions and counts | ✅ |
| 2 | Insert mode and simple edits | ✅ |
| 3 | Operators and text objects | ⬜ |
| 4 | Visual modes | ⬜ |
| 5 | Registers, marks, macros, `.` repeat | ⬜ |
| 6 | Ex command line, search, editor integration | ⬜ |
| 7 | Settings, docs, handoff to `ttt-plugins` | ⬜ |

`cheatsheet.md` is a vendored copy of [ibhagwan/vim-cheatsheet](https://github.com/ibhagwan/vim-cheatsheet),
used as the coverage checklist.

## Supported today

Every motion below accepts a `{count}` prefix (`3j`, `5w`, `2fx`, `10G`).

### Modes

| Key | Action |
|---|---|
| `i` `a` | Insert before / after the cursor |
| `I` `A` | Insert at the first non-blank / end of line |
| `o` `O` | Open a line below / above and insert |
| `gi` | Resume insert where it was last left |
| `R` | Replace (overtype) mode until `Esc` |
| `Esc`, `Ctrl-[` | Return to normal mode (cursor steps one column left and clamps onto a character) |
| — | Printable keys in normal mode are swallowed, never typed |

### Edits

| Key | Action |
|---|---|
| `x` `X` | Delete `{count}` characters forward / backward |
| `D` `C` | Delete / change to end of line (`{count}` extends over following lines) |
| `s` `S` | Substitute `{count}` characters / whole lines, then insert |
| `r{c}` | Replace `{count}` characters with `{c}` (refused if it runs past the line end) |
| `~` | Toggle case of `{count}` characters and advance |
| `J` `gJ` | Join `{count}` lines, with / without an inserted space |
| `u` `Ctrl-R` | Undo / redo `{count}` times (delegates to `editor.undo` / `editor.redo`) |
| `>>` `<<` | Indent / dedent `{count}` lines by one shiftwidth |
| `==` | Reindent `{count}` lines (heuristic, see below) |
| `Ctrl-A` `Ctrl-X` | Increment / decrement the number at or after the cursor by `{count}` |

### Motions

| Key | Action |
|---|---|
| `h` `j` `k` `l` | Left, down, up, right. `j`/`k` keep a sticky goal column. |
| `+` `-` | First non-blank of the next / previous line |
| `0` `^` `$` `g_` | Line start, first non-blank, line end, last non-blank |
| `w` `W` `b` `B` | Word / WORD forward and backward |
| `e` `E` `ge` `gE` | End of next / previous word / WORD |
| `gg` `G` | First / last line (`{n}gg` and `{n}G` jump to line *n*) |
| `f{c}` `F{c}` | Forward / backward to `{c}` on the current line |
| `t{c}` `T{c}` | Forward / backward until just before `{c}` |
| `;` `,` | Repeat the last `f`/`F`/`t`/`T`, same / opposite direction |
| `{` `}` | Previous / next blank line |
| `%` | Matching bracket (delegates to `editor.goToMatchingBracket`) |
| `H` `M` `L` | Top / middle / bottom of the screen |
| `Ctrl-D` `Ctrl-U` | Half page down / up |
| `Ctrl-F` `Ctrl-B`\* | Full page down / up |
| `PgDn` `PgUp` | Full page down / up (always available, see below) |
| `Ctrl-E` `Ctrl-Y` | Scroll view one line down / up |
| `zz` `zt` `zb` | Centre / top / bottom the view on the cursor |

\* **`Ctrl-B` does not reach the plugin.** Core registers it as a *force key*
for `sidebar.toggle`, and force keys are checked before the plugin key
interceptor by design (`internal/ui/root.go`). `PgUp` is provided as the
equivalent. The binding is still in the motion table, so rebinding
`sidebar.toggle` makes `Ctrl-B` work.

Normal mode also overrides `Ctrl-D`, `Ctrl-U`, `Ctrl-E`, `Ctrl-F`, `Ctrl-Y`,
`Ctrl-A`, `Ctrl-X` and `Ctrl-R`, which core binds to `multicursor.selectNext`,
`editor.autocomplete`, `editor.redo`, `search.find`, `editor.selectAll`,
`editor.cut` and `search.replace`. That is intended — those commands remain
reachable from insert mode, the command palette, and with Vim mode disabled.

### Known gaps in Phase 2

- **Counts on insert-entry commands are ignored.** `3o` / `3i` repeat the typed
  text on `Esc`, which needs the per-keystroke change log that arrives with `.`
  repeat in Phase 5.
- **`o` / `O` do not autoindent.** They open a column-1 line, matching vanilla
  Vim without `autoindent`.
- **Backspace in `R` only walks left.** Vim restores the overwritten character;
  that also needs the Phase 5 change log.
- **`==` is a heuristic, not an indent engine.** ttt has none, so `==` copies
  the previous non-blank line's (space) indent, adds one shiftwidth if that line
  ends with `{`/`(`/`[`/`:`, and removes one if this line starts with a closing
  bracket.
- **Shiftwidth is hardcoded to 4 spaces.** Reading `editor.tabSize` needs a
  `settings` permission the manifest does not request; wiring it is Phase 7.

Commands: `Vim: Toggle Vim Mode`, `Vim: Enable Vim Mode`, `Vim: Disable Vim Mode`.

## Design notes

**Single file.** The plugin sandbox strips `package.loaders` down to the preload
loader (`internal/plugin/sandbox.go`), so a plugin cannot `require` sibling
`.lua` files. `init.lua` is therefore one file, organized into delimited
sections.

**Key normalization.** `key.press` delivers `{key, rune, mod}`. Ctrl+letter
arrives with *both* `key="Ctrl-D"` and `mod="ctrl"`, so `token_of` normalizes
off the key name first. Canonical tokens: runes keep case (`g` vs `G`), named
keys lowercase (`esc`), control keys collapse to `ctrl-d`.

**Pass-through discipline.** The core key interceptor sits above Escape
handling and chords (`internal/ui/root.go`), so this plugin must return `false`
for every key it does not own — otherwise it silently breaks global bindings.
Two rules follow from that:

- Esc is passed through when there is nothing Vim-side to cancel, so core
  `EscapeDismissers` still run.
- Core skips the interceptor while a chord is in flight, so `ctrl+k s` and
  friends keep working without the plugin needing to know about chords.

**Cursor semantics.** Normal mode puts the cursor *on* a character, so columns
clamp to `#line`; insert mode allows one past the end. `leave_insert()` runs on
the insert → normal transition: it records the position for `gi`, steps the
cursor one column left the way Vim does, then clamps.

**Undo atomicity.** Every Vim operation is exactly one undo step — `3x` undoes
as one `u`. Each edit is bracketed by `editor.begin_undo_group()` /
`end_undo_group()`. Undo transactions do **not** nest: `BeginTransaction` resets
the transaction start index (`internal/core/undo/undo.go`), so a second `begin`
mid-operation silently drops everything before it from the group. Commands that
edit *and then* enter insert mode (`o`, `C`, `s`, `S`, `gi`) therefore open the
group themselves and let `leave_insert()` close it on `Esc`, which is what makes
`o` + typing + `Esc` a single undo.

**`o` on the last line.** `PluginEditorAPI.Insert` rejects `line >= len(Lines)`
(`internal/app/plugin_api.go`), so `o` cannot address the line *after* the last
one. It appends `"\n"` at the end of the current line instead, which produces
the same buffer.

**Cursor writes scroll.** `set_cursor` calls `EnsureCursorVisible` on the Go
side, so every scrolling routine moves the cursor *first* and calls `scroll_to`
*last* — the other order gets silently undone.

**Lua 5.1.** gopher-lua implements 5.1 — no `%g` character class, no `goto`.
Errors thrown inside a `key.press` listener are swallowed and the key falls
through, which reads exactly like "the plugin ignored my key". Suspect a Lua
error first when a binding mysteriously does nothing.

## Development

```sh
make build
bin/ttt --size 100x30 --plugin plugins/vim/init.lua file.txt \
  --exec "wait 300; type ihello; key esc; wait 100; screenshot /tmp/s.txt; quit"
```

Note that running from the repo root also auto-discovers `plugins/vim/` and
raises a one-time permission dialog — for the *discovered* copy, which is a
second instance alongside the `--plugin` one. That modal swallows every
keystroke, so a manual `--exec` probe launched from the repo root looks like the
plugin is ignoring keys. Launch probes from a scratch directory (with an
absolute `--plugin` path and an absolute file path) to avoid it; the functional
tests run from `tests/functional/`, which has no `plugins/` dir, so they are
unaffected.

Tests:

```sh
cd tests/functional && npx vitest run vim-mode.test.js vim-motions.test.js vim-edits.test.js
```
