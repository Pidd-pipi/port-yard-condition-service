package httpapi

import (
	"encoding/json"
	"example.com/port-yard-condition-service/domain"
	"example.com/port-yard-condition-service/store"
	"example.com/port-yard-condition-service/web"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func yardServer010() http.Handler {
	return NewHandler(store.New(), web.FS)
}

func yardStatus010(t *testing.T, h http.Handler, id string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/yard-zones", nil))
	var payload struct {
		Items []domain.YardZone `json:"items"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	for _, item := range payload.Items {
		if item.ID == id {
			return item.Status
		}
	}
	return ""
}

func TestYardRejectsClosedToClear(t *testing.T) {
	h := yardServer010()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"restricted"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("to restricted: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"closed"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("to closed: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"clear"}`)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("closed->clear status = %d, want 409", rec.Code)
	}
}

func TestYardStatePreservedOnRejected(t *testing.T) {
	h := yardServer010()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"restricted"}`)))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"closed"}`)))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"clear"}`)))
	if got := yardStatus010(t, h, "YARD-A1"); got != "closed" {
		t.Fatalf("zone status after rejected transition = %s, want closed", got)
	}
}

func TestYardRejectsRestrictedToClear(t *testing.T) {
	h := yardServer010()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"restricted"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("to restricted: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"clear"}`)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("restricted->clear status = %d, want 409", rec.Code)
	}
}

func TestYardClosedNotRestricted(t *testing.T) {
	h := yardServer010()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"restricted"}`)))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"closed"}`)))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"restricted"}`)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("closed->restricted status = %d, want 409", rec.Code)
	}
}
