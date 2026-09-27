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
			&cli.BoolFlag{
				Name:    "notify",
				Usage:   "enable desktop notifications for new alerts",
			},
			&cli.BoolFlag{
				Name:    "hourly",
				Aliases: []string{"H"},
				Usage:   "start in hourly forecast mode",
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
	if locInput == "" && c.Args().Present() {
		locInput = c.Args().First()
	}
	if locInput == "" {
		locInput = cfg.DefaultLocation
	}
	resolvedInput := cfg.ResolveLocation(locInput)

	ctx := c.Context
	loc, err := location.Resolve(ctx, resolvedInput, ch)
	if err != nil {
		return err
	}

	// Units precedence: CLI flag > per-location config > global config > imperial
	units := cfg.GetEffectiveUnits(resolvedInput, locInput, loc.DisplayName)
	if c.IsSet("units") {
		units = c.String("units")
	}

	if loc.DisplayName != "" {
		if cfg.AddRecent(loc.DisplayName) {
			if cfgPath, pathErr := config.Path(); pathErr == nil {
				_ = config.Save(cfgPath, cfg)
			}
		}
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

	// Notifications precedence: CLI flag > config file > built-in default (false)
	enableNotifications := false
	if cfg.Notifications != nil {
		enableNotifications = *cfg.Notifications
	}
	if c.Bool("notify") {
		enableNotifications = true
	}

	mcfg := monitor.MonitorConfig{
		WeatherProv:         weatherProv,
		RadarProv:           radarProv,
		Cache:               ch,
		Imperial:            units != "metric",
		RefreshInterval:     c.Duration("interval"),
		EnableNotifications: enableNotifications,
		Hourly:              c.Bool("hourly"),
		UserConfig:          cfg,
	}

	m := monitor.New(mcfg, loc)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
