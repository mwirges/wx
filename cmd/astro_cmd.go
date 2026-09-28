package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/astro"
	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/output"
)

func astroCommand() *cli.Command {
	return &cli.Command{
		Name:    "astro",
		Aliases: []string{"ephemeris", "sun", "moon"},
		Usage:   "display solar elevation arc, twilight horizons, golden hour, and lunar ephemeris",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank to use config/auto-detect",
			},
			&cli.StringFlag{
				Name:    "date",
				Aliases: []string{"d"},
				Usage:   "calculation date in YYYY-MM-DD format (default: today)",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "force JSON output",
			},
		},
		Action: runAstro,
	}
}

func runAstro(c *cli.Context) error {
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

	calcTime := time.Now()
	if dateStr := c.String("date"); dateStr != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", dateStr, time.Local); err == nil {
			calcTime = parsed
		} else {
			return fmt.Errorf("invalid date format %q, use YYYY-MM-DD: %w", dateStr, err)
		}
	}

	astronomy, err := astro.Calculate(loc.Lat, loc.Lon, calcTime)
	if err != nil {
		return fmt.Errorf("calculating astronomical ephemeris: %w", err)
	}

	payload := &models.AstroPayload{
		Location:     loc.DisplayName,
		Latitude:     loc.Lat,
		Longitude:    loc.Lon,
		CalculatedAt: calcTime,
		Astronomy:    astronomy,
	}

	return output.RenderAstro(payload, output.AstroOptions{
		ForceJSON: forceJSON,
	})
}
