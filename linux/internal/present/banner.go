package present

import (
	"sort"
	"strings"
)

// BannerKind is the pinned desk banner. Event text decides it.
// "Warning" and "Emergency" outrank "Watch". Advisories and statements
// stay in the alerts list. Severity is only a tie-break, and this does
// not feed the menu-bar pip.
type BannerKind string

const (
	BannerWarning BannerKind = "warning"
	BannerWatch   BannerKind = "watch"
)

// BannerAlert is one warning or watch in banner order.
type BannerAlert struct {
	Alert map[string]any
	Kind  BannerKind
	Index int
}

// DetailField is one labeled line in the alert popover.
type DetailField struct {
	Label string
	Value string
}

// BannerKindOf classifies an alert from its event name alone.
func BannerKindOf(alert map[string]any) (BannerKind, bool) {
	if alert == nil {
		return "", false
	}
	event := strings.ToLower(AsString(alert["event"]))
	switch {
	case strings.Contains(event, "warning"), strings.Contains(event, "emergency"):
		return BannerWarning, true
	case strings.Contains(event, "watch"):
		return BannerWatch, true
	default:
		return "", false
	}
}

// AlertMaps returns every alert object on the payload.
func AlertMaps(payload map[string]any) []map[string]any {
	raw := Alerts(payload)
	if len(raw) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if alert := Map(item); alert != nil {
			out = append(out, alert)
		}
	}
	return out
}

// RankedBannerAlerts returns warnings, then watches. Advisories are left out.
// Within a kind, NWS severity orders them, then the original index.
func RankedBannerAlerts(payload map[string]any) []BannerAlert {
	var ranked []BannerAlert
	for i, alert := range AlertMaps(payload) {
		kind, ok := BannerKindOf(alert)
		if !ok {
			continue
		}
		ranked = append(ranked, BannerAlert{Alert: alert, Kind: kind, Index: i})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if rank := kindRank(ranked[i].Kind) - kindRank(ranked[j].Kind); rank != 0 {
			return rank < 0
		}
		if rank := severityRank(AsString(ranked[i].Alert["severity"])) - severityRank(AsString(ranked[j].Alert["severity"])); rank != 0 {
			return rank < 0
		}
		return ranked[i].Index < ranked[j].Index
	})
	return ranked
}

func kindRank(kind BannerKind) int {
	if kind == BannerWarning {
		return 0
	}
	return 1
}

func severityRank(severity string) int {
	switch strings.ToLower(severity) {
	case "extreme":
		return 0
	case "severe":
		return 1
	case "moderate":
		return 2
	case "minor":
		return 3
	default:
		return 4
	}
}

// BannerExpiry is the Mac abbreviated local stamp. A non-timestamp is kept as written.
func BannerExpiry(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if text := formatStamp(raw, nil, "Jan 2, 3:04 PM"); text != "" {
		return text
	}
	return raw
}

// AlertDetailFields is the popover body. Empty fields are omitted.
// Expires uses BannerExpiry when the value is a timestamp.
func AlertDetailFields(alert map[string]any) []DetailField {
	if alert == nil {
		return nil
	}
	var fields []DetailField
	add := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		fields = append(fields, DetailField{Label: label, Value: value})
	}
	add("HEADLINE", AsString(alert["headline"]))
	add("AREA", AsString(alert["area"]))
	add("EXPIRES", BannerExpiry(AsString(alert["expires"])))
	add("DESCRIPTION", AsString(alert["description"]))
	add("INSTRUCTION", AsString(alert["instruction"]))
	return fields
}
