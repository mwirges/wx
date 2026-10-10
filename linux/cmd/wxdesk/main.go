// Command wxdesk is the Linux desk shell. Weather stays in the wx CLI.
package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/mwirges/wx/linux/internal/cli"
	"github.com/mwirges/wx/linux/internal/prefs"
	"github.com/mwirges/wx/linux/internal/ui"
)

func main() {
	application := app.NewWithID("com.github.mwirges.wx.desk")
	application.Settings().SetTheme(ui.NewTheme())
	shell := ui.NewShell(application, cli.New(""), prefs.New(""))
	shell.Start(true)
	application.Run()
}
