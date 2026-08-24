package readings

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func postReading007(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/readings", bytes.NewBufferString(body)))
	return rec
}

func TestReadingsNullBodyNoPanic(t *testing.T) {
	service := NewService(NewStore(50))
	handler := NewMux(service)
	rec := postReading007(t, handler, "null")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("null body status = %d, want 400", rec.Code)
	}
}

func TestReadingsColdchainNoPanic(t *testing.T) {
	service := NewService(NewStore(50))
	handler := NewMux(service)
	rec := postReading007(t, handler, `{"zone_id":"YARD-COLD","temp_c":12.0,"occupancy_pct":30,"refrigerated":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("coldchain status = %d, want 201", rec.Code)
	}
	var payload struct {
		Alerts []Alert `json:"alerts"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if len(payload.Alerts) == 0 {
		t.Fatal("expected a coldchain alert")
	}
}

func TestReadingsCongestionNoPanic(t *testing.T) {
	service := NewService(NewStore(50))
	handler := NewMux(service)
	rec := postReading007(t, handler, `{"zone_id":"YARD-BUSY","temp_c":5.0,"occupancy_pct":95,"refrigerated":false}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("congestion status = %d, want 201", rec.Code)
	}
	var payload struct {
		Alerts []Alert `json:"alerts"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if len(payload.Alerts) == 0 {
		t.Fatal("expected a congestion alert")
	}
}

func TestReadingsValidReadingOK(t *testing.T) {
	service := NewService(NewStore(50))
	handler := NewMux(service)
	rec := postReading007(t, handler, `{"zone_id":"YARD-OK","temp_c":5.0,"occupancy_pct":40,"refrigerated":false}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("valid reading status = %d, want 201", rec.Code)
	}
}

func TestReadingsBlankZoneNoPanic(t *testing.T) {
	service := NewService(NewStore(50))
	handler := NewMux(service)
	rec := postReading007(t, handler, `{"zone_id":"   ","temp_c":5.0,"occupancy_pct":40,"refrigerated":false}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("blank zone status = %d, want 400", rec.Code)
	}
}
