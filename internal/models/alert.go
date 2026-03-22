package models

import "time"

// Alert represents an active NWS weather alert or special statement.
type Alert struct {
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
