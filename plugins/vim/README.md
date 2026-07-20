# Vim Mode

A Vim compatibility layer for ttt, implemented entirely as a Lua plugin.
Tracks [issue #386](https://github.com/eugenioenko/ttt/issues/386).

## Status

Under construction, built in phases. Currently at **Phase 0**.

| Phase | Scope | State |
|---|---|---|
| 0 | Mode state machine, normal/insert, Esc, status indicator | ✅ |
| 1 | Motions and counts | ⬜ |
| 2 | Insert mode and simple edits | ⬜ |
| 3 | Operators and text objects | ⬜ |
| 4 | Visual modes | ⬜ |
| 5 | Registers, marks, macros, `.` repeat | ⬜ |
| 6 | Ex command line, search, editor integration | ⬜ |
| 7 | Settings, docs, handoff to `ttt-plugins` | ⬜ |

`cheatsheet.md` is a vendored copy of [ibhagwan/vim-cheatsheet](https://github.com/ibhagwan/vim-cheatsheet),
used as the coverage checklist.

## Supported today

| Key | Action |
|---|---|
| `i` | Enter insert mode |
| `Esc`, `Ctrl-[` | Return to normal mode |
| — | Printable keys in normal mode are swallowed, never typed |

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
raises a one-time permission dialog. Passing `--plugin` explicitly grants all
permissions and skips approval.

Tests: `cd tests/functional && npx vitest run vim-mode.test.js`
