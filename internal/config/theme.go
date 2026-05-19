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
	AccentColor     string      `json:"accentColor,omitempty"`
	SuccessColor    string      `json:"successColor,omitempty"`
	DangerColor     string      `json:"dangerColor,omitempty"`
	WarningColor    string      `json:"warningColor,omitempty"`
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
	Borders         BorderChars `json:"borders"`
}

func DefaultTheme() ThemeConfig {
	t := ThemeConfig{
		AccentColor: "darkcyan",

		MenuBar:   StyleDef{},
		StatusBar: StyleDef{},

		MenuBarActive: StyleDef{Bold: true},
		ActiveTab:     StyleDef{Bold: true},
		InactiveTab:   StyleDef{Fg: "gray"},

		SidebarHeader:   StyleDef{Bold: true},
		SidebarSelected: StyleDef{},

		PaletteSelected: StyleDef{},

		LineNumber: StyleDef{Fg: "gray"},

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
	ac := t.AccentColor
	if ac == "" {
		ac = "darkcyan"
	}
	fillBg(&t.MenuBarActive, ac)
	fillFg(&t.ActiveTab, ac)
	fillFg(&t.SidebarHeader, ac)
	fillBg(&t.SidebarSelected, ac)
	fillFg(&t.PaletteBorder, ac)
	fillBg(&t.PaletteSelected, ac)
	fillFg(&t.Border, ac)

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
