---
title: Settings
description: Complete settings reference for TTT.
sidebar:
  order: 1
---

Settings are stored in `~/.config/ttt/settings.json`. A complete example is available at [`config/settings.json`](https://github.com/eugenioenko/ttt/blob/main/config/settings.json) in the repository.

You can open your settings file directly from the command palette (**Ctrl+P**) with **Preferences: Open Settings**.

## Top-level

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `version` | int | `1` | Settings file format version |
| `theme` | string | `""` | Theme name (e.g. `"default-dark"`) |
| `debugMode` | bool | `false` | Enable debug logging |

## Editor

All editor settings are nested under the `editor` key.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `editor.tabSize` | int | `4` | Number of spaces per indentation level |
| `editor.insertSpaces` | bool | `true` | Use spaces instead of tabs for indentation |
| `editor.wordWrap` | bool | `false` | Wrap long lines at the editor width |
| `editor.lineNumbers` | bool | `true` | Show line numbers in the gutter |
| `editor.cursorStyle` | string | `""` | Cursor style: `"block"`, `"underline"`, or `"bar"` |
| `editor.formatOnSave` | bool | `false` | Auto-format the document via LSP on save |
| `editor.insertFinalNewline` | bool | `true` | Ensure files end with a newline on load and save |
| `editor.trimTrailingWhitespace` | bool | `false` | Remove trailing whitespace from lines on save |
| `editor.focusOnOpen` | bool | `false` | Focus the editor when opening a file |
| `editor.syntaxHighlight` | bool | `true` | Enable syntax highlighting |
| `editor.gitGutter` | bool | `true` | Show git change indicators in the gutter |
| `editor.gutterStyle` | string | `"compact"` | Gutter layout: `"minimal"`, `"compact"`, or `"extended"` |
| `editor.borderStyle` | string | `"default"` | Border style preset: `"default"`, `"rounded"`, `"sharp"`, `"double"`, `"bold"`, `"ascii"`, `"none"`. Use `"default"` or `"theme"` to defer to the active theme. |
| `editor.bracketPairColorization` | bool | `false` | Colorize matching bracket pairs by nesting depth |

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
| `lsp.enabled` | bool | `true` | Enable LSP support |
| `lsp.hover` | bool | `true` | Show hover information from the language server |
| `lsp.hoverDelay` | int | `500` | Milliseconds to wait before showing hover information |
| `lsp.saveOnRename` | bool | `false` | Auto-save files affected by a rename operation |
| `lsp.codeActionsOnSave` | string[] | `[]` | Code actions to run before save (e.g. `"source.organizeImports"`) |
| `lsp.notifyAvailability` | bool | `true` | Show a notification when a language server binary is not installed |
| `lsp.servers` | object | `{}` | Map of server key to `{ "command": [...], "languages": {...} }`. Configured automatically by LSP plugins (e.g. `lsp-go`, `lsp-typescript`). The optional `languages` field maps file extensions to language IDs for servers handling multiple file types. |

## Search

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `search.debounce` | int | `350` | Milliseconds to debounce global search input |

## Plugins

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `plugins.enabled` | bool | `true` | Enable the plugin system. When `false`, no plugins are loaded and the Plugins sidebar tab is hidden. |

## Autocomplete

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `autocomplete.enabled` | bool | `true` | Enable LSP-powered autocompletion |
| `autocomplete.autoSuggest` | bool | `true` | Show completions automatically as you type |
| `autocomplete.debounce` | int | `150` | Milliseconds to wait after typing before requesting completions |
| `autocomplete.signatureHelp` | bool | `true` | Show function signature help on `(` and `,` |

## Formatters

External code formatters configured per file extension. Each formatter receives the buffer content via stdin and must write the formatted output to stdout. Use `{file}` as a placeholder for the file path (needed by formatters like prettier for filetype detection).

When `editor.formatOnSave` is `true`, external formatters take priority over LSP formatting. If no external formatter is configured for the file type, LSP formatting is used as a fallback.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `formatters.<ext>` | string | — | Formatter command for files with this extension. `<ext>` is the extension without the dot (e.g. `go`, `js`, `py`). |

**Example:**

```json
{
  "formatters": {
    "go": "gofmt",
    "lua": "stylua -",
    "js": "prettier --stdin-filepath {file}",
    "py": "black -"
  }
}
```

**Keybinding:** `Ctrl+K F` runs the external formatter. `Ctrl+L F` runs the LSP formatter.

## Full Example

```json
{
  "version": 1,
  "theme": "default-dark",
  "debugMode": false,
  "editor": {
    "tabSize": 4,
    "insertSpaces": true,
    "wordWrap": false,
    "lineNumbers": true,
    "cursorStyle": "",
    "formatOnSave": false,
    "insertFinalNewline": true,
    "trimTrailingWhitespace": false,
    "focusOnOpen": false,
    "syntaxHighlight": true,
    "gitGutter": true,
    "gutterStyle": "compact",
    "borderStyle": "default",
    "bracketPairColorization": false
  },
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
    "enabled": true,
    "hover": true,
    "hoverDelay": 500,
    "saveOnRename": false,
    "notifyAvailability": true,
    "codeActionsOnSave": [
      "source.organizeImports",
      "source.fixAll"
    ]
  },
  "autocomplete": {
    "enabled": true,
    "autoSuggest": true,
    "debounce": 150,
    "signatureHelp": true
  },
  "plugins": {
    "enabled": true
  },
  "formatters": {
    "go": "gofmt",
    "lua": "stylua -",
    "js": "prettier --stdin-filepath {file}",
    "py": "black -"
  }
}
```
