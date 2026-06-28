# QA Bug Report — ttt Editor

Generated: 2026-06-27

## Summary

- **Total bugs found**: 43
- Critical: 2 | Major: 14 | Minor: 21 | Cosmetic: 6
- **Categories tested**: view, edit, transform, sidebar, tabs, terminal, folding, file, edge, mouse (partial), settings (partial)
- **Categories incomplete**: find (agent stopped before submitting)

## Critical

### ~~BUG-001: Multi-line text transform undo causes content loss and file corruption after one Ctrl+Z~~ FIXED
- **Category**: transform
- **Severity**: critical
- **Steps to reproduce**: 1. Open any text file with 3+ lines
2. Position cursor at start of line 1
3. Press shift+down three times to select lines 1-3
4. Run 'Transform to Uppercase' via command palette
5. Press Ctrl+Z (Undo) once
- **Expected**: One Ctrl+Z should fully restore the original three lines. The file should look exactly as before the transform.
- **Actual**: After one Ctrl+Z, the original lines 1-3 are GONE from the file. Lines that were previously below the selection (lines 4+) shift up and appear as lines 1+. If the user saves now, the original content is permanently lost. A second Ctrl+Z is required to fully restore the file.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/transform_final_multi1undo.txt shows original lines 1-3 (hello world, HELLO WORLD, Hello World) gone after 1 undo, replaced by former lines 4+. transform_final_multi2undo.txt shows correct restoration after 2 undos. Same root cause as transform-001: transformSelection() in /home/enko/Documents/ttt/internal/ui/editor_widget.go uses BreakGroup() creating non-atomic delete+paste undo groups.`

### ~~BUG-002: Terminal: Close All panics with slice bounds out of range when 2+ terminals are open~~ FIXED
- **Category**: terminal
- **Severity**: critical
- **Steps to reproduce**: 1. Open the editor
2. Run 'Terminal: New Terminal' to open first terminal (or 'Terminal: Toggle Terminal')
3. Wait for shell to load
4. Run 'Terminal: New Terminal' to open a second terminal
5. Wait for second shell to load
6. Run 'Terminal: Close All'
- **Expected**: All terminals close gracefully, the terminal panel shows 'No terminals. Press + to create one.'
- **Actual**: The app crashes with: panic: runtime error: slice bounds out of range [1:0] (or [2:1] with 3 terminals) inside CloseTerminal. This is a race condition: CloseTerminal calls tt.Term.Close() which blocks on <-t.done until the readLoop goroutine finishes. The readLoop goroutine calls OnExit() → PostEvent(panelID). In the exec harness (which runs on a separate goroutine), the main event loop can process the posted event and call CloseTerminal for the same terminal concurrently, causing the Terminals slice to be modified twice — leading to an out-of-bounds access on the second removal.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/terminal_crash.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/terminal_crash2.txt`

## Major

### ~~BUG-003: Non-existent file from CLI - silently ignored with no error shown~~ FIXED
- **Category**: file
- **Severity**: major
- **Steps to reproduce**: 1. Run: /home/enko/Documents/ttt/bin/ttt /path/to/nonexistent_file.txt
2. Wait for editor to start
3. Observe the tab and status bar
- **Expected**: Either open an empty tab named 'nonexistent_file.txt' ready to save to that path (VS Code behavior), or show a clear error message in the status bar explaining the file was not found
- **Actual**: The default 'untitled' tab is shown with no error message. The specified file path is silently discarded. Root cause: OpenFile() is called during BuildApp() before Init() registers the OnError→StatusError callback, so the error is only logged via slog.Error() and never reaches the UI.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/file_nonexist_check.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/file_nonexist.txt`

### ~~BUG-004: Text transform (uppercase/lowercase/titlecase) undo leaves line empty after one Ctrl+Z~~ FIXED (same fix as BUG-001)
- **Category**: transform
- **Severity**: major
- **Steps to reproduce**: 1. Open any text file
2. Navigate to line 1 (hello world)
3. Press shift+end to select the entire line
4. Run command palette: 'Transform to Uppercase'
5. Press Ctrl+Z (Undo) once
- **Expected**: One Ctrl+Z should fully restore the original text: 'hello world'
- **Actual**: After one Ctrl+Z, the line becomes EMPTY. The text is not restored. A second Ctrl+Z is required to get back to 'hello world'.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/transform_final_1undo.txt (line 1 empty after 1 undo), transform_final_2undo.txt (line 1 restored after 2 undos). Root cause: /home/enko/Documents/ttt/internal/ui/editor_widget.go in transformSelection() calls e.Undo.BreakGroup() before two separate e.exec() calls (DeleteSelectionCommand then InsertStringCommand), making them separate undo groups instead of one atomic operation. Fix: use BatchCommand like ToggleLineComment does at line 1930.`

