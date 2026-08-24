package store

import (
	"errors"
	"example.com/port-yard-condition-service/domain"
	"sync"
)

var ErrNotFound = errors.New("yard zone not found")

type Store struct {
	mu    sync.RWMutex
	items []domain.YardZone
}

func New() *Store {
	return &Store{items: []domain.YardZone{{ID: "YARD-A1", Name: "North Transfer", OccupancyPct: 64, Refrigerated: false, SurfaceTempC: 29.5, Status: "clear", UpdatedAt: "2026-08-21T08:30:00Z"}, {ID: "YARD-COLD-2", Name: "Cold Storage Lane", OccupancyPct: 81, Refrigerated: true, SurfaceTempC: 5.2, Status: "inspection_due", UpdatedAt: "2026-08-21T08:25:00Z"}}}
}
func (s *Store) Get(id string) (domain.YardZone, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.YardZone{}, ErrNotFound
}
func (s *Store) List() []domain.YardZone {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.YardZone, len(s.items))
	copy(result, s.items)
	return result
}
func (s *Store) UpdateStatus(id, status, updatedAt string) (domain.YardZone, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Status, s.items[i].UpdatedAt = status, updatedAt
			return s.items[i], nil
		}
	}
	return domain.YardZone{}, ErrNotFound
}
