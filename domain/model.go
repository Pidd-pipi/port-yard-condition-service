package domain

import "fmt"

// YardZone is a port yard operating area.
type YardZone struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	OccupancyPct float64 `json:"occupancy_pct"`
	Refrigerated bool    `json:"refrigerated"`
	SurfaceTempC float64 `json:"surface_temp_c"`
	Status       string  `json:"status"`
	UpdatedAt    string  `json:"updated_at"`
}

// Yard zone status values.
const (
	StatusClear         = "clear"
	StatusInspectionDue = "inspection_due"
	StatusRestricted    = "restricted"
	StatusClosed        = "closed"
)

// ErrInvalidTransition is returned when a yard zone status move violates the
// state machine.
var ErrInvalidTransition = fmt.Errorf("invalid yard zone status transition")

// yardTransitions maps each yard status to the statuses it may legally move to.
// A closed zone is terminal: once closed it cannot move to any other status.
// A restricted zone cannot be returned directly to clear; it must be closed
// and re-opened through the normal flow.
var yardTransitions = map[string][]string{
	StatusClear:         {StatusInspectionDue, StatusRestricted, StatusClosed},
	StatusInspectionDue: {StatusRestricted, StatusClosed},
	StatusRestricted:    {StatusClosed},
	StatusClosed:        {},
}

// CanTransitionYard reports whether a yard zone may move from one status to
// another according to the operating state machine. A no-op move to the same
// status is allowed; otherwise the transition table governs what is legal.
func CanTransitionYard(from, to string) bool {
	if from == to {
		return true
	}
	allowed := yardTransitions[from]
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
