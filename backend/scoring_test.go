package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func opsServer002() (*OpsService, http.Handler) {
	svc := newOpsService(seedOpsRecords())
	return svc, newOpsMux(svc)
}

func opsReq002(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestOpsAuditRetentionBounded(t *testing.T) {
	svc, h := opsServer002()
	rec := opsReq002(t, h, http.MethodPost, "/api/ops/records", `{"id":"ops-ret-1","subject":"retention check","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	const target = 520
	rev := 1
	for i := 0; i < target; i++ {
		rec = opsReq002(t, h, http.MethodPost, "/api/ops/records/ops-ret-1/status", fmt.Sprintf(`{"status":"active","expected_revision":%d}`, rev))
		if rec.Code != http.StatusOK {
			t.Fatalf("transition %d status = %d, want 200", i, rec.Code)
		}
		rev++
	}
	if got := svc.audit.Count(); got > 500 {
		t.Fatalf("audit count = %d, want bounded by retention cap 500", got)
	}
}

func TestOpsAuditForIsolated(t *testing.T) {
	_, h := opsServer002()
	opsReq002(t, h, http.MethodPost, "/api/ops/records", `{"id":"ops-aud-a","subject":"a","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`)
	opsReq002(t, h, http.MethodPost, "/api/ops/records", `{"id":"ops-aud-b","subject":"b","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`)
	opsReq002(t, h, http.MethodPost, "/api/ops/records/ops-aud-a/status", `{"status":"active","expected_revision":1}`)
	rec := opsReq002(t, h, http.MethodGet, "/api/ops/audit?record_id=ops-aud-a", "")
	var payload struct {
		Events []OpsEvent `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if len(payload.Events) == 0 {
		t.Fatal("expected audit events for ops-aud-a")
	}
	for _, e := range payload.Events {
		if e.RecordID != "ops-aud-a" {
			t.Fatalf("audit for ops-aud-a contains event of record %s", e.RecordID)
		}
	}
}

func TestOpsAuditRecentNewestFirst(t *testing.T) {
	_, h := opsServer002()
	for _, id := range []string{"ops-rec-1", "ops-rec-2"} {
		rec := opsReq002(t, h, http.MethodPost, "/api/ops/records", `{"id":"`+id+`","subject":"s","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %s: %d", id, rec.Code)
		}
	}
	rec := opsReq002(t, h, http.MethodGet, "/api/ops/audit?limit=1", "")
	var payload struct {
		Events []OpsEvent `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if len(payload.Events) != 1 {
		t.Fatalf("recent events len = %d, want 1", len(payload.Events))
	}
	if payload.Events[0].RecordID != "ops-rec-2" {
		t.Fatalf("recent event is %s, want the newest (ops-rec-2)", payload.Events[0].RecordID)
	}
}

func TestOpsAuditTrimKeepsNewest(t *testing.T) {
	svc, _ := opsServer002()
	svc.audit.Clear()
	for i := 0; i < 10; i++ {
		svc.audit.Add(fmt.Sprintf("rec-%d", i), "created", "tester")
	}
	svc.audit.Trim(3)
	recent := svc.audit.Recent(3)
	if len(recent) != 3 {
		t.Fatalf("after trim recent len = %d, want 3", len(recent))
	}
	if recent[0].RecordID != "rec-7" || recent[1].RecordID != "rec-8" || recent[2].RecordID != "rec-9" {
		t.Fatalf("trim kept wrong events: %v", []string{recent[0].RecordID, recent[1].RecordID, recent[2].RecordID})
	}
}

func TestOpsTimestampsUTC(t *testing.T) {
	_, h := opsServer002()
	rec := opsReq002(t, h, http.MethodPost, "/api/ops/records", `{"id":"ops-ts-1","subject":"ts","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d", rec.Code)
	}
	var created OpsRecord
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if !strings.HasSuffix(created.CreatedAt, "Z") && !strings.Contains(created.CreatedAt, "+00:00") {
		t.Fatalf("created timestamp not in UTC: %s", created.CreatedAt)
	}
	parsed, err := time.Parse(time.RFC3339Nano, created.UpdatedAt)
	if err != nil {
		t.Fatalf("updated_at unparseable: %v", err)
	}
	if _, offset := parsed.Zone(); offset != 0 {
		t.Fatalf("updated_at has non-UTC offset %d: %s", offset, created.UpdatedAt)
	}
}
