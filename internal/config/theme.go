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
}

type ThemeConfig struct {
	StatusBar         StyleDef `json:"statusBar"`
	ActiveTab         StyleDef `json:"activeTab"`
	InactiveTab       StyleDef `json:"inactiveTab"`
	ActivityBar       StyleDef `json:"activityBar"`
	ActivityBarActive StyleDef `json:"activityBarActive"`
	SidebarHeader     StyleDef `json:"sidebarHeader"`
	SidebarItem       StyleDef `json:"sidebarItem"`
	SidebarSelected   StyleDef `json:"sidebarSelected"`
	PaletteBorder     StyleDef `json:"paletteBorder"`
	PaletteInput      StyleDef `json:"paletteInput"`
	PaletteItem       StyleDef `json:"paletteItem"`
	PaletteSelected   StyleDef `json:"paletteSelected"`
	LineNumber        StyleDef `json:"lineNumber"`
	ResizeHandle      StyleDef `json:"resizeHandle"`
	MenuBar           StyleDef `json:"menuBar"`
	MenuBarActive     StyleDef `json:"menuBarActive"`
	Borders           BorderChars `json:"borders"`
}

func DefaultTheme() ThemeConfig {
	return ThemeConfig{
		StatusBar:         StyleDef{Fg: "white", Bg: "darkcyan"},
		ActiveTab:         StyleDef{Fg: "white", Bg: "darkblue", Bold: true},
		InactiveTab:       StyleDef{Fg: "silver", Bg: "darkgray"},
		ActivityBar:       StyleDef{Fg: "silver", Bg: "darkslategray"},
		ActivityBarActive: StyleDef{Fg: "white", Bg: "darkslategray", Bold: true},
		SidebarHeader:     StyleDef{Fg: "white", Bold: true},
		SidebarItem:       StyleDef{Fg: "silver"},
		SidebarSelected:   StyleDef{Fg: "white", Bg: "darkblue"},
		PaletteBorder:     StyleDef{Fg: "darkcyan"},
		PaletteInput:      StyleDef{Fg: "white"},
		PaletteItem:       StyleDef{Fg: "silver"},
		PaletteSelected:   StyleDef{Fg: "white", Bg: "darkblue"},
		LineNumber:        StyleDef{Fg: "darkgray"},
		ResizeHandle:      StyleDef{Fg: "gray", Bg: "darkslategray"},
		MenuBar:           StyleDef{Fg: "black", Bg: "silver"},
		MenuBarActive:     StyleDef{Fg: "white", Bg: "darkcyan", Bold: true},
		Borders: BorderChars{
			Horizontal:  "═",
			Vertical:    "║",
			TopLeft:     "╔",
			TopRight:    "╗",
			BottomLeft:  "╚",
			BottomRight: "╝",
		},
	}
}
