package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconPNG []byte

// AppIcon is the radar mark shared with the Mac app. The desk window and
// the launcher both use it.
func AppIcon() fyne.Resource {
	return fyne.NewStaticResource("wx.png", iconPNG)
}
