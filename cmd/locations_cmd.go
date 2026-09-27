package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/config"
)

var (
	styleLocHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)
	styleLocName   = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	styleLocValue  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleLocSaved  = lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true)
	styleLocMuted  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	styleLocIndex  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

// locationsCommand returns the `wx locations` subcommand.
func locationsCommand() *cli.Command {
	return &cli.Command{
		Name:    "locations",
		Aliases: []string{"loc", "location"},
		Usage:   "manage favorite locations and view recent searches",
		Action:  locationsList,
		Subcommands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "list favorite and recent locations (default)",
				Action:  locationsList,
			},
			{
				Name:      "add",
				Aliases:   []string{"set"},
				Usage:     "add or update a favorite location alias (e.g. wx locations add Home 'Fort Wayne, IN')",
				ArgsUsage: "<name> [location]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "units",
						Aliases: []string{"u"},
						Usage:   "units for this location: imperial or metric",
					},
					&cli.StringFlag{
						Name:    "radar-product",
						Aliases: []string{"p", "product"},
						Usage:   "default radar product (composite-reflectivity, base-reflectivity, storm-relative-velocity, echo-tops)",
					},
					&cli.Float64Flag{
						Name:    "radar-radius",
						Aliases: []string{"r", "radius"},
						Usage:   "default radar radius in km",
					},
					&cli.StringFlag{
						Name:    "station",
						Aliases: []string{"s"},
						Usage:   "default NEXRAD station ID",
					},
				},
				Action: locationsAdd,
			},
			{
				Name:      "config",
				Usage:     "configure per-location preferences (units, radar product/radius, station)",
				ArgsUsage: "<name>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "units",
						Aliases: []string{"u"},
						Usage:   "units for this location: imperial or metric (empty to clear)",
					},
					&cli.StringFlag{
						Name:    "radar-product",
						Aliases: []string{"p", "product"},
						Usage:   "default radar product: composite-reflectivity, base-reflectivity, storm-relative-velocity, echo-tops",
					},
					&cli.Float64Flag{
						Name:    "radar-radius",
						Aliases: []string{"r", "radius"},
						Usage:   "default radar radius in km (0 to clear)",
					},
					&cli.StringFlag{
						Name:    "station",
						Aliases: []string{"s"},
						Usage:   "default NEXRAD station ID (empty to clear)",
					},
					&cli.BoolFlag{
						Name:  "clear",
						Usage: "clear all custom settings for this location",
					},
				},
				Action: locationsConfig,
			},
			{
				Name:      "rm",
				Aliases:   []string{"remove", "delete"},
				Usage:     "remove a favorite location alias (e.g. wx locations rm Home)",
				ArgsUsage: "<name>",
				Action:    locationsRemove,
			},
			{
				Name:    "recent",
				Aliases: []string{"recents"},
				Usage:   "list recently queried locations",
				Action:  locationsRecent,
			},
			{
				Name:    "clear-recents",
				Aliases: []string{"clear"},
				Usage:   "clear recent locations history",
				Action:  locationsClearRecents,
			},
		},
	}
}

func locationsList(c *cli.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	hasFavorites := len(cfg.Favorites) > 0
	hasRecents := len(cfg.RecentLocations) > 0

	if !hasFavorites && !hasRecents {
		fmt.Println(styleLocMuted.Render("No favorite locations or recent searches."))
		fmt.Println(styleLocMuted.Render("Add a favorite with: wx locations add <name> <location>"))
		return nil
	}

	if hasFavorites {
		fmt.Println(styleLocHeader.Render("Favorite Locations:"))
		for _, f := range cfg.Favorites {
			fmt.Printf("  %-16s %s\n", styleLocName.Render(f.Name), styleLocValue.Render(f.Value))
			settings, hasSettings := cfg.GetLocationSettings(f.Name)
			if !hasSettings {
				settings, hasSettings = cfg.GetLocationSettings(f.Value)
			}
			if hasSettings {
				var details []string
				if settings.Units != "" {
					details = append(details, fmt.Sprintf("units: %s", settings.Units))
				}
				if settings.DefaultRadarProduct != "" {
					rad := fmt.Sprintf("radar: %s", settings.DefaultRadarProduct)
					if settings.DefaultRadarRadius > 0 {
						rad += fmt.Sprintf(" (%.0fkm)", settings.DefaultRadarRadius)
					}
					details = append(details, rad)
				} else if settings.DefaultRadarRadius > 0 {
					details = append(details, fmt.Sprintf("radius: %.0fkm", settings.DefaultRadarRadius))
				}
				if settings.RadarStation != "" {
					details = append(details, fmt.Sprintf("station: %s", settings.RadarStation))
				}
				if len(details) > 0 {
					fmt.Printf("                   %s %s\n", styleLocMuted.Render("└─"), styleLocMuted.Render(strings.Join(details, ", ")))
				}
			}
		}
		fmt.Println()
	} else {
		fmt.Println(styleLocHeader.Render("Favorite Locations:"))
		fmt.Printf("  %s\n\n", styleLocMuted.Render("(none configured — add with 'wx locations add <name> <location>')"))
	}

	if hasRecents {
		fmt.Println(styleLocHeader.Render("Recent Locations:"))
		for i, r := range cfg.RecentLocations {
			fmt.Printf("  %s %s\n", styleLocIndex.Render(fmt.Sprintf("%2d.", i+1)), styleLocName.Render(r))
		}
		fmt.Println()
	}

	return nil
}

