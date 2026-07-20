# Vim Mode

A Vim compatibility layer for ttt, implemented entirely as a Lua plugin.
Tracks [issue #386](https://github.com/eugenioenko/ttt/issues/386).

## Status

Under construction, built in phases. Currently at **Phase 5**.

| Phase | Scope | State |
|---|---|---|
| 0 | Mode state machine, normal/insert, Esc, status indicator | ✅ |
| 1 | Motions and counts | ✅ |
| 2 | Insert mode and simple edits | ✅ |
| 3 | Operators and text objects | ✅ |
| 4 | Visual modes | ✅ |
| 5 | Registers, marks, macros, `.` repeat | ✅ |
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
| `p` `P` | Paste `{count}` times after / before the cursor (linewise- and charwise-aware) |
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
`editor.cut` and `search.replace`. `Ctrl-V` (`editor.paste`) is overridden the
same way, for visual block mode. That is intended — those commands remain
reachable from insert mode, the command palette, and with Vim mode disabled.

### Operators

Full `{count}{operator}{count}{motion|textobject}` composition — the two counts
multiply, so `2d3w` deletes six words.

| Key | Action |
|---|---|
| `d` `c` `y` | Delete / change / yank over a motion or text object |
| `>` `<` `=` | Indent / dedent / reindent (always linewise) |
| `gu` `gU` `g~` | Lowercase / uppercase / swap case over a motion or text object |
| `dd` `cc` `yy` `>>` `<<` `==` | Doubled form: `{count}` whole lines |
| `guu` `gUU` `g~~` | Doubled form of the case operators (`gugu`, `gUgU`, `g~g~` also work) |
| `D` `C` `Y` | Shorthand for `d$`, `c$`, `y$` |

Every motion in the table above is a valid operator target, with counts on both
sides: `d2w`, `3dw`, `2d3w`, `c$`, `y3j`, `dfx`, `dtx`, `d}`, `dG`, `dgg`, `d%`.

Exclusive/inclusive is modelled explicitly, so `dw` stops before the character
it lands on while `de`, `d$`, `dfx` and `d%` include it, and `dtx` excludes the
target. Both of Vim's exclusive-motion adjustments are implemented (a motion
ending in column 1 backs up to the end of the previous line, and becomes
linewise when it started at or before the first non-blank), which is what makes
`d}` delete whole paragraph lines. `cw` on a non-blank behaves like `ce`, and a
`w` motion whose last word ends a line stops there instead of eating the
newline.

### Text objects

Only valid after an operator.

| Key | Object |
|---|---|
| `iw` `aw` `iW` `aW` | Word / WORD, inner or with surrounding whitespace |
| `i"` `a"` `i'` `a'` | Quoted string (line-scoped, as in Vim) |
| ``i` `` ``a` `` | Backtick-quoted string |
| `i(` `a(` `i)` `a)` `ib` `ab` | Parenthesised block |
| `i[` `a[` `i]` `a]` | Square-bracketed block |
| `i{` `a{` `i}` `a}` `iB` `aB` | Braced block |
| `i<` `a<` `i>` `a>` | Angle-bracketed block |
| `it` `at` | HTML/XML tag body, or the tag including its delimiters |
| `ip` `ap` | Paragraph (linewise) |
| `is` `as` | Sentence |

Bracket objects nest and span lines, and take a count (`d2i(` reaches the
enclosing pair). An inner block whose open brace ends a line and whose close
brace starts one becomes linewise, which is what makes `di{` clear a code body
without leaving a blank line.

### Visual modes

| Key | Action |
|---|---|
| `v` `V` `Ctrl-V` | Charwise / linewise / blockwise visual. The same key again exits; a different one switches mode and keeps the anchor. |
| `Esc` | Leave visual mode, cursor stays where the motion left it |
| `o` | Swap the cursor and the anchor |
| `O` | Blockwise: swap the corners horizontally. Elsewhere the same as `o`. |
| `gv` | Reselect the previous visual range (also after an operator ran on it) |

Every motion in the table above extends the selection from the anchor, counts
included (`v3l`, `v2e`, `Vjj`, `vfx`). Visual mode is *inclusive*, so `vwd`
deletes the character `w` lands on, unlike normal-mode `dw`.

| Operator | Action |
|---|---|
| `d` `x` | Delete the selection |
| `c` `s` | Delete the selection and insert |
| `y` | Yank the selection |
| `p` `P` | Replace the selection with the register; the replaced text becomes the new unnamed register |
| `>` `<` `=` | Shift / reindent the lines the selection touches (`{count}>` shifts repeatedly) |
| `gu` `gU` `g~` | Lowercase / uppercase / swap case |
| `u` `U` `~` | Visual-mode shorthands for `gu` / `gU` / `g~` |
| `r{c}` | Replace every selected character with `{c}` |
| `J` | Join the selected lines |
| `X` `D` `S` `C` `R` `Y` | Force the operation linewise |

