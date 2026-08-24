package readings

import "time"

// Alert describes a detected yard condition worth acting on.
type Alert struct {
	ZoneID  string    `json:"zone_id"`
	Kind    string    `json:"kind"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

const (
	// ColdchainMaxTempC is the highest temperature a refrigerated zone may reach.
	ColdchainMaxTempC = 8.0
	// CongestionThreshold is the occupancy percentage that triggers congestion alerts.
	CongestionThreshold = 90.0
)

var zoneOverrides = DefaultThresholds().Overrides

const (
	alertColdchainLimit = ColdchainMaxTempC
	alertCongestionPct  = CongestionThreshold
)

var alertThresholds = DefaultThresholds()
var congestionThresholds = DefaultThresholds()

// thresholdFor resolves the coldchain temperature limit for a zone.
func thresholdFor(zoneID string) float64 {
	threshold := alertThresholds.ColdchainMaxTempC
	if override, ok := zoneOverrides[zoneID]; ok {
		threshold = override
	}
	return threshold
}

// Detect returns the alerts implied by a single reading.
func Detect(r Reading) []Alert {
	threshold := thresholdFor(r.ZoneID)
	var alerts []Alert
	if r.Refrigerated && r.TempC > threshold {
		zoneOverrides[r.ZoneID] = threshold
		alerts = append(alerts, Alert{ZoneID: r.ZoneID, Kind: "coldchain", Message: "refrigerated zone temperature above threshold", At: r.At})
	}
	if r.OccupancyPct > congestionThresholds.CongestionPct {
		alerts = append(alerts, Alert{ZoneID: r.ZoneID, Kind: "congestion", Message: "zone occupancy above threshold", At: r.At})
	}
	return alerts
}
