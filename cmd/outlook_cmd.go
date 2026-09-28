package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/cpc"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/output"
)

func outlookCommand() *cli.Command {
	return &cli.Command{
		Name:    "outlook",
		Aliases: []string{"cpc", "pattern"},
		Usage:   "long-range NOAA Climate Prediction Center outlooks & pattern shift heads-up (6-10d, 8-14d, drought)",
		Description: "Fetches official NOAA Climate Prediction Center (CPC) probabilistic outlooks for 6–10\n" +
			"and 8–14 day horizons, analyzes synoptic atmospheric regime shifts (temperature and\n" +
			"precipitation pattern changes), and checks drought assessment trends.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank for auto-detect",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "output outlook data as structured JSON",
			},
			&cli.BoolFlag{
				Name:  "no-cache",
				Usage: "bypass local cache and fetch fresh data from NOAA",
			},
		},
		Action: outlookAction,
	}
}

func outlookAction(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

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
	loc, err := location.Resolve(ctx, resolvedInput, ch)
	if err != nil {
		return err
	}

	if loc.CountryCode != "US" {
		return fmt.Errorf("NOAA CPC outlooks are only available for US locations (detected: %s)", loc.CountryCode)
	}

	if loc.DisplayName != "" {
		if cfg.AddRecent(loc.DisplayName) {
			if cfgPath, pathErr := config.Path(); pathErr == nil {
				_ = config.Save(cfgPath, cfg)
			}
		}
	}

	client := cpc.NewClient()
	payload, err := client.FetchOutlooks(ctx, loc, ch)
	if err != nil {
		return fmt.Errorf("fetching CPC outlooks: %w", err)
	}

	opts := output.CPCOptions{
		ForceJSON: c.Bool("json"),
	}

	return output.RenderCPC(payload, opts)
}
