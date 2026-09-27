package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// TemplateContext is the root data object passed to Go text/templates.
type TemplateContext struct {
	Conditions *TemplateConditions
	Forecast   *TemplateForecast
	Alerts     []TemplateAlert
	Now        time.Time
	Units      string
	Imperial   bool
}

// TemplateConditions provides clean, dereferenced weather fields and strings.
type TemplateConditions struct {
	Location      string
	StationID     string
	StationName   string
	ObservedAt    time.Time
	Description   string
	ConditionCode string

	Temp          float64
	TempStr       string
	TempF         float64
	TempC         float64

	FeelsLike     *float64
	FeelsLikeStr  string
	FeelsLikeF    *float64
	FeelsLikeC    *float64

	DewPoint      *float64
	DewPointStr   string
	DewPointF     *float64
	DewPointC     *float64

	Humidity      *float64
	HumidityStr   string

	WindSpeed     *float64
	WindSpeedStr  string
	WindMph       *float64
	WindKph       *float64
	WindDirection string
	WindDegrees   *float64
	WindGust      *float64
	WindGustStr   string

	Pressure      *float64
	PressureStr   string
	PressureInhg  *float64
	PressureHpa   *float64

	Visibility    *float64
	VisibilityStr string
	VisibilityMi  *float64
	VisibilityKm  *float64

	Astronomy     *TemplateAstronomy
}

// TemplateAstronomy provides calculated solar times.
type TemplateAstronomy struct {
	Sunrise          string
	Sunset           string
	SolarNoon        string
	DayLength        string
	DayLengthSeconds int64
	IsPolarDay       bool
	IsPolarNight     bool
}

// TemplateForecast provides structured forecast information.
type TemplateForecast struct {
	GeneratedAt time.Time
	Periods     []TemplatePeriod
}

// TemplatePeriod represents a forecast time slice.
type TemplatePeriod struct {
	Name                       string
	StartTime                  time.Time
	EndTime                    time.Time
	IsDaytime                  bool
	Temp                       float64
	TempStr                    string
	TempF                      float64
	TempC                      float64
	WindSpeed                  float64
	WindSpeedStr               string
	WindDir                    string
	ShortDesc                  string
	DetailedDesc               string
	ProbabilityOfPrecipitation *float64
	PoP                        float64
	PoPStr                     string
	HumidityPct                *float64
	DewPoint                   *float64
	DewPointStr                string
}

// TemplateAlert represents an active weather advisory or warning.
type TemplateAlert struct {
	ID          string
	Event       string
	Severity    string
	Headline    string
	Description string
	Instruction string
	Expires     time.Time
	ExpiresStr  string
}

