package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/config"
)

var (
	styleConfigPath  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleConfigKey   = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Width(20)
	styleConfigValue = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	styleConfigEmpty = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	styleConfigSaved = lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
)

// configCommand returns the `wx config` subcommand.
func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "show or update wx configuration",
		Description: func() string {
			path, _ := config.Path()
			return fmt.Sprintf("Config file: %s", path)
		}(),
		Action: configShow,
		Subcommands: []*cli.Command{
			{
				Name:  "show",
				Usage: "show current configuration (default)",
				Action: configShow,
			},
			{
				Name:  "set",
				Usage: "set one or more configuration values",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "location",
						Aliases: []string{"l"},
						Usage:   "default location: zip code or 'City, ST' (empty to clear)",
					},
					&cli.StringFlag{
						Name:    "units",
						Aliases: []string{"u"},
						Usage:   "default units: imperial or metric (empty to clear)",
					},
				},
				Action: configSet,
			},
		},
	}
}

func configShow(c *cli.Context) error {
	path, err := config.Path()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Printf("%s %s\n\n", styleConfigPath.Render("Config file:"), styleConfigPath.Render(path))

	printConfigField("default_location", cfg.DefaultLocation)
	printConfigField("units", cfg.Units)
	fmt.Println()

	return nil
}

func printConfigField(key, value string) {
	k := styleConfigKey.Render(key)
	if value == "" {
		fmt.Printf("  %s %s\n", k, styleConfigEmpty.Render("(not set)"))
	} else {
		fmt.Printf("  %s %s\n", k, styleConfigValue.Render(value))
	}
}

func configSet(c *cli.Context) error {
	if !c.IsSet("location") && !c.IsSet("units") {
		return fmt.Errorf("provide at least one flag: --location or --units (see: wx config set --help)")
	}

	path, err := config.Path()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if c.IsSet("location") {
		cfg.DefaultLocation = c.String("location")
	}
	if c.IsSet("units") {
		v := c.String("units")
		if v != "" && v != "imperial" && v != "metric" {
			return fmt.Errorf("invalid units %q: must be imperial or metric", v)
		}
		cfg.Units = v
	}

	if err := config.Save(path, cfg); err != nil {
		return err
	}

	fmt.Printf("%s %s\n\n", styleConfigSaved.Render("Saved:"), styleConfigPath.Render(path))
	printConfigField("default_location", cfg.DefaultLocation)
	printConfigField("units", cfg.Units)
	fmt.Println()

	return nil
}
