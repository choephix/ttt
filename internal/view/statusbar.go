package view

type StatusBar struct {
	FileName string
	Line     int
	Col      int
	Dirty    bool
	Message  string
	Branch   string
	Blame    string
	Language string
	TabSize  int
}
