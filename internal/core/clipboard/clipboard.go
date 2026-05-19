package clipboard

var content string

func Set(s string) {
	content = s
}

func Get() string {
	return content
}
