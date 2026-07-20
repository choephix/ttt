# Vim Mode

A Vim compatibility layer for ttt, implemented entirely as a Lua plugin.
Tracks [issue #386](https://github.com/eugenioenko/ttt/issues/386).

## Status

Under construction, built in phases. Currently at **Phase 1**.

| Phase | Scope | State |
|---|---|---|
| 0 | Mode state machine, normal/insert, Esc, status indicator | ✅ |
| 1 | Motions and counts | ✅ |
| 2 | Insert mode and simple edits | ⬜ |
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
| `i` | Enter insert mode |
| `Esc`, `Ctrl-[` | Return to normal mode (cursor clamps back onto a character) |
| — | Printable keys in normal mode are swallowed, never typed |

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

Normal mode also overrides `Ctrl-D`, `Ctrl-U`, `Ctrl-E`, `Ctrl-F` and `Ctrl-Y`,
which core binds to `multicursor.selectNext`, `editor.autocomplete`,
`editor.redo` and `search.find`. That is intended — those commands remain
reachable from insert mode, the command palette, and with Vim mode disabled.

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
clamp to `#line`; insert mode allows one past the end. `clamp_cursor()` runs on
the insert → normal transition. (Vim additionally steps the cursor one column
left on `Esc`; that belongs with the Phase 2 insert-mode work and is not done
yet.)

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
cd tests/functional && npx vitest run vim-mode.test.js vim-motions.test.js
```
