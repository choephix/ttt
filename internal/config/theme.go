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

type TabStyles struct {
	Active   StyleDef `json:"active"`
	Inactive StyleDef `json:"inactive"`
}

type SidebarStyles struct {
	Header   StyleDef `json:"header"`
	Item     StyleDef `json:"item"`
	Selected StyleDef `json:"selected"`
}

type DialogStyles struct {
	Input    StyleDef `json:"input"`
	Item     StyleDef `json:"item"`
	Selected StyleDef `json:"selected"`
	Muted    StyleDef `json:"muted"`
}

type MenuStyles struct {
	Item   StyleDef `json:"item"`
	Active StyleDef `json:"active"`
}

type EditorStyles struct {
	LineNumber   StyleDef `json:"lineNumber"`
	ActiveLine   StyleDef `json:"activeLine"`
	Selection    StyleDef `json:"selection"`
	SearchMatch  StyleDef `json:"searchMatch"`
	SearchActive StyleDef `json:"searchActive"`
}

type DiffStyles struct {
	Added    StyleDef `json:"added"`
	Deleted  StyleDef `json:"deleted"`
	Modified StyleDef `json:"modified"`
}

type SyntaxStyles struct {
	Comment     StyleDef `json:"comment"`
	String      StyleDef `json:"string"`
	Keyword     StyleDef `json:"keyword"`
	Number      StyleDef `json:"number"`
	Operator    StyleDef `json:"operator"`
	Function    StyleDef `json:"function"`
	Type        StyleDef `json:"type"`
	Builtin     StyleDef `json:"builtin"`
	Variable    StyleDef `json:"variable"`
	Punctuation StyleDef `json:"punctuation"`
	Tag         StyleDef `json:"tag"`
	Attribute   StyleDef `json:"attribute"`
}

type ThemeConfig struct {
	Default      StyleDef `json:"default"`
	Success string `json:"success,omitempty"`
	Danger  string `json:"danger,omitempty"`
	Warning string `json:"warning,omitempty"`
	StatusBar       StyleDef    `json:"statusBar"`
	Tabs            TabStyles   `json:"tabs"`
	Sidebar         SidebarStyles `json:"sidebar"`
	Dialog          DialogStyles  `json:"dialog"`
	Editor          EditorStyles `json:"editor"`
	Menu            MenuStyles  `json:"menu"`
	Border          StyleDef    `json:"border"`
	Diff            DiffStyles  `json:"diff"`
	Scrollbar       StyleDef    `json:"scrollbar"`
	Syntax          SyntaxStyles `json:"syntax"`
	Borders         BorderChars `json:"borders"`
}

func DefaultTheme() ThemeConfig {
	t := ThemeConfig{
		Default: StyleDef{Fg: "#fafafa", Bg: "#1f1f1f"},

		Menu: MenuStyles{
			Active: StyleDef{Fg: "#ffffff", Bg: "#505050", Bold: true},
		},
		StatusBar:     StyleDef{},

		Tabs: TabStyles{
			Active:   StyleDef{Fg: "#ffffff", Bold: true},
			Inactive: StyleDef{Fg: "#888888"},
		},

		Sidebar: SidebarStyles{
			Header:   StyleDef{Fg: "#ffffff", Bold: true},
			Selected: StyleDef{Fg: "#ffffff", Bg: "#37373d"},
		},

		Dialog: DialogStyles{
			Selected: StyleDef{Fg: "#ffffff", Bg: "#37373d"},
			Muted:    StyleDef{Fg: "#888888"},
		},

		Border: StyleDef{Fg: "#555555"},

		Editor: EditorStyles{
			ActiveLine:   StyleDef{Bg: "#282828"},
			Selection:    StyleDef{Bg: "#282828"},
			LineNumber:   StyleDef{Fg: "#999999"},
			SearchMatch:  StyleDef{Bg: "#623800"},
			SearchActive: StyleDef{Bg: "#9e6a03"},
		},
		Scrollbar: StyleDef{Fg: "#888888", Bg: "#555555"},

		Syntax: SyntaxStyles{
			Comment:     StyleDef{Fg: "#6a9955"},
			String:      StyleDef{Fg: "#ce9178"},
			Keyword:     StyleDef{Fg: "#569cd6"},
			Number:      StyleDef{Fg: "#b5cea8"},
			Operator:    StyleDef{Fg: "#d4d4d4"},
			Function:    StyleDef{Fg: "#dcdcaa"},
			Type:        StyleDef{Fg: "#4ec9b0"},
			Builtin:     StyleDef{Fg: "#4ec9b0"},
			Variable:    StyleDef{Fg: "#9cdcfe"},
			Punctuation: StyleDef{Fg: "#d4d4d4"},
			Tag:         StyleDef{Fg: "#569cd6"},
			Attribute:   StyleDef{Fg: "#9cdcfe"},
		},

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
	sc := t.Success
	if sc == "" {
		sc = "#1e2e1e"
	}
	fillBg(&t.Diff.Added, sc)

	dc := t.Danger
	if dc == "" {
		dc = "#2e1e1e"
	}
	fillBg(&t.Diff.Deleted, dc)

	wc := t.Warning
	if wc == "" {
		wc = "#2e2e1e"
	}
	fillBg(&t.Diff.Modified, wc)
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
