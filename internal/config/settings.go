package config

import (
	_ "embed"
	"encoding/json"
	"os"
)

//go:embed lsp_servers.json
var lspServersJSON []byte

type TerminalSettings struct {
	Shell      string `json:"shell,omitempty"`
	Scrollback int    `json:"scrollback,omitempty"`
}

func DefaultTerminalSettings() TerminalSettings {
	return TerminalSettings{
		Scrollback: 1000,
	}
}

type AutocompleteSettings struct {
	Enabled       bool `json:"enabled"`
	AutoSuggest   bool `json:"autoSuggest"`
	Debounce      int  `json:"debounce"`
	SignatureHelp bool `json:"signatureHelp"`
}

func DefaultAutocompleteSettings() AutocompleteSettings {
	return AutocompleteSettings{
		Enabled:       true,
		AutoSuggest:   true,
		Debounce:      150,
		SignatureHelp: true,
	}
}

type LSPServerConfig struct {
	Command   []string          `json:"command"`
	Languages map[string]string `json:"languages,omitempty"`
}

type LSPSettings struct {
	Enabled            *bool                      `json:"enabled,omitempty"`
	Hover              *bool                      `json:"hover,omitempty"`
	Servers            map[string]LSPServerConfig `json:"servers,omitempty"`
	SaveOnRename       bool                       `json:"saveOnRename"`
	CodeActionsOnSave  []string                   `json:"codeActionsOnSave,omitempty"`
	HoverDelay         int                        `json:"hoverDelay,omitempty"`
	NotifyAvailability *bool                      `json:"notifyAvailability,omitempty"`
}

func (l LSPSettings) ShouldNotifyAvailability() bool {
	return l.NotifyAvailability == nil || *l.NotifyAvailability
}

func (l LSPSettings) IsEnabled() bool {
	return l.Enabled == nil || *l.Enabled
}

func (l LSPSettings) IsHoverEnabled() bool {
	return l.Hover == nil || *l.Hover
}

func DefaultLSPSettings() LSPSettings {
	var servers map[string]LSPServerConfig
	json.Unmarshal(lspServersJSON, &servers)
	return LSPSettings{
		HoverDelay: 500,
		Servers:    servers,
	}
}

type EditorSettings struct {
	TabSize                int    `json:"tabSize"`
	InsertSpaces           bool   `json:"insertSpaces"`
	WordWrap               bool   `json:"wordWrap"`
	LineNumbers            bool   `json:"lineNumbers"`
	CursorStyle            string `json:"cursorStyle,omitempty"`
	FormatOnSave           bool   `json:"formatOnSave"`
	InsertFinalNewline     bool   `json:"insertFinalNewline"`
	TrimTrailingWhitespace bool   `json:"trimTrailingWhitespace"`
	DiffView               string `json:"diffView,omitempty"`
	FocusOnOpen            bool   `json:"focusOnOpen"`
	GitGutter               *bool  `json:"gitGutter,omitempty"`
	GutterStyle             string `json:"gutterStyle,omitempty"`
	BracketPairColorization bool   `json:"bracketPairColorization"`
}

// IsGitGutterEnabled returns whether git gutter indicators are enabled.
// Defaults to true when the setting is not explicitly set.
func (e EditorSettings) IsGitGutterEnabled() bool {
	return e.GitGutter == nil || *e.GitGutter
}

func DefaultEditorSettings() EditorSettings {
	return EditorSettings{
		TabSize:            4,
		InsertSpaces:       true,
		LineNumbers:        true,
		InsertFinalNewline: true,
		GutterStyle:             "compact",
		BracketPairColorization: false,
	}
}

type SearchSettings struct {
	Debounce int `json:"debounce"`
}

func DefaultSearchSettings() SearchSettings {
	return SearchSettings{
		Debounce: 350,
	}
}

type ExplorerSettings struct {
	ShowHidden     bool `json:"showHidden"`
	ShowGitIgnored bool `json:"showGitIgnored"`
}

func DefaultExplorerSettings() ExplorerSettings {
	return ExplorerSettings{
		ShowHidden:     true,
		ShowGitIgnored: true,
	}
}

type Settings struct {
	Version      int                  `json:"version"`
	Theme        string               `json:"theme,omitempty"`
	DebugMode    bool                 `json:"debugMode,omitempty"`
	Editor       EditorSettings       `json:"editor,omitzero"`
	Search       SearchSettings       `json:"search,omitzero"`
	Explorer     ExplorerSettings     `json:"explorer,omitzero"`
	Terminal     TerminalSettings     `json:"terminal,omitzero"`
	LSP          LSPSettings          `json:"lsp,omitzero"`
	Autocomplete AutocompleteSettings `json:"autocomplete,omitzero"`
}

func DefaultSettings() Settings {
	return Settings{
		Version:      1,
		Editor:       DefaultEditorSettings(),
		Search:       DefaultSearchSettings(),
		Explorer:     DefaultExplorerSettings(),
		Terminal:     DefaultTerminalSettings(),
		LSP:          DefaultLSPSettings(),
		Autocomplete: DefaultAutocompleteSettings(),
	}
}

