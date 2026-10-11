// Command stage writes a Debian package root for the wx desk.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mwirges/wx/linux/packaging"
)

func main() {
	opts := packaging.Options{}
	flag.StringVar(&opts.Icon, "icon", "", "source PNG")
	flag.StringVar(&opts.Templates, "templates", "packaging", "desktop and maintainer scripts")
	flag.StringVar(&opts.Dest, "dest", "", "package root")
	flag.StringVar(&opts.Bins, "bins", "", "directory containing wx and wx-desk")
	flag.StringVar(&opts.Version, "version", "", "package version")
	flag.StringVar(&opts.Arch, "arch", "", "Debian architecture")
	flag.StringVar(&opts.Date, "date", "", "release date, YYYY-MM-DD")
	flag.Parse()
	if err := packaging.Stage(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
