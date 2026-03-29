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

var radiusPresets = []float64{50, 100, 150, 200, 300, 500}

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
		cfg:     cfg,
		product: cfg.Product,
		radius:  cfg.RadiusKM,
	}
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
			m.status = fmt.Sprintf("Zooming in (%.0f km)…", m.radius)
			if m.loopMode {
				return m, m.fetchLoop()
			}
			return m, m.fetchCurrent()

		case "-":
			m.radius = nextRadius(m.radius, 1) // larger radius = zoom out
			m.frames = nil
			m.frame = nil
			m.loading = true
			m.status = fmt.Sprintf("Zooming out (%.0f km)…", m.radius)
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

	// Reserve 1 line for the help bar at the bottom.
	renderH := m.height - 1

	// Pick which frame to show.
	var activeFrame *Frame
	if m.loopMode && len(m.frames) > 0 {
		activeFrame = m.frames[m.frameIdx]
	} else {
		activeFrame = m.frame
	}

	if m.loading || activeFrame == nil {
		msg := m.status
		if msg == "" {
			msg = "Loading…"
		}
		// Center the loading message.
		padY := renderH / 2
		for i := 0; i < padY; i++ {
			sb.WriteString("\n")
		}
		padX := (m.width - len(msg)) / 2
		if padX < 0 {
			padX = 0
		}
		sb.WriteString(strings.Repeat(" ", padX))
		sb.WriteString(msg)
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
		_ = RenderFrame(&sb, activeFrame, m.cfg.Loc.DisplayName, opts)
	}

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		sb.WriteString(errStyle.Render("Error: " + m.err.Error()))
	}

	// Help bar
	sb.WriteString(m.helpBar())

	return sb.String()
}

func (m InteractiveModel) helpBar() string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	key := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	val := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var parts []string

	// Current product
	prodStr := key.Render("p") + dim.Render(":") + val.Render(ProductLabel(m.product))
	parts = append(parts, prodStr)

	// Radius
	radStr := key.Render("+/-") + dim.Render(":") + val.Render(fmt.Sprintf("%.0fkm", m.radius))
	parts = append(parts, radStr)

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
			parts = append(parts, key.Render("space")+dim.Render(":play"))
		} else {
			parts = append(parts, key.Render("space")+dim.Render(":pause"))
		}
	}

	parts = append(parts, key.Render("r")+dim.Render(":refresh"))
	parts = append(parts, key.Render("q")+dim.Render(":quit"))

	bar := strings.Join(parts, dim.Render("  │  "))

	// Pad/truncate to terminal width and clear rest of line.
	return bar + "\x1b[K"
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
	return tea.Tick(600*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}
