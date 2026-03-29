// Package monitor provides a full-screen live weather TUI using bubbletea.
package monitor

import (
	"context"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mwirges/wx/internal/cache"
	"github.com/mwirges/wx/internal/location"
	"github.com/mwirges/wx/internal/models"
	"github.com/mwirges/wx/internal/provider"
	"github.com/mwirges/wx/internal/radar"
)

// ── Messages ──────────────────────────────────────────────────────────────────

type tickMsg time.Time

type weatherMsg struct {
	conditions *models.CurrentConditions
	forecast   *models.Forecast
	alerts     []models.Alert
	fetchedAt  time.Time
}

type radarMsg struct {
	lines []string
	err   error
}

type locationMsg struct {
	loc location.Location
}

type weatherErrMsg struct{ err error }
type locationErrMsg struct{ err error }

// ── Config ────────────────────────────────────────────────────────────────────

// MonitorConfig holds immutable configuration for the monitor TUI.
type MonitorConfig struct {
	WeatherProv     provider.WeatherProvider
	RadarProv       radar.Provider
	Cache           *cache.Cache
	Imperial        bool
	RefreshInterval time.Duration
}

// ── Model ─────────────────────────────────────────────────────────────────────

// MonitorModel is the bubbletea model for the live weather monitor.
type MonitorModel struct {
	cfg MonitorConfig

	// terminal dimensions
	width, height int

	// weather data
	conditions *models.CurrentConditions
	forecast   *models.Forecast
	alerts     []models.Alert
	lastFetch  time.Time
	fetchErr   error

	// radar
	radarVisible bool
	radarLines   []string
	radarErr     error

	// location
	loc       location.Location
	inputMode bool
	inputText string
	inputErr  error

	// forecast scroll
	forecastOffset  int
	forecastVisible int

	// loading flags
	weatherLoading bool
	radarLoading   bool

	quitting bool
}

// New creates the initial MonitorModel.
func New(cfg MonitorConfig, loc location.Location) MonitorModel {
	return MonitorModel{
		cfg:            cfg,
		loc:            loc,
		weatherLoading: true,
	}
}

// ── Init ──────────────────────────────────────────────────────────────────────

func (m MonitorModel) Init() tea.Cmd {
	return tea.Batch(
		fetchWeatherCmd(m.cfg, m.loc),
		tickCmd(m.cfg.RefreshInterval),
	)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m MonitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.forecastVisible = computeForecastVisible(m.height, alertCount(m.alerts))
		m.forecastOffset = clampOffset(m.forecastOffset, forecastLen(m.forecast), m.forecastVisible)
		return m, nil

	case tickMsg:
		m.weatherLoading = true
		cmds := []tea.Cmd{
			fetchWeatherCmd(m.cfg, m.loc),
			tickCmd(m.cfg.RefreshInterval),
		}
		if m.radarVisible {
			m.radarLoading = true
			cmds = append(cmds, fetchRadarCmd(m.cfg, m.loc, m.width, m.height))
		}
		return m, tea.Batch(cmds...)

	case weatherMsg:
		m.weatherLoading = false
		m.fetchErr = nil
		m.conditions = msg.conditions
		m.forecast = msg.forecast
		m.alerts = msg.alerts
		m.lastFetch = msg.fetchedAt
		m.forecastVisible = computeForecastVisible(m.height, alertCount(m.alerts))
		m.forecastOffset = clampOffset(m.forecastOffset, forecastLen(m.forecast), m.forecastVisible)
		return m, nil

	case weatherErrMsg:
		m.weatherLoading = false
		m.fetchErr = msg.err
		return m, nil

	case radarMsg:
		m.radarLoading = false
		if msg.err != nil {
			m.radarErr = msg.err
		} else {
			m.radarErr = nil
			m.radarLines = msg.lines
		}
		return m, nil

	case locationMsg:
		m.loc = msg.loc
		m.inputMode = false
		m.inputText = ""
		m.inputErr = nil
		m.forecastOffset = 0
		m.weatherLoading = true
		cmds := []tea.Cmd{fetchWeatherCmd(m.cfg, m.loc)}
		if m.radarVisible {
			m.radarLoading = true
			cmds = append(cmds, fetchRadarCmd(m.cfg, m.loc, m.width, m.height))
		}
		return m, tea.Batch(cmds...)

	case locationErrMsg:
		m.inputErr = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.inputMode {
			return m.handleInputKey(msg)
		}
		return m.handleKey(msg)
	}

	return m, nil
}

