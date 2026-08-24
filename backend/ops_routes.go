package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// seedOpsRecords returns the initial operations records served by the API.
func seedOpsRecords() []OpsRecord {
	now := timeNowOps()
	return []OpsRecord{
		{ID: "ops-berth-01", Subject: "Berth 3 crane maintenance window", Owner: "quay", Status: OpsStatusActive, Priority: OpsPriorityHigh, Revision: 1, Labels: map[string]string{"site": "south-yard", "operator": "quay"}, CreatedAt: now, UpdatedAt: now},
		{ID: "ops-cool-02", Subject: "Cold storage lane inspection", Owner: "yard", Status: OpsStatusQueued, Priority: OpsPriorityCritical, Revision: 1, Labels: map[string]string{"site": "north-yard", "operator": "yard"}, CreatedAt: now, UpdatedAt: now},
		{ID: "ops-gate-03", Subject: "Gate 2 restricted access review", Owner: "security", Status: OpsStatusPaused, Priority: OpsPriorityNormal, Revision: 2, Labels: map[string]string{"site": "west-yard", "operator": "security"}, CreatedAt: now, UpdatedAt: now},
	}
}

// newOpsMux wires the operations records HTTP surface.
func newOpsMux(service *OpsService) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ops/records", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			opsListHandler(w, r, service)
		case http.MethodPost:
			opsCreateHandler(w, r, service)
		default:
			writeOpsError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	mux.HandleFunc("/api/ops/records/", func(w http.ResponseWriter, r *http.Request) {
		rest := opsPathID(r.URL.Path, "/api/ops/records/")
		if rest == "" {
			writeOpsError(w, http.StatusBadRequest, "record id required")
			return
		}
		id := rest
		if strings.HasSuffix(rest, "/status") {
			id = strings.TrimSuffix(rest, "/status")
			if id == "" {
				writeOpsError(w, http.StatusBadRequest, "record id required")
				return
			}
			if r.Method != http.MethodPost {
				writeOpsError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			opsTransitionHandler(w, r, service, id)
			return
		}
		switch r.Method {
		case http.MethodGet:
			opsGetHandler(w, r, service, id)
		case http.MethodPut:
			opsUpdateHandler(w, r, service, id)
		case http.MethodDelete:
			opsDeleteHandler(w, r, service, id)
		default:
			writeOpsError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	mux.HandleFunc("/api/ops/snapshot", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeOpsError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		opsJSON(w, http.StatusOK, service.Snapshot())
	})
	mux.HandleFunc("/api/ops/audit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeOpsError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if recordID := strings.TrimSpace(r.URL.Query().Get("record_id")); recordID != "" {
			opsJSON(w, http.StatusOK, map[string][]OpsEvent{"events": service.Audit(recordID)})
			return
		}
		limit := 50
		if raw := r.URL.Query().Get("limit"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
				limit = n
			}
		}
		opsJSON(w, http.StatusOK, map[string][]OpsEvent{"events": service.RecentEvents(limit)})
	})
	mux.HandleFunc("/api/ops/report", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeOpsError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		opsJSON(w, http.StatusOK, buildOpsReport(service))
	})
	return mux
}

func opsListHandler(w http.ResponseWriter, r *http.Request, service *OpsService) {
	query := OpsQuery{
		Subject:  strings.TrimSpace(r.URL.Query().Get("subject")),
		Status:   OpsStatus(strings.TrimSpace(r.URL.Query().Get("status"))),
		Priority: OpsPriority(strings.TrimSpace(r.URL.Query().Get("priority"))),
		Owner:    strings.TrimSpace(r.URL.Query().Get("owner")),
	}
	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && page > 0 {
		query.Page = page
	}
	if pageSize, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && pageSize > 0 {
		query.PageSize = pageSize
	}
	result, err := service.Search(r.Context(), query)
	if err != nil {
		writeOpsError(w, opsStatusForError(err), err.Error())
		return
	}
	opsJSON(w, http.StatusOK, result)
}

func opsCreateHandler(w http.ResponseWriter, r *http.Request, service *OpsService) {
	var request struct {
		ID       string            `json:"id"`
		Subject  string            `json:"subject"`
		Owner    string            `json:"owner"`
		Priority OpsPriority       `json:"priority"`
		Status   OpsStatus         `json:"status"`
		Labels   map[string]string `json:"labels"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&request); err != nil || strings.TrimSpace(request.ID) == "" {
		writeOpsError(w, http.StatusBadRequest, "id and subject are required")
		return
	}
	record := OpsRecord{ID: strings.TrimSpace(request.ID), Subject: request.Subject, Owner: request.Owner, Priority: request.Priority, Status: request.Status, Labels: request.Labels}
	created, err := service.Create(r.Context(), record)
	if err != nil {
		writeOpsError(w, opsStatusForError(err), err.Error())
		return
	}
	opsJSON(w, http.StatusCreated, created)
}

func opsGetHandler(w http.ResponseWriter, r *http.Request, service *OpsService, id string) {
	record, err := service.Get(r.Context(), id)
	if err != nil {
		writeOpsError(w, opsStatusForError(err), err.Error())
		return
	}
	opsJSON(w, http.StatusOK, record)
}

func opsUpdateHandler(w http.ResponseWriter, r *http.Request, service *OpsService, id string) {
	var request struct {
		Subject  string            `json:"subject"`
		Owner    string            `json:"owner"`
		Priority OpsPriority       `json:"priority"`
		Status   OpsStatus         `json:"status"`
		Labels   map[string]string `json:"labels"`
		Expected int               `json:"expected_revision"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&request); err != nil {
		writeOpsError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	record := OpsRecord{ID: id, Subject: request.Subject, Owner: request.Owner, Priority: request.Priority, Status: request.Status, Labels: request.Labels}
	updated, err := service.Update(r.Context(), id, request.Expected, record)
	if err != nil {
		writeOpsError(w, opsStatusForError(err), err.Error())
		return
	}
	opsJSON(w, http.StatusOK, updated)
}

func opsDeleteHandler(w http.ResponseWriter, r *http.Request, service *OpsService, id string) {
	if err := service.Delete(r.Context(), id); err != nil {
		writeOpsError(w, opsStatusForError(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func opsTransitionHandler(w http.ResponseWriter, r *http.Request, service *OpsService, id string) {
	var request struct {
		Status   OpsStatus `json:"status"`
		Expected int       `json:"expected_revision"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&request); err != nil || !opsStatusValid(request.Status) {
		writeOpsError(w, http.StatusBadRequest, "status is required")
		return
	}
	updated, err := service.Transition(r.Context(), id, request.Expected, request.Status, opsActorFromRequest(r))
	if err != nil {
		writeOpsError(w, opsStatusForError(err), err.Error())
		return
	}
	opsJSON(w, http.StatusOK, updated)
}
