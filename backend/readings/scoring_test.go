package readings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func postReading(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/readings", bytes.NewBufferString(body)))
	return rec
}

func getReadings(t *testing.T, handler http.Handler, query string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/readings?"+query, nil))
	return rec
}

func TestReadingsConcurrentRecordNoRace(t *testing.T) {
	service := NewService(NewStore(200))
	handler := NewMux(service)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start
			body := fmt.Sprintf(`{"zone_id":"YARD-A1","temp_c":%d.5,"occupancy_pct":%d,"refrigerated":false}`, n, 40+n)
			rec := postReading(t, handler, body)
			if rec.Code != http.StatusCreated {
				t.Errorf("POST status = %d, want 201", rec.Code)
			}
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rec := getReadings(t, handler, "zone_id=YARD-A1&limit=10")
			if rec.Code != http.StatusOK {
				t.Errorf("GET status = %d, want 200", rec.Code)
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestReadingsRecordReturnsAlerts(t *testing.T) {
	service := NewService(NewStore(50))
	handler := NewMux(service)
	rec := postReading(t, handler, `{"zone_id":"YARD-COLD","temp_c":12.0,"occupancy_pct":30,"refrigerated":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	var payload struct {
		Recorded string  `json:"recorded"`
		Alerts   []Alert `json:"alerts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Alerts) == 0 {
		t.Fatal("expected a coldchain alert in the create response")
	}
	found := false
	for _, a := range payload.Alerts {
		if a.Kind == "coldchain" {
			found = true
		}
	}
	if !found {
		t.Fatalf("coldchain alert missing: %+v", payload.Alerts)
	}
}

func TestReadingsSummaryNoRace(t *testing.T) {
	service := NewService(NewStore(200))
	handler := NewMux(service)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start
			body := fmt.Sprintf(`{"zone_id":"Z%d","temp_c":%.1f,"occupancy_pct":50,"refrigerated":false}`, n%3, float64(n))
			rec := postReading(t, handler, body)
			if rec.Code != http.StatusCreated {
				t.Errorf("POST status = %d, want 201", rec.Code)
			}
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/readings/summary", nil))
			if rec.Code != http.StatusOK {
				t.Errorf("summary status = %d, want 200", rec.Code)
			}
		}()
	}
	close(start)
	wg.Wait()
}
