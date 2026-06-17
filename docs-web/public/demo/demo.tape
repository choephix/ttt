Output docs-web/public/demo/demo.gif
Output docs-web/public/demo/demo.webm

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Padding 10
Set Theme "Catppuccin Mocha"

# Launch ttt on the project
Type "bin/ttt ."
Enter
Sleep 1.5s

# 1. Fuzzy file open
Ctrl+p
Sleep 300ms
Backspace
Sleep 300ms
Type "git.go"
Sleep 300ms
Enter
Sleep 500ms

# 2. Global search (Ctrl+K F chord)
Ctrl+k
Sleep 200ms
Type "f"
Sleep 500ms
Type "RepoRoot"
Sleep 1s
Escape
Sleep 500ms

# 3. Multi-cursor edit
Ctrl+p
Sleep 300ms
Backspace
Sleep 300ms
Type "git_test"
Sleep 300ms
Enter
Sleep 500ms
Ctrl+f
Sleep 300ms
Type "want"
Sleep 300ms
Escape
Sleep 300ms
Ctrl+d
Ctrl+d
Ctrl+d
Sleep 500ms
Type "expected"
Sleep 800ms
Escape
Sleep 300ms
Ctrl+z
Sleep 500ms

# 4. Git changes — stage and commit
Ctrl+p
Sleep 300ms
Type "Git: Stage"
Sleep 300ms
Enter
Sleep 500ms
Ctrl+p
Sleep 300ms
Type "Git: Commit"
Sleep 300ms
Enter
Sleep 500ms
Type "demo: test commit"
Enter
Sleep 800ms

# 5. Integrated terminal (Ctrl+T)
Ctrl+t
Sleep 800ms
Type "make test"
Enter
Sleep 3s
Type "htop"
Enter
Sleep 1.5s
Type "q"
Sleep 300ms
Ctrl+t
Sleep 500ms

# 6. Theme switching — cycle through 3 themes
Ctrl+p
Sleep 300ms
Type "Switch Theme"
Sleep 500ms
Enter
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 1s

Ctrl+p
Sleep 300ms
Type "Switch Theme"
Sleep 500ms
Enter
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 1s

Ctrl+p
Sleep 300ms
Type "Switch Theme"
Sleep 500ms
Enter
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 1s
