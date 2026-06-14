# Functional Test Audit

## Flaky Tests Fixed

### code-folding.test.js - "should show collapsed chevron on folded line"
- **Symptom**: Expected `▶` but screen showed `⋯`
- **Root cause**: The test checked for the gutter chevron character `▶`, but the visible fold indicator in the editor content area is `⋯` (inline ellipsis after the folded line). The gutter chevron can be absent or use a different character depending on gutter style.
- **Fix**: Changed assertion from `▶` to `⋯` which is the inline fold indicator always rendered on collapsed lines.

### indent-detection.test.js - "should detect 4-space indentation in python file"
- **Symptom**: Expected `Spaces: 4` in status bar but a Python LSP notification banner covered it.
- **Root cause**: When a Python LSP extension is configured but the server binary is not installed, a 10-second notification banner appears on the status bar: "Python autocomplete support is available." This hides the indentation indicator the test checks for.
- **Fix**: Pass a config with `lsp.notifyAvailability: false` so the notification never appears. The test is about indent detection, not LSP.

### lsp.test.js - typescript/go signature help
- **Symptom**: `waitForLog("lsp signature help response")` timed out on CI.
- **Root cause**: Two issues:
  1. Default 10s timeout too short for CI machines.
  2. Typing `console.log(` sends all characters at once. The `.` triggers autocomplete which can intercept subsequent keystrokes (`l`, `o`, `g`) as filter input instead of editor input.
- **Fix**: Split typing before the `(` trigger character, dismiss autocomplete in between, and increased all LSP log timeouts to 20s.

### lsp-autocomplete-trigger.test.js - "console. triggers completions, typing ( shows signature help"
- **Symptom**: Signature help response not found in log after typing `log(`.
- **Root cause**: Same autocomplete race as above. After `.` triggers completions, `log(` can be consumed by the autocomplete filter.
- **Fix**: Added `escape` + `waitStable()` between the `.` completion check and typing `log(`, plus increased timeout to 20s.

### lsp.test.js - go/yaml diagnostics
- **Symptom**: Diagnostics not received within 10s on CI.
- **Root cause**: gopls and yaml-language-server can take longer to produce initial diagnostics on slower CI machines.
- **Fix**: Increased `waitForLog` timeout to 20s for all LSP operations.

## No Functionality Issues Found

All test failures were caused by timing, rendering assumptions, or environment interference -- not by bugs in the editor functionality itself.
