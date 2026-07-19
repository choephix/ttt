package app

import (
	"strconv"
	"strings"

	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
)

const (
	settingsTabID       = "settings"
	settingsControlCols = 42
)

type settingKind int

const (
	settingBool settingKind = iota
	settingInt
	settingString
	settingEnum
)

type settingField struct {
	Label   string
	Kind    settingKind
	Restart bool
	Options func() []widgets.SelectItem
	// Min is the smallest accepted value for settingInt. Fields whose json tag
	// carries omitempty must set Min >= 1, since a stored 0 would be dropped on
	// save and silently revert to the default on the next load.
	Min int

	GetBool func(*config.Settings) bool
	SetBool func(*config.Settings, bool)

	GetString func(*config.Settings) string
	SetString func(*config.Settings, string)

	GetInt func(*config.Settings) int
	SetInt func(*config.Settings, int)
}

type settingsCategory struct {
	Title  string
	Fields []settingField
}

func boolPtr(b bool) *bool { return &b }

// LSP servers and the formatters map are deliberately absent: both are
// structured config that a form handles badly, and stay JSON-only.
func settingsCategories() []settingsCategory {
	return []settingsCategory{
		{Title: "Editor", Fields: []settingField{
			{Label: "Tab size", Kind: settingInt, Min: 1,
				GetInt: func(s *config.Settings) int { return s.Editor.TabSize },
				SetInt: func(s *config.Settings, v int) { s.Editor.TabSize = v }},
			{Label: "Insert spaces", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Editor.InsertSpaces },
				SetBool: func(s *config.Settings, v bool) { s.Editor.InsertSpaces = v }},
			{Label: "Word wrap", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Editor.WordWrap },
				SetBool: func(s *config.Settings, v bool) { s.Editor.WordWrap = v }},
			{Label: "Line numbers", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Editor.LineNumbers },
				SetBool: func(s *config.Settings, v bool) { s.Editor.LineNumbers = v }},
			{Label: "Auto dedent", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Editor.IsAutoDedentEnabled() },
				SetBool: func(s *config.Settings, v bool) { s.Editor.AutoDedent = boolPtr(v) }},
			{Label: "Insert final newline", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Editor.InsertFinalNewline },
				SetBool: func(s *config.Settings, v bool) { s.Editor.InsertFinalNewline = v }},
			{Label: "Trim trailing whitespace", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Editor.TrimTrailingWhitespace },
				SetBool: func(s *config.Settings, v bool) { s.Editor.TrimTrailingWhitespace = v }},
			{Label: "Format on save", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Editor.FormatOnSave },
				SetBool: func(s *config.Settings, v bool) { s.Editor.FormatOnSave = v }},
			{Label: "Focus editor on open", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Editor.FocusOnOpen },
				SetBool: func(s *config.Settings, v bool) { s.Editor.FocusOnOpen = v }},
		}},
		{Title: "Appearance", Fields: []settingField{
			{Label: "Theme", Kind: settingEnum, Options: themeItems,
				GetString: func(s *config.Settings) string { return s.Theme },
				SetString: func(s *config.Settings, v string) { s.Theme = v }},
			{Label: "Border style", Kind: settingEnum, Options: borderStyleItems,
				GetString: func(s *config.Settings) string { return s.Editor.BorderStyle },
				SetString: func(s *config.Settings, v string) { s.Editor.BorderStyle = v }},
			{Label: "Gutter style", Kind: settingEnum, Options: gutterStyleItems,
				GetString: func(s *config.Settings) string { return s.Editor.GutterStyle },
				SetString: func(s *config.Settings, v string) { s.Editor.GutterStyle = v }},
			{Label: "Cursor style", Kind: settingEnum, Options: cursorStyleItems,
				GetString: func(s *config.Settings) string { return s.Editor.CursorStyle },
				SetString: func(s *config.Settings, v string) { s.Editor.CursorStyle = v }},
			{Label: "Syntax highlight", Kind: settingBool, Restart: true,
				GetBool: func(s *config.Settings) bool { return s.Editor.IsSyntaxHighlightEnabled() },
				SetBool: func(s *config.Settings, v bool) { s.Editor.SyntaxHighlight = boolPtr(v) }},
			{Label: "Bracket pair colors", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Editor.BracketPairColorization },
				SetBool: func(s *config.Settings, v bool) { s.Editor.BracketPairColorization = v }},
			{Label: "Git gutter", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Editor.IsGitGutterEnabled() },
				SetBool: func(s *config.Settings, v bool) { s.Editor.GitGutter = boolPtr(v) }},
			{Label: "Markdown wrap width", Kind: settingInt, Min: 1,
				GetInt: func(s *config.Settings) int { return s.Markdown.WrapWidth },
				SetInt: func(s *config.Settings, v int) { s.Markdown.WrapWidth = v }},
		}},
		{Title: "Completion", Fields: []settingField{
			{Label: "Enable completion", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Autocomplete.Enabled },
				SetBool: func(s *config.Settings, v bool) { s.Autocomplete.Enabled = v }},
			{Label: "Suggest as you type", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Autocomplete.AutoSuggest },
				SetBool: func(s *config.Settings, v bool) { s.Autocomplete.AutoSuggest = v }},
			{Label: "Signature help", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Autocomplete.SignatureHelp },
				SetBool: func(s *config.Settings, v bool) { s.Autocomplete.SignatureHelp = v }},
			{Label: "Debounce (ms)", Kind: settingInt,
				GetInt: func(s *config.Settings) int { return s.Autocomplete.Debounce },
				SetInt: func(s *config.Settings, v int) { s.Autocomplete.Debounce = v }},
		}},
		{Title: "Explorer", Fields: []settingField{
			{Label: "Show hidden files", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Explorer.ShowHidden },
				SetBool: func(s *config.Settings, v bool) { s.Explorer.ShowHidden = v }},
			{Label: "Show git-ignored files", Kind: settingBool,
				GetBool: func(s *config.Settings) bool { return s.Explorer.ShowGitIgnored },
				SetBool: func(s *config.Settings, v bool) { s.Explorer.ShowGitIgnored = v }},
		}},
		{Title: "Terminal", Fields: []settingField{
			{Label: "Shell", Kind: settingString, Restart: true,
				GetString: func(s *config.Settings) string { return s.Terminal.Shell },
				SetString: func(s *config.Settings, v string) { s.Terminal.Shell = v }},
			{Label: "Scrollback lines", Kind: settingInt, Restart: true, Min: 1,
				GetInt: func(s *config.Settings) int { return s.Terminal.Scrollback },
				SetInt: func(s *config.Settings, v int) { s.Terminal.Scrollback = v }},
		}},
		{Title: "Search", Fields: []settingField{
			{Label: "Debounce (ms)", Kind: settingInt,
				GetInt: func(s *config.Settings) int { return s.Search.Debounce },
				SetInt: func(s *config.Settings, v int) { s.Search.Debounce = v }},
		}},
		{Title: "Advanced", Fields: []settingField{
			{Label: "Enable plugins", Kind: settingBool, Restart: true,
				GetBool: func(s *config.Settings) bool { return s.Plugins.IsEnabled() },
				SetBool: func(s *config.Settings, v bool) { s.Plugins.Enabled = boolPtr(v) }},
			{Label: "Debug mode", Kind: settingBool, Restart: true,
				GetBool: func(s *config.Settings) bool { return s.DebugMode },
				SetBool: func(s *config.Settings, v bool) { s.DebugMode = v }},
		}},
	}
}

