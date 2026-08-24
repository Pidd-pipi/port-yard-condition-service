package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/port-yard-condition-service/httpapi"
	"example.com/port-yard-condition-service/readings"
	"example.com/port-yard-condition-service/store"
	"example.com/port-yard-condition-service/web"
)

func fullHandler008() http.Handler {
	yard := httpapi.NewHandler(store.New(), web.FS)
	opsMux := newOpsMux(newOpsService(seedOpsRecords()))
	readingsMux := readings.NewMux(readings.NewService(readings.NewStore(50)))
	server := newEnterpriseServer(":0", newRootHandler(yard, opsMux, readingsMux))
	return server.Handler
}

func opsPost008(t *testing.T, h http.Handler, body string, ctx context.Context, requestID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records", strings.NewReader(body))
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestOpsCanceledCreateAborts(t *testing.T) {
	h := fullHandler008()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec := opsPost008(t, h, `{"id":"ops-ctx-1","subject":"s","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`, ctx, "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("canceled create status = %d, want 500 (operation must abort)", rec.Code)
	}
}

func TestOpsFirstRequestCtxNotReused(t *testing.T) {
	h := fullHandler008()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opsPost008(t, h, `{"id":"ops-ctx-a","subject":"a","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`, ctx, "req-first")
	rec := opsPost008(t, h, `{"id":"ops-ctx-b","subject":"b","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`, context.Background(), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("second create status = %d, want 201 (first request ctx leaked)", rec.Code)
	}
}

func TestTrimOpsAuditHonorsCancel(t *testing.T) {
	svc := newOpsService(nil)
	svc.audit.Clear()
	for i := 0; i < 10; i++ {
		svc.audit.Add("rec", "created", "tester")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := trimOpsAudit(ctx, svc, 3)
	if err == nil {
		t.Fatal("trimOpsAudit should abort with a canceled context")
	}
	if got := svc.audit.Count(); got != 10 {
		t.Fatalf("audit count after canceled sweep = %d, want 10", got)
	}
}
