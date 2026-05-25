package main

import (
	"github.com/eugenioenko/ttt/internal/lsp"
	"github.com/eugenioenko/ttt/internal/ui"
)

type completionResult struct {
	items []ui.CompletionItem
}

func fileURI(path string) string {
	return "file://" + path
}

func lspToUICompletions(items []lsp.CompletionItem) []ui.CompletionItem {
	result := make([]ui.CompletionItem, 0, len(items))
	for _, item := range items {
		uiItem := ui.CompletionItem{
			Label:      item.Label,
			Detail:     item.Detail,
			InsertText: item.InsertText,
			Kind:       lspKindToUI(item.Kind),
		}
		if uiItem.InsertText == "" && item.TextEdit != nil {
			uiItem.InsertText = item.TextEdit.NewText
		}
		result = append(result, uiItem)
	}
	return result
}

func lspKindToUI(kind lsp.CompletionItemKind) ui.CompletionKind {
	switch kind {
	case lsp.CIKFunction, lsp.CIKConstructor:
		return ui.CompletionFunction
	case lsp.CIKMethod:
		return ui.CompletionMethod
	case lsp.CIKVariable:
		return ui.CompletionVariable
	case lsp.CIKConstant, lsp.CIKEnumMember:
		return ui.CompletionConstant
	case lsp.CIKClass, lsp.CIKInterface, lsp.CIKStruct, lsp.CIKEnum, lsp.CIKTypeParameter:
		return ui.CompletionType
	case lsp.CIKField, lsp.CIKProperty:
		return ui.CompletionField
	case lsp.CIKKeyword:
		return ui.CompletionKeyword
	case lsp.CIKSnippet:
		return ui.CompletionSnippet
	case lsp.CIKModule:
		return ui.CompletionModule
	default:
		return ui.CompletionVariable
	}
}