// BuildTemplateContext converts raw RenderData into a developer-friendly TemplateContext.
func BuildTemplateContext(data RenderData, opts RenderOptions) TemplateContext {
	imperial := opts.Units != "metric"
	units := opts.Units
	if units == "" {
		if imperial {
			units = "imperial"
		} else {
			units = "metric"
		}
	}

	ctx := TemplateContext{
		Now:      time.Now(),
		Units:    units,
		Imperial: imperial,
	}

	if c := data.Conditions; c != nil {
		tc := &TemplateConditions{
			Location:      c.Location,
			StationID:     c.StationID,
			StationName:   c.StationName,
			ObservedAt:    c.ObservedAt,
			Description:   c.Description,
			ConditionCode: c.ConditionCode,
		}

		if c.TempC != nil {
			tc.TempC = *c.TempC
			tc.TempF = CelsiusToFahrenheit(*c.TempC)
			if imperial {
				tc.Temp = tc.TempF
			} else {
				tc.Temp = tc.TempC
			}
			tc.TempStr = FormatTemp(*c.TempC, imperial)
		}

		if fl := FeelsLikeTemp(c.WindChillC, c.HeatIndexC); fl != nil {
			flC := *fl
			flF := CelsiusToFahrenheit(flC)
			tc.FeelsLikeC = &flC
			tc.FeelsLikeF = &flF
			if imperial {
				tc.FeelsLike = &flF
			} else {
				tc.FeelsLike = &flC
			}
			tc.FeelsLikeStr = FormatTemp(*fl, imperial)
		}

		if c.DewPointC != nil {
			dpC := *c.DewPointC
			dpF := CelsiusToFahrenheit(dpC)
			tc.DewPointC = &dpC
			tc.DewPointF = &dpF
			if imperial {
				tc.DewPoint = &dpF
			} else {
				tc.DewPoint = &dpC
			}
			tc.DewPointStr = FormatTemp(*c.DewPointC, imperial)
		}

		if c.HumidityPct != nil {
			h := *c.HumidityPct
			tc.Humidity = &h
			tc.HumidityStr = fmt.Sprintf("%.0f%%", h)
		}

		if c.WindKPH != nil {
			wKph := *c.WindKPH
			wMph := KphToMPH(wKph)
			tc.WindKph = &wKph
			tc.WindMph = &wMph
			if imperial {
				tc.WindSpeed = &wMph
			} else {
				tc.WindSpeed = &wKph
			}
			tc.WindSpeedStr = FormatWindSpeed(wKph, imperial)
		}
		if c.WindDegrees != nil {
			deg := *c.WindDegrees
			tc.WindDegrees = &deg
			tc.WindDirection = DegreesToCompass(deg)
		}
		if c.WindGustKPH != nil {
			gKph := *c.WindGustKPH
			gMph := KphToMPH(gKph)
			if imperial {
				tc.WindGust = &gMph
			} else {
				tc.WindGust = &gKph
			}
			tc.WindGustStr = FormatWindSpeed(gKph, imperial)
		}

		if c.PressureHPA != nil {
			pHpa := *c.PressureHPA
			pInhg := pHpa / 33.8639
			tc.PressureHpa = &pHpa
			tc.PressureInhg = &pInhg
			if imperial {
				tc.Pressure = &pInhg
			} else {
				tc.Pressure = &pHpa
			}
			tc.PressureStr = FormatPressure(pHpa, imperial)
		}

		if c.VisibilityM != nil {
			vM := *c.VisibilityM
			vMi := vM / 1609.344
			vKm := vM / 1000
			tc.VisibilityMi = &vMi
			tc.VisibilityKm = &vKm
			if imperial {
				tc.Visibility = &vMi
			} else {
				tc.Visibility = &vKm
			}
			tc.VisibilityStr = FormatVisibility(vM, imperial)
		}

		if a := c.Astronomy; a != nil {
			ta := &TemplateAstronomy{
				DayLength:        fmt.Sprintf("%dh %dm", int(a.DayLength.Hours()), int(a.DayLength.Minutes())%60),
				DayLengthSeconds: int64(a.DayLength.Seconds()),
				IsPolarDay:       a.IsPolarDay,
				IsPolarNight:     a.IsPolarNight,
			}
			if a.Sunrise != nil {
				ta.Sunrise = a.Sunrise.Local().Format("3:04 PM")
			}
			if a.Sunset != nil {
				ta.Sunset = a.Sunset.Local().Format("3:04 PM")
			}
			if !a.SolarNoon.IsZero() {
				ta.SolarNoon = a.SolarNoon.Local().Format("3:04 PM")
			}
			tc.Astronomy = ta
		}

		ctx.Conditions = tc
	}

	if f := data.Forecast; f != nil {
		tf := &TemplateForecast{
			GeneratedAt: f.GeneratedAt,
			Periods:     make([]TemplatePeriod, len(f.Periods)),
		}
		for i, p := range f.Periods {
			tp := TemplatePeriod{
				Name:         p.Name,
				StartTime:    p.StartTime,
				EndTime:      p.EndTime,
				IsDaytime:    p.IsDaytime,
				TempC:        p.TempC,
				TempF:        CelsiusToFahrenheit(p.TempC),
				WindDir:      p.WindDir,
				ShortDesc:    p.ShortDesc,
				DetailedDesc: p.DetailedDesc,
				HumidityPct:  p.HumidityPct,
			}
			if imperial {
				tp.Temp = tp.TempF
			} else {
				tp.Temp = tp.TempC
			}
			tp.TempStr = FormatTemp(p.TempC, imperial)

			wKph := p.WindKPH
			if imperial {
				tp.WindSpeed = KphToMPH(wKph)
			} else {
				tp.WindSpeed = wKph
			}
			tp.WindSpeedStr = FormatWindSpeed(wKph, imperial)

			if p.ProbabilityOfPrecipitation != nil {
				pop := *p.ProbabilityOfPrecipitation
				tp.ProbabilityOfPrecipitation = &pop
				tp.PoP = pop
				tp.PoPStr = fmt.Sprintf("%.0f%%", pop)
			}

			if p.DewPointC != nil {
				dp := *p.DewPointC
				if imperial {
					dp = CelsiusToFahrenheit(dp)
				}
				tp.DewPoint = &dp
				tp.DewPointStr = FormatTemp(*p.DewPointC, imperial)
			}

			tf.Periods[i] = tp
		}
		ctx.Forecast = tf
	}

	if len(data.Alerts) > 0 {
		ctx.Alerts = make([]TemplateAlert, len(data.Alerts))
		for i, a := range data.Alerts {
			ctx.Alerts[i] = TemplateAlert{
				ID:          a.ID,
				Event:       a.Event,
				Severity:    a.Severity,
				Headline:    a.Headline,
				Description: a.Description,
				Instruction: a.Instruction,
				Expires:     a.Expires,
				ExpiresStr:  a.Expires.Local().Format("Mon 3:04 PM"),
			}
		}
	}

	return ctx
}

