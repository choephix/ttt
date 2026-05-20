package config

type StyleDef struct {
	Fg   string `json:"fg,omitempty"`
	Bg   string `json:"bg,omitempty"`
	Bold bool   `json:"bold,omitempty"`
}

type BorderChars struct {
	Horizontal  string `json:"horizontal"`
	Vertical    string `json:"vertical"`
	TopLeft     string `json:"topLeft"`
	TopRight    string `json:"topRight"`
	BottomLeft  string `json:"bottomLeft"`
	BottomRight string `json:"bottomRight"`
	TopTee      string `json:"topTee"`
	BottomTee   string `json:"bottomTee"`
	LeftTee     string `json:"leftTee"`
	RightTee    string `json:"rightTee"`
}

type ThemeConfig struct {
	DefaultFg    string `json:"defaultFg,omitempty"`
	DefaultBg    string `json:"defaultBg,omitempty"`
	SuccessColor string `json:"successColor,omitempty"`
	DangerColor  string `json:"dangerColor,omitempty"`
	WarningColor string `json:"warningColor,omitempty"`
	StatusBar       StyleDef    `json:"statusBar"`
	ActiveTab       StyleDef    `json:"activeTab"`
	InactiveTab     StyleDef    `json:"inactiveTab"`
	SidebarHeader   StyleDef    `json:"sidebarHeader"`
	SidebarItem     StyleDef    `json:"sidebarItem"`
	SidebarSelected StyleDef    `json:"sidebarSelected"`
	PaletteBorder   StyleDef    `json:"paletteBorder"`
	PaletteInput    StyleDef    `json:"paletteInput"`
	PaletteItem     StyleDef    `json:"paletteItem"`
	PaletteSelected StyleDef    `json:"paletteSelected"`
	LineNumber      StyleDef    `json:"lineNumber"`
	MenuBar         StyleDef    `json:"menuBar"`
	MenuBarActive   StyleDef    `json:"menuBarActive"`
	Border          StyleDef    `json:"border"`
	Selection       StyleDef    `json:"selection"`
	SearchMatch     StyleDef    `json:"searchMatch"`
	SearchActive    StyleDef    `json:"searchActive"`
	DiffAdded       StyleDef    `json:"diffAdded"`
	DiffDeleted     StyleDef    `json:"diffDeleted"`
	DiffModified    StyleDef    `json:"diffModified"`
	ActiveLine      StyleDef    `json:"activeLine"`
	Scrollbar       StyleDef    `json:"scrollbar"`
	ScrollbarThumb  StyleDef    `json:"scrollbarThumb"`
	Borders         BorderChars `json:"borders"`
}

func DefaultTheme() ThemeConfig {
	t := ThemeConfig{
		DefaultFg: "#fafafa",
		DefaultBg: "#1f1f1f",

		MenuBar:       StyleDef{},
		MenuBarActive: StyleDef{Fg: "#ffffff", Bg: "#505050", Bold: true},
		StatusBar:     StyleDef{},

		ActiveTab:   StyleDef{Fg: "#ffffff", Bold: true},
		InactiveTab: StyleDef{Fg: "#888888"},

		SidebarHeader:   StyleDef{Fg: "#ffffff", Bold: true},
		SidebarSelected: StyleDef{Fg: "#ffffff", Bg: "#37373d"},

		PaletteBorder:   StyleDef{Fg: "#555555"},
		PaletteSelected: StyleDef{Fg: "#ffffff", Bg: "#37373d"},

		Border: StyleDef{Fg: "#555555"},

		ActiveLine:     StyleDef{Bg: "#282828"},
		LineNumber:     StyleDef{Fg: "#999999"},
		Scrollbar:      StyleDef{Fg: "#555555"},
		ScrollbarThumb: StyleDef{Fg: "#888888"},

		Borders: BorderChars{
			Horizontal:  "─",
			Vertical:    "│",
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "└",
			BottomRight: "┘",
			TopTee:      "┬",
			BottomTee:   "┴",
			LeftTee:     "├",
			RightTee:    "┤",
		},
	}
	return t
}

func (t *ThemeConfig) ResolveColors() {
	sc := t.SuccessColor
	if sc == "" {
		sc = "green"
	}
	fillFg(&t.DiffAdded, sc)

	dc := t.DangerColor
	if dc == "" {
		dc = "red"
	}
	fillFg(&t.DiffDeleted, dc)

	wc := t.WarningColor
	if wc == "" {
		wc = "yellow"
	}
	fillFg(&t.DiffModified, wc)
}

func fillFg(s *StyleDef, color string) {
	if s.Fg == "" {
		s.Fg = color
	}
}

func fillBg(s *StyleDef, color string) {
	if s.Bg == "" {
		s.Bg = color
	}
}
