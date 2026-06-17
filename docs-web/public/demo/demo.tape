Output docs-web/public/demo/demo.gif
Output docs-web/public/demo/demo.webm

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Padding 10
Set Theme "Catppuccin Mocha"

# Launch ttt on the project with default-dark theme
Type "bin/ttt --config config/settings.json ."
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

# 3. Integrated terminal (Ctrl+T)
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

# 4. Theme switching — search themes by name
Ctrl+p
Sleep 300ms
Type "Switch Theme"
Sleep 500ms
Enter
Sleep 500ms
Type "mo"
Sleep 1s
Backspace
Backspace
Type "nord"
Sleep 1s
Backspace
Backspace
Backspace
Backspace
Type "turbo"
Sleep 1s
Backspace
Backspace
Backspace
Backspace
Backspace
Type "nord"
Sleep 1s
Enter
Sleep 500ms

# 5. Menu bar navigation (Alt+F then Right to browse)
Alt+f
Sleep 400ms
Right
Sleep 400ms
Right
Sleep 400ms
Right
Sleep 400ms
Right
Sleep 400ms
Right
Sleep 400ms
Escape
Sleep 500ms

# 6. PR review
Ctrl+p
Sleep 300ms
Type "Review PR"
Sleep 300ms
Enter
Sleep 500ms
Type "https://github.com/eugenioenko/ttt/pull/192"
Enter
Sleep 3s
Down
Sleep 100ms
Down
Sleep 100ms
Down
Sleep 100ms
Down
Sleep 100ms
Enter
Sleep 1s
Down
Sleep 100ms
Down
Sleep 100ms
Enter
Sleep 1s

# Final pause
Sleep 1s
