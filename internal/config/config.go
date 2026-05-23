package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type AppConfig struct {
	Keybindings []KeyBinding
	Settings    Settings
	Theme       ThemeConfig
}

func Load() AppConfig {
	cfg := AppConfig{
		Keybindings: DefaultKeybindings(),
		Settings:    DefaultSettings(),
		Theme:       DefaultTheme(),
	}

	paths := configPaths()

	if data, err := readFirst(paths, "keybindings.json"); err == nil {
		var kb []KeyBinding
		if err := json.Unmarshal(data, &kb); err == nil {
			cfg.Keybindings = kb
		}
	}

	if data, err := readFirst(paths, "settings.json"); err == nil {
		json.Unmarshal(data, &cfg.Settings)
	}

	if cfg.Settings.Theme != "" {
		themeFile := "theme." + cfg.Settings.Theme + ".json"
		if data, err := readFirst(paths, themeFile); err == nil {
			json.Unmarshal(data, &cfg.Theme)
		}
	}

	cfg.Theme.ResolveColors()

	return cfg
}

func configPaths() []string {
	var paths []string

	paths = append(paths, ".config")

	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), ".config"))
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "ttt"))
	}

	return paths
}

func LoadThemeFromFile(path string) (ThemeConfig, error) {
	theme := DefaultTheme()
	data, err := os.ReadFile(path)
	if err != nil {
		return theme, err
	}
	if err := json.Unmarshal(data, &theme); err != nil {
		return theme, err
	}
	theme.ResolveColors()
	return theme, nil
}

func ListThemeFiles() []string {
	var files []string
	for _, dir := range configPaths() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "theme.") && strings.HasSuffix(name, ".json") {
				files = append(files, filepath.Join(dir, name))
			}
		}
	}
	return files
}

func ConfigFilePath(filename string) string {
	for _, dir := range configPaths() {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return filepath.Join(".config", filename)
}

func readFirst(dirs []string, filename string) ([]byte, error) {
	for _, dir := range dirs {
		data, err := os.ReadFile(filepath.Join(dir, filename))
		if err == nil {
			return data, nil
		}
	}
	return nil, os.ErrNotExist
}