Text objects (`iw`, `aw`, `i(`, `a"`, `ip`, `it`, … — the full table below) *set*
the selection to the object instead of running an operator, so `viw` selects a
word and `vipd` deletes a paragraph. A linewise object switches the mode to
`-- VISUAL LINE --`.

Blockwise adds:

| Key | Action |
|---|---|
| `I` | Insert at the left edge of every row |
| `A` | Append at the right edge of every row |
| `$` | Ragged right edge — every row runs to its own end |
| `d` `x` `c` `y` `r` `gu` `gU` `g~` | Operate on the column range of every row |

Blockwise `I`, `A` and `c` place a cursor on every row and hand typing to ttt's
native multi-cursor path, so the whole thing — the block edit plus everything
typed before `Esc` — is a single undo step. Rows too short to reach the block's
left edge are skipped rather than padded.

### Registers

A register holds text plus a *kind* — charwise, linewise or blockwise. `d`, `c`,
`y`, `x`, `X`, `s`, `S`, `D`, `C` and `Y` fill registers, and `p` / `P` read
them. A linewise register pastes onto a new line below / above, a charwise one
after / before the cursor, and a blockwise one re-inserts its rectangle
(padding short rows with spaces and creating lines past the end of the buffer).

`"{register}` prefixes an operator, a `p` / `P`, or a visual-mode operator.

| Register | Meaning |
|---|---|
| `""` | Unnamed — every yank and delete lands here |
| `"a`–`"z` | Named. `"A`–`"Z` *append* to the same slot. |
| `"0` | The last yank, untouched by deletes |
| `"1`–`"9` | The delete ring: each linewise or multi-line delete shifts it down |
| `"-` | The last small (single-line, charwise) delete |
| `"_` | Blackhole — writes are discarded and the unnamed register is left alone |
| `"+` `"*` | System clipboard (see the gaps below) |

### Marks

| Key | Action |
|---|---|
| `m{a-zA-Z}` | Set a mark at the cursor |
| `` `{mark} `` | Jump to the exact position |
| `'{mark}` | Jump to the first non-blank of the marked line |
| ``` `` ``` `''` | Jump back to the position before the last jump |
| `` `. `` `'.` | The position of the last change |
| ``d`a`` `d'a` | Marks are operator targets — backtick exclusive charwise, `'` linewise |

Marks are also settable and jumpable from visual mode, where a jump extends the
selection. Line numbers are kept correct across edits: an insert or delete above
a mark shifts it, and a mark inside a deleted span collapses onto the start of
that span.

### Macros

| Key | Action |
|---|---|
| `q{a-z}` | Start recording into a register |
| `q{A-Z}` | Append to an existing macro |
| `q` | Stop recording (the status bar shows `recording @q` while it runs) |
| `@{a-z}` | Replay |
| `@@` | Replay the last macro played |
| `{count}@{reg}` | Replay `{count}` times |

Recording captures canonical *tokens*, the same normalized form `token_of()`
produces, so a replay is literally "feed the tokens back through the dispatcher".
A replay is not itself recorded, recursion is capped at a depth of 10, and a
20000-key budget stops a macro that loops without recursing. Each operation in a
replay is its own undo step, as in Vim.

### Dot repeat

`.` repeats the last buffer-changing command, and `{count}.` re-runs it with a
new count. It replays a *resolved payload* — operator, target (motion, text
object, find, mark or doubled key), count, register and any text typed in insert
mode — rather than replaying keystrokes, so `.` is always exactly one undo step.

Covered: every operator with every target, the single-key edits (`x`, `X`, `D`,
`~`, `J`, `p`, `P`, `Ctrl-A`, `Ctrl-X`), `r{char}`, every insert-entry command
with its typed text (`i`, `I`, `a`, `A`, `o`, `O`, `s`, `S`, `C`, `c{motion}`),
and `R`. A yank changes nothing, so it leaves `.` armed with the previous change
rather than disarming it.

Counts on insert-entry commands work off the same change log: `3i`, `5a` and
`3o` type their text once and repeat it on `Esc`, all inside one undo step.

### Known gaps in Phase 4

- **The character under the cursor is not painted in a forward charwise
  selection.** ttt renders a selection as `Selection.Start` .. *live cursor*
  (`internal/core/selection`), so the anchor is the only value the plugin can
  choose. Keeping the real cursor on the Vim cursor means a forward range paints
  `[anchor, cursor)` and the cursor block itself stands in for the last
  character. A *backward* selection has no such problem — the anchor is shifted
  one column right and the range is exactly Vim's. Operators are unaffected:
  `visual_range()` is always inclusive.
- **`Ln, Col` is not the Vim cursor in `V` and `Ctrl-V` modes.** Linewise
  parks the real cursor at the end (or start) of the outer line so the whole
  line highlights, and blockwise draws its rows with `add_cursor`. The true Vim
  cursor is kept in Lua, so motions and operators are correct; only the status
  bar reading differs.
- **Blockwise `A` does not pad short rows.** Vim fills with spaces out to the
  block's right edge; here the append clamps to the end of the row.
- **Text objects replace the selection rather than growing it.** `viw` selects
  the word; repeating `iw` re-selects the same word instead of extending to the
  next one.
- **`Ctrl-V` overrides `editor.paste`** in normal and visual mode. It is an
  ordinary binding, not a force key, so the plugin interceptor wins. Paste stays
  reachable from insert mode, the Edit menu and the command palette.

### Known gaps in Phase 5

- **`"+` / `"*` cannot read or write text directly.** There is no clipboard
  binding in the plugin Lua API (`internal/core/clipboard` exists on the Go side
  but is not exposed), so `"+y` selects the range and runs `editor.copy`, and
  `"+p` positions the cursor and runs `editor.paste`. Two consequences: the
  clipboard carries no register kind, so `"+p` pastes whatever core makes of the
  text rather than honouring linewise/blockwise; and `{count}"+p` is one undo
  step per repetition, because core opens its own transaction and undo groups do
  not nest.
