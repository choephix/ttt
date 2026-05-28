package config

import (
	"encoding/json"
	"os"
)

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
	Enabled          *bool                      `json:"enabled,omitempty"`
	Servers          map[string]LSPServerConfig `json:"servers,omitempty"`
	SaveOnRename     bool                       `json:"saveOnRename"`
	CodeActionsOnSave []string                  `json:"codeActionsOnSave,omitempty"`
	HoverDelay       int                        `json:"hoverDelay,omitempty"`
}

func (l LSPSettings) IsEnabled() bool {
	return l.Enabled == nil || *l.Enabled
}

func DefaultLSPSettings() LSPSettings {
	return LSPSettings{
		HoverDelay: 400,
		Servers: map[string]LSPServerConfig{
			"go": {Command: []string{"gopls"}},
			"typescript": {
				Command: []string{"typescript-language-server", "--stdio"},
				Languages: map[string]string{
					".ts":  "typescript",
					".tsx": "typescriptreact",
					".js":  "javascript",
					".jsx": "javascriptreact",
					".mjs": "javascript",
					".mts": "typescript",
					".cjs": "javascript",
					".cts": "typescript",
				},
			},
			"python": {Command: []string{"pyright-langserver", "--stdio"}},
			"c": {
				Command: []string{"clangd"},
				Languages: map[string]string{
					".c":   "c",
					".h":   "c",
					".cpp": "cpp",
					".hpp": "cpp",
					".cc":  "cpp",
					".cxx": "cpp",
				},
			},
			"vue": {
				Command: []string{"vue-language-server", "--stdio"},
				Languages: map[string]string{
					".vue": "vue",
				},
			},
			"rust": {Command: []string{"rust-analyzer"}},
			"lua":  {Command: []string{"lua-language-server"}},
			"zig":  {Command: []string{"zls"}},
		},
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
	TabSize        int              `json:"tabSize"`
	InsertSpaces   bool             `json:"insertSpaces"`
	WordWrap       bool             `json:"wordWrap"`
	LineNumbers    bool             `json:"lineNumbers"`
	SidebarVisible bool             `json:"sidebarVisible"`
	SidebarWidth   int              `json:"sidebarWidth"`
	CursorStyle    string           `json:"cursorStyle,omitempty"`
	Theme          string           `json:"theme,omitempty"`
	DebugMode      bool             `json:"debugMode,omitempty"`
	FormatOnSave   bool             `json:"formatOnSave"`
	Explorer       ExplorerSettings     `json:"explorer,omitzero"`
	Terminal       TerminalSettings     `json:"terminal,omitzero"`
	LSP            LSPSettings          `json:"lsp,omitzero"`
	Autocomplete   AutocompleteSettings `json:"autocomplete,omitzero"`
}

func DefaultSettings() Settings {
	return Settings{
		TabSize:        4,
		InsertSpaces:   true,
		WordWrap:       false,
		LineNumbers:    true,
		SidebarVisible: true,
		SidebarWidth:   30,
		Explorer:       DefaultExplorerSettings(),
		Terminal:       DefaultTerminalSettings(),
		LSP:            DefaultLSPSettings(),
		Autocomplete:   DefaultAutocompleteSettings(),
	}
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
