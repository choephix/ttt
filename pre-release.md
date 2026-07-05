# Pre-Release Worklist — Plugin System

What the plugin system needs before v1.0.0 so third-party developers can build real
functionality. Sourced from the full API audit and QA sweep (PR #324). Work these
**one at a time** — each item is its own branch/PR.

## Critical — API contract, must land before 1.0

These are manifest/API format decisions. Everything else can be *added* compatibly
later; these can't be *changed* compatibly later.

- [ ] **1. Manifest API version** — DECIDED: doing it.
  Add optional `"api": 1` to `plugin.ttt.json`. A missing field is assumed to be v1,
  so existing plugins keep working unchanged. Loader refuses (with a clear OUTPUT
  error) to load plugins declaring an API newer than the editor supports. Update the
  manifest docs and, afterwards, the plugins in the ttt-plugins repo to declare it.

- [ ] **2. Per-plugin persistent storage (`ttt.storage`)**
  Sanctioned KV storage (JSON values) persisted per plugin, outside the workspace.
  Today plugins have nowhere to put state: notepad wrote `.ttt-notepad.json` into the
  workspace (it ended up committed to this repo), and `ttt.settings` writes into the
  user's settings.json. Consider a workspace-scoped variant (`storage.workspace`).
  Partial: `ttt.plugin_dir()` landed (fd6f87b) — solves path discovery. Still open:
  state via `fs.write` requires the broad fs.write permission, and the plugin dir is
  a git clone (update conflicts) that's deleted on uninstall.

- [x] **3. Timers (`ttt.set_interval` / `ttt.set_timeout`)** — DONE (PR pending).
  `set_timeout`/`set_interval` return ids; `clear_timeout`/`clear_interval` cancel
  (interchangeable). Callbacks dispatch through the PostAsync main-loop path, so they
  can touch state safely. `set_interval` floored at 50ms. All timers stopped on
  Destroy (disable/reload/uninstall). Unit + functional tests, docs-web section.

- [ ] **4. Streaming, cancellable exec (`sys.spawn`)**
  `exec_async` buffers all output until exit — test runners, build watchers, and log
  tails can't stream and can't be cancelled. `sys.spawn(binary, args, {on_line,
  on_exit}) -> handle` with `handle:kill()` and `handle:write()` (stdin). Kill
  processes on plugin destroy/reload. go-test-runner is the in-house proof case.

- [x] **5. Network domain scoping in the manifest** — DONE (PR pending).
  `network.http` accepts `bool | []string`: `true` = any host (unchanged),
  `["api.github.com"]` = exact-host allowlist, absent/`false` = no network.
  Enforced in `ttt.net` (all four get/post/async), approval dialog lists hosts,
  diff requires re-approval for new hosts or escalation to all-hosts. No `*` — `true`
  covers it. Honestly scoped in docs: governs `ttt.net` only, not exec of curl/gh/git.

## High value — makes plugins first-class, strong candidates before 1.0

- [ ] **6. Editor integration points**
  - `ttt.diagnostics.publish(path, items)` — plugins as linters, feeding the existing
    squiggles/problems pipeline
  - Status bar segments (git-blame style) registered by plugins
  - Gutter marks / line decorations (coverage, bookmarks, review comments)
  - `editor.save()`, access to non-active buffers, read-only text tabs
- [ ] **7. More events**
  `tab.change` (active file switched), `workspace.change`, `theme.change`,
  `selection.change`, app shutdown hook. Each is a one-line DispatchEvent at an
  existing site.
- [ ] **8. Small API/UI gaps** (each trivial)
  - `ttt.prompt(title, placeholder, cb)` — input dialog (only confirm/show_info exist)
  - `ttt.notify(message)` — status bar toast (StatusNotify exists internally)
  - Expose `p:checkbox` and `p:select` (widgets exist in Go, not exposed to Lua)
  - Tree node `badge_style` / `icon_style` parseable from Lua (Go fields exist)

## Nice to have — post-1.0, as the ecosystem demands

- [ ] **9. "Plugins: Create New Plugin" scaffold command** — manifest + init.lua in
  the local plugins dir, opens the entry file
- [ ] **10. Registry version pinning** — install/update to tags instead of `git pull`
  whatever main is
- [x] **11. Plugin testing recipe docs** — DONE. New `guides/plugin-testing.md`:
  `--plugin`/`--exec` harness, screenshot-vs-debug assertion pattern, `TTT_CONFIG_DIR`
  isolation, testing scoped plugins via a pre-approved registry, timer example, and
  `tui.js` wiring. Sidebar entry added.
- [ ] **12. Render-time watchdog** — plugin render runs on the UI thread; log a
  warning when a plugin's render exceeds a budget (the 10-error auto-disable catches
  crashes, not slowness)

## Working agreement

- One item at a time, one PR per item, review between items.
- Items 1–5 gate the 1.0 release; 6–8 are judgment calls; 9–12 explicitly don't block.
- After item 1 lands, update the plugins in the ttt-plugins repo to declare `"api": 1`.
- This file is a working document — delete it before the release.
