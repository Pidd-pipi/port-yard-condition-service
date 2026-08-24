package readings

import (
	"sync"
	"time"
)

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

// thresholdsConfig holds the mutable, per-zone alert thresholds. It is read and
// written from concurrent Record/Detect calls, so every access must hold the
// mutex.
type thresholdsConfig struct {
	mu         sync.RWMutex
	defaults   Thresholds
	overrides  map[string]float64 // per-zone coldchain max temp overrides
	congestion float64
}

var alertThresholds = newThresholdsConfig(DefaultThresholds())

func newThresholdsConfig(t *Thresholds) *thresholdsConfig {
	if t == nil {
		t = DefaultThresholds()
	}
	return &thresholdsConfig{
		defaults:   *t,
		overrides:  t.Overrides,
		congestion: t.CongestionPct,
	}
}

// thresholdFor resolves the coldchain temperature limit for a zone.
func thresholdFor(zoneID string) float64 {
	alertThresholds.mu.RLock()
	defer alertThresholds.mu.RUnlock()
	if override, ok := alertThresholds.overrides[zoneID]; ok {
		return override
	}
	return alertThresholds.defaults.ColdchainMaxTempC
}

// recordThreshold pins the threshold that applied to a violating reading so
// later detections for the same zone compare against a stable limit.
func recordThreshold(zoneID string, threshold float64) {
	alertThresholds.mu.Lock()
	defer alertThresholds.mu.Unlock()
	alertThresholds.overrides[zoneID] = threshold
}

// congestionLimit returns the occupancy percentage above which a zone is congested.
func congestionLimit() float64 {
	alertThresholds.mu.RLock()
	defer alertThresholds.mu.RUnlock()
	return alertThresholds.congestion
}

// Detect returns the alerts implied by a single reading.
func Detect(r Reading) []Alert {
	var alerts []Alert
	if r.Refrigerated && r.TempC > thresholdFor(r.ZoneID) {
		recordThreshold(r.ZoneID, thresholdFor(r.ZoneID))
		alerts = append(alerts, Alert{ZoneID: r.ZoneID, Kind: "coldchain", Message: "refrigerated zone temperature above threshold", At: r.At})
	}
	if r.OccupancyPct > congestionLimit() {
		alerts = append(alerts, Alert{ZoneID: r.ZoneID, Kind: "congestion", Message: "zone occupancy above threshold", At: r.At})
	}
	return alerts
}
