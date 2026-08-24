package readings

// Thresholds configures the alert detection limits.
type Thresholds struct {
	ColdchainMaxTempC float64
	CongestionPct     float64
	Overrides         map[string]float64 // per-zone coldchain max temp overrides
}

// DefaultThresholds returns the standard alert thresholds.
func DefaultThresholds() *Thresholds {
	return &Thresholds{ColdchainMaxTempC: 8.0, CongestionPct: 90.0, Overrides: map[string]float64{}}
}
