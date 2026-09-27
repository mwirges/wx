package output

import (
	"os"

	"golang.org/x/term"

	"github.com/mwirges/wx/internal/models"
)

// RenderOptions controls output formatting.
type RenderOptions struct {
	ForceJSON    bool
	Units        string // "imperial" or "metric"
	ShowForecast bool
	ShowAlerts   bool
	ShowHourly   bool
	HourlyLimit  int    // Number of hours to display in hourly forecast (default: 24)
	Short        bool   // Single-line compact output
	Template     string // Go text/template string or @path/to/template
}

// RenderData holds all data to be rendered.
type RenderData struct {
	Conditions *models.CurrentConditions
	Forecast   *models.Forecast  // nil if not requested
	Alerts     []models.Alert    // nil if not requested
}

var isTTY bool

func init() {
	isTTY = term.IsTerminal(int(os.Stdout.Fd()))
}

// Render dispatches to JSON, template, short, or pretty (TTY) output.
func Render(data RenderData, opts RenderOptions) error {
	if opts.ForceJSON {
		return renderJSON(data, opts)
	}
	if opts.Template != "" {
		return renderTemplate(data, opts)
	}
	if opts.Short {
		return renderShort(data, opts)
	}
	if !isTTY {
		return renderJSON(data, opts)
	}
	return renderPretty(data, opts)
}
