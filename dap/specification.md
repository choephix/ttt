# Debug Adapter Protocol (DAP) Specification Reference

Source: https://microsoft.github.io/debug-adapter-protocol/specification

## Base Protocol

DAP uses JSON-RPC-style messaging with Content-Length framing (same as LSP):

- **Headers**: `Content-Length: <bytes>\r\n\r\n`
- **Body**: UTF-8 JSON object

### Message Types

**ProtocolMessage** (base):
- `seq`: Sequence number (starts at 1, increments per actor)
- `type`: `'request'` | `'response'` | `'event'`

**Request**: `command` + optional `arguments`
**Response**: `request_seq` + `success` + `command` + optional `body`/`message`
**Event**: `event` + optional `body`

## Initialization Sequence

1. Client sends `initialize` request with capabilities
2. Adapter responds with `InitializeResponse` (supported capabilities)
3. Adapter sends `initialized` event
4. Client sends configuration requests (setBreakpoints, setExceptionBreakpoints, etc.)
5. Client sends `configurationDone` request
6. Execution begins

## Request Types

### Session Control

| Request | Purpose | Key Arguments |
|---------|---------|---------------|
| `initialize` | Establish protocol version and capabilities | clientID, clientName, adapterID, locale |
| `launch` | Start debuggee | noDebug flag, implementation-specific params |
| `attach` | Connect to running debuggee | implementation-specific connection details |
| `configurationDone` | Signal setup complete | none |
| `disconnect` | End debugging session | restart, terminateDebuggee, suspendDebuggee |
| `terminate` | Gracefully shut down debuggee | restart flag |
| `restart` | Restart debug session | launch or attach config |

### Execution Control

| Request | Purpose | Key Arguments |
|---------|---------|---------------|
| `continue` | Resume execution | threadId, singleThread |
| `next` | Step over (one statement) | threadId, singleThread, granularity |
| `stepIn` | Step into function | threadId, singleThread, targetId, granularity |
| `stepOut` | Step out of function | threadId, singleThread, granularity |
| `stepBack` | Reverse one step | threadId, singleThread, granularity |
| `reverseContinue` | Resume backward | threadId, singleThread |
| `pause` | Suspend execution | threadId |
| `goto` | Jump to target location | threadId, targetId |
| `restartFrame` | Re-execute stack frame | frameId |

### Breakpoint Management

| Request | Purpose | Key Arguments |
|---------|---------|---------------|
| `setBreakpoints` | Set line breakpoints | source, breakpoints[], sourceModified |
| `setFunctionBreakpoints` | Set function breakpoints | breakpoints[] (name-based) |
| `setDataBreakpoints` | Set memory/variable breakpoints | breakpoints[] with dataId, accessType |
| `setInstructionBreakpoints` | Set disassembly breakpoints | breakpoints[] with instructionReference |
| `setExceptionBreakpoints` | Configure exception handling | filters[], filterOptions[], exceptionOptions[] |
| `breakpointLocations` | Query valid positions | source, line, column range |

### Stack Inspection

| Request | Purpose | Key Arguments |
|---------|---------|---------------|
| `stackTrace` | Get call stack | threadId, startFrame, levels, format |
| `scopes` | Get variable scopes for frame | frameId |
| `variables` | Get child variables | variablesReference, filter, start, count |
| `setVariable` | Modify variable value | variablesReference, name, value |
| `evaluate` | Execute expression | expression, frameId, context |

### Thread & Source

| Request | Purpose | Key Arguments |
|---------|---------|---------------|
| `threads` | List all threads | none |
| `source` | Retrieve source content | source, sourceReference |
| `modules` | List loaded modules | startModule, moduleCount |
| `loadedSources` | Get all source files | none |

### Completions

| Request | Purpose | Key Arguments |
|---------|---------|---------------|
| `completions` | Code completion in debug console | frameId, text, column, line |

## Event Types

### Execution State

| Event | Purpose | Key Body Fields |
|-------|---------|-----------------|
| `initialized` | Adapter ready for config | none |
| `stopped` | Execution paused | reason (step/breakpoint/exception/pause/entry), threadId, allThreadsStopped, hitBreakpointIds[] |
| `continued` | Execution resumed | threadId, allThreadsContinued |
| `exited` | Debuggee exited | exitCode |
| `terminated` | Debugging ended | restart flag |

### Resources

| Event | Purpose | Key Body Fields |
|-------|---------|-----------------|
| `thread` | Thread created/destroyed | reason (started/exited), threadId |
| `module` | Module loaded/changed/unloaded | reason, module |
| `loadedSource` | Source added/changed/removed | reason, source |
| `process` | Attached to process | name, systemProcessId, startMethod |
| `breakpoint` | Breakpoint state changed | reason (changed/new/removed), breakpoint |
| `memory` | Memory modified | memoryReference, offset, count |

### Output

| Event | Purpose | Key Body Fields |
|-------|---------|-----------------|
| `output` | Debuggee produced output | category (console/stdout/stderr), output, source, line |

### Progress

| Event | Purpose | Key Body Fields |
|-------|---------|-----------------|
| `progressStart` | Long operation started | progressId, title, cancellable, percentage |
| `progressUpdate` | Progress changed | progressId, message, percentage |
| `progressEnd` | Progress completed | progressId, message |

## Reverse Requests (Adapter-to-Client)

| Request | Purpose | Key Arguments |
|---------|---------|---------------|
| `runInTerminal` | Run command in client terminal | kind (integrated/external), cwd, args[], env |
| `startDebugging` | Start new debug session | configuration, request (launch/attach) |

## Key Types

### Breakpoint
```
id?, verified, message?, source?, line?, column?, endLine?, endColumn?
```

### SourceBreakpoint
```
line, column?, condition?, hitCondition?, logMessage?
```

### StackFrame
```
id, name, source?, line, column?, endLine?, endColumn?, canRestart?, moduleId?
```

### Scope
```
name, presentationHint?, variablesReference, namedVariables?, indexedVariables?, expensive
```

### Variable
```
name, value, type?, evaluateName?, variablesReference, namedVariables?, indexedVariables?, memoryReference?
```

### Source
```
name?, path?, sourceReference?, presentationHint?, origin?, sources?, checksums?
```

### Thread
```
id, name, state? (running/stopped)
```

### Capabilities (subset most relevant to implementation)
```
supportsConfigurationDoneRequest
supportsFunctionBreakpoints
supportsConditionalBreakpoints
supportsHitConditionalBreakpoints
supportsEvaluateForHovers
supportsSetVariable
supportsLogPoints
supportsTerminateRequest
supportsDataBreakpoints
supportsCancelRequest
supportsBreakpointLocationsRequest
supportsStepBack
supportsRestartFrame
supportsCompletionsRequest
supportsRestartRequest
supportsRunInTerminalRequest
supportsProgressReporting
```

## Granularity

Step granularity for next/stepIn/stepOut/stepBack:
- `statement` — single statement (default)
- `line` — single line
- `instruction` — single instruction

## Stopped Reasons

Values for `stopped` event `reason` field:
- `step` — step completed
- `breakpoint` — breakpoint hit
- `exception` — exception thrown
- `pause` — user paused
- `entry` — entry point reached
- `goto` — goto completed
- `function breakpoint` — function breakpoint hit
- `data breakpoint` — data breakpoint hit
- `instruction breakpoint` — instruction breakpoint hit
