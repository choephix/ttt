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
	Terminal       TerminalSettings `json:"terminal,omitzero"`
	LSP            LSPSettings      `json:"lsp,omitzero"`
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
