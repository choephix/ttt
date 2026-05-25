# Cleanup Opportunities

## 1. `surface.DrawText()` helper on RenderSurface

11+ instances of the same "iterate runes, SetCell one-by-one" loop across widgets.

**Files:** confirmdialog_widget.go, inputdialog_widget.go, contextmenu_widget.go, autocomplete_widget.go, palette_widget.go, findbar_widget.go, replacebar_widget.go

**Before:**
```go
for _, ch := range label {
    if x < maxX {
        surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
        x++
    }
}
```

**After:**
```go
surface.DrawText(x, y, label, maxW, style)
```

## 2. `surface.ClearRect()` helper on RenderSurface

4-5 nested loops clearing rectangular areas with spaces. Same pattern as `DrawBorder`.

**Files:** inputdialog_widget.go, confirmdialog_widget.go, palette_widget.go

**Before:**
```go
for y := boxY; y < boxY+boxH; y++ {
    for x := boxX; x < boxX+boxW; x++ {
        surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
    }
}
```

**After:**
```go
surface.ClearRect(boxX, boxY, boxW, boxH, style)
```

## 3. FindBar / ReplaceBar code sharing

ReplaceBar is basically FindBar plus a second input and replace buttons. They duplicate:
- Cursor movement (Home/End/Left/Right) — identical logic
- Rune insert/delete with cursor position tracking — identical logic
- Circular match navigation `(Current +/- 1 + len) % len` — 10+ instances

Could extract shared text-editing helpers or a common base.

## 4. Dialog boilerplate in commands.go

Every dialog repeats: `dialog.Borders = app.borders` + `OnDismiss = func() { app.DismissDialog() }` + `app.ShowDialog(dialog)`.

**InputDialog pattern (7 instances):**
```go
dialog := ui.NewInputDialogWidget(title, initial)
dialog.Borders = app.borders
dialog.OnSubmit = func(value string) { app.DismissDialog(); /* action */ }
dialog.OnDismiss = func() { app.DismissDialog() }
app.ShowDialog(dialog)
```

Could become:
```go
app.ShowInputDialog(title, initial, func(value string) { /* action */ })
```

**ConfirmDialog pattern (2 instances):**
```go
app.ShowConfirmDialog(message, func() { /* on confirm */ })
```

## 5. Palette/Picker reuse in commands.go

`openPalette` helper exists but theme.switch and editor.indentation commands create their own palette widgets with the same boilerplate instead of reusing it. A generic picker helper would deduplicate:

```go
app.ShowPicker(items []command.Command, onSelect func(id string))
```

**Files:** cmd/ttt/commands.go (theme.switch, editor.indentation, workspace.removeFolder)

## 6. Git command registration

git.pull, git.push, git.sync are nearly identical — only the function name and status message change. Could use a helper:

```go
registerGitCommand(reg, app, "git.pull", "Git Pull", git.Pull, "pulled")
```
