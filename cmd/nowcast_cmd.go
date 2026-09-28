package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/nowcast"
	"github.com/mwirges/wx/internal/output"
)

func nowcastCommand() *cli.Command {
	return &cli.Command{
		Name:    "nowcast",
		Aliases: []string{"precip", "rain", "qpf"},
		Usage:   "display quantitative precipitation nowcast, rain timeline, and onset countdown",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank to use config/auto-detect",
			},
			&cli.StringFlag{
				Name:    "units",
				Aliases: []string{"u"},
				Usage:   "units: imperial or metric",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "force JSON output",
			},
		},
		Action: runNowcast,
	}
}

func runNowcast(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}

	forceJSON := c.Bool("json")

	var locParts []string
	args := c.Args().Slice()
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--location" || arg == "-l":
			if i+1 < len(args) {
				locParts = append(locParts, args[i+1])
				i++
			}
		case strings.HasPrefix(arg, "--location="):
			locParts = append(locParts, strings.TrimPrefix(arg, "--location="))
		case arg == "--json" || arg == "-j":
			forceJSON = true
		case strings.HasPrefix(arg, "--units=") || arg == "--units" || arg == "-u":
			// Handled by c.String("units")
		case !strings.HasPrefix(arg, "-"):
			locParts = append(locParts, arg)
		}
	}

	ch, err := cache.New()
	if err != nil {
		ch = cache.NewNoOp()
	}

	locInput := c.String("location")
	if locInput == "" && len(locParts) > 0 {
		locInput = strings.Join(locParts, " ")
	}
	if locInput == "" {
		locInput = cfg.DefaultLocation
	}
	resolvedInput := cfg.ResolveLocation(locInput)

	loc, err := location.Resolve(ctx, resolvedInput, ch)
	if err != nil {
		return err
	}

	if loc.DisplayName != "" {
		if cfg.AddRecent(loc.DisplayName) {
			if cfgPath, pathErr := config.Path(); pathErr == nil {
				_ = config.Save(cfgPath, cfg)
			}
		}
	}

	units := c.String("units")
	if units == "" {
		units = cfg.Units
	}
	if units == "" {
		units = "imperial"
	}

	nc, err := nowcast.Fetch(ctx, loc.Lat, loc.Lon, loc.DisplayName, ch)
	if err != nil {
		return fmt.Errorf("fetching nowcast: %w", err)
	}

	payload := &models.NowcastPayload{
		Location: loc.DisplayName,
		Nowcast:  nc,
	}

	return output.RenderNowcast(payload, output.NowcastOptions{
		ForceJSON: forceJSON,
		Units:     units,
	})
}
