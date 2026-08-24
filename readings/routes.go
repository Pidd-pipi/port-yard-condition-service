package readings

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// NewMux wires the readings HTTP surface.
func NewMux(service *Service) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/readings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listHandler(w, r, service)
		case http.MethodPost:
			recordHandler(w, r, service)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	mux.HandleFunc("/api/readings/alerts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string][]Alert{"alerts": service.Alerts(parseLimit(r, 50))})
	})
	mux.HandleFunc("/api/readings/summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, service.Summary())
	})
	return mux
}

func recordHandler(w http.ResponseWriter, r *http.Request, service *Service) {
	var request struct {
		ZoneID       string  `json:"zone_id"`
		TempC        float64 `json:"temp_c"`
		OccupancyPct float64 `json:"occupancy_pct"`
		Refrigerated bool    `json:"refrigerated"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&request); err != nil || strings.TrimSpace(request.ZoneID) == "" {
		writeError(w, http.StatusBadRequest, "zone_id and reading are required")
		return
	}
	alerts := service.Record(Reading{
		ZoneID:       strings.TrimSpace(request.ZoneID),
		TempC:        request.TempC,
		OccupancyPct: request.OccupancyPct,
		Refrigerated: request.Refrigerated,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"recorded": request.ZoneID, "alerts": alerts})
}

func listHandler(w http.ResponseWriter, r *http.Request, service *Service) {
	zoneID := strings.TrimSpace(r.URL.Query().Get("zone_id"))
	if zoneID == "" {
		writeError(w, http.StatusBadRequest, "zone_id required")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]Reading{"readings": service.Recent(zoneID, parseLimit(r, 50))})
}

func parseLimit(r *http.Request, def int) int {
	raw := r.URL.Query().Get("limit")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 500 {
		return def
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
