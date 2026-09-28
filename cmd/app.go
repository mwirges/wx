package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/airquality"
	"github.com/mwirges/wx/internal/astro"
	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/nowcast"
	"github.com/mwirges/wx/internal/output"
	"github.com/mwirges/wx/internal/provider"
	_ "github.com/mwirges/wx/internal/provider/nws"       // register NWS provider
	_ "github.com/mwirges/wx/internal/provider/openmeteo" // register Open-Meteo fallback provider
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
			&cli.StringFlag{
				Name:    "provider",
				Aliases: []string{"p"},
				Usage:   "weather provider to use: nws or openmeteo (default auto-detected)",
			},
			&cli.BoolFlag{
				Name:    "forecast",
				Aliases: []string{"f"},
				Usage:   "show 7-day forecast",
			},
			&cli.BoolFlag{
				Name:    "hourly",
				Aliases: []string{"H"},
				Usage:   "show hourly forecast (implies --forecast)",
			},
			&cli.IntFlag{
				Name:    "hours",
				Value:   24,
				Usage:   "number of hours to show in hourly forecast",
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
				Name:    "short",
				Aliases: []string{"s"},
				Usage:   "print a single-line summary for statuslines/prompts",
			},
			&cli.StringFlag{
				Name:    "template",
				Aliases: []string{"T"},
				Usage:   "format output using a Go template string or @file",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "force JSON output",
			},
			&cli.BoolFlag{
				Name:  "exit-code-on-alerts",
				Usage: "exit with code 2 on active warnings, 1 on active watches/advisories",
			},
		},
		Action:   action,
		Commands: []*cli.Command{configCommand(), locationsCommand(), radarCommand(), monitorCommand(), hourlyCommand(), historyCommand(), chaseCommand(), outlookCommand(), spcCommand(), aqiCommand(), astroCommand(), nowcastCommand()},
		ExitErrHandler: func(c *cli.Context, err error) {
			if err != nil && err.Error() != "" {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		},
		Description: fmt.Sprintf(
			"Weather data from weather.gov (US) and Open-Meteo (global). Config file: %s", cfgPath,
		),
	}
	return app
}

type weatherOpts struct {
	showForecast bool
	showHourly   bool
	showAlerts   bool
}

func action(c *cli.Context) error {
	showHourly := c.Bool("hourly")
	showForecast := c.Bool("forecast") || showHourly
	showAlerts := c.Bool("alerts") || c.Bool("exit-code-on-alerts")

	return runWeather(c, weatherOpts{
		showForecast: showForecast,
		showHourly:   showHourly,
		showAlerts:   showAlerts,
	})
}

func runWeather(c *cli.Context, opts weatherOpts) error {
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

	// Location precedence: --location flag > positional arg (e.g. `wx home`) > config default_location > IP auto-detect
	locInput := c.String("location")
	if locInput == "" && c.Args().Present() {
		locInput = c.Args().First()
	}
	if locInput == "" {
		locInput = cfg.DefaultLocation
	}
	resolvedInput := cfg.ResolveLocation(locInput)

	// Resolve location
	loc, err := location.Resolve(ctx, resolvedInput, ch)
	if err != nil {
		return err
	}

	// Track in recents if resolution gave a display name
	if loc.DisplayName != "" {
		if cfg.AddRecent(loc.DisplayName) {
			if cfgPath, pathErr := config.Path(); pathErr == nil {
				_ = config.Save(cfgPath, cfg)
			}
		}
	}

	// Select provider
	preferredProv := c.String("provider")
	if preferredProv == "" {
		preferredProv = cfg.GetEffectiveProvider(resolvedInput, locInput, loc.DisplayName)
	}

	prov, err := provider.ForLocationWithPreference(loc, preferredProv)
	if err != nil {
		return err
	}

	// Units precedence: --units flag > per-location setting > config units > "imperial"
	units := cfg.GetEffectiveUnits(resolvedInput, locInput, loc.DisplayName)
	if c.IsSet("units") {
		units = c.String("units")
	}

	fetchData := func(p provider.WeatherProvider) (*models.CurrentConditions, *models.Forecast, []models.Alert, error, error, error) {
		var (
			condRes  *models.CurrentConditions
			fcRes    *models.Forecast
			alRes    []models.Alert
			aqRes    *models.AirQuality
			ncRes    *models.Nowcast
			cErr     error
			fErr     error
			aErr     error
			fetchWg  sync.WaitGroup
		)

		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			condRes, cErr = p.CurrentConditions(ctx, loc, ch)
		}()

		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			aqRes, _ = airquality.Fetch(ctx, loc.Lat, loc.Lon, ch)
		}()

		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			ncRes, _ = nowcast.Fetch(ctx, loc.Lat, loc.Lon, loc.DisplayName, ch)
		}()

		if opts.showForecast {
			fetchWg.Add(1)
			go func() {
				defer fetchWg.Done()
				fcRes, fErr = p.Forecast(ctx, loc, opts.showHourly, ch)
			}()
		}

		if opts.showAlerts {
			fetchWg.Add(1)
			go func() {
				defer fetchWg.Done()
				alRes, aErr = p.Alerts(ctx, loc, ch)
			}()
		}

		fetchWg.Wait()
		if condRes != nil {
			if aqRes != nil {
				condRes.AirQuality = aqRes
			}
			if ncRes != nil {
				condRes.Nowcast = ncRes
			}
		}
		return condRes, fcRes, alRes, cErr, fErr, aErr
	}

	cond, fc, alerts, condErr, fcErr, alErr := fetchData(prov)

	// Fallback to alternative provider if primary failed and provider was not explicitly forced by flag
	if condErr != nil && !c.IsSet("provider") {
		if fallbackProv, fbErr := provider.FallbackForLocation(loc, prov.Name()); fbErr == nil {
			fmt.Fprintf(os.Stderr, "warning: provider %s failed (%v); falling back to %s\n", prov.Name(), condErr, fallbackProv.Name())
			prov = fallbackProv
			cond, fc, alerts, condErr, fcErr, alErr = fetchData(prov)
		}
	}

	if condErr != nil {
		return fmt.Errorf("weather (%s): %w", prov.Name(), condErr)
	}
	if fcErr != nil {
		fmt.Fprintf(os.Stderr, "warning: forecast unavailable: %v\n", fcErr)
	}
	if alErr != nil {
		fmt.Fprintf(os.Stderr, "warning: alerts unavailable: %v\n", alErr)
	}

	if cond != nil && cond.Astronomy == nil && (loc.Lat != 0 || loc.Lon != 0) {
		obsTime := time.Now()
		if !cond.ObservedAt.IsZero() {
			obsTime = cond.ObservedAt
		}
		if a, err := astro.Calculate(loc.Lat, loc.Lon, obsTime); err == nil {
			cond.Astronomy = a
		}
	}

	renderErr := output.Render(output.RenderData{
		Conditions: cond,
		Forecast:   fc,
		Alerts:     alerts,
	}, output.RenderOptions{
		ForceJSON:    c.Bool("json"),
		Units:        units,
		ShowForecast: opts.showForecast,
		ShowAlerts:   c.Bool("alerts"),
		ShowHourly:   opts.showHourly,
		HourlyLimit:  c.Int("hours"),
		Short:        c.Bool("short"),
		Template:     c.String("template"),
	})
	if renderErr != nil {
		return renderErr
	}

	if c.Bool("exit-code-on-alerts") && len(alerts) > 0 {
		for _, a := range alerts {
			if a.IsWarning() {
				return cli.Exit("", 2)
			}
		}
		return cli.Exit("", 1)
	}

	return nil
}
