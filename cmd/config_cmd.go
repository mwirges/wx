package cmd

import (
	"fmt"
	"strings"

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
					&cli.StringFlag{
						Name:    "provider",
						Aliases: []string{"p"},
						Usage:   "default weather provider: nws or openmeteo (empty to clear)",
					},
					&cli.StringFlag{
						Name:    "notifications",
						Aliases: []string{"n"},
						Usage:   "default desktop notifications: true or false (empty to clear)",
					},
					&cli.StringFlag{
						Name:  "menu-bar-format",
						Usage: "default macOS menu bar format: compact, standard, or tactical (empty to clear)",
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
	printConfigField("provider", cfg.Provider)

	notifVal := ""
	if cfg.Notifications != nil {
		notifVal = fmt.Sprintf("%t", *cfg.Notifications)
	}
	printConfigField("notifications", notifVal)
	printConfigField("menu_bar_format", cfg.MenuBarFormat)
	fmt.Println()

	if len(cfg.Favorites) > 0 {
		fmt.Println(styleConfigKey.Render("Favorites:"))
		for _, f := range cfg.Favorites {
			fmt.Printf("  %-16s %s\n", styleConfigValue.Render(f.Name), styleConfigPath.Render(f.Value))
		}
		fmt.Println()
	}

	if len(cfg.RecentLocations) > 0 {
		fmt.Println(styleConfigKey.Render("Recent Locations:"))
		for i, r := range cfg.RecentLocations {
			fmt.Printf("  %2d. %s\n", i+1, styleConfigValue.Render(r))
		}
		fmt.Println()
	}

	if len(cfg.PerLocation) > 0 {
		fmt.Println(styleConfigKey.Render("Per-Location Settings:"))
		for loc, s := range cfg.PerLocation {
			var details []string
			if s.Units != "" {
				details = append(details, fmt.Sprintf("units: %s", s.Units))
			}
			if s.Provider != "" {
				details = append(details, fmt.Sprintf("provider: %s", s.Provider))
			}
			if s.DefaultRadarProduct != "" {
				details = append(details, fmt.Sprintf("radar: %s", s.DefaultRadarProduct))
			}
			if s.DefaultRadarRadius > 0 {
				details = append(details, fmt.Sprintf("radius: %.0fkm", s.DefaultRadarRadius))
			}
			if s.RadarStation != "" {
				details = append(details, fmt.Sprintf("station: %s", s.RadarStation))
			}
			fmt.Printf("  %-16s %s\n", styleConfigValue.Render(loc), styleConfigPath.Render(strings.Join(details, ", ")))
		}
		fmt.Println()
	}

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
	if !c.IsSet("location") && !c.IsSet("units") && !c.IsSet("provider") && !c.IsSet("notifications") && !c.IsSet("menu-bar-format") {
		return fmt.Errorf("provide at least one flag: --location, --units, --provider, --notifications, or --menu-bar-format (see: wx config set --help)")
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
	if c.IsSet("provider") {
		v := strings.ToLower(c.String("provider"))
		if v != "" && v != "nws" && v != "openmeteo" {
			return fmt.Errorf("invalid provider %q: must be nws, openmeteo, or empty", v)
		}
		cfg.Provider = v
	}
	if c.IsSet("notifications") {
		v := c.String("notifications")
		if v == "" {
			cfg.Notifications = nil
		} else if v == "true" {
			val := true
			cfg.Notifications = &val
		} else if v == "false" {
			val := false
			cfg.Notifications = &val
		} else {
			return fmt.Errorf("invalid notifications %q: must be true or false", v)
		}
	}
	if c.IsSet("menu-bar-format") {
		v := strings.ToLower(c.String("menu-bar-format"))
		if v != "" && v != "compact" && v != "standard" && v != "tactical" {
			return fmt.Errorf("invalid menu-bar-format %q: must be compact, standard, or tactical", v)
		}
		cfg.MenuBarFormat = v
	}

	if err := config.Save(path, cfg); err != nil {
		return err
	}

	fmt.Printf("%s %s\n\n", styleConfigSaved.Render("Saved:"), styleConfigPath.Render(path))
	printConfigField("default_location", cfg.DefaultLocation)
	printConfigField("units", cfg.Units)
	printConfigField("provider", cfg.Provider)

	notifVal := ""
	if cfg.Notifications != nil {
		notifVal = fmt.Sprintf("%t", *cfg.Notifications)
	}
	printConfigField("notifications", notifVal)
	printConfigField("menu_bar_format", cfg.MenuBarFormat)
	fmt.Println()

	return nil
}