// ExecuteTemplate parses and executes a template string or @file path against RenderData.
func ExecuteTemplate(tmplStr string, data RenderData, opts RenderOptions, w io.Writer) error {
	imperial := opts.Units != "metric"

	if strings.HasPrefix(tmplStr, "@") {
		path := strings.TrimPrefix(tmplStr, "@")
		if strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				path = filepath.Join(home, path[2:])
			}
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading template file: %w", err)
		}
		tmplStr = string(content)
	}

	funcMap := template.FuncMap{
		"formatTemp": func(celsius any) string {
			return FormatTemp(toFloat(celsius), imperial)
		},
		"formatWind": func(kph any) string {
			return FormatWindSpeed(toFloat(kph), imperial)
		},
		"formatPressure": func(hpa any) string {
			return FormatPressure(toFloat(hpa), imperial)
		},
		"formatVisibility": func(meters any) string {
			return FormatVisibility(toFloat(meters), imperial)
		},
		"temp": func(celsius any) string {
			return FormatTemp(toFloat(celsius), imperial)
		},
		"cToF": func(celsius any) float64 {
			return CelsiusToFahrenheit(toFloat(celsius))
		},
		"fToC": func(fahrenheit any) float64 {
			return FahrenheitToCelsius(toFloat(fahrenheit))
		},
		"kphToMph": func(kph any) float64 {
			return KphToMPH(toFloat(kph))
		},
		"mphToKph": func(mph any) float64 {
			return MphToKPH(toFloat(mph))
		},
		"compass": func(deg any) string {
			return DegreesToCompass(toFloat(deg))
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"trim":  strings.TrimSpace,
		"time": func(t time.Time, layout string) string {
			return t.Format(layout)
		},
		"timeLocal": func(t time.Time, layout string) string {
			return t.Local().Format(layout)
		},
		"age": func(t time.Time) string {
			d := time.Since(t)
			if d < 0 {
				d = 0
			}
			if d < time.Minute {
				return "just now"
			}
			if d < time.Hour {
				return fmt.Sprintf("%dm ago", int(d.Minutes()))
			}
			if d < 24*time.Hour {
				return fmt.Sprintf("%dh ago", int(d.Hours()))
			}
			return fmt.Sprintf("%dd ago", int(d.Hours()/24))
		},
		"round": func(args ...any) string {
			if len(args) == 0 {
				return ""
			}
			if len(args) == 1 {
				return fmt.Sprintf("%.0f", toFloat(args[0]))
			}
			// When piped in Go template: val | round 0 -> args[0] = 0, args[1] = val
			decimals := toInt(args[0])
			val := toFloat(args[1])
			format := fmt.Sprintf("%%.%df", decimals)
			return fmt.Sprintf(format, val)
		},
		"default": func(def any, val any) any {
			if val == nil {
				return def
			}
			if s, ok := val.(string); ok && s == "" {
				return def
			}
			return val
		},
		"deref": func(v *float64) float64 {
			if v == nil {
				return 0
			}
			return *v
		},
		"derefStr": func(v *string) string {
			if v == nil {
				return ""
			}
			return *v
		},
		"json": func(v any) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	}

	tmpl, err := template.New("wx").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	ctx := BuildTemplateContext(data, opts)
	return tmpl.Execute(w, ctx)
}

func renderTemplate(data RenderData, opts RenderOptions) error {
	return ExecuteTemplate(opts.Template, data, opts, os.Stdout)
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case *float64:
		if n != nil {
			return *n
		}
		return 0
	default:
		return 0
	}
}