### ~~BUG-005: View: Close All Tabs silently discards unsaved changes without confirmation~~ FIXED
- **Category**: view
- **Severity**: major
- **Steps to reproduce**: 1. Open a file (e.g. bin/ttt main.go)
2. Type any text to modify the buffer
3. Run command 'View: Close All Tabs'
4. Observe result
- **Expected**: A dialog should appear asking the user to Save, Discard, or Cancel for each modified file before closing — matching VS Code behavior for this command
- **Actual**: All tabs are silently closed immediately. The unsaved changes are lost with no prompt. A new empty 'untitled' tab opens instead.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/view_step60_close_all_no_confirm.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/view_state60_close_all_no_confirm.json, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/view_step70_closeall_named_modified.txt. Root cause: the 'tab.closeAll' command calls app.EditorGroup.CloseAllTabs() directly (internal/app/commands_editor.go:291), which unconditionally replaces all tabs without checking dirty state, unlike app.CloseTab() which correctly checks IsDirty() and shows a confirmation dialog.`

### BUG-006: View: Close Other Tabs silently discards unsaved changes in non-active tabs
- **Category**: view
- **Severity**: major
- **Steps to reproduce**: 1. Open two files (e.g. bin/ttt main.go app.go)
2. app.go is active by default — type any text to modify it
3. Press 'View: Previous Tab' to switch to main.go
4. Run command 'View: Close Other Tabs'
5. Observe result
- **Expected**: A dialog should appear asking to Save, Discard, or Cancel for app.go (which has unsaved changes) before closing it
- **Actual**: app.go is silently closed with all unsaved changes lost. No confirmation is shown. Only main.go remains.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/view_step62_close_other_test.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/view_state62d_after_close.json. The debug state confirms: after typing in app.go (modified: True) and running Close Other Tabs, only main.go (modified: False) remains. Root cause: 'tab.closeOthers' command calls app.EditorGroup.CloseOtherTabs() directly (internal/app/commands_editor.go:283), which replaces the tabs array unconditionally without checking dirty state on the tabs being discarded.`

### BUG-007: Terminal: Toggle Fullscreen off hides the terminal panel instead of restoring to split view
- **Category**: terminal
- **Severity**: major
- **Steps to reproduce**: 1. Open the editor
2. Run 'Terminal: Toggle Terminal' to open the terminal in split view (bottom ~39% of screen)
3. Run 'Terminal: Toggle Fullscreen' — terminal expands to fill full right-side area (editor is hidden)
4. Run 'Terminal: Toggle Fullscreen' again to exit fullscreen
- **Expected**: The terminal panel returns to its previous split-view position, still visible at the bottom ~39% with the editor above it (matching VS Code behavior where un-maximizing the terminal restores the split view)
- **Actual**: The terminal panel is hidden entirely — ShowBottom is set to false via HideBottomPanel(). The code at commands_view.go line 26-27 checks 'if ShowBottom && BottomH >= fullH { HideBottomPanel() }' which treats exiting fullscreen the same as toggling the panel off. The user must manually re-open the terminal after exiting fullscreen.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/terminal_step8a_before_fs.txt (TERMINAL visible), /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/terminal_step8b_during_fs.txt (fullscreen), /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/terminal_step8c_after_fs.txt (panel gone)`

### BUG-008: Search result click: matched line not visible (viewport off by 1 row)
- **Category**: sidebar
- **Severity**: major
- **Steps to reproduce**: 1. Open editor with a project directory
2. Open Find panel (click Find tab or exec 'Show Search')
3. Type a search query (e.g. 'func')
4. Wait for results to appear
5. Click on a specific line result (e.g. '3: func Helper() string {')
6. Observe the editor viewport
- **Expected**: Editor scrolls to show the matched line (line 3) at or near the top of the viewport, with cursor placed on that line
- **Actual**: Editor shows from line N+1 (e.g. line 4 when the match is on line 3). The cursor is correctly placed at Ln 3 Col 1 per status bar, but that line is above the viewport and not visible. The user cannot see the matched code.
- **Evidence**: `sidebar_step_main_func_result.txt (Ln 3 cursor, viewport shows line 4+), sidebar_step31_click_result_line.txt (Ln 5 cursor, viewport shows line 6+), sidebar_step_final_click_result.txt (Ln 3 cursor, viewport shows line 4+). Files at /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/`

### BUG-009: After clicking search result, keypresses modify search query instead of editing opened file
- **Category**: sidebar
- **Severity**: major
- **Steps to reproduce**: 1. Open Find panel (exec 'Show Search')
2. Type a search query (e.g. 'TestButton')
3. Wait for results
4. Click on the specific result line (e.g. '5: func TestButton(t *testing.T) {}')
5. Immediately type some text (e.g. 'hello')
- **Expected**: After clicking a search result to open/navigate to a file, focus moves to the editor. Subsequent typing edits the file at the cursor position.
- **Actual**: Focus returns to the search input field after clicking the result. Typing 'hello' changes the search query from 'TestButton' to 'TestButtonhello', triggering a new search (showing 'No results'). The opened file is not modified.
- **Evidence**: `sidebar_step32_type_after_result.txt shows search input changed to 'TestButtonhello' and 'No results', file was not modified. Files at /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/`

### BUG-010: Ctrl+Right / Ctrl+Left keybindings move by 1 char instead of jumping by word
- **Category**: edit
- **Severity**: major
- **Steps to reproduce**: 1. Open any file with text (e.g. 'Hello World'). 2. Cursor is at col 0. 3. Press Ctrl+Right (bound to editor.moveWordRight). 4. Observe cursor position.
- **Expected**: Cursor jumps forward to the end of the current word (col 5 after 'Hello' in 'Hello World'), matching standard editor and VS Code behavior for word movement.
- **Actual**: Cursor advances only 1 character (col 0 → col 1). Ctrl+Left behaves identically — moves 1 character backward instead of jumping to the previous word boundary. The exec 'Move Word Right' command (via command palette) correctly jumps to col 5, confirming the command logic is correct but the keybinding is broken.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edit_state19_ctrl_right.json (cursor at col 1), edit_state44_ctrl_left.json (cursor at col 10 instead of word start), edit_state17_word_right.json (exec correctly lands at col 5)`

