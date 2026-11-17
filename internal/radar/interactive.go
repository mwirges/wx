package radar

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
)

// ── Messages ─────────────────────────────────────────────────────────────────

type frameMsg struct{ frame *Frame }
type framesMsg struct{ frames []*Frame }
type errMsg struct{ err error }
type tickMsg struct{}

// ── Radius presets (km) ──────────────────────────────────────────────────────

var radiusPresets = []float64{50, 100, 150, 200, 300, 500, 800, 1200, 2000}

func nextRadius(cur float64, delta int) float64 {
	// Find closest preset, then step by delta.
	best := 0
	for i, r := range radiusPresets {
		if abs(r-cur) < abs(radiusPresets[best]-cur) {
			best = i
		}
	}
	next := best + delta
	if next < 0 {
		next = 0
	}
	if next >= len(radiusPresets) {
		next = len(radiusPresets) - 1
	}
	return radiusPresets[next]
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// ── Products ─────────────────────────────────────────────────────────────────

var products = []Product{
	ProductCompositeReflectivity,
	ProductBaseReflectivity,
	ProductStormRelativeVelocity,
	ProductEchoTops,
	ProductPrecipType,
	ProductOneHourPrecip,
	ProductStormTotalPrecip,
}

var loopSpeeds = []time.Duration{
	150 * time.Millisecond,
	300 * time.Millisecond,
	600 * time.Millisecond,
	1000 * time.Millisecond,
	1800 * time.Millisecond,
}

func nextProduct(cur Product) Product {
	for i, p := range products {
		if p == cur {
			return products[(i+1)%len(products)]
		}
	}
	return products[0]
}

// ── Model ────────────────────────────────────────────────────────────────────

// InteractiveConfig holds the initial parameters for the interactive radar.
type InteractiveConfig struct {
	Loc       location.Location
	Provider  Provider
	Cache     *cache.Cache
	Product   Product
	RadiusKM  float64
	TermMode  TermCapability
	NumFrames int
}

// InteractiveModel is the bubbletea model for the interactive radar viewer.
type InteractiveModel struct {
	cfg InteractiveConfig

	// Current settings (user can change these)
	product  Product
	radius   float64
	loopMode bool
	paused   bool
	speedIdx int // index into loopSpeeds; default 2 (600ms)

	// Data
	frame    *Frame   // current single frame
	frames   []*Frame // loop frames
	frameIdx int

	// UI state
	width    int
	height   int
	loading  bool
	status   string
	err      error
	quitting bool
}

// NewInteractiveModel creates the initial model.
func NewInteractiveModel(cfg InteractiveConfig) InteractiveModel {
	return InteractiveModel{
		cfg:      cfg,
		product:  cfg.Product,
		radius:   cfg.RadiusKM,
		speedIdx: 2, // 600ms
	}
}

// Product returns the currently selected radar product.
func (m InteractiveModel) Product() Product {
	return m.product
}

// Radius returns the currently selected radar radius in km.
func (m InteractiveModel) Radius() float64 {
	return m.radius
}

func (m InteractiveModel) Init() tea.Cmd {
	return m.fetchCurrent()
}

// ── Update ───────────────────────────────────────────────────────────────────

func (m InteractiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "p":
			m.product = nextProduct(m.product)
			m.frames = nil
			m.frame = nil
			m.loading = true
			m.status = "Switching product…"
			if m.loopMode {
				return m, m.fetchLoop()
			}
			return m, m.fetchCurrent()

		case "+", "=":
			m.radius = nextRadius(m.radius, -1) // smaller radius = zoom in
			m.frames = nil
			m.frame = nil
			m.loading = true
			if m.radius > 200 {
				sites := len(StationsWithinRadius(m.cfg.Loc.Lat, m.cfg.Loc.Lon, m.radius))
				m.status = fmt.Sprintf("Zooming in (%.0f km — Mosaic · %d Sites)…", m.radius, sites)
			} else {
				m.status = fmt.Sprintf("Zooming in (%.0f km)…", m.radius)
			}
			if m.loopMode {
				return m, m.fetchLoop()
			}
			return m, m.fetchCurrent()

		case "-":
			m.radius = nextRadius(m.radius, 1) // larger radius = zoom out
			m.frames = nil
			m.frame = nil
			m.loading = true
			if m.radius > 200 {
				sites := len(StationsWithinRadius(m.cfg.Loc.Lat, m.cfg.Loc.Lon, m.radius))
				m.status = fmt.Sprintf("Zooming out (%.0f km — Multi-Radar Mosaic · %d Sites)…", m.radius, sites)
			} else {
				m.status = fmt.Sprintf("Zooming out (%.0f km)…", m.radius)
			}
			if m.loopMode {
				return m, m.fetchLoop()
			}
			return m, m.fetchCurrent()

		case "1":
			m.radius = 150
			m.frames = nil
			m.frame = nil
			m.loading = true
			m.status = "Local Scan (150 km — Single Site)…"
			if m.loopMode {
				return m, m.fetchLoop()
			}
			return m, m.fetchCurrent()

		case "2":
			m.radius = 250
			m.frames = nil
			m.frame = nil
			m.loading = true
			sites := len(StationsWithinRadius(m.cfg.Loc.Lat, m.cfg.Loc.Lon, m.radius))
			m.status = fmt.Sprintf("Metro Scan (250 km — Near-Field Composite · %d Sites)…", sites)
			if m.loopMode {
				return m, m.fetchLoop()
			}
			return m, m.fetchCurrent()

		case "3":
			m.radius = 500
			m.frames = nil
			m.frame = nil
			m.loading = true
			sites := len(StationsWithinRadius(m.cfg.Loc.Lat, m.cfg.Loc.Lon, m.radius))
			m.status = fmt.Sprintf("Regional Mosaic (500 km — Multi-Radar Mosaic · %d Sites)…", sites)
			if m.loopMode {
				return m, m.fetchLoop()
			}
			return m, m.fetchCurrent()

		case "4":
			m.radius = 1000
			m.frames = nil
			m.frame = nil
			m.loading = true
			sites := len(StationsWithinRadius(m.cfg.Loc.Lat, m.cfg.Loc.Lon, m.radius))
			m.status = fmt.Sprintf("Synoptic Sector (1000 km — Multi-State Sector Mosaic · %d Sites)…", sites)
			if m.loopMode {
				return m, m.fetchLoop()
			}
			return m, m.fetchCurrent()

		case "5":
			m.radius = 2000
			m.frames = nil
			m.frame = nil
			m.loading = true
			sites := len(StationsWithinRadius(m.cfg.Loc.Lat, m.cfg.Loc.Lon, m.radius))
			m.status = fmt.Sprintf("CONUS National Mosaic (2000 km — Seamless CONUS Mosaic · %d Sites)…", sites)
			if m.loopMode {
				return m, m.fetchLoop()
			}
			return m, m.fetchCurrent()

		case "l":
			m.loopMode = !m.loopMode
			if m.loopMode {
				m.loading = true
				m.status = "Fetching loop frames…"
				return m, m.fetchLoop()
			}
			// Exiting loop mode — show current frame
			m.frames = nil
			m.frameIdx = 0
			if m.frame == nil {
				m.loading = true
				m.status = "Fetching current frame…"
				return m, m.fetchCurrent()
			}
			return m, nil

		case " ":
			if m.loopMode {
				m.paused = !m.paused
			}
			return m, nil

		case "r":
			m.loading = true
			m.status = "Refreshing…"
			if m.loopMode {
				return m, m.fetchLoop()
			}
			return m, m.fetchCurrent()

		case "left", "h", "[":
			if m.loopMode && len(m.frames) > 0 {
				m.frameIdx = (m.frameIdx - 1 + len(m.frames)) % len(m.frames)
			}
			return m, nil

		case "right", "]":
			if m.loopMode && len(m.frames) > 0 {
				m.frameIdx = (m.frameIdx + 1) % len(m.frames)
			}
			return m, nil

		case "f", ">", ".":
			if m.loopMode && m.speedIdx > 0 {
				m.speedIdx--
			}
			return m, nil

		case "s", "<", ",":
			if m.loopMode && m.speedIdx < len(loopSpeeds)-1 {
				m.speedIdx++
			}
			return m, nil
		}

	case frameMsg:
		m.frame = msg.frame
		m.loading = false
		m.status = ""
		m.err = nil
		return m, nil

	case framesMsg:
		m.frames = msg.frames
		m.frameIdx = 0
		m.loading = false
		m.status = ""
		m.err = nil
		return m, m.tickCmd()

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.status = ""
		return m, nil

	case tickMsg:
		if !m.loopMode || len(m.frames) == 0 {
			return m, nil
		}
		if !m.paused {
			m.frameIdx = (m.frameIdx + 1) % len(m.frames)
		}
		return m, m.tickCmd()
	}

	return m, nil
}