type locConfigFlags struct {
	posArgs      []string
	units        string
	hasUnits     bool
	radarProduct string
	hasProduct   bool
	radarRadius  float64
	hasRadius    bool
	station      string
	hasStation   bool
	clear        bool
}

func parseLocConfigArgs(c *cli.Context) *locConfigFlags {
	flags := &locConfigFlags{}
	if c.IsSet("units") {
		flags.units = c.String("units")
		flags.hasUnits = true
	}
	if c.IsSet("radar-product") {
		flags.radarProduct = c.String("radar-product")
		flags.hasProduct = true
	}
	if c.IsSet("radar-radius") {
		flags.radarRadius = c.Float64("radar-radius")
		flags.hasRadius = true
	}
	if c.IsSet("station") {
		flags.station = c.String("station")
		flags.hasStation = true
	}
	if c.Bool("clear") {
		flags.clear = true
	}

	rawArgs := c.Args().Slice()
	for i := 0; i < len(rawArgs); i++ {
		arg := rawArgs[i]
		switch {
		case arg == "--clear":
			flags.clear = true
		case arg == "--units" || arg == "-u":
			if i+1 < len(rawArgs) {
				flags.units = rawArgs[i+1]
				flags.hasUnits = true
				i++
			}
		case strings.HasPrefix(arg, "--units="):
			flags.units = strings.TrimPrefix(arg, "--units=")
			flags.hasUnits = true
		case arg == "--radar-product" || arg == "-p" || arg == "--product":
			if i+1 < len(rawArgs) {
				flags.radarProduct = rawArgs[i+1]
				flags.hasProduct = true
				i++
			}
		case strings.HasPrefix(arg, "--radar-product="):
			flags.radarProduct = strings.TrimPrefix(arg, "--radar-product=")
			flags.hasProduct = true
		case strings.HasPrefix(arg, "--product="):
			flags.radarProduct = strings.TrimPrefix(arg, "--product=")
			flags.hasProduct = true
		case arg == "--radar-radius" || arg == "-r" || arg == "--radius":
			if i+1 < len(rawArgs) {
				var r float64
				if _, err := fmt.Sscanf(rawArgs[i+1], "%f", &r); err == nil {
					flags.radarRadius = r
					flags.hasRadius = true
				}
				i++
			}
		case strings.HasPrefix(arg, "--radar-radius="):
			var r float64
			if _, err := fmt.Sscanf(strings.TrimPrefix(arg, "--radar-radius="), "%f", &r); err == nil {
				flags.radarRadius = r
				flags.hasRadius = true
			}
		case strings.HasPrefix(arg, "--radius="):
			var r float64
			if _, err := fmt.Sscanf(strings.TrimPrefix(arg, "--radius="), "%f", &r); err == nil {
				flags.radarRadius = r
				flags.hasRadius = true
			}
		case arg == "--station" || arg == "-s":
			if i+1 < len(rawArgs) {
				flags.station = rawArgs[i+1]
				flags.hasStation = true
				i++
			}
		case strings.HasPrefix(arg, "--station="):
			flags.station = strings.TrimPrefix(arg, "--station=")
			flags.hasStation = true
		default:
			flags.posArgs = append(flags.posArgs, arg)
		}
	}
	return flags
}

func locationsAdd(c *cli.Context) error {
	flags := parseLocConfigArgs(c)
	if len(flags.posArgs) == 0 {
		return fmt.Errorf("usage: wx locations add <name> <location> (e.g. wx locations add Home 'Fort Wayne, IN')")
	}

	name := strings.TrimSpace(flags.posArgs[0])
	val := name
	if len(flags.posArgs) >= 2 {
		val = strings.TrimSpace(strings.Join(flags.posArgs[1:], " "))
	}

	if name == "" || val == "" {
		return fmt.Errorf("location name and target location cannot be empty")
	}

	path, err := config.Path()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.SetFavorite(name, val)

	if flags.hasUnits || flags.hasProduct || flags.hasRadius || flags.hasStation {
		settings, _ := cfg.GetLocationSettings(name)
		if flags.hasUnits {
			if flags.units != "" && flags.units != "imperial" && flags.units != "metric" {
				return fmt.Errorf("invalid units %q: must be imperial or metric", flags.units)
			}
			settings.Units = flags.units
		}
		if flags.hasProduct {
			if flags.radarProduct != "" && !isValidRadarProduct(flags.radarProduct) {
				return fmt.Errorf("invalid radar product %q (valid: composite-reflectivity, base-reflectivity, storm-relative-velocity, echo-tops)", flags.radarProduct)
			}
			settings.DefaultRadarProduct = flags.radarProduct
		}
		if flags.hasRadius {
			settings.DefaultRadarRadius = flags.radarRadius
		}
		if flags.hasStation {
			settings.RadarStation = flags.station
		}
		cfg.SetLocationSettings(name, settings)
	}

	if err := config.Save(path, cfg); err != nil {
		return err
	}

	fmt.Printf("%s %s → %s\n",
		styleLocSaved.Render("Saved favorite:"),
		styleLocName.Render(name),
		styleLocValue.Render(val),
	)
	return nil
}