### BUG-011: Alt+Backspace / Alt+Delete / Ctrl+Delete keybindings delete only 1 char instead of a whole word
- **Category**: edit
- **Severity**: major
- **Steps to reproduce**: 1. Open file with 'Hello World' on line 1. 2. Press End to move to col 11 (end of line). 3. Press Alt+Backspace (bound to editor.deleteWordLeft). 4. Check cursor position and line content.
- **Expected**: Alt+Backspace deletes the word to the left ('World', 5 chars), leaving 'Hello ' with cursor at col 6. Alt+Delete and Ctrl+Delete (bound to editor.deleteWordRight) should delete the word to the right of cursor.
- **Actual**: Alt+Backspace deletes only the single character immediately to the left ('d'), leaving 'Hello Worl' with cursor at col 10. Alt+Delete and Ctrl+Delete also each delete only 1 character forward. Root cause: EditorPaneWidget.HandleEvent matches tcell.KeyBackspace and tcell.KeyDelete without checking for Alt/Ctrl modifiers, consuming the event before global key handlers run. The exec 'Delete Word Left' command correctly deletes the full word (cursor moves to col 6), confirming the command is working but the keybinding is broken.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edit_state45_alt_bs.json (cursor at col 10 instead of col 6), edit_screen46_alt_del.txt ('ello World' after alt+delete from col 0), edit_state_key_delword.json vs edit_state_palette_delword.json (col 10 vs col 6)`

### BUG-012: View: Close All Tabs discards unsaved changes without prompting
- **Category**: edge
- **Severity**: major
- **Steps to reproduce**: 1. Open a named file (e.g. bin/ttt /tmp/test.txt)
2. Type some text to modify the file (tab shows ● indicator)
3. Run 'View: Close All Tabs' from command palette
4. Observe result
- **Expected**: A 'Save changes to [filename]?' dialog should appear with Save/Discard/Cancel options, matching VS Code behavior
- **Actual**: The modified file is closed immediately without any dialog. Unsaved changes are silently discarded. A new untitled tab opens.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edge_step32a_before.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edge_step32b_after.txt`

### BUG-013: View: Close Other Tabs discards unsaved changes in background tabs without prompting
- **Category**: edge
- **Severity**: major
- **Steps to reproduce**: 1. Open 3 files: bin/ttt file1.txt file2.txt file3.txt
2. Navigate to file2.txt and type text to modify it (shows ● indicator)
3. Navigate to another tab (e.g. file3.txt)
4. Run 'View: Close Other Tabs' from command palette
- **Expected**: A 'Save changes to file2.txt?' dialog should appear for any other tab with unsaved changes before closing
- **Actual**: All other tabs including the modified file2.txt are closed immediately without any dialog. Unsaved changes are silently discarded.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edge_step34b_before_close.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edge_step34c_after_close_other.txt`

## Minor

### BUG-014: Tab bar clips partial tab names in overflow mode without truncation indicator
- **Category**: tabs
- **Severity**: minor
- **Steps to reproduce**: 1. Open 12 or more files: `bin/ttt file1.go file2.go ... file12.go`
2. Note file_8.go (last file) is the active tab
3. Observe tab bar: there are hidden tabs to the left (◀ indicator visible)
4. Navigate to the first tab by pressing Ctrl+K , eleven times
5. Observe both edges of the tab bar
- **Expected**: Tabs that do not fully fit in the visible area should either be hidden completely (not rendered) or shown with an ellipsis truncation (e.g., 'gam...' for 'gamma.go'). The ◀/▶ arrows should indicate hidden tabs.
- **Actual**: Partially-visible tabs show raw clipped filenames at both edges. On the LEFT: 'a.go' is displayed instead of 'gamma.go' (showing only the last 4 characters of the filename). On the RIGHT: 'file_6.g' is displayed instead of 'file_6.go' (missing the trailing 'o'). These appear as genuine filenames and can mislead users into thinking there is a file called 'a.go'.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/tabs_step58_overflow_confirm.txt (left overflow 'a.go'), /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/tabs_step59_both_overflow.txt (right overflow 'file_6.g')`

