package models

import "time"

// AlertCell represents a specific warning or critical hazard cell.
type AlertCell struct {
	ID          string    `json:"id"`
	Event       string    `json:"event"`
	Headline    string    `json:"headline"`
	AreaDesc    string    `json:"area_desc"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Urgency     string    `json:"urgency"`
	Certainty   string    `json:"certainty"`
	SenderName  string    `json:"sender_name"`
	States      []string  `json:"states"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	HasPolygon  bool      `json:"has_polygon"`
	HazardText  string    `json:"hazard_text,omitempty"`
	RadarSite   string    `json:"radar_site,omitempty"`
	Effective   time.Time `json:"effective"`
	Expires     time.Time `json:"expires"`
}

// StormCluster groups adjacent, contiguous active alerts into a synoptic weather system.
type StormCluster struct {
	ID            int            `json:"id"`
	Name          string         `json:"name"`
	States        []string       `json:"states"`
	CenterLat     float64        `json:"center_lat"`
	CenterLon     float64        `json:"center_lon"`
	NearestRadar  string         `json:"nearest_radar"`
	TotalAlerts   int            `json:"total_alerts"`
	Score         int            `json:"score"`
	HazardsCount  map[string]int `json:"hazards_count"`
	PrimaryHazard string         `json:"primary_hazard"`
	SPCRisk       string         `json:"spc_risk,omitempty"`
	MCDWatch      string         `json:"mcd_watch,omitempty"`
	Cells         []AlertCell    `json:"cells"`
}

// ChasePayload represents the complete nationwide active storm clusters for remote storm chasing.
type ChasePayload struct {
	GeneratedAt   time.Time      `json:"generated_at"`
	TotalAlerts   int            `json:"total_alerts"`
	TotalClusters int            `json:"total_clusters"`
	SPC           *SPCPayload    `json:"spc,omitempty"`
	Clusters      []StormCluster `json:"clusters"`
}
