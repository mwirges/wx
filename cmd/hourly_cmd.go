package cmd

import (
	"github.com/urfave/cli/v2"
)

func hourlyCommand() *cli.Command {
	return &cli.Command{
		Name:    "hourly",
		Aliases: []string{"H"},
		Usage:   "display hourly forecast for the current or specified location",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank to use config/auto-detect",
			},
			&cli.StringFlag{
				Name:    "units",
				Aliases: []string{"u"},
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
			&cli.BoolFlag{
				Name:    "alerts",
				Aliases: []string{"a"},
				Usage:   "show active weather alerts",
			},
		},
		Action: func(c *cli.Context) error {
			return runWeather(c, weatherOpts{
				showForecast: true,
				showHourly:   true,
				showAlerts:   c.Bool("alerts"),
			})
		},
	}
}