// ── View ─────────────────────────────────────────────────────────────────────

func (m InteractiveModel) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return "Initializing…"
	}

	var sb strings.Builder

	// Reserve 1 line for the help bar at the bottom (and 1 inline error banner line if present).
	errLines := 0
	if m.err != nil && m.height > 2 {
		errLines = 1
	}
	renderH := m.height - 1 - errLines
	if renderH < 1 {
		renderH = 1
	}

	// Pick which frame to show.
	var activeFrame *Frame
	if m.loopMode && len(m.frames) > 0 {
		activeFrame = m.frames[m.frameIdx]
	} else {
		activeFrame = m.frame
	}

	if m.loading || activeFrame == nil {
		msg := sanitizeSingleLine(m.status)
		if msg == "" {
			msg = "Loading…"
		}
		pill := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#38E1FF")).
			Bold(true).
			Render("╭─ ◆ " + msg + " ─╮")
		if lipgloss.Width(pill) > m.width {
			pill = lipgloss.NewStyle().MaxWidth(m.width).Render("◆ " + msg)
		}
		padY := renderH / 2
		for i := 0; i < padY; i++ {
			sb.WriteString("\n")
		}
		padX := (m.width - lipgloss.Width(pill)) / 2
		if padX < 0 {
			padX = 0
		}
		sb.WriteString(strings.Repeat(" ", padX))
		sb.WriteString(pill)
		sb.WriteString("\n")
		for i := padY + 1; i < renderH; i++ {
			sb.WriteString("\n")
		}
	} else {
		opts := RenderOptions{
			TermWidth:  m.width,
			TermHeight: renderH,
			Mode:       m.cfg.TermMode,
		}
		if frameErr := RenderFrame(&sb, activeFrame, m.cfg.Loc.DisplayName, opts); frameErr != nil {
			fallback := lipgloss.NewStyle().MaxWidth(m.width).Render(sanitizeSingleLine(frameErr.Error()))
			sb.WriteString(fallback)
			sb.WriteString("\n")
			for i := 1; i < renderH; i++ {
				sb.WriteString("\n")
			}
		}
	}

	if errLines == 1 {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B56")).Bold(true).MaxWidth(m.width)
		sb.WriteString(errStyle.Render("▲ Error: " + sanitizeSingleLine(m.err.Error())))
		sb.WriteString("\x1b[K\n")
	}

	// Help bar
	sb.WriteString(m.helpBar())

	return sb.String()
}

