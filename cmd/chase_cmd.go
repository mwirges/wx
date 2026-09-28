package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/chase"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/output"
	"github.com/mwirges/wx/internal/radar"
)

func chaseCommand() *cli.Command {
	return &cli.Command{
		Name:    "chase",
		Aliases: []string{"stormchase", "hotspots"},
		Usage:   "remote storm chasing — monitor and drill into active severe weather clusters across the US",
		Description: "Clusters active nationwide severe weather warnings into regional storm systems,\n" +
			"ranks them by severity, and lets you inspect storm cells or jump straight into Doppler radar.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "radar",
				Aliases: []string{"r"},
				Usage:   "jump directly to interactive radar for the top hotspot or selected cluster",
			},
			&cli.BoolFlag{
				Name:  "tornado",
				Usage: "filter to only clusters with active tornado warnings",
			},
			&cli.BoolFlag{
				Name:    "severe",
				Aliases: []string{"s"},
				Usage:   "filter to only convective severe thunderstorm and tornado warnings",
			},
			&cli.BoolFlag{
				Name:  "spc",
				Usage: "display official SPC convective outlooks and active mesoscale discussions",
			},
			&cli.BoolFlag{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "list clusters and exit without interactive menu",
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
		},
		Action: runChase,
	}
}

func runChase(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	forceJSON := c.Bool("json")
	listOnly := c.Bool("list")
	radarJump := c.Bool("radar")
	onlyTornado := c.Bool("tornado")
	onlySevere := c.Bool("severe")
	onlySPC := c.Bool("spc")
	noCache := c.Bool("no-cache")

	var targetID int
	for _, arg := range c.Args().Slice() {
		switch arg {
		case "--json", "-j":
			forceJSON = true
		case "--list", "-l":
			listOnly = true
		case "--radar", "-r":
			radarJump = true
		case "--tornado":
			onlyTornado = true
		case "--severe", "-s":
			onlySevere = true
		case "--spc":
			onlySPC = true
		case "--no-cache":
			noCache = true
		default:
			if !strings.HasPrefix(arg, "-") {
				if id, err := strconv.Atoi(arg); err == nil && targetID == 0 {
					targetID = id
				}
			}
		}
	}

	var ch *cache.Cache
	var err error
	if noCache {
		ch = cache.NewNoOp()
	} else {
		ch, err = cache.New()
		if err != nil {
			ch = cache.NewNoOp()
		}
	}

	payload, err := chase.FetchClusters(ctx, ch)
	if err != nil {
		return fmt.Errorf("chase: %w", err)
	}

	// Direct SPC outlook mode if requested
	if onlySPC {
		if payload.SPC != nil {
			return output.RenderSPC(payload.SPC, output.SPCOptions{ForceJSON: forceJSON, ForcePretty: !forceJSON})
		}
		return fmt.Errorf("no SPC convective data available")
	}

	// Apply filters if requested
	if onlyTornado {
		var filtered []models.StormCluster
		for _, cl := range payload.Clusters {
			if cl.HazardsCount["Tornado Warning"] > 0 {
				filtered = append(filtered, cl)
			}
		}
		payload.Clusters = filtered
		payload.TotalClusters = len(filtered)
	} else if onlySevere {
		var filtered []models.StormCluster
		for _, cl := range payload.Clusters {
			if cl.HazardsCount["Tornado Warning"] > 0 || cl.HazardsCount["Severe Thunderstorm Warning"] > 0 {
				filtered = append(filtered, cl)
			}
		}
		payload.Clusters = filtered
		payload.TotalClusters = len(filtered)
	}

	// JSON mode
	if forceJSON {
		return output.RenderChase(payload, output.ChaseOptions{ForceJSON: true})
	}

	isTTY := term.IsTerminal(int(os.Stdout.Fd()))

	// Check if a specific cluster ID was requested
	var targetCluster *models.StormCluster
	if targetID > 0 {
		for i := range payload.Clusters {
			if payload.Clusters[i].ID == targetID {
				targetCluster = &payload.Clusters[i]
				break
			}
		}
	}

	// Direct radar jump flag
	if radarJump {
		if targetCluster != nil {
			return launchChaseRadar(targetCluster.CenterLat, targetCluster.CenterLon, targetCluster.Name, ch)
		}
		if len(payload.Clusters) > 0 {
			top := &payload.Clusters[0]
			return launchChaseRadar(top.CenterLat, top.CenterLon, top.Name, ch)
		}
		return fmt.Errorf("no active storm clusters to track on radar")
	}

	// If a specific cluster was targeted without --radar
	if targetCluster != nil {
		if !isTTY || listOnly {
			return output.RenderClusterDetail(os.Stdout, targetCluster, output.ChaseOptions{ForcePretty: true})
		}
		return runInteractiveCluster(targetCluster, ch)
	}

	// Non-interactive list output
	if !isTTY || listOnly {
		return output.RenderChase(payload, output.ChaseOptions{ForcePretty: true})
	}

	// Interactive master-detail navigation
	return runInteractiveChase(payload, ch)
}

