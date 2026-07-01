# v1.0.0 Release Issues

## Critical
- [x] #291 Race: RequestHover timer goroutine reads shared editor state
- [x] #292 Race: RequestCodeAction reads Buf.Lines from background goroutine

## High
- [x] #293 Data loss: CloseTab Save does not check save success before closing
- [x] #294 Stale undo stack after external file reload causes buffer corruption
- [ ] #295 Long lines (>64KB) silently abort file loading
- [ ] #296 Race: Terminal OnUpdate/OnExit callbacks assigned after goroutine starts
- [ ] #297 clampCursor does not clamp column to line length
- [ ] #298 Autocomplete hover tracks mouse outside menu bounds
- [ ] #299 Autocomplete mouse interaction broken with sidebar visible
- [ ] #308 LSP call() blocks forever if server hangs

## Medium
- [ ] #290 Plugin: auto-disable after repeated Lua errors and surface errors in output panel
- [ ] #300 Incorrect 256-color palette in terminal emulator (colors 16-231)
- [ ] #301 Missing Shift+Tab (BackTab) key translation in terminal
- [ ] #302 MoveLineUp/Down and JoinLines with selection produce non-atomic undo
- [ ] #303 Missing HasOverlay() guards on 6 keybinding-reachable commands
- [ ] #304 Terminal modifier keys not encoded for arrow/nav keys
- [ ] #305 TrimTrailingWhitespace and InsertFinalNewline mutate buffer outside undo system
- [ ] #306 DeleteWordLeft/Right ignores active selection
- [ ] #307 Dialog footer unreachable on small terminal screens
- [ ] #309 DeleteLine allows emptying buffer to zero lines
