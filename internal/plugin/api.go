package plugin

type EditorAPI interface {
	BufferText() string
	BufferLines() []string
	CurrentLine() string
	CursorPos() (line, col int)
	Selection() (active bool, startLine, startCol, endLine, endCol int)
	SelectionText() string
	FilePath() string
	FileName() string
	Language() string

	Insert(line, col int, text string)
	Replace(startLine, startCol, endLine, endCol int, text string)
	SetCursor(line, col int)
	SetSelection(startLine, startCol, endLine, endCol int)
	ClearSelection()
}

type FileEntry struct {
	Name  string
	IsDir bool
}

type FilesystemAPI interface {
	ReadFile(path string) (string, error)
	WriteFile(path, content string) error
	FileExists(path string) bool
	ListDir(path string) ([]FileEntry, error)
}

type SystemAPI interface {
	Exec(binary string, args []string) (stdout, stderr string, exitCode int, err error)
	Env(name string) string
}

type NetworkAPI interface {
	Get(url string, headers map[string]string) (status int, body string, respHeaders map[string]string, err error)
	Post(url string, headers map[string]string, body string) (status int, respBody string, respHeaders map[string]string, err error)
}

type PluginAsyncResult struct {
	Plugin   *Plugin
	Callback func()
}
