package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/output"
	"github.com/mwirges/wx/internal/provider"
	_ "github.com/mwirges/wx/internal/provider/nws" // register NWS provider
)

// NewApp returns the configured urfave/cli application.
func NewApp() *cli.App {
	cfgPath, _ := config.Path()

	app := &cli.App{
		Name:    "wx",
		Usage:   "current weather conditions and forecasts",
		Version: "1.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank to use config/auto-detect",
			},
			&cli.BoolFlag{
				Name:    "forecast",
				Aliases: []string{"f"},
				Usage:   "show 7-day forecast",
			},
			&cli.BoolFlag{
				Name:    "alerts",
				Aliases: []string{"a"},
				Usage:   "show active weather alerts",
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
		Action:   action,
		Commands: []*cli.Command{configCommand()},
		ExitErrHandler: func(c *cli.Context, err error) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		},
		Description: fmt.Sprintf(
			"Weather data from weather.gov (US). Config file: %s", cfgPath,
		),
	}
	return app
}

func action(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load user config (missing file is not an error)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		cfg = &config.Config{}
	}

	// Build cache
	var ch *cache.Cache
	if c.Bool("no-cache") {
		ch = cache.NewNoOp()
	} else {
		ch, err = cache.New()
		if err != nil {
			return fmt.Errorf("cache init: %w", err)
		}
	}

	// Location precedence: --location flag > config default_location > IP auto-detect
	locInput := c.String("location")
	if locInput == "" {
		locInput = cfg.DefaultLocation
	}

	// Resolve location
	loc, err := location.Resolve(ctx, locInput, ch)
	if err != nil {
		return err
	}

	// Select provider
	prov, err := provider.ForLocation(loc)
	if err != nil {
		return err
	}

	showForecast := c.Bool("forecast")
	showAlerts := c.Bool("alerts")

	// Units precedence: --units flag > config units > "imperial"
	units := "imperial"
	if cfg.Units != "" {
		units = cfg.Units
	}
	if c.IsSet("units") {
		units = c.String("units")
	}

	// Fetch concurrently
	var (
		cond    *models.CurrentConditions
		fc      *models.Forecast
		alerts  []models.Alert
		condErr error
		fcErr   error
		alErr   error
		wg      sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		cond, condErr = prov.CurrentConditions(ctx, loc, ch)
	}()

	if showForecast {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fc, fcErr = prov.Forecast(ctx, loc, false, ch)
		}()
	}

	if showAlerts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			alerts, alErr = prov.Alerts(ctx, loc, ch)
		}()
	}

	wg.Wait()

	if condErr != nil {
		return fmt.Errorf("weather: %w", condErr)
	}
	if fcErr != nil {
		fmt.Fprintf(os.Stderr, "warning: forecast unavailable: %v\n", fcErr)
	}
	if alErr != nil {
		fmt.Fprintf(os.Stderr, "warning: alerts unavailable: %v\n", alErr)
	}

	return output.Render(output.RenderData{
		Conditions: cond,
		Forecast:   fc,
		Alerts:     alerts,
	}, output.RenderOptions{
		ForceJSON:    c.Bool("json"),
		Units:        units,
		ShowForecast: showForecast,
		ShowAlerts:   showAlerts,
	})
}