func themeItems() []widgets.SelectItem {
	names := config.ListThemes()
	items := make([]widgets.SelectItem, 0, len(names)+1)
	items = append(items, widgets.SelectItem{ID: "", Label: "Default"})
	for _, n := range names {
		items = append(items, widgets.SelectItem{ID: n, Label: n})
	}
	return items
}

// Mirrors term.ParseCursorStyle. "" means unset and behaves as a blinking bar,
// so it is offered as "Default" rather than duplicated as "Bar".
func cursorStyleItems() []widgets.SelectItem {
	return []widgets.SelectItem{
		{ID: "", Label: "Default"},
		{ID: "bar", Label: "Bar (blinking)"},
		{ID: "steadyBar", Label: "Bar (steady)"},
		{ID: "block", Label: "Block (blinking)"},
		{ID: "steadyBlock", Label: "Block (steady)"},
		{ID: "underline", Label: "Underline (blinking)"},
		{ID: "steadyUnderline", Label: "Underline (steady)"},
	}
}

type settingsView struct {
	app     *App
	working config.Settings
	adapter *ui.WidgetAdapter
	status  *widgets.LabelWidget
	inputs  []func() string
	selects []*widgets.SelectWidget
}

func (v *settingsView) closeSelectsExcept(keep *widgets.SelectWidget) {
	for _, s := range v.selects {
		if s != keep {
			s.ClosePopup()
		}
	}
}

