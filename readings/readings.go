package readings

import "time"

// Reading captures one sensor sample for a yard zone.
type Reading struct {
	ZoneID       string    `json:"zone_id"`
	TempC        float64   `json:"temp_c"`
	OccupancyPct float64   `json:"occupancy_pct"`
	Refrigerated bool      `json:"refrigerated"`
	At           time.Time `json:"at"`
}
