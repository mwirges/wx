// Package present formats CLI JSON for the desk and the tray.
// Nothing here forecasts. Strings match the Mac menu bar.
package present

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// Formats are the Mac menu-bar formats, in menu order.
var Formats = []string{"compact", "standard", "tactical"}

// FormatLabels are the tray radio titles.
var FormatLabels = map[string]string{
	"compact":  "Compact (72°)",
	"standard": "Standard (☀️ 72°)",
	"tactical": "Tactical ([FWA] 72° ↘12mph)",
}

// ConditionIcons are Freedesktop names for the same condition_code the Mac shell maps to SF Symbols.
var ConditionIcons = map[string]string{
	"clear-day":           "weather-clear",
	"clear-night":         "weather-clear-night",
	"partly-cloudy-day":   "weather-few-clouds",
	"partly-cloudy-night": "weather-few-clouds-night",
	"cloudy":              "weather-overcast",
	"rain":                "weather-showers",
	"heavy-rain":          "weather-showers",
	"snow":                "weather-snow",
	"sleet":               "weather-snow-rain",
	"thunder":             "weather-storm",
	"fog":                 "weather-fog",
	"wind":                "weather-windy",
}

// Item is one DBus menu row. Root is the invisible submenu root.
type Item struct {
	Label     string
	Enabled   bool
	Action    string
	Target    string
	Toggle    *bool
	Separator bool
	Submenu   bool
	Root      bool
	Children  []Item
}

// Props is the DBus menu property map.
func (it Item) Props() map[string]any {
	if it.Separator {
		return map[string]any{"type": "separator"}
	}
	if it.Root {
		return map[string]any{"children-display": "submenu"}
	}
	props := map[string]any{"label": it.Label, "enabled": it.Enabled}
	if it.Submenu {
		props["children-display"] = "submenu"
	}
	if it.Toggle != nil {
		props["toggle-type"] = "radio"
		if *it.Toggle {
			props["toggle-state"] = 1
		} else {
			props["toggle-state"] = 0
		}
	}
	return props
}

// MenuLabels walks labels in menu order.
func MenuLabels(item Item) []string {
	var found []string
	if item.Label != "" {
		found = append(found, item.Label)
	}
	for _, child := range item.Children {
		found = append(found, MenuLabels(child)...)
	}
	return found
}

// Conditions returns the conditions object, or an empty map.
func Conditions(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	cond, _ := payload["conditions"].(map[string]any)
	if cond == nil {
		return map[string]any{}
	}
	return cond
}

// Alerts returns the alerts array, or nil.
func Alerts(payload map[string]any) []any {
	if payload == nil {
		return nil
	}
	raw, _ := payload["alerts"].([]any)
	return raw
}

// IsWarning is true for a warning event or an extreme/severe severity.
func IsWarning(alert map[string]any) bool {
	event := strings.ToLower(AsString(alert["event"]))
	severity := strings.ToLower(AsString(alert["severity"]))
	return strings.Contains(event, "warning") || severity == "extreme" || severity == "severe"
}

// IsWatch is true when the event name contains "watch".
func IsWatch(alert map[string]any) bool {
	return strings.Contains(strings.ToLower(AsString(alert["event"])), "watch")
}

// IsAdvisory is true for an advisory or a statement.
func IsAdvisory(alert map[string]any) bool {
	event := strings.ToLower(AsString(alert["event"]))
	return strings.Contains(event, "advisory") || strings.Contains(event, "statement")
}

// PipKind is "warning", "watch", or "". A warning wins. The pip does not blink.
func PipKind(payload map[string]any) string {
	for _, raw := range Alerts(payload) {
		alert, ok := raw.(map[string]any)
		if ok && IsWarning(alert) {
			return "warning"
		}
	}
	for _, raw := range Alerts(payload) {
		alert, ok := raw.(map[string]any)
		if ok && (IsWatch(alert) || IsAdvisory(alert)) {
			return "watch"
		}
	}
	return ""
}

// PipGlyph is the steady menu-bar pip, including its trailing space.
func PipGlyph(payload map[string]any) string {
	switch PipKind(payload) {
	case "warning":
		return "🔴 "
	case "watch":
		return "🟠 "
	default:
		return ""
	}
}