func (a *App) ShowSettings() {
	// Reopening while the tab is already open must not discard pending edits.
	if v := a.settingsView; v != nil {
		a.EditorGroup.OpenPluginTab(settingsTabID, "Settings", v.adapter)
		a.FocusEditor()
		v.adapter.SetFocused(true)
		return
	}

	v := &settingsView{app: a, working: *a.Settings}
	a.settingsView = v

	categories := settingsCategories()
	tabItems := make([]widgets.TabItem, 0, len(categories))
	panes := make([]widgets.Widget, 0, len(categories))
	for _, cat := range categories {
		tabItems = append(tabItems, widgets.TabItem{ID: cat.Title, Label: cat.Title})
		panes = append(panes, v.buildPane(cat))
	}
	tabs := widgets.NewTabsWidget(widgets.TabsConfig{Items: tabItems, Align: "left"})
	tabbed := widgets.NewTabbedWidget(tabs, panes)
	tabbed.Fill = true

	v.status = widgets.NewLabelWidget(widgets.LabelConfig{Style: term.StyleMuted})
	applyBtn := widgets.NewButtonWidget(widgets.ButtonConfig{Label: "Apply", OnClick: v.apply})
	resetBtn := widgets.NewButtonWidget(widgets.ButtonConfig{Label: "Reset", OnClick: v.reset})
	jsonBtn := widgets.NewButtonWidget(widgets.ButtonConfig{Label: "Edit JSON", OnClick: func() {
		a.Reg.Execute("settings.open")
	}})
	buttons := widgets.NewHStackWidget(v.status, applyBtn, resetBtn, jsonBtn)
	buttons.Gap = 1
	buttons.FixedHeight = 1
	buttons.Box.PaddingLeft = 1
	buttons.Box.PaddingRight = 1

	root := widgets.NewVStackWidget(
		tabbed,
		widgets.NewDividerWidget(widgets.DividerConfig{}),
		buttons,
	)

	// NewWidgetAdapter wires TabbedWidget.OnChange to rebuild focus on tab change.
	v.adapter = ui.NewWidgetAdapter(root)
	v.adapter.EnableScrollIntoView()

	a.EditorGroup.OpenPluginTab(settingsTabID, "Settings", v.adapter)
	a.FocusEditor()
	v.adapter.SetFocused(true)
}

func (v *settingsView) buildPane(cat settingsCategory) widgets.Widget {
	rows := make([]widgets.Widget, 0, len(cat.Fields))
	for _, f := range cat.Fields {
		rows = append(rows, v.buildRow(cat.Title, f))
	}
	stack := widgets.NewVStackWidget(rows...)
	stack.Gap = 1 // exactly one blank row between every field
	stack.MeasureGrow = true
	stack.Box.PaddingLeft = 1
	stack.Box.PaddingTop = 1

	// The divider sits inside the pane so it reads as the tab strip's bottom
	// border, and stays put while the fields scroll under it.
	return widgets.NewVStackWidget(
		widgets.NewDividerWidget(widgets.DividerConfig{}),
		widgets.NewScrollViewWidget(stack),
	)
}

