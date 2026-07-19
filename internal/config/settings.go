package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
)

// Validated by normalizeSettings and used to populate the settings UI pickers.
var (
	GutterStyles = []string{"minimal", "compact", "extended"}
	BorderStyles = []string{"default", "theme", "rounded", "sharp", "double", "bold", "ascii", "none"}
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
	return LSPSettings{
		HoverDelay: 500,
		Servers:    make(map[string]LSPServerConfig),
	}
}

type EditorSettings struct {
	TabSize                 int    `json:"tabSize"`
	InsertSpaces            bool   `json:"insertSpaces"`
	WordWrap                bool   `json:"wordWrap"`
	LineNumbers             bool   `json:"lineNumbers"`
	CursorStyle             string `json:"cursorStyle,omitempty"`
	FormatOnSave            bool   `json:"formatOnSave"`
	InsertFinalNewline      bool   `json:"insertFinalNewline"`
	TrimTrailingWhitespace  bool   `json:"trimTrailingWhitespace"`
	FocusOnOpen             bool   `json:"focusOnOpen"`
	SyntaxHighlight         *bool  `json:"syntaxHighlight,omitempty"`
	GitGutter               *bool  `json:"gitGutter,omitempty"`
	AutoDedent              *bool  `json:"autoDedent,omitempty"`
	GutterStyle             string `json:"gutterStyle,omitempty"`
	BorderStyle             string `json:"borderStyle,omitempty"`
	BracketPairColorization bool   `json:"bracketPairColorization"`
}

func (e EditorSettings) IsSyntaxHighlightEnabled() bool {
	return e.SyntaxHighlight == nil || *e.SyntaxHighlight
}

func (e EditorSettings) IsGitGutterEnabled() bool {
	return e.GitGutter == nil || *e.GitGutter
}

func (e EditorSettings) IsAutoDedentEnabled() bool {
	return e.AutoDedent == nil || *e.AutoDedent
}

func DefaultEditorSettings() EditorSettings {
	return EditorSettings{
		TabSize:                 4,
		InsertSpaces:            true,
		LineNumbers:             true,
		InsertFinalNewline:      true,
		GutterStyle:             "compact",
		BorderStyle:             "default",
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

type PluginSettings struct {
	Enabled *bool `json:"enabled,omitempty"`
}

func (p PluginSettings) IsEnabled() bool {
	return p.Enabled == nil || *p.Enabled
}

type MarkdownSettings struct {
	WrapWidth int `json:"wrapWidth,omitempty"`
}

func DefaultMarkdownSettings() MarkdownSettings {
	return MarkdownSettings{
		WrapWidth: 80,
	}
}

type Settings struct {
	Version   int    `json:"version"`
	Theme     string `json:"theme,omitempty"`
	DebugMode bool   `json:"debugMode,omitempty"`
	// These sections must NOT use omitzero: their defaults are non-zero, so an
	// all-false/all-zero section would be omitted on save and silently revert to
	// the defaults on the next load.
	Editor       EditorSettings       `json:"editor"`
	Search       SearchSettings       `json:"search"`
	Explorer     ExplorerSettings     `json:"explorer"`
	Terminal     TerminalSettings     `json:"terminal"`
	LSP          LSPSettings          `json:"lsp"`
	Autocomplete AutocompleteSettings `json:"autocomplete"`
	Markdown     MarkdownSettings     `json:"markdown"`
	// Plugins is safe: its only field is a tri-state *bool where nil means the
	// default, so the zero value and "unset" mean the same thing.
	Plugins    PluginSettings    `json:"plugins,omitzero"`
	Formatters map[string]string `json:"formatters,omitempty"`
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
		Markdown:     DefaultMarkdownSettings(),
	}
}

func (s Settings) FormatterForExt(ext string) string {
	if s.Formatters == nil {
		return ""
	}
	return s.Formatters[ext]
}

func normalizeSettings(s *Settings) {
	if !slices.Contains(GutterStyles, s.Editor.GutterStyle) {
		s.Editor.GutterStyle = "compact"
	}
	if !slices.Contains(BorderStyles, s.Editor.BorderStyle) {
		s.Editor.BorderStyle = "default"
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

func SaveSettings(s Settings) error {
	path := ConfigFilePath("settings.json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data)
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
