package main

import (
	"encoding/json"
	"strings"

	"github.com/eugenioenko/ttt/internal/lsp"
	"github.com/eugenioenko/ttt/internal/ui"
)

type completionResult struct {
	items []ui.CompletionItem
}

type locationResult struct {
	locations []lsp.Location
}

type hoverResult struct {
	text string
}

type autocompleteTrigger struct{}

type signatureHelpResult struct {
	label      string
	paramStart int
	paramEnd   int
}

func fileURI(path string) string {
	return "file://" + path
}

func uriToPath(uri string) string {
	return strings.TrimPrefix(uri, "file://")
}

func lspToUICompletions(items []lsp.CompletionItem) []ui.CompletionItem {
	result := make([]ui.CompletionItem, 0, len(items))
	for _, item := range items {
		uiItem := ui.CompletionItem{
			Label:      item.Label,
			Detail:     item.Detail,
			InsertText: item.InsertText,
			FilterText: item.FilterText,
			SortText:   item.SortText,
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

func lspToSignatureHelpResult(sig *lsp.SignatureHelp) *signatureHelpResult {
	idx := sig.ActiveSignature
	if idx < 0 || idx >= len(sig.Signatures) {
		idx = 0
	}
	info := sig.Signatures[idx]
	result := &signatureHelpResult{label: info.Label}

	paramIdx := sig.ActiveParameter
	if paramIdx >= 0 && paramIdx < len(info.Parameters) {
		param := info.Parameters[paramIdx]
		var offsets [2]int
		if err := json.Unmarshal(param.Label, &offsets); err == nil {
			result.paramStart = offsets[0]
			result.paramEnd = offsets[1]
		} else {
			var label string
			if err := json.Unmarshal(param.Label, &label); err == nil {
				start := strings.Index(info.Label, label)
				if start >= 0 {
					result.paramStart = start
					result.paramEnd = start + len(label)
				}
			}
		}
	}
	return result
}