// DisplayTemp is the menu-bar temperature. Missing data is "--".
func DisplayTemp(payload map[string]any, units string) string {
	cond := Conditions(payload)
	var value any
	if units == "metric" {
		value = cond["temperature_c"]
	} else {
		value = cond["temperature_f"]
	}
	f, ok := AsFloat(value)
	if !ok {
		return "--"
	}
	return fmt.Sprintf("%.0f°", f)
}

// ConditionIcon is the Freedesktop icon for standard format.
func ConditionIcon(payload map[string]any) string {
	code := strings.ToLower(AsString(Conditions(payload)["condition_code"]))
	if icon, ok := ConditionIcons[code]; ok {
		return icon
	}
	return "weather-few-clouds"
}

// WindArrow points where the wind blows. Same table as the Mac shell.
func WindArrow(direction string) string {
	if direction == "" {
		return ""
	}
	return map[string]string{
		"N": "↓", "NNE": "↙", "NE": "↙", "ENE": "←",
		"E": "←", "ESE": "↖", "SE": "↖", "SSE": "↑",
		"S": "↑", "SSW": "↗", "SW": "↗", "WSW": "→",
		"W": "→", "WNW": "↘", "NW": "↘", "NNW": "↓",
	}[strings.ToUpper(strings.TrimSpace(direction))]
}

// TacticalStationTag is the bracketed station in the tactical title.
func TacticalStationTag(payload map[string]any) string {
	cond := Conditions(payload)
	station := strings.ToUpper(AsString(cond["station"]))
	if station != "" {
		if len(station) == 4 && strings.HasPrefix(station, "K") {
			return station[1:]
		}
		if len(station) <= 4 && station != "OPENMETEO" {
			return station
		}
	}
	location := AsString(cond["location"])
	if location != "" {
		first := strings.Fields(location)[0]
		var letters []rune
		for _, r := range first {
			if unicode.IsLetter(r) {
				letters = append(letters, r)
				if len(letters) == 3 {
					break
				}
			}
		}
		if len(letters) > 0 {
			return strings.ToUpper(string(letters))
		}
	}
	return "WX"
}

// TacticalWind is the arrow plus the speed, or "".
func TacticalWind(payload map[string]any, units string) string {
	cond := Conditions(payload)
	var text string
	if units == "metric" {
		if speed, ok := AsFloat(cond["wind_kph"]); ok {
			text = fmt.Sprintf("%.0fkm/h", speed)
		}
	} else if speed, ok := AsFloat(cond["wind_mph"]); ok {
		text = fmt.Sprintf("%.0fmph", speed)
	}
	if text == "" {
		return ""
	}
	return WindArrow(AsString(cond["wind_direction"])) + text
}

// TacticalStatus is "[TAG] temp" plus wind when the station reported it.
func TacticalStatus(payload map[string]any, units string) string {
	tag := TacticalStationTag(payload)
	temp := DisplayTemp(payload, units)
	wind := TacticalWind(payload, units)
	if wind == "" {
		return fmt.Sprintf("[%s] %s", tag, temp)
	}
	return fmt.Sprintf("[%s] %s %s", tag, temp, wind)
}

// StatusText is the menu-bar title. It matches StatusItemController.refreshButton.
func StatusText(payload map[string]any, units, format, errText string) string {
	if errText != "" && len(Conditions(payload)) == 0 {
		return " wx?"
	}
	pip := PipGlyph(payload)
	switch format {
	case "compact":
		return pip + DisplayTemp(payload, units)
	case "tactical":
		return pip + TacticalStatus(payload, units)
	default:
		if pip != "" {
			return " " + pip + DisplayTemp(payload, units)
		}
		return " " + DisplayTemp(payload, units)
	}
}

// PixmapText is the status text without the emoji pip. The pixmap draws a steady dot.
func PixmapText(payload map[string]any, units, format, errText string) string {
	if errText != "" && len(Conditions(payload)) == 0 {
		return "wx?"
	}
	switch format {
	case "compact":
		return DisplayTemp(payload, units)
	case "tactical":
		return TacticalStatus(payload, units)
	default:
		return DisplayTemp(payload, units)
	}
}

