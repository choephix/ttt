package main

import (
	"ttt/internal/config"
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

func buildStyleMap(theme config.ThemeConfig) term.StyleMap {
	m := term.DefaultStyleMap()

	base := tcell.StyleDefault
	if theme.Default.Fg != "" {
		base = base.Foreground(tcell.GetColor(theme.Default.Fg))
	}
	if theme.Default.Bg != "" {
		base = base.Background(tcell.GetColor(theme.Default.Bg))
	}
	for i := range m {
		m[i] = base
	}
	m[term.StyleSelection] = base.Reverse(true)

	applyStyleDef(&m, term.StyleStatusBar, theme.StatusBar)
	applyStyleDef(&m, term.StyleActiveTab, theme.Tabs.Active)
	applyStyleDef(&m, term.StyleInactiveTab, theme.Tabs.Inactive)
	applyStyleDef(&m, term.StyleSidebarSelected, theme.Sidebar.Selected)
	applyStyleDef(&m, term.StylePaletteItem, theme.Dialog.Item)
	applyStyleDef(&m, term.StylePaletteSelected, theme.Dialog.Selected)
	applyStyleDef(&m, term.StyleLineNumber, theme.Editor.LineNumber)
	applyStyleDef(&m, term.StyleMenuBar, theme.Menu.Item)
	applyStyleDef(&m, term.StyleMenuBarActive, theme.Menu.Active)
	applyStyleDef(&m, term.StyleBorder, theme.Border)
	applyStyleDef(&m, term.StyleSelection, theme.Editor.Selection)
	applyStyleDef(&m, term.StyleSearchMatch, theme.Editor.SearchMatch)
	applyStyleDef(&m, term.StyleSearchActive, theme.Editor.SearchActive)
	applyStyleDef(&m, term.StyleDiffAdded, theme.Diff.Added)
	applyStyleDef(&m, term.StyleDiffDeleted, theme.Diff.Deleted)
	applyStyleDef(&m, term.StyleDiffModified, theme.Diff.Modified)
	applyStyleDef(&m, term.StyleActiveLine, theme.Editor.ActiveLine)
	applyStyleDef(&m, term.StyleScrollbar, config.StyleDef{Bg: theme.Scrollbar.Bg})
	applyStyleDef(&m, term.StyleScrollbarThumb, config.StyleDef{Fg: theme.Scrollbar.Fg})
	applyStyleDef(&m, term.StyleSyntaxComment, theme.Syntax.Comment)
	applyStyleDef(&m, term.StyleSyntaxString, theme.Syntax.String)
	applyStyleDef(&m, term.StyleSyntaxKeyword, theme.Syntax.Keyword)
	applyStyleDef(&m, term.StyleSyntaxNumber, theme.Syntax.Number)
	applyStyleDef(&m, term.StyleSyntaxOperator, theme.Syntax.Operator)
	applyStyleDef(&m, term.StyleSyntaxFunction, theme.Syntax.Function)
	applyStyleDef(&m, term.StyleSyntaxType, theme.Syntax.Type)
	applyStyleDef(&m, term.StyleSyntaxBuiltin, theme.Syntax.Builtin)
	applyStyleDef(&m, term.StyleSyntaxVariable, theme.Syntax.Variable)
	applyStyleDef(&m, term.StyleSyntaxPunctuation, theme.Syntax.Punctuation)
	applyStyleDef(&m, term.StyleSyntaxTag, theme.Syntax.Tag)
	applyStyleDef(&m, term.StyleSyntaxAttribute, theme.Syntax.Attribute)
	applyStyleDef(&m, term.StyleMuted, theme.Dialog.Muted)
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
