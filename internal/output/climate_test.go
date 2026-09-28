package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/models"
)

func sampleClimatePayload() *models.ClimatePayload {
	date := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	depCurF := 4.5
	depCurC := 2.5
	obsCurF := 76.5
	obsCurC := 24.7

	return &models.ClimatePayload{
		Location: "Fort Wayne, IN",
		Climate: &models.ClimateReport{
			Date:          date,
			Location:      "Fort Wayne, IN",
			StationID:     "14827 1",
			StationName:   "FORT WAYNE INTL AP",
			Latitude:      40.97,
			Longitude:     -85.20,
			ElevationFt:   797,
			ElevationM:    243,
			NormalsPeriod: "1991–2020",
			TodayNormals: models.DailyNormals{
				Date:           date,
				NormalHighF:    72.0,
				NormalHighC:    22.2,
				NormalLowF:     48.0,
				NormalLowC:     8.9,
				NormalMeanF:    60.0,
				NormalMeanC:    15.6,
				NormalPrecipIn: 0.10,
				NormalPrecipMM: 2.54,
			},
			Records: models.DailyRecords{
				RecordHigh: models.DailyRecord{
					ValueF: 89.0,
					ValueC: 31.7,
					Years:  []int{2021},
				},
				RecordLow: models.DailyRecord{
					ValueF: 36.0,
					ValueC: 2.2,
					Years:  []int{2024},
				},
				ColdestHigh: models.DailyRecord{
					ValueF: 58.0,
					ValueC: 14.4,
					Years:  []int{1982},
				},
				WarmestLow: models.DailyRecord{
					ValueF: 64.0,
					ValueC: 17.8,
					Years:  []int{2017},
				},
				RecordPrecip: models.DailyRecord{
					ValueIn: 1.25,
					ValueMM: 31.75,
					Years:   []int{2024},
				},
				PeriodOfRecord:    "1940–2025",
				TotalYearsSampled: 86,
			},
			Departure: &models.ClimateDeparture{
				ObservedCurrentF:  &obsCurF,
				ObservedCurrentC:  &obsCurC,
				DepartureCurrentF: &depCurF,
				DepartureCurrentC: &depCurC,
				Summary:           "+4.5°F Departure (Above Normal)",
			},
			MonthlyNormals: &models.MonthlyNormals{
				MonthName:           "September",
				NormalAvgHighF:      75.5,
				NormalAvgHighC:      24.2,
				NormalAvgLowF:       52.3,
				NormalAvgLowC:       11.3,
				NormalTotalPrecipIn: 3.12,
				NormalTotalPrecipMM: 79.25,
			},
		},
	}
}

func TestRenderClimate_JSON(t *testing.T) {
	payload := sampleClimatePayload()
	var buf bytes.Buffer

	err := RenderClimateTo(&buf, payload, ClimateOptions{ForceJSON: true})
	if err != nil {
		t.Fatalf("RenderClimateTo error: %v", err)
	}

	var decoded models.ClimatePayload
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if decoded.Location != "Fort Wayne, IN" {
		t.Errorf("expected location 'Fort Wayne, IN', got %q", decoded.Location)
	}
	if decoded.Climate == nil || decoded.Climate.StationName != "FORT WAYNE INTL AP" {
		t.Errorf("station name mismatch: %v", decoded.Climate)
	}
	if decoded.Climate.TodayNormals.NormalHighF != 72.0 {
		t.Errorf("expected normal high 72.0, got %f", decoded.Climate.TodayNormals.NormalHighF)
	}
}

func TestRenderClimate_PrettyImperial(t *testing.T) {
	payload := sampleClimatePayload()
	var buf bytes.Buffer

	err := RenderClimateTo(&buf, payload, ClimateOptions{ForcePretty: true, Units: "imperial"})
	if err != nil {
		t.Fatalf("RenderClimateTo error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "NOAA 30-Year Climate Normals") {
		t.Errorf("missing header in pretty output: %s", out)
	}
	if !strings.Contains(out, "FORT WAYNE INTL AP") {
		t.Errorf("missing station name in pretty output")
	}
	if !strings.Contains(out, "WARM ANOMALY") {
		t.Errorf("missing warm anomaly badge in pretty output")
	}
	if !strings.Contains(out, "72.0°F") {
		t.Errorf("missing normal high in pretty output")
	}
	if !strings.Contains(out, "89.0°F (2021)") {
		t.Errorf("missing record high in pretty output")
	}
	if !strings.Contains(out, "1.25 in (2024)") {
		t.Errorf("missing record precip in pretty output")
	}
}

func TestRenderClimate_PrettyMetric(t *testing.T) {
	payload := sampleClimatePayload()
	var buf bytes.Buffer

	err := RenderClimateTo(&buf, payload, ClimateOptions{ForcePretty: true, Units: "metric"})
	if err != nil {
		t.Fatalf("RenderClimateTo error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "22.2°C") {
		t.Errorf("missing metric normal high in pretty output: %s", out)
	}
	if !strings.Contains(out, "31.7°C (2021)") {
		t.Errorf("missing metric record high in pretty output: %s", out)
	}
	if !strings.Contains(out, "31.8 mm (2024)") && !strings.Contains(out, "31.7 mm (2024)") {
		t.Errorf("missing metric record precip in pretty output: %s", out)
	}
}

func TestRenderClimate_Nil(t *testing.T) {
	var buf bytes.Buffer
	err := RenderClimateTo(&buf, nil, ClimateOptions{})
	if err == nil {
		t.Errorf("expected error for nil payload")
	}
}
