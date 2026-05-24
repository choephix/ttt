package main

import (
	"github.com/eugenioenko/ttt/internal/config"

	"github.com/gdamore/tcell/v2"
)

var canonicalKeyToTcell = map[string]tcell.Key{
	"Escape":    tcell.KeyEscape,
	"Enter":     tcell.KeyEnter,
	"Tab":       tcell.KeyTab,
	"Backspace": tcell.KeyBackspace,
	"Delete":    tcell.KeyDelete,
	"Insert":    tcell.KeyInsert,
	"Up":        tcell.KeyUp,
	"Down":      tcell.KeyDown,
	"Left":      tcell.KeyLeft,
	"Right":     tcell.KeyRight,
	"Home":      tcell.KeyHome,
	"End":       tcell.KeyEnd,
	"PgUp":      tcell.KeyPgUp,
	"PgDn":      tcell.KeyPgDn,
	"Space":     tcell.KeyRune,
	"F1":        tcell.KeyF1,
	"F2":        tcell.KeyF2,
	"F3":        tcell.KeyF3,
	"F4":        tcell.KeyF4,
	"F5":        tcell.KeyF5,
	"F6":        tcell.KeyF6,
	"F7":        tcell.KeyF7,
	"F8":        tcell.KeyF8,
	"F9":        tcell.KeyF9,
	"F10":       tcell.KeyF10,
	"F11":       tcell.KeyF11,
	"F12":       tcell.KeyF12,
}

func comboToTcell(combo config.KeyCombo) (tcell.Key, tcell.ModMask, rune) {
	var mod tcell.ModMask
	if combo.Ctrl {
		mod |= tcell.ModCtrl
	}
	if combo.Alt {
		mod |= tcell.ModAlt
	}
	if combo.Shift {
		mod |= tcell.ModShift
	}

	if combo.KeyName != "" {
		if combo.KeyName == "Space" {
			return tcell.KeyRune, mod, ' '
		}
		if combo.KeyName == "Backtick" {
			if combo.Ctrl {
				return tcell.KeyCtrlSpace, mod, 0
			}
			return tcell.KeyRune, mod, '`'
		}
		key := canonicalKeyToTcell[combo.KeyName]
		return key, mod, 0
	}

	if combo.Ctrl && combo.Rune >= 'a' && combo.Rune <= 'z' {
		key := tcell.KeyCtrlA + tcell.Key(combo.Rune-'a')
		return key, mod, 0
	}

	return tcell.KeyRune, mod, combo.Rune
}
