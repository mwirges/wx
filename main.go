package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/cmd"
)

func main() {
	if err := cmd.NewApp().Run(os.Args); err != nil {
		if exitErr, ok := err.(cli.ExitCoder); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
