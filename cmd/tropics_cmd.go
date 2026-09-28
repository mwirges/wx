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
	"github.com/mwirges/wx/internal/output"
	"github.com/mwirges/wx/internal/tropics"
)

func tropicsCommand() *cli.Command {
	return &cli.Command{
		Name:    "tropics",
		Aliases: []string{"nhc", "hurricane", "cyclone", "tropical"},
		Usage:   "display active NOAA National Hurricane Center tropical cyclones, intensity, and invest disturbances",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "optional reference location for storm distance & proximity calculations",
			},
			&cli.StringFlag{
				Name:    "storm",
				Aliases: []string{"s"},
				Usage:   "filter by storm name or ID (e.g. 'Polo', 'Fay', 'al062026')",
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
			&cli.BoolFlag{
				Name:  "no-cache",
				Usage: "bypass the local cache",
			},
			&cli.BoolFlag{
				Name:  "force-pretty",
				Usage: "force pretty TTY output even when piped",
			},
		},
		Action: runTropics,
	}
}

func runTropics(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}

	locInput := c.String("location")
	if locInput == "" && c.NArg() > 0 {
		locInput = c.Args().First()
	}
	if locInput == "" {
		locInput = cfg.DefaultLocation
	}

	units := c.String("units")
	if units == "" {
		units = cfg.Units
	}
	if units == "" {
		units = "imperial"
	}

	noCache := c.Bool("no-cache")
	var diskCache *cache.Cache
	if !noCache {
		diskCache, _ = cache.New()
	}

	var resolvedLoc *location.Location
	if strings.TrimSpace(locInput) != "" {
		var cacher location.Cacher
		if diskCache != nil {
			cacher = diskCache
		} else {
			cacher = cache.NewNoOp()
		}
		if resolved, err := location.Resolve(ctx, locInput, cacher); err == nil {
			resolvedLoc = &resolved
		}
	}

	cacheKey := "tropics:global"
	if resolvedLoc != nil {
		cacheKey = fmt.Sprintf("tropics:%.4f,%.4f", resolvedLoc.Lat, resolvedLoc.Lon)
	}

	var report *models.TropicsReport
	if diskCache != nil {
		var cached models.TropicsReport
		if diskCache.Get(cacheKey, &cached) {
			report = &cached
		}
	}

	if report == nil {
		client := tropics.NewClient(nil)
		var err error
		report, err = client.FetchTropics(ctx, resolvedLoc)
		if err != nil {
			return fmt.Errorf("tropics: %w", err)
		}

		if diskCache != nil {
			_ = diskCache.Set(cacheKey, report, 5*time.Minute)
		}
	}

	// Filter by storm if requested
	filterStorm := strings.TrimSpace(c.String("storm"))
	if filterStorm != "" {
		var matched []models.TropicalStorm
		for _, s := range report.Storms {
			if strings.EqualFold(s.Name, filterStorm) || strings.EqualFold(s.ID, filterStorm) || strings.EqualFold(s.BinNumber, filterStorm) {
				matched = append(matched, s)
			}
		}
		if len(matched) == 0 {
			return fmt.Errorf("no active storm matching '%s' found", filterStorm)
		}
		report.Storms = matched
		report.TotalActive = len(matched)
	}

	locLabel := ""
	if resolvedLoc != nil {
		locLabel = resolvedLoc.DisplayName
	}

	payload := &models.TropicsPayload{
		Location: locLabel,
		Tropics:  report,
	}

	opts := output.TropicsOptions{
		ForceJSON:   c.Bool("json"),
		ForcePretty: c.Bool("force-pretty"),
		Units:       units,
	}

	return output.RenderTropics(payload, opts)
}
