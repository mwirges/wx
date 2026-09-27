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
				Action:    locationsAdd,
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

func locationsAdd(c *cli.Context) error {
	args := c.Args().Slice()
	if len(args) == 0 {
		return fmt.Errorf("usage: wx locations add <name> <location> (e.g. wx locations add Home 'Fort Wayne, IN')")
	}

	name := strings.TrimSpace(args[0])
	val := name
	if len(args) >= 2 {
		val = strings.TrimSpace(strings.Join(args[1:], " "))
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
