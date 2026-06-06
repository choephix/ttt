# DAP Implementation Plan

## Overview

Add debug adapter protocol support to ttt. DAP uses the same Content-Length framing as LSP, so `internal/lsp/jsonrpc.go` can be extracted into a shared package and reused. The implementation is split into 5 phases, each independently testable and shippable.

## Architecture

```
internal/
  jsonrpc/          <- extracted from lsp/, shared by both lsp/ and dap/
    codec.go        <- Codec, Request, Response types
    codec_test.go
  dap/
    protocol.go     <- DAP message types
    client.go       <- DAP client (mirrors lsp/client.go pattern)
    client_test.go
    manager.go      <- adapter lifecycle management
  ui/
    debug_toolbar.go  <- play/pause/step buttons
    debug_panel.go    <- variables, call stack, breakpoints (tabbed in bottom panel)
    breakpoint.go     <- breakpoint gutter markers + state
  app/
    app_dap.go        <- DAP integration (mirrors app_lsp.go)
    commands_debug.go <- debug commands
```

## Debug Adapters

DAP adapters are standalone executables (like LSP servers). Common ones:

| Language | Adapter | Command |
|----------|---------|---------|
| Go | Delve | `dlv dap` |
| Python | debugpy | `python -m debugpy --listen 5678` |
| Node.js | node debug | `node --inspect` |
| C/C++ | lldb-dap | `lldb-dap` |
| Rust | lldb-dap / codelldb | `lldb-dap` |

Configuration will go in `settings.json` under a `"debug"` key, similar to `"lsp"`:

```json
{
  "debug": {
    "adapters": {
      "go": { "command": ["dlv", "dap"] },
      "python": { "command": ["python", "-m", "debugpy.adapter"] }
    }
  }
}
```

---

## Phase 1: Protocol Layer

Extract shared JSON-RPC codec, implement DAP client with basic lifecycle.

### Scope
- Extract `internal/lsp/jsonrpc.go` into `internal/jsonrpc/` package
- Update `internal/lsp/` to import from `internal/jsonrpc/`
- Create `internal/dap/protocol.go` with core DAP types:
  - ProtocolMessage, Request, Response, Event
  - InitializeRequest/Response, LaunchRequest, AttachRequest
  - ConfigurationDoneRequest, DisconnectRequest
  - Capabilities
- Create `internal/dap/client.go`:
  - Start adapter process via stdio
  - Read loop for responses and events
  - `call()` and `notify()` methods (same pattern as LSP client)
  - `Initialize()`, `Launch()`, `ConfigurationDone()`, `Disconnect()`
- Create `internal/dap/manager.go`:
  - Adapter config loading from settings
  - Adapter lifecycle (start, shutdown, timeout kill)

### Testing
- **Unit**: `internal/jsonrpc/codec_test.go` — verify framing still works after extraction
- **Unit**: `internal/dap/client_test.go` — mock adapter (pipe-based), test initialize handshake, test event dispatch, test disconnect cleanup
- **Manual**: `dlv dap` subprocess starts and responds to initialize

### Deliverable
DAP client can connect to an adapter, complete the initialization handshake, launch a program, and disconnect cleanly. No UI yet.

---

## Phase 2: Breakpoints and Execution Control

Add breakpoint state management, stepping commands, and the stopped/continued event loop.

### Scope
- Add to `internal/dap/protocol.go`:
  - SetBreakpointsRequest/Response, Source, SourceBreakpoint, Breakpoint
  - ContinueRequest/Response, NextRequest, StepInRequest, StepOutRequest
  - PauseRequest, TerminateRequest
  - StoppedEvent, ContinuedEvent, ExitedEvent, TerminatedEvent
  - ThreadsRequest/Response, Thread
- Add to `internal/dap/client.go`:
  - `SetBreakpoints()`, `Continue()`, `Next()`, `StepIn()`, `StepOut()`
  - `Pause()`, `Terminate()`, `Threads()`
  - Event handlers: `OnStopped`, `OnContinued`, `OnExited`, `OnTerminated`, `OnOutput`
- Create `internal/ui/breakpoint.go`:
  - Breakpoint state per file: `map[string]map[int]bool` (path -> line -> enabled)
  - Toggle breakpoint at cursor line
  - Gutter rendering: red dot marker for breakpoint lines
- Wire breakpoint toggle into editor gutter click + keyboard shortcut

### Testing
- **Unit**: `internal/dap/client_test.go` — mock adapter returns setBreakpoints response with verified=true, test stepping sequence (launch -> stopped -> next -> stopped -> continue -> exited)
- **Unit**: `internal/ui/breakpoint_test.go` — toggle on/off, verify gutter markers
- **E2E**: `tests/e2e/debug_test.go` — toggle breakpoint via command, verify state
- **Manual**: Set breakpoint in a Go file, launch `dlv dap`, verify stopped event fires at the right line

### Deliverable
Can set breakpoints visually, launch a debug session, hit a breakpoint, and step through code. Output goes to the integrated terminal.

---

## Phase 3: Stack Inspection UI

Show call stack, variables, and scopes when stopped at a breakpoint.

### Scope
- Add to `internal/dap/protocol.go`:
  - StackTraceRequest/Response, StackFrame
  - ScopesRequest/Response, Scope
  - VariablesRequest/Response, Variable
- Add to `internal/dap/client.go`:
  - `StackTrace()`, `Scopes()`, `Variables()`