func normalizeSettings(s *Settings) {
	switch s.Editor.GutterStyle {
	case "minimal", "compact", "extended":
	default:
		s.Editor.GutterStyle = "compact"
	}
}

func LoadSettings() Settings {
	s := DefaultSettings()
	paths := configPaths()
	if data, err := readFirst(paths, "settings.json"); err == nil {
		json.Unmarshal(data, &s)
	}
	normalizeSettings(&s)
	return s
}

// DefaultSettingsText returns a formatted reference document showing all
// available settings with their default values and descriptions. The output
// uses JSONC-style comments (// ...) so it can serve as inline documentation.
func DefaultSettingsText() string {
	return `// Default Settings Reference
// This is a read-only reference of all available settings and their defaults.
// To customize, open your settings file via "Preferences: Open Settings"
// and override only the values you want to change.
//
// Settings file location: ~/.config/ttt/settings.json
{
  // Schema version (do not change)
  "version": 1,

  // Color theme name (string, empty = default theme)
  // Use "Switch Theme" command to browse available themes.
  "theme": "",

  // Enable debug mode (bool, default: false)
  "debugMode": false,

  // ── Editor ──────────────────────────────────────────────
  "editor": {
    // Number of spaces per tab stop (int, default: 4)
    "tabSize": 4,

    // Use spaces instead of tabs for indentation (bool, default: true)
    "insertSpaces": true,

    // Wrap long lines to fit the viewport (bool, default: false)
    "wordWrap": false,

    // Show line numbers in the gutter (bool, default: true)
    "lineNumbers": true,

    // Cursor style: "block", "underline", or "bar" (string, default: "")
    "cursorStyle": "",

    // Automatically format the file on save (bool, default: false)
    "formatOnSave": false,

    // Ensure the file ends with a newline on save (bool, default: true)
    "insertFinalNewline": true,

    // Remove trailing whitespace on save (bool, default: false)
    "trimTrailingWhitespace": false,

    // Diff view layout: "inline" or "side-by-side" (string, default: "")
    "diffView": "",

    // Focus the editor when opening a file from the sidebar (bool, default: false)
    "focusOnOpen": false,

    // Show git change indicators in the gutter (bool, default: true)
    // Uses *bool — omitted means true.
    "gitGutter": true,

    // Gutter style: "minimal", "compact", or "extended" (string, default: "compact")
    "gutterStyle": "compact",

    // Colorize matching bracket pairs (bool, default: false)
    "bracketPairColorization": false
  },

  // ── Search ──────────────────────────────────────────────
  "search": {
    // Debounce delay in ms before triggering search (int, default: 350)
    "debounce": 350
  },

  // ── Explorer ────────────────────────────────────────────
  "explorer": {
    // Show hidden files (dotfiles) in the file explorer (bool, default: true)
    "showHidden": true,

    // Show files ignored by .gitignore (bool, default: true)
    "showGitIgnored": true
  },

  // ── Terminal ────────────────────────────────────────────
  "terminal": {
    // Shell command for the integrated terminal (string, default: "" = $SHELL or /bin/sh)
    "shell": "",

    // Number of scrollback lines in the terminal (int, default: 1000)
    "scrollback": 1000
  },

  // ── LSP (Language Server Protocol) ──────────────────────
  "lsp": {
    // Enable LSP support (bool, default: true)
    // Uses *bool — omitted means true.
    "enabled": true,

    // Enable hover tooltips on mouse rest (bool, default: true)
    // Uses *bool — omitted means true. Set to false to disable mouse-triggered hover.
    // The "Show Hover" command still works when disabled.
    "hover": true,

    // Auto-save files when renamed via LSP (bool, default: false)
    "saveOnRename": false,

    // Code actions to run automatically on save (string array, default: [])
    "codeActionsOnSave": [],

    // Delay in ms before showing hover information (int, default: 500)
    "hoverDelay": 500,

    // Show notification when an LSP server is available but not installed
    // (bool, default: true). Uses *bool — omitted means true.
    "notifyAvailability": true,

    // LSP server configurations per language.
    // Each entry maps a language key to a server config with "command" (string array)
    // and optional "languages" (map of file extension to language ID).
    // See ~/.config/ttt/extensions.json for additional configuration.
    "servers": {}
  },

  // ── Autocomplete ────────────────────────────────────────
  "autocomplete": {
    // Enable autocomplete suggestions (bool, default: true)
    "enabled": true,

    // Automatically show suggestions as you type (bool, default: true)
    "autoSuggest": true,

    // Debounce delay in ms before triggering autocomplete (int, default: 150)
    "debounce": 150,

    // Show function signature help on ( and , characters (bool, default: true)
    "signatureHelp": true
  }
}
`
}

func SaveSettings(s Settings) error {
	path := ConfigFilePath("settings.json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
