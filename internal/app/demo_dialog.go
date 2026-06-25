package app

import (
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
)

func showDemoDialog(app *App) {
	dialog := widgets.NewDialogWidget(50)
	dialog.Title = "Please Confirm"
	dialog.Borders = *app.Borders

	content := widgets.NewParagraphWidget(
		"Are you sure you want to continue? This action cannot be undone and will apply changes to all open files in the workspace.",
	)

	dialog.SetContent(content)
	dialog.Buttons = []widgets.DialogButton{
		{Label: "&No", Handler: func() { app.DismissDialog() }},
		{Label: "&Yes", Handler: func() {
			app.DismissDialog()
			app.StatusNotify("Confirmed!")
		}},
	}
	dialog.OnDismiss = func() { app.DismissDialog() }
	dialog.Build()

	adapter := ui.NewWidgetAdapter(dialog)
	app.ShowDialog(adapter)
}
