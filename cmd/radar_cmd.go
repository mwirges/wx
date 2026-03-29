package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/radar"
)

func radarCommand() *cli.Command {
	return &cli.Command{
		Name:  "radar",
		Usage: "display radar for the current or specified location",
		Description: "Renders NEXRAD radar in the terminal using half-block characters.\n" +
			"Current conditions use the NWS MRMS WMS; loop frames use Iowa State IEM.\n" +
			"Requires a truecolor terminal (iTerm2, VS Code, modern xterm, etc.).",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "location",
				Aliases: []string{"l"},
				Usage:   "zip code, 'City, ST', or blank for auto-detect",
			},
			&cli.StringFlag{
				Name:    "station",
				Aliases: []string{"s"},
				Usage:   "NEXRAD station ID to center radar on (e.g. KIWX, KMKE); overrides --location",
			},
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Usage:   "interactive mode: change product, zoom, and loop with keyboard shortcuts",
			},
			&cli.BoolFlag{
				Name:    "loop",
				Aliases: []string{"L"},
				Usage:   "animate recent frames as a radar loop (Ctrl+C to exit)",
			},
			&cli.IntFlag{
				Name:  "frames",
				Value: 6,
				Usage: "number of frames to fetch for --loop",
			},
			&cli.IntFlag{
				Name:  "interval",
				Value: 600,
				Usage: "milliseconds between loop frames",
			},
			&cli.StringFlag{
				Name:  "product",
				Value: string(radar.ProductCompositeReflectivity),
				Usage: "radar product: composite-reflectivity, base-reflectivity, storm-relative-velocity, echo-tops",
			},
			&cli.Float64Flag{
				Name:  "radius",
				Value: 200,
				Usage: "km radius around the location center",
			},
			&cli.BoolFlag{
				Name:  "no-cache",
				Usage: "bypass the local cache",
			},
			&cli.BoolFlag{
				Name:  "no-inline",
				Usage: "force half-block rendering even when the terminal supports inline images",
			},
		},
		Action: radarAction,
	}
}

func radarAction(c *cli.Context) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("radar rendering requires a TTY — pipe output is not supported")
	}

	termW, termH, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termW, termH = 120, 40
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cfg, _ := config.Load()

	var ch *cache.Cache
	if c.Bool("no-cache") {
		ch = cache.NewNoOp()
	} else {
		ch, err = cache.New()
		if err != nil {
			ch = cache.NewNoOp()
		}
	}

	locInput := c.String("location")
	if locInput == "" {
		locInput = cfg.DefaultLocation
	}
	loc, err := location.Resolve(ctx, locInput, ch)
	if err != nil {
		return err
	}

	prov, err := radar.ForLocation(loc)
	if err != nil {
		return err
	}

	// If a station ID was provided, override the center location with the
	// station's coordinates. The provider selection above uses the resolved
	// loc for country-code matching; we type-assert to StationLookup after.
	if stationID := c.String("station"); stationID != "" {
		sl, ok := prov.(radar.StationLookup)
		if !ok {
			return fmt.Errorf("radar provider %q does not support station lookup", prov.Name())
		}
		st, err := sl.LookupStation(ctx, stationID, ch)
		if err != nil {
			return fmt.Errorf("station lookup: %w", err)
		}
		loc.Lat = st.Lat
		loc.Lon = st.Lon
		loc.DisplayName = fmt.Sprintf("%s — %s", st.ID, st.Name)
	}

	opts := radar.Options{
		Product:  radar.Product(c.String("product")),
		RadiusKM: c.Float64("radius"),
	}
	mode := radar.DetectTerminal()
	if c.Bool("no-inline") {
		mode = radar.TermHalfBlock
	}

	// Interactive mode — full TUI with keyboard controls.
	if c.Bool("interactive") {
		icfg := radar.InteractiveConfig{
			Loc:       loc,
			Provider:  prov,
			Cache:     ch,
			Product:   opts.Product,
			RadiusKM:  opts.RadiusKM,
			TermMode:  mode,
			NumFrames: c.Int("frames"),
		}
		m := radar.NewInteractiveModel(icfg)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err := p.Run()
		return err
	}

	renderOpts := radar.RenderOptions{
		TermWidth:  termW,
		TermHeight: termH,
		Mode:       mode,
	}

	if c.Bool("loop") {
		return runRadarLoop(ctx, prov, loc, opts, renderOpts,
			c.Int("frames"), c.Int("interval"), ch)
	}

	frame, err := prov.CurrentFrame(ctx, loc, opts, ch)
	if err != nil {
		return fmt.Errorf("radar: %w", err)
	}
	return radar.RenderFrame(os.Stdout, frame, loc.DisplayName, renderOpts)
}

func runRadarLoop(
	ctx context.Context,
	prov radar.Provider,
	loc location.Location,
	opts radar.Options,
	renderOpts radar.RenderOptions,
	nFrames, intervalMS int,
	ch *cache.Cache,
) error {
	fmt.Fprintf(os.Stderr, "Fetching %d radar frames…\n", nFrames)

	frames, err := prov.RecentFrames(ctx, loc, opts, nFrames, ch)
	if err != nil {
		return fmt.Errorf("radar loop: %w", err)
	}

	// Enter alternate screen so we don't trash the user's scrollback.
	fmt.Print("\033[?1049h")
	defer fmt.Print("\033[?1049l\033[?25h") // exit alternate screen + restore cursor

	// Hide cursor while animating.
	fmt.Print("\033[?25l")

	// Handle Ctrl+C gracefully.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	ticker := time.NewTicker(time.Duration(intervalMS) * time.Millisecond)
	defer ticker.Stop()

	i := 0
	for {
		// Render current frame.
		fmt.Print("\033[H") // cursor home (overwrite, no flicker)
		if err := radar.RenderFrame(os.Stdout, frames[i], loc.DisplayName, renderOpts); err != nil {
			return err
		}
		hint := fmt.Sprintf("  Frame %d/%d  ·  Ctrl+C to exit", i+1, len(frames))
		fmt.Print("\033[K" + hint) // clear to EOL then print hint

		i = (i + 1) % len(frames)

		select {
		case <-sig:
			return nil
		case <-ticker.C:
		}
	}
}
