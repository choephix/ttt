package config

import (
	"fmt"
	"strings"
)

type KeyCombo struct {
	KeyName string
	Rune    rune
	Ctrl    bool
	Alt     bool
	Shift   bool
}

type KeyBinding struct {
	Key     string     `json:"key"`
	Command string     `json:"command"`
	Steps   []KeyCombo `json:"-"`
}

func (kb *KeyBinding) IsChord() bool {
	return len(kb.Steps) > 1
}

func ParseKeyBindings(bindings []KeyBinding) error {
	var firstErr error
	for i := range bindings {
		steps, err := ParseKeyString(bindings[i].Key)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("keybinding %q: %w", bindings[i].Key, err)
			}
			continue
		}
		bindings[i].Steps = steps
	}
	return firstErr
}

func ParseKeyString(s string) ([]KeyCombo, error) {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty key string")
	}
	steps := make([]KeyCombo, 0, len(parts))
	for _, part := range parts {
		combo, err := parseSingleKey(strings.ToLower(strings.TrimSpace(part)))
		if err != nil {
			return nil, err
		}
		steps = append(steps, combo)
	}
	return steps, nil
}

func parseSingleKey(s string) (KeyCombo, error) {
	combo := KeyCombo{}
	tokens := strings.Split(s, "+")

	for i, token := range tokens {
		if i == len(tokens)-1 {
			if name, ok := specialKeyNames[token]; ok {
				combo.KeyName = name
			} else if runes := []rune(token); len(runes) == 1 {
				combo.Rune = runes[0]
			} else {
				return combo, fmt.Errorf("unknown key %q", token)
			}
		} else {
			switch token {
			case "ctrl":
				combo.Ctrl = true
			case "alt":
				combo.Alt = true
			case "shift":
				combo.Shift = true
			default:
				return combo, fmt.Errorf("unknown modifier %q", token)
			}
		}
	}
	return combo, nil
}

var specialKeyNames = map[string]string{
	"escape":    "Escape",
	"esc":       "Escape",
	"enter":     "Enter",
	"return":    "Enter",
	"tab":       "Tab",
	"backspace": "Backspace",
	"delete":    "Delete",
	"insert":    "Insert",
	"up":        "Up",
	"down":      "Down",
	"left":      "Left",
	"right":     "Right",
	"home":      "Home",
	"end":       "End",
	"pgup":      "PgUp",
	"pgdn":      "PgDn",
	"pageup":    "PgUp",
	"pagedown":  "PgDn",
	"space":     "Space",
	"`":         "Backtick",
	"backtick":  "Backtick",
	"f1":        "F1",
	"f2":        "F2",
	"f3":        "F3",
	"f4":        "F4",
	"f5":        "F5",
	"f6":        "F6",
	"f7":        "F7",
	"f8":        "F8",
	"f9":        "F9",
	"f10":       "F10",
	"f11":       "F11",
	"f12":       "F12",
}

func DefaultKeybindings() []KeyBinding {
	return []KeyBinding{
		{Key: "ctrl+b", Command: "sidebar.toggle"},
		{Key: "ctrl+e", Command: "sidebar.explorer"},
		{Key: "ctrl+d", Command: "sidebar.changes"},
		{Key: "ctrl+f", Command: "search.find"},
		{Key: "ctrl+h", Command: "search.replace"},
		{Key: "ctrl+g", Command: "editor.goToLine"},
		{Key: "ctrl+p", Command: "file.quickOpen"},
		{Key: "ctrl+shift+p", Command: "command.palette"},
		{Key: "ctrl+s", Command: "file.save"},
		{Key: "ctrl+z", Command: "editor.undo"},
		{Key: "ctrl+y", Command: "editor.redo"},
		{Key: "ctrl+q", Command: "editor.quit"},
		{Key: "ctrl+0", Command: "sidebar.focus"},
		{Key: "ctrl+pgdn", Command: "tab.next"},
		{Key: "ctrl+pgup", Command: "tab.prev"},
		{Key: "ctrl+w", Command: "tab.close"},
		{Key: "ctrl+a", Command: "editor.selectAll"},
		{Key: "ctrl+c", Command: "editor.copy"},
		{Key: "ctrl+x", Command: "editor.cut"},
		{Key: "ctrl+v", Command: "editor.paste"},
		{Key: "escape", Command: "editor.focus"},
		{Key: "f3", Command: "search.findNext"},
		{Key: "shift+f3", Command: "search.findPrev"},
		{Key: "ctrl+`", Command: "panel.toggle"},
		{Key: "ctrl+shift+f", Command: "sidebar.search"},
		{Key: "ctrl+k ctrl+t", Command: "theme.switch"},
		{Key: "f10", Command: "menu.file"},
		{Key: "alt+f", Command: "menu.file"},
		{Key: "alt+e", Command: "menu.edit"},
		{Key: "alt+s", Command: "menu.selection"},
		{Key: "alt+v", Command: "menu.view"},
		{Key: "alt+h", Command: "menu.help"},
	}
}
