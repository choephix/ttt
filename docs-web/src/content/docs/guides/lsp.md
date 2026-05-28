---
title: LSP
description: Language Server Protocol support in TTT.
---

TTT has built-in LSP support for language-aware editing features. Language servers run as external processes and communicate over JSON-RPC 2.0 via stdio.

## Built-in Language Support

TTT ships with built-in configurations for popular language servers. You just need to install the server binary — TTT will detect and use it automatically. If a server isn't installed, TTT shows a brief notification when you open a file of that language.

To disable LSP entirely, set `lsp.enabled` to `false` in your settings:

```json
{
  "lsp": {
    "enabled": false
  }
}
```

### Go {#go}

```sh
go install golang.org/x/tools/gopls@latest
```

### TypeScript / JavaScript {#typescript}

```sh
npm install -g typescript typescript-language-server
```

Handles `.ts`, `.tsx`, `.js`, and `.jsx` files.

### Python {#python}

```sh
pip install pyright
# or
npm install -g pyright
```

### C / C++ {#c}

```sh
# Ubuntu/Debian
sudo apt install clangd

# macOS
brew install llvm

# Arch
sudo pacman -S clang
```

Handles `.c`, `.h`, `.cpp`, `.hpp`, `.cc`, and `.cxx` files.

### Rust {#rust}

```sh
rustup component add rust-analyzer
```

### Lua {#lua}

Install from [LuaLS releases](https://github.com/LuaLS/lua-language-server/releases) or via your package manager:

```sh
# macOS
brew install lua-language-server

# Arch
sudo pacman -S lua-language-server
```

### Zig {#zig}

```sh
# From zigtools releases or package manager
# See https://github.com/zigtools/zls
```

## Custom Language Servers

Add or override language servers in `~/.config/ttt/settings.json` under the `lsp.servers` key. Each entry maps a server key to a config object with a `command` array:

```json
{
  "lsp": {
    "servers": {
      "ruby": { "command": ["solargraph", "stdio"] }
    }
  }
}
```

For simple cases, the server key is used as the language ID and files are matched via the syntax highlighter name. No extra configuration is needed.

### The `languages` field

The optional `languages` field is for servers that handle multiple file types requiring different `languageId` values. It maps file extensions to language IDs:

```json
{
  "lsp": {
    "servers": {
      "typescript": {
        "command": ["typescript-language-server", "--stdio"],
        "languages": {
          ".ts": "typescript",
          ".tsx": "typescriptreact",
          ".js": "javascript",
          ".jsx": "javascriptreact"
        }
      }
    }
  }
}
```

Without the `languages` field, you would need to register the same server command multiple times under different keys.

The server is started lazily on first use and shut down when the editor exits.

## Supported Features

| Feature | Keybinding | Description |
|---------|-----------|-------------|
| Autocomplete | Ctrl+U | Trigger completion at cursor position |
| Signature Help | *(automatic)* | Parameter hints shown on `(` and `,` |
| Go to Definition | F12 | Jump to the definition of the symbol under the cursor |
| Go to Implementation | Shift+F12 | Jump to the implementation |
| Go to Type Definition | Ctrl+L T | Jump to the type definition |
| Find References | Ctrl+L R | Find all references (results in bottom panel) |
| Rename Symbol | F2 | Rename across all files in the workspace |
| Hover | Ctrl+K I | Show type information and documentation |
| Format Document | Ctrl+L F | Format the entire document |
| Format Selection | Ctrl+L S | Format the selected range |
| Organize Imports | Ctrl+L O | Organize imports via code action |
| Fix All | Ctrl+L X | Apply all available fixes |
| Diagnostics | *(automatic)* | Error/warning squiggles, status bar summary, hover popup |

## Auto-Completion

Completions trigger automatically as you type with a configurable debounce (default 150ms). The completion list filters live as you continue typing, and supports `completionItem/resolve` for additional details and auto-imports.

**Ctrl+U** also works as a manual trigger.

### Configuration

```json
{
  "autocomplete": {
    "enabled": true,
    "autoSuggest": true,
    "debounce": 150,
    "signatureHelp": true
  }
}
```

## Diagnostics

The LSP server publishes diagnostics (errors, warnings, hints) which are displayed as:

- **Inline squiggles** on the affected range, colored by severity
- **Problems panel** in the bottom panel listing all diagnostics grouped by file
- **Hover popup** over a squiggle to see the diagnostic message
- **Status bar** error/warning counts

## Formatting

- **Format Document** (Ctrl+L F) formats the entire file
- **Format Selection** (Ctrl+L S) formats only the selected range
- **Format on Save** can be enabled in settings

```json
{
  "formatOnSave": true
}
```

## References and Rename

- **Find All References** (Ctrl+L R) shows all usages in the bottom panel REFERENCES tab
- **Rename Symbol** (F2) renames across all files in the workspace, applying multi-file workspace edits automatically

Enable `lsp.saveOnRename` to auto-save files affected by a rename:

```json
{
  "lsp": {
    "saveOnRename": true
  }
}
```

## Code Actions on Save

Configure automatic code actions that run before each save:

```json
{
  "lsp": {
    "codeActionsOnSave": [
      "source.organizeImports",
      "source.fixAll"
    ]
  }
}
```
