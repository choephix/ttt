package lsp

import "encoding/json"

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type InitializeParams struct {
	ProcessID int    `json:"processId"`
	RootURI   string `json:"rootUri"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
}

type TextDocumentClientCapabilities struct {
	Completion *CompletionClientCapabilities `json:"completion,omitempty"`
}

type CompletionClientCapabilities struct {
	CompletionItem *CompletionItemClientCapabilities `json:"completionItem,omitempty"`
}

type CompletionItemClientCapabilities struct {
	SnippetSupport bool `json:"snippetSupport"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	CompletionProvider *CompletionOptions      `json:"completionProvider,omitempty"`
	TextDocumentSync   *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type TextDocumentSyncOptions struct {
	OpenClose bool `json:"openClose,omitempty"`
	Change    int  `json:"change,omitempty"`
}

func (o *TextDocumentSyncOptions) UnmarshalJSON(data []byte) error {
	var kind int
	if err := json.Unmarshal(data, &kind); err == nil {
		o.Change = kind
		o.OpenClose = true
		return nil
	}
	type alias TextDocumentSyncOptions
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*o = TextDocumentSyncOptions(a)
	return nil
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CompletionItemKind int

const (
	CIKText          CompletionItemKind = 1
	CIKMethod        CompletionItemKind = 2
	CIKFunction      CompletionItemKind = 3
	CIKConstructor   CompletionItemKind = 4
	CIKField         CompletionItemKind = 5
	CIKVariable      CompletionItemKind = 6
	CIKClass         CompletionItemKind = 7
	CIKInterface     CompletionItemKind = 8
	CIKModule        CompletionItemKind = 9
	CIKProperty      CompletionItemKind = 10
	CIKUnit          CompletionItemKind = 11
	CIKValue         CompletionItemKind = 12
	CIKEnum          CompletionItemKind = 13
	CIKKeyword       CompletionItemKind = 14
	CIKSnippet       CompletionItemKind = 15
	CIKColor         CompletionItemKind = 16
	CIKFile          CompletionItemKind = 17
	CIKReference     CompletionItemKind = 18
	CIKFolder        CompletionItemKind = 19
	CIKEnumMember    CompletionItemKind = 20
	CIKConstant      CompletionItemKind = 21
	CIKStruct        CompletionItemKind = 22
	CIKEvent         CompletionItemKind = 23
	CIKOperator      CompletionItemKind = 24
	CIKTypeParameter CompletionItemKind = 25
)

type CompletionItem struct {
	Label         string             `json:"label"`
	Kind          CompletionItemKind `json:"kind,omitempty"`
	Detail        string             `json:"detail,omitempty"`
	InsertText    string             `json:"insertText,omitempty"`
	FilterText    string             `json:"filterText,omitempty"`
	SortText      string             `json:"sortText,omitempty"`
	TextEdit      *TextEdit          `json:"textEdit,omitempty"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}
