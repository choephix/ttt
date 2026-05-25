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
	Command []string `json:"command"`
}

type LSPSettings struct {
	Servers map[string]LSPServerConfig `json:"servers,omitempty"`
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
		Terminal:       DefaultTerminalSettings(),
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
