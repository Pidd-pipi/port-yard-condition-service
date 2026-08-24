package readings

import (
	"sort"
	"sync"
	"time"
)

// AlertLog retains the most recent alerts produced by the service.
type AlertLog struct {
	mu    sync.RWMutex
	items []Alert
	limit int
}

func newAlertLog(limit int) *AlertLog {
	if limit < 1 {
		limit = 200
	}
	return &AlertLog{items: []Alert{}, limit: limit}
}

func (l *AlertLog) Add(alerts []Alert) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, alerts...)
	if len(l.items) > l.limit {
		l.items = l.items[len(l.items)-l.limit:]
	}
}

func (l *AlertLog) Recent(limit int) []Alert {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if limit <= 0 || limit > len(l.items) {
		limit = len(l.items)
	}
	out := make([]Alert, 0, limit)
	for i := len(l.items) - limit; i < len(l.items); i++ {
		out = append(out, l.items[i])
	}
	return out
}

// Service records sensor readings and evaluates yard conditions.
type Service struct {
	store  *Store
	alerts *AlertLog
	now    func() time.Time
}

// NewService builds a readings service over the given store.
func NewService(store *Store) *Service {
	return &Service{store: store, alerts: newAlertLog(200), now: time.Now}
}

// Record stores a reading and returns any alerts it triggers.
func (s *Service) Record(r Reading) []Alert {
	if r.At.IsZero() {
		r.At = s.now().UTC()
	}
	s.store.Append(r)
	detected := Detect(r)
	s.alerts.Add(detected)
	return detected
}

// Recent returns the retained readings for a zone.
func (s *Service) Recent(zoneID string, limit int) []Reading {
	return s.store.Recent(zoneID, limit)
}

// Alerts returns the most recent alerts.
func (s *Service) Alerts(limit int) []Alert {
	return s.alerts.Recent(limit)
}

// Summary aggregates retained readings across zones.
type Summary struct {
	Zones    int                `json:"zones"`
	Readings int                `json:"readings"`
	Alerts   int                `json:"alerts"`
	AvgTempC map[string]float64 `json:"avg_temp_c"`
	MaxTempC map[string]float64 `json:"max_temp_c"`
}

// Summary computes per-zone aggregates over all retained readings.
func (s *Service) Summary() Summary {
	out := Summary{AvgTempC: map[string]float64{}, MaxTempC: map[string]float64{}}
	for _, zoneID := range s.store.Zones() {
		recent := s.store.Recent(zoneID, 0)
		out.Zones++
		out.Readings += len(recent)
		var sum float64
		var max float64
		for i, r := range recent {
			sum += r.TempC
			if i == 0 || r.TempC > max {
				max = r.TempC
			}
		}
		if len(recent) > 0 {
			out.AvgTempC[zoneID] = sum / float64(len(recent))
			out.MaxTempC[zoneID] = max
		}
	}
	out.Alerts = len(s.alerts.Recent(0))
	return out
}

// sortedZones returns zone ids ordered by descending reading count.
func sortedZones(store *Store) []string {
	counts := map[string]int{}
	for _, zoneID := range store.Zones() {
		counts[zoneID] = store.Count(zoneID)
	}
	zones := make([]string, 0, len(counts))
	for zoneID := range counts {
		zones = append(zones, zoneID)
	}
	sort.Slice(zones, func(i, j int) bool {
		if counts[zones[i]] != counts[zones[j]] {
			return counts[zones[i]] > counts[zones[j]]
		}
		return zones[i] < zones[j]
	})
	return zones
}

// Trim enforces the retention cap for every zone with retained readings.
func (s *Service) Trim() error {
	for _, zoneID := range s.store.Zones() {
		s.store.Trim(zoneID)
	}
	return nil
}