### BUG-015: Tab bar overflow menu (⋮) only shows 'Close All', missing list of hidden tabs
- **Category**: tabs
- **Severity**: minor
- **Steps to reproduce**: 1. Open 12 or more files to cause tab bar overflow
2. Click the ⋮ button at the far right of the tab bar
- **Expected**: The overflow menu should list all open tabs (especially those hidden off-screen) so users can navigate to them directly, similar to VS Code's tab overflow menu which shows a list of all tabs.
- **Actual**: The ⋮ dropdown only contains a single option: 'Close All'. There is no way to see or navigate to tabs that have scrolled off-screen via this menu. Users must use Ctrl+K , / Ctrl+K . repeatedly to cycle through hidden tabs.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/tabs_step43_more_button.txt`

### BUG-016: New File (Ctrl+N) always appends tab at end of list instead of after current tab
- **Category**: tabs
- **Severity**: minor
- **Steps to reproduce**: 1. Open three files: `bin/ttt alpha.go beta.go gamma.go`
2. Navigate to beta.go (middle tab) with Ctrl+K ,
3. Press Ctrl+N (or use 'New File' from command palette)
4. Observe the position of the new 'untitled' tab
- **Expected**: The new untitled tab should be inserted immediately after the currently active tab: [alpha.go, beta.go, untitled, gamma.go] with untitled active. This matches VS Code behavior where new tabs open next to the current tab.
- **Actual**: The new untitled tab is always appended at the end of the tab list regardless of which tab is currently active: [alpha.go, beta.go, gamma.go, untitled]. The new tab is at the end even when the active tab is in the middle.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/tabs_state54_new_file_position.json (active_tab=1 was beta.go, untitled at index 3 = end), /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/tabs_state55_new_from_middle.json`

### BUG-017: File explorer single-click on a file replaces current tab content instead of opening new tab
- **Category**: tabs
- **Severity**: minor
- **Steps to reproduce**: 1. Open a directory: `bin/ttt /path/to/dir/`
2. The editor starts with one 'untitled' empty tab and the file explorer visible
3. Click on a file in the explorer (e.g., alpha.go) — it replaces 'untitled' and opens in the same tab (1 tab total)
4. Click on a different file (e.g., beta.go) — it replaces alpha.go (still 1 tab total)
5. Compare: opening files via command line creates separate tabs
- **Expected**: Clicking a file in the explorer should open it in a new tab (or VS Code-style 'preview' tab). Each file click should add a new tab, similar to how multiple command-line arguments create multiple tabs.
- **Actual**: When the current tab is 'untitled', clicking a file in the explorer replaces the untitled tab content (expected). However, subsequent explorer clicks on different files replace the current tab's content rather than opening new tabs. Only if the file is already open in another tab does it switch to the existing tab. Starting from a directory with no explicit files, clicking through multiple files leaves only 1 tab open at a time.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/tabs_state3_all_open.json (after clicking 4 files in explorer, only 1 tab 'gamma.go' remains)`

### BUG-018: Go to File picker - filename truncated and fused with directory name without separator
- **Category**: file
- **Severity**: minor
- **Steps to reproduce**: 1. Open ttt on a project with deep git worktree paths (e.g., .claude/worktrees/...)
2. Invoke Go to File (Ctrl+K P)
3. Observe list items where the directory path is very long
- **Expected**: The filename column should remain readable. When space is tight, the directory path should be truncated with an ellipsis, and there should always be a visual gap or separator between the filename and directory columns.
- **Actual**: The filename is truncated to only a few characters and the directory path is appended directly with no separator. For example: 'settings.local.json' becomes 'settin.claude/worktrees/agent-a07104541439a4bc5/.claude' making it unreadable and ambiguous.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/file_gotofile_display.txt`

### BUG-019: No read-only file indicator when opening files with restricted permissions
- **Category**: file
- **Severity**: minor
- **Steps to reproduce**: 1. Create a read-only file: echo 'content' > /tmp/test.txt && chmod 444 /tmp/test.txt
2. Open it in ttt
3. Edit and save (Ctrl+S)
- **Expected**: A clear read-only indicator should appear in the tab title (e.g., a lock icon or '(read-only)') or in the status bar. Saving should either warn the user or be blocked until they explicitly choose to overwrite.
- **Actual**: No indicator is shown anywhere. The file opens and can be edited normally. Ctrl+S saves silently without any warning. On Linux, the atomic save (temp file + rename in same directory) can succeed even for chmod 444 files when the containing directory is writable, so the user unknowingly overwrites a read-only file.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/file_readonly_open.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/file_readonly_save.txt`

### BUG-020: Go to Line dialog stays open and does not navigate when entering line number 0
- **Category**: folding
- **Severity**: minor
- **Steps to reproduce**: 1. Open any file in the editor
2. Press Ctrl+G (or run 'Go to Line' from command palette)
3. Type '0' in the dialog (it shows ':0')
4. Press Enter
5. Wait several seconds
- **Expected**: Dialog closes and cursor moves to line 1 (first line, clamped from 0), or an error message is shown with the dialog remaining open as feedback
- **Actual**: Dialog stays open indefinitely with no feedback and no navigation. The cursor does not move. User must press Escape to close the stuck dialog. The same issue occurs for any non-positive number (0, -1, etc.).
- **Evidence**: `folding_step52_goto0_verify.txt shows dialog still open after 500ms wait. folding_state52_goto0.json shows cursor.line=0 (unchanged initial position, no navigation occurred). Root cause: /home/enko/Documents/ttt/internal/ui/selectdialog_widget.go line 305 has condition `n > 0` which silently rejects 0 without calling OnGoToLine or OnDismiss. For comparison, line 1 works correctly: folding_step53_goto1.txt shows dialog dismissed and cursor at line 1.`

