package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	if data, err := readFirst(paths, "theme.json"); err == nil {
		json.Unmarshal(data, &cfg.Theme)
	}

	return cfg
}

func configPaths() []string {
	var paths []string

	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "config"))
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "pico"))
	}

	return paths
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
