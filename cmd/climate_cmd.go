package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/climate"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/output"
	"github.com/mwirges/wx/internal/provider"
)

func climateCommand() *cli.Command {
	return &cli.Command{
		Name:    "climate",
		Aliases: []string{"normals", "records"},
		Usage:   "display NOAA 30-year climate normals (1991–2020), daily records, and departure anomalies",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank to use config/auto-detect",
			},
			&cli.StringFlag{
				Name:    "date",
				Aliases: []string{"d"},
				Usage:   "target date in YYYY-MM-DD format (default: today)",
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
		Action: runClimate,
	}
}

func runClimate(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
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
		case arg == "--date" || arg == "-d":
			if i+1 < len(args) {
				i++
			}
		case strings.HasPrefix(arg, "--date="):
			// handled by c.String("date")
		case strings.HasPrefix(arg, "--units=") || arg == "--units" || arg == "-u":
			// handled by c.String("units")
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

	targetDate := time.Now()
	if dateStr := c.String("date"); dateStr != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", dateStr, time.Local); err == nil {
			targetDate = parsed
		} else {
			return fmt.Errorf("invalid date format %q, use YYYY-MM-DD: %w", dateStr, err)
		}
	}

	units := c.String("units")
	if units == "" {
		units = cfg.Units
	}
	if units == "" {
		units = "imperial"
	}

	// Fetch current conditions concurrently to compute departure anomaly if viewing today
	var obsCurrent *models.CurrentConditions
	isToday := targetDate.Format("2006-01-02") == time.Now().Format("2006-01-02")
	if isToday {
		if p, err := provider.ForLocation(loc); err == nil {
			obsCurrent, _ = p.CurrentConditions(ctx, loc, ch)
		}
	}

	report, err := climate.Fetch(ctx, loc.Lat, loc.Lon, loc.DisplayName, obsCurrent, targetDate, ch)
	if err != nil {
		return fmt.Errorf("fetching climate normals: %w", err)
	}

	payload := &models.ClimatePayload{
		Location: loc.DisplayName,
		Climate:  report,
	}

	return output.RenderClimate(payload, output.ClimateOptions{
		ForceJSON: forceJSON,
		Units:     units,
	})
}