### BUG-021: Fold gutter indicators (▼) not shown on fresh file open — only appear after tab switching
- **Category**: folding
- **Severity**: minor
- **Steps to reproduce**: 1. Open a Go file with foldable blocks
2. Observe the gutter area next to line numbers
3. Compare with opening two files, then switching between tabs
- **Expected**: Downward triangle (▼) affordance indicators appear in the gutter on all foldable lines immediately when a file is opened, making foldable regions discoverable (consistent with VS Code behavior)
- **Actual**: No ▼ gutter indicators appear on fresh open. Indicators appear only after switching between two tabs (clicking away to another file and back). They also do not appear after running Fold All + Unfold All. This makes fold affordances invisible to users who never switch tabs.
- **Evidence**: `folding_step54_fresh_gutter.txt: fresh open shows lines 3, 10, 15, 20, 26, 32 without ▼ indicators. folding_step57_gutter_after_keys.txt: after ctrl+k 0 (Fold All) + ctrl+k 9 (Unfold All) keyboard shortcuts, still no ▼ indicators. folding_step56_gutter_after_tabswitch.txt: after clicking away to second_file.go and back, lines 3 and 10 show ▼ indicators. folding_step55_gutter_after_foldall.txt: after exec Fold All + Unfold All via command palette, still no ▼ indicators.`

### BUG-022: Toggle Syntax Highlight requires editor restart to take effect
- **Category**: folding
- **Severity**: minor
- **Steps to reproduce**: 1. Open any syntax-highlighted source file
2. Run 'Toggle Syntax Highlight' from the command palette (or View menu)
3. Observe the status bar and editor content
- **Expected**: Syntax highlighting toggles immediately without restarting the editor, as in VS Code where the toggle takes effect instantly
- **Actual**: Status bar shows notification: 'Restart to apply syntax highlight changes [OK]'. The syntax highlight state does not change until the editor is restarted. Toggling again shows the same notification again.
- **Evidence**: `folding_step33_syntax_off.txt: notification 'Restart to apply syntax highlight changes [OK]' visible in status bar after first toggle. folding_step50_syntax_toggle1.txt and folding_step51_syntax_toggle2.txt: same notification on both first and second toggle invocations.`

