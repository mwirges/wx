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
	"github.com/mwirges/wx/internal/spc"
)

func outlookCommand() *cli.Command {
	return &cli.Command{
		Name:    "outlook",
		Aliases: []string{"cpc", "pattern"},
		Usage:   "long-range NOAA Climate Prediction Center & SPC convective outlooks",
		Description: "Fetches official NOAA Climate Prediction Center (CPC) probabilistic outlooks for 6–10\n" +
			"and 8–14 day horizons, and Storm Prediction Center (SPC) convective severe storm outlooks,\n" +
			"mesoscale discussions, and severe watches.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank for auto-detect",
			},
			&cli.BoolFlag{
				Name:    "spc",
				Aliases: []string{"s"},
				Usage:   "fetch official Storm Prediction Center (SPC) Day 1-3 convective severe weather outlooks & MCDs",
			},
			&cli.BoolFlag{
				Name:  "cpc",
				Usage: "force NOAA CPC 6-10 / 8-14 day pattern shift outlooks",
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "render both CPC long-range outlooks and SPC severe convective outlooks",
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

func spcCommand() *cli.Command {
	return &cli.Command{
		Name:    "spc",
		Aliases: []string{"convective", "mcd", "stormoutlook"},
		Usage:   "official Storm Prediction Center (SPC) convective outlooks, mesoscale discussions, and severe watches",
		Description: "Fetches official NOAA Storm Prediction Center (SPC) Day 1, 2, and 3 categorical convective risk\n" +
			"areas (TSTM, MRGL, SLGT, ENH, MDT, HIGH), probabilistic tornado/hail/wind threats,\n" +
			"active Mesoscale Discussions (MCD), and active severe watches.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank for auto-detect",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "output SPC data as structured JSON",
			},
			&cli.BoolFlag{
				Name:  "no-cache",
				Usage: "bypass local cache and fetch fresh data from NOAA",
			},
		},
		Action: func(c *cli.Context) error {
			return runSPC(c)
		},
	}
}

func outlookAction(c *cli.Context) error {
	if c.Bool("spc") && !c.Bool("all") {
		return runSPC(c)
	}
	if c.Bool("all") {
		if err := runSPC(c); err != nil {
			return err
		}
		if !c.Bool("json") {
			fmt.Println()
		}
		return runCPC(c)
	}
	return runCPC(c)
}

func runCPC(c *cli.Context) error {
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

func runSPC(c *cli.Context) error {
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
		return fmt.Errorf("NOAA SPC outlooks are only available for US locations (detected: %s)", loc.CountryCode)
	}

	if loc.DisplayName != "" {
		if cfg.AddRecent(loc.DisplayName) {
			if cfgPath, pathErr := config.Path(); pathErr == nil {
				_ = config.Save(cfgPath, cfg)
			}
		}
	}

	payload, err := spc.FetchSPCOutlooks(ctx, loc, ch)
	if err != nil {
		return fmt.Errorf("fetching SPC outlooks: %w", err)
	}

	opts := output.SPCOptions{
		ForceJSON: c.Bool("json"),
	}

	return output.RenderSPC(payload, opts)
}
