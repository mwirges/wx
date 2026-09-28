package models

import "time"

// SPCRiskCategory defines an official SPC convective risk level.
type SPCRiskCategory struct {
	DN          int    `json:"dn"`          // 0: None, 2: TSTM, 3: MRGL, 4: SLGT, 5: ENH, 6: MDT, 8: HIGH
	Code        string `json:"code"`        // NONE, TSTM, MRGL, SLGT, ENH, MDT, HIGH
	Name        string `json:"name"`        // e.g. "Slight Risk"
	Description string `json:"description"` // e.g. "Scattered severe storms possible"
	Color       string `json:"color"`       // Hex color: e.g. "#E6E600"
}

// SPCOutlookItem represents a single day's convective outlook (Day 1, Day 2, or Day 3).
type SPCOutlookItem struct {
	Day         int             `json:"day"`
	Valid       time.Time       `json:"valid"`
	Expires     time.Time       `json:"expires"`
	Issue       time.Time       `json:"issue"`
	Category    SPCRiskCategory `json:"category"`
	TornadoProb string          `json:"tornado_prob,omitempty"` // e.g. "5%", "10%", or "None"
	TornadoSig  bool            `json:"tornado_sig,omitempty"`  // Hatched / significant 10%+
	HailProb    string          `json:"hail_prob,omitempty"`    // e.g. "15%"
	HailSig     bool            `json:"hail_sig,omitempty"`     // Hatched / 2"+ hail
	WindProb    string          `json:"wind_prob,omitempty"`    // e.g. "15%"
	WindSig     bool            `json:"wind_sig,omitempty"`     // Hatched / 65kt+ wind
	SevereProb  string          `json:"severe_prob,omitempty"`  // for Day 3 (e.g. "15%")
	SevereSig   bool            `json:"severe_sig,omitempty"`   // Day 3 significant severe
}

// MesoscaleDiscussion represents an active SPC Mesoscale Discussion (MCD).
type MesoscaleDiscussion struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`              // "MD 2329"
	Title            string    `json:"title"`             // "MD 2329 Active Till 0100 UTC"
	URL              string    `json:"url"`               // "https://www.spc.noaa.gov/products/md/md2329.html"
	AreasAffected    string    `json:"areas_affected"`    // "Parts of the NE Sand Hills region"
	Concerning       string    `json:"concerning"`        // "Severe potential...Watch unlikely"
	WatchProbability string    `json:"watch_probability"` // "20%"
	Summary          string    `json:"summary"`
	Discussion       string    `json:"discussion,omitempty"`
	Sent             time.Time `json:"sent"`
	Expires          time.Time `json:"expires"`
	Lat              float64   `json:"lat,omitempty"`
	Lon              float64   `json:"lon,omitempty"`
}

// SPCWatch represents an active SPC Tornado or Severe Thunderstorm Watch.
type SPCWatch struct {
	ID          string    `json:"id"`
	WatchNumber int       `json:"watch_number"`
	Type        string    `json:"type"` // "Tornado Watch" or "Severe Thunderstorm Watch"
	Headline    string    `json:"headline"`
	AreaDesc    string    `json:"area_desc"`
	States      []string  `json:"states"`
	Effective   time.Time `json:"effective"`
	Expires     time.Time `json:"expires"`
	Active      bool      `json:"active"`
	Severity    string    `json:"severity,omitempty"`
	Urgency     string    `json:"urgency,omitempty"`
	URL         string    `json:"url,omitempty"`
}

// SPCPayload contains complete SPC outlooks, active MCDs, and watches.
type SPCPayload struct {
	Location          string                `json:"location,omitempty"`
	Coordinates       [2]float64            `json:"coordinates"`
	FetchedAt         time.Time             `json:"fetched_at"`
	Day1              SPCOutlookItem        `json:"day1"`
	Day2              SPCOutlookItem        `json:"day2"`
	Day3              SPCOutlookItem        `json:"day3"`
	ActiveMCDs        []MesoscaleDiscussion `json:"active_mcds"`
	ActiveWatches     []SPCWatch            `json:"active_watches"`
	MaxNationalRisk   SPCRiskCategory       `json:"max_national_risk"`
	ConvectiveSummary string                `json:"convective_summary"`
}