### BUG-023: Toggle Fold (ctrl+k [) has no effect when cursor is on closing brace
- **Category**: folding
- **Severity**: minor
- **Steps to reproduce**: 1. Open a Go file with functions
2. Navigate cursor to a closing brace line (e.g., line 12 '}'  of func Add)
3. Press Ctrl+K [ (Toggle Fold keybinding)
4. Observe whether the containing block folds
- **Expected**: Fold toggles the containing block (VS Code allows fold toggle from the closing brace position as well as the opening brace). Cursor should be able to initiate a fold from the closing bracket.
- **Actual**: Nothing happens. The fold is only triggered from the opening line of a block (e.g., the 'func Foo() {' line). Pressing ctrl+k [ while on the closing '}' line has no effect and gives no feedback.
- **Evidence**: `folding_step48_on_closing.txt: cursor on line 12 ('}' of Add function). folding_step49_fold_from_closing.txt: identical content after pressing ctrl+k [, no fold applied, no visible change.`

### BUG-024: Modified indicator (dot) not cleared after Undo restores file to saved state
- **Category**: transform
- **Severity**: minor
- **Steps to reproduce**: 1. Open a file (no dot indicator shown)
2. Run 'Toggle Line Comment' on line 1 (dot indicator appears)
3. Press Ctrl+Z to undo the comment toggle
4. Observe tab title
- **Expected**: After undoing all changes, the dot (●) modified indicator should disappear from the tab title since the file now matches its saved state on disk (as in VS Code)
- **Actual**: The dot (●) remains in the tab title even after fully undoing all changes. The file content is correctly restored, but the unsaved-changes indicator is not cleared.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/transform_t40_initial.txt (no dot), transform_t40_modified.txt (dot appears after comment), transform_t40_after_undo.txt (dot still shown after undo). Root cause: Buffer.Dirty in /home/enko/Documents/ttt/internal/core/buffer/buffer.go is only set to false on save/load, not when the undo stack returns to the save point.`

### BUG-025: Sort Lines Ascending/Descending without selection moves trailing empty line to top
- **Category**: transform
- **Severity**: minor
- **Steps to reproduce**: 1. Open a file that ends with a trailing newline (e.g., sort_test.txt: cherry, apple, banana, zebra, mango)
2. Do NOT make any selection
3. Run 'Sort Lines Ascending' (or ctrl+k o) without selecting anything
- **Expected**: All non-empty lines are sorted alphabetically. The trailing empty line (file terminator) should either be excluded from sorting or remain at the end.
- **Actual**: The trailing empty line is treated as a sortable line (empty string sorts before all words alphabetically) and moves to the first position. Result: (empty line), apple, banana, cherry, mango, zebra. The empty line is now at the top of the file.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/transform_t30_before.txt (original: cherry, apple, banana, zebra, mango, empty), transform_t30_after.txt (after sort: empty line at position 1 followed by sorted words).`

### BUG-026: View: Show Panel Tab dialog displays internal IDs instead of friendly panel names
- **Category**: view
- **Severity**: minor
- **Steps to reproduce**: 1. Open the bottom panel: run 'View: Toggle Panel'
2. Run command 'View: Show Panel Tab'
3. Observe the list of panel options in the dialog
- **Expected**: The dialog should show user-friendly display names matching the tab bar labels — e.g. 'Notepad', 'TODOs' for plugin panels
- **Actual**: The dialog shows raw internal IDs: 'terminal', 'problems', 'references', 'output', 'plugin.notepad', 'plugin.todo-scanner'. The 'plugin.' prefix and raw ID format are not user-friendly.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/view_step23_show_panel_tab.txt. Root cause: the 'panel.show' command handler (internal/app/commands_view.go:344) iterates over BottomPanel.PanelIDs() and uses each ID as both the SelectItem ID and Label. PanelIDs() returns internal IDs, not display titles. The fix is to expose panel titles alongside IDs (e.g. a PanelItems() method on TabbedPanel) and use those as labels.`

### BUG-027: View: Focus Terminal does not give keyboard focus when terminal panel is not currently visible
- **Category**: terminal
- **Severity**: minor
- **Steps to reproduce**: 1. Start the editor with no terminal open
2. Run 'View: Focus Terminal'
- **Expected**: The terminal panel opens AND keyboard focus is given to the terminal, so the user can immediately start typing commands without additional clicks (matching VS Code's Ctrl+` behavior)
- **Actual**: The terminal panel opens and a new terminal is spawned (ShowBottom is set, showTerminalPanel is called), but keyboard focus is NOT transferred to the terminal. Focus remains on the previously focused widget (the file explorer tree). The bug is in focusTerminal() in commands_view.go lines 132-139: the early 'return' statement after showTerminalPanel() exits before calling a.Root.SetFocus(), unlike the else branch (when panel is already visible) which correctly calls SetFocus.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/terminal_step10_focus_terminal.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/terminal_state10_focus_terminal.json`

### BUG-028: Opening file from Explorer keeps focus in sidebar tree; typing is silently consumed
- **Category**: sidebar
- **Severity**: minor
- **Steps to reproduce**: 1. Open editor with a project directory (sidebar showing Explorer)
2. Click on a file in the Explorer tree to open it (e.g. click on main.go)
3. Observe the opened file in the editor
4. Without clicking the editor, type some text
- **Expected**: After clicking a file to open it, focus should either move to the editor (VS Code default) or at minimum have a clear visual indicator that focus is in the sidebar. Typed characters should edit the file or at least produce a visible response.
- **Actual**: Focus stays on the Explorer Tree widget (confirmed via debug state: focused widget is Tree). The file opens correctly in the editor, but typing is silently consumed by the tree as navigation shortcuts with no visible effect on screen. The status bar still shows the untouched file state. Users must click the editor or press Escape before they can type.
- **Evidence**: `sidebar_step16_type_after_click_open.txt (typed 'test' but file unchanged, no modification indicator), sidebar_state15_click_open_file.json (focus field shows 'other', focused widget is Tree after file open). Files at /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/`

### BUG-029: Clicking on search result group header collapses the group instead of opening file
- **Category**: sidebar
- **Severity**: minor
- **Steps to reproduce**: 1. Open Find panel and search for text
2. Observe results grouped by file: '▼ tests/button_test.go (1)' followed by '  5: func TestButton ...'
3. Click on the file group header line '▼ tests/button_test.go (1)'
- **Expected**: Clicking on a file group header in search results should open the file (same as clicking the specific result line), or at minimum be a neutral action. In VS Code, clicking the file header opens the file.
- **Actual**: Clicking the file group header collapses the group (arrow changes from ▼ to ▶), hiding the child result lines. The file is NOT opened. No navigation occurs.
- **Evidence**: `sidebar_step30_click_testbutton.txt shows group collapsed (▶ icon) after click, editor still shows 'untitled' with no file opened. Files at /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/`

### BUG-030: Tab dirty indicator (●) does not clear after undoing back to the last saved state
- **Category**: edit
- **Severity**: minor
- **Steps to reproduce**: 1. Open file. 2. Type 'X' (file is now modified, ● appears). 3. Save the file (exec 'Save File'). 4. Confirm ● disappears after save. 5. Type 'Y'. 6. Press Ctrl+Z to undo. 7. Content is now identical to the last saved state.
- **Expected**: The tab's ● (modified) indicator clears when the buffer content matches the last saved state, matching VS Code behavior. The editor should track which undo step corresponds to the last save ('clean point') and clear the dirty flag when reached.
- **Actual**: After undoing back to the saved content, the tab still shows ● and the buffer reports modified=true. The indicator only clears if the file is saved again. Simpler case also reproduces: type ABC on a fresh file, then exec 'Undo' three times — content returns to original but ● remains and buffer reports modified=true.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edit_screen49_undo_to_save.txt (shows ● after undo to saved state), edit_state49_undo_to_save.json (modified: true), edit_screen41_dirty.txt (same pattern after simpler type+undo)`

### BUG-031: Cut and Copy without a selection are no-ops instead of acting on the current line
- **Category**: edit
- **Severity**: minor
- **Steps to reproduce**: 1. Open file with text. 2. Ensure no selection is active (cursor on line 1, no selection). 3. Execute 'Cut' command. 4. Observe buffer and clipboard.
- **Expected**: Cut without selection should cut the entire current line and place it on the clipboard (VS Code behavior). Copy without selection should copy the entire current line to the clipboard. This allows quick line operations without needing to manually select the whole line.
- **Actual**: Cut without selection is a complete no-op: buffer is unchanged (modified=false) and the clipboard is not updated with line content. Copy without selection also leaves the clipboard unchanged. The clipboard retains whatever was there from a prior operation.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edit_screen33_cut_nsel.txt (buffer unchanged after Cut with no selection), edit_state33_cut_nsel.json (modified: false, same 6 lines)`

### BUG-032: Delete Line on empty file causes viewport horizontal offset — first character typed after is invisible
- **Category**: edge
- **Severity**: minor
- **Steps to reproduce**: 1. Open editor with no files (new untitled buffer)
2. Run 'Delete Line' from command palette
3. Type any text (e.g. 'ABCDEFGH')
4. Observe the display
- **Expected**: All typed characters should be visible, starting from the first character. The line should show 'ABCDEFGH'.
- **Actual**: The first character typed is not rendered. Display shows 'BCDEFGH' (missing 'A'). Status bar correctly reports Ln 1, Col 9 indicating 8 chars exist. Pressing Home corrects the display, revealing the missing first character.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edge_step31f_type_after_del.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edge_step31g_after_home.txt`

### BUG-033: Go to Line dialog gives no feedback when input is invalid (non-numeric, 0, or negative)
- **Category**: edge
- **Severity**: minor
- **Steps to reproduce**: 1. Open any file
2. Run 'Go to Line' from command palette (shows dialog with ':' prefix)
3. Type 'abc' (or '0' or '-1')
4. Press Enter
- **Expected**: Either show an inline error message (e.g. 'Invalid line number') and keep dialog open for correction, or accept 0/-1 by clamping to the nearest valid line (line 1)
- **Actual**: The dialog stays open with the invalid input still showing, and the cursor does not move. No error message or visual feedback is provided. The user has no indication why Enter had no effect.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edge_step15a_gotoline_afterenter.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edge_step26a_gotoline0.txt, /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/edge_step26b_gotolineneg.txt`

## Cosmetic

### BUG-034: Close button (x) is only visible on the active tab; inactive tabs show no close button
- **Category**: tabs
- **Severity**: cosmetic
- **Steps to reproduce**: 1. Open two or more files: `bin/ttt alpha.go beta.go gamma.go`
2. Observe the tab bar — the active tab (gamma.go) shows '│ gamma.go x │' with the x close button
3. Observe the inactive tabs (alpha.go, beta.go) in the tab bar
- **Expected**: All tabs should show a close button (or show it on hover in environments that support hover). Users should be able to close any tab with a single click without first needing to activate it.
- **Actual**: Only the active tab shows the 'x' close button. Inactive tabs display only the filename with no close affordance. To close an inactive tab, users must first click it to make it active (which changes the editor content), then click the 'x' button. This requires 2 clicks and changes the active buffer unnecessarily.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/tabs_step6_multi_files.txt (row 3 shows inactive tabs without x, active tab 'delta.txt x' with x)`

### BUG-035: New untitled file naming skips '-1', jumping directly to '-2'
- **Category**: file
- **Severity**: cosmetic
- **Steps to reproduce**: 1. Launch ttt with no files (default untitled tab opens)
2. Run 'New File' three times via command palette or Ctrl+N
3. Observe the tab names
- **Expected**: Consistent sequential numbering: 'untitled-1', 'untitled-2', 'untitled-3', 'untitled-4' (VS Code: 'Untitled-1', 'Untitled-2', etc.)
- **Actual**: Tabs are named: 'untitled', 'untitled-2', 'untitled-3', 'untitled-4'. The first new file has no number and the subsequent ones start from -2, skipping -1.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/file_naming2.txt`

### BUG-036: Inconsistent editor border box-drawing style across sessions (rounded vs straight corners)
- **Category**: folding
- **Severity**: cosmetic
- **Steps to reproduce**: 1. Open a file and observe the editor pane border style
2. Close and reopen the editor (or run tests in separate invocations)
3. Compare border styles between runs
- **Expected**: Editor window border uses a consistent box-drawing character style across all sessions (e.g., always rounded corners ╭──╮ or always straight corners ┌──┐)
- **Actual**: Editor border alternates between rounded corners (╭──╮ with U+256D) and straight corners (┌──┐ with U+250C) across different invocations with no user-visible action to explain the change. A third style (double-line ╔══╗) appears momentarily during mouse tab-click operations.
- **Evidence**: `folding_step1_initial.txt: rounded corners ╭──╮ on first invocation. folding_step26_linenums_on.txt: straight corners ┌──┐ in a later invocation. folding_step44_before_nav.txt: rounded corners ╭──╮ again. folding_step20_fold_active.txt: double-line ╔══╗ during tab switching.`

### BUG-037: Debug state 'focus' field always returns 'other' regardless of which pane is focused
- **Category**: view
- **Severity**: cosmetic
- **Steps to reproduce**: 1. Open the editor with any file
2. Run 'View: Focus Editor', capture debug state
3. Run 'View: Focus Sidebar', capture debug state
4. Run 'View: Focus Panel', capture debug state
5. Check the 'focus' field in all three debug states
- **Expected**: The 'focus' field should reflect the actual focused region: 'editor', 'sidebar', or 'bottom_panel' respectively
- **Actual**: All three states return 'focus: other'. The describeFocus() function checks for *ui.EditorPaneWidget but the editor group uses *ui.EditorGroupWidget as Root.Focused. Similarly, focus on sidebar/panel is set to their active child widgets (WidgetAdapter, TerminalPanelWidget), not the SidebarWidget/BottomPanelWidget containers that describeFocus() checks for. The only case that works is 'search' (SearchWidget). This only affects the debug dump — functionality is not impacted.
- **Evidence**: `/tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/view_state5_focus_editor.json (shows focus: other after View: Focus Editor), /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/view_state7_focus_panel.json (shows focus: other after View: Focus Panel). Root cause in internal/app/debug_dump.go:156 — the type switch checks EditorPaneWidget instead of EditorGroupWidget, and checks SidebarWidget/BottomPanelWidget instead of the widgets that are actually set as Root.Focused.`

### BUG-038: Sidebar tab bar shows no visual indicator of which panel is currently active
- **Category**: sidebar
- **Severity**: cosmetic
- **Steps to reproduce**: 1. Open editor with project directory
2. Note tab bar shows 'Explore  Find  Changes  >>  ...'
3. Click the Find tab to switch to Search panel
4. Note tab bar still shows 'Explore  Find  Changes  >>  ...' with no visual change to indicate Find is now active
- **Expected**: The active sidebar tab should have a visual distinction (underline, different background, bold text, or other indicator) so users can see which panel is currently active at a glance
- **Actual**: All tab labels appear identical in text rendering regardless of which panel is active. While this may use colors in a real terminal that are not captured in text screenshots, even the text layout contains no structural indicator (no underline character, no bracket, no arrow) marking the active tab.
- **Evidence**: `sidebar_step8_click_find.txt vs sidebar_step8_click_changes.txt vs sidebar_step8_click_explore.txt all show identical tab bar text 'Explore  Find  Changes  >>  ...'. Files at /tmp/claude-1000/-home-enko-Documents-ttt/f9eb676f-0e80-45ca-9c81-8abb71498225/scratchpad/qa/`

### BUG-039: Bottom panel tab clicks do not switch the active tab
- **Category**: mouse
- **Severity**: major
- **Steps to reproduce**: 1. Open terminal (Terminal: Toggle Terminal) to show bottom panel
2. Bottom panel shows tabs: TERMINAL, PROBLEMS, REFERENCES, OUTPUT, etc.
3. Click on PROBLEMS tab, or any other non-active tab
- **Expected**: Clicking a tab should switch the active bottom panel to that tab
- **Actual**: The active panel stays on "terminal" regardless of which tab is clicked. Tested clicks at X positions 3, 5, 13, 15, 23, 25, 33, 35, 41, 43, 51, 53 — all kept active=terminal.
- **Evidence**: `mouse_bptab_x*.json files all show bottom_panel.active=terminal`

### BUG-040: Command palette not dismissed by clicking outside its bounds
- **Category**: mouse
- **Severity**: minor
- **Steps to reproduce**: 1. Open command palette (Ctrl+P)
2. Click outside the palette area (e.g. in the editor below)
- **Expected**: Clicking outside the palette should dismiss it (standard behavior in VS Code)
- **Actual**: The palette stays open after clicking outside. Escape works correctly to dismiss.
- **Evidence**: `mouse_test19b_state.json shows overlay still present after clicking at (5, 30)`

### BUG-041: Debug state focus field always reports "other" regardless of actual focus
- **Category**: mouse
- **Severity**: cosmetic
- **Steps to reproduce**: 1. Click in editor, sidebar, or bottom panel
2. Capture debug state
- **Expected**: Focus field should report "editor", "sidebar", "bottom", etc.
- **Actual**: Focus field always shows "other". Same root cause as BUG-037 — the debug dump type switch doesn't match the actual focused widget types.
- **Evidence**: `All mouse_*.json state files show focus="other"`

> Note: BUG-041 is a duplicate of BUG-037, listed here for mouse category completeness.
