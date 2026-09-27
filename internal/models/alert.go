package models

import (
	"strings"
	"time"
)

// Alert represents an active NWS weather alert or special statement.
type Alert struct {
	ID          string // Unique alert ID
	Event       string // "Tornado Warning", "Winter Storm Watch", etc.
	Headline    string
	Description string
	Instruction string
	Severity    string // "Extreme", "Severe", "Moderate", "Minor", "Unknown"
	Urgency     string // "Immediate", "Expected", "Future", "Past", "Unknown"
	Effective   time.Time
	Expires     time.Time
	AreaDesc    string
}

// IsWarning returns true if the alert is a warning or has extreme/severe severity.
func (a Alert) IsWarning() bool {
	return strings.Contains(strings.ToLower(a.Event), "warning") ||
		a.Severity == "Extreme" || a.Severity == "Severe"
}

// IsWatch returns true if the alert is a watch.
func (a Alert) IsWatch() bool {
	return strings.Contains(strings.ToLower(a.Event), "watch")
}

// IsAdvisory returns true if the alert is an advisory or special statement.
func (a Alert) IsAdvisory() bool {
	return strings.Contains(strings.ToLower(a.Event), "advisory") ||
		strings.Contains(strings.ToLower(a.Event), "statement")
}
