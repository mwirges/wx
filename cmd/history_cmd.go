package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/history"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/output"
)

func historyCommand() *cli.Command {
	return &cli.Command{
		Name:    "history",
		Aliases: []string{"hist"},
		Usage:   "display historical weather observations and trends",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank to use config/auto-detect",
			},
			&cli.IntFlag{
				Name:    "days",
				Aliases: []string{"d"},
				Value:   7,
				Usage:   "number of past days to display (1–90, default: 7)",
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
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "force JSON output",
			},
		},
		Action: runHistory,
	}
}

func runHistory(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Load user config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		cfg = &config.Config{}
	}

	days := c.Int("days")
	if days <= 0 {
		days = 7
	}

	units := c.String("units")
	if !c.IsSet("units") && cfg.Units != "" {
		units = cfg.Units
	}

	forceJSON := c.Bool("json")
	noCache := c.Bool("no-cache")

	// Parse arguments in case flags were passed after positional location argument
	var locParts []string
	args := c.Args().Slice()
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--days" || arg == "-d":
			if i+1 < len(args) {
				if v, err := strconv.Atoi(args[i+1]); err == nil && v > 0 {
					days = v
				}
				i++
			}
		case strings.HasPrefix(arg, "--days="):
			if v, err := strconv.Atoi(strings.TrimPrefix(arg, "--days=")); err == nil && v > 0 {
				days = v
			}
		case arg == "--units" || arg == "-u":
			if i+1 < len(args) {
				units = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--units="):
			units = strings.TrimPrefix(arg, "--units=")
		case arg == "--json" || arg == "-j":
			forceJSON = true
		case arg == "--no-cache":
			noCache = true
		case !strings.HasPrefix(arg, "-"):
			locParts = append(locParts, arg)
		}
	}

	// Build cache
	var ch *cache.Cache
	if noCache {
		ch = cache.NewNoOp()
	} else {
		ch, err = cache.New()
		if err != nil {
			return fmt.Errorf("cache init: %w", err)
		}
	}

	// Location resolution
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

	// Track in recents
	if loc.DisplayName != "" {
		if cfg.AddRecent(loc.DisplayName) {
			if cfgPath, pathErr := config.Path(); pathErr == nil {
				_ = config.Save(cfgPath, cfg)
			}
		}
	}

	hist, err := history.Fetch(ctx, loc.Lat, loc.Lon, days, ch)
	if err != nil {
		return fmt.Errorf("fetch history: %w", err)
	}

	if loc.DisplayName != "" {
		hist.Location = loc.DisplayName
	} else if loc.City != "" {
		if loc.State != "" {
			hist.Location = fmt.Sprintf("%s, %s", loc.City, loc.State)
		} else {
			hist.Location = loc.City
		}
	}

	return output.RenderHistory(hist, output.HistoryOptions{
		ForceJSON: forceJSON,
		Units:     units,
	})
}
