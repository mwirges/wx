package cmd

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/monitor"
	"github.com/mwirges/wx/internal/provider"
	"github.com/mwirges/wx/internal/radar"
)

func monitorCommand() *cli.Command {
	return &cli.Command{
		Name:  "monitor",
		Usage: "full-screen live weather monitor with optional radar",
		Description: "Displays current conditions, alerts, and a scrollable forecast in a full-screen TUI.\n" +
			"Weather data refreshes automatically in the background. Press R to toggle the radar panel.\n" +
			"Press l to switch locations, r to refresh now, and q to quit.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank to use config/auto-detect",
			},
			&cli.StringFlag{
				Name:    "units",
				Aliases: []string{"u"},
				Value:   "imperial",
				Usage:   "temperature/wind units: imperial or metric",
			},
			&cli.BoolFlag{
				Name:  "no-cache",
				Usage: "bypass the local cache",
			},
			&cli.DurationFlag{
				Name:  "interval",
				Value: 15 * time.Minute,
				Usage: "background weather refresh interval (e.g. 5m, 1h)",
			},
		},
		Action: monitorAction,
	}
}

func monitorAction(c *cli.Context) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("monitor mode requires a TTY")
	}

	cfg, _ := config.Load()

	var (
		ch  *cache.Cache
		err error
	)
	if c.Bool("no-cache") {
		ch = cache.NewNoOp()
	} else {
		ch, err = cache.New()
		if err != nil {
			ch = cache.NewNoOp()
		}
	}

	locInput := c.String("location")
	if locInput == "" {
		locInput = cfg.DefaultLocation
	}

	units := "imperial"
	if cfg.Units != "" {
		units = cfg.Units
	}
	if c.IsSet("units") {
		units = c.String("units")
	}

	ctx := c.Context
	loc, err := location.Resolve(ctx, locInput, ch)
	if err != nil {
		return err
	}

	weatherProv, err := provider.ForLocation(loc)
	if err != nil {
		return err
	}

	// Radar provider is optional — non-US locations won't have one.
	var radarProv radar.Provider
	if rp, rerr := radar.ForLocation(loc); rerr == nil {
		radarProv = rp
	}

	mcfg := monitor.MonitorConfig{
		WeatherProv:     weatherProv,
		RadarProv:       radarProv,
		Cache:           ch,
		Imperial:        units != "metric",
		RefreshInterval: c.Duration("interval"),
	}

	m := monitor.New(mcfg, loc)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