func (m MonitorModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "r":
		m.weatherLoading = true
		cmds := []tea.Cmd{fetchWeatherCmd(m.cfg, m.loc)}
		if m.radarVisible {
			m.radarLoading = true
			cmds = append(cmds, fetchRadarCmd(m.cfg, m.loc, m.width, m.height))
		}
		return m, tea.Batch(cmds...)

	case "R":
		m.radarVisible = !m.radarVisible
		if m.radarVisible && len(m.radarLines) == 0 {
			m.radarLoading = true
			return m, fetchRadarCmd(m.cfg, m.loc, m.width, m.height)
		}
		return m, nil

	case "l":
		m.inputMode = true
		m.inputText = ""
		m.inputErr = nil
		return m, nil

	case "up", "k":
		if m.forecastOffset > 0 {
			m.forecastOffset--
		}
		return m, nil

	case "down", "j":
		max := forecastLen(m.forecast) - m.forecastVisible
		if max < 0 {
			max = 0
		}
		if m.forecastOffset < max {
			m.forecastOffset++
		}
		return m, nil
	}

	return m, nil
}

func (m MonitorModel) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputMode = false
		m.inputText = ""
		m.inputErr = nil
		return m, nil

	case "enter":
		if strings.TrimSpace(m.inputText) == "" {
			return m, nil
		}
		return m, resolveLocationCmd(m.cfg.Cache, m.inputText)

	case "backspace":
		if len(m.inputText) > 0 {
			runes := []rune(m.inputText)
			m.inputText = string(runes[:len(runes)-1])
		}
		return m, nil

	default:
		if len(msg.Runes) > 0 {
			m.inputText += string(msg.Runes)
		}
		return m, nil
	}
}

// ── Async commands ────────────────────────────────────────────────────────────

func fetchWeatherCmd(cfg MonitorConfig, loc location.Location) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var (
			cond     *models.CurrentConditions
			fc       *models.Forecast
			alerts   []models.Alert
			condErr  error
			wg       sync.WaitGroup
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			cond, condErr = cfg.WeatherProv.CurrentConditions(ctx, loc, cfg.Cache)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			fc, _ = cfg.WeatherProv.Forecast(ctx, loc, false, cfg.Cache)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			alerts, _ = cfg.WeatherProv.Alerts(ctx, loc, cfg.Cache)
		}()

		wg.Wait()

		if condErr != nil {
			return weatherErrMsg{condErr}
		}
		return weatherMsg{
			conditions: cond,
			forecast:   fc,
			alerts:     alerts,
			fetchedAt:  time.Now(),
		}
	}
}

func fetchRadarCmd(cfg MonitorConfig, loc location.Location, termW, termH int) tea.Cmd {
	return func() tea.Msg {
		if cfg.RadarProv == nil {
			return radarMsg{err: nil, lines: nil}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		radarW := termW / 2
		radarH := termH - headerLines - helpBarLines
		if radarW < 10 || radarH < 5 {
			return radarMsg{err: nil, lines: nil}
		}

		opts := radar.DefaultOptions()
		frame, err := cfg.RadarProv.CurrentFrame(ctx, loc, opts, cfg.Cache)
		if err != nil {
			return radarMsg{err: err}
		}

		var sb strings.Builder
		renderOpts := radar.RenderOptions{
			TermWidth:  radarW,
			TermHeight: radarH,
			Mode:       radar.TermHalfBlock, // inline image protocols don't work in a split TUI layout
		}
		if err := radar.RenderFrame(&sb, frame, loc.DisplayName, renderOpts); err != nil {
			return radarMsg{err: err}
		}

		lines := splitAndStripRadarLines(sb.String())
		return radarMsg{lines: lines}
	}
}

func resolveLocationCmd(ch *cache.Cache, input string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		loc, err := location.Resolve(ctx, input, ch)
		if err != nil {
			return locationErrMsg{err}
		}
		return locationMsg{loc}
	}
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func forecastLen(fc *models.Forecast) int {
	if fc == nil {
		return 0
	}
	return len(fc.Periods)
}

func alertCount(alerts []models.Alert) int {
	n := len(alerts)
	if n > maxAlertLines {
		return maxAlertLines
	}
	return n
}

// clampOffset ensures offset stays within valid range.
func clampOffset(offset, total, visible int) int {
	max := total - visible
	if max < 0 {
		max = 0
	}
	if offset > max {
		return max
	}
	if offset < 0 {
		return 0
	}
	return offset
}

// splitAndStripRadarLines splits radar output on newlines and strips
// the trailing \x1b[K (erase to end of line) that RenderFrame appends.
func splitAndStripRadarLines(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		out = append(out, strings.TrimSuffix(line, "\x1b[K"))
	}
	return out
}

// computeForecastVisible returns how many forecast periods fit given the
// terminal height, after reserving space for chrome and the conditions block.
func computeForecastVisible(termH, numAlerts int) int {
	reserved := headerLines + helpBarLines + conditionsBlockLines + numAlerts + 3 // spacers + forecast header
	available := termH - reserved
	if available < 0 {
		return 0
	}
	if available > maxForecastPeriods {
		return maxForecastPeriods
	}
	return available
}
