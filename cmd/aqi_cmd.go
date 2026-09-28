package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/airquality"
	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/output"
)

func aqiCommand() *cli.Command {
	return &cli.Command{
		Name:    "aqi",
		Aliases: []string{"air", "airquality", "smoke"},
		Usage:   "display air quality index (AQI), smoke plume particulates, and EPA health advisories",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank to use config/auto-detect",
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
		Action: runAQI,
	}
}

func runAQI(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}

	forceJSON := c.Bool("json")
	noCache := c.Bool("no-cache")

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
		case arg == "--no-cache":
			noCache = true
		case !strings.HasPrefix(arg, "-"):
			locParts = append(locParts, arg)
		}
	}

	var ch *cache.Cache
	if noCache {
		ch = cache.NewNoOp()
	} else {
		var err error
		ch, err = cache.New()
		if err != nil {
			ch = cache.NewNoOp()
		}
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

	aq, err := airquality.Fetch(ctx, loc.Lat, loc.Lon, ch)
	if err != nil {
		return fmt.Errorf("fetching air quality: %w", err)
	}

	payload := &models.AirQualityPayload{
		Location:  loc.DisplayName,
		Latitude:  loc.Lat,
		Longitude: loc.Lon,
		FetchedAt: time.Now().UTC(),
		AirQuality: aq,
	}

	if aq != nil {
		if aq.AQI != nil {
			payload.HealthAdvisory = models.EPAHealthAdvisory(*aq.AQI)
		}
		if aq.PM25 != nil {
			payload.SmokeAdvisory = models.SmokeAdvisory(*aq.PM25)
		}
	}

	return output.RenderAQI(payload, output.AQIOptions{
		ForceJSON: forceJSON,
	})
}
