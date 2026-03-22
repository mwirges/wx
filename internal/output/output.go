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

// Render dispatches to pretty (TTY) or JSON output.
func Render(data RenderData, opts RenderOptions) error {
	if opts.ForceJSON || !isTTY {
		return renderJSON(data, opts)
	}
	return renderPretty(data, opts)
}