- Create `internal/ui/debug_panel.go`:
  - Tabbed panel with three sub-panels: Call Stack, Variables, Breakpoints
  - Call Stack: list of stack frames, click to navigate to source location
  - Variables: tree view of scopes -> variables (expand nested objects)
  - Breakpoints: list of all set breakpoints with file:line, click to navigate
- Add debug panel as a tab in the bottom panel (alongside Terminal, Problems)
- Wire stopped event: auto-fetch threads -> stackTrace -> scopes -> variables
- Navigate editor to stopped location (highlight current line)

### Testing
- **Unit**: `internal/dap/client_test.go` — mock stack trace, scopes, variables responses
- **Unit**: `internal/ui/debug_panel_test.go` — render call stack list, expand variable tree
- **E2E**: `tests/e2e/debug_test.go` — verify debug panel appears when session starts, verify stack frame navigation opens correct file
- **Manual**: Hit breakpoint in Go program, verify call stack shows correct frames, variables panel shows locals with correct values

### Deliverable
Full debugging inspection: hit a breakpoint, see where you are in the call stack, inspect local variables, navigate between frames.

---

## Phase 4: Debug Toolbar and Session Management

Add debug controls toolbar and proper session lifecycle.

### Scope
- Create `internal/ui/debug_toolbar.go`:
  - Floating toolbar (or status bar integration) with: Continue, Step Over, Step In, Step Out, Restart, Stop
  - Show/hide based on active debug session
  - Visual state: running (grayed step buttons) vs stopped (all active)
- Add to `internal/app/commands_debug.go`:
  - `debug.start` — launch debug session for current file
  - `debug.continue`, `debug.stepOver`, `debug.stepIn`, `debug.stepOut`
  - `debug.pause`, `debug.stop`, `debug.restart`
  - `debug.toggleBreakpoint`
- Add keybindings:
  - F5: Start/Continue
  - F10: Step Over
  - F11: Step In
  - Shift+F11: Step Out
  - Shift+F5: Stop
  - Ctrl+Shift+F5: Restart
  - F9: Toggle Breakpoint
- Add debug adapter configuration in `settings.json`:
  - `debug.adapters` map (language -> command)
- Add `internal/app/app_dap.go`:
  - Debug session state on App struct
  - Wire DAP events to UI updates via PostEvent
  - Handle stopped -> fetch state -> update panels
  - Handle exited/terminated -> cleanup session, remove toolbar
- Current-line highlight in editor when stopped (distinct from selection highlight)

### Testing
- **Unit**: `internal/ui/debug_toolbar_test.go` — render states (running vs stopped vs no session)
- **E2E**: `tests/e2e/debug_test.go` — F9 toggles breakpoint, F5 starts session (mock adapter), toolbar appears, Stop cleans up
- **Manual**: Full debug workflow with `dlv dap` — set breakpoint, F5 to start, hit breakpoint, F10 to step, inspect variables, F5 to continue, program exits, toolbar disappears

### Deliverable
Complete debug UX: toolbar with step controls, keyboard shortcuts, session lifecycle from start to finish.

---

## Phase 5: Advanced Features

Polish and extend with less critical but useful features.

### Scope
- **Evaluate expression**: debug console input in bottom panel, evaluate in current frame context
- **Conditional breakpoints**: right-click breakpoint to add condition expression
- **Exception breakpoints**: configure via `setExceptionBreakpoints`
- **Attach mode**: connect to running process (pid picker dialog or port input)
- **Output event routing**: route `stdout`/`stderr` to debug console panel, `console` category to debug output
- **Set variable**: click variable value to edit in-place
- **Watch expressions**: persistent list of expressions re-evaluated on each stop
- **Launch configurations**: `launch.json` file support for project-specific debug configs
- **Multi-adapter**: run different adapters for different languages in the same session

### Testing
- **Unit**: evaluate request/response parsing, conditional breakpoint serialization
- **E2E**: evaluate expression returns result, conditional breakpoint only stops when condition is true
- **Manual**: attach to running Go process, set watch expression, modify variable value

### Deliverable
Feature-complete DAP implementation suitable for daily use.

---

## Reuse Summary

| Existing Code | Reuse For |
|---------------|-----------|
| `internal/lsp/jsonrpc.go` | Extract to `internal/jsonrpc/`, shared codec |
| `internal/lsp/client.go` pattern | Same `call()`/`notify()`/`readLoop()` pattern for DAP client |
| `internal/lsp/manager.go` pattern | Same adapter lifecycle management |
| `internal/ui/problems_widget.go` | Reference for tree-view panel (variables panel) |
| `internal/ui/references_widget.go` | Reference for navigable list panel (call stack) |
| Bottom panel infrastructure | Debug panel tabs slot in alongside Terminal/Problems |
| `PostEvent(EventInterrupt)` pattern | Async DAP events -> UI updates |
| `internal/config/` | Debug adapter config alongside LSP config |
| Command registry + keybindings | Debug commands and F-key bindings |

## Estimated Size

| Phase | New Lines | Files |
|-------|-----------|-------|
| 1: Protocol | ~600 | 5-6 |
| 2: Breakpoints | ~800 | 4-5 |
| 3: Stack UI | ~700 | 3-4 |
| 4: Toolbar | ~800 | 4-5 |
| 5: Advanced | ~600 | 3-4 |
| **Total** | **~3,500** | **~20** |