func locationsConfig(c *cli.Context) error {
	flags := parseLocConfigArgs(c)
	if len(flags.posArgs) == 0 {
		return fmt.Errorf("usage: wx locations config <name> [--units ...] [--radar-product ...] [--radar-radius ...] [--station ...] [--clear]")
	}

	name := strings.TrimSpace(strings.Join(flags.posArgs, " "))
	if name == "" {
		return fmt.Errorf("location name cannot be empty")
	}

	path, err := config.Path()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if flags.clear {
		cfg.DeleteLocationSettings(name)
		if err := config.Save(path, cfg); err != nil {
			return err
		}
		fmt.Printf("%s cleared custom settings for %s\n", styleLocSaved.Render("Success:"), styleLocName.Render(name))
		return nil
	}

	if !flags.hasUnits && !flags.hasProduct && !flags.hasRadius && !flags.hasStation {
		settings, ok := cfg.GetLocationSettings(name)
		if !ok {
			fmt.Printf("No custom settings configured for %s\n", styleLocName.Render(name))
			return nil
		}
		fmt.Printf("%s %s\n", styleLocHeader.Render("Settings for"), styleLocName.Render(name))
		if settings.Units != "" {
			fmt.Printf("  %-16s %s\n", styleLocValue.Render("Units:"), styleLocName.Render(settings.Units))
		}
		if settings.DefaultRadarProduct != "" {
			fmt.Printf("  %-16s %s\n", styleLocValue.Render("Radar Product:"), styleLocName.Render(settings.DefaultRadarProduct))
		}
		if settings.DefaultRadarRadius > 0 {
			fmt.Printf("  %-16s %s\n", styleLocValue.Render("Radar Radius:"), styleLocName.Render(fmt.Sprintf("%.0f km", settings.DefaultRadarRadius)))
		}
		if settings.RadarStation != "" {
			fmt.Printf("  %-16s %s\n", styleLocValue.Render("Radar Station:"), styleLocName.Render(settings.RadarStation))
		}
		return nil
	}

	settings, _ := cfg.GetLocationSettings(name)

	if flags.hasUnits {
		if flags.units != "" && flags.units != "imperial" && flags.units != "metric" {
			return fmt.Errorf("invalid units %q: must be imperial or metric", flags.units)
		}
		settings.Units = flags.units
	}
	if flags.hasProduct {
		if flags.radarProduct != "" && !isValidRadarProduct(flags.radarProduct) {
			return fmt.Errorf("invalid radar product %q (valid: composite-reflectivity, base-reflectivity, storm-relative-velocity, echo-tops)", flags.radarProduct)
		}
		settings.DefaultRadarProduct = flags.radarProduct
	}
	if flags.hasRadius {
		settings.DefaultRadarRadius = flags.radarRadius
	}
	if flags.hasStation {
		settings.RadarStation = flags.station
	}

	cfg.SetLocationSettings(name, settings)
	if err := config.Save(path, cfg); err != nil {
		return err
	}

	fmt.Printf("%s updated settings for %s\n", styleLocSaved.Render("Saved:"), styleLocName.Render(name))
	return nil
}

func isValidRadarProduct(p string) bool {
	switch p {
	case "composite-reflectivity", "base-reflectivity", "storm-relative-velocity", "echo-tops":
		return true
	default:
		return false
	}
}

func locationsRemove(c *cli.Context) error {
	args := c.Args().Slice()
	if len(args) == 0 {
		return fmt.Errorf("usage: wx locations rm <name> (e.g. wx locations rm Home)")
	}

	name := strings.TrimSpace(strings.Join(args, " "))
	path, err := config.Path()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !cfg.RemoveFavorite(name) {
		return fmt.Errorf("favorite location %q not found", name)
	}

	if err := config.Save(path, cfg); err != nil {
		return err
	}

	fmt.Printf("%s %s\n", styleLocSaved.Render("Removed favorite:"), styleLocName.Render(name))
	return nil
}

func locationsRecent(c *cli.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.RecentLocations) == 0 {
		fmt.Println(styleLocMuted.Render("No recent locations recorded."))
		return nil
	}

	fmt.Println(styleLocHeader.Render("Recent Locations:"))
	for i, r := range cfg.RecentLocations {
		fmt.Printf("  %s %s\n", styleLocIndex.Render(fmt.Sprintf("%2d.", i+1)), styleLocName.Render(r))
	}
	return nil
}

func locationsClearRecents(c *cli.Context) error {
	path, err := config.Path()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.ClearRecents()
	if err := config.Save(path, cfg); err != nil {
		return err
	}

	fmt.Println(styleLocSaved.Render("Cleared recent locations history."))
	return nil
}
