package ui

import (
	"github.com/mwirges/wx/linux/internal/cli"
	"github.com/mwirges/wx/linux/internal/prefs"
)

func newBootCLI(binary string) *cli.WxCLI { return cli.New(binary) }

func newBootPrefs(path string) *prefs.Prefs { return prefs.New(path) }
