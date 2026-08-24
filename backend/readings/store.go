package readings

import (
	"sort"
	"sync"
)

// Store keeps a bounded per-zone history of sensor readings.
type Store struct {
	mu       sync.RWMutex
	byZone   map[string][]Reading
	capacity int
}

// NewStore builds a store that retains at most capacity samples per zone.
func NewStore(capacity int) *Store {
	if capacity < 1 {
		capacity = 100
	}
	return &Store{byZone: map[string][]Reading{}, capacity: capacity}
}

// Append records a new reading, evicting the oldest sample of the zone when the
// per-zone history exceeds the store capacity.
func (s *Store) Append(r Reading) {
	zone := s.byZone[r.ZoneID]
	zone = append(zone, r)
	if len(zone) > s.capacity {
		zone = zone[len(zone)-s.capacity:]
	}
	s.byZone[r.ZoneID] = zone
}

// Recent returns up to limit newest readings for a zone, newest first.
func (s *Store) Recent(zoneID string, limit int) []Reading {
	zone := s.byZone[zoneID]
	if limit <= 0 || limit > len(zone) {
		limit = len(zone)
	}
	out := make([]Reading, 0, limit)
	for i := len(zone) - limit; i < len(zone); i++ {
		out = append(out, zone[i])
	}
	return out
}

// Count returns how many readings are retained for a zone.
func (s *Store) Count(zoneID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byZone[zoneID])
}

// Zones returns the distinct zone ids with retained readings, sorted.
func (s *Store) Zones() []string {
	out := make([]string, 0, len(s.byZone))
	for id := range s.byZone {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Trim drops the oldest readings of a zone until its history fits the capacity.
func (s *Store) Trim(zoneID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	zone := s.byZone[zoneID]
	if len(zone) > s.capacity {
		s.byZone[zoneID] = append([]Reading(nil), zone[len(zone)-s.capacity:]...)
	}
}
