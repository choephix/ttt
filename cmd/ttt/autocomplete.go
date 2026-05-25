package main

import "github.com/eugenioenko/ttt/internal/ui"

func mockCompletions() []ui.CompletionItem {
	return []ui.CompletionItem{
		{Label: "fmt.Println", Kind: ui.CompletionFunction},
		{Label: "fmt.Printf", Kind: ui.CompletionFunction},
		{Label: "fmt.Sprintf", Kind: ui.CompletionFunction},
		{Label: "fmt.Errorf", Kind: ui.CompletionFunction},
		{Label: "append", Kind: ui.CompletionFunction},
		{Label: "len", Kind: ui.CompletionFunction},
		{Label: "make", Kind: ui.CompletionFunction},
		{Label: "string", Kind: ui.CompletionType},
		{Label: "int", Kind: ui.CompletionType},
		{Label: "bool", Kind: ui.CompletionType},
		{Label: "error", Kind: ui.CompletionType},
		{Label: "context", Kind: ui.CompletionModule},
		{Label: "MaxRetries", Kind: ui.CompletionConstant},
		{Label: "err", Kind: ui.CompletionVariable},
		{Label: "ctx", Kind: ui.CompletionVariable},
		{Label: "buf", Kind: ui.CompletionVariable},
		{Label: "Close", Kind: ui.CompletionMethod},
		{Label: "Read", Kind: ui.CompletionMethod},
		{Label: "Write", Kind: ui.CompletionMethod},
		{Label: "Name", Kind: ui.CompletionField},
	}
}