// ParseISO parses an RFC3339 timestamp from the CLI.
func ParseISO(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if strings.HasSuffix(raw, "Z") {
		raw = raw[:len(raw)-1] + "+00:00"
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, raw)
	}
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// HourLabel is a local "3 PM" label, or the period name.
func HourLabel(period map[string]any) string {
	stamp, ok := ParseISO(AsString(period["start_time"]))
	if !ok {
		return AsString(period["name"])
	}
	local := stamp.In(time.Local)
	return local.Format("3") + " " + local.Format("PM")
}

// PeriodTemp formats one forecast period.
func PeriodTemp(period map[string]any, units string) string {
	if units == "metric" {
		if v, ok := AsFloat(period["temperature_c"]); ok {
			return fmt.Sprintf("%.0f°", v)
		}
	}
	if v, ok := AsFloat(period["temperature_f"]); ok {
		return fmt.Sprintf("%.0f°", v)
	}
	return "—"
}

// NowcastOf finds the nowcast object on a separate payload, the conditions, or the root.
func NowcastOf(payload, separate map[string]any) map[string]any {
	if separate != nil {
		if nc, ok := separate["nowcast"].(map[string]any); ok {
			return nc
		}
	}
	cond := Conditions(payload)
	if nc, ok := cond["nowcast"].(map[string]any); ok {
		return nc
	}
	if payload != nil {
		if nc, ok := payload["nowcast"].(map[string]any); ok {
			return nc
		}
	}
	return nil
}

// NowcastBadge is the nowcast kicker.
func NowcastBadge(nowcast map[string]any) string {
	if AsBool(nowcast["is_active_precip"]) {
		return "ACTIVE PRECIPITATION"
	}
	if present(nowcast["next_precip_time"]) {
		return "PRECIPITATION EXPECTED"
	}
	return "ALL CLEAR // DRY"
}

func present(v any) bool {
	if v == nil {
		return false
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) != ""
	}
	return true
}

// TrayMenu is the nested menu matching the Mac status item.
func TrayMenu(location, temp string, favorites []map[string]any, menuFormat string, cards []map[string]string) Item {
	var children []Item
	if location != "" {
		children = append(children,
			item(fmt.Sprintf("ACTIVE // %s (%s)", strings.ToUpper(location), temp), false, "", "", nil),
			sep(),
		)
	}
	children = append(children, item("PINNED LOCATIONS // COMMAND GRID", false, "", "", nil))
	shown := cards
	useFavorites := cards == nil
	if useFavorites {
		if len(favorites) == 0 {
			children = append(children, item("  No pinned locations (add in Command Grid)", false, "", "", nil))
		} else {
			for _, fav := range favorites {
				name := AsString(fav["name"])
				value := AsString(fav["value"])
				if name == "" {
					name = value
				}
				if value == "" {
					value = name
				}
				children = append(children, item(name, true, "location", value, nil))
			}
		}
	} else if len(shown) == 0 {
		children = append(children, item("  No pinned locations (add in Command Grid)", false, "", "", nil))
	} else {
		for _, card := range shown {
			label := card["label"]
			target := card["target"]
			if target == "" {
				target = label
			}
			children = append(children, item(label, true, "location", target, nil))
		}
	}
	children = append(children, item("Manage Command Grid...", true, "grid", "", nil), sep())
	var formats []Item
	for _, key := range Formats {
		on := key == menuFormat
		formats = append(formats, item(FormatLabels[key], true, "format", key, &on))
	}
	children = append(children,
		Item{Label: "Menu Bar Format", Enabled: true, Submenu: true, Children: formats},
		sep(),
		item("Open Desk Window", true, "open-desk", "", nil),
		item("Refresh Telemetry", true, "refresh", "", nil),
		sep(),
		item("Quit wx", true, "quit", "", nil),
	)
	return Item{Root: true, Children: children}
}

func item(label string, enabled bool, action, target string, toggle *bool) Item {
	return Item{Label: label, Enabled: enabled, Action: action, Target: target, Toggle: toggle}
}

func sep() Item { return Item{Separator: true} }

// AsString stringifies a JSON value. Nil is empty.
func AsString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

// AsFloat reads a JSON number.
func AsFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case interface{ Float64() (float64, error) }:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// AsBool reads a JSON bool.
func AsBool(v any) bool {
	b, _ := v.(bool)
	return b
}

// Map reads a JSON object.
func Map(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// Slice reads a JSON array.
func Slice(v any) []any {
	s, _ := v.([]any)
	return s
}
