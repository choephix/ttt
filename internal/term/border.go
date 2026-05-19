package term

type BorderSet struct {
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
	TopTee      rune
	BottomTee   rune
	LeftTee     rune
	RightTee    rune
}

func DoubleBorderSet() BorderSet {
	return BorderSet{
		Horizontal:  '═',
		Vertical:    '║',
		TopLeft:     '╔',
		TopRight:    '╗',
		BottomLeft:  '╚',
		BottomRight: '╝',
		TopTee:      '╦',
		BottomTee:   '╩',
		LeftTee:     '╠',
		RightTee:    '╣',
	}
}

func SingleBorderSet() BorderSet {
	return BorderSet{
		Horizontal:  '─',
		Vertical:    '│',
		TopLeft:     '┌',
		TopRight:    '┐',
		BottomLeft:  '└',
		BottomRight: '┘',
		TopTee:      '┬',
		BottomTee:   '┴',
		LeftTee:     '├',
		RightTee:    '┤',
	}
}
