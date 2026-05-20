package main

import (
	"ttt/internal/config"
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

func buildStyleMap(theme config.ThemeConfig) term.StyleMap {
	m := term.DefaultStyleMap()

	base := tcell.StyleDefault
	if theme.DefaultFg != "" {
		base = base.Foreground(tcell.GetColor(theme.DefaultFg))
	}
	if theme.DefaultBg != "" {
		base = base.Background(tcell.GetColor(theme.DefaultBg))
	}
	for i := range m {
		m[i] = base
	}
	m[term.StyleSelection] = base.Reverse(true)

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
	applyStyleDef(&m, term.StyleDiffAdded, theme.DiffAdded)
	applyStyleDef(&m, term.StyleDiffDeleted, theme.DiffDeleted)
	applyStyleDef(&m, term.StyleDiffModified, theme.DiffModified)
	applyStyleDef(&m, term.StyleScrollbar, theme.Scrollbar)
	applyStyleDef(&m, term.StyleScrollbarThumb, theme.ScrollbarThumb)
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
	s := m[idx]
	if def.Fg != "" {
		s = s.Foreground(tcell.GetColor(def.Fg))
	}
	if def.Bg != "" {
		s = s.Background(tcell.GetColor(def.Bg))
	}
	if def.Bold {
		s = s.Bold(true)
	}
	m[idx] = s
}
