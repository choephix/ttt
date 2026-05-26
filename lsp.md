# LSP Features

## Implemented

- [x] initialize / initialized / shutdown / exit
- [x] textDocument/didOpen
- [x] textDocument/didChange
- [x] textDocument/didClose
- [x] textDocument/didSave — save notifications to trigger server re-analysis
- [x] textDocument/publishDiagnostics — error/warning squiggles, problems panel, hover popup, status bar
- [x] textDocument/completion
- [x] completionItem/resolve (with additionalTextEdits for auto-import)
- [x] textDocument/signatureHelp
- [x] textDocument/hover
- [x] textDocument/definition
- [x] textDocument/implementation
- [x] textDocument/typeDefinition
- [x] textDocument/references — find all references in bottom panel
- [x] textDocument/rename — rename symbol across files (F2)
- [x] textDocument/formatting — format document on demand or on save
- [x] textDocument/rangeFormatting — format selection
- [x] textDocument/codeAction — source actions (organize imports, fix all) on save and via command palette

## Not Implemented

### Medium Impact

- [ ] textDocument/documentSymbol — outline view, go-to-symbol within file
- [ ] workspace/symbol — search symbols across the project
- [ ] textDocument/documentHighlight — highlight other occurrences of symbol under cursor
- [ ] textDocument/foldingRange — code folding

### Low Impact

- [ ] textDocument/selectionRange — smart expand/shrink selection
- [ ] textDocument/codeLens — inline actions above functions
- [ ] textDocument/inlayHint — inline type annotations
- [ ] textDocument/linkedEditingRange — linked editing (e.g. rename HTML tag pairs)