func (m InteractiveModel) helpBar() string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#8CA1C1"))
	key := lipgloss.NewStyle().Foreground(lipgloss.Color("#38E1FF")).Bold(true)
	val := lipgloss.NewStyle().Foreground(lipgloss.Color("#F3F8FF"))

	var parts []string

	// Current product (compact label on narrow terminals)
	prodName := ProductLabel(m.product)
	if m.width > 0 && m.width < 105 {
		if idx := strings.IndexByte(prodName, ' '); idx > 0 {
			prodName = prodName[:idx]
		}
	}
	prodStr := key.Render("p") + dim.Render(":") + val.Render(prodName)
	parts = append(parts, prodStr)

	// Radius
	radLabel := fmt.Sprintf("%.0fkm", m.radius)
	if m.radius > 200 && (m.width == 0 || m.width >= 115) {
		sites := len(StationsWithinRadius(m.cfg.Loc.Lat, m.cfg.Loc.Lon, m.radius))
		if sites > 1 {
			radLabel = fmt.Sprintf("%.0fkm (mosaic · %d sites)", m.radius, sites)
		} else {
			radLabel = fmt.Sprintf("%.0fkm (mosaic)", m.radius)
		}
	}
	radStr := key.Render("+/-") + dim.Render(":") + val.Render(radLabel)
	parts = append(parts, radStr)
	if m.width == 0 || m.width >= 96 {
		parts = append(parts, key.Render("1-5")+dim.Render(":scale"))
	}

	// Loop status
	loopLabel := "off"
	if m.loopMode {
		loopLabel = "on"
		if len(m.frames) > 0 {
			loopLabel = fmt.Sprintf("%d/%d", m.frameIdx+1, len(m.frames))
		}
	}
	loopStr := key.Render("l") + dim.Render(":loop ") + val.Render(loopLabel)
	parts = append(parts, loopStr)

	if m.loopMode {
		parts = append(parts, key.Render("←→")+dim.Render(":step"))
		if m.paused {
			parts = append(parts, key.Render("␣")+dim.Render(":play"))
		} else {
			parts = append(parts, key.Render("␣")+dim.Render(":pause"))
		}
		if m.width == 0 || m.width >= 100 {
			speedMs := loopSpeeds[m.speedIdx] / time.Millisecond
			parts = append(parts, key.Render("<>")+dim.Render(fmt.Sprintf(":%dms", speedMs)))
		}
	}

	parts = append(parts, key.Render("r")+dim.Render(":refresh"))
	parts = append(parts, key.Render("q")+dim.Render(":quit"))

	sep := dim.Render(" │ ")
	if m.width == 0 || m.width >= 110 {
		sep = dim.Render("  │  ")
	}
	bar := strings.Join(parts, sep)
	if m.width > 0 && lipgloss.Width(bar) > m.width {
		bar = lipgloss.NewStyle().MaxWidth(m.width).Render(bar)
	}

	// Clear rest of line without wrapping.
	return bar + "\x1b[K"
}

func sanitizeSingleLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

// ── Commands ─────────────────────────────────────────────────────────────────

func (m InteractiveModel) fetchCurrent() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		opts := Options{Product: m.product, RadiusKM: m.radius}
		f, err := m.cfg.Provider.CurrentFrame(ctx, m.cfg.Loc, opts, m.cfg.Cache)
		if err != nil {
			return errMsg{err}
		}
		return frameMsg{f}
	}
}

func (m InteractiveModel) fetchLoop() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		opts := Options{Product: m.product, RadiusKM: m.radius}
		n := m.cfg.NumFrames
		if n < 1 {
			n = 6
		}
		frames, err := m.cfg.Provider.RecentFrames(ctx, m.cfg.Loc, opts, n, m.cfg.Cache)
		if err != nil {
			return errMsg{err}
		}
		return framesMsg{frames}
	}
}

func (m InteractiveModel) tickCmd() tea.Cmd {
	speed := 600 * time.Millisecond
	if m.speedIdx >= 0 && m.speedIdx < len(loopSpeeds) {
		speed = loopSpeeds[m.speedIdx]
	}
	return tea.Tick(speed, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}