func runInteractiveChase(payload *models.ChasePayload, ch *cache.Cache) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		_ = output.RenderChase(payload, output.ChaseOptions{ForcePretty: true})
		if len(payload.Clusters) == 0 {
			return nil
		}

		fmt.Print("⚡ Select cluster [1-N], [r]adar on top hotspot, [s]pc outlook, or [q]uit: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil
		}
		input = strings.TrimSpace(input)
		if input == "q" || input == "quit" || input == "exit" {
			return nil
		}
		if input == "s" || input == "spc" {
			if payload.SPC != nil {
				_ = output.RenderSPC(payload.SPC, output.SPCOptions{ForcePretty: true})
				fmt.Print("\nPress Enter to return to clusters...")
				_, _ = reader.ReadString('\n')
			}
			continue
		}
		if input == "r" || input == "radar" {
			top := &payload.Clusters[0]
			_ = launchChaseRadar(top.CenterLat, top.CenterLon, top.Name, ch)
			continue
		}

		// Check if user entered `<id> r` (e.g. `2 r`)
		parts := strings.Fields(input)
		if len(parts) > 0 {
			if id, err := strconv.Atoi(parts[0]); err == nil {
				var selected *models.StormCluster
				for i := range payload.Clusters {
					if payload.Clusters[i].ID == id {
						selected = &payload.Clusters[i]
						break
					}
				}
				if selected != nil {
					if len(parts) > 1 && (parts[1] == "r" || parts[1] == "radar") {
						_ = launchChaseRadar(selected.CenterLat, selected.CenterLon, selected.Name, ch)
						continue
					}
					// Drill down into cluster
					if err := runInteractiveCluster(selected, ch); err != nil {
						return err
					}
				} else {
					fmt.Printf("Invalid cluster #%d. Please pick a number from 1 to %d.\n\n", id, len(payload.Clusters))
				}
			}
		}
	}
}

func runInteractiveCluster(c *models.StormCluster, ch *cache.Cache) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		_ = output.RenderClusterDetail(os.Stdout, c, output.ChaseOptions{ForcePretty: true})

		fmt.Print("⚡ Actions: [r]adar on cluster center, cell #[1-M] for cell radar, [b]ack, or [q]uit: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil
		}
		input = strings.TrimSpace(input)
		if input == "q" || input == "quit" || input == "exit" {
			os.Exit(0)
		}
		if input == "b" || input == "back" {
			return nil
		}
		if input == "r" || input == "radar" {
			_ = launchChaseRadar(c.CenterLat, c.CenterLon, c.Name, ch)
			continue
		}

		// Check if cell number selected
		cleanInput := strings.TrimPrefix(input, fmt.Sprintf("%d.", c.ID))
		if cellIdx, err := strconv.Atoi(cleanInput); err == nil {
			if cellIdx >= 1 && cellIdx <= len(c.Cells) {
				cell := c.Cells[cellIdx-1]
				cellName := fmt.Sprintf("%s — %s", cell.Event, cell.AreaDesc)
				_ = launchChaseRadar(cell.Latitude, cell.Longitude, cellName, ch)
				continue
			}
		}
	}
}

func launchChaseRadar(lat, lon float64, name string, ch *cache.Cache) error {
	loc := location.Location{
		Lat:         lat,
		Lon:         lon,
		DisplayName: name,
	}

	prov, err := radar.ForLocation(loc)
	if err != nil {
		return fmt.Errorf("radar provider: %w", err)
	}

	mode := radar.DetectTerminal()

	icfg := radar.InteractiveConfig{
		Loc:       loc,
		Provider:  prov,
		Cache:     ch,
		Product:   radar.ProductCompositeReflectivity,
		RadiusKM:  180,
		TermMode:  mode,
		NumFrames: 6,
	}

	m := radar.NewInteractiveModel(icfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
