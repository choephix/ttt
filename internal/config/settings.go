package config

type Settings struct {
	TabSize        int  `json:"tabSize"`
	InsertSpaces   bool `json:"insertSpaces"`
	WordWrap       bool `json:"wordWrap"`
	LineNumbers    bool `json:"lineNumbers"`
	SidebarVisible bool `json:"sidebarVisible"`
	SidebarWidth   int  `json:"sidebarWidth"`
}

func DefaultSettings() Settings {
	return Settings{
		TabSize:        4,
		InsertSpaces:   true,
		WordWrap:       false,
		LineNumbers:    true,
		SidebarVisible: true,
		SidebarWidth:   30,
	}
}