- **`.` does not repeat a visual-mode operator.** Vim re-applies it to a
  same-sized region from the cursor; here a visual operator records no payload,
  so `.` keeps repeating the last normal-mode change instead.
- **A macro only replays keys the plugin owns.** Pass-through keys (arrows,
  `Ctrl-S`, chords) are recorded but do nothing on replay, because the replay
  feeds tokens to the plugin dispatcher rather than to the terminal. Text typed
  in insert mode *is* replayed — the dispatcher inserts it directly.
- **An insert session containing an unrepeatable key is not repeatable.** Arrow
  keys or a paste inside insert mode mark the session dirty, and `.` then keeps
  the previous change rather than repeating a half-known one.
- **Marks are not per-buffer and uppercase marks are not global.** `m{A-Z}` sets
  a mark in the same table as `m{a-z}`; switching tabs does not switch marks.
- **A mark inside deleted text collapses rather than being invalidated.** Vim
  drops such a mark; here it moves to the start of the deleted span.

### Known gaps in Phases 2-3

- **`o` / `O` do not autoindent.** They open a column-1 line, matching vanilla
  Vim without `autoindent`.
- **`==` is a heuristic, not an indent engine.** ttt has none, so `==` copies
  the previous non-blank line's (space) indent, adds one shiftwidth if that line
  ends with `{`/`(`/`[`/`:`, and removes one if this line starts with a closing
  bracket.
- **Shiftwidth is hardcoded to 4 spaces.** Reading `editor.tabSize` needs a
  `settings` permission the manifest does not request; wiring it is Phase 7.
- **`Y` is `y$`, not `yy`.** ttt follows Neovim's default rather than Vim's, so
  `Y` lines up with `D` and `C`.
- **`cc` / `S` always preserve indentation**, where Vim only does so with
  `autoindent` set.
- **`it` / `at` scan a 500-line window** around the cursor rather than the whole
  buffer, to keep the key-press path bounded. Tags nested further apart than
  that are not found.
- **`is` / `as` are scoped to the paragraph** around the cursor, and recognise
  `.`/`!`/`?` (plus trailing quotes and brackets) as sentence ends. Vim's
  abbreviation handling is not reproduced.

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

**Marks ride on an editor shim.** There is no buffer-change event to hang mark
adjustment off, so `init.lua` wraps `editor.insert`, `editor.replace` and
`editor.set_line` in a table that forwards everything else to the real module
through `__index`. The wrappers shift mark line numbers by the newline delta of
the edit and record the `.` mark. The module itself is never mutated, so no
other plugin sees the override.

**Cursor writes scroll.** `set_cursor` calls `EnsureCursorVisible` on the Go
side, so every scrolling routine moves the cursor *first* and calls `scroll_to`
*last* — the other order gets silently undone.

**gopher-lua does not swap.** Multiple assignment is *not* simultaneous when a
target also appears on the right-hand side: `a, b = b, a` evaluates to `b, b`,
and `sl, sc, el, ec = el, ec, sl, sc` to `el, ec, el, ec`. This is a gopher-lua
register-allocation bug, not a Lua semantic — real Lua 5.1 swaps correctly. Every
swap in `init.lua` therefore goes through explicit temporaries. It is a silent
wrong-answer bug, so treat any `x, y = y, x` in a ttt plugin as broken.

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
cd tests/functional && npx vitest run vim-mode.test.js vim-motions.test.js \
  vim-edits.test.js vim-operators.test.js vim-textobjects.test.js \
  vim-visual.test.js vim-registers.test.js vim-macros.test.js
```
