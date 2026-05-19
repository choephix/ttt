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
	StatusBar         StyleDef    `json:"statusBar"`
	ActiveTab         StyleDef    `json:"activeTab"`
	InactiveTab       StyleDef    `json:"inactiveTab"`
	ActivityBar       StyleDef    `json:"activityBar"`
	ActivityBarActive StyleDef    `json:"activityBarActive"`
	SidebarHeader     StyleDef    `json:"sidebarHeader"`
	SidebarItem       StyleDef    `json:"sidebarItem"`
	SidebarSelected   StyleDef    `json:"sidebarSelected"`
	PaletteBorder     StyleDef    `json:"paletteBorder"`
	PaletteInput      StyleDef    `json:"paletteInput"`
	PaletteItem       StyleDef    `json:"paletteItem"`
	PaletteSelected   StyleDef    `json:"paletteSelected"`
	LineNumber        StyleDef    `json:"lineNumber"`
	ResizeHandle      StyleDef    `json:"resizeHandle"`
	MenuBar           StyleDef    `json:"menuBar"`
	MenuBarActive     StyleDef    `json:"menuBarActive"`
	Border            StyleDef    `json:"border"`
	Borders           BorderChars `json:"borders"`
}

func DefaultTheme() ThemeConfig {
	return ThemeConfig{
		// Frame: menu bar and status bar match as bookends
		MenuBar:       StyleDef{Fg: "black", Bg: "silver"},
		MenuBarActive: StyleDef{Fg: "white", Bg: "darkcyan", Bold: true},
		StatusBar:     StyleDef{Fg: "black", Bg: "silver"},

		// Tabs: accent for active, default bg for inactive
		ActiveTab:   StyleDef{Fg: "darkcyan", Bold: true},
		InactiveTab: StyleDef{Fg: "gray"},

		// Activity bar (kept for theme compatibility)
		ActivityBar:       StyleDef{Fg: "gray"},
		ActivityBarActive: StyleDef{Fg: "darkcyan", Bold: true},

		// Sidebar: clean with accent selection
		SidebarHeader:   StyleDef{Fg: "darkcyan", Bold: true},
		SidebarItem:     StyleDef{Fg: "silver"},
		SidebarSelected: StyleDef{Fg: "white", Bg: "darkcyan"},

		// Command palette: accent borders, accent selection
		PaletteBorder:   StyleDef{Fg: "darkcyan"},
		PaletteInput:    StyleDef{Fg: "white"},
		PaletteItem:     StyleDef{Fg: "silver"},
		PaletteSelected: StyleDef{Fg: "white", Bg: "darkcyan"},

		// Editor chrome
		LineNumber: StyleDef{Fg: "gray"},

		// Borders and separators: accent color
		ResizeHandle: StyleDef{Fg: "darkcyan"},
		Border:       StyleDef{Fg: "darkcyan"},

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
}
