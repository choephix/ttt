package term

type Style int

const (
	StyleDefault  Style = iota
	StyleStatusBar
	StyleActiveTab
	StyleInactiveTab
	StyleActivityBar
	StyleActivityBarActive
	StyleSidebarHeader
	StyleSidebarItem
	StyleSidebarSelected
	StylePaletteBorder
	StylePaletteInput
	StylePaletteItem
	StylePaletteSelected
	StyleLineNumber
)

// Cell represents a single character cell on the screen.
type Cell struct {
	Ch    rune
	Style Style
}

// Screen abstracts the terminal screen.
type Screen interface {
	Size() (w, h int)
	SetCell(x, y int, c Cell)
	Show()
	Clear()
	ShowCursor(x, y int)
}

// MockScreen is a test/mock implementation of Screen.
type MockScreen struct {
	Width, Height int
	Cells         map[[2]int]Cell
}

func NewMockScreen(w, h int) *MockScreen {
	return &MockScreen{
		Width:  w,
		Height: h,
		Cells:  make(map[[2]int]Cell),
	}
}

func (m *MockScreen) Size() (int, int) {
	return m.Width, m.Height
}

func (m *MockScreen) SetCell(x, y int, c Cell) {
	m.Cells[[2]int{x, y}] = c
}

func (m *MockScreen) Show()               {}
func (m *MockScreen) Clear()              { m.Cells = make(map[[2]int]Cell) }
func (m *MockScreen) ShowCursor(x, y int) {}
