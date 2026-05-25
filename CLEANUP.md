# Cleanup Opportunities

## Resolved

- [x] Race condition on `docVersions` map — added `sync.Mutex`, initialized in `buildApp()`
- [x] Silent `SaveFile` / `LoadFile` errors — added `OnError` callback, surfaced via status bar
- [x] Empty `workDir` passed to LSP — extracted `lspWorkDir()` helper, used consistently
- [x] `docVersions` lazy init in multiple places — initialized once in `buildAppFromConfig()`
- [x] `registerCommands()` is 1000+ lines — split into 7 domain functions
- [x] Terminal tab dual source of truth — stored `id` in `terminalTab`, clean removal on close
- [x] `StatusNotify/Warn/Error` are 3 copies — consolidated into `statusMessage()`, removed dead wrappers from StatusBar
- [x] LSP server availability check repeated — extracted `lspReady()` helper
- [x] Shared click-area helper — `HitRegion` type applied to ConfirmDialog and FindBar

## Not applicable

- TabBar hit-testing — uses scroll-offset virtual coordinates, not screen coordinates. `tabSpan` is already a clean per-tab structure computed in Render. HitRegion doesn't fit this pattern.
