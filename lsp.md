# LSP Features

## Implemented

- [x] initialize / initialized / shutdown / exit
- [x] textDocument/didOpen
- [x] textDocument/didChange
- [x] textDocument/didClose
- [x] textDocument/completion
- [x] completionItem/resolve (with additionalTextEdits for auto-import)
- [x] textDocument/signatureHelp
- [x] textDocument/hover
- [x] textDocument/definition
- [x] textDocument/implementation
- [x] textDocument/typeDefinition

## Not Implemented

### High Impact

- [x] textDocument/publishDiagnostics — error/warning squiggles, problems panel, hover popup, status bar
- [x] textDocument/didSave — save notifications to trigger server re-analysis
- [x] textDocument/references — find all references to a symbol
- [x] textDocument/rename — rename symbol across files
- [x] textDocument/formatting — format document on demand or on save
- [x] textDocument/rangeFormatting — format selection
- [x] textDocument/codeAction — source actions (organize imports, fix all) on save and via command palette

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
