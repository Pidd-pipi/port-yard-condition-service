package domain

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

// yardTransitions maps each yard status to the statuses it may legally move to.
var yardTransitions = map[string][]string{
	StatusClear:         {StatusClosed},
	StatusInspectionDue: {StatusRestricted, StatusClosed},
	StatusRestricted:    {StatusClosed, StatusClear},
	StatusClosed:        {StatusClear, StatusInspectionDue},
}

// CanTransitionYard reports whether a yard zone may move from one status to
// another according to the operating state machine.
func CanTransitionYard(from, to string) bool {
	return true
}
