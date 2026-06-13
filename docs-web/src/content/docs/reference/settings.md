---
title: Settings
description: Complete settings reference for TTT.
sidebar:
  order: 1
---

Settings are stored in `~/.config/ttt/settings.json`. A complete example is available at [`config/settings.json`](https://github.com/eugenioenko/ttt/blob/main/config/settings.json) in the repository.

You can open your settings file directly from the command palette (**Ctrl+P**) with **Preferences: Open Settings**.

## Editor

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `tabSize` | int | `4` | Number of spaces per indentation level |
| `insertSpaces` | bool | `true` | Use spaces instead of tabs for indentation |
| `wordWrap` | bool | `false` | Wrap long lines at the editor width |
| `lineNumbers` | bool | `true` | Show line numbers in the gutter |
| `sidebarVisible` | bool | `true` | Show the sidebar on startup |
| `sidebarWidth` | int | `30` | Width of the sidebar in columns |
| `cursorStyle` | string | `""` | Cursor style: `"block"`, `"underline"`, or `"bar"` |
| `theme` | string | `""` | Theme name |
| `debugMode` | bool | `false` | Enable debug logging |
| `formatOnSave` | bool | `false` | Auto-format the document via LSP on save |
| `insertFinalNewline` | bool | `true` | Ensure files end with a newline on load and save |
| `bracketPairColorization` | bool | `false` | Colorize matching bracket pairs by nesting depth |

## Explorer

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `explorer.showHidden` | bool | `true` | Show hidden files (dot-prefixed) in the file explorer |
| `explorer.showGitIgnored` | bool | `true` | Show gitignored files in the file explorer |

## Terminal

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `terminal.shell` | string | `""` | Shell command (empty uses system default) |
| `terminal.scrollback` | int | `1000` | Number of scrollback lines to retain |

## LSP

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `lsp.saveOnRename` | bool | `false` | Auto-save files affected by a rename operation |
| `lsp.codeActionsOnSave` | string[] | `[]` | Code actions to run before save (e.g. `"source.organizeImports"`) |
| `lsp.servers` | object | `{}` | Map of server key to `{ "command": [...], "languages": {...} }`. The optional `languages` field maps file extensions to language IDs for servers handling multiple file types. |

## Search

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `search.debounce` | int | `350` | Milliseconds to debounce global search input |

## Autocomplete

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `autocomplete.enabled` | bool | `true` | Enable LSP-powered autocompletion |
| `autocomplete.autoSuggest` | bool | `true` | Show completions automatically as you type |
| `autocomplete.debounce` | int | `150` | Milliseconds to wait after typing before requesting completions |
| `autocomplete.signatureHelp` | bool | `true` | Show function signature help on `(` and `,` |

## Full Example

```json
{
  "tabSize": 4,
  "insertSpaces": true,
  "wordWrap": false,
  "lineNumbers": true,
  "sidebarVisible": true,
  "sidebarWidth": 30,
  "theme": "default-dark",
  "formatOnSave": true,
  "insertFinalNewline": true,
  "bracketPairColorization": false,
  "search": {
    "debounce": 350
  },
  "explorer": {
    "showHidden": true,
    "showGitIgnored": true
  },
  "terminal": {
    "shell": "/bin/zsh",
    "scrollback": 1000
  },
  "lsp": {
    "saveOnRename": true,
    "codeActionsOnSave": [
      "source.organizeImports",
      "source.fixAll"
    ],
    "servers": {
      "go": { "command": ["gopls"] },
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
  },
  "autocomplete": {
    "enabled": true,
    "autoSuggest": true,
    "debounce": 150,
    "signatureHelp": true
  }
}
```
