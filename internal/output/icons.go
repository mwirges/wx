package output

import "github.com/charmbracelet/lipgloss"

// IconWidth is the visual width each icon line is padded to via lipgloss.
const IconWidth = 13

// WeatherIcon holds 5 lines of ASCII art with per-line foreground colors.
type WeatherIcon struct {
	Lines  [5]string
	Colors [5]lipgloss.Color
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
// Each line is ≤11 visible characters; lipgloss pads to IconWidth on render.
var conditionIcons = map[string]WeatherIcon{
	"clear-day": {
		Lines:  [5]string{`   \   /   `, `    .-.    `, `  -(   )-  `, "    `-'    ", `   /   \   `},
		Colors: [5]lipgloss.Color{clrYellow, clrYellow, clrYellow, clrYellow, clrYellow},
	},
	"clear-night": {
		Lines:  [5]string{`           `, `    .-.    `, `   (   |   `, "    `- '   ", `           `},
		Colors: [5]lipgloss.Color{clrMoon, clrMoon, clrMoon, clrMoon, clrMoon},
	},
	"partly-cloudy-day": {
		Lines:  [5]string{`  \  /     `, `_ /"".--.  `, `  \_(  ).  `, ` (___(__)  `, `           `},
		Colors: [5]lipgloss.Color{clrYellow, clrYellow, clrGray, clrGray, clrGray},
	},
	"partly-cloudy-night": {
		Lines:  [5]string{`           `, `    .-.    `, `   (   |   `, " .--`- '   ", `(    ).    `},
		Colors: [5]lipgloss.Color{clrMoon, clrMoon, clrMoon, clrGray, clrGray},
	},
	"cloudy": {
		Lines:  [5]string{`           `, `   .--.    `, ` .-(    ). `, `(___.__)_) `, `           `},
		Colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrGray, clrGray},
	},
	"rain": {
		Lines:  [5]string{`   .--.    `, ` .-(    ). `, `(___.__)_) `, ` ' ' ' '   `, `           `},
		Colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrBlue, clrBlue},
	},
	"heavy-rain": {
		Lines:  [5]string{`   .--.    `, ` .-(    ). `, `(___.__)_) `, `'' '' ''   `, `'' '' ''   `},
		Colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrDarkBlue, clrDarkBlue},
	},
	"snow": {
		Lines:  [5]string{`   .--.    `, ` .-(    ). `, `(___.__)_) `, ` * * * *   `, `  * * *    `},
		Colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrCyan, clrCyan},
	},
	"sleet": {
		Lines:  [5]string{`   .--.    `, ` .-(    ). `, `(___.__)_) `, ` '* '* '*  `, `           `},
		Colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrCyan, clrCyan},
	},
	"thunder": {
		Lines:  [5]string{`   .--.    `, ` .-(    ). `, `(___.__)_) `, `  / ' '    `, ` /         `},
		Colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrYellow, clrYellow},
	},
	"fog": {
		Lines:  [5]string{`           `, `_ - _ - _  `, ` _ - _ -   `, `_ - _ - _  `, `           `},
		Colors: [5]lipgloss.Color{clrDimGray, clrDimGray, clrDimGray, clrDimGray, clrDimGray},
	},
	"wind": {
		Lines:  [5]string{`           `, `  ~  ~  ~  `, ` ~  ~  ~   `, `  ~  ~  ~  `, `           `},
		Colors: [5]lipgloss.Color{clrGray, clrGray, clrGray, clrGray, clrGray},
	},
}

// GetIcon returns the WeatherIcon for code, or a blank icon if code is unknown.
func GetIcon(code string) WeatherIcon {
	if ic, ok := conditionIcons[code]; ok {
		return ic
	}
	var blank WeatherIcon
	for i := range blank.Lines {
		blank.Lines[i] = `           `
		blank.Colors[i] = clrGray
	}
	return blank
}
