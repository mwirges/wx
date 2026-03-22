package output

import "github.com/charmbracelet/lipgloss"

// iconWidth is the visual width each icon line is padded to via lipgloss.
const iconWidth = 13

// weatherIcon holds 5 lines of ASCII art with per-line foreground colors.
type weatherIcon struct {
	lines  [5]string
	colors [5]lipgloss.Color
}

// Named colors used across icons.
const (
	clrYellow   = lipgloss.Color("226") // sun
	clrMoon     = lipgloss.Color("189") // moon / clear night
	clrGray     = lipgloss.Color("250") // clouds
	clrDimGray  = lipgloss.Color("244") // fog / dim elements
	clrBlue     = lipgloss.Color("39")  // rain
	clrDarkBlue = lipgloss.Color("27")  // heavy rain
	clrCyan     = lipgloss.Color("159") // snow / sleet
)

// conditionIcons maps normalized condition codes to ASCII weather icons.
// Each line is ≤11 visible characters; lipgloss pads to iconWidth on render.
var conditionIcons = map[string]weatherIcon{
	"clear-day": {
		lines:  [5]string{`   \   /   `, `    .-.    `, `  -(   )-  `, "    `-'    ", `   /   \   `},
		colors: [5]lipgloss.Color{clrYellow, clrYellow, clrYellow, clrYellow, clrYellow},
	},
	"clear-night": {
		lines:  [5]string{`           `, `    .-.    `, `   (   |   `, "    `- '   ", `           `},
		colors: [5]lipgloss.Color{clrMoon, clrMoon, clrMoon, clrMoon, clrMoon},
	},
	"partly-cloudy-day": {
		lines:  [5]string{`  \  /     `, `_ /"".--.  `, `  \_(  ).  `, ` (___(__)  `, `           `},
		colors: [5]lipgloss.Color{clrYellow, clrYellow, clrGray, clrGray, clrGray},
	},
	"partly-cloudy-night": {
		lines:  [5]string{`           `, `    .-.    `, `   (   |   `, " .--`- '   ", `(    ).    `},
		colors: [5]lipgloss.Color{clrMoon, clrMoon, clrMoon, clrGray, clrGray},
	},
	"cloudy": {
		lines:  [5]string{`           `, `   .--.    `, ` .-(    ). `, `(___.__)_) `, `           `},
		colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrGray, clrGray},
	},
	"rain": {
		lines:  [5]string{`   .--.    `, ` .-(    ). `, `(___.__)_) `, ` ' ' ' '   `, `           `},
		colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrBlue, clrBlue},
	},
	"heavy-rain": {
		lines:  [5]string{`   .--.    `, ` .-(    ). `, `(___.__)_) `, `'' '' ''   `, `'' '' ''   `},
		colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrDarkBlue, clrDarkBlue},
	},
	"snow": {
		lines:  [5]string{`   .--.    `, ` .-(    ). `, `(___.__)_) `, ` * * * *   `, `  * * *    `},
		colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrCyan, clrCyan},
	},
	"sleet": {
		lines:  [5]string{`   .--.    `, ` .-(    ). `, `(___.__)_) `, ` '* '* '*  `, `           `},
		colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrCyan, clrCyan},
	},
	"thunder": {
		lines:  [5]string{`   .--.    `, ` .-(    ). `, `(___.__)_) `, `  / ' '    `, ` /         `},
		colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrYellow, clrYellow},
	},
	"fog": {
		lines:  [5]string{`           `, `_ - _ - _  `, ` _ - _ -   `, `_ - _ - _  `, `           `},
		colors: [5]lipgloss.Color{clrDimGray, clrDimGray, clrDimGray, clrDimGray, clrDimGray},
	},
	"wind": {
		lines:  [5]string{`           `, `  ~  ~  ~  `, ` ~  ~  ~   `, `  ~  ~  ~  `, `           `},
		colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrGray, clrGray},
	},
}

// getIcon returns the weatherIcon for code, or a blank icon if code is unknown.
func getIcon(code string) weatherIcon {
	if ic, ok := conditionIcons[code]; ok {
		return ic
	}
	var blank weatherIcon
	for i := range blank.lines {
		blank.lines[i] = `           `
		blank.colors[i] = clrGray
	}
	return blank
}