// Rows are a single column: booleans are a self-labelled bordered checkbox,
// everything else is a label with its bordered control underneath. Checkboxes
// stack without a gap since their borders already separate them.
func (v *settingsView) buildRow(category string, f settingField) widgets.Widget {
	label := f.Label
	if f.Restart {
		label += " (restart)"
	}

	if f.Kind == settingBool {
		return widgets.NewCheckboxWidget(widgets.CheckboxConfig{
			Label:    label,
			Bordered: true,
			Checked:  f.GetBool(&v.working),
			OnChange: func(checked bool) { f.SetBool(&v.working, checked) },
		})
	}

	var control widgets.Widget
	if f.Kind == settingEnum {
		control = v.enumControl(f)
	} else {
		control = v.textControl(category, f)
	}

	// The HStack constrains the control to its fixed width; inside a VStack it
	// would otherwise stretch across the whole pane.
	sized := widgets.NewHStackWidget(control)
	sized.FixedHeight = 3

	return widgets.NewVStackWidget(
		widgets.NewLabelWidget(widgets.LabelConfig{Text: label}),
		sized,
	)
}

func (v *settingsView) enumControl(f settingField) widgets.Widget {
	var sel *widgets.SelectWidget
	sel = widgets.NewSelectWidget(widgets.SelectConfig{
		Items:       f.Options(),
		Collapsible: true,
		Bordered:    true,
		OnOpen:      func() { v.closeSelectsExcept(sel) },
		OnSelect: func(id string) {
			f.SetString(&v.working, id)
			sel.SetSelectedID(id)
		},
	})
	v.selects = append(v.selects, sel)
	sel.FixedWidth = settingsControlCols
	sel.SetSelectedID(f.GetString(&v.working))
	return sel
}

// Text and numeric fields are parsed on Apply rather than per keystroke, so a
// half-typed value never reaches the working copy.
func (v *settingsView) textControl(category string, f settingField) widgets.Widget {
	current := ""
	if f.Kind == settingInt {
		current = strconv.Itoa(f.GetInt(&v.working))
	} else {
		current = f.GetString(&v.working)
	}

	inp := widgets.NewInputWidget(widgets.InputConfig{Bordered: true})
	inp.FixedWidth = settingsControlCols
	inp.SetText(current)

	// Returns a description of the offending field, or "" when the value is good.
	v.inputs = append(v.inputs, func() string {
		text := inp.Text()
		if f.Kind == settingString {
			f.SetString(&v.working, text)
			return ""
		}
		n, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || n < f.Min {
			inp.SetText(strconv.Itoa(f.GetInt(&v.working)))
			return category + " → " + f.Label
		}
		f.SetInt(&v.working, n)
		return ""
	})
	return inp
}

func (v *settingsView) apply() {
	// Validate every field before committing, so one bad value does not hide the
	// rest and the user fixes them in a single pass.
	var bad []string
	for _, commit := range v.inputs {
		if msg := commit(); msg != "" {
			bad = append(bad, msg)
		}
	}
	if len(bad) > 0 {
		v.setStatus("Invalid value for " + strings.Join(bad, ", "))
		return
	}
	*v.app.Settings = v.working
	v.app.SaveAndApplySettings()
	v.working = *v.app.Settings
	v.setStatus("Settings applied")
}

func (v *settingsView) reset() {
	v.app.settingsView = nil
	v.app.ShowSettings()
	if nv := v.app.settingsView; nv != nil {
		nv.setStatus("Changes discarded")
	}
}

func (v *settingsView) setStatus(msg string) {
	if v.status != nil {
		v.status.Config.Text = msg
	}
}

func (a *App) ApplySettingsView() {
	if a.settingsView == nil {
		a.StatusNotify("No settings editor open")
		return
	}
	a.settingsView.apply()
}

func (a *App) ResetSettingsView() {
	if a.settingsView == nil {
		a.StatusNotify("No settings editor open")
		return
	}
	a.settingsView.reset()
}

func (a *App) cleanupSettingsTab(id string) {
	if id == settingsTabID {
		a.settingsView = nil
	}
}
