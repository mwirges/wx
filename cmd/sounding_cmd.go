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
	"github.com/mwirges/wx/internal/sounding"
)

func soundingCommand() *cli.Command {
	return &cli.Command{
		Name:    "sounding",
		Aliases: []string{"cape", "instability"},
		Usage:   "display upper-air atmospheric sounding, convective instability (CAPE/CIN), and vertical wind shear",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank to use config/auto-detect",
			},
			&cli.StringFlag{
				Name:    "station",
				Aliases: []string{"s"},
				Usage:   "force specific NOAA upper-air radiosonde station (e.g. OUN, ILX, DVN, ILN, DTX)",
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
		Action: runSounding,
	}
}

func runSounding(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}

	forceJSON := c.Bool("json")
	forceStation := c.String("station")

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
		case arg == "--station" || arg == "-s":
			if i+1 < len(args) {
				forceStation = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--station="):
			forceStation = strings.TrimPrefix(arg, "--station=")
		case arg == "--json" || arg == "-j":
			forceJSON = true
		case arg == "--force-pretty":
			// flag
		case strings.HasPrefix(arg, "-"):
			// skip unknown flags
		default:
			locParts = append(locParts, arg)
		}
	}

	locInput := c.String("location")
	if len(locParts) > 0 {
		locInput = strings.Join(locParts, " ")
	}
	if locInput == "" && forceStation == "" {
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
	for _, a := range args {
		if a == "--no-cache" {
			noCache = true
			break
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

	var lat, lon float64
	var displayName string

	if locInput != "" || forceStation == "" {
		loc, err := location.Resolve(ctx, locInput, ch)
		if err != nil {
			return fmt.Errorf("resolve location: %w", err)
		}
		lat = loc.Lat
		lon = loc.Lon
		displayName = loc.DisplayName
	}

	// If station forced and location is empty, lookup station lat/lon
	if forceStation != "" && locInput == "" {
		stnID := strings.ToUpper(strings.TrimSpace(forceStation))
		if s, ok := sounding.FindStationByID(stnID); ok {
			lat = s.Lat
			lon = s.Lon
			displayName = fmt.Sprintf("%s, %s", s.Name, s.State)
		}
	}

	report, err := sounding.Fetch(ctx, lat, lon, displayName, forceStation, ch)
	if err != nil {
		return fmt.Errorf("sounding fetch: %w", err)
	}

	payload := &models.SoundingPayload{
		Location: report.Location,
		Sounding: report,
	}

	return output.RenderSounding(payload, output.SoundingOptions{
		ForceJSON:   forceJSON,
		ForcePretty: c.Bool("force-pretty"),
		Units:       units,
	})
}
