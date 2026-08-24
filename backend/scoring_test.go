package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func opsServer003() (*OpsService, http.Handler) {
	svc := newOpsService(seedOpsRecords())
	return svc, newOpsMux(svc)
}

func opsReq003(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func create003(t *testing.T, h http.Handler, id string) {
	rec := opsReq003(t, h, http.MethodPost, "/api/ops/records", `{"id":"`+id+`","subject":"s","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s status = %d", id, rec.Code)
	}
}

func TestOpsTransitionConflictNoPhantomAudit(t *testing.T) {
	_, h := opsServer003()
	create003(t, h, "ops-ph-1")
	rec := opsReq003(t, h, http.MethodPost, "/api/ops/records/ops-ph-1/status", `{"status":"active","expected_revision":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("first transition status = %d, want 200", rec.Code)
	}
	rec = opsReq003(t, h, http.MethodPost, "/api/ops/records/ops-ph-1/status", `{"status":"paused","expected_revision":1}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("stale transition status = %d, want 409", rec.Code)
	}
	rec = opsReq003(t, h, http.MethodGet, "/api/ops/audit?record_id=ops-ph-1", "")
	var payload struct {
		Events []OpsEvent `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if len(payload.Events) != 2 {
		t.Fatalf("audit events for ops-ph-1 = %d, want 2 (created + one real transition)", len(payload.Events))
	}
}

func TestOpsTransitionIllegalRejected(t *testing.T) {
	_, h := opsServer003()
	create003(t, h, "ops-il-1")
	rec := opsReq003(t, h, http.MethodPost, "/api/ops/records/ops-il-1/status", `{"status":"closed","expected_revision":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("close transition status = %d, want 200", rec.Code)
	}
	rec = opsReq003(t, h, http.MethodPost, "/api/ops/records/ops-il-1/status", `{"status":"active","expected_revision":2}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("illegal closed->active status = %d, want 422", rec.Code)
	}
	rec = opsReq003(t, h, http.MethodGet, "/api/ops/records/ops-il-1", "")
	var got OpsRecord
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Status != "closed" {
		t.Fatalf("record status after rejected transition = %s, want closed", got.Status)
	}
}

func TestOpsUpdateResponseFresh(t *testing.T) {
	_, h := opsServer003()
	create003(t, h, "ops-up-1")
	rec := opsReq003(t, h, http.MethodPut, "/api/ops/records/ops-up-1", `{"subject":"v2","owner":"yard","priority":"critical","status":"queued","labels":{"site":"east"},"expected_revision":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200", rec.Code)
	}
	var updated OpsRecord
	_ = json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated.Revision != 2 {
		t.Fatalf("update response revision = %d, want 2", updated.Revision)
	}
}

func TestOpsTransitionPolicyStatus(t *testing.T) {
	_, h := opsServer003()
	rec := opsReq003(t, h, http.MethodPost, "/api/ops/records", `{"id":"ops-no-lab","subject":"no labels","owner":"yard","priority":"high","status":"queued"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("policy rejection status = %d, want 422", rec.Code)
	}
}
