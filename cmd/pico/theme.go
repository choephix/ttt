package main

import (
	"macro/internal/config"
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

func buildStyleMap(theme config.ThemeConfig) term.StyleMap {
	m := term.DefaultStyleMap()
	applyStyleDef(&m, term.StyleStatusBar, theme.StatusBar)
	applyStyleDef(&m, term.StyleActiveTab, theme.ActiveTab)
	applyStyleDef(&m, term.StyleInactiveTab, theme.InactiveTab)
	applyStyleDef(&m, term.StyleSidebarHeader, theme.SidebarHeader)
	applyStyleDef(&m, term.StyleSidebarItem, theme.SidebarItem)
	applyStyleDef(&m, term.StyleSidebarSelected, theme.SidebarSelected)
	applyStyleDef(&m, term.StylePaletteBorder, theme.PaletteBorder)
	applyStyleDef(&m, term.StylePaletteInput, theme.PaletteInput)
	applyStyleDef(&m, term.StylePaletteItem, theme.PaletteItem)
	applyStyleDef(&m, term.StylePaletteSelected, theme.PaletteSelected)
	applyStyleDef(&m, term.StyleLineNumber, theme.LineNumber)
	applyStyleDef(&m, term.StyleMenuBar, theme.MenuBar)
	applyStyleDef(&m, term.StyleMenuBarActive, theme.MenuBarActive)
	applyStyleDef(&m, term.StyleBorder, theme.Border)
	applyStyleDef(&m, term.StyleSelection, theme.Selection)
	applyStyleDef(&m, term.StyleSearchMatch, theme.SearchMatch)
	applyStyleDef(&m, term.StyleSearchActive, theme.SearchActive)
	return m
}

func firstRune(s string, fallback rune) rune {
	for _, r := range s {
		return r
	}
	return fallback
}

func buildBorderSet(bc config.BorderChars) term.BorderSet {
	d := term.SingleBorderSet()
	return term.BorderSet{
		Horizontal:  firstRune(bc.Horizontal, d.Horizontal),
		Vertical:    firstRune(bc.Vertical, d.Vertical),
		TopLeft:     firstRune(bc.TopLeft, d.TopLeft),
		TopRight:    firstRune(bc.TopRight, d.TopRight),
		BottomLeft:  firstRune(bc.BottomLeft, d.BottomLeft),
		BottomRight: firstRune(bc.BottomRight, d.BottomRight),
		TopTee:      firstRune(bc.TopTee, d.TopTee),
		BottomTee:   firstRune(bc.BottomTee, d.BottomTee),
		LeftTee:     firstRune(bc.LeftTee, d.LeftTee),
		RightTee:    firstRune(bc.RightTee, d.RightTee),
	}
}

func applyStyleDef(m *term.StyleMap, idx term.Style, def config.StyleDef) {
	if def.Fg == "" && def.Bg == "" && !def.Bold {
		return
	}
	base := tcell.StyleDefault
	if def.Fg != "" {
		base = base.Foreground(tcell.GetColor(def.Fg))
	}
	if def.Bg != "" {
		base = base.Background(tcell.GetColor(def.Bg))
	}
	if def.Bold {
		base = base.Bold(true)
	}
	m[idx] = base
}
